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
func (s *BinPackSortedWattsSortedOffers) takeOffer(spc SchedPolicyContext, offer *mesos.Offer, task def.Task,
	totalCPU, totalRAM, totalWatts float64) bool {
	baseSchedRef := spc.(*baseScheduler)
	offerCPU, offerRAM, offerWatts := offerUtils.OfferAgg(offer)

	//TODO: Insert watts calculation here instead of taking them as a parameter
	wattsConsideration, err := def.WattsToConsider(task, baseSchedRef.classMapWatts, offer)
	if err != nil {
		// Error in determining wattsConsideration
		log.Fatal(err)
	}
	if (offerCPU >= (totalCPU + task.CPU)) && (offerRAM >= (totalRAM + task.RAM)) &&
		(!baseSchedRef.wattsAsAResource || (offerWatts >= (totalWatts + wattsConsideration))) {
		return true
	}
	return false
}

type BinPackSortedWattsSortedOffers struct {
	SchedPolicyState
}

func (s *BinPackSortedWattsSortedOffers) ConsumeOffers(spc SchedPolicyContext, driver sched.SchedulerDriver, offers []*mesos.Offer) {
	fmt.Println("BPSWSO scheduling...")
	baseSchedRef := spc.(*baseScheduler)
	def.SortTasks(baseSchedRef.tasks, def.SortByWatts)
	baseSchedRef.LogOffersReceived(offers)

	// Sorting the offers
	sort.Sort(offerUtils.OffersSorter(offers))

	// Printing the sorted offers and the corresponding CPU resource availability
	buffer := bytes.Buffer{}
	buffer.WriteString(fmt.Sprint("Sorted Offers:\n"))
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

		offerTaken := false
		totalWatts := 0.0
		totalCPU := 0.0
		totalRAM := 0.0
		for i := 0; i < len(baseSchedRef.tasks); i++ {
			task := baseSchedRef.tasks[i]
			wattsConsideration, err := def.WattsToConsider(task, baseSchedRef.classMapWatts, offer)
			if err != nil {
				// Error in determining wattsConsideration
				log.Fatal(err)
			}

			// Don't take offer if it doesn't match our task's host requirement
			if offerUtils.HostMismatch(*offer.Hostname, task.Host) {
				continue
			}

			for *task.Instances > 0 {
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

					if *task.Instances <= 0 {
						// All instances of task have been scheduled, remove it
						baseSchedRef.tasks = append(baseSchedRef.tasks[:i],
							baseSchedRef.tasks[i+1:]...)

						if len(baseSchedRef.tasks) <= 0 {
							baseSchedRef.LogTerminateScheduler()
							close(baseSchedRef.Shutdown)
						}
					}
				} else {
					break // Continue on to next offer
				}
			}
		}

		if offerTaken {
			baseSchedRef.LogTaskStarting(nil, offer)
			driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, mesosUtils.DefaultFilter)
		} else {

			// If there was no match for the task
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
