package schedulers

import (
	"bitbucket.org/sunybingcloud/electron/def"
	"fmt"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

// Decides if to take an offer or not
func (*BinPackSortedWatts) takeOffer(offer *mesos.Offer, task def.Task) bool {

	cpus, mem, watts := OfferAgg(offer)

	//TODO: Insert watts calculation here instead of taking them as a parameter

	if cpus >= task.CPU && mem >= task.RAM && watts >= task.Watts {
		return true
	}

	return false
}

type BinPackSortedWatts struct {
	base         // Type embedded to inherit common functions
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

// New electron scheduler
func NewBinPackSortedWatts(tasks []def.Task, ignoreWatts bool, schedTracePrefix string) *BinPackSortedWatts {
	sort.Sort(def.WattsSorter(tasks))

	logFile, err := os.Create("./" + schedTracePrefix + "_schedTrace.log")
	if err != nil {
		log.Fatal(err)
	}

	s := &BinPackSortedWatts{
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

func (s *BinPackSortedWatts) newTask(offer *mesos.Offer, task def.Task) *mesos.TaskInfo {
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
		resources = append(resources, mesosutil.NewScalarResource("watts", task.Watts))
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

func (s *BinPackSortedWatts) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	log.Printf("Received %d resource offers", len(offers))

	for _, offer := range offers {
		select {
		case <-s.Shutdown:
			log.Println("Done scheduling tasks: declining offer on [", offer.GetHostname(), "]")
			driver.DeclineOffer(offer.Id, longFilter)

			log.Println("Number of tasks still running: ", s.tasksRunning)
			continue
		default:
		}

		tasks := []*mesos.TaskInfo{}

		offer_cpu, offer_ram, offer_watts := OfferAgg(offer)

		taken := false
		totalWatts := 0.0
		totalCPU := 0.0
		totalRAM := 0.0
		for i := 0; i < len(s.tasks); i++ {
			task := s.tasks[i]

			// Check host if it exists
			if task.Host != "" {
				// Don't take offer if it doesn't match our task's host requirement
				if !strings.HasPrefix(*offer.Hostname, task.Host) {
					continue
				}
			}

			for *task.Instances > 0 {
				// Does the task fit
				if (s.ignoreWatts || offer_watts >= (totalWatts+task.Watts)) &&
					(offer_cpu >= (totalCPU + task.CPU)) &&
					(offer_ram >= (totalRAM + task.RAM)) {

					taken = true
					totalWatts += task.Watts
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

		if taken {
			log.Printf("Starting on [%s]\n", offer.GetHostname())
			driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, defaultFilter)
		} else {

			// If there was no match for the task
			fmt.Println("There is not enough resources to launch a task:")
			cpus, mem, watts := OfferAgg(offer)

			log.Printf("<CPU: %f, RAM: %f, Watts: %f>\n", cpus, mem, watts)
			driver.DeclineOffer(offer.Id, defaultFilter)
		}
	}
}

func (s *BinPackSortedWatts) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
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