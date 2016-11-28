/*
Constants that are used across scripts
1. The available hosts = stratos-00x (x varies from 1 to 8)
2. cap_margin = percentage of the requested power to allocate
3. power_threshold = overloading factor
4. total_power = total power per node
5. window_size = number of tasks to consider for computation of the dynamic cap.

Also, exposing functions to update or initialize some of the constants.
*/
package constants

var Hosts = []string{"stratos-001.cs.binghamton.edu", "stratos-002.cs.binghamton.edu",
	"stratos-003.cs.binghamton.edu", "stratos-004.cs.binghamton.edu",
	"stratos-005.cs.binghamton.edu", "stratos-006.cs.binghamton.edu",
	"stratos-007.cs.binghamton.edu", "stratos-008.cs.binghamton.edu"}

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

// Lower bound of the percentage of requested power, that can be allocated to a task.
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

// Threshold factor that would make (Cap_margin * task.Watts) equal to (60/100 * task.Watts).
var StarvationFactor = 0.8

// Total power per node.
var TotalPower map[string]float64

// Initialize the total power per node. This should be done before accepting any set of tasks for scheduling.
func AddTotalPowerForHost(host string, totalPower float64) bool {
	// Validation
	isCorrectHost := false
	for _, existingHost := range Hosts {
		if host == existingHost {
			isCorrectHost = true
		}
	}

	if !isCorrectHost {
		return false
	} else {
		TotalPower[host] = totalPower
		return true
	}
}

// Window size for running average
var WindowSize = 160

// Update the window size.
func UpdateWindowSize(newWindowSize int) bool {
	// Validation
	if newWindowSize == 0 {
		return false
	} else {
		WindowSize = newWindowSize
		return true
	}
}
