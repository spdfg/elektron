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
	"fmt"

	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spdfg/elektron/constants"
	"github.com/spdfg/elektron/def"
	elekLog "github.com/spdfg/elektron/logging"
	elekLogTypes "github.com/spdfg/elektron/logging/types"
	"github.com/spdfg/elektron/utilities"
	"github.com/spdfg/elektron/utilities/mesosUtils"
)

func coLocated(tasks map[string]bool, s BaseScheduler) {

	for _, task := range tasks {
		elekLog.ElektronLogger.WithFields(log.Fields{"Task": task}).Log(elekLogTypes.CONSOLE, log.InfoLevel, "")
	}

	elekLog.ElektronLogger.Log(elekLogTypes.CONSOLE, log.InfoLevel, "---------------------")
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
type SchedulerOptions func(e ElectronScheduler) error

func WithSchedPolicy(schedPolicyName string) SchedulerOptions {
	return func(s ElectronScheduler) error {
		if schedPolicy, ok := SchedPolicies[schedPolicyName]; !ok {
			return errors.New("Incorrect scheduling policy.")
		} else {
			s.(*BaseScheduler).curSchedPolicy = schedPolicy
			return nil
		}
	}
}

func WithTasks(ts []def.Task) SchedulerOptions {
	return func(s ElectronScheduler) error {
		if ts == nil {
			return errors.New("Task[] is empty.")
		} else {
			s.(*BaseScheduler).tasks = ts
			return nil
		}
	}
}

func WithWattsAsAResource(waar bool) SchedulerOptions {
	return func(s ElectronScheduler) error {
		s.(*BaseScheduler).wattsAsAResource = waar
		return nil
	}
}

func WithClassMapWatts(cmw bool) SchedulerOptions {
	return func(s ElectronScheduler) error {
		s.(*BaseScheduler).classMapWatts = cmw
		return nil
	}
}

func WithRecordPCP(recordPCP *bool) SchedulerOptions {
	return func(s ElectronScheduler) error {
		s.(*BaseScheduler).RecordPCP = recordPCP
		return nil
	}
}

func WithShutdown(shutdown chan struct{}) SchedulerOptions {
	return func(s ElectronScheduler) error {
		if shutdown == nil {
			return errors.New("Shutdown channel is nil.")
		} else {
			s.(*BaseScheduler).Shutdown = shutdown
			return nil
		}
	}
}

func WithDone(done chan struct{}) SchedulerOptions {
	return func(s ElectronScheduler) error {
		if done == nil {
			return errors.New("Done channel is nil.")
		} else {
			s.(*BaseScheduler).Done = done
			return nil
		}
	}
}

func WithPCPLog(pcpLog chan struct{}) SchedulerOptions {
	return func(s ElectronScheduler) error {
		if pcpLog == nil {
			return errors.New("PCPLog channel is nil.")
		} else {
			s.(*BaseScheduler).PCPLog = pcpLog
			return nil
		}
	}
}

func WithSchedPolSwitchEnabled(enableSchedPolicySwitch bool, switchingCriteria string) SchedulerOptions {
	return func(s ElectronScheduler) error {
		s.(*BaseScheduler).schedPolSwitchEnabled = enableSchedPolicySwitch
		// Checking if valid switching criteria.
		if _, ok := switchBasedOn[switchingCriteria]; !ok {
			return errors.New("Invalid scheduling policy switching criteria.")
		}
		s.(*BaseScheduler).schedPolSwitchCriteria = switchingCriteria
		return nil
	}
}

func WithNameOfFirstSchedPolToFix(nameOfFirstSchedPol string) SchedulerOptions {
	return func(s ElectronScheduler) error {
		if nameOfFirstSchedPol == "" {
			log.Println("First scheduling policy to deploy not mentioned. This is now" +
				" going to be determined at runtime.")
			return nil
		}
		if _, ok := SchedPolicies[nameOfFirstSchedPol]; !ok {
			return errors.New("Invalid name of scheduling policy.")
		}
		s.(*BaseScheduler).nameOfFstSchedPolToDeploy = nameOfFirstSchedPol
		return nil
	}
}

func WithFixedSchedulingWindow(toFixSchedWindow bool, fixedSchedWindowSize int) SchedulerOptions {
	return func(s ElectronScheduler) error {
		if toFixSchedWindow {
			if fixedSchedWindowSize <= 0 {
				return errors.New("Invalid value of scheduling window size. Please provide a value > 0.")
			}
			log.Println(fmt.Sprintf("Fixing the size of the scheduling window to %d.."+
				".", fixedSchedWindowSize))
			s.(*BaseScheduler).toFixSchedWindow = toFixSchedWindow
			s.(*BaseScheduler).schedWindowSize = fixedSchedWindowSize
		}
		// There shouldn't be any error for this scheduler option.
		// If fixSchedWindow is set to false, then the fixedSchedWindowSize would be ignored. In this case,
		// 	the size of the scheduling window would be determined at runtime.
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
