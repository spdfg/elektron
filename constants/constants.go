// TODO: Clean this up and use Mesos Attributes instead.
package constants

var Hosts = make(map[string]struct{})

/*
 Classification of the nodes in the cluster based on their Thermal Design Power (TDP).
*/
var PowerClasses = make(map[string]map[string]struct{})

// Threshold below which a host should be capped
var LowerCapLimit = 12.5
