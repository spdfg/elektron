package schedulers

import (
	"encoding/json"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	"github.com/pkg/errors"
	"os"
)

// Names of different scheduling policies.
const (
	ff  = "first-fit"
	bp  = "bin-packing"
	mgm = "max-greedymins"
	mm  = "max-min"
)

// Scheduling policy factory
var SchedPolicies map[string]SchedPolicyState = map[string]SchedPolicyState{
	ff:  &FirstFit{},
	bp:  &BinPackSortedWatts{},
	mgm: &MaxGreedyMins{},
	mm:  &MaxMin{},
}

// Initialize scheduling policy characteristics using the provided config file.
func InitSchedPolicyCharacteristics(schedPoliciesConfigFilename string) error {
	var schedPolConfig map[string]baseSchedPolicyState
	if file, err := os.Open(schedPoliciesConfigFilename); err != nil {
		return errors.Wrap(err, "Error opening file")
	} else {
		err := json.NewDecoder(file).Decode(&schedPolConfig)
		if err != nil {
			return errors.Wrap(err, "Error unmarshalling")
		}

		// Initializing.
		// TODO: Be able to unmarshal a schedPolConfig JSON into any number of scheduling policies.
		for schedPolName, schedPolState := range SchedPolicies {
			switch t := schedPolState.(type) {
			case *FirstFit:
				t.TaskDistribution = schedPolConfig[schedPolName].TaskDistribution
				t.VarianceCpuSharePerTask = schedPolConfig[schedPolName].VarianceCpuSharePerTask
			case *BinPackSortedWatts:
				t.TaskDistribution = schedPolConfig[schedPolName].TaskDistribution
				t.VarianceCpuSharePerTask = schedPolConfig[schedPolName].VarianceCpuSharePerTask
			case *MaxMin:
				t.TaskDistribution = schedPolConfig[schedPolName].TaskDistribution
				t.VarianceCpuSharePerTask = schedPolConfig[schedPolName].VarianceCpuSharePerTask
			case *MaxGreedyMins:
				t.TaskDistribution = schedPolConfig[schedPolName].TaskDistribution
				t.VarianceCpuSharePerTask = schedPolConfig[schedPolName].VarianceCpuSharePerTask
			}
		}
	}

	return nil
}

// build the scheduler with the options being applied
func buildScheduler(s sched.Scheduler, opts ...schedulerOptions) {
	s.(ElectronScheduler).init(opts...)
}

func SchedFactory(opts ...schedulerOptions) sched.Scheduler {
	s := &BaseScheduler{}
	buildScheduler(s, opts...)
	return s
}
