package pcp

import (
	"bitbucket.org/sunybingcloud/electron/rapl"
	"bufio"
	"container/ring"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"math"
	"bitbucket.org/sunybingcloud/electron/constants"
)

func round(num float64) int {
	return int(math.Floor(num + math.Copysign(0.5, num)))
}

func getNextCapValue(curCapValue float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(curCapValue * output)) / output
}

func StartPCPLogAndProgressiveExtremaCap(quit chan struct{}, logging *bool, prefix string, hiThreshold, loThreshold float64) {
	const pcpCommand string = "pmdumptext -m -l -f '' -t 1.0 -d , -c config"
	cmd := exec.Command("sh", "-c", pcpCommand)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if hiThreshold < loThreshold {
		log.Println("High threshold is lower than the low threshold")
	}

	logFile, err := os.Create("./" + prefix + ".pcplog")
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
			metricSplit := strings.Split(hostMetric, ":")
			//log.Printf("%d Host %s: Metric: %s\n", i, split[0], split[1])

			if strings.Contains(metricSplit[1], "RAPL_ENERGY_PKG") ||
				strings.Contains(metricSplit[1], "RAPL_ENERGY_DRAM") {
				//fmt.Println("Index: ", i)
				powerIndexes = append(powerIndexes, i)
				indexToHost[i] = metricSplit[0]

				// Only create one ring per host
				if _, ok := powerHistories[metricSplit[0]]; !ok {
					powerHistories[metricSplit[0]] = ring.New(20) // Two PKGS, two DRAM per node,  20 = 5 seconds of tracking
				}
			}
		}

		// Throw away first set of results
		scanner.Scan()

		//cappedHosts := make(map[string]bool)
		// Keep track of the capped victims and the corresponding cap value.
		cappedVictims := make(map[string]float64)
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

				clusterMean := averageClusterPowerHistory(clusterPowerHist)

				log.Printf("Total power: %f, %d Sec Avg: %f", clusterPower, clusterPowerHist.Len(), clusterMean)

				if clusterMean >= hiThreshold {
					log.Printf("Need to cap a node")
					// Create statics for all victims and choose one to cap
					victims := make([]Victim, 0, 8)

					// TODO: Just keep track of the largest to reduce fron nlogn to n
					for name, history := range powerHistories {

						histMean := averageNodePowerHistory(history)

						// Consider doing mean calculations using go routines if we need to speed up
						victims = append(victims, Victim{Watts: histMean, Host: name})
					}

					sort.Sort(VictimSorter(victims)) // Sort by average wattage

					// Finding the best victim to cap.
					for i := 0; i < len(victims); i++ {
						if curCapValue, ok := cappedVictims[victims[i].Host]; ok {
							// checking whether we can continue to cap this host.
							// If yes, then we cap it to half the current cap value.
							// Else, we push it to the orderedCapped and continue.
							if curCapValue > constants.CapThreshold {
								newCapValue := getNextCapValue(curCapValue/2.0, 1)
								if err := rapl.Cap(victims[0].Host, "rapl", newCapValue); err != nil {
									log.Print("Error capping host")
								}
								// Updating the curCapValue in cappedVictims
								cappedVictims[victims[0].Host] = newCapValue
								break
							} else {
								// deleting entry in cappedVictims
								delete(cappedVictims, victims[i].Host)
								// Now this host can be uncapped.
								orderCapped = append(orderCapped, victims[i].Host)
							}
						}
					}

				} else if clusterMean < loThreshold {
					if len(orderCapped) > 0 {
						host := orderCapped[len(orderCapped)-1]
						orderCapped = orderCapped[:len(orderCapped)-1]
						// cappedVictims would contain the latest cap value for this host.
						newCapValue := getNextCapValue(cappedVictims[host]/2.0, 1)
						if err := rapl.Cap(host, "rapl", newCapValue); err != nil {
							log.Print("Error capping host")
						}
						// Adding entry for the host to cappedVictims
						cappedVictims[host] = newCapValue // Now this host can be capped again.
					}
				}
			}
			seconds++
		}

	}(logging, hiThreshold, loThreshold)
}
