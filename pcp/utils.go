package pcp

import (
	"container/ring"
	"math"
	"strconv"
	"strings"
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

func sumAndNormalize(tokenSlice []string, normalizer float64) float64 {
	sum := 0.0
	for _, value := range tokenSlice {
		i, _ := strconv.ParseFloat(value, 64)
		sum += i
	}
	return sum / normalizer
}

func utilization(used string, free string) float64 {
	u, _ := strconv.ParseFloat(used, 64)
	f, _ := strconv.ParseFloat(free, 64)
	return u / (u + f)
}

func cpuUtilPerNode(text string) []float64 {
	tokenSlice := strings.Split(text, ",")
	cpuUtil := make([]float64, 8)
	for i := 0; i < 8; i++ {
		cpuUtil[i] = utilization(tokenSlice[8+i], tokenSlice[24+i])
	}
	return cpuUtil
}

func memUtilPerNode(text string) []float64 {
	tokenSlice := strings.Split(text, ",")
	memUtil := make([]float64, 8)
	for i := 0; i < 8; i++ {
		memUtil[i] = utilization(tokenSlice[40+i], tokenSlice[32+i])
	}
	return memUtil
}
