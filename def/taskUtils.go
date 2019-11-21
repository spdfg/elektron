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
	"errors"
	"fmt"
	"sort"

	"github.com/mash/gokmeans"
	"github.com/montanaflynn/stats"
	log "github.com/sirupsen/logrus"
	elekLog "github.com/spdfg/elektron/elektronLogging"
	elekLogTypes "github.com/spdfg/elektron/elektronLogging/types"
)

// Information about a cluster of tasks.
type TaskCluster struct {
	ClusterIndex int
	Tasks        []Task
	SizeScore    int // How many other clusters is this cluster bigger than
}

// Classification of Tasks using KMeans clustering using the watts consumption observations.
type TasksToClassify []Task

// Basic taskObservation calculator. This returns an array consisting of the MMPU requirements of a task.
func (tc TasksToClassify) taskObservationCalculator(task Task) []float64 {
	if task.ClassToWatts != nil {
		// Taking the aggregate.
		observations := []float64{}
		for _, watts := range task.ClassToWatts {
			observations = append(observations, watts)
		}
		return observations
	} else if task.Watts != 0.0 {
		return []float64{task.Watts}
	} else {
		elekLog.ElektronLog.Log(elekLogTypes.CONSOLE, log.FatalLevel,
			log.Fields{}, "Unable to classify tasks. Missing Watts or ClassToWatts attribute in workload")
		return []float64{0.0} // Won't reach here.
	}
}

func ClassifyTasks(tasks []Task, numberOfClusters int) []TaskCluster {
	tc := TasksToClassify(tasks)
	return tc.classify(numberOfClusters, tc.taskObservationCalculator)
}

func (tc TasksToClassify) classify(numberOfClusters int, taskObservation func(task Task) []float64) []TaskCluster {
	clusters := make(map[int][]Task)
	observations := getObservations(tc, taskObservation)
	// TODO: Make the max number of rounds configurable based on the size of the workload.
	// The max number of rounds (currently defaulted to 100) is the number of iterations performed to obtain
	// 	distinct clusters. When the data size becomes very large, we would need more iterations for clustering.
	if trained, centroids := gokmeans.Train(observations, numberOfClusters, 100); trained {
		for i := 0; i < len(observations); i++ {
			observation := observations[i]
			classIndex := gokmeans.Nearest(observation, centroids)
			if _, ok := clusters[classIndex]; ok {
				clusters[classIndex] = append(clusters[classIndex], tc[i])
			} else {
				clusters[classIndex] = []Task{tc[i]}
			}
		}
	}
	return labelAndOrder(clusters, numberOfClusters, taskObservation)
}

// Record observations.
func getObservations(tasks []Task, taskObservation func(task Task) []float64) []gokmeans.Node {
	observations := []gokmeans.Node{}
	for i := 0; i < len(tasks); i++ {
		observations = append(observations, taskObservation(tasks[i]))
	}
	return observations
}

// Sizing each task cluster using the average MMMPU requirement of the task in the cluster.
func clusterSizeAvgMMMPU(tasks []Task, taskObservation func(task Task) []float64) float64 {
	mmmpuValues := []float64{}
	// Total sum of the Median of Median Max Power Usage values for all tasks.
	total := 0.0
	for _, task := range tasks {
		observations := taskObservation(task)
		if len(observations) > 0 {
			// taskObservation would give us the mmpu values. We would need to take the median of these
			// values to obtain the Median of Median Max Power Usage value.
			if medianValue, err := stats.Median(observations); err == nil {
				mmmpuValues = append(mmmpuValues, medianValue)
				total += medianValue
			} else {
				// skip this value
				// there is an error in the task config.
				elekLog.ElektronLog.Log(elekLogTypes.CONSOLE, log.ErrorLevel,
					log.Fields{}, err.Error())
			}
		} else {
			// There is only one observation for the task.
			mmmpuValues = append(mmmpuValues, observations[0])
		}
	}

	return total / float64(len(mmmpuValues))
}

// Order clusters in increasing order of task heaviness.
func labelAndOrder(clusters map[int][]Task, numberOfClusters int, taskObservation func(task Task) []float64) []TaskCluster {
	// Determine the position of the cluster in the ordered list of clusters.
	sizedClusters := []TaskCluster{}

	// Initializing.
	for i := 0; i < numberOfClusters; i++ {
		sizedClusters = append(sizedClusters, TaskCluster{
			ClusterIndex: i,
			Tasks:        clusters[i],
			SizeScore:    0,
		})
	}

	for i := 0; i < numberOfClusters-1; i++ {
		// Sizing the current cluster based on average Median of Median Max Power Usage of tasks.
		sizeI := clusterSizeAvgMMMPU(clusters[i], taskObservation)

		// Comparing with the other clusters.
		for j := i + 1; j < numberOfClusters; j++ {
			sizeJ := clusterSizeAvgMMMPU(clusters[j], taskObservation)
			if sizeI > sizeJ {
				sizedClusters[i].SizeScore++
			} else {
				sizedClusters[j].SizeScore++
			}
		}
	}

	// Sorting the clusters based on sizeScore.
	sort.SliceStable(sizedClusters, func(i, j int) bool {
		return sizedClusters[i].SizeScore <= sizedClusters[j].SizeScore
	})
	return sizedClusters
}

// Generic Task Sorter.
// Be able to sort an array of tasks based on any of the tasks' resources.
func SortTasks(ts []Task, sb SortBy) {
	sort.SliceStable(ts, func(i, j int) bool {
		return sb(&ts[i]) <= sb(&ts[j])
	})
}

// Map taskIDs to resource requirements.
type TaskResources struct {
	CPU   float64
	Ram   float64
	Watts float64
}

var taskResourceRequirement map[string]*TaskResources

// Record resource requirements for all the tasks.
func initTaskResourceRequirements(tasks []Task) {
	taskResourceRequirement = make(map[string]*TaskResources)
	baseTaskID := "electron-"
	for _, task := range tasks {
		for i := *task.Instances; i > 0; i-- {
			taskID := fmt.Sprintf("%s-%d", baseTaskID+task.Name, i)
			taskResourceRequirement[taskID] = &TaskResources{
				CPU:   task.CPU,
				Ram:   task.RAM,
				Watts: task.Watts,
			}
		}
	}
}

// Retrieve the resource requirement of a task specified by the TaskID
func GetResourceRequirement(taskID string) (TaskResources, error) {
	if tr, ok := taskResourceRequirement[taskID]; ok {
		return *tr, nil
	} else {
		// Shouldn't be here.
		return TaskResources{}, errors.New("Invalid TaskID: " + taskID)
	}
}

// Determine the distribution of light power consuming and heavy power consuming tasks in a given window.
func GetTaskDistributionInWindow(windowSize int, tasks []Task) (float64, error) {
	getTotalInstances := func(ts []Task, taskExceedingWindow struct {
		taskName       string
		instsToDiscard int
	}) int {
		total := 0
		for _, t := range ts {
			if t.Name == taskExceedingWindow.taskName {
				total += (*t.Instances - taskExceedingWindow.instsToDiscard)
				continue
			}
			total += *t.Instances
		}
		return total
	}

	getTasksInWindow := func() (tasksInWindow []Task, taskExceedingWindow struct {
		taskName       string
		instsToDiscard int
	}) {
		tasksTraversed := 0
		// Name of task, only few instances of which fall within the window.
		lastTaskName := ""
		for _, task := range tasks {
			tasksInWindow = append(tasksInWindow, task)
			tasksTraversed += *task.Instances
			lastTaskName = task.Name
			if tasksTraversed >= windowSize {
				taskExceedingWindow.taskName = lastTaskName
				taskExceedingWindow.instsToDiscard = tasksTraversed - windowSize
				break
			}
		}

		return
	}

	// Retrieving the tasks that are in the window.
	tasksInWIndow, taskExceedingWindow := getTasksInWindow()
	// Classifying the tasks based on Median of Median Max Power Usage values.
	taskClusters := ClassifyTasks(tasksInWIndow, 2)
	// First we'll need to check if the tasks in the window could be classified into 2 clusters.
	// If yes, then we proceed with determining the distribution.
	// Else, we throw an error stating that the distribution is even as only one cluster could be formed.
	if len(taskClusters[1].Tasks) == 0 {
		return -1.0, errors.New("Only one cluster could be formed.")
	}

	// The first cluster would corresponding to the light power consuming tasks.
	// The second cluster would corresponding to the high power consuming tasks.
	lpcTasksTotalInst := getTotalInstances(taskClusters[0].Tasks, taskExceedingWindow)
	fmt.Printf("lpc:%d\n", lpcTasksTotalInst)
	hpcTasksTotalInst := getTotalInstances(taskClusters[1].Tasks, taskExceedingWindow)
	fmt.Printf("hpc:%d\n", hpcTasksTotalInst)
	return float64(lpcTasksTotalInst) / float64(hpcTasksTotalInst), nil
}
