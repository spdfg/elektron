package pcp

import (
	"bitbucket.org/sunybingcloud/electron/constants"
	"bitbucket.org/sunybingcloud/electron/rapl"
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
)

func round(num float64) int {
	return int(math.Floor(num + math.Copysign(0.5, num)))
}

func getNextCapValue(curCapValue float64, precision int) float64 {
	curCapValue /= 2.0
	output := math.Pow(10, float64(precision))
	return float64(round(curCapValue*output)) / output
}

func getNextUncapValue(curCapValue float64, precision int) float64 {
	curCapValue *= 2.0
	output := math.Pow(10, float64(precision))
	return float64(round(curCapValue*output)) / output
}

func StartPCPLogAndProgressiveExtremaCap(quit chan struct{}, logging *bool, prefix string, hiThreshold, loThreshold float64) {
	log.Println("Inside Log and Progressive Extrema")
	const pcpCommand string = "pmdumptext -m -l -f '' -t 1.0 -d , -c config"
	cmd := exec.Command("sh", "-c", pcpCommand)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if hiThreshold < loThreshold {
		log.Println("High threshold is lower than low threshold!")
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

		// cappedHosts := make(map[string]bool)
		// Keep track of the capped victims and the corresponding cap value.
		cappedVictims := make(map[string]float64)
		orderCapped := make([]string, 0, 8)
		orderCappedVictims := make(map[string]float64) // Parallel data structure to orderCapped to make it possible to search victims, that are to be uncapped, faster.
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
					log.Println("Need to cap a node")
					// Create statics for all victims and choose one to cap
					victims := make([]Victim, 0, 8)

					// TODO: Just keep track of the largest to reduce fron nlogn to n
					for name, history := range powerHistories {

						histMean := averageNodePowerHistory(history)

						// Consider doing mean calculations using go routines if we need to speed up
						victims = append(victims, Victim{Watts: histMean, Host: name})
					}

					sort.Sort(VictimSorter(victims)) // Sort by average wattage

					// Finding the best victim to cap in a round robin manner
					newVictimFound := false
					alreadyCappedHosts := []string{} // Hosts of already capped hosts in decreasing order of recent power consumption
					for i := 0; i < len(victims); i++ {
						// Try to pick a victim that hasn't been capped yet.
						if _, ok := cappedVictims[victims[i].Host]; !ok {
							// If this victim is present in orderCapped, then we move on to find another victim that we can cap.
							if _, ok := orderCappedVictims[victims[i].Host]; ok {
								// Adding the host to the alreadyCappedHosts
								alreadyCappedHosts = append(alreadyCappedHosts, victims[i].Host)
								continue
							}
							// Need to cap this victim and add it to the cappedVictims
							if err := rapl.Cap(victims[i].Host, "rapl", 50.0); err != nil {
								log.Printf("Error capping host %s", victims[i].Host)
							} else {
								log.Printf("Capped host[%s] at %f", victims[i].Host, 50.0)
								cappedVictims[victims[i].Host] = 50.0
								newVictimFound = true
								// This node can be uncapped and hence adding to orderCapped
								orderCapped = append(orderCapped, victims[i].Host)
								orderCappedVictims[victims[i].Host] = 50.0
								break // breaking only on successful cap
							}
						} else {
							alreadyCappedHosts = append(alreadyCappedHosts, victims[i].Host)
						}
					}
					// If no new victim found, then we need to cap the best victim among the already capped victims
					if !newVictimFound {
						for i := 0; i < len(alreadyCappedHosts); i++ {
							// Checking if this node can be uncapped too
							if capValue, ok := orderCappedVictims[alreadyCappedHosts[i]]; ok {
								// if capValue is greater than the threshold then cap, else continue
								if capValue > constants.CapThreshold {
									newCapValue := getNextCapValue(capValue, 2)
									if err := rapl.Cap(alreadyCappedHosts[i], "rapl", newCapValue); err != nil {
										log.Printf("Error capping host %s", alreadyCappedHosts[i])
									} else {
										// successful cap
										log.Printf("Capped host[%s] at %f", alreadyCappedHosts[i], newCapValue)
										// Checking whether this victim can be capped further
										if newCapValue <= constants.CapThreshold {
											// deleting victim from cappedVictims
											delete(cappedVictims, alreadyCappedHosts[i])
											// updating the cap value in orderCappedVictims
											orderCappedVictims[alreadyCappedHosts[i]] = newCapValue
										} else {
											// updating the cap value
											cappedVictims[alreadyCappedHosts[i]] = newCapValue
											orderCappedVictims[alreadyCappedHosts[i]] = newCapValue
										}
										break // exiting only on a successful cap.
									}
								} else {
									// Do nothing
								}
							} else {
								// This host can definitely be capped.
								// Cap this host to half it's current cap value and update the new cap value in cappedVictims and orderCappedVictims
								// If we have hit the threshold then we add this host to orderCapped to indicate that it needs to be uncapped.
								newCapValue := getNextCapValue(cappedVictims[alreadyCappedHosts[i]], 2)
								if err := rapl.Cap(alreadyCappedHosts[i], "rapl", newCapValue); err != nil {
									log.Printf("Error capping host %s", alreadyCappedHosts[i])
								} else {
									log.Printf("Capped host[%s] at %f", alreadyCappedHosts[i], newCapValue)
									// Checking whether this victim can be capped further
									if newCapValue <= constants.CapThreshold {
										// deleting victim from cappedVictims
										delete(cappedVictims, alreadyCappedHosts[i])
										// staging victim for uncapping
										orderCapped = append(orderCapped, alreadyCappedHosts[i])
										orderCappedVictims[alreadyCappedHosts[i]] = constants.CapThreshold
									} else {
										// Updating the cap value of the victim
										cappedVictims[alreadyCappedHosts[i]] = newCapValue
									}
									break // exiting only on a successful uncap
								}
							}
						}
					}

				} else if clusterMean < loThreshold {
					log.Println("Need to uncap a node")
					if len(orderCapped) > 0 {
						host := orderCapped[len(orderCapped)-1]
						// Removing victim from orderCapped only if it has been completely uncapped to 100%
						if cappedVictims[host] == 100.0 {
							orderCapped = orderCapped[:len(orderCapped)-1]
							delete(orderCappedVictims, host)
						} else {
							newCapValue := getNextUncapValue(cappedVictims[host], 2)
							// Uncapping the victim
							if err := rapl.Cap(host, "rapl", newCapValue); err != nil {
								log.Printf("Error uncapping host %s", host)
							} else {
								// Successful uncap
								log.Printf("Uncapped host[%s] to %f", host, newCapValue)
								// If the new cap value is 100, then this node cannot be uncapped
								if newCapValue == 100.0 {
									orderCapped = orderCapped[:len(orderCapped)-1]
									delete(orderCappedVictims, host)
									// Updating cappedVictims
									cappedVictims[host] = newCapValue
								} else if newCapValue > constants.CapThreshold {
									// This host can be capped
									cappedVictims[host] = newCapValue
									// Updating orderCappedVictims
									orderCappedVictims[host] = newCapValue
								}
							}
						}
					} else {
						// No node has been capped until now.
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
