package main

import (
  "constants"
  "fmt"
  "math/rand"
  "task"
  "proactive_dynamic_capping"
  )

func sample_available_power() map[string]float64{
  return map[string]float64{
    "stratos-001":100.0,
    "stratos-002":150.0,
    "stratos-003":80.0,
    "stratos-004":90.0,
  }
}

func get_random_power(min, max int) int {
  return rand.Intn(max - min) + min
}

func cap_value_one_task_fcfs(capper *proactive_dynamic_capping.Capper) {
  fmt.Println("==== FCFS, Number of tasks: 1 ====")
  available_power := sample_available_power()
  tsk := task.NewTask("gouravr/minife:v5", "minife:v5", "stratos-001",
    "minife_command", 4.0, 10, 50, 1)
  if cap_value, err := capper.Fcfs_determine_cap(available_power, tsk); err == nil {
    fmt.Println("task = " + tsk.String())
    fmt.Printf("cap value = %f\n", cap_value)
  }
}

func cap_value_window_size_tasks_fcfs(capper *proactive_dynamic_capping.Capper) {
  fmt.Println()
  fmt.Println("==== FCFS, Number of tasks: 3 (window size) ====")
  available_power := sample_available_power()
  for i := 0; i < constants.Window_size; i++ {
    tsk := task.NewTask("gouravr/minife:v5", "minife:v5", "stratos-001",
      "minife_command", 4.0, 10, get_random_power(30, 150), 1)
    fmt.Printf("task%d = %s\n", i, tsk.String())
    if cap_value, err := capper.Fcfs_determine_cap(available_power, tsk); err == nil {
      fmt.Printf("CAP: %f\n", cap_value)
    }
  }
}

func cap_value_more_than_window_size_tasks_fcfs(capper *proactive_dynamic_capping.Capper) {
  fmt.Println()
  fmt.Println("==== FCFS, Number of tasks: >3 (> window_size) ====")
  available_power := sample_available_power()
  for i := 0; i < constants.Window_size + 2; i++ {
    tsk := task.NewTask("gouravr/minife:v5", "minife:v5", "stratos-001",
      "minife_command", 4.0, 10, get_random_power(30, 150), 1)
    fmt.Printf("task%d = %s\n", i, tsk.String())
    if cap_value, err := capper.Fcfs_determine_cap(available_power, tsk); err == nil {
      fmt.Printf("CAP: %f\n", cap_value)
    }
  }
}

func cap_values_for_ranked_tasks(capper *proactive_dynamic_capping.Capper) {
  fmt.Println()
  fmt.Println("==== Ranked, Number of tasks: 5 (window size + 2) ====")
  available_power := sample_available_power()
  var tasks_to_schedule []*task.Task
  for i := 0; i < constants.Window_size + 2; i++ {
    tasks_to_schedule = append(tasks_to_schedule,
      task.NewTask("gouravr/minife:v5", "minife:v5", "stratos-001",
        "minife_command", 4.0, 10, get_random_power(30, 150), 1))
  }
  // Printing the tasks that need to be scheduled.
  index := 0
  for _, tsk := range tasks_to_schedule {
    fmt.Printf("task%d = %s\n", index, tsk.String())
    index++
  }
  if sorted_tasks_to_be_scheduled, cwcv, err := capper.Ranked_determine_cap(available_power, tasks_to_schedule); err == nil {
    fmt.Printf("The cap values are: ")
    fmt.Println(cwcv)
    fmt.Println("The order of tasks to be scheduled :-")
    for _, tsk := range sorted_tasks_to_be_scheduled {
      fmt.Println(tsk.String())
    }
  }
}

func main() {
  capper := proactive_dynamic_capping.GetInstance()
  cap_value_one_task_fcfs(capper)
  capper.Clear()
  cap_value_window_size_tasks_fcfs(capper)
  capper.Clear()
  cap_value_more_than_window_size_tasks_fcfs(capper)
  capper.Clear()
  cap_values_for_ranked_tasks(capper)
  capper.Clear()
}
