package schedulers

import (
	"bitbucket.org/sunybingcloud/electron/def"
	"bitbucket.org/sunybingcloud/electron/utilities/mesosUtils"
	"bitbucket.org/sunybingcloud/electron/utilities/offerUtils"
	"fmt"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	"log"
	"os"
	"sort"
	"time"
)

// Decides if to take an offer or not
func (s *BinPackSortedWattsSortedOffers) takeOffer(offer *mesos.Offer, task def.Task,
	totalCPU, totalRAM, totalWatts float64) bool {

	offerCPU, offerRAM, offerWatts := offerUtils.OfferAgg(offer)

	//TODO: Insert watts calculation here instead of taking them as a parameter
	wattsConsideration, err := def.WattsToConsider(task, s.classMapWatts, offer)
	if err != nil {
		// Error in determining wattsConsideration
		log.Fatal(err)
	}
	if (offerCPU >= (totalCPU + task.CPU))&& (offerRAM >= (totalRAM + task.RAM)) &&
		(!s.wattsAsAResource || (offerWatts >= (totalWatts + wattsConsideration))) {
		return true
	}
	return false
}

type BinPackSortedWattsSortedOffers struct {
	base // Type embedded to inherit common functions
}

// New electron scheduler
func NewBinPackSortedWattsSortedOffers(tasks []def.Task, wattsAsAResource bool, schedTracePrefix string,
	classMapWatts bool) *BinPackSortedWattsSortedOffers {
	sort.Sort(def.WattsSorter(tasks))

	logFile, err := os.Create("./" + schedTracePrefix + "_schedTrace.log")
	if err != nil {
		log.Fatal(err)
	}

	s := &BinPackSortedWattsSortedOffers{
		base: base{
			tasks:            tasks,
			wattsAsAResource: wattsAsAResource,
			classMapWatts:    classMapWatts,
			Shutdown:         make(chan struct{}),
			Done:             make(chan struct{}),
			PCPLog:           make(chan struct{}),
			running:          make(map[string]map[string]bool),
			RecordPCP:        false,
			schedTrace:       log.New(logFile, "", log.LstdFlags),
		},
	}
	return s
}

func (s *BinPackSortedWattsSortedOffers) newTask(offer *mesos.Offer, task def.Task) *mesos.TaskInfo {
	taskName := fmt.Sprintf("%s-%d", task.Name, *task.Instances)
	s.tasksCreated++

	if !s.RecordPCP {
		// Turn on logging
		s.RecordPCP = true
		time.Sleep(1 * time.Second) // Make sure we're recording by the time the first task starts
	}

	// If this is our first time running into this Agent
	if _, ok := s.running[offer.GetSlaveId().GoString()]; !ok {
		s.running[offer.GetSlaveId().GoString()] = make(map[string]bool)
	}

	// Add task to list of tasks running on node
	s.running[offer.GetSlaveId().GoString()][taskName] = true

	resources := []*mesos.Resource{
		mesosutil.NewScalarResource("cpus", task.CPU),
		mesosutil.NewScalarResource("mem", task.RAM),
	}

	if s.wattsAsAResource {
		if wattsToConsider, err := def.WattsToConsider(task, s.classMapWatts, offer); err == nil {
			log.Printf("Watts considered for host[%s] and task[%s] = %f", *offer.Hostname, task.Name, wattsToConsider)
			resources = append(resources, mesosutil.NewScalarResource("watts", wattsToConsider))
		} else {
			// Error in determining wattsConsideration
			log.Fatal(err)
		}
	}

	return &mesos.TaskInfo{
		Name: proto.String(taskName),
		TaskId: &mesos.TaskID{
			Value: proto.String("electron-" + taskName),
		},
		SlaveId:   offer.SlaveId,
		Resources: resources,
		Command: &mesos.CommandInfo{
			Value: proto.String(task.CMD),
		},
		Container: &mesos.ContainerInfo{
			Type: mesos.ContainerInfo_DOCKER.Enum(),
			Docker: &mesos.ContainerInfo_DockerInfo{
				Image:   proto.String(task.Image),
				Network: mesos.ContainerInfo_DockerInfo_BRIDGE.Enum(), // Run everything isolated
			},
		},
	}
}

func (s *BinPackSortedWattsSortedOffers) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	log.Printf("Received %d resource offers", len(offers))

	// Sorting the offers
	sort.Sort(offerUtils.OffersSorter(offers))

	// Printing the sorted offers and the corresponding CPU resource availability
	log.Println("Sorted Offers:")
	for i := 0; i < len(offers); i++ {
		offer := offers[i]
		offerCPU, _, _ := offerUtils.OfferAgg(offer)
		log.Printf("Offer[%s].CPU = %f\n", offer.GetHostname(), offerCPU)
	}

	for _, offer := range offers {
		select {
		case <-s.Shutdown:
			log.Println("Done scheduling tasks: declining offer on [", offer.GetHostname(), "]")
			driver.DeclineOffer(offer.Id, mesosUtils.LongFilter)

			log.Println("Number of tasks still running: ", s.tasksRunning)
			continue
		default:
		}

		tasks := []*mesos.TaskInfo{}

		offerTaken := false
		totalWatts := 0.0
		totalCPU := 0.0
		totalRAM := 0.0
		for i := 0; i < len(s.tasks); i++ {
			task := s.tasks[i]
			wattsConsideration, err := def.WattsToConsider(task, s.classMapWatts, offer)
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
				if s.takeOffer(offer, task, totalCPU, totalRAM, totalWatts) {

					offerTaken = true
					totalWatts += wattsConsideration
					totalCPU += task.CPU
					totalRAM += task.RAM
					log.Println("Co-Located with: ")
					coLocated(s.running[offer.GetSlaveId().GoString()])
					taskToSchedule := s.newTask(offer, task)
					tasks = append(tasks, taskToSchedule)

					fmt.Println("Inst: ", *task.Instances)
					s.schedTrace.Print(offer.GetHostname() + ":" + taskToSchedule.GetTaskId().GetValue())
					*task.Instances--

					if *task.Instances <= 0 {
						// All instances of task have been scheduled, remove it
						s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)

						if len(s.tasks) <= 0 {
							log.Println("Done scheduling all tasks")
							close(s.Shutdown)
						}
					}
				} else {
					break // Continue on to next offer
				}
			}
		}

		if offerTaken {
			log.Printf("Starting on [%s]\n", offer.GetHostname())
			driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, mesosUtils.DefaultFilter)
		} else {

			// If there was no match for the task
			fmt.Println("There is not enough resources to launch a task:")
			cpus, mem, watts := offerUtils.OfferAgg(offer)

			log.Printf("<CPU: %f, RAM: %f, Watts: %f>\n", cpus, mem, watts)
			driver.DeclineOffer(offer.Id, mesosUtils.DefaultFilter)
		}
	}
}

func (s *BinPackSortedWattsSortedOffers) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Printf("Received task status [%s] for task [%s]", NameFor(status.State), *status.TaskId.Value)

	if *status.State == mesos.TaskState_TASK_RUNNING {
		s.tasksRunning++
	} else if IsTerminal(status.State) {
		delete(s.running[status.GetSlaveId().GoString()], *status.TaskId.Value)
		s.tasksRunning--
		if s.tasksRunning == 0 {
			select {
			case <-s.Shutdown:
				close(s.Done)
			default:
			}
		}
	}
	log.Printf("DONE: Task status [%s] for task [%s]", NameFor(status.State), *status.TaskId.Value)
}
