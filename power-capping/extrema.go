package pcp

import (
	"bufio"
	"container/ring"
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	elecLogDef "gitlab.com/spdf/elektron/logging/def"
	"gitlab.com/spdf/elektron/pcp"
	"gitlab.com/spdf/elektron/rapl"
)

func StartPCPLogAndExtremaDynamicCap(quit chan struct{}, logging *bool, hiThreshold, loThreshold float64,
	logMType chan elecLogDef.LogMessageType, logMsg chan string) {

	const pcpCommand string = "pmdumptext -m -l -f '' -t 1.0 -d , -c config"
	cmd := exec.Command("sh", "-c", pcpCommand)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if hiThreshold < loThreshold {
		log.Println("High threshold is lower than low threshold!")
	}

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	//cmd.Stdout = stdout

	scanner := bufio.NewScanner(pipe)

	go func(logging *bool, hiThreshold, loThreshold float64) {
		// Get names of the columns.
		scanner.Scan()

		// Write to logfile
		logMType <- elecLogDef.PCP
		logMsg <- scanner.Text()

		headers := strings.Split(scanner.Text(), ",")

		powerIndexes := make([]int, 0, 0)
		powerHistories := make(map[string]*ring.Ring)
		indexToHost := make(map[int]string)

		for i, hostMetric := range headers {
			metricSplit := strings.Split(hostMetric, ":")

			if strings.Contains(metricSplit[1], "RAPL_ENERGY_PKG") ||
				strings.Contains(metricSplit[1], "RAPL_ENERGY_DRAM") {
				powerIndexes = append(powerIndexes, i)
				indexToHost[i] = metricSplit[0]

				// Only create one ring per host.
				if _, ok := powerHistories[metricSplit[0]]; !ok {
					// Two PKGS, two DRAM per node, 20 - 5 seconds of tracking.
					powerHistories[metricSplit[0]] = ring.New(20)
				}
			}
		}

		// Throw away first set of results.
		scanner.Scan()

		cappedHosts := make(map[string]bool)
		orderCapped := make([]string, 0, 8)
		clusterPowerHist := ring.New(5)
		seconds := 0

		for scanner.Scan() {

			if *logging {
				logMType <- elecLogDef.GENERAL
				logMsg <- "Logging PCP..."
				split := strings.Split(scanner.Text(), ",")
				logMType <- elecLogDef.PCP
				logMsg <- scanner.Text()

				totalPower := 0.0
				for _, powerIndex := range powerIndexes {
					power, _ := strconv.ParseFloat(split[powerIndex], 64)

					host := indexToHost[powerIndex]

					powerHistories[host].Value = power
					powerHistories[host] = powerHistories[host].Next()

					logMType <- elecLogDef.GENERAL
					logMsg <- fmt.Sprintf("Host: %s, Power: %f", indexToHost[powerIndex], (power * pcp.RAPLUnits))

					totalPower += power
				}
				clusterPower := totalPower * pcp.RAPLUnits

				clusterPowerHist.Value = clusterPower
				clusterPowerHist = clusterPowerHist.Next()

				clusterMean := pcp.AverageClusterPowerHistory(clusterPowerHist)

				logMType <- elecLogDef.GENERAL
				logMsg <- fmt.Sprintf("Total power: %f, %d Sec Avg: %f", clusterPower, clusterPowerHist.Len(), clusterMean)

				if clusterMean > hiThreshold {
					logMType <- elecLogDef.GENERAL
					logMsg <- "Need to cap a node"
					// Create statics for all victims and choose one to cap
					victims := make([]pcp.Victim, 0, 8)

					// TODO: Just keep track of the largest to reduce fron nlogn to n
					for name, history := range powerHistories {

						histMean := pcp.AverageNodePowerHistory(history)

						// Consider doing mean calculations using go routines if we need to speed up.
						victims = append(victims, pcp.Victim{Watts: histMean, Host: name})
					}

					sort.Sort(pcp.VictimSorter(victims)) // Sort by average wattage.

					// From  best victim to worst, if everyone is already capped NOOP.
					for _, victim := range victims {
						// Only cap if host hasn't been capped yet.
						if !cappedHosts[victim.Host] {
							cappedHosts[victim.Host] = true
							orderCapped = append(orderCapped, victim.Host)
							logMType <- elecLogDef.GENERAL
							logMsg <- fmt.Sprintf("Capping Victim %s Avg. Wattage: %f", victim.Host, victim.Watts*pcp.RAPLUnits)
							if err := rapl.Cap(victim.Host, "rapl", 50); err != nil {
								logMType <- elecLogDef.ERROR
								logMsg <- "Error capping host"
							}
							break // Only cap one machine at at time.
						}
					}

				} else if clusterMean < loThreshold {

					if len(orderCapped) > 0 {
						host := orderCapped[len(orderCapped)-1]
						orderCapped = orderCapped[:len(orderCapped)-1]
						cappedHosts[host] = false
						// User RAPL package to send uncap.
						log.Printf("Uncapping host %s", host)
						logMType <- elecLogDef.GENERAL
						logMsg <- fmt.Sprintf("Uncapped host %s", host)
						if err := rapl.Cap(host, "rapl", 100); err != nil {
							logMType <- elecLogDef.ERROR
							logMsg <- "Error capping host"
						}
					}
				}
			}

			seconds++
		}
	}(logging, hiThreshold, loThreshold)

	logMType <- elecLogDef.GENERAL
	logMsg <- "PCP logging started"

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)

	select {
	case <-quit:
		logMType <- elecLogDef.GENERAL
		logMsg <- "Stopping PCP logging in 5 seconds"
		time.Sleep(5 * time.Second)

		// http://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
		// Kill process and all children processes.
		syscall.Kill(-pgid, 15)
		return
	}
}
