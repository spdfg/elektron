package schedulers

import (
	"bitbucket.org/sunybingcloud/elektron/def"
	"bitbucket.org/sunybingcloud/elektron/utilities/mesosUtils"
	"bitbucket.org/sunybingcloud/elektron/utilities/offerUtils"
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	"log"
	"math/rand"
)

// Decides if to take an offer or not
func (s *MaxMin) takeOffer(spc SchedPolicyContext, offer *mesos.Offer, task def.Task,
	totalCPU, totalRAM, totalWatts float64) bool {
	baseSchedRef := spc.(*BaseScheduler)
	cpus, mem, watts := offerUtils.OfferAgg(offer)

	//TODO: Insert watts calculation here instead of taking them as a parameter

	wattsConsideration, err := def.WattsToConsider(task, baseSchedRef.classMapWatts, offer)
	if err != nil {
		// Error in determining wattsConsideration
		log.Fatal(err)
	}
	if (cpus >= (totalCPU + task.CPU)) && (mem >= (totalRAM + task.RAM)) &&
		(!baseSchedRef.wattsAsAResource || (watts >= (totalWatts + wattsConsideration))) {
		return true
	}
	return false
}

type MaxMin struct {
	baseSchedPolicyState
}

// Determine if the remaining space inside of the offer is enough for this
// task that we need to create. If it is, create a TaskInfo and return it.
func (s *MaxMin) CheckFit(
	spc SchedPolicyContext,
	i int,
	task def.Task,
	wattsConsideration float64,
	offer *mesos.Offer,
	totalCPU *float64,
	totalRAM *float64,
	totalWatts *float64) (bool, *mesos.TaskInfo) {

	baseSchedRef := spc.(*BaseScheduler)
	// Does the task fit.
	if s.takeOffer(spc, offer, task, *totalCPU, *totalRAM, *totalWatts) {

		*totalWatts += wattsConsideration
		*totalCPU += task.CPU
		*totalRAM += task.RAM
		baseSchedRef.LogCoLocatedTasks(offer.GetSlaveId().GoString())

		taskToSchedule := baseSchedRef.newTask(offer, task)

		baseSchedRef.LogSchedTrace(taskToSchedule, offer)
		*task.Instances--

		if *task.Instances <= 0 {
			// All instances of task have been scheduled, remove it.
			baseSchedRef.tasks = append(baseSchedRef.tasks[:i], baseSchedRef.tasks[i+1:]...)

			if len(baseSchedRef.tasks) <= 0 {
				baseSchedRef.LogTerminateScheduler()
				close(baseSchedRef.Shutdown)
			}
		}

		return true, taskToSchedule
	}
	return false, nil
}

func (s *MaxMin) ConsumeOffers(spc SchedPolicyContext, driver sched.SchedulerDriver, offers []*mesos.Offer) {
	log.Println("Max-Min scheduling...")
	baseSchedRef := spc.(*BaseScheduler)
	def.SortTasks(baseSchedRef.tasks, def.SortByWatts)
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

		// Assumes s.tasks is ordered in non-decreasing median max-peak order

		// Attempt to schedule a single instance of the heaviest workload available first.
		// Start from the back until one fits.

		direction := false // True = Min Max, False = Max Min
		var index int
		start := true // If false then index has changed and need to keep it that way
		for i := 0; i < len(baseSchedRef.tasks); i++ {
			// We need to pick a min task or a max task
			// depending on the value of direction.
			if direction && start {
				index = 0
			} else if start {
				index = len(baseSchedRef.tasks) - i - 1
			}
			task := baseSchedRef.tasks[index]

			wattsConsideration, err := def.WattsToConsider(task, baseSchedRef.classMapWatts, offer)
			if err != nil {
				// Error in determining wattsConsideration.
				log.Fatal(err)
			}

			// Don't take offer if it doesn't match our task's host requirement.
			if offerUtils.HostMismatch(*offer.Hostname, task.Host) {
				continue
			}

			// TODO: Fix this so index doesn't need to be passed.
			taken, taskToSchedule := s.CheckFit(spc, index, task, wattsConsideration, offer,
				&totalCPU, &totalRAM, &totalWatts)

			if taken {
				offerTaken = true
				tasks = append(tasks, taskToSchedule)
				// Need to change direction and set start to true.
				// Setting start to true would ensure that index be set accurately again.
				direction = !direction
				start = true
				i--
			} else {
				// Need to move index depending on the value of direction.
				if direction {
					index++
					start = false
				} else {
					index--
					start = false
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

	// Switch scheduling policy only if feature enabled from CLI
	if baseSchedRef.schedPolSwitchEnabled {
		// Need to recompute the schedWindow for the next offer cycle.
		// The next scheduling policy will schedule at max schedWindow number of tasks.
		baseSchedRef.curSchedWindow = baseSchedRef.schedWindowResStrategy.Apply(
			func() interface{} { return baseSchedRef.tasks })
		// Switching to a random scheduling policy.
		// TODO: Switch based on some criteria.
		index := rand.Intn(len(SchedPolicies))
		for k, v := range SchedPolicies {
			if index == 0 {
				baseSchedRef.LogSchedPolicySwitch(k, v)
				spc.SwitchSchedPol(v)
				break
			}
			index--
		}
	}
}
