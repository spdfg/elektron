package schedulers

import (
	"bitbucket.org/sunybingcloud/electron/def"
	"fmt"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	"log"
	"strings"
	"time"
	"sort"
	"os"
	"bitbucket.org/sunybingcloud/electron/pcp"
	"sync"
	"math"
	"bitbucket.org/sunybingcloud/electron/constants"
	"bitbucket.org/sunybingcloud/electron/rapl"
)

// electron scheduler implements the Scheduler interface
type FirstFitSortedWattsClassMapWattsProacCC struct {
	base // Type embedded to inherit common features.
	tasksCreated int
	tasksRunning int
	tasks []def.Task
	metrics map[string]def.Metric
	running map[string]map[string]bool
	taskMonitor    map[string][]def.Task
	availablePower map[string]float64
	totalPower     map[string]float64
	ignoreWatts bool
	capper         *pcp.ClusterwideCapper
	ticker         *time.Ticker
	recapTicker    *time.Ticker
	isCapping      bool // indicate whether we are currently performing cluster-wide capping.
	isRecapping    bool // indicate whether we are currently performing cluster-wide recapping.

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
func NewFirstFitSortedWattsClassMapWattsProacCC(tasks []def.Task, ignoreWatts bool, schedTracePrefix string) *FirstFitSortedWattsClassMapWattsProacCC {
	sort.Sort(def.WattsSorter(tasks))

	logFile, err := os.Create("./" + schedTracePrefix + "_schedTrace.log")
	if err != nil {
		log.Fatal(err)
	}

	s := &FirstFitSortedWattsClassMapWattsProacCC{
		tasks:          tasks,
		ignoreWatts:    ignoreWatts,
		Shutdown:       make(chan struct{}),
		Done:           make(chan struct{}),
		PCPLog:         make(chan struct{}),
		running:        make(map[string]map[string]bool),
		taskMonitor:    make(map[string][]def.Task),
		availablePower: make(map[string]float64),
		totalPower:     make(map[string]float64),
		RecordPCP:      false,
		capper:         pcp.GetClusterwideCapperInstance(),
		ticker:         time.NewTicker(10 * time.Second),
		recapTicker:    time.NewTicker(20 * time.Second),
		isCapping:      false,
		isRecapping:    false,
		schedTrace:     log.New(logFile, "", log.LstdFlags),

	}
	return s
}

// mutex
var ffswClassMapWattsProacCCMutex sync.Mutex

func (s *FirstFitSortedWattsClassMapWattsProacCC) newTask(offer *mesos.Offer, task def.Task,  newTaskClass string) *mesos.TaskInfo {
	taskName := fmt.Sprintf("%s-%d", task.Name, *task.Instances)
	s.tasksCreated++

	if !s.RecordPCP {
		// Turn on logging.
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
	// Add task to the list of tasks running on the node.
	s.running[offer.GetSlaveId().GoString()][taskName] = true
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
		resources = append(resources, mesosutil.NewScalarResource("watts", task.ClassToWatts[newTaskClass]))
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

func (s *FirstFitSortedWattsClassMapWattsProacCC) Disconnected(sched.SchedulerDriver) {
	// Need to stop the capping process
	s.ticker.Stop()
	s.recapTicker.Stop()
	ffswClassMapWattsProacCCMutex.Lock()
	s.isCapping = false
	ffswClassMapWattsProacCCMutex.Unlock()
	log.Println("Framework disconnected with master")
}

// go routine to cap the entire cluster in regular intervals of time
var ffswClassMapWattsProacCCCapValue = 0.0 // initial value to indicate that we haven't capped the cluster yet.
var ffswClassMapWattsProacCCNewCapValue = 0.0 // newly computed cap value
func (s *FirstFitSortedWattsClassMapWattsProacCC) startCapping() {
	go func() {
		for {
			select {
			case <-s.ticker.C:
				// Need to cap the cluster only if new cap value different from the old cap value.
				// This way we don't unnecessarily cap the cluster.
				ffswClassMapWattsProacCCMutex.Lock()
				if s.isCapping {
					if int(math.Floor(ffswClassMapWattsProacCCNewCapValue+0.5)) != int(math.Floor(ffswClassMapWattsProacCCCapValue+0.5)) {
						// updating cap value
						ffswClassMapWattsProacCCCapValue = ffswClassMapWattsProacCCNewCapValue
						if ffswClassMapWattsProacCCCapValue > 0.0 {
							for _, host := range constants.Hosts {
								// Rounding cap value to the nearest int
								if err := rapl.Cap(host, "rapl", int(math.Floor(ffswClassMapWattsProacCCCapValue+0.5))); err != nil {
									log.Println(err)
								}
							}
							log.Printf("Capped the cluster to %d", int(math.Floor(ffswClassMapWattsProacCCCapValue+0.5)))
						}
					}
				}
				ffswClassMapWattsProacCCMutex.Unlock()
			}
		}
	}()
}

// go routine to recap the entire cluster in regular intervals of time.
var ffswClassMapWattsProacCCRecapValue = 0.0 // The cluster-wide cap value when recapping.
func (s *FirstFitSortedWattsClassMapWattsProacCC) startRecapping() {
	go func() {
		for {
			select {
			case <-s.recapTicker.C:
				ffswClassMapWattsProacCCMutex.Lock()
				// If stopped performing cluster wide capping, then we need to recap
				if s.isRecapping && ffswClassMapWattsProacCCRecapValue > 0.0 {
					for _, host := range constants.Hosts {
						// Rounding the cap value to the nearest int
						if err := rapl.Cap(host, "rapl", int(math.Floor(ffswClassMapWattsProacCCRecapValue+0.5))); err != nil {
							log.Println(err)
						}
					}
					log.Printf("Recapping the cluster to %d", int(math.Floor(ffswClassMapWattsProacCCRecapValue+0.5)))
				}
				// Setting recapping to false
				s.isRecapping = false
				ffswClassMapWattsProacCCMutex.Unlock()
			}
		}
	}()
}

// Stop the cluster wide capping
func (s *FirstFitSortedWattsClassMapWattsProacCC) stopCapping() {
	if s.isCapping {
		log.Println("Stopping the cluster-wide capping.")
		s.ticker.Stop()
		ffswClassMapWattsProacCCMutex.Lock()
		s.isCapping = false
		s.isRecapping = true
		ffswClassMapWattsProacCCMutex.Unlock()
	}
}

// Stop the cluster wide recapping
func (s *FirstFitSortedWattsClassMapWattsProacCC) stopRecapping() {
	// If not capping, then definitely recapping.
	if !s.isCapping && s.isRecapping {
		log.Println("Stopping the cluster-wide re-capping.")
		s.recapTicker.Stop()
		ffswClassMapWattsProacCCMutex.Lock()
		s.isRecapping = false
		ffswClassMapWattsProacCCMutex.Unlock()
	}
}

func (s *FirstFitSortedWattsClassMapWattsProacCC) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	log.Printf("Received %d resource offers", len(offers))

	// retrieving the available power for all the hosts in the offers.
	for _, offer := range offers {
		_, _, offerWatts := OfferAgg(offer)
		s.availablePower[*offer.Hostname] = offerWatts
		// setting total power if the first time
		if _, ok := s.totalPower[*offer.Hostname]; !ok {
			s.totalPower[*offer.Hostname] = offerWatts
		}
	}

	for host, tpower := range s.totalPower {
		log.Printf("TotalPower[%s] = %f", host, tpower)
	}

	for _, offer := range offers {
		select {
		case <-s.Shutdown:
			log.Println("Done scheduling tasks: declining offer on [", offer.GetHostname(), "]")
			driver.DeclineOffer(offer.Id, longFilter)

			log.Println("Number of tasks still running: ", s.tasksRunning)
			continue
		default:
		}

		offerCPU, offerRAM, offerWatts := OfferAgg(offer)

		// First fit strategy
		taken := false
		for i, task := range s.tasks {
			// Check host if it exists
			if task.Host != "" {
				// Don't take offer if it doens't match our task's host requirement.
				if !strings.HasPrefix(*offer.Hostname, task.Host) {
					continue
				}
			}

			// Retrieving the node class from the offer
			var nodeClass string
			for _, attr := range offer.GetAttributes() {
				if attr.GetName() == "class" {
					nodeClass = attr.GetText().GetValue()
				}
			}

			// Decision to take the offer or not
			if (s.ignoreWatts || (offerWatts >= task.ClassToWatts[nodeClass])) &&
				(offerCPU >= task.CPU) && (offerRAM >= task.RAM) {

				// Capping the cluster if haven't yet started
				if !s.isCapping {
					ffswClassMapWattsProacCCMutex.Lock()
					s.isCapping = true
					ffswClassMapWattsProacCCMutex.Unlock()
					s.startCapping()
				}

				fmt.Println("Watts being used: ", task.ClassToWatts[nodeClass])
				tempCap, err := s.capper.FCFSDeterminedCap(s.totalPower, &task)
				if err == nil {
					ffswClassMapWattsProacCCMutex.Lock()
					ffswClassMapWattsProacCCNewCapValue = tempCap
					ffswClassMapWattsProacCCMutex.Unlock()
				} else {
					log.Println("Failed to determine new cluster-wide cap: ")
					log.Println(err)
				}

				log.Println("Co-Located with: ")
				coLocated(s.running[offer.GetSlaveId().GoString()])

				taskToSchedule := s.newTask(offer, task, nodeClass)
				s.schedTrace.Print(offer.GetHostname() + ":" + taskToSchedule.GetTaskId().GetValue())
				log.Printf("Starting %s on [%s]\n", task.Name, offer.GetHostname())
				driver.LaunchTasks([]*mesos.OfferID{offer.Id}, []*mesos.TaskInfo{taskToSchedule}, defaultFilter)

				taken = true
				fmt.Println("Inst: ", *task.Instances)
				*task.Instances--
				if *task.Instances <= 0 {
					// All instances of task have been scheduled, remove it
					s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)

					if len(s.tasks) == 0 {
						log.Println("Done scheduling all tasks")
						// Need to stop the cluster-wide capping as there aren't any more tasks to schedule
						s.stopCapping()
						s.startRecapping() // Load changes after every task finishes and hence, we need to change the capping of the cluster
						close(s.Shutdown)
					}
				}
				break // Offer taken, move on
			}
		}

		// If there was no match for the task
		if !taken {
			fmt.Println("There is not enough resources to launch a task:")
			cpus, mem, watts := OfferAgg(offer)

			log.Printf("<CPU: %f, RAM: %f, Watts: %f>\n", cpus, mem, watts)
			driver.DeclineOffer(offer.Id, defaultFilter)
		}
	}
}

func (s *FirstFitSortedWattsClassMapWattsProacCC) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Printf("Received task status [%s] for task [%s]", NameFor(status.State), *status.TaskId.Value)

	if *status.State == mesos.TaskState_TASK_RUNNING {
		s.tasksRunning++
	} else if IsTerminal(status.State) {
		delete(s.running[status.GetSlaveId().GoString()], *status.TaskId.Value)
		// Need to remove the task from the window
		s.capper.TaskFinished(*status.TaskId.Value)
		// Determining the new cluster wide recap value
		//tempCap, err := s.capper.NaiveRecap(s.totalPower, s.taskMonitor, *status.TaskId.Value)
		tempCap, err := s.capper.CleverRecap(s.totalPower, s.taskMonitor, *status.TaskId.Value)
		if err == nil {
			// If new determined cap value is different from the current recap value, then we need to recap
			if int(math.Floor(tempCap+0.5)) != int(math.Floor(ffswClassMapWattsProacCCRecapValue+0.5)) {
				ffswClassMapWattsProacCCRecapValue = tempCap
				ffswClassMapWattsProacCCMutex.Lock()
				s.isRecapping = true
				ffswClassMapWattsProacCCMutex.Unlock()
				log.Printf("Determined re-cap value: %f\n", ffswClassMapWattsProacCCRecapValue)
			} else {
				ffswClassMapWattsProacCCMutex.Lock()
				s.isRecapping = false
				ffswClassMapWattsProacCCMutex.Unlock()
			}
		} else {
			log.Println(err)
		}

		s.tasksRunning--
		if s.tasksRunning == 0 {
			select {
			case <-s.Shutdown:
			// Need to stop the cluster-wide recapping
				s.stopRecapping()
				close(s.Done)
			default:
			}
		}
	}
	log.Printf("DONE: Task status [%s] for task[%s]", NameFor(status.State), *status.TaskId.Value)
}
