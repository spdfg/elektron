package schedulers

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"log"
)

var (
	defaultFilter = &mesos.Filters{RefuseSeconds: proto.Float64(1)}
	longFilter    = &mesos.Filters{RefuseSeconds: proto.Float64(1000)}
)

func OfferAgg(offer *mesos.Offer) (float64, float64, float64) {
	var cpus, mem, watts float64

	for _, resource := range offer.Resources {
		switch resource.GetName() {
		case "cpus":
			cpus += *resource.GetScalar().Value
		case "mem":
			mem += *resource.GetScalar().Value
		case "watts":
			watts += *resource.GetScalar().Value
		}
	}

	return cpus, mem, watts
}

func coLocated(tasks map[string]bool) {

	for task := range tasks {
		log.Println(task)
	}

	fmt.Println("---------------------")
}
