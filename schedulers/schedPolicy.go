package schedulers

import (
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	"math/rand"
)

type SchedPolicyContext interface {
	// Change the state of scheduling.
	SwitchSchedPol(s SchedPolicyState)
}

type SchedPolicyState interface {
	// Define the particular scheduling policy's methodology of resource offer consumption.
	ConsumeOffers(SchedPolicyContext, sched.SchedulerDriver, []*mesos.Offer)
}

type baseSchedPolicyState struct {
	SchedPolicyState
	// Keep track of the number of tasks that have been scheduled.
	numTasksScheduled int
}

func (bsps *baseSchedPolicyState) switchIfNecessary(spc SchedPolicyContext) {
	baseSchedRef := spc.(*BaseScheduler)
	// Switch scheduling policy only if feature enabled from CLI
	if baseSchedRef.schedPolSwitchEnabled {
		// Need to recompute schedulWindow for the next offer cycle.
		// The next scheduling policy will schedule at max schedWindow number of tasks.
		baseSchedRef.curSchedWindow = baseSchedRef.schedWindowResStrategy.Apply(
			func() interface{} { return baseSchedRef.tasks })
		// Switching to a random scheduling policy.
		// TODO: Switch based on some criteria.
		index := rand.Intn(len(SchedPolicies))
		for _, v := range SchedPolicies {
			if index == 0 {
				spc.SwitchSchedPol(v)
				// If switched to a different scheduling policy,
				// then setting the numberTasksScheduled to 0.
				if v != bsps {
					bsps.numTasksScheduled = 0
				}
				break
			}
			index--
		}
	}
}
