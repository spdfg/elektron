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
