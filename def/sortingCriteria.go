package def

// Creating an enumeration of the resources in def.Task.
var taskResourceNames []string

type sortCriteria int

// Map a task's resource name to the sorting criteria (an integer corresponding to the enumeration).
func resourceToSortCriteria(resourceName string) sortCriteria {
	// Appending resourceName to TaskResourceNames.
	taskResourceNames = append(taskResourceNames, resourceName)

	// Considering index of resource in TaskResourceNames to be the int mapping.
	return sortCriteria(len(taskResourceNames) - 1)
}

func (sc sortCriteria) String() string {
	return taskResourceNames[int(sc)]
}

// Possible Sorting Criteria
// Note: the value of the string passed as argument to resourceToSortCriteria() should be the same (case-sensitive)
// 	as the name of the of the corresponding resource in the struct.
var (
	CPU = resourceToSortCriteria("CPU")
	RAM = resourceToSortCriteria("RAM")
	Watts = resourceToSortCriteria("Watts")
	Instances = resourceToSortCriteria("Instances")
)
