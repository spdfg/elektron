// Copyright (C) 2018 spdfg
//
// This file is part of Elektron.
//
// Elektron is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Elektron is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Elektron.  If not, see <http://www.gnu.org/licenses/>.
//

package schedulers

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	"github.com/mesos/mesos-go/api/v0/mesosutil"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	log "github.com/sirupsen/logrus"
	"github.com/spdfg/elektron/def"
	elekLog "github.com/spdfg/elektron/logging"
	. "github.com/spdfg/elektron/logging/types"
	"github.com/spdfg/elektron/utilities"
	"github.com/spdfg/elektron/utilities/schedUtils"
)

type BaseScheduler struct {
	ElectronScheduler
	SchedPolicyContext
	// Current scheduling policy used for resource offer consumption.
	curSchedPolicy SchedPolicyState

	tasksCreated                      int
	tasksRunning                      int
	tasks                             []def.Task
	metrics                           map[string]def.Metric
	Running                           map[string]map[string]bool
	wattsAsAResource                  bool
	classMapWatts                     bool
	TasksRunningMutex                 sync.Mutex
	HostNameToSlaveID                 map[string]string
	totalResourceAvailabilityRecorded bool

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

	mutex sync.Mutex

	// Whether switching of scheduling policies at runtime has been enabled
	schedPolSwitchEnabled bool
	// Name of the first scheduling policy to be deployed, if provided.
	// This scheduling policy would be deployed first regardless of the distribution of tasks in the TaskQueue.
	// Note: Scheduling policy switching needs to be enabled.
	nameOfFstSchedPolToDeploy string
	// Scheduling policy switching criteria.
	schedPolSwitchCriteria string

	// Size of window of tasks that can be scheduled in the next offer cycle.
	// The window size can be adjusted to make the most use of every resource offer.
	// By default, the schedWindowSize would correspond to the number of remaining tasks that haven't yet
	// been scheduled.
	schedWindowSize int
	// Number of tasks in the window
	numTasksInSchedWindow int

	// Whether the scheduling window needs to be fixed.
	toFixSchedWindow bool // If yes, then schedWindowSize is initialized and kept unchanged.

	// Strategy to resize the schedulingWindow.
	schedWindowResStrategy schedUtils.SchedWindowResizingStrategy

	// Indicate whether the any resource offers from mesos have been received.
	hasReceivedResourceOffers bool
}

func (s *BaseScheduler) init(opts ...SchedulerOptions) {
	for _, opt := range opts {
		// applying options
		if err := opt(s); err != nil {
			log.Fatal(err)
		}
	}
	s.TasksRunningMutex.Lock()
	s.Running = make(map[string]map[string]bool)
	s.TasksRunningMutex.Unlock()
	s.HostNameToSlaveID = make(map[string]string)
	s.mutex = sync.Mutex{}
	s.schedWindowResStrategy = schedUtils.SchedWindowResizingCritToStrategy["fillNextOfferCycle"]
	// Initially no resource offers would have been received.
	s.hasReceivedResourceOffers = false
}

func (s *BaseScheduler) SwitchSchedPol(newSchedPol SchedPolicyState) {
	s.curSchedPolicy = newSchedPol
}

func (s *BaseScheduler) newTask(offer *mesos.Offer, task def.Task) *mesos.TaskInfo {
	taskName := fmt.Sprintf("%s-%d", task.Name, *task.Instances)
	s.tasksCreated++

	if !*s.RecordPCP {
		// Turn on elecLogDef
		*s.RecordPCP = true
		time.Sleep(1 * time.Second) // Make sure we're recording by the time the first task starts
	}

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

func (s *BaseScheduler) OfferRescinded(_ sched.SchedulerDriver, offerID *mesos.OfferID) {
	s.LogOfferRescinded(offerID)
}
func (s *BaseScheduler) SlaveLost(_ sched.SchedulerDriver, slaveID *mesos.SlaveID) {
	s.LogSlaveLost(slaveID)
}
func (s *BaseScheduler) ExecutorLost(_ sched.SchedulerDriver, executorID *mesos.ExecutorID,
	slaveID *mesos.SlaveID, status int) {
	s.LogExecutorLost(executorID, slaveID)
}

func (s *BaseScheduler) Error(_ sched.SchedulerDriver, err string) {
	s.LogMesosError(err)
}

func (s *BaseScheduler) FrameworkMessage(
	driver sched.SchedulerDriver,
	executorID *mesos.ExecutorID,
	slaveID *mesos.SlaveID,
	message string) {
	s.LogFrameworkMessage(executorID, slaveID, message)
}

func (s *BaseScheduler) Registered(
	_ sched.SchedulerDriver,
	frameworkID *mesos.FrameworkID,
	masterInfo *mesos.MasterInfo) {
	s.LogFrameworkRegistered(frameworkID, masterInfo)
}

func (s *BaseScheduler) Reregistered(_ sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	s.LogFrameworkReregistered(masterInfo)
}

func (s *BaseScheduler) Disconnected(sched.SchedulerDriver) {
	s.LogDisconnected()
}

func (s *BaseScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	// Recording the total amount of resources available across the cluster.
	utilities.RecordTotalResourceAvailability(offers)
	for _, offer := range offers {
		if _, ok := s.HostNameToSlaveID[offer.GetHostname()]; !ok {
			s.HostNameToSlaveID[offer.GetHostname()] = *offer.SlaveId.Value
		}
	}
	// Switch just before consuming the resource offers.
	s.curSchedPolicy.SwitchIfNecessary(s)
	//	s.Log(elecLogDef.GENERAL, fmt.Sprintf("SchedWindowSize[%d], #TasksInWindow[%d]",
	//		s.schedWindowSize, s.numTasksInSchedWindow))
	s.curSchedPolicy.ConsumeOffers(s, driver, offers)
	s.hasReceivedResourceOffers = true
}

func (s *BaseScheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	s.LogTaskStatusUpdate(status)
	if *status.State == mesos.TaskState_TASK_RUNNING {
		// If this is our first time running into this Agent
		s.TasksRunningMutex.Lock()
		if _, ok := s.Running[*status.SlaveId.Value]; !ok {
			s.Running[*status.SlaveId.Value] = make(map[string]bool)
		}
		// Add task to list of tasks running on node
		s.Running[*status.SlaveId.Value][*status.TaskId.Value] = true
		s.tasksRunning++
		s.TasksRunningMutex.Unlock()
	} else if IsTerminal(status.State) {
		// Update resource availability.
		utilities.ResourceAvailabilityUpdate("ON_TASK_TERMINAL_STATE",
			*status.TaskId, *status.SlaveId)
		s.TasksRunningMutex.Lock()
		delete(s.Running[*status.SlaveId.Value], *status.TaskId.Value)
		s.tasksRunning--
		s.TasksRunningMutex.Unlock()
		if s.tasksRunning == 0 {
			select {
			case <-s.Shutdown:
				close(s.Done)
			default:
			}
		}
	}
}

func (s *BaseScheduler) LogTaskStarting(ts *def.Task, offer *mesos.Offer) {
	if ts == nil {
		elekLog.WithField("host", offer.GetHostname()).Log(CONSOLE, log.InfoLevel, "TASKS STARTING...")
	} else {
		elekLog.WithFields(log.Fields{
			"task":     ts.Name,
			"Instance": fmt.Sprintf("%d", *ts.Instances),
			"host":     offer.GetHostname(),
		}).Log(CONSOLE, log.InfoLevel, "TASK STARTING... ")
	}
}

func (s *BaseScheduler) LogTaskWattsConsideration(ts def.Task, host string, wattsToConsider float64) {
	elekLog.WithFields(log.Fields{
		"task":  ts.Name,
		"host":  host,
		"Watts": fmt.Sprintf("%f", wattsToConsider),
	}).Log(CONSOLE, log.InfoLevel, "Watts considered for ")
}

func (s *BaseScheduler) LogOffersReceived(offers []*mesos.Offer) {
	elekLog.WithField("numOffers", fmt.Sprintf("%d", len(offers))).Log(CONSOLE, log.InfoLevel, "Resource offers received")
}

func (s *BaseScheduler) LogNoPendingTasksDeclineOffers(offer *mesos.Offer) {
	elekLog.Logf(CONSOLE, log.WarnLevel, "DECLINING OFFER for host %s. No tasks left to schedule", offer.GetHostname())
}

func (s *BaseScheduler) LogNumberOfRunningTasks() {
	elekLog.Logf(CONSOLE, log.InfoLevel, "Number of tasks still running %d", s.tasksRunning)
}

func (s *BaseScheduler) LogCoLocatedTasks(slaveID string) {
	buffer := bytes.Buffer{}
	s.TasksRunningMutex.Lock()
	for taskName := range s.Running[slaveID] {
		buffer.WriteString(fmt.Sprintln(taskName))
	}
	s.TasksRunningMutex.Unlock()
	elekLog.WithField("Tasks", buffer.String()).Log(CONSOLE, log.InfoLevel, "Colocated with")
}

func (s *BaseScheduler) LogSchedTrace(taskToSchedule *mesos.TaskInfo, offer *mesos.Offer) {
	elekLog.WithField(offer.GetHostname(), taskToSchedule.GetTaskId().GetValue()).Log(SCHED_TRACE, log.InfoLevel, "")
}

func (s *BaseScheduler) LogTerminateScheduler() {
	elekLog.Log(CONSOLE, log.InfoLevel, "Done scheduling all tasks!")
}

func (s *BaseScheduler) LogInsufficientResourcesDeclineOffer(offer *mesos.Offer,
	offerResources ...interface{}) {
	buffer := bytes.Buffer{}
	buffer.WriteString(fmt.Sprintf("<CPU: %f, RAM: %f, Watts: %f>", offerResources...))
	elekLog.WithField("Offer Resources", buffer.String()).Log(CONSOLE,
		log.WarnLevel, "DECLINING OFFER... Offer has insufficient resources to launch a task")
}

func (s *BaseScheduler) LogOfferRescinded(offerID *mesos.OfferID) {
	elekLog.WithField("OfferID", *offerID.Value).Log(CONSOLE, log.ErrorLevel, "OFFER RESCINDED")
}

func (s *BaseScheduler) LogSlaveLost(slaveID *mesos.SlaveID) {
	elekLog.WithField("SlaveID", *slaveID.Value).Log(CONSOLE, log.ErrorLevel, "SLAVE LOST")
}

func (s *BaseScheduler) LogExecutorLost(executorID *mesos.ExecutorID, slaveID *mesos.SlaveID) {
	elekLog.WithFields(log.Fields{
		"ExecutorID": executorID,
		"SlaveID":    slaveID,
	}).Log(CONSOLE, log.ErrorLevel, "EXECUTOR LOST")
}

func (s *BaseScheduler) LogFrameworkMessage(executorID *mesos.ExecutorID,
	slaveID *mesos.SlaveID, message string) {
	elekLog.Logf(CONSOLE, log.InfoLevel, "Received Framework message from executor %v", executorID)
}

func (s *BaseScheduler) LogMesosError(err string) {
	elekLog.Logf(CONSOLE, log.ErrorLevel, "MESOS CONSOLE %v", err)
}

func (s *BaseScheduler) LogElectronError(err error) {
	elekLog.Logf(CONSOLE, log.ErrorLevel, "ELEKTRON CONSOLE %v", err)
}

func (s *BaseScheduler) LogFrameworkRegistered(frameworkID *mesos.FrameworkID,
	masterInfo *mesos.MasterInfo) {
	elekLog.WithFields(log.Fields{
		"frameworkID": frameworkID,
		"master":      fmt.Sprintf("%v", masterInfo),
	}).Log(CONSOLE, log.InfoLevel, "FRAMEWORK REGISTERED!")
}

func (s *BaseScheduler) LogFrameworkReregistered(masterInfo *mesos.MasterInfo) {
	elekLog.WithField("master", fmt.Sprintf("%v", masterInfo)).Log(CONSOLE, log.InfoLevel, "Framework re-registered")
}

func (s *BaseScheduler) LogDisconnected() {
	elekLog.Log(CONSOLE, log.WarnLevel, "Framework disconnected with master")
}

func (s *BaseScheduler) LogTaskStatusUpdate(status *mesos.TaskStatus) {
	level := log.InfoLevel
	switch *status.State {
	case mesos.TaskState_TASK_ERROR, mesos.TaskState_TASK_FAILED,
		mesos.TaskState_TASK_KILLED, mesos.TaskState_TASK_LOST:
		level = log.ErrorLevel
	default:
		level = log.InfoLevel
	}
	elekLog.WithFields(log.Fields{
		"task":  *status.TaskId.Value,
		"state": NameFor(status.State),
	}).Log(CONSOLE, level, "Task Status received")
}

func (s *BaseScheduler) LogSchedPolicySwitch(name string, nextPolicy SchedPolicyState) {
	logSPS := func() {
		elekLog.WithField("Name", name).Log(SPS, log.InfoLevel, "")
	}
	if s.hasReceivedResourceOffers && (s.curSchedPolicy != nextPolicy) {
		logSPS()
	} else if !s.hasReceivedResourceOffers {
		logSPS()
	}
	// Logging the size of the scheduling window and the scheduling policy
	// 	that is going to schedule the tasks in the scheduling window.
	elekLog.WithFields(log.Fields{
		"Window size": fmt.Sprintf("%d", s.schedWindowSize),
		"Name":        name,
	}).Log(SCHED_WINDOW, log.InfoLevel, "")
}

func (s *BaseScheduler) LogClsfnAndTaskDistOverhead(overhead time.Duration) {
	// Logging the overhead in microseconds.
	elekLog.WithField("Overhead in microseconds", fmt.Sprintf("%f", float64(overhead.Nanoseconds())/1000.0)).Log(CLSFN_TASKDISTR_OVERHEAD, log.InfoLevel, "")
}
