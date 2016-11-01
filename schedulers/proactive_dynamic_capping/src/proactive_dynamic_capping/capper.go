/*
Cluster wide dynamic capping
Step1. Compute running average of tasks in window.
Step2. Compute what percentage of available power of each node, is the running average.
Step3. Compute the median of the percentages and this is the percentage that the cluster needs to be cpaped at.

1. First Fit Scheduling -- Perform the above steps for each task that needs to be scheduled.
2. Rank based Scheduling -- Sort a set of tasks to be scheduled, in ascending order of power, and then perform the above steps for each of them in the sorted order.
*/

package proactive_dynamic_capping

import (
  "constants"
  "container/list"
  "errors"
  "github.com/montanaflynn/stats"
  "task"
  "sort"
  "sync"
)

// Structure containing utility data structures used to compute cluster wide dyanmic cap.
type Capper struct {
  // window of tasks.
  window_of_tasks list.List
  // The current sum of requested powers of the tasks in the window.
  current_sum float64
  // The current number of tasks in the window.
  number_of_tasks_in_window int
}

// Defining constructor for Capper.
func NewCapper() *Capper {
  return &Capper{current_sum: 0.0, number_of_tasks_in_window: 0}
}

// For locking on operations that may result in race conditions.
var mutex sync.Mutex

// Singleton instance of Capper
var singleton_capper *Capper
// Retrieve the singleton instance of Capper.
func GetInstance() *Capper {
  if singleton_capper == nil {
    mutex.Lock()
    singleton_capper = NewCapper()
    mutex.Unlock()
  } else {
    // Do nothing
  }
  return singleton_capper
}

// Clear and initialize all the members of Capper.
func (capper Capper) Clear() {
  capper.window_of_tasks.Init()
  capper.current_sum = 0
  capper.number_of_tasks_in_window = 0
}

// Compute the average of watts of all the tasks in the window.
func (capper Capper) average() float64 {
  return capper.current_sum / float64(capper.window_of_tasks.Len())
}

/*
 Compute the running average

 Using Capper#window_of_tasks to store the tasks in the window. Task at position 0 (oldest task) removed when window is full and new task arrives.
*/
func (capper Capper) running_average_of_watts(tsk *task.Task) float64 {
  var average float64
  if capper.number_of_tasks_in_window < constants.Window_size {
    capper.window_of_tasks.PushBack(tsk)
    capper.number_of_tasks_in_window++
    capper.current_sum += float64(tsk.Watts)
  } else {
    task_to_remove_element := capper.window_of_tasks.Front()
    if task_to_remove, ok := task_to_remove_element.Value.(*task.Task); ok {
      capper.current_sum -= float64(task_to_remove.Watts)
      capper.window_of_tasks.Remove(task_to_remove_element)
    }
    capper.window_of_tasks.PushBack(tsk)
    capper.current_sum += float64(tsk.Watts)
  }
  average = capper.average()
  return average
}

/*
 Calculating cap value

 1. Sorting the values of running_average_available_power_percentage in ascending order.
 2. Computing the median of the above sorted values.
 3. The median is now the cap value.
*/
func (capper Capper) get_cap(running_average_available_power_percentage map[string]float64) float64 {
  var values []float64
  // Validation
  if running_average_available_power_percentage == nil {
    return 100.0
  }
  for _, apower := range running_average_available_power_percentage {
    values = append(values, apower)
  }
  // sorting the values in ascending order
  sort.Float64s(values)
  // Calculating the median
  if median, err := stats.Median(values); err == nil {
    return median
  }
  // should never reach here. If here, then just setting the cap value to be 100
  return 100.0
}

// In place sorting of tasks to be scheduled based on the requested watts.
func qsort_tasks(low int, high int, tasks_to_sort []*task.Task) {
  i := low
  j := high
  // calculating the pivot
  pivot_index := low + (high - low)/2
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
    qsort_tasks(low, j, tasks_to_sort)
  }
  if i < high {
    qsort_tasks(i, high, tasks_to_sort)
  }
}

// Sorting tasks in ascending order of requested watts.
func (capper Capper) sort_tasks(tasks_to_sort []*task.Task) {
  qsort_tasks(0, len(tasks_to_sort)-1, tasks_to_sort)
}

/*
Remove entry for finished task.
Electron needs to call this whenever a task completes so that the finished task no longer contributes to the computation of the cluster wide cap.
*/
func (capper Capper) Task_finished(finished_task *task.Task) {
  // If the window is empty then just return. Should not be entering this condition as it would mean that there is a bug.
  if capper.window_of_tasks.Len() == 0 {
    return
  }

  // Checking whether the finished task is currently present in the window of tasks.
  var task_element_to_remove *list.Element
  for task_element := capper.window_of_tasks.Front(); task_element != nil; task_element = task_element.Next() {
    if tsk, ok := task_element.Value.(*task.Task); ok {
      if task.Compare(tsk, finished_task) {
        task_element_to_remove = task_element
      }
    }
  }

  // If finished task is there in the window of tasks, then we need to remove the task from the same and modify the members of Capper accordingly.
  if task_to_remove, ok := task_element_to_remove.Value.(*task.Task); ok {
    capper.window_of_tasks.Remove(task_element_to_remove)
    capper.number_of_tasks_in_window -= 1
    capper.current_sum -= float64(task_to_remove.Watts)
  }
}

// Ranked based scheduling
func (capper Capper) Ranked_determine_cap(available_power map[string]float64, tasks_to_schedule []*task.Task) ([]*task.Task, map[int]float64, error) {
  // Validation
  if available_power == nil || len(tasks_to_schedule) == 0 {
    return nil, nil, errors.New("No available power and no tasks to schedule.")
  } else {
    // Need to sort the tasks in ascending order of requested power
    capper.sort_tasks(tasks_to_schedule)

    // Now, for each task in the sorted set of tasks, we need to use the Fcfs_determine_cap logic.
    cluster_wide_cap_values := make(map[int]float64)
    index := 0
    for _, tsk := range tasks_to_schedule {
      /*
        Note that even though Fcfs_determine_cap is called, we have sorted the tasks aprior and thus, the tasks are scheduled in the sorted fashion.
        Calling Fcfs_determine_cap(...) just to avoid redundant code.
      */
      if cap, err := capper.Fcfs_determine_cap(available_power, tsk); err == nil {
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
func (capper Capper) Fcfs_determine_cap(available_power map[string]float64, new_task *task.Task) (float64, error) {
  // Validation
  if available_power == nil {
    // If no power available power, then capping the cluster at 100%. Electron might choose to queue the task.
    return 100.0, errors.New("No available power.")
  } else {
    mutex.Lock()
    // Need to calcualte the running average
    running_average := capper.running_average_of_watts(new_task)
    // What percent of available power for each node is the running average
    running_average_available_power_percentage := make(map[string]float64)
    for node, apower := range available_power {
      if apower >= running_average {
        running_average_available_power_percentage[node] = (running_average/apower) * 100
      } else {
        // We don't consider this node in the offers
      }
    }

    // Determine the cluster wide cap value.
    cap_value := capper.get_cap(running_average_available_power_percentage)
    // Electron has to now cap the cluster to this value before launching the next task.
    mutex.Unlock()
    return cap_value, nil
  }
}
