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

type OffersSorter []*mesos.Offer

func (offersSorter OffersSorter) Len() int {
	return len(offersSorter)
}

func (offersSorter OffersSorter) Swap(i, j int) {
	offersSorter[i], offersSorter[j] = offersSorter[j], offersSorter[i]
}

func (offersSorter OffersSorter) Less(i, j int) bool {
	// getting CPU resource availability of offersSorter[i]
	cpu1, _, _ := OfferAgg(offersSorter[i])
	// getting CPU resource availability of offersSorter[j]
	cpu2, _, _ := OfferAgg(offersSorter[j])
	return cpu1 <= cpu2
}

func coLocated(tasks map[string]bool) {

	for task := range tasks {
		log.Println(task)
	}

	fmt.Println("---------------------")
}
