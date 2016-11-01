package task

import (
  "constants"
  "encoding/json"
  "reflect"
  "strconv"
  "utilities"
)

/*
  Blueprint for the task.
	Members:
		image: <image tag>
		name: <benchmark name>
		host: <host on which the task needs to be run>
		cmd: <command to run the task>
		cpu: <CPU requirement>
		ram: <RAM requirement>
		watts: <Power requirement>
		inst: <Number of instances>
*/
type Task struct {
  Image string
  Name string
  Host string
  CMD string
  CPU float64
  RAM int
  Watts int
  Inst int
}

// Defining a constructor for Task
func NewTask(image string, name string, host string,
  cmd string, cpu float64, ram int, watts int, inst int) *Task {
  return &Task{Image: image, Name: name, Host: host, CPU: cpu,
                RAM: ram, Watts: watts, Inst: inst}
}

// Update the host on which the task needs to be scheduled.
func (task Task) Update_host(new_host string) {
  // Validation
  if _, ok := constants.Total_power[new_host]; ok {
    task.Host = new_host
  }
}

// Stringify task instance
func (task Task) String() string {
  task_map := make(map[string]string)
  task_map["image"] = task.Image
  task_map["name"] = task.Name
  task_map["host"] = task.Host
  task_map["cmd"] = task.CMD
  task_map["cpu"] = utils.FloatToString(task.CPU)
  task_map["ram"] =  strconv.Itoa(task.RAM)
  task_map["watts"] =  strconv.Itoa(task.Watts)
  task_map["inst"] = strconv.Itoa(task.Inst)

  json_string, _ := json.Marshal(task_map)
  return string(json_string)
}

// Compare one task to another. 2 tasks are the same if all the corresponding members are the same.
func Compare(task *Task, other_task *Task) bool {
  // If comparing the same pointers (checking the addresses).
  if task == other_task {
    return true
  }
  // Checking member equality
  return reflect.DeepEqual(*task, *other_task)
}
