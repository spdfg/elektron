package offerUtils

import (
	mesos "github.com/mesos/mesos-go/mesosproto"
	"strings"
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

// Determine the power class of the host in the offer
func PowerClass(offer *mesos.Offer) string {
	var powerClass string
	for _, attr := range offer.GetAttributes() {
		if attr.GetName() == "class" {
			powerClass = attr.GetText().GetValue()
		}
	}
	return powerClass
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

// Is there a mismatch between the task's host requirement and the host corresponding to the offer.
func HostMismatch(offerHost string, taskHost string) bool {
	if taskHost != "" && !strings.HasPrefix(offerHost, taskHost) {
		return true
	}
	return false
}
