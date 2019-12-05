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

package schedUtils

import (
	log "github.com/sirupsen/logrus"
	"github.com/spdfg/elektron/def"
	elekLog "github.com/spdfg/elektron/logging"
	elekLogTypes "github.com/spdfg/elektron/logging/types"
	"github.com/spdfg/elektron/utilities"
)

// Criteria for resizing the scheduling window.
type SchedulingWindowResizingCriteria string

var SchedWindowResizingCritToStrategy = map[SchedulingWindowResizingCriteria]SchedWindowResizingStrategy{
	"fillNextOfferCycle": &fillNextOfferCycle{},
}

// Interface for a scheduling window resizing strategy.
type SchedWindowResizingStrategy interface {
	// Apply the window resizing strategy and return the size of the scheduling window and the number tasks that
	// were traversed in the process.
	// The size of the scheduling window would correspond to the total number of
	// instances (flattened) that can be scheduled in the next offer cycle.
	// The number of tasks would correspond to number of different tasks (instances not included).
	Apply(func() interface{}) (int, int)
}

// Scheduling window resizing strategy that attempts to resize the scheduling window
// to include as many tasks as possible so as to make the most use of the next offer cycle.
type fillNextOfferCycle struct{}

func (s *fillNextOfferCycle) Apply(getArgs func() interface{}) (int, int) {
	return s.apply(getArgs().([]def.Task))
}

// Loop over the unscheduled tasks, in submission order, and determine the maximum
// number of tasks that can be scheduled in the next offer cycle.
// As the offers get smaller and smaller, this approach might lead to an increase in internal fragmentation.
//
// Note: To be able to make the most use of the next offer cycle, one would need to perform a non-polynomial search
// which is computationally expensive.
func (s *fillNextOfferCycle) apply(taskQueue []def.Task) (int, int) {
	clusterwideResourceCount := utilities.GetClusterwideResourceAvailability()
	newSchedWindow := 0
	filledCPU := 0.0
	filledRAM := 0.0

	// Can we schedule another task.
	canSchedule := func(t def.Task) bool {
		if ((filledCPU + t.CPU) <= clusterwideResourceCount.UnusedCPU) &&
			((filledRAM + t.RAM) <= clusterwideResourceCount.UnusedRAM) {
			return true
		}
		return false
	}

	done := false
	// Track of number of tasks traversed.
	numberOfTasksTraversed := 0
	for _, task := range taskQueue {
		numberOfTasksTraversed++
		for i := *task.Instances; i > 0; i-- {
			elekLog.ElektronLogger.Logf(elekLogTypes.CONSOLE, log.InfoLevel,
				"Checking if Instance #%d of Task[%s] can be scheduled "+
					"during the next offer cycle...", i, task.Name)
			if canSchedule(task) {
				filledCPU += task.CPU
				filledRAM += task.RAM
				newSchedWindow++
			} else {
				done = true
				if i == *task.Instances {
					// We don't count this task if none of the instances could be scheduled.
					numberOfTasksTraversed--
				}
				break
			}
		}
		if done {
			break
		}
	}
	// Hacking...
	// 2^window is window<=7
	//	if newSchedWindow <= 7 {
	//		newSchedWindow = int(math.Pow(2.0, float64(newSchedWindow)))
	//	}
	// Another hack. Getting rid of window to see whether the idle power consumption can be amortized.
	// Setting window as the length of the entire queue.
	// Also setting numberOfTasksTraversed to the number of tasks in the entire queue.
	// TODO: Create another resizing strategy that sizes the window to the length of the entire pending queue.
	//	flattenedLength := 0
	//	numTasks := 0
	//	for _, ts := range taskQueue {
	//		numTasks++
	//		flattenedLength += *ts.Instances
	//	}
	//	newSchedWindow = flattenedLength
	//	numberOfTasksTraversed = numTasks

	return newSchedWindow, numberOfTasksTraversed
}
