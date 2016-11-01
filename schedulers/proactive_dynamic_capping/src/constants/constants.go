/*
Constants that are used across scripts
1. The available hosts = stratos-00x (x varies from 1 to 8)
2. cap_margin = percentage of the requested power to allocate
3. power_threshold = overloading factor
4. total_power = total power per node
5. window_size = number of tasks to consider for computation of the dynamic cap.
*/
package constants

var Hosts = []string{"stratos-001", "stratos-002",
            "stratos-003", "stratos-004",
            "stratos-005", "stratos-006",
            "stratos-007", "stratos-008"}

/*
  Margin with respect to the required power for a job.
  So, if power required = 10W, the node would be capped to 75%*10W.
  This value can be changed upon convenience.
*/
var Cap_margin = 0.75

// Lower bound of the power threshold for a tasks
var Power_threshold = 0.6

// Total power per node
var Total_power = map[string]float64 {
  "stratos-001": 100.0,
  "stratos-002": 150.0,
  "stratos-003": 80.0,
  "stratos-004": 90.0,
  "stratos-005": 200.0,
  "stratos-006": 100.0,
  "stratos-007": 175.0,
  "stratos-008": 175.0,
}

// Window size for running average
var Window_size = 3
