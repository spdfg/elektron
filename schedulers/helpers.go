package schedulers

import (
	"fmt"
	"log"
)

func coLocated(tasks map[string]bool) {

	for task := range tasks {
		log.Println(task)
	}

	fmt.Println("---------------------")
}
