package utilities

import (
	"gitlab.com/spdf/elektron/def"
	"gitlab.com/spdf/elektron/utilities/offerUtils"
	"errors"
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	"sync"
)

type TrackResourceUsage struct {
	perHostResourceAvailability map[string]*ResourceCount
	sync.Mutex
}

// Maintain information regarding the usage of the cluster resources.
// This information is maintained for each node in the cluster.
type ResourceCount struct {
	// Total resources available.
	TotalCPU   float64
	TotalRAM   float64
	TotalWatts float64

	// Resources currently unused.
	UnusedCPU   float64
	UnusedRAM   float64
	UnusedWatts float64
}

// Increment unused resources.
func (rc *ResourceCount) IncrUnusedResources(tr def.TaskResources) {
	rc.UnusedCPU += tr.CPU
	rc.UnusedRAM += tr.Ram
	rc.UnusedWatts += tr.Watts
}

// Decrement unused resources.
func (rc *ResourceCount) DecrUnusedResources(tr def.TaskResources) {
	rc.UnusedCPU -= tr.CPU
	rc.UnusedRAM -= tr.Ram
	rc.UnusedWatts -= tr.Watts
}

var truInstance *TrackResourceUsage

func getTRUInstance() *TrackResourceUsage {
	if truInstance == nil {
		truInstance = newResourceUsageTracker()
	}
	return truInstance
}

func newResourceUsageTracker() *TrackResourceUsage {
	return &TrackResourceUsage{
		perHostResourceAvailability: make(map[string]*ResourceCount),
	}
}

// Determine the total available resources from the first round of mesos resource offers.
func RecordTotalResourceAvailability(offers []*mesos.Offer) {
	tru := getTRUInstance()
	tru.Lock()
	defer tru.Unlock()
	for _, offer := range offers {
		// If first offer received from Mesos Agent.
		if _, ok := tru.perHostResourceAvailability[*offer.SlaveId.Value]; !ok {
			cpu, mem, watts := offerUtils.OfferAgg(offer)
			tru.perHostResourceAvailability[*offer.SlaveId.Value] = &ResourceCount{
				TotalCPU:   cpu,
				TotalRAM:   mem,
				TotalWatts: watts,

				// Initially, all resources are used.
				UnusedCPU:   cpu,
				UnusedRAM:   mem,
				UnusedWatts: watts,
			}
		}
	}
}

// Resource availability update scenarios.
var resourceAvailabilityUpdateScenario = map[string]func(mesos.TaskID, mesos.SlaveID) error{
	"ON_TASK_TERMINAL_STATE": func(taskID mesos.TaskID, slaveID mesos.SlaveID) error {
		tru := getTRUInstance()
		tru.Lock()
		defer tru.Unlock()
		if taskResources, err := def.GetResourceRequirement(*taskID.Value); err != nil {
			return err
		} else {
			// Checking if first resource offer already recorded for slaveID.
			if resCount, ok := tru.perHostResourceAvailability[*slaveID.Value]; ok {
				resCount.IncrUnusedResources(taskResources)
			} else {
				// Shouldn't be here.
				// First round of mesos resource offers not recorded.
				return errors.New("Recource Availability not recorded for " + *slaveID.Value)
			}
			return nil
		}
	},
	"ON_TASK_ACTIVE_STATE": func(taskID mesos.TaskID, slaveID mesos.SlaveID) error {
		tru := getTRUInstance()
		tru.Lock()
		defer tru.Unlock()
		if taskResources, err := def.GetResourceRequirement(*taskID.Value); err != nil {
			return err
		} else {
			// Checking if first resource offer already recorded for slaveID.
			if resCount, ok := tru.perHostResourceAvailability[*slaveID.Value]; ok {
				resCount.DecrUnusedResources(taskResources)
			} else {
				// Shouldn't be here.
				// First round of mesos resource offers not recorded.
				return errors.New("Resource Availability not recorded for " + *slaveID.Value)
			}
			return nil
		}
	},
}

// Updating cluster resource availability based on the given scenario.
func ResourceAvailabilityUpdate(scenario string, taskID mesos.TaskID, slaveID mesos.SlaveID) error {
	if updateFunc, ok := resourceAvailabilityUpdateScenario[scenario]; ok {
		// Applying the update function
		updateFunc(taskID, slaveID)
		return nil
	} else {
		// Incorrect scenario specified.
		return errors.New("Incorrect scenario specified for resource availability update: " + scenario)
	}
}

// Retrieve clusterwide resource availability.
func GetClusterwideResourceAvailability() ResourceCount {
	tru := getTRUInstance()
	tru.Lock()
	defer tru.Unlock()
	clusterwideResourceCount := ResourceCount{}
	for _, resCount := range tru.perHostResourceAvailability {
		// Aggregating the total CPU, RAM and Watts.
		clusterwideResourceCount.TotalCPU += resCount.TotalCPU
		clusterwideResourceCount.TotalRAM += resCount.TotalRAM
		clusterwideResourceCount.TotalWatts += resCount.TotalWatts

		// Aggregating the total unused CPU, RAM and Watts.
		clusterwideResourceCount.UnusedCPU += resCount.UnusedCPU
		clusterwideResourceCount.UnusedRAM += resCount.UnusedRAM
		clusterwideResourceCount.UnusedWatts += resCount.UnusedWatts
	}

	return clusterwideResourceCount
}

// Retrieve resource availability for each host in the cluster.
func GetPerHostResourceAvailability() map[string]*ResourceCount {
	tru := getTRUInstance()
	tru.Lock()
	defer tru.Unlock()
	return tru.perHostResourceAvailability
}
