package powerCap

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

	"github.com/mesos/mesos-go/api/v0/scheduler"
	"github.com/montanaflynn/stats"
	elekLogDef "gitlab.com/spdf/elektron/logging/def"
	"gitlab.com/spdf/elektron/pcp"
	"gitlab.com/spdf/elektron/rapl"
	"gitlab.com/spdf/elektron/schedulers"
)

func StartPCPLogAndExtremaDynamicCap(quit chan struct{}, logging *bool, hiThreshold, loThreshold float64,
	logMType chan elekLogDef.LogMessageType, logMsg chan string, pcpConfigFile string, s scheduler.Scheduler) {

	baseSchedRef := s.(*schedulers.BaseScheduler)
	var pcpCommand string = "pmdumptext -m -l -f '' -t 1.0 -d , -c " + pcpConfigFile
	cmd := exec.Command("sh", "-c", pcpCommand, pcpConfigFile)
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
		logMType <- elekLogDef.PCP
		logMsg <- scanner.Text()

		headers := strings.Split(scanner.Text(), ",")

		logMType <- elekLogDef.DEG_COL
		logMsg <- "CPU Variance, CPU Task Share Variance, Memory Variance, Memory Task Share Variance"

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
				logMType <- elekLogDef.GENERAL
				logMsg <- "Logging PCP..."
				text := scanner.Text()
				split := strings.Split(text, ",")
				logMType <- elekLogDef.PCP
				logMsg <- text

				memUtils := pcp.MemUtilPerNode(text)
				memTaskShares := make([]float64, len(memUtils))

				cpuUtils := pcp.CpuUtilPerNode(text)
				cpuTaskShares := make([]float64, len(cpuUtils))

				for i := 0; i < 8; i++ {
					host := fmt.Sprintf("stratos-00%d.cs.binghamton.edu", i+1)
					if slaveID, ok := baseSchedRef.HostNameToSlaveID[host]; ok {
						baseSchedRef.TasksRunningMutex.Lock()
						tasksRunning := len(baseSchedRef.Running[slaveID])
						baseSchedRef.TasksRunningMutex.Unlock()
						if tasksRunning > 0 {
							cpuTaskShares[i] = cpuUtils[i] / float64(tasksRunning)
							memTaskShares[i] = memUtils[i] / float64(tasksRunning)
						}
					}
				}

				// Variance in resource utilization shows how the current workload has been distributed.
				// However, if the number of tasks running are not equally distributed, utilization variance figures become
				// less relevant as they do not express the distribution of CPU intensive tasks.
				// We thus also calculate `task share variance`, which basically signifies how the workload is distributed
				// across each node per share.

				cpuVariance, _ := stats.Variance(cpuUtils)
				cpuTaskSharesVariance, _ := stats.Variance(cpuTaskShares)
				memVariance, _ := stats.Variance(memUtils)
				memTaskSharesVariance, _ := stats.Variance(memTaskShares)

				logMType <- elekLogDef.DEG_COL
				logMsg <- fmt.Sprintf("%f, %f, %f, %f", cpuVariance, cpuTaskSharesVariance, memVariance, memTaskSharesVariance)

				totalPower := 0.0
				for _, powerIndex := range powerIndexes {
					power, _ := strconv.ParseFloat(split[powerIndex], 64)

					host := indexToHost[powerIndex]

					powerHistories[host].Value = power
					powerHistories[host] = powerHistories[host].Next()

					logMType <- elekLogDef.GENERAL
					logMsg <- fmt.Sprintf("Host: %s, Power: %f", indexToHost[powerIndex], (power * pcp.RAPLUnits))

					totalPower += power
				}
				clusterPower := totalPower * pcp.RAPLUnits

				clusterPowerHist.Value = clusterPower
				clusterPowerHist = clusterPowerHist.Next()

				clusterMean := pcp.AverageClusterPowerHistory(clusterPowerHist)

				logMType <- elekLogDef.GENERAL
				logMsg <- fmt.Sprintf("Total power: %f, %d Sec Avg: %f", clusterPower, clusterPowerHist.Len(), clusterMean)

				if clusterMean > hiThreshold {
					logMType <- elekLogDef.GENERAL
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
							logMType <- elekLogDef.GENERAL
							logMsg <- fmt.Sprintf("Capping Victim %s Avg. Wattage: %f", victim.Host, victim.Watts*pcp.RAPLUnits)
							if err := rapl.Cap(victim.Host, "rapl", 50); err != nil {
								logMType <- elekLogDef.ERROR
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
						logMType <- elekLogDef.GENERAL
						logMsg <- fmt.Sprintf("Uncapped host %s", host)
						if err := rapl.Cap(host, "rapl", 100); err != nil {
							logMType <- elekLogDef.ERROR
							logMsg <- "Error capping host"
						}
					}
				}
			}

			seconds++
		}
	}(logging, hiThreshold, loThreshold)

	logMType <- elekLogDef.GENERAL
	logMsg <- "PCP logging started"

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)

	select {
	case <-quit:
		logMType <- elekLogDef.GENERAL
		logMsg <- "Stopping PCP logging in 5 seconds"
		time.Sleep(5 * time.Second)

		// http://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
		// Kill process and all children processes.
		syscall.Kill(-pgid, 15)
		return
	}
}
