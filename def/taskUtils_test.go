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

package def

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var instances = 1
var tasks = []Task{
	{
		Name:      "task1",
		CPU:       4.0,
		RAM:       1024,
		Watts:     50.0,
		TaskID:    "electron-task1-1",
		Instances: &instances,
	},
	{
		Name:      "task2",
		CPU:       3.0,
		RAM:       128,
		Watts:     55.0,
		TaskID:    "electron-task2-1",
		Instances: &instances,
	},
	{
		Name:      "task3",
		CPU:       2.0,
		RAM:       64,
		Watts:     75.0,
		TaskID:    "electron-task3-1",
		Instances: &instances,
	},
	{
		Name:      "task4",
		CPU:       1.0,
		RAM:       3072,
		Watts:     81.0,
		TaskID:    "electron-task4-1",
		Instances: &instances,
	},
}

// Test task classification based on Watts requirement.
func TestClassifyTasks(t *testing.T) {
	// Classifying tasks into two clusters.
	taskClusters := ClassifyTasks(tasks, 2)
	assert.True(t, len(taskClusters) == 2, "failed to classify tasks into 2 clusters")

	// Test whether the clusters are ordered.
	firstCluster := taskClusters[0]
	secondCluster := taskClusters[1]
	assert.Less(t, firstCluster.SizeScore, secondCluster.SizeScore, "failed to order task clusters")

	// Tasks in the first cluster should be less power-intensive when compared to the tasks in the second cluster.
	for _, taskFirstCluster := range firstCluster.Tasks {
		for _, taskSecondCluster := range secondCluster.Tasks {
			assert.LessOrEqual(t, taskFirstCluster.Watts,
				taskSecondCluster.Watts, "failed to correctly classify tasks")
		}
	}
}

func TestSortTasks(t *testing.T) {
	testSortedOrder := func(sortingCriteria string, sortedValues []float64, value func(i int) float64) {
		assert.Equal(t, len(sortedValues), len(tasks))
		for i := 0; i < len(tasks); i++ {
			assert.Equal(t, sortedValues[i], value(i), "failed to sort tasks by "+sortingCriteria)
		}
	}

	// Sorting by CPU.
	SortTasks(tasks, SortByCPU)
	testSortedOrder(
		"CPU",
		[]float64{1.0, 2.0, 3.0, 4.0},
		func(i int) float64 {
			return tasks[i].CPU
		})

	// Sorting by RAM.
	SortTasks(tasks, SortByRAM)
	testSortedOrder(
		"RAM",
		[]float64{64, 128, 1024, 3072},
		func(i int) float64 {
			return tasks[i].RAM
		})

	// Sorting by Watts.
	SortTasks(tasks, SortByWatts)
	testSortedOrder(
		"Watts",
		[]float64{50.0, 55.0, 75.0, 81.0},
		func(i int) float64 {
			return tasks[i].Watts
		})
}

func TestGetResourceRequirement(t *testing.T) {
	initTaskResourceRequirements(tasks)

	for _, task := range tasks {
		resources, err := GetResourceRequirement(task.TaskID)
		assert.NoError(t, err)
		assert.Equal(t, resources.CPU, task.CPU)
		assert.Equal(t, resources.Ram, task.RAM)
		assert.Equal(t, resources.Watts, task.Watts)
	}
}

func TestGetTaskDistributionInWindow(t *testing.T) {
	// Using a window of size 4.
	taskDistribution, err := GetTaskDistributionInWindow(4, tasks)
	assert.NoError(t, err)
	// The tasks above are evenly distributed hence, task distribution should be 1.0.
	assert.Equal(t, taskDistribution, 1.0, "task distribution determined is incorrect")
}
