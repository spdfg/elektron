package schedulers

import (
	"bitbucket.org/sunybingcloud/elektron/def"
	"bitbucket.org/sunybingcloud/elektron/utilities/mesosUtils"
	"bitbucket.org/sunybingcloud/elektron/utilities/offerUtils"
	"fmt"
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	"math/rand"
)

// Decides if to take an offer or not
func (s *FirstFit) takeOffer(spc SchedPolicyContext, offer *mesos.Offer, task def.Task) bool {
	baseSchedRef := spc.(*baseScheduler)
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
	SchedPolicyState
}

func (s *FirstFit) ConsumeOffers(spc SchedPolicyContext, driver sched.SchedulerDriver, offers []*mesos.Offer) {
	fmt.Println("FirstFit scheduling...")
	baseSchedRef := spc.(*baseScheduler)
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
				driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, mesosUtils.DefaultFilter)

				offerTaken = true

				baseSchedRef.LogSchedTrace(taskToSchedule, offer)
				*task.Instances--

				if *task.Instances <= 0 {
					// All instances of task have been scheduled, remove it
					baseSchedRef.tasks[i] = baseSchedRef.tasks[len(baseSchedRef.tasks)-1]
					baseSchedRef.tasks = baseSchedRef.tasks[:len(baseSchedRef.tasks)-1]

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

	// Switch scheduling policy only if feature enabled from CLI
	if baseSchedRef.schedPolSwitchEnabled {
		// Switching to a random scheduling policy.
		// TODO: Switch based on some criteria.
		index := rand.Intn(len(SchedPolicies))
		for _, v := range SchedPolicies {
			if index == 0 {
				spc.SwitchSchedPol(v)
				break
			}
			index--
		}
	}
}
