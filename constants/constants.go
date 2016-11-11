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

var Hosts = []string{"stratos-001", "stratos-002",
            "stratos-003", "stratos-004",
            "stratos-005", "stratos-006",
            "stratos-007", "stratos-008"}

// Add a new host to the slice of hosts.
func AddNewHost(new_host string) bool {
  // Validation
  if new_host == "" {
    return false
  } else {
    Hosts = append(Hosts, new_host)
    return true
  }
}

// Lower bound of the percentage of requested power, that can be allocated to a task.
var Power_threshold = 0.6 // Right now saying that a task will never be given lesser than 60% of the power it requested.

/*
  Margin with respect to the required power for a job.
  So, if power required = 10W, the node would be capped to 75%*10W.
  This value can be changed upon convenience.
*/
var Cap_margin = 0.75

// Modify the cap margin.
func UpdateCapMargin(new_cap_margin float64) bool {
  // Checking if the new_cap_margin is less than the power threshold.
  if new_cap_margin < Starvation_factor {
    return false
  } else {
    Cap_margin = new_cap_margin
    return true
  }
}


// Threshold factor that would make (Cap_margin * task.Watts) equal to (60/100 * task.Watts).
var Starvation_factor = 0.8

// Total power per node.
var Total_power map[string]float64

// Initialize the total power per node. This should be done before accepting any set of tasks for scheduling.
func AddTotalPowerForHost(host string, total_power float64) bool {
  // Validation
  is_correct_host := false
  for _, existing_host := range Hosts {
    if host == existing_host {
      is_correct_host = true
    }
  }

  if !is_correct_host {
    return false
  } else {
    Total_power[host] = total_power
    return true
  }
}

// Window size for running average
var Window_size = 10

// Update the window size.
func UpdateWindowSize(new_window_size int) bool {
  // Validation
  if new_window_size == 0 {
      return false
  } else{
    Window_size = new_window_size
    return true
  }
}

// // Time duration between successive cluster wide capping.
// var Clusterwide_cap_interval = 10 // Right now capping the cluster at 10 second intervals.
//
// // Modify the cluster wide capping interval. We can update the interval depending on the workload.
// // TODO: If the workload is heavy then we can set a longer interval, while on the other hand,
// //  if the workload is light then a smaller interval is sufficient.
// func UpdateClusterwideCapInterval(new_interval int) bool {
//   // Validation
//   if new_interval == 0.0 {
//     return false
//   } else {
//     Clusterwide_cap_interval = new_interval
//     return true
//   }
// }
