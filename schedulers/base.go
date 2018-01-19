package schedulers

import (
	"bitbucket.org/sunybingcloud/elektron/def"
	elecLogDef "bitbucket.org/sunybingcloud/elektron/logging/def"
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	"log"
	"sync"
	"time"
)

type baseScheduler struct {
	ElectronScheduler
	SchedPolicyContext
	// Current scheduling policy used for resource offer consumption.
	curSchedPolicy SchedPolicyState

	tasksCreated     int
	tasksRunning     int
	tasks            []def.Task
	metrics          map[string]def.Metric
	running          map[string]map[string]bool
	wattsAsAResource bool
	classMapWatts    bool

	// First set of PCP values are garbage values, signal to logger to start recording when we're
	// about to schedule a new task
	RecordPCP *bool

	// This channel is closed when the program receives an interrupt,
	// signalling that the program should shut down.
	Shutdown chan struct{}
	// This channel is closed after shutdown is closed, and only when all
	// outstanding tasks have been cleaned up.
	Done chan struct{}

	// Controls when to shutdown pcp logging.
	PCPLog chan struct{}

	schedTrace *log.Logger

	// Send the type of the message to be logged
	logMsgType chan elecLogDef.LogMessageType
	// Send the message to be logged
	logMsg chan string

	mutex sync.Mutex

	// Whether switching of scheduling policies at runtime has been enabled
	schedPolSwitchEnabled bool
}

func (s *baseScheduler) init(opts ...schedPolicyOption) {
	for _, opt := range opts {
		// applying options
		if err := opt(s); err != nil {
			log.Fatal(err)
		}
	}
	s.running = make(map[string]map[string]bool)
	s.mutex = sync.Mutex{}
}

func (s *baseScheduler) SwitchSchedPol(newSchedPol SchedPolicyState) {
	s.curSchedPolicy = newSchedPol
}

func (s *baseScheduler) newTask(offer *mesos.Offer, task def.Task) *mesos.TaskInfo {
	taskName := fmt.Sprintf("%s-%d", task.Name, *task.Instances)
	s.tasksCreated++

	if !*s.RecordPCP {
		// Turn on elecLogDef
		*s.RecordPCP = true
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

	if s.wattsAsAResource {
		if wattsToConsider, err := def.WattsToConsider(task, s.classMapWatts, offer); err == nil {
			s.LogTaskWattsConsideration(task, *offer.Hostname, wattsToConsider)
			resources = append(resources, mesosutil.NewScalarResource("watts", wattsToConsider))
		} else {
			// Error in determining wattsConsideration
			s.LogElectronError(err)
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

func (s *baseScheduler) OfferRescinded(_ sched.SchedulerDriver, offerID *mesos.OfferID) {
	s.LogOfferRescinded(offerID)
}
func (s *baseScheduler) SlaveLost(_ sched.SchedulerDriver, slaveID *mesos.SlaveID) {
	s.LogSlaveLost(slaveID)
}
func (s *baseScheduler) ExecutorLost(_ sched.SchedulerDriver, executorID *mesos.ExecutorID,
	slaveID *mesos.SlaveID, status int) {
	s.LogExecutorLost(executorID, slaveID)
}

func (s *baseScheduler) Error(_ sched.SchedulerDriver, err string) {
	s.LogMesosError(err)
}

func (s *baseScheduler) FrameworkMessage(
	driver sched.SchedulerDriver,
	executorID *mesos.ExecutorID,
	slaveID *mesos.SlaveID,
	message string) {
	s.LogFrameworkMessage(executorID, slaveID, message)
}

func (s *baseScheduler) Registered(
	_ sched.SchedulerDriver,
	frameworkID *mesos.FrameworkID,
	masterInfo *mesos.MasterInfo) {
	s.LogFrameworkRegistered(frameworkID, masterInfo)
}

func (s *baseScheduler) Reregistered(_ sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	s.LogFrameworkReregistered(masterInfo)
}

func (s *baseScheduler) Disconnected(sched.SchedulerDriver) {
	s.LogDisconnected()
}

func (s *baseScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	s.curSchedPolicy.ConsumeOffers(s, driver, offers)
}

func (s *baseScheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	s.LogTaskStatusUpdate(status)
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
}

func (s *baseScheduler) Log(lmt elecLogDef.LogMessageType, msg string) {
	s.mutex.Lock()
	s.logMsgType <- lmt
	s.logMsg <- msg
	s.mutex.Unlock()
}

func (s *baseScheduler) LogTaskStarting(ts *def.Task, offer *mesos.Offer) {
	lmt := elecLogDef.GENERAL
	msgColor := elecLogDef.LogMessageColors[lmt]
	var msg string
	if ts == nil {
		msg = msgColor.Sprintf("TASKS STARTING... host = [%s]", offer.GetHostname())
	} else {
		msg = msgColor.Sprintf("TASK STARTING... task = [%s], Instance = %d, host = [%s]",
			ts.Name, *ts.Instances, offer.GetHostname())
	}
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogTaskWattsConsideration(ts def.Task, host string, wattsToConsider float64) {
	lmt := elecLogDef.GENERAL
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := msgColor.Sprintf("Watts considered for task[%s] and host[%s] = %f Watts",
		ts.Name, host, wattsToConsider)
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogOffersReceived(offers []*mesos.Offer) {
	lmt := elecLogDef.GENERAL
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := msgColor.Sprintf("Received %d resource offers", len(offers))
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogNoPendingTasksDeclineOffers(offer *mesos.Offer) {
	lmt := elecLogDef.WARNING
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := msgColor.Sprintf("DECLINING OFFER for host[%s]... "+
		"No tasks left to schedule", offer.GetHostname())
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogNumberOfRunningTasks() {
	lmt := elecLogDef.GENERAL
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := msgColor.Sprintf("Number of tasks still running = %d", s.tasksRunning)
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogCoLocatedTasks(slaveID string) {
	lmt := elecLogDef.GENERAL
	msgColor := elecLogDef.LogMessageColors[lmt]
	buffer := bytes.Buffer{}
	buffer.WriteString(fmt.Sprintln("Colocated with:"))
	for taskName := range s.running[slaveID] {
		buffer.WriteString(fmt.Sprintln(taskName))
	}
	msg := msgColor.Sprintf(buffer.String())
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogSchedTrace(taskToSchedule *mesos.TaskInfo, offer *mesos.Offer) {
	msg := fmt.Sprint(offer.GetHostname() + ":" + taskToSchedule.GetTaskId().GetValue())
	s.Log(elecLogDef.SCHED_TRACE, msg)
}

func (s *baseScheduler) LogTerminateScheduler() {
	lmt := elecLogDef.GENERAL
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := msgColor.Sprint("Done scheduling all tasks!")
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogInsufficientResourcesDeclineOffer(offer *mesos.Offer,
	offerResources ...interface{}) {
	lmt := elecLogDef.WARNING
	msgColor := elecLogDef.LogMessageColors[lmt]
	buffer := bytes.Buffer{}
	buffer.WriteString(fmt.Sprintln("DECLINING OFFER... Offer has insufficient resources to launch a task"))
	buffer.WriteString(fmt.Sprintf("Offer Resources <CPU: %f, RAM: %f, Watts: %f>", offerResources...))
	msg := msgColor.Sprint(buffer.String())
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogOfferRescinded(offerID *mesos.OfferID) {
	lmt := elecLogDef.ERROR
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := msgColor.Sprintf("OFFER RESCINDED: OfferID = %s", offerID)
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogSlaveLost(slaveID *mesos.SlaveID) {
	lmt := elecLogDef.ERROR
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := msgColor.Sprintf("SLAVE LOST: SlaveID = %s", slaveID)
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogExecutorLost(executorID *mesos.ExecutorID, slaveID *mesos.SlaveID) {
	lmt := elecLogDef.ERROR
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := msgColor.Sprintf("EXECUTOR LOST: ExecutorID = %s, SlaveID = %s", executorID, slaveID)
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogFrameworkMessage(executorID *mesos.ExecutorID,
	slaveID *mesos.SlaveID, message string) {
	lmt := elecLogDef.GENERAL
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := msgColor.Sprintf("Received Framework message from executor [%s]: %s", executorID, message)
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogMesosError(err string) {
	lmt := elecLogDef.ERROR
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := msgColor.Sprintf("MESOS ERROR: %s", err)
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogElectronError(err error) {
	lmt := elecLogDef.ERROR
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := msgColor.Sprintf("ELECTRON ERROR: %v", err)
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogFrameworkRegistered(frameworkID *mesos.FrameworkID,
	masterInfo *mesos.MasterInfo) {
	lmt := elecLogDef.SUCCESS
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := msgColor.Sprintf("FRAMEWORK REGISTERED! frameworkID = %s, master = %s",
		frameworkID, masterInfo)
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogFrameworkReregistered(masterInfo *mesos.MasterInfo) {
	lmt := elecLogDef.GENERAL
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := msgColor.Sprintf("Framework re-registered with master %s", masterInfo)
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogDisconnected() {
	lmt := elecLogDef.WARNING
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := msgColor.Sprint("Framework disconnected with master")
	s.Log(lmt, msg)
}

func (s *baseScheduler) LogTaskStatusUpdate(status *mesos.TaskStatus) {
	var lmt elecLogDef.LogMessageType
	switch *status.State {
	case mesos.TaskState_TASK_ERROR, mesos.TaskState_TASK_FAILED,
		mesos.TaskState_TASK_KILLED, mesos.TaskState_TASK_LOST:
		lmt = elecLogDef.ERROR
	case mesos.TaskState_TASK_FINISHED:
		lmt = elecLogDef.SUCCESS
	default:
		lmt = elecLogDef.GENERAL
	}
	msgColor := elecLogDef.LogMessageColors[lmt]
	msg := elecLogDef.LogMessageColors[elecLogDef.GENERAL].Sprintf("Task Status received for task [%s] --> %s",
		*status.TaskId.Value, msgColor.Sprint(NameFor(status.State)))
	s.Log(lmt, msg)
}
