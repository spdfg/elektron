package schedulers

import (
	"bitbucket.org/sunybingcloud/electron/constants"
	"bitbucket.org/sunybingcloud/electron/def"
	"bitbucket.org/sunybingcloud/electron/rapl"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	"log"
	"math"
	"strings"
	"sync"
	"time"
)

/*
 Piston Capper implements the Scheduler interface

 This is basically extending the BinPacking algorithm to also cap each node at a different values,
  corresponding to the load on that node.
*/
type PistonCapper struct {
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
	// about to schedule the new task.
	RecordPCP bool

	// This channel is closed when the program receives an interrupt,
	// signalling that the program should shut down.
	Shutdown chan struct{}

	// This channel is closed after shutdown is closed, and only when all
	// outstanding tasks have been cleaned up.
	Done chan struct{}

	// Controls when to shutdown pcp logging.
	PCPLog chan struct{}
}

// New electron scheduler.
func NewPistonCapper(tasks []def.Task, ignoreWatts bool) *PistonCapper {
	s := &PistonCapper{
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
	}
	return s
}

// mutex
var mutex sync.Mutex

func (s *PistonCapper) newTask(offer *mesos.Offer, task def.Task) *mesos.TaskInfo {
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

	// Setting the task ID to the task. This is done so that we can consider each task to be different,
	// even though they have the same parameters.
	task.SetTaskID(*proto.String("electron-" + taskName))
	// Add task to list of tasks running on node
	s.running[offer.GetSlaveId().GoString()][taskName] = true
	// Adding the task to the taskMonitor
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

func (s *PistonCapper) Registered(
	_ sched.SchedulerDriver,
	frameworkID *mesos.FrameworkID,
	masterInfo *mesos.MasterInfo) {
	log.Printf("Framework %s registered with master %s", frameworkID, masterInfo)
}

func (s *PistonCapper) Reregistered(_ sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	log.Printf("Framework re-registered with master %s", masterInfo)
}

func (s *PistonCapper) Disconnected(sched.SchedulerDriver) {
	log.Println("Framework disconnected with master")
}

// go routine to cap the each node in the cluster at regular intervals of time.
var capValues = make(map[string]float64)

// Storing the previous cap value for each host so as to not repeatedly cap the nodes to the same value. (reduces overhead)
var previousRoundedCapValues = make(map[string]int)

func (s *PistonCapper) startCapping() {
	go func() {
		for {
			select {
			case <-s.ticker.C:
				// Need to cap each node
				mutex.Lock()
				for host, capValue := range capValues {
					roundedCapValue := int(math.Floor(capValue + 0.5))
					// has the cap value changed
					if prevRoundedCap, ok := previousRoundedCapValues[host]; ok {
						if prevRoundedCap != roundedCapValue {
							if err := rapl.Cap(host, "rapl", roundedCapValue); err != nil {
								log.Println(err)
							} else {
								log.Printf("Capped [%s] at %d", host, int(math.Floor(capValue+0.5)))
							}
							previousRoundedCapValues[host] = roundedCapValue
						}
					} else {
						if err := rapl.Cap(host, "rapl", roundedCapValue); err != nil {
							log.Println(err)
						} else {
							log.Printf("Capped [%s] at %d", host, int(math.Floor(capValue+0.5)))
						}
						previousRoundedCapValues[host] = roundedCapValue
					}
				}
				mutex.Unlock()
			}
		}
	}()
}

// Stop the capping
func (s *PistonCapper) stopCapping() {
	if s.isCapping {
		log.Println("Stopping the capping.")
		s.ticker.Stop()
		mutex.Lock()
		s.isCapping = false
		mutex.Unlock()
	}
}

func (s *PistonCapper) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	log.Printf("Received %d resource offers", len(offers))

	// retrieving the total power for each host in the offers
	for _, offer := range offers {
		if _, ok := s.totalPower[*offer.Hostname]; !ok {
			_, _, offer_watts := OfferAgg(offer)
			s.totalPower[*offer.Hostname] = offer_watts
		}
	}

	// Displaying the totalPower
	for host, tpower := range s.totalPower {
		log.Printf("TotalPower[%s] = %f", host, tpower)
	}

	/*
		Piston capping strategy

		Perform bin-packing of tasks on nodes in the cluster, making sure that no task is given less hard-limit resources than requested.
		For each set of tasks that are scheduled, compute the new cap values for each host in the cluster.
		At regular intervals of time, cap each node in the cluster.
	*/
	for _, offer := range offers {
		select {
		case <-s.Shutdown:
			log.Println("Done scheduling tasks: declining offer on [", offer.GetHostname(), "]")
			driver.DeclineOffer(offer.Id, longFilter)

			log.Println("Number of tasks still running: ", s.tasksRunning)
			continue
		default:
		}

		fitTasks := []*mesos.TaskInfo{}
		offerCPU, offerRAM, offerWatts := OfferAgg(offer)
		taken := false
		totalWatts := 0.0
		totalCPU := 0.0
		totalRAM := 0.0
		// Store the partialLoad for host corresponding to this offer.
		// Once we can't fit any more tasks, we update capValue for this host with partialLoad and then launch the fit tasks.
		partialLoad := 0.0
		for i, task := range s.tasks {
			// Check host if it exists
			if task.Host != "" {
				// Don't take offer if it doens't match our task's host requirement.
				if !strings.HasPrefix(*offer.Hostname, task.Host) {
					continue
				}
			}

			for *task.Instances > 0 {
				// Does the task fit
				if (s.ignoreWatts || (offerWatts >= (totalWatts + task.Watts))) &&
					(offerCPU >= (totalCPU + task.CPU)) &&
					(offerRAM >= (totalRAM + task.RAM)) {

					// Start piston capping if haven't started yet
					if !s.isCapping {
						s.isCapping = true
						s.startCapping()
					}

					taken = true
					totalWatts += task.Watts
					totalCPU += task.CPU
					totalRAM += task.RAM
					log.Println("Co-Located with: ")
					coLocated(s.running[offer.GetSlaveId().GoString()])
					fitTasks = append(fitTasks, s.newTask(offer, task))

					log.Println("Inst: ", *task.Instances)
					*task.Instances--
					// updating the cap value for offer.Hostname
					partialLoad += ((task.Watts * constants.CapMargin) / s.totalPower[*offer.Hostname]) * 100

					if *task.Instances <= 0 {
						// All instances of task have been scheduled. Remove it
						s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
						if len(s.tasks) <= 0 {
							log.Println("Done scheduling all tasks")
							close(s.Shutdown)
						}
					}
				} else {
					break // Continue on to next task
				}
			}
		}

		if taken {
			// Updating the cap value for offer.Hostname
			mutex.Lock()
			capValues[*offer.Hostname] += partialLoad
			mutex.Unlock()
			log.Printf("Starting on [%s]\n", offer.GetHostname())
			driver.LaunchTasks([]*mesos.OfferID{offer.Id}, fitTasks, defaultFilter)
		} else {
			// If there was no match for task
			log.Println("There is not enough resources to launch task: ")
			cpus, mem, watts := OfferAgg(offer)

			log.Printf("<CPU: %f, RAM: %f, Watts: %f>\n", cpus, mem, watts)
			driver.DeclineOffer(offer.Id, defaultFilter)
		}
	}
}

// Remove finished task from the taskMonitor
func (s *PistonCapper) deleteFromTaskMonitor(finishedTaskID string) (def.Task, string, error) {
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

func (s *PistonCapper) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Printf("Received task status [%s] for task [%s]\n", NameFor(status.State), *status.TaskId.Value)

	if *status.State == mesos.TaskState_TASK_RUNNING {
		mutex.Lock()
		s.tasksRunning++
		mutex.Unlock()
	} else if IsTerminal(status.State) {
		delete(s.running[status.GetSlaveId().GoString()], *status.TaskId.Value)
		// Deleting the task from the taskMonitor
		finishedTask, hostOfFinishedTask, err := s.deleteFromTaskMonitor(*status.TaskId.Value)
		if err != nil {
			log.Println(err)
		}

		// Need to update the cap values for host of the finishedTask
		mutex.Lock()
		capValues[hostOfFinishedTask] -= ((finishedTask.Watts * constants.CapMargin) / s.totalPower[hostOfFinishedTask]) * 100
		// Checking to see if the cap value has become 0, in which case we uncap the host.
		if int(math.Floor(capValues[hostOfFinishedTask]+0.5)) == 0 {
			capValues[hostOfFinishedTask] = 100
		}
		s.tasksRunning--
		mutex.Unlock()

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

func (s *PistonCapper) FrameworkMessage(
	driver sched.SchedulerDriver,
	executorID *mesos.ExecutorID,
	slaveID *mesos.SlaveID,
	message string) {

	log.Println("Getting a framework message: ", message)
	log.Printf("Received a framework message from some unknown source: %s", *executorID.Value)
}

func (s *PistonCapper) OfferRescinded(_ sched.SchedulerDriver, offerID *mesos.OfferID) {
	log.Printf("Offer %s rescinded", offerID)
}
func (s *PistonCapper) SlaveLost(_ sched.SchedulerDriver, slaveID *mesos.SlaveID) {
	log.Printf("Slave %s lost", slaveID)
}
func (s *PistonCapper) ExecutorLost(_ sched.SchedulerDriver, executorID *mesos.ExecutorID, slaveID *mesos.SlaveID, status int) {
	log.Printf("Executor %s on slave %s was lost", executorID, slaveID)
}

func (s *PistonCapper) Error(_ sched.SchedulerDriver, err string) {
	log.Printf("Receiving an error: %s", err)
}
