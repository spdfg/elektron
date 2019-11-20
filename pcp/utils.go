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

package pcp

import (
	"container/ring"
	"math"
)

var RAPLUnits = math.Pow(2, -32)

func AverageNodePowerHistory(history *ring.Ring) float64 {

	total := 0.0
	count := 0.0

	history.Do(func(x interface{}) {
		if val, ok := x.(float64); ok { //Add it if we can get a float
			total += val
			count++
		}
	})

	if count == 0.0 {
		return 0.0
	}

	// TODO (pkaushik) handle cases when DRAM power is not being monitored.
	count /= 4 // Two PKGs, two DRAM for all nodes currently.

	return (total / count)
}

// TODO: Figure a way to merge this and avgpower.
func AverageClusterPowerHistory(history *ring.Ring) float64 {

	total := 0.0
	count := 0.0

	history.Do(func(x interface{}) {
		if val, ok := x.(float64); ok { // Add it if we can get a float.
			total += val
			count++
		}
	})

	if count == 0.0 {
		return 0.0
	}

	return (total / count)
}
