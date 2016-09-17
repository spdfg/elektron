package main

import (
	"encoding/json"
	"os"
	"github.com/pkg/errors"
)

type Task struct {
	CPU float64 `json:"cpu"`
	RAM float64 `json:"ram"`
	Watts float64 `json:"watts"`
	Image string `json:"image"`
	CMD string `json:"cmd"`
	Instances *int `json:"inst"`
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