package pcp

import (
	"bufio"
	"container/ring"
	"log"
	"math"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"bitbucket.org/bingcloud/electron/rapl"
)

var RAPLUnits = math.Pow(2, -32)

func meanPKG(history *ring.Ring) float64 {

	total := 0.0
	count := 0.0

	history.Do(func(x interface{}) {
		if val, ok := x.(float64); ok { //Add it if we can get a float
			total += val
			count++
		}
	})

	if count == 0.0 {
		return 0.0
	}

	count /= 2

	return (total / count)
}

func meanCluster(history *ring.Ring) float64 {

	total := 0.0
	count := 0.0

	history.Do(func(x interface{}) {
		if val, ok := x.(float64); ok { //Add it if we can get a float
			total += val
			count++
		}
	})

	if count == 0.0 {
		return 0.0
	}

	return (total / count)
}

func StartLogAndDynamicCap(quit chan struct{}, logging *bool, prefix string, hiThreshold, loThreshold float64) {
	const pcpCommand string = "pmdumptext -m -l -f '' -t 1.0 -d , -c config"
	cmd := exec.Command("sh", "-c", pcpCommand)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	startTime := time.Now().Format("20060102150405")

	if hiThreshold < loThreshold {
		log.Println("High threshold is lower than low threshold!")
	}

	logFile, err := os.Create("./" + prefix + startTime + ".pcplog")
	if err != nil {
		log.Fatal(err)
	}

	defer logFile.Close()

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	//cmd.Stdout = stdout

	scanner := bufio.NewScanner(pipe)

	go func(logging *bool, hiThreshold, loThreshold float64) {
		// Get names of the columns
		scanner.Scan()

		// Write to logfile
		logFile.WriteString(scanner.Text() + "\n")

		headers := strings.Split(scanner.Text(), ",")

		powerIndexes := make([]int, 0, 0)
		powerHistories := make(map[string]*ring.Ring)
		indexToHost := make(map[int]string)

		for i, hostMetric := range headers {
			split := strings.Split(hostMetric, ":")
			//log.Printf("%d Host %s: Metric: %s\n", i, split[0], split[1])

			if strings.Contains(split[1], "RAPL_ENERGY_PKG") {
				//fmt.Println("Index: ", i)
				powerIndexes = append(powerIndexes, i)
				indexToHost[i] = split[0]
				powerHistories[split[0]] = ring.New(10) // Two PKGS per node, 10 = 5 seconds tracking
			}
		}

		// Throw away first set of results
		scanner.Scan()

		cappedHosts := make(map[string]bool)
		orderCapped := make([]string, 0, 8)
		clusterPowerHist := ring.New(5)
		seconds := 0

		for scanner.Scan() {

			if *logging {
				log.Println("Logging PCP...")
				split := strings.Split(scanner.Text(), ",")
				logFile.WriteString(scanner.Text() + "\n")

				totalPower := 0.0
				for _, powerIndex := range powerIndexes {
					power, _ := strconv.ParseFloat(split[powerIndex], 64)

					host := indexToHost[powerIndex]

					powerHistories[host].Value = power
					powerHistories[host] = powerHistories[host].Next()

					log.Printf("Host: %s, Power: %f", indexToHost[powerIndex], (power * RAPLUnits))

					totalPower += power
				}
				clusterPower := totalPower * RAPLUnits

				clusterPowerHist.Value = clusterPower
				clusterPowerHist = clusterPowerHist.Next()

				clusterMean := meanCluster(clusterPowerHist)

				log.Printf("Total power: %f, %d Sec Avg: %f", clusterPower, clusterPowerHist.Len(), clusterMean)

				if clusterMean > hiThreshold {
					log.Printf("Need to cap a node")
					// Create statics for all victims and choose one to cap
					victims := make([]Victim, 0, 8)

					// TODO: Just keep track of the largest to reduce fron nlogn to n
					for name, history := range powerHistories {

						histMean := meanPKG(history)
						// Consider doing mean calculations using go routines if we need to speed up
						victims = append(victims, Victim{Watts: histMean, Host: name})
						//log.Printf("host: %s, Avg: %f", name, histMean * RAPLUnits)
					}

					sort.Sort(VictimSorter(victims)) // Sort by average wattage

					// From  best victim to worst, if everyone is already capped NOOP
					for _, victim := range victims {
						// Only cap if host hasn't been capped yet
						if !cappedHosts[victim.Host] {
							cappedHosts[victim.Host] = true
							orderCapped = append(orderCapped, victim.Host)
							log.Printf("Capping Victim %s Avg. Wattage: %f", victim.Host, victim.Watts*RAPLUnits)
							if err := rapl.Cap(victim.Host, "rapl", 50); err != nil {
								log.Print("Error capping host")
							}
							break // Only cap one machine at at time
						}
					}

				} else if clusterMean < loThreshold {

					if len(orderCapped) > 0 {
						host := orderCapped[len(orderCapped)-1]
						orderCapped = orderCapped[:len(orderCapped)-1]
						cappedHosts[host] = false
						// User RAPL package to send uncap
						log.Printf("Uncapping host %s", host)
						if err := rapl.Cap(host, "rapl", 100); err != nil {
							log.Print("Error uncapping host")
						}
					}
				}
			}

			seconds++
		}
	}(logging, hiThreshold, loThreshold)

	log.Println("PCP logging started")

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)

	select {
	case <-quit:
		log.Println("Stopping PCP logging in 5 seconds")
		time.Sleep(5 * time.Second)

		// http://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
		// kill process and all children processes
		syscall.Kill(-pgid, 15)
		return
	}
}
