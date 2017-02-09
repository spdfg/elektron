package schedulers

import (
	"bitbucket.org/sunybingcloud/electron/constants"
	"bitbucket.org/sunybingcloud/electron/def"
	powCap "bitbucket.org/sunybingcloud/electron/powerCapping"
	"bitbucket.org/sunybingcloud/electron/rapl"
	"bitbucket.org/sunybingcloud/electron/utilities/mesosUtils"
	"bitbucket.org/sunybingcloud/electron/utilities/offerUtils"
	"fmt"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"time"
)

// Decides if to take an offer or not
func (s *FirstFitProacCC) takeOffer(offer *mesos.Offer, task def.Task) bool {
	offer_cpu, offer_mem, offer_watts := offerUtils.OfferAgg(offer)

	wattsConsideration, err := def.WattsToConsider(task, s.classMapWatts, offer)
	if err != nil {
		// Error in determining wattsConsideration
		log.Fatal(err)
	}
	if offer_cpu >= task.CPU && offer_mem >= task.RAM && (!s.wattsAsAResource || (offer_watts >= wattsConsideration)) {
		return true
	}
	return false
}

// electronScheduler implements the Scheduler interface.
type FirstFitProacCC struct {
	base             // Type embedded to inherit common functions
	tasksCreated     int
	tasksRunning     int
	tasks            []def.Task
	metrics          map[string]def.Metric
	running          map[string]map[string]bool
	taskMonitor      map[string][]def.Task // store tasks that are currently running.
	availablePower   map[string]float64    // available power for each node in the cluster.
	totalPower       map[string]float64    // total power for each node in the cluster.
	wattsAsAResource bool
	classMapWatts    bool
	capper           *powCap.ClusterwideCapper
	ticker           *time.Ticker
	recapTicker      *time.Ticker
	isCapping        bool // indicate whether we are currently performing cluster wide capping.
	isRecapping      bool // indicate whether we are currently performing cluster wide re-capping.

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

	schedTrace *log.Logger
}

// New electron scheduler.
func NewFirstFitProacCC(tasks []def.Task, wattsAsAResource bool, schedTracePrefix string,
	classMapWatts bool) *FirstFitProacCC {

	logFile, err := os.Create("./" + schedTracePrefix + "_schedTrace.log")
	if err != nil {
		log.Fatal(err)
	}

	s := &FirstFitProacCC{
		tasks:            tasks,
		wattsAsAResource: wattsAsAResource,
		classMapWatts:    classMapWatts,
		Shutdown:         make(chan struct{}),
		Done:             make(chan struct{}),
		PCPLog:           make(chan struct{}),
		running:          make(map[string]map[string]bool),
		taskMonitor:      make(map[string][]def.Task),
		availablePower:   make(map[string]float64),
		totalPower:       make(map[string]float64),
		RecordPCP:        false,
		capper:           powCap.GetClusterwideCapperInstance(),
		ticker:           time.NewTicker(10 * time.Second),
		recapTicker:      time.NewTicker(20 * time.Second),
		isCapping:        false,
		isRecapping:      false,
		schedTrace:       log.New(logFile, "", log.LstdFlags),
	}
	return s
}

// mutex
var fcfsMutex sync.Mutex

func (s *FirstFitProacCC) newTask(offer *mesos.Offer, task def.Task) *mesos.TaskInfo {
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

func (s *FirstFitProacCC) Disconnected(sched.SchedulerDriver) {
	// Need to stop the capping process.
	s.ticker.Stop()
	s.recapTicker.Stop()
	fcfsMutex.Lock()
	s.isCapping = false
	fcfsMutex.Unlock()
	log.Println("Framework disconnected with master")
}

// go routine to cap the entire cluster in regular intervals of time.
var fcfsCurrentCapValue = 0.0 // initial value to indicate that we haven't capped the cluster yet.
func (s *FirstFitProacCC) startCapping() {
	go func() {
		for {
			select {
			case <-s.ticker.C:
				// Need to cap the cluster to the fcfsCurrentCapValue.
				fcfsMutex.Lock()
				if fcfsCurrentCapValue > 0.0 {
					for _, host := range constants.Hosts {
						// Rounding curreCapValue to the nearest int.
						if err := rapl.Cap(host, "rapl", int(math.Floor(fcfsCurrentCapValue+0.5))); err != nil {
							log.Println(err)
						}
					}
					log.Printf("Capped the cluster to %d", int(math.Floor(fcfsCurrentCapValue+0.5)))
				}
				fcfsMutex.Unlock()
			}
		}
	}()
}

// go routine to cap the entire cluster in regular intervals of time.
var fcfsRecapValue = 0.0 // The cluster wide cap value when recapping.
func (s *FirstFitProacCC) startRecapping() {
	go func() {
		for {
			select {
			case <-s.recapTicker.C:
				fcfsMutex.Lock()
				// If stopped performing cluster wide capping then we need to explicitly cap the entire cluster.
				if s.isRecapping && fcfsRecapValue > 0.0 {
					for _, host := range constants.Hosts {
						// Rounding curreCapValue to the nearest int.
						if err := rapl.Cap(host, "rapl", int(math.Floor(fcfsRecapValue+0.5))); err != nil {
							log.Println(err)
						}
					}
					log.Printf("Recapped the cluster to %d", int(math.Floor(fcfsRecapValue+0.5)))
				}
				// setting recapping to false
				s.isRecapping = false
				fcfsMutex.Unlock()
			}
		}
	}()
}

// Stop cluster wide capping
func (s *FirstFitProacCC) stopCapping() {
	if s.isCapping {
		log.Println("Stopping the cluster wide capping.")
		s.ticker.Stop()
		fcfsMutex.Lock()
		s.isCapping = false
		s.isRecapping = true
		fcfsMutex.Unlock()
	}
}

// Stop cluster wide Recapping
func (s *FirstFitProacCC) stopRecapping() {
	// If not capping, then definitely recapping.
	if !s.isCapping && s.isRecapping {
		log.Println("Stopping the cluster wide re-capping.")
		s.recapTicker.Stop()
		fcfsMutex.Lock()
		s.isRecapping = false
		fcfsMutex.Unlock()
	}
}

func (s *FirstFitProacCC) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	log.Printf("Received %d resource offers", len(offers))

	// retrieving the available power for all the hosts in the offers.
	for _, offer := range offers {
		_, _, offer_watts := offerUtils.OfferAgg(offer)
		s.availablePower[*offer.Hostname] = offer_watts
		// setting total power if the first time.
		if _, ok := s.totalPower[*offer.Hostname]; !ok {
			s.totalPower[*offer.Hostname] = offer_watts
		}
	}

	for host, tpower := range s.totalPower {
		log.Printf("TotalPower[%s] = %f", host, tpower)
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

		/*
		   Clusterwide Capping strategy

		   For each task in s.tasks,
		     1. Need to check whether the offer can be taken or not (based on CPU and RAM requirements).
		     2. If the tasks fits the offer, then I need to detemrine the cluster wide cap.
		     3. fcfsCurrentCapValue is updated with the determined cluster wide cap.

		   Cluster wide capping is currently performed at regular intervals of time.
		*/
		offerTaken := false

		for i := 0; i < len(s.tasks); i++ {
			task := s.tasks[i]
			// Don't take offer if it doesn't match our task's host requirement.
			if !strings.HasPrefix(*offer.Hostname, task.Host) {
				continue
			}

			// Does the task fit.
			if s.takeOffer(offer, task) {
				// Capping the cluster if haven't yet started,
				if !s.isCapping {
					fcfsMutex.Lock()
					s.isCapping = true
					fcfsMutex.Unlock()
					s.startCapping()
				}
				offerTaken = true
				tempCap, err := s.capper.FCFSDeterminedCap(s.totalPower, &task)

				if err == nil {
					fcfsMutex.Lock()
					fcfsCurrentCapValue = tempCap
					fcfsMutex.Unlock()
				} else {
					log.Println("Failed to determine new cluster wide cap: ")
					log.Println(err)
				}
				log.Printf("Starting on [%s]\n", offer.GetHostname())
				taskToSchedule := s.newTask(offer, task)
				toSchedule := []*mesos.TaskInfo{taskToSchedule}
				driver.LaunchTasks([]*mesos.OfferID{offer.Id}, toSchedule, mesosUtils.DefaultFilter)
				log.Printf("Inst: %d", *task.Instances)
				s.schedTrace.Print(offer.GetHostname() + ":" + taskToSchedule.GetTaskId().GetValue())
				*task.Instances--
				if *task.Instances <= 0 {
					// All instances of the task have been scheduled. Need to remove it from the list of tasks to schedule.
					s.tasks[i] = s.tasks[len(s.tasks)-1]
					s.tasks = s.tasks[:len(s.tasks)-1]

					if len(s.tasks) <= 0 {
						log.Println("Done scheduling all tasks")
						// Need to stop the cluster wide capping as there aren't any more tasks to schedule.
						s.stopCapping()
						s.startRecapping() // Load changes after every task finishes and hence we need to change the capping of the cluster.
						close(s.Shutdown)
					}
				}
				break // Offer taken, move on.
			} else {
				// Task doesn't fit the offer. Move onto the next offer.
			}
		}

		// If no task fit the offer, then declining the offer.
		if !offerTaken {
			log.Printf("There is not enough resources to launch a task on Host: %s\n", offer.GetHostname())
			cpus, mem, watts := offerUtils.OfferAgg(offer)

			log.Printf("<CPU: %f, RAM: %f, Watts: %f>\n", cpus, mem, watts)
			driver.DeclineOffer(offer.Id, mesosUtils.DefaultFilter)
		}
	}
}

func (s *FirstFitProacCC) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Printf("Received task status [%s] for task [%s]\n", NameFor(status.State), *status.TaskId.Value)

	if *status.State == mesos.TaskState_TASK_RUNNING {
		fcfsMutex.Lock()
		s.tasksRunning++
		fcfsMutex.Unlock()
	} else if IsTerminal(status.State) {
		delete(s.running[status.GetSlaveId().GoString()], *status.TaskId.Value)
		// Need to remove the task from the window of tasks.
		s.capper.TaskFinished(*status.TaskId.Value)
		// Determining the new cluster wide cap.
		//tempCap, err := s.capper.NaiveRecap(s.totalPower, s.taskMonitor, *status.TaskId.Value)
		tempCap, err := s.capper.CleverRecap(s.totalPower, s.taskMonitor, *status.TaskId.Value)
		if err == nil {
			// if new determined cap value is different from the current recap value then we need to recap.
			if int(math.Floor(tempCap+0.5)) != int(math.Floor(fcfsRecapValue+0.5)) {
				fcfsRecapValue = tempCap
				fcfsMutex.Lock()
				s.isRecapping = true
				fcfsMutex.Unlock()
				log.Printf("Determined re-cap value: %f\n", fcfsRecapValue)
			} else {
				fcfsMutex.Lock()
				s.isRecapping = false
				fcfsMutex.Unlock()
			}
		} else {
			// Not updating fcfsCurrentCapValue
			log.Println(err)
		}

		fcfsMutex.Lock()
		s.tasksRunning--
		fcfsMutex.Unlock()
		if s.tasksRunning == 0 {
			select {
			case <-s.Shutdown:
				// Need to stop the recapping process.
				s.stopRecapping()
				close(s.Done)
			default:
			}
		}
	}
	log.Printf("DONE: Task status [%s] for task [%s]", NameFor(status.State), *status.TaskId.Value)
}
