package schedulers

import "github.com/mesos/mesos-go/scheduler"

// Names of different scheduling policies.
const (
	ff  = "first-fit"
	bp  = "bin-packing"
	mgm = "max-greedymins"
	mm  = "max-min"
)

// Scheduler class factory.
var Schedulers map[string]scheduler.Scheduler = map[string]scheduler.Scheduler{
	ff:  &FirstFit{base: base{}},
	bp:  &BinPacking{base: base{}},
	mgm: &MaxGreedyMins{base: base{}},
	mm:  &MaxMin{base: base{}},
}

// Build the scheduling policy with the options being applied.
func BuildSchedPolicy(s scheduler.Scheduler, opts ...schedPolicyOption) {
	s.(ElectronScheduler).init(opts...)
}

func SchedFactory(schedPolicyName string, opts ...schedPolicyOption) scheduler.Scheduler {
	s := Schedulers[schedPolicyName]
	BuildSchedPolicy(s, opts...)
	return s
}
