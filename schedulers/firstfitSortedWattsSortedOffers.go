package schedulers

import (
	"bitbucket.org/sunybingcloud/electron/def"
	elecLogDef "bitbucket.org/sunybingcloud/electron/logging/def"
	"bitbucket.org/sunybingcloud/electron/utilities/mesosUtils"
	"bitbucket.org/sunybingcloud/electron/utilities/offerUtils"
	"bytes"
	"fmt"
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	"log"
	"math/rand"
	"sort"
)

// Decides if to take an offer or not
func (s *FirstFitSortedWattsSortedOffers) takeOffer(spc SchedPolicyContext, offer *mesos.Offer, task def.Task) bool {
	baseSchedRef := spc.(*baseScheduler)
	cpus, mem, watts := offerUtils.OfferAgg(offer)

	//TODO: Insert watts calculation here instead of taking them as a parameter

	wattsConsideration, err := def.WattsToConsider(task, baseSchedRef.classMapWatts, offer)
	if err != nil {
		// Error in determining wattsConsideration
		log.Fatal(err)
	}
	if cpus >= task.CPU && mem >= task.RAM && (!baseSchedRef.wattsAsAResource || watts >= wattsConsideration) {
		return true
	}

	return false
}

// electronScheduler implements the Scheduler interface
type FirstFitSortedWattsSortedOffers struct {
	SchedPolicyState
}

func (s *FirstFitSortedWattsSortedOffers) ConsumeOffers(spc SchedPolicyContext, driver sched.SchedulerDriver,
	offers []*mesos.Offer) {
	fmt.Println("FFSWSO scheduling...")
	baseSchedRef := spc.(*baseScheduler)
	def.SortTasks(baseSchedRef.tasks, def.SortByWatts)
	baseSchedRef.LogOffersReceived(offers)
	// Sorting the offers
	sort.Sort(offerUtils.OffersSorter(offers))

	// Printing the sorted offers and the corresponding CPU resource availability
	buffer := bytes.Buffer{}
	buffer.WriteString(fmt.Sprintln("Sorted Offers:"))
	for i := 0; i < len(offers); i++ {
		offer := offers[i]
		offerUtils.UpdateEnvironment(offer)
		offerCPU, _, _ := offerUtils.OfferAgg(offer)
		buffer.WriteString(fmt.Sprintf("Offer[%s].CPU = %f\n", offer.GetHostname(), offerCPU))
	}
	baseSchedRef.Log(elecLogDef.GENERAL, buffer.String())

	for _, offer := range offers {
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

			// Don't take offer if it doesn't match our task's host requirement
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
					baseSchedRef.tasks = append(baseSchedRef.tasks[:i],
						baseSchedRef.tasks[i+1:]...)

					if len(baseSchedRef.tasks) <= 0 {
						baseSchedRef.LogTerminateScheduler()
						close(baseSchedRef.Shutdown)
					}
				}
				break // Offer taken, move on
			}
		}

		// If there was no match for the task
		if !offerTaken {
			cpus, mem, watts := offerUtils.OfferAgg(offer)
			baseSchedRef.LogInsufficientResourcesDeclineOffer(offer, cpus, mem, watts)
			driver.DeclineOffer(offer.Id, mesosUtils.DefaultFilter)
		}
	}

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
