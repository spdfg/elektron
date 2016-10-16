package def

import (
	"encoding/json"
	"github.com/pkg/errors"
	"os"
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
