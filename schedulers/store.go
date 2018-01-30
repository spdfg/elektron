package schedulers

import (
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
)

// Names of different scheduling policies.
const (
	ff          = "first-fit"
	bp          = "bin-packing"
	mgm         = "max-greedymins"
	mm          = "max-min"
)

// Scheduling policy factory
var SchedPolicies map[string]SchedPolicyState = map[string]SchedPolicyState{
	ff:          &FirstFit{},
	bp:          &BinPackSortedWatts{},
	mgm:         &MaxGreedyMins{},
	mm:          &MaxMin{},
}

// build the scheduling policy with the options being applied
func buildScheduler(s sched.Scheduler, opts ...schedPolicyOption) {
	s.(ElectronScheduler).init(opts...)
}

func SchedFactory(opts ...schedPolicyOption) sched.Scheduler {
	s := &BaseScheduler{}
	buildScheduler(s, opts...)
	return s
}
