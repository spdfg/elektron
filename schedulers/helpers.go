package schedulers

import (
	"bitbucket.org/sunybingcloud/elektron/constants"
	"fmt"
	"log"
)

func coLocated(tasks map[string]bool) {

	for task := range tasks {
		log.Println(task)
	}

	fmt.Println("---------------------")
}

// Get the powerClass of the given hostname
func hostToPowerClass(hostName string) string {
	for powerClass, hosts := range constants.PowerClasses {
		if _, ok := hosts[hostName]; ok {
			return powerClass
		}
	}
	return ""
}
