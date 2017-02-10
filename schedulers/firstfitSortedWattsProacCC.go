/*
Ranked based cluster wide capping.

Note: Sorting the tasks right in the beginning, in ascending order of watts.
	You are hence certain that the tasks that didn't fit are the ones that require more resources,
		and hence, you can find a way to address that issue.
	On the other hand, if you use first fit to fit the tasks and then sort them to determine the cap,
		you are never certain as which tasks are the ones that don't fit and hence, it becomes much harder
			to address this issue.
*/
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

// Decides if to taken an offer or not
func (s *FirstFitSortedWattsProacCC) takeOffer(offer *mesos.Offer, task def.Task) bool {
	offer_cpu, offer_mem, offer_watts := offerUtils.OfferAgg(offer)

	wattsConsideration, err := def.WattsToConsider(task, s.classMapWatts, offer)
	if err != nil {
		// Error in determining wattsToConsider
		log.Fatal(err)
	}
	if offer_cpu >= task.CPU && offer_mem >= task.RAM && (!s.wattsAsAResource || offer_watts >= wattsConsideration) {
		return true
	}
	return false
}

// electronScheduler implements the Scheduler interface
type FirstFitSortedWattsProacCC struct {
	base                                 // Type embedded to inherit common functions
	taskMonitor    map[string][]def.Task // store tasks that are currently running.
	availablePower map[string]float64    // available power for each node in the cluster.
	totalPower     map[string]float64    // total power for each node in the cluster.
	capper         *powCap.ClusterwideCapper
	ticker         *time.Ticker
	recapTicker    *time.Ticker
	isCapping      bool // indicate whether we are currently performing cluster wide capping.
	isRecapping    bool // indicate whether we are currently performing cluster wide re-capping.
}

// New electron scheduler.
func NewFirstFitSortedWattsProacCC(tasks []def.Task, wattsAsAResource bool, schedTracePrefix string,
	classMapWatts bool) *FirstFitSortedWattsProacCC {

	// Sorting tasks in ascending order of watts
	sort.Sort(def.WattsSorter(tasks))

	logFile, err := os.Create("./" + schedTracePrefix + "_schedTrace.log")
	if err != nil {
		log.Fatal(err)
	}

	s := &FirstFitSortedWattsProacCC{
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
		taskMonitor:    make(map[string][]def.Task),
		availablePower: make(map[string]float64),
		totalPower:     make(map[string]float64),
		capper:         powCap.GetClusterwideCapperInstance(),
		ticker:         time.NewTicker(10 * time.Second),
		recapTicker:    time.NewTicker(20 * time.Second),
		isCapping:      false,
		isRecapping:    false,
	}
	return s
}

// mutex
var rankedMutex sync.Mutex

func (s *FirstFitSortedWattsProacCC) newTask(offer *mesos.Offer, task def.Task) *mesos.TaskInfo {
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
			resources = append(resources, mesosutil.NewScalarResource("watts", wattsToConsider))
		} else {
			// Error in determining wattsToConsider
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

func (s *FirstFitSortedWattsProacCC) Disconnected(sched.SchedulerDriver) {
	// Need to stop the capping process.
	s.ticker.Stop()
	s.recapTicker.Stop()
	rankedMutex.Lock()
	s.isCapping = false
	rankedMutex.Unlock()
	log.Println("Framework disconnected with master")
}

// go routine to cap the entire cluster in regular intervals of time.
var rankedCurrentCapValue = 0.0 // initial value to indicate that we haven't capped the cluster yet.
func (s *FirstFitSortedWattsProacCC) startCapping() {
	go func() {
		for {
			select {
			case <-s.ticker.C:
				// Need to cap the cluster to the rankedCurrentCapValue.
				rankedMutex.Lock()
				if rankedCurrentCapValue > 0.0 {
					for _, host := range constants.Hosts {
						// Rounding currentCapValue to the nearest int.
						if err := rapl.Cap(host, "rapl", int(math.Floor(rankedCurrentCapValue+0.5))); err != nil {
							log.Println(err)
						}
					}
					log.Printf("Capped the cluster to %d", int(math.Floor(rankedCurrentCapValue+0.5)))
				}
				rankedMutex.Unlock()
			}
		}
	}()
}

// go routine to cap the entire cluster in regular intervals of time.
var rankedRecapValue = 0.0 // The cluster wide cap value when recapping.
func (s *FirstFitSortedWattsProacCC) startRecapping() {
	go func() {
		for {
			select {
			case <-s.recapTicker.C:
				rankedMutex.Lock()
				// If stopped performing cluster wide capping then we need to explicitly cap the entire cluster.
				if s.isRecapping && rankedRecapValue > 0.0 {
					for _, host := range constants.Hosts {
						// Rounding currentCapValue to the nearest int.
						if err := rapl.Cap(host, "rapl", int(math.Floor(rankedRecapValue+0.5))); err != nil {
							log.Println(err)
						}
					}
					log.Printf("Recapped the cluster to %d", int(math.Floor(rankedRecapValue+0.5)))
				}
				// setting recapping to false
				s.isRecapping = false
				rankedMutex.Unlock()
			}
		}
	}()
}

// Stop cluster wide capping
func (s *FirstFitSortedWattsProacCC) stopCapping() {
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
func (s *FirstFitSortedWattsProacCC) stopRecapping() {
	// If not capping, then definitely recapping.
	if !s.isCapping && s.isRecapping {
		log.Println("Stopping the cluster wide re-capping.")
		s.recapTicker.Stop()
		fcfsMutex.Lock()
		s.isRecapping = false
		fcfsMutex.Unlock()
	}
}

func (s *FirstFitSortedWattsProacCC) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
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
			Ranked cluster wide capping strategy

			For each task in the sorted tasks,
				1. Need to check whether the offer can be taken or not (based on CPU, RAM and WATTS requirements).
				2. If the task fits the offer, then need to determine the cluster wide cap.'
				3. rankedCurrentCapValue is updated with the determined cluster wide cap.

			Once we are done scheduling all the tasks,
				we start recalculating the cluster wide cap each time a task finishes.

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
				// Capping the cluster if haven't yet started
				if !s.isCapping {
					rankedMutex.Lock()
					s.isCapping = true
					rankedMutex.Unlock()
					s.startCapping()
				}
				offerTaken = true
				tempCap, err := s.capper.FCFSDeterminedCap(s.totalPower, &task)

				if err == nil {
					rankedMutex.Lock()
					rankedCurrentCapValue = tempCap
					rankedMutex.Unlock()
				} else {
					log.Println("Failed to determine the new cluster wide cap: ", err)
				}
				log.Printf("Starting on [%s]\n", offer.GetHostname())
				taskToSchedule := s.newTask(offer, task)
				to_schedule := []*mesos.TaskInfo{taskToSchedule}
				driver.LaunchTasks([]*mesos.OfferID{offer.Id}, to_schedule, mesosUtils.DefaultFilter)
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
						s.startRecapping()
						close(s.Shutdown)
					}
				}
				break // Offer taken, move on.
			} else {
				// Task doesn't fit the offer. Move onto the next offer.
			}
		}

		// If no tasks fit the offer, then declining the offer.
		if !offerTaken {
			log.Printf("There is not enough resources to launch a task on Host: %s\n", offer.GetHostname())
			cpus, mem, watts := offerUtils.OfferAgg(offer)

			log.Printf("<CPU: %f, RAM: %f, Watts: %f>\n", cpus, mem, watts)
			driver.DeclineOffer(offer.Id, mesosUtils.DefaultFilter)
		}
	}
}

func (s *FirstFitSortedWattsProacCC) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Printf("Received task status [%s] for task [%s]\n", NameFor(status.State), *status.TaskId.Value)

	if *status.State == mesos.TaskState_TASK_RUNNING {
		rankedMutex.Lock()
		s.tasksRunning++
		rankedMutex.Unlock()
	} else if IsTerminal(status.State) {
		delete(s.running[status.GetSlaveId().GoString()], *status.TaskId.Value)
		rankedMutex.Lock()
		s.tasksRunning--
		rankedMutex.Unlock()
		if s.tasksRunning == 0 {
			select {
			case <-s.Shutdown:
				// Need to stop the recapping process.
				s.stopRecapping()
				close(s.Done)
			default:
			}
		} else {
			// Need to remove the task from the window
			s.capper.TaskFinished(*status.TaskId.Value)
			// Determining the new cluster wide cap.
			//tempCap, err := s.capper.NaiveRecap(s.totalPower, s.taskMonitor, *status.TaskId.Value)
			tempCap, err := s.capper.CleverRecap(s.totalPower, s.taskMonitor, *status.TaskId.Value)

			if err == nil {
				// If new determined cap value is different from the current recap value then we need to recap.
				if int(math.Floor(tempCap+0.5)) != int(math.Floor(rankedRecapValue+0.5)) {
					rankedRecapValue = tempCap
					rankedMutex.Lock()
					s.isRecapping = true
					rankedMutex.Unlock()
					log.Printf("Determined re-cap value: %f\n", rankedRecapValue)
				} else {
					rankedMutex.Lock()
					s.isRecapping = false
					rankedMutex.Unlock()
				}
			} else {
				// Not updating rankedCurrentCapValue
				log.Println(err)
			}
		}
	}
	log.Printf("DONE: Task status [%s] for task [%s]", NameFor(status.State), *status.TaskId.Value)
}
