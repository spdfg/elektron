/*
Cluster wide dynamic capping

This is a capping strategy that can be used with schedulers to improve the power consumption.
*/
package powerCapping

import (
	"bitbucket.org/sunybingcloud/electron/constants"
	"bitbucket.org/sunybingcloud/electron/def"
	"bitbucket.org/sunybingcloud/electron/utilities/runAvg"
	"errors"
	"github.com/montanaflynn/stats"
	"log"
	"sort"
)

// wrapper around def.Task that implements runAvg.Interface
type taskWrapper struct {
	task def.Task
}

func (tw taskWrapper) Val() float64 {
	return tw.task.Watts * constants.CapMargin
}

func (tw taskWrapper) ID() string {
	return tw.task.TaskID
}

// Cluster wide capper
type ClusterwideCapper struct{}

// Defining constructor for clusterwideCapper. Please don't call this directly and instead use GetClusterwideCapperInstance()
func newClusterwideCapper() *ClusterwideCapper {
	return &ClusterwideCapper{}
}

// Singleton instance of clusterwideCapper
var singletonCapper *ClusterwideCapper

// Retrieve the singleton instance of clusterwideCapper.
func GetClusterwideCapperInstance() *ClusterwideCapper {
	if singletonCapper == nil {
		singletonCapper = newClusterwideCapper()
	} else {
		// Do nothing
	}
	return singletonCapper
}

// Clear and initialize the runAvg calculator
func (capper ClusterwideCapper) clear() {
	runAvg.Init()
}

/*
Calculating cap value.

1. Sorting the values of ratios ((running average/totalPower) per node) in ascending order.
2. Computing the median of above sorted values.
3. The median is now the cap.
*/
func (capper ClusterwideCapper) getCap(ratios map[string]float64) float64 {
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
A Recapping strategy which decides between 2 different Recapping schemes.
1. the regular scheme based on the average power usage across the cluster.
2. A scheme based on the average of the loads on each node in the cluster.

The Recap value picked the least among the two.

The CleverRecap scheme works well when the cluster is relatively idle and until then,
	the primitive Recapping scheme works better.
*/
func (capper ClusterwideCapper) CleverRecap(totalPower map[string]float64,
	taskMonitor map[string][]def.Task, finishedTaskId string) (float64, error) {
	// Validation
	if totalPower == nil || taskMonitor == nil {
		return 100.0, errors.New("Invalid argument: totalPower, taskMonitor")
	}

	// determining the Recap value by calling the regular Recap(...)
	toggle := false
	RecapValue, err := capper.NaiveRecap(totalPower, taskMonitor, finishedTaskId)
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

	// Updating task monitor. If Recap(...) has deleted the finished task from the taskMonitor,
	// then this will be ignored. Else (this is only when an error occured with Recap(...)), we remove it here.
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
		// If toggle is true, then we need to return the least Recap value.
		if toggle {
			if averageLoad <= RecapValue {
				return averageLoad, nil
			} else {
				return RecapValue, nil
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
func (capper ClusterwideCapper) NaiveRecap(totalPower map[string]float64,
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

/*
Remove entry for finished task from the window

This  function is called when a task completes.
This completed task needs to be removed from the window of tasks (if it is still present)
	so that it doesn't contribute to the computation of the next cap value.
*/
func (capper ClusterwideCapper) TaskFinished(taskID string) {
	runAvg.Remove(taskID)
}

// First come first serve scheduling.
func (capper ClusterwideCapper) FCFSDeterminedCap(totalPower map[string]float64,
	newTask *def.Task) (float64, error) {
	// Validation
	if totalPower == nil {
		return 100, errors.New("Invalid argument: totalPower")
	} else {
		// Need to calculate the running average
		runningAverage := runAvg.Calc(taskWrapper{task: *newTask}, constants.ConsiderationWindowSize)
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
func (capper ClusterwideCapper) String() string {
	return "Cluster-wide Capper -- Proactively cap the entire cluster."
}
