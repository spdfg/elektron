/*
Cluster wide dynamic capping
Step1. Compute the running average of watts of tasks in window.
Step2. Compute what percentage of total power of each node, is the running average.
Step3. Compute the median of the percetages and this is the percentage that the cluster needs to be capped at.

1. First fit scheduling -- Perform the above steps for each task that needs to be scheduled.
2. Ranked based scheduling -- Sort the tasks to be scheduled, in ascending order, and then determine the cluster wide cap.

This is not a scheduler but a scheduling scheme that schedulers can use.
*/
package schedulers

import (
	"bitbucket.org/sunybingcloud/electron/constants"
	"bitbucket.org/sunybingcloud/electron/def"
	"container/list"
	"errors"
	"github.com/montanaflynn/stats"
	"log"
	"sort"
)

// Structure containing utility data structures used to compute cluster-wide dynamic cap.
type clusterwideCapper struct {
	// window of tasks.
	windowOfTasks list.List
	// The current sum of requested powers of the tasks in the window.
	currentSum float64
	// The current number of tasks in the window.
	numberOfTasksInWindow int
}

// Defining constructor for clusterwideCapper. Please don't call this directly and instead use getClusterwideCapperInstance().
func newClusterwideCapper() *clusterwideCapper {
	return &clusterwideCapper{currentSum: 0.0, numberOfTasksInWindow: 0}
}

// Singleton instance of clusterwideCapper
var singletonCapper *clusterwideCapper

// Retrieve the singleton instance of clusterwideCapper.
func getClusterwideCapperInstance() *clusterwideCapper {
	if singletonCapper == nil {
		singletonCapper = newClusterwideCapper()
	} else {
		// Do nothing
	}
	return singletonCapper
}

// Clear and initialize all the members of clusterwideCapper.
func (capper clusterwideCapper) clear() {
	capper.windowOfTasks.Init()
	capper.currentSum = 0
	capper.numberOfTasksInWindow = 0
}

// Compute the average of watts of all the tasks in the window.
func (capper clusterwideCapper) average() float64 {
	return capper.currentSum / float64(capper.windowOfTasks.Len())
}

/*
Compute the running average.

Using clusterwideCapper#windowOfTasks to store the tasks.
Task at position 0 (oldest task) is removed when the window is full and new task arrives.
*/
func (capper clusterwideCapper) runningAverageOfWatts(tsk *def.Task) float64 {
	var average float64
	if capper.numberOfTasksInWindow < constants.WindowSize {
		capper.windowOfTasks.PushBack(tsk)
		capper.numberOfTasksInWindow++
		capper.currentSum += float64(tsk.Watts) * constants.CapMargin
	} else {
		taskToRemoveElement := capper.windowOfTasks.Front()
		if taskToRemove, ok := taskToRemoveElement.Value.(*def.Task); ok {
			capper.currentSum -= float64(taskToRemove.Watts) * constants.CapMargin
			capper.windowOfTasks.Remove(taskToRemoveElement)
		}
		capper.windowOfTasks.PushBack(tsk)
		capper.currentSum += float64(tsk.Watts) * constants.CapMargin
	}
	average = capper.average()
	return average
}

/*
Calculating cap value.

1. Sorting the values of ratios ((running average/totalPower) per node) in ascending order.
2. Computing the median of above sorted values.
3. The median is now the cap.
*/
func (capper clusterwideCapper) getCap(ratios map[string]float64) float64 {
	var values []float64
	// Validation
	if ratios == nil {
		return 100.0
	}
	for _, apower := range ratios {
		values = append(values, apower)
	}
	// sorting the values in ascending order.
	sort.Float64s(values)
	// Calculating the median
	if median, err := stats.Median(values); err == nil {
		return median
	}
	// should never reach here. If here, then just setting the cap value to be 100
	return 100.0
}

/*
A recapping strategy which decides between 2 different recapping schemes.
1. the regular scheme based on the average power usage across the cluster.
2. A scheme based on the average of the loads on each node in the cluster.

The recap value picked the least among the two.

The cleverRecap scheme works well when the cluster is relatively idle and until then,
	the primitive recapping scheme works better.
*/
func (capper clusterwideCapper) cleverRecap(totalPower map[string]float64,
	taskMonitor map[string][]def.Task, finishedTaskId string) (float64, error) {
	// Validation
	if totalPower == nil || taskMonitor == nil {
		return 100.0, errors.New("Invalid argument: totalPower, taskMonitor")
	}

	// determining the recap value by calling the regular recap(...)
	toggle := false
	recapValue, err := capper.recap(totalPower, taskMonitor, finishedTaskId)
	if err == nil {
		toggle = true
	}

	// watts usage on each node in the cluster.
	wattsUsages := make(map[string][]float64)
	hostOfFinishedTask := ""
	indexOfFinishedTask := -1
	for _, host := range constants.Hosts {
		wattsUsages[host] = []float64{0.0}
	}
	for host, tasks := range taskMonitor {
		for i, task := range tasks {
			if task.TaskID == finishedTaskId {
				hostOfFinishedTask = host
				indexOfFinishedTask = i
				// Not considering this task for the computation of totalAllocatedPower and totalRunningTasks
				continue
			}
			wattsUsages[host] = append(wattsUsages[host], float64(task.Watts)*constants.CapMargin)
		}
	}

	// Updating task monitor. If recap(...) has deleted the finished task from the taskMonitor,
	// then this will be ignored. Else (this is only when an error occured with recap(...)), we remove it here.
	if hostOfFinishedTask != "" && indexOfFinishedTask != -1 {
		log.Printf("Removing task with task [%s] from the list of running tasks\n",
			taskMonitor[hostOfFinishedTask][indexOfFinishedTask].TaskID)
		taskMonitor[hostOfFinishedTask] = append(taskMonitor[hostOfFinishedTask][:indexOfFinishedTask],
			taskMonitor[hostOfFinishedTask][indexOfFinishedTask+1:]...)
	}

	// Need to check whether there are still tasks running on the cluster. If not then we return an error.
	clusterIdle := true
	for _, tasks := range taskMonitor {
		if len(tasks) > 0 {
			clusterIdle = false
		}
	}

	if !clusterIdle {
		// load on each node in the cluster.
		loads := []float64{0.0}
		for host, usages := range wattsUsages {
			totalUsage := 0.0
			for _, usage := range usages {
				totalUsage += usage
			}
			loads = append(loads, totalUsage/totalPower[host])
		}

		// Now need to compute the average load.
		totalLoad := 0.0
		for _, load := range loads {
			totalLoad += load
		}
		averageLoad := (totalLoad / float64(len(loads)) * 100.0) // this would be the cap value.
		// If toggle is true, then we need to return the least recap value.
		if toggle {
			if averageLoad <= recapValue {
				return averageLoad, nil
			} else {
				return recapValue, nil
			}
		} else {
			return averageLoad, nil
		}
	}
	return 100.0, errors.New("No task running on the cluster.")
}

/*
Recapping the entire cluster.

1. Remove the task that finished from the list of running tasks.
2. Compute the average allocated power of each of the tasks that are currently running.
3. For each host, determine the ratio of the average to the total power.
4. Determine the median of the ratios and this would be the new cluster wide cap.

This needs to be called whenever a task finishes execution.
*/
func (capper clusterwideCapper) recap(totalPower map[string]float64,
	taskMonitor map[string][]def.Task, finishedTaskId string) (float64, error) {
	// Validation
	if totalPower == nil || taskMonitor == nil {
		return 100.0, errors.New("Invalid argument: totalPower, taskMonitor")
	}
	totalAllocatedPower := 0.0
	totalRunningTasks := 0

	hostOfFinishedTask := ""
	indexOfFinishedTask := -1
	for host, tasks := range taskMonitor {
		for i, task := range tasks {
			if task.TaskID == finishedTaskId {
				hostOfFinishedTask = host
				indexOfFinishedTask = i
				// Not considering this task for the computation of totalAllocatedPower and totalRunningTasks
				continue
			}
			totalAllocatedPower += (float64(task.Watts) * constants.CapMargin)
			totalRunningTasks++
		}
	}

	// Updating task monitor
	if hostOfFinishedTask != "" && indexOfFinishedTask != -1 {
		log.Printf("Removing task with task [%s] from the list of running tasks\n",
			taskMonitor[hostOfFinishedTask][indexOfFinishedTask].TaskID)
		taskMonitor[hostOfFinishedTask] = append(taskMonitor[hostOfFinishedTask][:indexOfFinishedTask],
			taskMonitor[hostOfFinishedTask][indexOfFinishedTask+1:]...)
	}

	// For the last task, totalAllocatedPower and totalRunningTasks would be 0
	if totalAllocatedPower == 0 && totalRunningTasks == 0 {
		return 100, errors.New("No task running on the cluster.")
	}

	average := totalAllocatedPower / float64(totalRunningTasks)
	ratios := []float64{}
	for _, tpower := range totalPower {
		ratios = append(ratios, (average/tpower)*100)
	}
	sort.Float64s(ratios)
	median, err := stats.Median(ratios)
	if err == nil {
		return median, nil
	} else {
		return 100, err
	}
}

/* Quick sort algorithm to sort tasks, in place, in ascending order of power.*/
func (capper clusterwideCapper) quickSort(low int, high int, tasksToSort *[]def.Task) {
	i := low
	j := high
	// calculating the pivot
	pivotIndex := low + (high-low)/2
	pivot := (*tasksToSort)[pivotIndex]
	for i <= j {
		for (*tasksToSort)[i].Watts < pivot.Watts {
			i++
		}
		for (*tasksToSort)[j].Watts > pivot.Watts {
			j--
		}
		if i <= j {
			temp := (*tasksToSort)[i]
			(*tasksToSort)[i] = (*tasksToSort)[j]
			(*tasksToSort)[j] = temp
			i++
			j--
		}
	}
	if low < j {
		capper.quickSort(low, j, tasksToSort)
	}
	if i < high {
		capper.quickSort(i, high, tasksToSort)
	}
}

// Sorting tasks in ascending order of requested watts.
func (capper clusterwideCapper) sortTasks(tasksToSort *[]def.Task) {
	capper.quickSort(0, len(*tasksToSort)-1, tasksToSort)
}

/*
Remove entry for finished task.
This function is called when a task completes.
This completed task needs to be removed from the window of tasks (if it is still present)
  so that it doesn't contribute to the computation of the cap value.
*/
func (capper clusterwideCapper) taskFinished(taskID string) {
	// If the window is empty the just return. This condition should technically return false.
	if capper.windowOfTasks.Len() == 0 {
		return
	}

	// Checking whether the task with the given taskID is currently present in the window of tasks.
	var taskElementToRemove *list.Element
	for taskElement := capper.windowOfTasks.Front(); taskElement != nil; taskElement = taskElement.Next() {
		if tsk, ok := taskElement.Value.(*def.Task); ok {
			if tsk.TaskID == taskID {
				taskElementToRemove = taskElement
			}
		}
	}

	// we need to remove the task from the window.
	if taskToRemove, ok := taskElementToRemove.Value.(*def.Task); ok {
		capper.windowOfTasks.Remove(taskElementToRemove)
		capper.numberOfTasksInWindow -= 1
		capper.currentSum -= float64(taskToRemove.Watts) * constants.CapMargin
	}
}

// First come first serve scheduling.
func (capper clusterwideCapper) fcfsDetermineCap(totalPower map[string]float64,
	newTask *def.Task) (float64, error) {
	// Validation
	if totalPower == nil {
		return 100, errors.New("Invalid argument: totalPower")
	} else {
		// Need to calculate the running average
		runningAverage := capper.runningAverageOfWatts(newTask)
		// For each node, calculate the percentage of the running average to the total power.
		ratios := make(map[string]float64)
		for host, tpower := range totalPower {
			if tpower >= runningAverage {
				ratios[host] = (runningAverage / tpower) * 100
			} else {
				// We don't consider this host for the computation of the cluster wide cap.
			}
		}

		// Determine the cluster wide cap value.
		capValue := capper.getCap(ratios)
		// Need to cap the cluster to this value.
		return capValue, nil
	}
}

// Stringer for an instance of clusterwideCapper
func (capper clusterwideCapper) string() string {
	return "Cluster Capper -- Proactively cap the entire cluster."
}
