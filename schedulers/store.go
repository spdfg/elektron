package schedulers

import (
	"gitlab.com/spdf/elektron/utilities"
	"encoding/json"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	"github.com/pkg/errors"
	"os"
	"sort"
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

// Scheduling policies to choose when switching
var schedPoliciesToSwitch map[int]struct {
	spName string
	sp     SchedPolicyState
} = make(map[int]struct {
	spName string
	sp     SchedPolicyState
})

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

		// Initialize schedPoliciesToSwitch to allow binary searching for scheduling policy switching.
		spInformation := map[string]float64{}
		for spName, sp := range SchedPolicies {
			spInformation[spName] = sp.GetInfo().taskDist
		}
		spInformationPairList := utilities.GetPairList(spInformation)
		// Sorting spInformationPairList in non-increasing order of taskDist.
		sort.SliceStable(spInformationPairList, func(i, j int) bool {
			return spInformationPairList[i].Value < spInformationPairList[j].Value
		})
		// Initializing scheduling policies that are setup for switching.
		index := 0
		for _, spInformationPair := range spInformationPairList {
			if spInformationPair.Value != 0 {
				schedPoliciesToSwitch[index] = struct {
					spName string
					sp     SchedPolicyState
				}{
					spName: spInformationPair.Key,
					sp:     SchedPolicies[spInformationPair.Key],
				}
				index++
			}
		}

		// Initializing the next and previous policy based on the the round-robin ordering.
		// The next policy for policy at N would correspond to the value at index N+1 in schedPoliciesToSwitch.
		for curPolicyIndex := 0; curPolicyIndex < len(schedPoliciesToSwitch); curPolicyIndex++ {
			info := struct {
				nextPolicyName string
				prevPolicyName string
			}{}
			if curPolicyIndex == 0 {
				info.prevPolicyName = schedPoliciesToSwitch[len(schedPoliciesToSwitch)-1].spName
			} else {
				info.prevPolicyName = schedPoliciesToSwitch[curPolicyIndex-1].spName
			}
			info.nextPolicyName = schedPoliciesToSwitch[(curPolicyIndex+1)%len(schedPoliciesToSwitch)].spName
			schedPoliciesToSwitch[curPolicyIndex].sp.UpdateLinks(info)
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
