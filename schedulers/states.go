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
)

// NameFor returns the string name for a TaskState.
func NameFor(state *mesos.TaskState) string {
	switch *state {
	case mesos.TaskState_TASK_STAGING:
		return "TASK_STAGING"
	case mesos.TaskState_TASK_STARTING:
		return "TASK_STARTING"
	case mesos.TaskState_TASK_RUNNING:
		return "TASK_RUNNING"
	case mesos.TaskState_TASK_FINISHED:
		return "TASK_FINISHED" // TERMINAL
	case mesos.TaskState_TASK_FAILED:
		return "TASK_FAILED" // TERMINAL
	case mesos.TaskState_TASK_KILLED:
		return "TASK_KILLED" // TERMINAL
	case mesos.TaskState_TASK_LOST:
		return "TASK_LOST" // TERMINAL
	case mesos.TaskState_TASK_ERROR:
		return "TASK_ERROR"
	default:
		return fmt.Sprintf("UNKNOWN: %d", *state)
	}
}

// IsTerminal determines if a TaskState is a terminal state, i.e. if it singals
// that the task has stopped running.
func IsTerminal(state *mesos.TaskState) bool {
	switch *state {
	case mesos.TaskState_TASK_FINISHED,
		mesos.TaskState_TASK_FAILED,
		mesos.TaskState_TASK_KILLED,
		mesos.TaskState_TASK_LOST,
		mesos.TaskState_TASK_ERROR:
		return true
	default:
		return false
	}
}
