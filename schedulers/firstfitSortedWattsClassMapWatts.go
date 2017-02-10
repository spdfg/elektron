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
func (s *FirstFitSortedWattsClassMapWatts) takeOffer(offer *mesos.Offer, powerClass string, task def.Task) bool {
	offerCPU, offerRAM, offerWatts := offerUtils.OfferAgg(offer)

	//TODO: Insert watts calculation here instead of taking them as a parameter
	// Decision to take the offer or not
	if (s.ignoreWatts || (offerWatts >= task.ClassToWatts[powerClass])) &&
		(offerCPU >= task.CPU) && (offerRAM >= task.RAM) {
		return true
	}
	return false
}

// electron scheduler implements the Scheduler interface
type FirstFitSortedWattsClassMapWatts struct {
	base         // Type embedded to inherit common features.
	tasksCreated int
	tasksRunning int
	tasks        []def.Task
	metrics      map[string]def.Metric
	running      map[string]map[string]bool
	ignoreWatts  bool

	// First set of PCP values are garbage values, signal to logger to start recording when we're
	// about to schedule a new task
	RecordPCP bool

	// This channel is closed when the program receives an interrupt,
	// signalling that the program should shut down.
	Shutdown chan struct{}
	// This channel is closed after shutdown is closed, and only when all
	// outstanding tasks have been cleaned up
	Done chan struct{}

	// Controls when to shutdown pcp logging
	PCPLog chan struct{}

	schedTrace *log.Logger
}

// New electorn scheduler
func NewFirstFitSortedWattsClassMapWatts(tasks []def.Task, ignoreWatts bool, schedTracePrefix string) *FirstFitSortedWattsClassMapWatts {
	sort.Sort(def.WattsSorter(tasks))

	logFile, err := os.Create("./" + schedTracePrefix + "_schedTrace.log")
	if err != nil {
		log.Fatal(err)
	}

	s := &FirstFitSortedWattsClassMapWatts{
		tasks:       tasks,
		ignoreWatts: ignoreWatts,
		Shutdown:    make(chan struct{}),
		Done:        make(chan struct{}),
		PCPLog:      make(chan struct{}),
		running:     make(map[string]map[string]bool),
		RecordPCP:   false,
		schedTrace:  log.New(logFile, "", log.LstdFlags),
	}
	return s
}

func (s *FirstFitSortedWattsClassMapWatts) newTask(offer *mesos.Offer, task def.Task, powerClass string) *mesos.TaskInfo {
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

	if !s.ignoreWatts {
		resources = append(resources, mesosutil.NewScalarResource("watts", task.ClassToWatts[powerClass]))
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

func (s *FirstFitSortedWattsClassMapWatts) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	log.Printf("Received %d resource offers", len(offers))

	for _, offer := range offers {
		select {
		case <-s.Shutdown:
			log.Println("Done scheduling tasks: declining offer on [", offer.GetHostname(), "]")
			driver.DeclineOffer(offer.Id, mesosUtils.LongFilter)

			log.Println("Number of tasks still running: ", s.tasksRunning)
			continue
		default:
		}

		// First fit strategy
		offerTaken := false
		for i := 0; i < len(s.tasks); i++ {
			task := s.tasks[i]
			// Don't take offer if it doesn't match our task's host requirement
			if offerUtils.HostMismatch(*offer.Hostname, task.Host) {
				continue
			}

			// retrieving the powerClass from the offer
			powerClass := offerUtils.PowerClass(offer)

			// Decision to take the offer or not
			if s.takeOffer(offer, powerClass, task) {
				fmt.Println("Watts being used: ", task.ClassToWatts[powerClass])
				log.Println("Co-Located with: ")
				coLocated(s.running[offer.GetSlaveId().GoString()])

				taskToSchedule := s.newTask(offer, task, powerClass)
				s.schedTrace.Print(offer.GetHostname() + ":" + taskToSchedule.GetTaskId().GetValue())
				log.Printf("Starting %s on [%s]\n", task.Name, offer.GetHostname())
				driver.LaunchTasks([]*mesos.OfferID{offer.Id}, []*mesos.TaskInfo{taskToSchedule}, mesosUtils.DefaultFilter)

				offerTaken = true
				fmt.Println("Inst: ", *task.Instances)
				*task.Instances--
				if *task.Instances <= 0 {
					// All instances of task have been scheduled, remove it
					s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)

					if len(s.tasks) == 0 {
						log.Println("Done scheduling all tasks")
						close(s.Shutdown)
					}
				}
				break // Offer taken, move on
			}
		}

		// If there was no match for the task
		if !offerTaken {
			fmt.Println("There is not enough resources to launch a task:")
			cpus, mem, watts := offerUtils.OfferAgg(offer)

			log.Printf("<CPU: %f, RAM: %f, Watts: %f>\n", cpus, mem, watts)
			driver.DeclineOffer(offer.Id, mesosUtils.DefaultFilter)

		}
	}
}

func (s *FirstFitSortedWattsClassMapWatts) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
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
