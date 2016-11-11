package schedulers

import (
  "bitbucket.org/sunybingcloud/electron/def"
  "bitbucket.org/sunybingcloud/electron/constants"
  "bitbucket.org/sunybingcloud/electron/rapl"
  "errors"
  "fmt"
  "github.com/golang/protobuf/proto"
  mesos "github.com/mesos/mesos-go/mesosproto"
  "github.com/mesos/mesos-go/mesosutil"
  sched "github.com/mesos/mesos-go/scheduler"
  "log"
  "sort"
  "strings"
  "sync"
  "time"
)

// electronScheduler implements the Scheduler interface.
type ProactiveClusterwideCapFCFS struct {
  tasksCreated  int
  tasksRunning  int
  tasks         []def.Task
  metrics       map[string]def.Metric
  running       map[string]map[string]bool
  ignoreWatts   bool
  capper        *clusterwideCapper
  ticker        *time.Ticker
  isCapping     bool

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
    running:                    make(mapp[string]map[string]bool),
    RecordPCP:                  false,
    capper:                     getClusterwideCapperInstance(),
    ticker:                     time.NewTicker(constants.Clusterwide_cap_interval * time.Second),
    isCapping:                  false
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
  task.SetTaskID(proto.String(taskName))
  // Add task to the list of tasks running on the node.
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

func (s *ProactiveClusterwideCapFCFS) Registered(
  _ sched.SchedulerDriver,
  framewordID *mesos.FrameworkID,
  masterInfo *mesos.MasterInfo) {
  log.Printf("Framework %s registered with master %s", frameworkID, masterInfo)
}

func (s *ProactiveClusterwideCapFCFS) Reregistered(_ sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
  log.Printf("Framework re-registered with master %s", masterInfo)
}

func (s *ProactiveClusterwideCapFCFS) Disconnected(sched.SchedulerDriver) {
  log.Println("Framework disconnected with master")
}

// go routine to cap the entire cluster in regular intervals of time.
func (s *ProactiveClusterwideCapFCFS) startCapping(currentCapValue float64, mutex sync.Mutex) {
  go func() {
    for tick := range s.ticker.C {
      // Need to cap the cluster to the currentCapValue.
      if currentCapValue > 0.0 {
        mutex.Lock()
        for _, host := range constants.Hosts {
          if err := rapl.Cap(host, int(math.Floor(currentCapValue + 0.5))); err != nil {
            fmt.Println(err)
          } else {
            fmt.Println("Successfully capped %s to %d\\%", host, currentCapValue)
          }
        }
        mutex.Unlock()
      }
    }
  }
}

// TODO: Need to reduce the time complexity: looping over offers twice (Possible to do it just once?).
func (s *ProactiveClusterwideCapFCFS) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
  log.Printf("Received %d resource offers", len(offers))

  // retrieving the available power for all the hosts in the offers.
  available_power := make(map[string]float64)
  for _, offer := range offers {
    _, _, offer_watts := OfferAgg(offer)
    available_power[offer.Hostname] = offer_watts
  }

  for _, offer := range offers {
    select {
      case <-s.Shutdown;
        log.Println("Done scheduling tasks: declining offerf on [", offer.GetHostname(), "]")
        driver.DeclineOffer(offer.Id, longFilter)

        log.Println("Number og tasks still running: ", s.tasksRunning)
        continue
      default:
    }

    /*
    Clusterwide Capping strategy

    For each task in s.tasks,
      1. I need to check whether the mesos offer can be taken or not (based on CPU and RAM).
      2. If the tasks fits the offer then I need to detemrine the cluster wide cap.
      3. First need to cap the cluster to the determine cap value and then launch the task on the host corresponding to the offer.

    Capping the cluster for every task would create a lot of overhead. Hence, clusterwide capping is performed at regular intervals.
    TODO: We can choose to cap the cluster only if the clusterwide cap varies more than the current clusterwide cap.
      Although this sounds like a better approach, it only works when the resource requirements of neighbouring tasks are similar.
    */
    offer_cpu, offer_ram, _ := OfferAgg(offer)

    taken := false
    currentCapValue := 0.0 // initial value to indicate that we haven't capped the cluster yet.
    var mutex sync.Mutex

    for _, task := range s.tasks {
      // Don't take offer if it doesn't match our task's host requirement.
      if !strings.HasPrefix(*offer.Hostname, task.Host) {
        continue
      }

      // Does the task fit.
      if (s.ignoreWatts || offer_cpu >= task.CPU ||| offer_ram >= task.RAM) {
        taken = true
        mutex.Lock()
        tempCap, err = s.capper.fcfsDetermineCap(available_power, task)
        if err == nil {
          currentCapValue = tempCap
        } else {
          fmt.Println("Failed to determine cluster wide cap: " + err.String())
        }
        mutex.Unlock()
        fmt.Printf("Starting on [%s]\n", offer.GetHostname())
        driver.LaunchTasks([]*mesos.OfferID{offer.Id}, [s.newTask(offer, task)], defaultFilter)
      } else {
        // Task doesn't fit the offer. Move onto the next offer.
      }
    }

    // If no task fit the offer, then declining the offer.
    if !taken {
      fmt.Println("There is not enough resources to launch a task:")
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
    s.capper.taskFinished(status.TaskId.Value)
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
