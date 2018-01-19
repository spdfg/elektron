package schedulers

import (
	"bitbucket.org/sunybingcloud/electron/constants"
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
)

/*
Tasks are categorized into small and large tasks based on watts requirements.
All the large tasks are packed into offers from agents belonging to power classes A and B, using Bin-Packing.
All the small tasks are spread among offers from agents belonging to power class C and D, using First-Fit.

Bin-Packing has the most effect when co-scheduling of tasks is increased. Large tasks typically utilize more resources and hence,
co-scheduling them has a great impact on the total power utilization.
*/

func (s *BottomHeavy) takeOfferBinPack(spc SchedPolicyContext, offer *mesos.Offer, totalCPU, totalRAM, totalWatts,
	wattsToConsider float64, task def.Task) bool {
	baseSchedRef := spc.(*baseScheduler)
	offerCPU, offerRAM, offerWatts := offerUtils.OfferAgg(offer)

	//TODO: Insert watts calculation here instead of taking them as a parameter
	if (!baseSchedRef.wattsAsAResource || (offerWatts >= (totalWatts + wattsToConsider))) &&
		(offerCPU >= (totalCPU + task.CPU)) &&
		(offerRAM >= (totalRAM + task.RAM)) {
		return true
	}
	return false

}

func (s *BottomHeavy) takeOfferFirstFit(spc SchedPolicyContext, offer *mesos.Offer,
	wattsConsideration float64, task def.Task) bool {
	baseSchedRef := spc.(*baseScheduler)
	offerCPU, offerRAM, offerWatts := offerUtils.OfferAgg(offer)

	//TODO: Insert watts calculation here instead of taking them as a parameter
	if (!baseSchedRef.wattsAsAResource || (offerWatts >= wattsConsideration)) &&
		(offerCPU >= task.CPU) && (offerRAM >= task.RAM) {
		return true
	}
	return false
}

type BottomHeavy struct {
	SchedPolicyState
	smallTasks, largeTasks []def.Task
}

// Shut down scheduler if no more tasks to schedule
func (s *BottomHeavy) shutDownIfNecessary(spc SchedPolicyContext) {
	baseSchedRef := spc.(*baseScheduler)
	if len(s.smallTasks) <= 0 && len(s.largeTasks) <= 0 {
		baseSchedRef.LogTerminateScheduler()
		close(baseSchedRef.Shutdown)
	}
}

// create TaskInfo and log scheduling trace
func (s *BottomHeavy) createTaskInfoAndLogSchedTrace(spc SchedPolicyContext, offer *mesos.Offer,
	task def.Task) *mesos.TaskInfo {
	baseSchedRef := spc.(*baseScheduler)
	baseSchedRef.LogCoLocatedTasks(offer.GetSlaveId().GoString())
	taskToSchedule := baseSchedRef.newTask(offer, task)

	baseSchedRef.LogSchedTrace(taskToSchedule, offer)
	*task.Instances--
	return taskToSchedule
}

// Using BinPacking to pack large tasks into the given offers.
func (s *BottomHeavy) pack(spc SchedPolicyContext, offers []*mesos.Offer, driver sched.SchedulerDriver) {
	baseSchedRef := spc.(*baseScheduler)
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
		totalWatts := 0.0
		totalCPU := 0.0
		totalRAM := 0.0
		offerTaken := false
		for i := 0; i < len(s.largeTasks); i++ {
			task := s.largeTasks[i]
			wattsConsideration, err := def.WattsToConsider(task, baseSchedRef.classMapWatts, offer)
			if err != nil {
				// Error in determining wattsConsideration
				log.Fatal(err)
			}

			for *task.Instances > 0 {
				// Does the task fit
				// OR lazy evaluation. If ignore watts is set to true, second statement won't
				// be evaluated.
				if s.takeOfferBinPack(spc, offer, totalCPU, totalRAM, totalWatts,
					wattsConsideration, task) {
					offerTaken = true
					totalWatts += wattsConsideration
					totalCPU += task.CPU
					totalRAM += task.RAM
					tasks = append(tasks, s.createTaskInfoAndLogSchedTrace(spc, offer, task))

					if *task.Instances <= 0 {
						// All instances of task have been scheduled, remove it
						s.largeTasks = append(s.largeTasks[:i], s.largeTasks[i+1:]...)
						s.shutDownIfNecessary(spc)
					}
				} else {
					break // Continue on to next task
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
}

// Using First-Fit to spread small tasks among the given offers.
func (s *BottomHeavy) spread(spc SchedPolicyContext, offers []*mesos.Offer, driver sched.SchedulerDriver) {
	baseSchedRef := spc.(*baseScheduler)
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
		taken := false
		for i := 0; i < len(s.smallTasks); i++ {
			task := s.smallTasks[i]
			wattsConsideration, err := def.WattsToConsider(task, baseSchedRef.classMapWatts, offer)
			if err != nil {
				// Error in determining wattsConsideration
				log.Fatal(err)
			}

			// Decision to take the offer or not
			if s.takeOfferFirstFit(spc, offer, wattsConsideration, task) {
				taken = true
				tasks = append(tasks, s.createTaskInfoAndLogSchedTrace(spc, offer, task))
				baseSchedRef.LogTaskStarting(&s.smallTasks[i], offer)
				driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, mesosUtils.DefaultFilter)

				if *task.Instances <= 0 {
					// All instances of task have been scheduled, remove it
					s.smallTasks = append(s.smallTasks[:i], s.smallTasks[i+1:]...)
					s.shutDownIfNecessary(spc)
				}
				break // Offer taken, move on
			}
		}

		if !taken {
			// If there was no match for the task
			cpus, mem, watts := offerUtils.OfferAgg(offer)
			baseSchedRef.LogInsufficientResourcesDeclineOffer(offer, cpus, mem, watts)
			driver.DeclineOffer(offer.Id, mesosUtils.DefaultFilter)
		}
	}
}

func (s *BottomHeavy) ConsumeOffers(spc SchedPolicyContext, driver sched.SchedulerDriver, offers []*mesos.Offer) {
	fmt.Println("BottomHeavy scheduling...")
	baseSchedRef := spc.(*baseScheduler)
	baseSchedRef.LogOffersReceived(offers)
	// Sorting tasks based on Watts
	def.SortTasks(baseSchedRef.tasks, def.SortByWatts)
	// Classification done based on MMPU watts requirements, into 2 clusters.
	classifiedTasks := def.ClassifyTasks(baseSchedRef.tasks, 2)
	// Separating small tasks from large tasks.
	s.smallTasks = classifiedTasks[0].Tasks
	s.largeTasks = classifiedTasks[1].Tasks

	// We need to separate the offers into
	// offers from ClassA and ClassB and offers from ClassC and ClassD.
	// Nodes in ClassA and ClassB will be packed with the large tasks.
	// Small tasks will be spread out among the nodes in ClassC and ClassD.
	offersHeavyPowerClasses := []*mesos.Offer{}
	offersLightPowerClasses := []*mesos.Offer{}

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

		if _, ok := constants.PowerClasses["A"][*offer.Hostname]; ok {
			offersHeavyPowerClasses = append(offersHeavyPowerClasses, offer)
		}
		if _, ok := constants.PowerClasses["B"][*offer.Hostname]; ok {
			offersHeavyPowerClasses = append(offersHeavyPowerClasses, offer)
		}
		if _, ok := constants.PowerClasses["C"][*offer.Hostname]; ok {
			offersLightPowerClasses = append(offersLightPowerClasses, offer)
		}
		if _, ok := constants.PowerClasses["D"][*offer.Hostname]; ok {
			offersLightPowerClasses = append(offersLightPowerClasses, offer)
		}

	}

	buffer := bytes.Buffer{}
	buffer.WriteString(fmt.Sprintln("Packing Large tasks into ClassAB offers:"))
	for _, o := range offersHeavyPowerClasses {
		buffer.WriteString(fmt.Sprintln(*o.Hostname))

	}
	baseSchedRef.Log(elecLogDef.GENERAL, buffer.String())
	buffer.Reset()
	// Packing tasks into offersHeavyPowerClasses
	s.pack(spc, offersHeavyPowerClasses, driver)

	buffer.WriteString(fmt.Sprintln("Packing Small tasks among ClassCD offers:"))
	for _, o := range offersLightPowerClasses {
		buffer.WriteString(*o.Hostname)
	}
	baseSchedRef.Log(elecLogDef.GENERAL, buffer.String())
	// Spreading tasks among offersLightPowerClasses
	s.spread(spc, offersLightPowerClasses, driver)

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
