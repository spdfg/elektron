package offerUtils

import (
	"bitbucket.org/sunybingcloud/electron-archive/utilities/offerUtils"
	"bitbucket.org/sunybingcloud/electron/constants"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"log"
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

// Implements the sort.Sort interface to sort Offers based on CPU.
// TODO: Have a generic sorter that sorts based on a defined requirement (CPU, RAM, DISK or Watts)
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

// If the host in the offer is a new host, add the host to the set of Hosts and
// register the powerclass of this host.
func UpdateEnvironment(offer *mesos.Offer) {
	var host = offer.GetHostname()
	// If this host is not present in the set of hosts.
	if _, ok := constants.Hosts[host]; !ok {
		log.Printf("New host detected. Adding host [%s]", host)
		// Add this host.
		constants.Hosts[host] = struct{}{}
		// Get the power class of this host.
		class := offerUtils.PowerClass(offer)
		log.Printf("Registering the power class... Host [%s] --> PowerClass [%s]", host, class)
		// If new power class, register the power class.
		if _, ok := constants.PowerClasses[class]; !ok {
			constants.PowerClasses[class] = make(map[string]struct{})
		}
		// If the host of this class is not yet present in PowerClasses[class], add it.
		if _, ok := constants.PowerClasses[class][host]; !ok {
			constants.PowerClasses[class][host] = struct{}{}
		}
	}
}
