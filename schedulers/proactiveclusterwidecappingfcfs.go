package schedulers

import (
  "bitbucket.org/sunybingcloud/electron/def"
  "bitbucket.org/sunybingcloud/electron/constants"
  "bitbucket.org/sunybingcloud/electron/rapl"
  "fmt"
  "github.com/golang/protobuf/proto"
  mesos "github.com/mesos/mesos-go/mesosproto"
  "github.com/mesos/mesos-go/mesosutil"
  sched "github.com/mesos/mesos-go/scheduler"
  "log"
  "math"
  "strings"
  "time"
)

// Decides if to take an offer or not
func (_ *ProactiveClusterwideCapFCFS) takeOffer(offer *mesos.Offer, task def.Task) bool {
  offer_cpu, offer_mem, _ := OfferAgg(offer)

  if offer_cpu >= task.CPU && offer_mem >= task.RAM {
    return true
  }
  return false
}

// electronScheduler implements the Scheduler interface.
type ProactiveClusterwideCapFCFS struct {
  tasksCreated  int
  tasksRunning  int
  tasks         []def.Task
  metrics       map[string]def.Metric
  running       map[string]map[string]bool
  taskMonitor   map[string][]def.Task // store tasks that are currently running.
  availablePower map[string]float64 // available power for each node in the cluster.
  totalPower   map[string]float64 // total power for each node in the cluster.
  ignoreWatts   bool
  capper        *clusterwideCapper
  ticker        *time.Ticker
  isCapping     bool // indicate whether we are currently performing cluster wide capping.
  //lock          *sync.Mutex

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
func NewProactiveClusterwideCapFCFS(tasks []def.Task, ignoreWatts bool) *ProactiveClusterwideCapFCFS {
  s := &ProactiveClusterwideCapFCFS {
    tasks:                      tasks,
    ignoreWatts:                ignoreWatts,
    Shutdown:                   make(chan struct{}),
    Done:                       make(chan struct{}),
    PCPLog:                     make(chan struct{}),
    running:                    make(map[string]map[string]bool),
    taskMonitor:                make(map[string][]def.Task),
    availablePower:             make(map[string]float64),
    totalPower:                 make(map[string]float64),
    RecordPCP:                  false,
    capper:                     getClusterwideCapperInstance(),
    ticker:                     time.NewTicker(5 * time.Second),
    isCapping:                  false,
    //lock:                       new(sync.Mutex),
  }
  return s
}

func (s *ProactiveClusterwideCapFCFS) newTask(offer *mesos.Offer, task def.Task) *mesos.TaskInfo {
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
  task.SetTaskID(*proto.String(taskName))
  // Add task to the list of tasks running on the node.
  s.running[offer.GetSlaveId().GoString()][taskName] = true
  s.taskMonitor[offer.GetSlaveId().GoString()] = []def.Task{task}

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

func (s *ProactiveClusterwideCapFCFS) Registered(
  _ sched.SchedulerDriver,
  frameworkID *mesos.FrameworkID,
  masterInfo *mesos.MasterInfo) {
  log.Printf("Framework %s registered with master %s", frameworkID, masterInfo)
}

func (s *ProactiveClusterwideCapFCFS) Reregistered(_ sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
  log.Printf("Framework re-registered with master %s", masterInfo)
}

func (s *ProactiveClusterwideCapFCFS) Disconnected(sched.SchedulerDriver) {
  // Need to stop the capping process.
  s.ticker.Stop()
  s.isCapping = false
  log.Println("Framework disconnected with master")
}

// go routine to cap the entire cluster in regular intervals of time.
var currentCapValue = 0.0 // initial value to indicate that we haven't capped the cluster yet.
func (s *ProactiveClusterwideCapFCFS) startCapping() {
  go func() {
    for {
      select {
        case <- s.ticker.C:
          // Need to cap the cluster to the currentCapValue.
          if currentCapValue > 0.0 {
            //mutex.Lock()
            //s.lock.Lock()
            for _, host := range constants.Hosts {
              // Rounding curreCapValue to the nearest int.
              if err := rapl.Cap(host, "rapl", int(math.Floor(currentCapValue + 0.5))); err != nil {
                fmt.Println(err)
              } else {
                fmt.Printf("Successfully capped %s to %f%\n", host, currentCapValue)
              }
            }
            //mutex.Unlock()
            //s.lock.Unlock()
          }
      }
    }
  }()
}

// Stop cluster wide capping
func (s *ProactiveClusterwideCapFCFS) stopCapping() {
  if s.isCapping {
    log.Println("Stopping the cluster wide capping.")
    s.ticker.Stop()
    s.isCapping = false
  }
}

// TODO: Need to reduce the time complexity: looping over offers twice (Possible to do it just once?).
func (s *ProactiveClusterwideCapFCFS) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
  log.Printf("Received %d resource offers", len(offers))

  // retrieving the available power for all the hosts in the offers.
  for _, offer := range offers {
    _, _, offer_watts := OfferAgg(offer)
    s.availablePower[*offer.Hostname] = offer_watts
    // setting total power if the first time.
    if _, ok := s.totalPower[*offer.Hostname]; !ok {
      s.totalPower[*offer.Hostname] = offer_watts
    }
  }

  for host, tpower := range s.totalPower {
    fmt.Printf("TotalPower[%s] = %f\n", host, tpower)
  }
  for host, apower := range s.availablePower {
    fmt.Printf("AvailablePower[%s] = %f\n", host, apower)
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

    /*
    Clusterwide Capping strategy

    For each task in s.tasks,
      1. Need to check whether the offer can be taken or not (based on CPU and RAM requirements).
      2. If the tasks fits the offer, then I need to detemrine the cluster wide cap.
      3. currentCapValue is updated with the determined cluster wide cap.

    Cluster wide capping is currently performed at regular intervals of time.
    TODO: We can choose to cap the cluster only if the clusterwide cap varies more than the current clusterwide cap.
      Although this sounds like a better approach, it only works when the resource requirements of neighbouring tasks are similar.
    */
    //offer_cpu, offer_ram, _ := OfferAgg(offer)

    taken := false
    //var mutex sync.Mutex

    for i, task := range s.tasks {
      // Don't take offer if it doesn't match our task's host requirement.
      if !strings.HasPrefix(*offer.Hostname, task.Host) {
        continue
      }

      // Does the task fit.
      if s.takeOffer(offer, task) {
        // Capping the cluster if haven't yet started,
        if !s.isCapping {
          s.startCapping()
          s.isCapping = true
        }
        taken = true
        //mutex.Lock()
        //s.lock.Lock()
        //tempCap, err := s.capper.fcfsDetermineCap(s.availablePower, &task)
        tempCap, err := s.capper.fcfsDetermineCap(s.totalPower, &task)

        if err == nil {
          currentCapValue = tempCap
        } else {
          fmt.Printf("Failed to determine new cluster wide cap: ")
          fmt.Println(err)
        }
        //mutex.Unlock()
        //s.lock.Unlock()
        fmt.Printf("Starting on [%s]\n", offer.GetHostname())
        to_schedule := []*mesos.TaskInfo{s.newTask(offer, task)}
        driver.LaunchTasks([]*mesos.OfferID{offer.Id}, to_schedule, defaultFilter)
        fmt.Printf("Inst: %d", *task.Instances)
        *task.Instances--
        if *task.Instances <= 0 {
          // All instances of the task have been scheduled. Need to remove it from the list of tasks to schedule.
          s.tasks[i] = s.tasks[len(s.tasks)-1]
					s.tasks = s.tasks[:len(s.tasks)-1]

					if len(s.tasks) <= 0 {
						log.Println("Done scheduling all tasks")
            // Need to stop the cluster wide capping as there aren't any more tasks to schedule.
            s.stopCapping()
						close(s.Shutdown)
					}
        }
        break // Offer taken, move on.
      } else {
        // Task doesn't fit the offer. Move onto the next offer.
      }
    }

    // If no task fit the offer, then declining the offer.
    if !taken {
      fmt.Printf("There is not enough resources to launch a task on Host: %s\n", offer.GetHostname())
      cpus, mem, watts := OfferAgg(offer)

      log.Printf("<CPU: %f, RAM: %f, Watts: %f>\n", cpus, mem, watts)
      driver.DeclineOffer(offer.Id, defaultFilter)
    }
  }
}

func (s *ProactiveClusterwideCapFCFS) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
  log.Printf("Received task status [%s] for task [%s]", NameFor(status.State), *status.TaskId.Value)

	if *status.State == mesos.TaskState_TASK_RUNNING {
		s.tasksRunning++
	} else if IsTerminal(status.State) {
		delete(s.running[status.GetSlaveId().GoString()], *status.TaskId.Value)
    // Need to remove the task from the window of tasks.
    s.capper.taskFinished(*status.TaskId.Value)
    //currentCapValue, _ = s.capper.recap(s.availablePower, s.taskMonitor, *status.TaskId.Value)
    // Determining the new cluster wide cap.
    currentCapValue, _ = s.capper.recap(s.totalPower, s.taskMonitor, *status.TaskId.Value)
    log.Printf("Recapping the cluster to %f\n", currentCapValue)

		s.tasksRunning--
		if s.tasksRunning == 0 {
			select {
			case <-s.Shutdown:
        // Need to stop the capping process.
        s.stopCapping()
				close(s.Done)
			default:
			}
		}
	}
	log.Printf("DONE: Task status [%s] for task [%s]", NameFor(status.State), *status.TaskId.Value)
}

func (s *ProactiveClusterwideCapFCFS) FrameworkMessage(driver sched.SchedulerDriver,
  executorID *mesos.ExecutorID,
  slaveID *mesos.SlaveID,
  message string) {

	log.Println("Getting a framework message: ", message)
	log.Printf("Received a framework message from some unknown source: %s", *executorID.Value)
}

func (s *ProactiveClusterwideCapFCFS) OfferRescinded(_ sched.SchedulerDriver, offerID *mesos.OfferID) {
  log.Printf("Offer %s rescinded", offerID)
}

func (s *ProactiveClusterwideCapFCFS) SlaveLost(_ sched.SchedulerDriver, slaveID *mesos.SlaveID) {
  log.Printf("Slave %s lost", slaveID)
}

func (s *ProactiveClusterwideCapFCFS) ExecutorLost(_ sched.SchedulerDriver, executorID *mesos.ExecutorID, slaveID *mesos.SlaveID, status int) {
	log.Printf("Executor %s on slave %s was lost", executorID, slaveID)
}

func (s *ProactiveClusterwideCapFCFS) Error(_ sched.SchedulerDriver, err string) {
	log.Printf("Receiving an error: %s", err)
}
