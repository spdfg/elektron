// Copyright (C) 2018 spdfg
//
// This file is part of Elektron.
//
// Elektron is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Elektron is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Elektron.  If not, see <http://www.gnu.org/licenses/>.
//

package powerCap

import (
	"bufio"
	"container/ring"
	"fmt"
	"math"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spdfg/elektron/constants"
	elekLog "github.com/spdfg/elektron/elektronLogging"
	elekLogTypes "github.com/spdfg/elektron/elektronLogging/types"
	"github.com/spdfg/elektron/pcp"
	"github.com/spdfg/elektron/rapl"
	"github.com/spdfg/elektron/utilities"
)

func round(num float64) int {
	return int(math.Floor(num + math.Copysign(0.5, num)))
}

func getNextCapValue(curCapValue float64, precision int) float64 {
	curCapValue /= 2.0
	output := math.Pow(10, float64(precision))
	return float64(round(curCapValue*output)) / output
}

func StartPCPLogAndProgressiveExtremaCap(quit chan struct{}, logging *bool, hiThreshold, loThreshold float64, pcpConfigFile string) {

	var pcpCommand string = "pmdumptext -m -l -f '' -t 1.0 -d , -c " + pcpConfigFile
	cmd := exec.Command("sh", "-c", pcpCommand, pcpConfigFile)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if hiThreshold < loThreshold {
		elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
			log.InfoLevel,
			log.Fields{}, "High threshold is lower than low threshold!")
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
		elekLog.ElektronLog.Log(elekLogTypes.PCP,
			log.InfoLevel,
			log.Fields{}, scanner.Text())

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
					// Two PKGS, two DRAM per node, 20 = 5 seconds of tracking.
					powerHistories[metricSplit[0]] = ring.New(20)
				}
			}
		}

		// Throw away first set of results.
		scanner.Scan()

		// To keep track of the capped states of the capped victims.
		cappedVictims := make(map[string]float64)
		// TODO: Come with a better name for this.
		orderCapped := make([]string, 0, 8)
		// TODO: Change this to a priority queue ordered by the cap value. This will get rid of the sorting performed in the code.
		// Parallel data structure to orderCapped to keep track of the uncapped states of the uncapped victims.
		orderCappedVictims := make(map[string]float64)
		clusterPowerHist := ring.New(5)
		seconds := 0

		for scanner.Scan() {
			if *logging {
				elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
					log.InfoLevel,
					log.Fields{}, "Logging PCP...")
				split := strings.Split(scanner.Text(), ",")

				text := scanner.Text()
				elekLog.ElektronLog.Log(elekLogTypes.PCP,
					log.InfoLevel,
					log.Fields{}, text)

				totalPower := 0.0
				for _, powerIndex := range powerIndexes {
					power, _ := strconv.ParseFloat(split[powerIndex], 64)

					host := indexToHost[powerIndex]

					powerHistories[host].Value = power
					powerHistories[host] = powerHistories[host].Next()

					elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
						log.InfoLevel,
						log.Fields{"Host": fmt.Sprintf("%s", indexToHost[powerIndex]), "Power": fmt.Sprintf("%f", (power * pcp.RAPLUnits))},
						"")
					totalPower += power
				}
				clusterPower := totalPower * pcp.RAPLUnits

				clusterPowerHist.Value = clusterPower
				clusterPowerHist = clusterPowerHist.Next()

				clusterMean := pcp.AverageClusterPowerHistory(clusterPowerHist)

				elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
					log.InfoLevel,
					log.Fields{"Total power": fmt.Sprintf("%f %d", clusterPower, clusterPowerHist.Len()),
						"Sec Avg": fmt.Sprintf("%f", clusterMean)},
					"")

				if clusterMean >= hiThreshold {
					elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
						log.InfoLevel,
						log.Fields{}, "Need to cap a node")

					elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
						log.InfoLevel,
						log.Fields{"Cap values of capped victims": fmt.Sprintf("%v", cappedVictims)}, "")

					elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
						log.InfoLevel,
						log.Fields{"Cap values of victims to uncap": fmt.Sprintf("%v", orderCappedVictims)}, "")
					// Create statics for all victims and choose one to cap
					victims := make([]pcp.Victim, 0, 8)

					// TODO: Just keep track of the largest to reduce fron nlogn to n
					for name, history := range powerHistories {

						histMean := pcp.AverageNodePowerHistory(history)

						// Consider doing mean calculations using go routines if we need to speed up.
						victims = append(victims, pcp.Victim{Watts: histMean, Host: name})
					}

					sort.Sort(pcp.VictimSorter(victims)) // Sort by average wattage.

					// Finding the best victim to cap in a round robin manner.
					newVictimFound := false
					alreadyCappedHosts := []string{} // Host-names of victims that are already capped.
					for i := 0; i < len(victims); i++ {
						// Try to pick a victim that hasn't been capped yet.
						if _, ok := cappedVictims[victims[i].Host]; !ok {
							// If this victim can't be capped further, then we move on to find another victim.
							if _, ok := orderCappedVictims[victims[i].Host]; ok {
								continue
							}
							// Need to cap this victim.
							if err := rapl.Cap(victims[i].Host, "rapl", 50.0); err != nil {

								elekLog.ElektronLog.Log(elekLogTypes.ERROR,
									log.ErrorLevel,
									log.Fields{"Error capping host": fmt.Sprintf("%s", victims[i].Host)}, "")
							} else {

								elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
									log.InfoLevel,
									log.Fields{}, fmt.Sprintf("Capped host[%s] at %f", victims[i].Host, 50.0))
								// Keeping track of this victim and it's cap value
								cappedVictims[victims[i].Host] = 50.0
								newVictimFound = true
								// This node can be uncapped and hence adding to orderCapped.
								orderCapped = append(orderCapped, victims[i].Host)
								orderCappedVictims[victims[i].Host] = 50.0
								break // Breaking only on successful cap.
							}
						} else {
							alreadyCappedHosts = append(alreadyCappedHosts, victims[i].Host)
						}
					}
					// If no new victim found, then we need to cap the best victim among the ones that are already capped.
					if !newVictimFound {
						canCapAlreadyCappedVictim := false
						for i := 0; i < len(alreadyCappedHosts); i++ {
							// If already capped then the host must be present in orderCappedVictims.
							capValue := orderCappedVictims[alreadyCappedHosts[i]]
							// If capValue is greater than the threshold then cap, else continue.
							if capValue > constants.LowerCapLimit {
								newCapValue := getNextCapValue(capValue, 2)
								if err := rapl.Cap(alreadyCappedHosts[i], "rapl", newCapValue); err != nil {

									elekLog.ElektronLog.Log(elekLogTypes.ERROR,
										log.ErrorLevel,
										log.Fields{"Error capping host": fmt.Sprintf("%s", alreadyCappedHosts[i])}, "")
								} else {
									// Successful cap
									elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
										log.InfoLevel,
										log.Fields{}, fmt.Sprintf("Capped host[%s] at %f", alreadyCappedHosts[i], newCapValue))
									// Checking whether this victim can be capped further
									if newCapValue <= constants.LowerCapLimit {
										// Deleting victim from cappedVictims.
										delete(cappedVictims, alreadyCappedHosts[i])
										// Updating the cap value in orderCappedVictims.
										orderCappedVictims[alreadyCappedHosts[i]] = newCapValue
									} else {
										// Updating the cap value.
										cappedVictims[alreadyCappedHosts[i]] = newCapValue
										orderCappedVictims[alreadyCappedHosts[i]] = newCapValue
									}
									canCapAlreadyCappedVictim = true
									break // Breaking only on successful cap.
								}
							} else {
								// Do nothing.
								// Continue to find another victim to cap.
								// If cannot find any victim, then all nodes have been
								// capped to the maximum and we stop capping at this point.
							}
						}
						if !canCapAlreadyCappedVictim {
							elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
								log.InfoLevel,
								log.Fields{}, "No Victim left to cap")
						}
					}

				} else if clusterMean < loThreshold {

					elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
						log.InfoLevel,
						log.Fields{}, "Need to uncap a node")
					elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
						log.InfoLevel,
						log.Fields{"Cap values of capped victims": fmt.Sprintf("%v", cappedVictims)}, "")
					elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
						log.InfoLevel,
						log.Fields{"Cap values of victims to uncap": fmt.Sprintf("%v", orderCappedVictims)}, "")
					if len(orderCapped) > 0 {
						// We pick the host that is capped the most to uncap.
						orderCappedToSort := utilities.GetPairList(orderCappedVictims)
						sort.Sort(orderCappedToSort) // Sorted hosts in non-decreasing order of capped states.
						hostToUncap := orderCappedToSort[0].Key
						// Uncapping the host.
						// This is a floating point operation and might suffer from precision loss.
						newUncapValue := orderCappedVictims[hostToUncap] * 2.0
						if err := rapl.Cap(hostToUncap, "rapl", newUncapValue); err != nil {

							elekLog.ElektronLog.Log(elekLogTypes.ERROR,
								log.ErrorLevel,
								log.Fields{"Error uncapping host": fmt.Sprintf("%s", hostToUncap)}, "")
						} else {
							// Successful uncap
							elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
								log.InfoLevel,
								log.Fields{}, fmt.Sprintf("Uncapped host[%s] to %f", hostToUncap, newUncapValue))
							// Can we uncap this host further. If not, then we remove its entry from orderCapped
							if newUncapValue >= 100.0 { // can compare using ==
								// Deleting entry from orderCapped
								for i, victimHost := range orderCapped {
									if victimHost == hostToUncap {
										orderCapped = append(orderCapped[:i], orderCapped[i+1:]...)
										break // We are done removing host from orderCapped.
									}
								}
								// Removing entry for host from the parallel data structure.
								delete(orderCappedVictims, hostToUncap)
								// Removing entry from cappedVictims as this host is no longer capped.
								delete(cappedVictims, hostToUncap)
							} else if newUncapValue > constants.LowerCapLimit { // This check is unnecessary and can be converted to 'else'.
								// Updating the cap value.
								orderCappedVictims[hostToUncap] = newUncapValue
								cappedVictims[hostToUncap] = newUncapValue
							}
						}
					} else {
						elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
							log.InfoLevel,
							log.Fields{}, "No host staged for Uncapped")
					}
				}
			}
			seconds++
		}

	}(logging, hiThreshold, loThreshold)

	elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
		log.InfoLevel,
		log.Fields{}, "PCP logging started")
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)

	select {
	case <-quit:
		elekLog.ElektronLog.Log(elekLogTypes.GENERAL,
			log.InfoLevel,
			log.Fields{}, "Stopping PCP logging in 5 seconds")
		time.Sleep(5 * time.Second)

		// http://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
		// Kill process and all children processes.
		syscall.Kill(-pgid, 15)
		return
	}

}
