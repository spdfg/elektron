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

package schedulers

import (
	"fmt"
	"time"

	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	log "github.com/sirupsen/logrus"
	"github.com/spdfg/elektron/def"
	elekLog "github.com/spdfg/elektron/logging"
	. "github.com/spdfg/elektron/logging/types"
)

type SchedPolicyContext interface {
	// Change the state of scheduling.
	SwitchSchedPol(s SchedPolicyState)
}

type SchedPolicyState interface {
	// Define the particular scheduling policy's methodology of resource offer consumption.
	ConsumeOffers(SchedPolicyContext, sched.SchedulerDriver, []*mesos.Offer)
	// Get information about the scheduling policy.
	GetInfo() (info struct {
		taskDist       float64
		varCpuShare    float64
		nextPolicyName string
		prevPolicyName string
	})
	// Update links to next and previous scheduling policy.
	UpdateLinks(info struct {
		nextPolicyName string
		prevPolicyName string
	})
	// Switch scheduling policy if necessary.
	SwitchIfNecessary(SchedPolicyContext)
}

type baseSchedPolicyState struct {
	SchedPolicyState
	// Keep track of the number of tasks that have been scheduled.
	numTasksScheduled int
	// Distribution of tasks that the scheduling policy is most appropriate for.
	// This distribution corresponds to the ratio of low power consuming tasks to
	// high power consuming tasks.
	TaskDistribution float64 `json:"taskDist"`
	// The average variance in cpu-share per task that this scheduling policy can cause.
	// Note: This number corresponds to a given workload.
	VarianceCpuSharePerTask float64 `json:"varCpuShare"`
	// Next and Previous scheduling policy in round-robin order.
	// This order is determined by the sorted order (non-decreasing or non-increasing) of taskDistribution.
	nextPolicyName string
	prevPolicyName string
}

// Scheduling policy switching criteria.
// Takes a pointer to the BaseScheduler and returns the name of the scheduling policy to switch to.
type switchBy func(*BaseScheduler) string

var switchBasedOn map[string]switchBy = map[string]switchBy{
	"taskDist":        switchTaskDistBased,
	"round-robin":     switchRoundRobinBased,
	"rev-round-robin": switchRevRoundRobinBased,
}

func switchTaskDistBased(baseSchedRef *BaseScheduler) string {
	// Name of the scheduling policy to switch to.
	switchToPolicyName := ""
	// Record overhead to classify the tasks in the scheduling window and using the classification results
	// 	to determine the distribution of low power consuming and high power consuming tasks.
	startTime := time.Now()
	// Determine the distribution of tasks in the new scheduling window.
	taskDist, err := def.GetTaskDistributionInWindow(baseSchedRef.schedWindowSize, baseSchedRef.tasks)
	baseSchedRef.LogClsfnAndTaskDistOverhead(time.Now().Sub(startTime))
	elekLog.WithFields(log.Fields{"Task Distribution": fmt.Sprintf("%f", taskDist)}).Log(CONSOLE,
		log.InfoLevel, "Switching... ")
	if err != nil {
		// All the tasks in the window were only classified into 1 cluster.
		// Max-Min and Max-GreedyMins would work the same way as Bin-Packing for this situation.
		// So, we have 2 choices to make. First-Fit or Bin-Packing.
		// If choose Bin-Packing, then there might be a performance degradation due to increase in
		// 	resource contention. So, First-Fit might be a better option to cater to the worst case
		// 	where all the tasks are power intensive tasks.
		// TODO: Another possibility is to do the exact opposite and choose Bin-Packing.
		// TODO[2]: Determine scheduling policy based on the distribution of tasks in the whole queue.
		switchToPolicyName = bp
	} else {
		// The tasks in the scheduling window were classified into 2 clusters, meaning that there is
		// 	some variety in the kind of tasks.
		// We now select the scheduling policy which is most appropriate for this distribution of tasks.
		first := schedPoliciesToSwitch[0]
		last := schedPoliciesToSwitch[len(schedPoliciesToSwitch)-1]
		if taskDist < first.sp.GetInfo().taskDist {
			switchToPolicyName = first.spName
		} else if taskDist > last.sp.GetInfo().taskDist {
			switchToPolicyName = last.spName
		} else {
			low := 0
			high := len(schedPoliciesToSwitch) - 1
			for low <= high {
				mid := (low + high) / 2
				if taskDist < schedPoliciesToSwitch[mid].sp.GetInfo().taskDist {
					high = mid - 1
				} else if taskDist > schedPoliciesToSwitch[mid].sp.GetInfo().taskDist {
					low = mid + 1
				} else {
					switchToPolicyName = schedPoliciesToSwitch[mid].spName
					break
				}
			}
			// We're here if low == high+1.
			// If haven't yet found the closest match.
			if switchToPolicyName == "" {
				lowDiff := schedPoliciesToSwitch[low].sp.GetInfo().taskDist - taskDist
				highDiff := taskDist - schedPoliciesToSwitch[high].sp.GetInfo().taskDist
				if lowDiff > highDiff {
					switchToPolicyName = schedPoliciesToSwitch[high].spName
				} else if highDiff > lowDiff {
					switchToPolicyName = schedPoliciesToSwitch[low].spName
				} else {
					// index doens't matter as the values at high and low are equidistant
					// 	from taskDist.
					switchToPolicyName = schedPoliciesToSwitch[high].spName
				}
			}
		}
	}
	return switchToPolicyName
}

// Switching based on a round-robin approach.
// Not being considerate to the state of TaskQueue or the state of the cluster.
func switchRoundRobinBased(baseSchedRef *BaseScheduler) string {
	// If haven't received any resource offers.
	if !baseSchedRef.hasReceivedResourceOffers {
		return schedPoliciesToSwitch[0].spName
	}
	return baseSchedRef.curSchedPolicy.GetInfo().nextPolicyName
}

// Switching based on a round-robin approach, but in the reverse order.
// Not being considerate to the state of the TaskQueue or the state of the cluster.
func switchRevRoundRobinBased(baseSchedRef *BaseScheduler) string {
	// If haven't received any resource offers.
	if !baseSchedRef.hasReceivedResourceOffers {
		return schedPoliciesToSwitch[len(schedPoliciesToSwitch)-1].spName
	}
	return baseSchedRef.curSchedPolicy.GetInfo().prevPolicyName
}

func (bsps *baseSchedPolicyState) SwitchIfNecessary(spc SchedPolicyContext) {
	baseSchedRef := spc.(*BaseScheduler)
	// Switching scheduling policy only if feature enabled from CLI.
	if baseSchedRef.schedPolSwitchEnabled {
		// Name of scheduling policy to switch to.
		switchToPolicyName := ""
		// Size of the new scheduling window.
		newSchedWindowSize := 0
		// If scheduling window has not been fixed, then determine the scheduling window based on the current
		// 	availability of resources on the cluster (Mesos perspective).
		if !baseSchedRef.toFixSchedWindow {
			// Need to compute the size of the scheduling window.
			// The next scheduling policy will schedule at max schedWindowSize number of tasks.
			newSchedWindowSize, baseSchedRef.numTasksInSchedWindow =
				baseSchedRef.schedWindowResStrategy.Apply(func() interface{} { return baseSchedRef.tasks })
		}

		// Now, we need to switch if the new scheduling window is > 0.
		if (!baseSchedRef.toFixSchedWindow && (newSchedWindowSize > 0)) ||
			(baseSchedRef.toFixSchedWindow && (baseSchedRef.schedWindowSize > 0)) {
			// If we haven't received any resource offers, then
			// 	check whether we need to fix the first scheduling policy to deploy.
			// 	If not, then determine the first scheduling policy based on the distribution of tasks
			//		in the scheduling window.
			// Else,
			// 	Check whether the currently deployed scheduling policy has already scheduled the
			// 		schedWindowSize number of tasks.
			// 	If yes, then we switch to the scheduling policy based on the distribution of tasks in
			//		the scheduling window.
			// 	If not, then we continue to use the currently deployed scheduling policy.
			if !baseSchedRef.hasReceivedResourceOffers {
				if baseSchedRef.nameOfFstSchedPolToDeploy != "" {
					switchToPolicyName = baseSchedRef.nameOfFstSchedPolToDeploy
					if !baseSchedRef.toFixSchedWindow {
						baseSchedRef.schedWindowSize = newSchedWindowSize
					}
				} else {
					// Decided to switch, so updating the window size.
					if !baseSchedRef.toFixSchedWindow {
						baseSchedRef.schedWindowSize = newSchedWindowSize
					}
					switchToPolicyName = switchBasedOn[baseSchedRef.schedPolSwitchCriteria](baseSchedRef)
				}
			} else {
				// Checking if the currently deployed scheduling policy has scheduled all the tasks in the window.
				if bsps.numTasksScheduled >= baseSchedRef.schedWindowSize {
					// Decided to switch, so updating the window size.
					if !baseSchedRef.toFixSchedWindow {
						baseSchedRef.schedWindowSize = newSchedWindowSize
					}
					switchToPolicyName = switchBasedOn[baseSchedRef.schedPolSwitchCriteria](baseSchedRef)
				} else {
					// We continue working with the currently deployed scheduling policy.
					log.Println("Continuing with the current scheduling policy...")
					log.Printf("TasksScheduled[%d], SchedWindowSize[%d]", bsps.numTasksScheduled,
						baseSchedRef.schedWindowSize)
					return
				}
			}
			// Switching scheduling policy.
			baseSchedRef.LogSchedPolicySwitch(switchToPolicyName, SchedPolicies[switchToPolicyName])
			baseSchedRef.SwitchSchedPol(SchedPolicies[switchToPolicyName])
			// Resetting the number of tasks scheduled as this is a new scheduling policy that has been
			// 	deployed.
			bsps.numTasksScheduled = 0
		} else {
			// We continue working with the currently deployed scheduling policy.
			log.Println("Continuing with the current scheduling policy...")
			log.Printf("TasksScheduled[%d], SchedWindowSize[%d]", bsps.numTasksScheduled,
				baseSchedRef.schedWindowSize)
			return
		}
	}
}

func (bsps *baseSchedPolicyState) GetInfo() (info struct {
	taskDist       float64
	varCpuShare    float64
	nextPolicyName string
	prevPolicyName string
}) {
	info.taskDist = bsps.TaskDistribution
	info.varCpuShare = bsps.VarianceCpuSharePerTask
	info.nextPolicyName = bsps.nextPolicyName
	info.prevPolicyName = bsps.prevPolicyName
	return info
}

func (bsps *baseSchedPolicyState) UpdateLinks(info struct {
	nextPolicyName string
	prevPolicyName string
}) {
	bsps.nextPolicyName = info.nextPolicyName
	bsps.prevPolicyName = info.prevPolicyName
}
