/*
Constants that are used across scripts
1. The available hosts = stratos-00x (x varies from 1 to 8)
2. CapMargin = percentage of the requested power to allocate
3. ConsiderationWindowSize = number of tasks to consider for computation of the dynamic cap.
*/
package constants

var Hosts = []string{"stratos-001.cs.binghamton.edu", "stratos-002.cs.binghamton.edu",
	"stratos-003.cs.binghamton.edu", "stratos-004.cs.binghamton.edu",
	"stratos-005.cs.binghamton.edu", "stratos-006.cs.binghamton.edu",
	"stratos-007.cs.binghamton.edu", "stratos-008.cs.binghamton.edu"}

// Classification of the nodes in the cluster based on their power consumption.
var PowerClasses = map[string]map[string]bool{
	"A": map[string]bool{
		"stratos-005.cs.binghamton.edu": true,
		"stratos-006.cs.binghamton.edu": true,
	},
	"B": map[string]bool{
		"stratos-007.cs.binghamton.edu": true,
		"stratos-008.cs.binghamton.edu": true,
	},
	"C": map[string]bool{
		"stratos-001.cs.binghamton.edu": true,
		"stratos-002.cs.binghamton.edu": true,
		"stratos-003.cs.binghamton.edu": true,
		"stratos-004.cs.binghamton.edu": true,
	},
}

/*
  Margin with respect to the required power for a job.
  So, if power required = 10W, the node would be capped to CapMargin * 10W.
  This value can be changed upon convenience.
*/
var CapMargin = 0.70

// Window size for running average
var ConsiderationWindowSize = 20
