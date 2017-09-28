package def

// The sortBy function that takes a task reference and returns the resource to consider when sorting.
type sortBy func(task *Task) float64

// Possible Sorting Criteria.
// Each holds a closure that fetches the required resource from the
// 	given task reference.
var (
	SortByCPU   = func(task *Task) float64 { return task.CPU }
	SortByRAM   = func(task *Task) float64 { return task.RAM }
	SortByWatts = func(task *Task) float64 { return task.Watts }
)
