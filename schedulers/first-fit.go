package schedulers

import (
	"bitbucket.org/sunybingcloud/elektron/def"
	"bitbucket.org/sunybingcloud/elektron/utilities/mesosUtils"
	"bitbucket.org/sunybingcloud/elektron/utilities/offerUtils"
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	"log"
)

// Decides if to take an offer or not
func (s *FirstFit) takeOffer(spc SchedPolicyContext, offer *mesos.Offer, task def.Task) bool {
	baseSchedRef := spc.(*BaseScheduler)
	cpus, mem, watts := offerUtils.OfferAgg(offer)

	//TODO: Insert watts calculation here instead of taking them as a parameter

	wattsConsideration, err := def.WattsToConsider(task, baseSchedRef.classMapWatts, offer)
	if err != nil {
		// Error in determining wattsConsideration
		baseSchedRef.LogElectronError(err)
	}
	if cpus >= task.CPU && mem >= task.RAM && (!baseSchedRef.wattsAsAResource || watts >= wattsConsideration) {
		return true
	}

	return false
}

// Elektron scheduler implements the Scheduler interface.
type FirstFit struct {
	baseSchedPolicyState
}

func (s *FirstFit) ConsumeOffers(spc SchedPolicyContext, driver sched.SchedulerDriver, offers []*mesos.Offer) {
	baseSchedRef := spc.(*BaseScheduler)
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

		// First fit strategy
		offerTaken := false
		for i := 0; i < len(baseSchedRef.tasks); i++ {
			// If scheduling policy switching enabled, then
			// stop scheduling if the #baseSchedRef.schedWindowSize tasks have been scheduled.
			if baseSchedRef.schedPolSwitchEnabled && (s.numTasksScheduled >= baseSchedRef.schedWindowSize) {
				break // Offers will automatically get declined.
			}
			task := baseSchedRef.tasks[i]

			// Don't take offer if it doesn't match our task's host requirement.
			if offerUtils.HostMismatch(*offer.Hostname, task.Host) {
				continue
			}

			// Decision to take the offer or not
			if s.takeOffer(spc, offer, task) {

				baseSchedRef.LogCoLocatedTasks(offer.GetSlaveId().GoString())

				taskToSchedule := baseSchedRef.newTask(offer, task)
				tasks = append(tasks, taskToSchedule)

				baseSchedRef.LogTaskStarting(&task, offer)
				LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, driver)
				offerTaken = true

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
				break // Offer taken, move on.
			}
		}

		// If there was no match for the task.
		if !offerTaken {
			cpus, mem, watts := offerUtils.OfferAgg(offer)
			baseSchedRef.LogInsufficientResourcesDeclineOffer(offer, cpus, mem, watts)
			driver.DeclineOffer(offer.Id, mesosUtils.DefaultFilter)
		}
	}
}
