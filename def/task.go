package def

import (
	"bitbucket.org/sunybingcloud/electron/constants"
	"encoding/json"
	"github.com/pkg/errors"
	"os"
	"reflect"
)

type Task struct {
	Name      string  `json:"name"`
	CPU       float64 `json:"cpu"`
	RAM       float64 `json:"ram"`
	Watts     float64 `json:"watts"`
	Image     string  `json:"image"`
	CMD       string  `json:"cmd"`
	Instances *int    `json:"inst"`
	Host      string  `json:"host"`
	TaskID		string	`json:"taskID"`
}

func TasksFromJSON(uri string) ([]Task, error) {

	var tasks []Task

	file, err := os.Open(uri)
	if err != nil {
		return nil, errors.Wrap(err, "Error opening file")
	}

	err = json.NewDecoder(file).Decode(&tasks)
	if err != nil {
		return nil, errors.Wrap(err, "Error unmarshalling")
	}

	return tasks, nil
}

// Update the host on which the task needs to be scheduled.
func (tsk *Task) UpdateHost(new_host string) bool {
	// Validation
	is_correct_host := false
	for _, existing_host := range constants.Hosts {
    if host == existing_host {
      is_correct_host = true
    }
  }
	if !is_correct_host {
		return false
	} else {
		tsk.Host = new_host
		return true
	}
}

// Set the taskID of the task.
func (tsk *Task) SetTaskID(taskID string) bool {
	// Validation
	if taskID == "" {
		return false
	} else {
		tsk.TaskID = taskID
		return true
	}
}

type WattsSorter []Task

func (slice WattsSorter) Len() int {
	return len(slice)
}

func (slice WattsSorter) Less(i, j int) bool {
	return slice[i].Watts < slice[j].Watts
}

func (slice WattsSorter) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// Compare two tasks.
func Compare(task1 *Task, task2 *Task) bool {
	// If comparing the same pointers (checking the addresses).
	if task1 == task2 {
		return true
	}
	// Checking member equality
	if reflect.DeepEqual(*task1, *task2) {
		// Need to check for the task ID
		if task1.TaskID == task2.TaskID {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}
