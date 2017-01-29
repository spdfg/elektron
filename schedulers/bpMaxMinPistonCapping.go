package schedulers

import (
	"bitbucket.org/sunybingcloud/electron/constants"
	"bitbucket.org/sunybingcloud/electron/def"
	"bitbucket.org/sunybingcloud/electron/rapl"
	"bitbucket.org/sunybingcloud/electron/utilities/mesosUtils"
	"bitbucket.org/sunybingcloud/electron/utilities/offerUtils"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// Decides if to take an offer or not
func (s *BPMaxMinPistonCapping) takeOffer(offer *mesos.Offer, task def.Task) bool {

	cpus, mem, watts := offerUtils.OfferAgg(offer)

	//TODO: Insert watts calculation here instead of taking them as a parameter

	if cpus >= task.CPU && mem >= task.RAM && watts >= task.Watts {
		return true
	}

	return false
}

type BPMaxMinPistonCapping struct {
	base         //Type embedding to inherit common functions
	tasksCreated int
	tasksRunning int
	tasks        []def.Task
	metrics      map[string]def.Metric
	running      map[string]map[string]bool
	taskMonitor  map[string][]def.Task
	totalPower   map[string]float64
	ignoreWatts  bool
	ticker       *time.Ticker
	isCapping    bool

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
func NewBPMaxMinPistonCapping(tasks []def.Task, ignoreWatts bool, schedTracePrefix string) *BPMaxMinPistonCapping {
	sort.Sort(def.WattsSorter(tasks))

	logFile, err := os.Create("./" + schedTracePrefix + "_schedTrace.log")
	if err != nil {
		log.Fatal(err)
	}

	s := &BPMaxMinPistonCapping{
		tasks:       tasks,
		ignoreWatts: ignoreWatts,
		Shutdown:    make(chan struct{}),
		Done:        make(chan struct{}),
		PCPLog:      make(chan struct{}),
		running:     make(map[string]map[string]bool),
		taskMonitor: make(map[string][]def.Task),
		totalPower:  make(map[string]float64),
		RecordPCP:   false,
		ticker:      time.NewTicker(5 * time.Second),
		isCapping:   false,
		schedTrace:  log.New(logFile, "", log.LstdFlags),
	}
	return s

}

func (s *BPMaxMinPistonCapping) newTask(offer *mesos.Offer, task def.Task) *mesos.TaskInfo {
	taskName := fmt.Sprintf("%s-%d", task.Name, *task.Instances)
	s.tasksCreated++

	// Start recording only when we're creating the first task
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

	// Setting the task ID to the task. This is done so that we can consider each task to be different
	// even though they have the same parameters.
	task.SetTaskID(*proto.String("electron-" + taskName))
	// Add task to list of tasks running on node
	if len(s.taskMonitor[*offer.Hostname]) == 0 {
		s.taskMonitor[*offer.Hostname] = []def.Task{task}
	} else {
		s.taskMonitor[*offer.Hostname] = append(s.taskMonitor[*offer.Hostname], task)
	}

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

func (s *BPMaxMinPistonCapping) Disconnected(sched.SchedulerDriver) {
	// Need to stop the capping process
	s.ticker.Stop()
	bpMaxMinPistonCappingMutex.Lock()
	s.isCapping = false
	bpMaxMinPistonCappingMutex.Unlock()
	log.Println("Framework disconnected with master")
}

// mutex
var bpMaxMinPistonCappingMutex sync.Mutex

// go routine to cap each node in the cluster at regular intervals of time
var bpMaxMinPistonCappingCapValues = make(map[string]float64)

// Storing the previous cap value for each host so as to not repeatedly cap the nodes to the same value. (reduces overhead)
var bpMaxMinPistonCappingPreviousRoundedCapValues = make(map[string]int)

func (s *BPMaxMinPistonCapping) startCapping() {
	go func() {
		for {
			select {
			case <-s.ticker.C:
				// Need to cap each node
				bpMaxMinPistonCappingMutex.Lock()
				for host, capValue := range bpMaxMinPistonCappingCapValues {
					roundedCapValue := int(math.Floor(capValue + 0.5))
					// has the cap value changed
					if previousRoundedCap, ok := bpMaxMinPistonCappingPreviousRoundedCapValues[host]; ok {
						if previousRoundedCap != roundedCapValue {
							if err := rapl.Cap(host, "rapl", roundedCapValue); err != nil {
								log.Println(err)
							} else {
								log.Printf("Capped [%s] at %d", host, int(math.Floor(capValue)))
							}
							bpMaxMinPistonCappingPreviousRoundedCapValues[host] = roundedCapValue
						}
					} else {
						if err := rapl.Cap(host, "rapl", roundedCapValue); err != nil {
							log.Println(err)
						} else {
							log.Printf("Capped [%s] at %d", host, int(math.Floor(capValue+0.5)))
						}
						bpMaxMinPistonCappingPreviousRoundedCapValues[host] = roundedCapValue
					}
				}
				bpMaxMinPistonCappingMutex.Unlock()
			}
		}
	}()

}

// Stop the capping
func (s *BPMaxMinPistonCapping) stopCapping() {
	if s.isCapping {
		log.Println("Stopping the capping.")
		s.ticker.Stop()
		bpMaxMinPistonCappingMutex.Lock()
		s.isCapping = false
		bpMaxMinPistonCappingMutex.Unlock()
	}
}

// Determine if the remaining sapce inside of the offer is enough for
// the task we need to create. If it is, create a TaskInfo and return it.
func (s *BPMaxMinPistonCapping) CheckFit(i int,
	task def.Task,
	offer *mesos.Offer,
	totalCPU *float64,
	totalRAM *float64,
	totalWatts *float64,
	partialLoad *float64) (bool, *mesos.TaskInfo) {

	offerCPU, offerRAM, offerWatts := offerUtils.OfferAgg(offer)

	// Does the task fit
	if (s.ignoreWatts || (offerWatts >= (*totalWatts + task.Watts))) &&
		(offerCPU >= (*totalCPU + task.CPU)) &&
		(offerRAM >= (*totalRAM + task.RAM)) {

		// Start piston capping if haven't started yet
		if !s.isCapping {
			s.isCapping = true
			s.startCapping()
		}

		*totalWatts += task.Watts
		*totalCPU += task.CPU
		*totalRAM += task.RAM
		log.Println("Co-Located with: ")
		coLocated(s.running[offer.GetSlaveId().GoString()])

		taskToSchedule := s.newTask(offer, task)

		fmt.Println("Inst: ", *task.Instances)
		s.schedTrace.Print(offer.GetHostname() + ":" + taskToSchedule.GetTaskId().GetValue())
		*task.Instances--
		*partialLoad += ((task.Watts * constants.CapMargin) / s.totalPower[*offer.Hostname]) * 100

		if *task.Instances <= 0 {
			// All instances of task have been scheduled, remove it
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)

			if len(s.tasks) <= 0 {
				log.Println("Done scheduling all tasks")
				close(s.Shutdown)
			}
		}

		return true, taskToSchedule
	}

	return false, nil
}

func (s *BPMaxMinPistonCapping) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
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

		tasks := []*mesos.TaskInfo{}

		offerTaken := false
		totalWatts := 0.0
		totalCPU := 0.0
		totalRAM := 0.0
		// Store the partialLoad for host corresponding to this offer
		// Once we can't fit any more tasks, we update the capValue for this host using partialLoad and then launch the fit tasks.
		partialLoad := 0.0

		// Assumes s.tasks is ordered in non-decreasing median max peak order

		// Attempt to schedule a single instance of the heaviest workload available first
		// Start from the back until one fits
		for i := len(s.tasks) - 1; i >= 0; i-- {

			task := s.tasks[i]
			// Check host if it exists
			if task.Host != "" {
				// Don't take offer if it doesn't match our task's host requirement
				if !strings.HasPrefix(*offer.Hostname, task.Host) {
					continue
				}
			}

			// TODO: Fix this so index doesn't need to be passed
			taken, taskToSchedule := s.CheckFit(i, task, offer, &totalCPU, &totalRAM, &totalWatts, &partialLoad)

			if taken {
				offerTaken = true
				tasks = append(tasks, taskToSchedule)
				break
			}
		}

		// Pack the rest of the offer with the smallest tasks
		for i, task := range s.tasks {

			// Check host if it exists
			if task.Host != "" {
				// Don't take offer if it doesn't match our task's host requirement
				if !strings.HasPrefix(*offer.Hostname, task.Host) {
					continue
				}
			}

			for *task.Instances > 0 {
				// TODO: Fix this so index doesn't need to be passed
				taken, taskToSchedule := s.CheckFit(i, task, offer, &totalCPU, &totalRAM, &totalWatts, &partialLoad)

				if taken {
					offerTaken = true
					tasks = append(tasks, taskToSchedule)
				} else {
					break // Continue on to next task
				}
			}
		}

		if offerTaken {
			// Updating the cap value for offer.Hostname
			bpMaxMinPistonCappingMutex.Lock()
			bpMaxMinPistonCappingCapValues[*offer.Hostname] += partialLoad
			bpMaxMinPistonCappingMutex.Unlock()
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

// Remove finished task from the taskMonitor
func (s *BPMaxMinPistonCapping) deleteFromTaskMonitor(finishedTaskID string) (def.Task, string, error) {
	hostOfFinishedTask := ""
	indexOfFinishedTask := -1
	found := false
	var finishedTask def.Task

	for host, tasks := range s.taskMonitor {
		for i, task := range tasks {
			if task.TaskID == finishedTaskID {
				hostOfFinishedTask = host
				indexOfFinishedTask = i
				found = true
			}
		}
		if found {
			break
		}
	}

	if hostOfFinishedTask != "" && indexOfFinishedTask != -1 {
		finishedTask = s.taskMonitor[hostOfFinishedTask][indexOfFinishedTask]
		log.Printf("Removing task with TaskID [%s] from the list of running tasks\n",
			s.taskMonitor[hostOfFinishedTask][indexOfFinishedTask].TaskID)
		s.taskMonitor[hostOfFinishedTask] = append(s.taskMonitor[hostOfFinishedTask][:indexOfFinishedTask],
			s.taskMonitor[hostOfFinishedTask][indexOfFinishedTask+1:]...)
	} else {
		return finishedTask, hostOfFinishedTask, errors.New("Finished Task not present in TaskMonitor")
	}
	return finishedTask, hostOfFinishedTask, nil
}

func (s *BPMaxMinPistonCapping) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Printf("Received task status [%s] for task [%s]", NameFor(status.State), *status.TaskId.Value)

	if *status.State == mesos.TaskState_TASK_RUNNING {
		bpMaxMinPistonCappingMutex.Lock()
		s.tasksRunning++
		bpMaxMinPistonCappingMutex.Unlock()
	} else if IsTerminal(status.State) {
		delete(s.running[status.GetSlaveId().GoString()], *status.TaskId.Value)
		// Deleting the task from the taskMonitor
		finishedTask, hostOfFinishedTask, err := s.deleteFromTaskMonitor(*status.TaskId.Value)
		if err != nil {
			log.Println(err)
		}

		// Need to update the cap values for host of the finishedTask
		bpMaxMinPistonCappingMutex.Lock()
		bpMaxMinPistonCappingCapValues[hostOfFinishedTask] -= ((finishedTask.Watts * constants.CapMargin) / s.totalPower[hostOfFinishedTask]) * 100
		// Checking to see if the cap value has become 0, in which case we uncap the host.
		if int(math.Floor(bpMaxMinPistonCappingCapValues[hostOfFinishedTask]+0.5)) == 0 {
			bpMaxMinPistonCappingCapValues[hostOfFinishedTask] = 100
		}
		s.tasksRunning--
		bpMaxMinPistonCappingMutex.Unlock()

		if s.tasksRunning == 0 {
			select {
			case <-s.Shutdown:
				s.stopCapping()
				close(s.Done)
			default:
			}
		}
	}
	log.Printf("DONE: Task status [%s] for task [%s]", NameFor(status.State), *status.TaskId.Value)

}
