package schedulers

import (
	"gitlab.com/spdf/elektron/def"
	"gitlab.com/spdf/elektron/utilities/mesosUtils"
	"gitlab.com/spdf/elektron/utilities/offerUtils"
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	"log"
)

// Decides if to take an offer or not
func (s *BinPackSortedWatts) takeOffer(spc SchedPolicyContext, offer *mesos.Offer, task def.Task, totalCPU, totalRAM, totalWatts float64) bool {

	baseSchedRef := spc.(*BaseScheduler)
	cpus, mem, watts := offerUtils.OfferAgg(offer)

	//TODO: Insert watts calculation here instead of taking them as a parameter

	wattsConsideration, err := def.WattsToConsider(task, baseSchedRef.classMapWatts, offer)
	if err != nil {
		// Error in determining wattsConsideration.
		log.Fatal(err)
	}
	if (cpus >= (totalCPU + task.CPU)) && (mem >= (totalRAM + task.RAM)) &&
		(!baseSchedRef.wattsAsAResource || (watts >= (totalWatts + wattsConsideration))) {
		return true
	}
	return false
}

type BinPackSortedWatts struct {
	baseSchedPolicyState
}

func (s *BinPackSortedWatts) ConsumeOffers(spc SchedPolicyContext, driver sched.SchedulerDriver, offers []*mesos.Offer) {
	baseSchedRef := spc.(*BaseScheduler)
	if baseSchedRef.schedPolSwitchEnabled {
		SortNTasks(baseSchedRef.tasks, baseSchedRef.numTasksInSchedWindow, def.SortByWatts)
	} else {
		def.SortTasks(baseSchedRef.tasks, def.SortByWatts)
	}
	baseSchedRef.LogOffersReceived(offers)

	for _, offer := range offers {
		offerUtils.UpdateEnvironment(offer)
		select {
		case <-baseSchedRef.Shutdown:
			baseSchedRef.LogNoPendingTasksDeclineOffers(offer)
			driver.DeclineOffer(offer.Id, mesosUtils.LongFilter)
			baseSchedRef.LogNumberOfRunningTasks()
			continue
		default:
		}

		tasks := []*mesos.TaskInfo{}

		offerTaken := false
		totalWatts := 0.0
		totalCPU := 0.0
		totalRAM := 0.0
		for i := 0; i < len(baseSchedRef.tasks); i++ {
			task := baseSchedRef.tasks[i]
			wattsConsideration, err := def.WattsToConsider(task, baseSchedRef.classMapWatts, offer)
			if err != nil {
				// Error in determining wattsConsideration.
				log.Fatal(err)
			}

			// Don't take offer if it doesn't match our task's host requirement.
			if offerUtils.HostMismatch(*offer.Hostname, task.Host) {
				continue
			}

			for *task.Instances > 0 {
				// If scheduling policy switching enabled, then
				// stop scheduling if the #baseSchedRef.schedWindowSize tasks have been scheduled.
				if baseSchedRef.schedPolSwitchEnabled &&
					(s.numTasksScheduled >= baseSchedRef.schedWindowSize) {
					break // Offers will automatically get declined.
				}
				// Does the task fit
				if s.takeOffer(spc, offer, task, totalCPU, totalRAM, totalWatts) {

					offerTaken = true
					totalWatts += wattsConsideration
					totalCPU += task.CPU
					totalRAM += task.RAM
					baseSchedRef.LogCoLocatedTasks(offer.GetSlaveId().GoString())
					taskToSchedule := baseSchedRef.newTask(offer, task)
					tasks = append(tasks, taskToSchedule)

					baseSchedRef.LogSchedTrace(taskToSchedule, offer)
					*task.Instances--
					s.numTasksScheduled++

					if *task.Instances <= 0 {
						// All instances of task have been scheduled, remove it
						baseSchedRef.tasks = append(baseSchedRef.tasks[:i], baseSchedRef.tasks[i+1:]...)

						if len(baseSchedRef.tasks) <= 0 {
							baseSchedRef.LogTerminateScheduler()
							close(baseSchedRef.Shutdown)
						}
					}
				} else {
					break // Continue on to next task
				}
			}
		}

		if offerTaken {
			baseSchedRef.LogTaskStarting(nil, offer)
			LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, driver)
		} else {

			// If there was no match for the task
			cpus, mem, watts := offerUtils.OfferAgg(offer)
			baseSchedRef.LogInsufficientResourcesDeclineOffer(offer, cpus, mem, watts)
			driver.DeclineOffer(offer.Id, mesosUtils.DefaultFilter)
		}
	}
}
