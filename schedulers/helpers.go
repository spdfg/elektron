package schedulers

import (
	"bitbucket.org/sunybingcloud/elektron/constants"
	"bitbucket.org/sunybingcloud/elektron/def"
	elecLogDef "bitbucket.org/sunybingcloud/elektron/logging/def"
	"bitbucket.org/sunybingcloud/elektron/utilities"
	"bitbucket.org/sunybingcloud/elektron/utilities/mesosUtils"
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	"github.com/pkg/errors"
)

func coLocated(tasks map[string]bool, s BaseScheduler) {

	for task := range tasks {
		s.Log(elecLogDef.GENERAL, task)
	}

	s.Log(elecLogDef.GENERAL, "---------------------")
}

// Get the powerClass of the given hostname.
func hostToPowerClass(hostName string) string {
	for powerClass, hosts := range constants.PowerClasses {
		if _, ok := hosts[hostName]; ok {
			return powerClass
		}
	}
	return ""
}

// scheduler policy options to help initialize schedulers
type schedulerOptions func(e ElectronScheduler) error

func WithSchedPolicy(schedPolicyName string) schedulerOptions {
	return func(s ElectronScheduler) error {
		if schedPolicy, ok := SchedPolicies[schedPolicyName]; !ok {
			return errors.New("Incorrect scheduling policy.")
		} else {
			s.(*BaseScheduler).curSchedPolicy = schedPolicy
			return nil
		}
	}
}

func WithTasks(ts []def.Task) schedulerOptions {
	return func(s ElectronScheduler) error {
		if ts == nil {
			return errors.New("Task[] is empty.")
		} else {
			s.(*BaseScheduler).tasks = ts
			return nil
		}
	}
}

func WithWattsAsAResource(waar bool) schedulerOptions {
	return func(s ElectronScheduler) error {
		s.(*BaseScheduler).wattsAsAResource = waar
		return nil
	}
}

func WithClassMapWatts(cmw bool) schedulerOptions {
	return func(s ElectronScheduler) error {
		s.(*BaseScheduler).classMapWatts = cmw
		return nil
	}
}

func WithRecordPCP(recordPCP *bool) schedulerOptions {
	return func(s ElectronScheduler) error {
		s.(*BaseScheduler).RecordPCP = recordPCP
		return nil
	}
}

func WithShutdown(shutdown chan struct{}) schedulerOptions {
	return func(s ElectronScheduler) error {
		if shutdown == nil {
			return errors.New("Shutdown channel is nil.")
		} else {
			s.(*BaseScheduler).Shutdown = shutdown
			return nil
		}
	}
}

func WithDone(done chan struct{}) schedulerOptions {
	return func(s ElectronScheduler) error {
		if done == nil {
			return errors.New("Done channel is nil.")
		} else {
			s.(*BaseScheduler).Done = done
			return nil
		}
	}
}

func WithPCPLog(pcpLog chan struct{}) schedulerOptions {
	return func(s ElectronScheduler) error {
		if pcpLog == nil {
			return errors.New("PCPLog channel is nil.")
		} else {
			s.(*BaseScheduler).PCPLog = pcpLog
			return nil
		}
	}
}

func WithLoggingChannels(lmt chan elecLogDef.LogMessageType, msg chan string) schedulerOptions {
	return func(s ElectronScheduler) error {
		s.(*BaseScheduler).logMsgType = lmt
		s.(*BaseScheduler).logMsg = msg
		return nil
	}
}

func WithSchedPolSwitchEnabled(enableSchedPolicySwitch bool) schedulerOptions {
	return func(s ElectronScheduler) error {
		s.(*BaseScheduler).schedPolSwitchEnabled = enableSchedPolicySwitch
		return nil
	}
}

func WithNameOfFirstSchedPolToFix(nameOfFirstSchedPol string) schedulerOptions {
	return func(s ElectronScheduler) error {
		if nameOfFirstSchedPol == "" {
			lmt := elecLogDef.WARNING
			msgColor := elecLogDef.LogMessageColors[lmt]
			msg := msgColor.Sprintf("First scheduling policy to deploy not mentioned. This is now going to be determined at runtime.")
			s.(*BaseScheduler).Log(lmt, msg)
			return nil
		}
		if _, ok := SchedPolicies[nameOfFirstSchedPol]; !ok {
			return errors.New("Invalid name of scheduling policy.")
		}
		s.(*BaseScheduler).nameOfFstSchedPolToDeploy = nameOfFirstSchedPol
		return nil
	}
}

// Launch tasks.
func LaunchTasks(offerIDs []*mesos.OfferID, tasksToLaunch []*mesos.TaskInfo, driver sched.SchedulerDriver) {
	driver.LaunchTasks(offerIDs, tasksToLaunch, mesosUtils.DefaultFilter)
	// Update resource availability
	for _, task := range tasksToLaunch {
		utilities.ResourceAvailabilityUpdate("ON_TASK_ACTIVE_STATE", *task.TaskId, *task.SlaveId)
	}
}

// Sort N tasks in the TaskQueue
func SortNTasks(tasks []def.Task, n int, sb def.SortBy) {
	def.SortTasks(tasks[:n], sb)
}
