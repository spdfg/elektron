/*
Constants that are used across scripts
1. The available hosts = stratos-00x (x varies from 1 to 8)
2. cap_margin = percentage of the requested power to allocate
3. power_threshold = overloading factor
5. window_size = number of tasks to consider for computation of the dynamic cap.

Also, exposing functions to update or initialize some of the constants.

TODO: Clean this up and use Mesos Attributes instead.
*/
package constants

var Hosts = []string{"stratos-001.cs.binghamton.edu", "stratos-002.cs.binghamton.edu",
	"stratos-003.cs.binghamton.edu", "stratos-004.cs.binghamton.edu",
	"stratos-005.cs.binghamton.edu", "stratos-006.cs.binghamton.edu",
	"stratos-007.cs.binghamton.edu", "stratos-008.cs.binghamton.edu"}

// Classification of the nodes in the cluster based on their power consumption.
var PowerClasses = map[string]map[string]bool{
	"ClassA": map[string]bool{
		"stratos-005.cs.binghamton.edu": true,
		"stratos-006.cs.binghamton.edu": true,
	},
	"ClassB": map[string]bool{
		"stratos-007.cs.binghamton.edu": true,
		"stratos-008.cs.binghamton.edu": true,
	},
	"ClassC": map[string]bool{
		"stratos-001.cs.binghamton.edu": true,
		"stratos-002.cs.binghamton.edu": true,
		"stratos-003.cs.binghamton.edu": true,
		"stratos-004.cs.binghamton.edu": true,
	},
}

// Add a new host to the slice of hosts.
func AddNewHost(newHost string) bool {
	// Validation
	if newHost == "" {
		return false
	} else {
		Hosts = append(Hosts, newHost)
		return true
	}
}

/*
	Lower bound of the percentage of requested power, that can be allocated to a task.

	Note: This constant is not used for the proactive cluster wide capping schemes.
*/
var PowerThreshold = 0.6 // Right now saying that a task will never be given lesser than 60% of the power it requested.

/*
  Margin with respect to the required power for a job.
  So, if power required = 10W, the node would be capped to 75%*10W.
  This value can be changed upon convenience.
*/
var CapMargin = 0.70

// Modify the cap margin.
func UpdateCapMargin(newCapMargin float64) bool {
	// Checking if the new_cap_margin is less than the power threshold.
	if newCapMargin < StarvationFactor {
		return false
	} else {
		CapMargin = newCapMargin
		return true
	}
}

/*
	The factor, that when multiplied with (task.Watts * CapMargin) results in (task.Watts * PowerThreshold).
	This is used to check whether available power, for a host in an offer, is not less than (PowerThreshold * task.Watts),
		which is assumed to result in starvation of the task.
	Here is an example,
		Suppose a task requires 100W of power. Assuming CapMargin = 0.75 and PowerThreshold = 0.6.
		So, the assumed allocated watts is 75W.
		Now, when we get an offer, we need to check whether the available power, for the host in that offer, is
			not less than 60% (the PowerTreshold) of the requested power (100W).
		To put it in other words,
			availablePower >= 100W * 0.75 * X
		where X is the StarvationFactor (80% in this case)

	Note: This constant is not used for the proactive cluster wide capping schemes.
*/
var StarvationFactor = PowerThreshold / CapMargin

// Window size for running average
var ConsiderationWindowSize = 20

// Update the window size.
func UpdateWindowSize(newWindowSize int) bool {
	// Validation
	if newWindowSize == 0 {
		return false
	} else {
		ConsiderationWindowSize = newWindowSize
		return true
	}
}
