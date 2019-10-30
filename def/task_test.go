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

package def

import (
	"fmt"
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	"github.com/spdfg/elektron/constants"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestTasksFromJSON(t *testing.T) {
	tasks, err := TasksFromJSON("../workload_sample.json")
	assert.Equal(t, len(tasks), 2)
	assert.NoError(t, err)

	instances := 10
	minifeTask := Task{
		Name:      "minife",
		CPU:       3.0,
		RAM:       4096,
		Watts:     63.141,
		Image:     "rdelvalle/minife:electron1",
		CMD:       "cd src && mpirun -np 3 miniFE.x -nx 100 -ny 100 -nz 100",
		Instances: &instances,
		ClassToWatts: map[string]float64{
			"A": 93.062,
			"B": 65.552,
			"C": 57.897,
			"D": 60.729,
		},
	}

	dgemmTask := Task{
		Name:      "dgemm",
		CPU:       3.0,
		RAM:       32,
		Watts:     85.903,
		Image:     "rdelvalle/dgemm:electron1",
		CMD:       "/./mt-dgemm 1024",
		Instances: &instances,
		ClassToWatts: map[string]float64{
			"A": 114.789,
			"B": 89.133,
			"C": 82.672,
			"D": 81.944,
		},
	}

	assert.True(t, reflect.DeepEqual(minifeTask, tasks[0]))
	assert.True(t, reflect.DeepEqual(dgemmTask, tasks[1]))
}

func TestTask_UpdateHost(t *testing.T) {
	task := Task{}
	// UpdateHost should fail if the host is unknown.
	assert.False(t, task.UpdateHost("unknown-host"))

	constants.Hosts["known-host"] = struct{}{}
	// UpdateHost should not succeed as hostname (known-host) is known.
	task.UpdateHost("known-host")
	assert.Equal(t, task.Host, "known-host")
}

func TestTask_SetTaskID(t *testing.T) {
	instances := 1
	task := Task{
		Name:      "test-task",
		Instances: &instances,
	}

	taskID := fmt.Sprintf("electron-%s-%d", task.Name, *task.Instances)
	task.SetTaskID(taskID)
	assert.Equal(t, taskID, task.TaskID, "failed to set task ID")
}

func TestWattsToConsider(t *testing.T) {
	task := Task{
		Name:  "minife",
		Watts: 50.0,
		ClassToWatts: map[string]float64{
			"A": 30.2475289996,
			"B": 35.6491229228,
			"C": 24.0476734352,
		},
	}

	powerClass := "A"
	classAttribute := "class"
	offerClassA := &mesos.Offer{
		Attributes: []*mesos.Attribute{
			{
				Name: &classAttribute,
				Text: &mesos.Value_Text{Value: &powerClass},
			},
		},
	}
	// Without class to watts mapping.
	// Should return the Watts value.
	wattsClassMapWattsDisabled, err := WattsToConsider(task, false, offerClassA)
	assert.NoError(t, err)
	assert.Equal(t, task.Watts, wattsClassMapWattsDisabled)

	// with class to watts mapping.
	// Should return task.ClassToWatts[pc] where pc is the power-class
	// of the host corresponding to the offer.
	wattsClassMapWattsEnabled, err := WattsToConsider(task, true, offerClassA)
	assert.NoError(t, err)
	assert.Equal(t, task.ClassToWatts["A"], wattsClassMapWattsEnabled)
}

func TestCompare(t *testing.T) {
	task1 := Task{
		Name:   "test-task1",
		TaskID: "electron-test-task1-1",
	}

	task2 := Task{
		TaskID: "electron-test-task1-1",
	}

	// If two tasks have the same Task ID they should be the same task.
	assert.True(t, Compare(&task1, &task2))
}
