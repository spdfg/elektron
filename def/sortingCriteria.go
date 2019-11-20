// Copyright (C) 2018 spdfg
//
// This file is part of Elektron.
//
// Elektron is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Elektron is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Elektron.  If not, see <http://www.gnu.org/licenses/>.
//

package def

// the sortBy function that takes a task reference and returns the resource to consider when sorting.
type SortBy func(task *Task) float64

// Possible Sorting Criteria.
// Each holds a closure that fetches the required resource from the
// 	given task reference.
var (
	SortByCPU   = func(task *Task) float64 { return task.CPU }
	SortByRAM   = func(task *Task) float64 { return task.RAM }
	SortByWatts = func(task *Task) float64 { return task.Watts }
)
