package def

// Creating an enumeration of the resources in def.Task.
var TaskResourceNames []string

type SortCriteria int

// Map a task's resource name to the sorting criteria (an integer corresponding to the enumeration).
func resourceToSortCriteria(resourceName string) SortCriteria {
	// Appending resourceName to TaskResourceNames.
	TaskResourceNames = append(TaskResourceNames, resourceName)

	// Considering index of resource in TaskResourceNames to be the int mapping.
	return SortCriteria(len(TaskResourceNames) - 1)
}

func (sc SortCriteria) String() string {
	return TaskResourceNames[int(sc)]
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
