package def

import (
	"bitbucket.org/sunybingcloud/electron/constants"
	"bitbucket.org/sunybingcloud/electron/utilities/offerUtils"
	"encoding/json"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/pkg/errors"
	"os"
)

type Task struct {
	Name         string             `json:"name"`
	CPU          float64            `json:"cpu"`
	RAM          float64            `json:"ram"`
	Watts        float64            `json:"watts"`
	Image        string             `json:"image"`
	CMD          string             `json:"cmd"`
	Instances    *int               `json:"inst"`
	Host         string             `json:"host"`
	TaskID       string             `json:"taskID"`
	ClassToWatts map[string]float64 `json:"class_to_watts"`
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
func (tsk *Task) UpdateHost(newHost string) bool {
	// Validation
	isCorrectHost := false
	for _, existingHost := range constants.Hosts {
		if newHost == existingHost {
			isCorrectHost = true
		}
	}
	if !isCorrectHost {
		return false
	} else {
		tsk.Host = newHost
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

/*
 Determine the watts value to consider for each task.

 This value could either be task.Watts or task.ClassToWatts[<power class>]
 If task.ClassToWatts is not present, then return task.Watts (this would be for workloads which don't have classMapWatts)
*/
func WattsToConsider(task Task, classMapWatts bool, offer *mesos.Offer) (float64, error) {
	if classMapWatts {
		// checking if ClassToWatts was present in the workload.
		if task.ClassToWatts != nil {
			return task.ClassToWatts[offerUtils.PowerClass(offer)], nil
		} else {
			// Checking whether task.Watts is 0.0. If yes, then throwing an error.
			if task.Watts == 0.0 {
				return task.Watts, errors.New("Configuration error in task. Watts attribute is 0 for " + task.Name)
			}
			return task.Watts, nil
		}
	} else {
		// Checking whether task.Watts is 0.0. If yes, then throwing an error.
		if task.Watts == 0.0 {
			return task.Watts, errors.New("Configuration error in task. Watts attribute is 0 for " + task.Name)
		}
		return task.Watts, nil
	}
}

// Sorter implements sort.Sort interface to sort tasks by Watts requirement.
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

// Sorter implements sort.Sort interface to sort tasks by CPU requirement.
type CPUSorter []Task

func (slice CPUSorter) Len() int {
	return len(slice)
}

func (slice CPUSorter) Less(i, j int) bool {
	return slice[i].CPU < slice[j].CPU
}

func (slice CPUSorter) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// Compare two tasks.
func Compare(task1 *Task, task2 *Task) bool {
	// If comparing the same pointers (checking the addresses).
	if task1 == task2 {
		return true
	}
	if task1.TaskID != task2.TaskID {
		return false
	} else {
		return true
	}
}
