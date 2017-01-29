package schedulers

import (
	"fmt"
	"log"
	"bitbucket.org/sunybingcloud/electron/def"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"bitbucket.org/sunybingcloud/electron/utilities/offerUtils"
)

func coLocated(tasks map[string]bool) {

	for task := range tasks {
		log.Println(task)
	}

	fmt.Println("---------------------")
}

/*
 Determine the watts value to consider for each task.

 This value could either be task.Watts or task.ClassToWatts[<power class>]
 If task.ClassToWatts is not present, then return task.Watts (this would be for workloads which don't have classMapWatts)
*/
func wattsToConsider(task def.Task, classMapWatts bool, offer *mesos.Offer) float64 {
	if classMapWatts {
		// checking if ClassToWatts was present in the workload.
		if task.ClassToWatts != nil {
			return task.ClassToWatts[offerUtils.PowerClass(offer)]
		} else {
			return task.Watts
		}
	} else {
		return task.Watts
	}
}
