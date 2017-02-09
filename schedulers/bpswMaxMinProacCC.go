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
	"sort"
	"strings"
	"sync"
	"time"
)

// Decides if to take an offer or not
func (s *BPSWMaxMinProacCC) takeOffer(offer *mesos.Offer, task def.Task) bool {
	cpus, mem, watts := offerUtils.OfferAgg(offer)

	//TODO: Insert watts calculation here instead of taking them as a parameter

	wattsConsideration, err := def.WattsToConsider(task, s.classMapWatts, offer)
	if err != nil {
		// Error in determining wattsConsideration
		log.Fatal(err)
	}
	if cpus >= task.CPU && mem >= task.RAM && (!s.wattsAsAResource || (watts >= wattsConsideration)) {
		return true
	}

	return false
}

type BPSWMaxMinProacCC struct {
	base             // Type embedding to inherit common functions
	tasksCreated     int
	tasksRunning     int
	tasks            []def.Task
	metrics          map[string]def.Metric
	running          map[string]map[string]bool
	taskMonitor      map[string][]def.Task
	availablePower   map[string]float64
	totalPower       map[string]float64
	wattsAsAResource bool
	classMapWatts    bool
	capper           *powCap.ClusterwideCapper
	ticker           *time.Ticker
	recapTicker      *time.Ticker
	isCapping        bool // indicate whether we are currently performing cluster-wide capping.
	isRecapping      bool // indicate whether we are currently performing cluster-wide recapping.

	// First set of PCP values are garbage values, signal to logger to start recording when we're
	// about to schedule a new task
	RecordPCP bool

	// This channel is closed when the program receives an interrupt,
	// signalling that the program should shut down
	Shutdown chan struct{}
	// This channel is closed after shutdown is closed, and only when all
	// outstanding tasks have been cleaned up
	Done chan struct{}

	// Controls when to shutdown pcp logging
	PCPLog chan struct{}

	schedTrace *log.Logger
}

// New electron scheduler
func NewBPSWMaxMinProacCC(tasks []def.Task, wattsAsAResource bool, schedTracePrefix string, classMapWatts bool) *BPSWMaxMinProacCC {
	sort.Sort(def.WattsSorter(tasks))

	logFile, err := os.Create("./" + schedTracePrefix + "_schedTrace.log")
	if err != nil {
		log.Fatal(err)
	}

	s := &BPSWMaxMinProacCC{
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
var bpMaxMinProacCCMutex sync.Mutex

func (s *BPSWMaxMinProacCC) newTask(offer *mesos.Offer, task def.Task) *mesos.TaskInfo {
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

// go routine to cap the entire cluster in regular intervals of time.
var bpMaxMinProacCCCapValue = 0.0    // initial value to indicate that we haven't capped the cluster yet.
var bpMaxMinProacCCNewCapValue = 0.0 // newly computed cap value
func (s *BPSWMaxMinProacCC) startCapping() {
	go func() {
		for {
			select {
			case <-s.ticker.C:
				// Need to cap the cluster only if new cap value different from old cap value.
				// This way we don't unnecessarily cap the cluster.
				bpMaxMinProacCCMutex.Lock()
				if s.isCapping {
					if int(math.Floor(bpMaxMinProacCCNewCapValue+0.5)) != int(math.Floor(bpMaxMinProacCCCapValue+0.5)) {
						// updating cap value
						bpMaxMinProacCCCapValue = bpMaxMinProacCCNewCapValue
						if bpMaxMinProacCCCapValue > 0.0 {
							for _, host := range constants.Hosts {
								// Rounding cap value to nearest int
								if err := rapl.Cap(host, "rapl", int(math.Floor(bpMaxMinProacCCCapValue+0.5))); err != nil {
									log.Println(err)
								}
							}
							log.Printf("Capped the cluster to %d", int(math.Floor(bpMaxMinProacCCCapValue+0.5)))
						}
					}
				}
				bpMaxMinProacCCMutex.Unlock()
			}
		}
	}()
}

// go routine to recap the entire cluster in regular intervals of time.
var bpMaxMinProacCCRecapValue = 0.0 // The cluster-wide cap value when recapping.
func (s *BPSWMaxMinProacCC) startRecapping() {
	go func() {
		for {
			select {
			case <-s.recapTicker.C:
				bpMaxMinProacCCMutex.Lock()
				// If stopped performing cluster-wide capping, then we need to recap.
				if s.isRecapping && bpMaxMinProacCCRecapValue > 0.0 {
					for _, host := range constants.Hosts {
						// Rounding the recap value to the nearest int
						if err := rapl.Cap(host, "rapl", int(math.Floor(bpMaxMinProacCCRecapValue+0.5))); err != nil {
							log.Println(err)
						}
					}
					log.Printf("Capped the cluster to %d", int(math.Floor(bpMaxMinProacCCRecapValue+0.5)))
				}
				// Setting the recapping to false
				s.isRecapping = false
				bpMaxMinProacCCMutex.Unlock()
			}
		}
	}()

}

// Stop cluster-wide capping
func (s *BPSWMaxMinProacCC) stopCapping() {
	if s.isCapping {
		log.Println("Stopping the cluster-wide capping.")
		s.ticker.Stop()
		bpMaxMinProacCCMutex.Lock()
		s.isCapping = false
		s.isRecapping = true
		bpMaxMinProacCCMutex.Unlock()
	}
}

// Stop the cluster-wide recapping
func (s *BPSWMaxMinProacCC) stopRecapping() {
	// If not capping, then definitely recapping.
	if !s.isCapping && s.isRecapping {
		log.Println("Stopping the cluster-wide re-capping.")
		s.recapTicker.Stop()
		bpMaxMinProacCCMutex.Lock()
		s.isRecapping = false
		bpMaxMinProacCCMutex.Unlock()
	}
}

// Determine if the remaining space inside of the offer is enough for
// the task we need to create. If it is, create TaskInfo and return it.
func (s *BPSWMaxMinProacCC) CheckFit(i int,
	task def.Task,
	wattsConsideration float64,
	offer *mesos.Offer,
	totalCPU *float64,
	totalRAM *float64,
	totalWatts *float64) (bool, *mesos.TaskInfo) {

	offerCPU, offerRAM, offerWatts := offerUtils.OfferAgg(offer)

	// Does the task fit
	if (!s.wattsAsAResource || (offerWatts >= (*totalWatts + wattsConsideration))) &&
		(offerCPU >= (*totalCPU + task.CPU)) &&
		(offerRAM >= (*totalRAM + task.RAM)) {

		// Capping the cluster if haven't yet started
		if !s.isCapping {
			bpMaxMinProacCCMutex.Lock()
			s.isCapping = true
			bpMaxMinProacCCMutex.Unlock()
			s.startCapping()
		}

		tempCap, err := s.capper.FCFSDeterminedCap(s.totalPower, &task)
		if err == nil {
			bpMaxMinProacCCMutex.Lock()
			bpMaxMinProacCCNewCapValue = tempCap
			bpMaxMinProacCCMutex.Unlock()
		} else {
			log.Println("Failed to determine new cluster-wide cap:")
			log.Println(err)
		}

		*totalWatts += wattsConsideration
		*totalCPU += task.CPU
		*totalRAM += task.RAM
		log.Println("Co-Located with: ")
		coLocated(s.running[offer.GetSlaveId().GoString()])

		taskToSchedule := s.newTask(offer, task)

		fmt.Println("Inst: ", *task.Instances)
		s.schedTrace.Print(offer.GetHostname() + ":" + taskToSchedule.GetTaskId().GetValue())
		*task.Instances--

		if *task.Instances <= 0 {
			// All instances of task have been scheduled, remove it
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)

			if len(s.tasks) <= 0 {
				log.Println("Done scheduling all tasks")
				// Need to stop the cluster wide capping
				s.stopCapping()
				s.startRecapping() // Load changes after every task finishes and hence, we need to change the capping of the cluster.
				close(s.Shutdown)
			}
		}

		return true, taskToSchedule
	}

	return false, nil

}

func (s *BPSWMaxMinProacCC) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	log.Printf("Received %d resource offers", len(offers))

	// retrieving the available power for all the hosts in the offers.
	for _, offer := range offers {
		_, _, offerWatts := offerUtils.OfferAgg(offer)
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

		// Assumes s.tasks is ordered in non-decreasing median max peak order

		// Attempt to schedule a single instance of the heaviest workload available first
		// Start from the back until one fits
		for i := len(s.tasks) - 1; i >= 0; i-- {

			task := s.tasks[i]
			wattsConsideration, err := def.WattsToConsider(task, s.classMapWatts, offer)
			if err != nil {
				// Error in determining wattsConsideration
				log.Fatal(err)
			}
			// Check host if it exists
			if task.Host != "" {
				// Don't take offer if it doesn't match our task's host requirement
				if !strings.HasPrefix(*offer.Hostname, task.Host) {
					continue
				}
			}

			// TODO: Fix this so index doesn't need to be passed
			taken, taskToSchedule := s.CheckFit(i, task, wattsConsideration, offer,
				&totalCPU, &totalRAM, &totalWatts)

			if taken {
				offerTaken = true
				tasks = append(tasks, taskToSchedule)
				break
			}
		}

		// Pack the rest of the offer with the smallest tasks
		for i := 0; i < len(s.tasks); i++ {
			task := s.tasks[i]
			wattsConsideration, err := def.WattsToConsider(task, s.classMapWatts, offer)
			if err != nil {
				// Error in determining wattsConsideration
				log.Fatal(err)
			}

			// Check host if it exists
			if task.Host != "" {
				// Don't take offer if it doesn't match our task's host requirement
				if !strings.HasPrefix(*offer.Hostname, task.Host) {
					continue
				}
			}

			for *task.Instances > 0 {
				// TODO: Fix this so index doesn't need to be passed
				taken, taskToSchedule := s.CheckFit(i, task, wattsConsideration, offer,
					&totalCPU, &totalRAM, &totalWatts)

				if taken {
					offerTaken = true
					tasks = append(tasks, taskToSchedule)
				} else {
					break // Continue on to next task
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

func (s *BPSWMaxMinProacCC) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Printf("Received task status [%s] for task [%s]", NameFor(status.State), *status.TaskId.Value)

	if *status.State == mesos.TaskState_TASK_RUNNING {
		s.tasksRunning++
	} else if IsTerminal(status.State) {
		delete(s.running[status.GetSlaveId().GoString()], *status.TaskId.Value)
		// Need to remove the task from the window
		s.capper.TaskFinished(*status.TaskId.Value)
		// Determining the new cluster wide recap value
		tempCap, err := s.capper.NaiveRecap(s.totalPower, s.taskMonitor, *status.TaskId.Value)
		//tempCap, err := s.capper.CleverRecap(s.totalPower, s.taskMonitor, *status.TaskId.Value)
		if err == nil {
			// If new determined recap value is different from the current recap value, then we need to recap.
			if int(math.Floor(tempCap+0.5)) != int(math.Floor(bpMaxMinProacCCRecapValue+0.5)) {
				bpMaxMinProacCCRecapValue = tempCap
				bpMaxMinProacCCMutex.Lock()
				s.isRecapping = true
				bpMaxMinProacCCMutex.Unlock()
				log.Printf("Determined re-cap value: %f\n", bpMaxMinProacCCRecapValue)
			} else {
				bpMaxMinProacCCMutex.Lock()
				s.isRecapping = false
				bpMaxMinProacCCMutex.Unlock()
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
	log.Printf("DONE: Task status [%s] for task [%s]", NameFor(status.State), *status.TaskId.Value)

}
