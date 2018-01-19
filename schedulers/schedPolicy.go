package schedulers

import (
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"
)

type SchedPolicyContext interface {
	// Change the state of scheduling.
	SwitchSchedPol(s SchedPolicyState)
}

type SchedPolicyState interface {
	// Define the particular scheduling policy's methodology of resource offer consumption.
	ConsumeOffers(SchedPolicyContext, sched.SchedulerDriver, []*mesos.Offer)
}
