// Copyright (C) 2018 spdf
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
	"time"

	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	"gitlab.com/spdf/elektron/def"
	elekLogDef "gitlab.com/spdf/elektron/logging/def"
)

// Implements mesos scheduler.
type ElectronScheduler interface {
	sched.Scheduler
	init(opts ...SchedulerOptions)

	// Interface for log messages.
	// Every ElectronScheduler implementer should provide definitions for these functions.
	// This interface serves as a template to maintain consistent log messages.
	// Each of these functions are supposed to call the Log(...) that sends the
	//   log message type, and the log message to the corresponding channels.

	// Pass the logMessageType and the logMessage to the loggers for logging.
	Log(logMType elekLogDef.LogMessageType, logMsg string)
	// To be called when about to launch a task.
	// Log message indicating that a task is about to start executing.
	//   Also, log the host on which the task is going to be launched.
	LogTaskStarting(ts *def.Task, offer *mesos.Offer)
	// To be called when an offer is taken.
	// Log the chosen watts attribute for the task that has fit an offer.
	LogTaskWattsConsideration(ts def.Task, host string, wattsToConsider float64)
	// To be called when offers are received from Mesos.
	// Log the number of offers received and/or information about the received offers.
	LogOffersReceived(offers []*mesos.Offer)
	// To be called when a scheduling policy declines Mesos offers, as
	//   there are no tasks pending to be scheduled.
	// Log the host information corresponding to the offers that were declined.
	LogNoPendingTasksDeclineOffers(offers *mesos.Offer)
	// Log the number of tasks that are currently executing on the cluster.
	LogNumberOfRunningTasks()
	// To be called when a task fits a Mesos offer.
	// Log information on the tasks that the new task is going to be coLocated with.
	// Uses the coLocated(...) utility in helpers.go.
	LogCoLocatedTasks(slaveID string)
	// Log the scheduled trace of task.
	// The schedTrace includes the TaskID and the hostname of the node
	//   where is the task is going to be launched.
	LogSchedTrace(taskToSchedule *mesos.TaskInfo, offer *mesos.Offer)
	// To be called when all the tasks have completed executing.
	// Log message indicating that Electron has scheduled all the tasks.
	LogTerminateScheduler()
	// To be called when the offer is not consumed.
	// Log message to indicate that the offer had insufficient resources.
	LogInsufficientResourcesDeclineOffer(offer *mesos.Offer, offerResources ...interface{})
	// To be called when offer is rescinded by Mesos.
	LogOfferRescinded(offerID *mesos.OfferID)
	// To be called when Mesos agent is lost
	LogSlaveLost(slaveID *mesos.SlaveID)
	// To be called when executor lost.
	LogExecutorLost(executorID *mesos.ExecutorID, slaveID *mesos.SlaveID)
	// Log a mesos error
	LogMesosError(err string)
	// Log an Electron error
	LogElectronError(err error)
	// Log Framework message
	LogFrameworkMessage(executorID *mesos.ExecutorID, slaveID *mesos.SlaveID, message string)
	// Log Framework has been registered
	LogFrameworkRegistered(frameworkID *mesos.FrameworkID, masterInfo *mesos.MasterInfo)
	// Log Framework has been re-registered
	LogFrameworkReregistered(masterInfo *mesos.MasterInfo)
	// Log Framework has been disconnected from the Mesos master
	LogDisconnected()
	// Log Status update of a task
	LogTaskStatusUpdate(status *mesos.TaskStatus)
	// Log Scheduling policy switches (if any)
	LogSchedPolicySwitch(name string, nextPolicy SchedPolicyState)
	// Log the computation overhead of classifying tasks in the scheduling window.
	LogClsfnAndTaskDistOverhead(overhead time.Duration)
}
