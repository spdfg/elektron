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
A recapping strategy which decides between 2 different recapping schemes.
1. the regular scheme based on the average power usage across the cluster.
2. A scheme based on the average of the loads on each node in the cluster.

The recap value picked the least among the two.
*/
func (capper clusterwideCapper) cleverRecap(total_power map[string]float64, 
	task_monitor map[string][]def.Task, finished_taskId string) (float64, error) {
	// Validation
	if total_power == nil || task_monitor == nil {
		return 100.0, errors.New("Invalid argument: total_power, task_monitor")
	}

	// determining the recap value by calling the regular recap(...)
	toggle := false
	recapValue, err := capper.recap(total_power, task_monitor, finished_taskId)
	if err == nil {
		toggle = true
	}

	// watts usage on each node in the cluster.
	watts_usages := make(map[string][]float64)
	host_of_finished_task := ""
  	index_of_finished_task := -1
	for _, host := range constants.Hosts {
		watts_usages[host] = []float64{0.0}
	}
	for host, tasks := range task_monitor {
		for i, task := range tasks {
			if task.TaskID == finished_taskId {
        		host_of_finished_task = host
        		index_of_finished_task = i
        		// Not considering this task for the computation of total_allocated_power and total_running_tasks
        		continue
      		}
			watts_usages[host] = append(watts_usages[host], float64(task.Watts) * constants.Cap_margin)
		}
	}

	// Updating task monitor. If recap(...) has deleted the finished task from the taskMonitor,
	// then this will be ignored.
  	if host_of_finished_task != "" && index_of_finished_task != -1 {
    	log.Printf("Removing task with task [%s] from the list of running tasks\n",
     		task_monitor[host_of_finished_task][index_of_finished_task].TaskID)
    	task_monitor[host_of_finished_task] = append(task_monitor[host_of_finished_task][:index_of_finished_task],
     		task_monitor[host_of_finished_task][index_of_finished_task+1:]...)
  	}

  	// Need to check whether there are still tasks running on the cluster. If not then we return an error.
  	clusterIdle := true
  	for _, tasks := range task_monitor {
  		if len(tasks) > 0 {
  			clusterIdle = false
  		}
  	}

  	if !clusterIdle {
  		// load on each node in the cluster.
		loads := []float64{0.0}
		for host, usages := range watts_usages {
			total_usage := 0.0
			for _, usage := range usages {
				total_usage += usage
			}
			loads = append(loads, total_usage / total_power[host])
		}

		// Now need to compute the average load.
		total_load := 0.0
		for _, load := range loads {
			total_load += load
		}
		average_load := (total_load / float64(len(loads)) * 100.0) // this would be the cap value.
		// If toggle is true, then we need to return the least recap value.
		if toggle {
			if average_load <= recapValue {
				return average_load, nil
			} else {
				return recapValue, nil
			}
		} else {
			return average_load, nil
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
func (capper clusterwideCapper) recap(total_power map[string]float64,
	task_monitor map[string][]def.Task, finished_taskId string) (float64, error) {
	// Validation
	if total_power == nil || task_monitor == nil {
		return 100.0, errors.New("Invalid argument: total_power, task_monitor")
	}
	total_allocated_power := 0.0
	total_running_tasks := 0

  	host_of_finished_task := ""
  	index_of_finished_task := -1
  	for host, tasks := range task_monitor {
    	for i, task := range tasks {
      	if task.TaskID == finished_taskId {
        	host_of_finished_task = host
        	index_of_finished_task = i
        	// Not considering this task for the computation of total_allocated_power and total_running_tasks
        	continue
      	}
      	total_allocated_power += (float64(task.Watts) * constants.Cap_margin)
      	total_running_tasks++
    	}
  	}

  	// Updating task monitor
  	if host_of_finished_task != "" && index_of_finished_task != -1 {
    	log.Printf("Removing task with task [%s] from the list of running tasks\n",
     		task_monitor[host_of_finished_task][index_of_finished_task].TaskID)
    	task_monitor[host_of_finished_task] = append(task_monitor[host_of_finished_task][:index_of_finished_task],
     		task_monitor[host_of_finished_task][index_of_finished_task+1:]...)
  	}

  	// For the last task, total_allocated_power and total_running_tasks would be 0
  	if total_allocated_power == 0 && total_running_tasks == 0 {
    	return 100, errors.New("No task running on the cluster.")
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
func (capper clusterwideCapper) quick_sort(low int, high int, tasks_to_sort *[]def.Task) {
	i := low
	j := high
	// calculating the pivot
	pivot_index := low + (high-low)/2
	pivot := (*tasks_to_sort)[pivot_index]
	for i <= j {
		for (*tasks_to_sort)[i].Watts < pivot.Watts {
			i++
		}
		for (*tasks_to_sort)[j].Watts > pivot.Watts {
			j--
		}
		if i <= j {
			temp := (*tasks_to_sort)[i]
			(*tasks_to_sort)[i] = (*tasks_to_sort)[j]
			(*tasks_to_sort)[j] = temp
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
func (capper clusterwideCapper) sort_tasks(tasks_to_sort *[]def.Task) {
	capper.quick_sort(0, len(*tasks_to_sort)-1, tasks_to_sort)
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

	// we need to remove the task from the window.
	if task_to_remove, ok := task_element_to_remove.Value.(*def.Task); ok {
		capper.window_of_tasks.Remove(task_element_to_remove)
		capper.number_of_tasks_in_window -= 1
		capper.current_sum -= float64(task_to_remove.Watts) * constants.Cap_margin
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
