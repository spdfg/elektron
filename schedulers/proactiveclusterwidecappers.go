/*
Cluster wide dynamic capping
Step1. Compute the running average of watts of tasks in window.
Step2. Compute what percentage of total power of each node, is the running average.
Step3. Compute the median of the percetages and this is the percentage that the cluster needs to be capped at.

1. First fit scheduling -- Perform the above steps for each task that needs to be scheduled.

This is not a scheduler but a scheduling scheme that schedulers can use.
*/
package schedulers

import (
	"bitbucket.org/sunybingcloud/electron/constants"
	"bitbucket.org/sunybingcloud/electron/def"
	"container/list"
	"errors"
	"github.com/montanaflynn/stats"
	"sort"
)

// Structure containing utility data structures used to compute cluster-wide dynamic cap.
type clusterwideCapper struct {
	// window of tasks.
	window_of_tasks list.List
	// The current sum of requested powers of the tasks in the window.
	current_sum float64
	// The current number of tasks in the window.
	number_of_tasks_in_window int
}

// Defining constructor for clusterwideCapper. Please don't call this directly and instead use getClusterwideCapperInstance().
func newClusterwideCapper() *clusterwideCapper {
	return &clusterwideCapper{current_sum: 0.0, number_of_tasks_in_window: 0}
}

// Singleton instance of clusterwideCapper
var singleton_capper *clusterwideCapper

// Retrieve the singleton instance of clusterwideCapper.
func getClusterwideCapperInstance() *clusterwideCapper {
	if singleton_capper == nil {
		singleton_capper = newClusterwideCapper()
	} else {
		// Do nothing
	}
	return singleton_capper
}

// Clear and initialize all the members of clusterwideCapper.
func (capper clusterwideCapper) clear() {
	capper.window_of_tasks.Init()
	capper.current_sum = 0
	capper.number_of_tasks_in_window = 0
}

// Compute the average of watts of all the tasks in the window.
func (capper clusterwideCapper) average() float64 {
	return capper.current_sum / float64(capper.window_of_tasks.Len())
}

/*
Compute the running average.

Using clusterwideCapper#window_of_tasks to store the tasks.
Task at position 0 (oldest task) is removed when the window is full and new task arrives.
*/
func (capper clusterwideCapper) running_average_of_watts(tsk *def.Task) float64 {
	var average float64
	if capper.number_of_tasks_in_window < constants.Window_size {
		capper.window_of_tasks.PushBack(tsk)
		capper.number_of_tasks_in_window++
		capper.current_sum += float64(tsk.Watts) * constants.Cap_margin
	} else {
		task_to_remove_element := capper.window_of_tasks.Front()
		if task_to_remove, ok := task_to_remove_element.Value.(*def.Task); ok {
			capper.current_sum -= float64(task_to_remove.Watts) * constants.Cap_margin
			capper.window_of_tasks.Remove(task_to_remove_element)
		}
		capper.window_of_tasks.PushBack(tsk)
		capper.current_sum += float64(tsk.Watts) * constants.Cap_margin
	}
	average = capper.average()
	return average
}

/*
Calculating cap value.

1. Sorting the values of running_average_to_total_power_percentage in ascending order.
2. Computing the median of above sorted values.
3. The median is now the cap.
*/
func (capper clusterwideCapper) get_cap(running_average_to_total_power_percentage map[string]float64) float64 {
	var values []float64
	// Validation
	if running_average_to_total_power_percentage == nil {
		return 100.0
	}
	for _, apower := range running_average_to_total_power_percentage {
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
Recapping the entire cluster.

1. Remove the task that finished from the list of running tasks.
2. Compute the average allocated power of each of the tasks that are currently running.
3. For each host, determine the ratio of the average to the total power.
4. Determine the median of the ratios and this would be the new cluster wide cap.

This needs to be called whenever a task finishes execution.
*/
func (capper clusterwideCapper) recap(total_power map[string]float64,
	task_monitor map[string][]def.Task, finished_taskId string) (float64, error) {
	// Validation
	if total_power == nil || task_monitor == nil {
		return 100.0, errors.New("Invalid argument: total_power, task_monitor")
	}
	total_allocated_power := 0.0
	total_running_tasks := 0
	for _, tasks := range task_monitor {
		index := 0
		for i, task := range tasks {
			if task.TaskID == finished_taskId {
				index = i
				continue
			}
			total_allocated_power += float64(task.Watts) * constants.Cap_margin
			total_running_tasks++
		}
		tasks = append(tasks[:index], tasks[index+1:]...)
	}
	average := total_allocated_power / float64(total_running_tasks)
	ratios := []float64{}
	for _, tpower := range total_power {
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
func (capper clusterwideCapper) quick_sort(low int, high int, tasks_to_sort []*def.Task) {
	i := low
	j := high
	// calculating the pivot
	pivot_index := low + (high-low)/2
	pivot := tasks_to_sort[pivot_index]
	for i <= j {
		for tasks_to_sort[i].Watts < pivot.Watts {
			i++
		}
		for tasks_to_sort[j].Watts > pivot.Watts {
			j--
		}
		if i <= j {
			temp := tasks_to_sort[i]
			tasks_to_sort[i] = tasks_to_sort[j]
			tasks_to_sort[j] = temp
			i++
			j--
		}
	}
	if low < j {
		capper.quick_sort(low, j, tasks_to_sort)
	}
	if i < high {
		capper.quick_sort(i, high, tasks_to_sort)
	}
}

// Sorting tasks in ascending order of requested watts.
func (capper clusterwideCapper) sort_tasks(tasks_to_sort []*def.Task) {
	capper.quick_sort(0, len(tasks_to_sort)-1, tasks_to_sort)
}

/*
Remove entry for finished task.
This function is called when a task completes.
This completed task needs to be removed from the window of tasks (if it is still present)
  so that it doesn't contribute to the computation of the cap value.
*/
func (capper clusterwideCapper) taskFinished(taskID string) {
	// If the window is empty the just return. This condition should technically return false.
	if capper.window_of_tasks.Len() == 0 {
		return
	}

	// Checking whether the task with the given taskID is currently present in the window of tasks.
	var task_element_to_remove *list.Element
	for task_element := capper.window_of_tasks.Front(); task_element != nil; task_element = task_element.Next() {
		if tsk, ok := task_element.Value.(*def.Task); ok {
			if tsk.TaskID == taskID {
				task_element_to_remove = task_element
			}
		}
	}

	// Ee need to remove the task from the window.
	if task_to_remove, ok := task_element_to_remove.Value.(*def.Task); ok {
		capper.window_of_tasks.Remove(task_element_to_remove)
		capper.number_of_tasks_in_window -= 1
		capper.current_sum -= float64(task_to_remove.Watts) * constants.Cap_margin
	}
}

// Ranked based scheduling.
func (capper clusterwideCapper) rankedDetermineCap(available_power map[string]float64,
	tasks_to_schedule []*def.Task) ([]*def.Task, map[int]float64, error) {
	// Validation
	if available_power == nil || len(tasks_to_schedule) == 0 {
		return nil, nil, errors.New("Invalid argument: available_power, tasks_to_schedule")
	} else {
		// Need to sort the tasks in ascending order of requested power.
		capper.sort_tasks(tasks_to_schedule)

		// Now, for each task in the sorted set of tasks, we need to use the Fcfs_determine_cap logic.
		cluster_wide_cap_values := make(map[int]float64)
		index := 0
		for _, tsk := range tasks_to_schedule {
			/*
			   Note that even though Fcfs_determine_cap is called, we have sorted the tasks aprior and thus, the tasks are scheduled in the sorted fashion.
			   Calling Fcfs_determine_cap(...) just to avoid redundant code.
			*/
			if cap, err := capper.fcfsDetermineCap(available_power, tsk); err == nil {
				cluster_wide_cap_values[index] = cap
			} else {
				return nil, nil, err
			}
			index++
		}
		// Now returning the sorted set of tasks and the cluster wide cap values for each task that is launched.
		return tasks_to_schedule, cluster_wide_cap_values, nil
	}
}

// First come first serve scheduling.
func (capper clusterwideCapper) fcfsDetermineCap(total_power map[string]float64,
	new_task *def.Task) (float64, error) {
	// Validation
	if total_power == nil {
		return 100, errors.New("Invalid argument: total_power")
	} else {
		// Need to calculate the running average
		running_average := capper.running_average_of_watts(new_task)
		// For each node, calculate the percentage of the running average to the total power.
		running_average_to_total_power_percentage := make(map[string]float64)
		for host, tpower := range total_power {
			if tpower >= running_average {
				running_average_to_total_power_percentage[host] = (running_average / tpower) * 100
			} else {
				// We don't consider this host for the computation of the cluster wide cap.
			}
		}

		// Determine the cluster wide cap value.
		cap_value := capper.get_cap(running_average_to_total_power_percentage)
		// Need to cap the cluster to this value.
		return cap_value, nil
	}
}

// Stringer for an instance of clusterwideCapper
func (capper clusterwideCapper) string() string {
	return "Cluster Capper -- Proactively cap the entire cluster."
}
