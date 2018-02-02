package schedUtils

import (
	"bitbucket.org/sunybingcloud/elektron/def"
	"bitbucket.org/sunybingcloud/elektron/utilities"
	"log"
)

// Criteria for resizing the scheduling window.
type SchedulingWindowResizingCriteria string

var SchedWindowResizingCritToStrategy = map[SchedulingWindowResizingCriteria]SchedWindowResizingStrategy{
	"fillNextOfferCycle": &fillNextOfferCycle{},
}

// Interface for a scheduling window resizing strategy.
type SchedWindowResizingStrategy interface {
	// Apply the window resizing strategy and return the news scheduling window size.
	Apply(func() interface{}) int
}

// Scheduling window resizing strategy that attempts to resize the scheduling window
// to include as many tasks as possible so as to make the most use of the next offer cycle.
type fillNextOfferCycle struct {}

func (s *fillNextOfferCycle) Apply(getArgs func() interface{}) int {
	return s.apply(getArgs().([]def.Task))
}

// Loop over the unscheduled tasks, in submission order, and determine the maximum
// number of tasks that can be scheduled in the next offer cycle.
// As the offers get smaller and smaller, this approach might lead to an increase in internal fragmentation.
//
// Note: To be able to make the most use of the next offer cycle, one would need to perform a non-polynomial search
// which is computationally expensive.
func (s *fillNextOfferCycle) apply(taskQueue []def.Task) int {
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
	for _, task := range taskQueue {
		for i := *task.Instances; i > 0; i-- {
			log.Printf("Checking if Instance #%d of Task[%s] can be scheduled "+
				"during the next offer cycle...", i, task.Name)
			if canSchedule(task) {
				filledCPU += task.CPU
				filledRAM += task.RAM
				newSchedWindow++
			} else {
				done = true
				break
			}
		}
		if done {
			break
		}
	}
	return newSchedWindow
}
