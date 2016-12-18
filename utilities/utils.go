package utilities

import "errors"

/*
The Pair and PairList have been taken from google groups forum,
https://groups.google.com/forum/#!topic/golang-nuts/FT7cjmcL7gw
*/

// Utility that helps in sorting a map[string]float64 by value.
type Pair struct {
	Key   string
	Value float64
}

// A slice of pairs that implements the sort.Interface to sort by value.
type PairList []Pair

// Swap pairs in the PairList
func (plist PairList) Swap(i, j int) {
	plist[i], plist[j] = plist[j], plist[i]
}

// function to return the length of the pairlist.
func (plist PairList) Len() int {
	return len(plist)
}

// function to compare two elements in pairlist.
func (plist PairList) Less(i, j int) bool {
	return plist[i].Value < plist[j].Value
}

// convert a PairList to a map[string]float64
func OrderedKeys(plist PairList) ([]string, error) {
	// Validation
	if plist == nil {
		return nil, errors.New("Invalid argument: plist")
	}
	orderedKeys := make([]string, len(plist))
	for _, pair := range plist {
		orderedKeys = append(orderedKeys, pair.Key)
	}
	return orderedKeys, nil
}

// determine the max value
func Max(a, b float64) float64 {
	if a > b {
		return a
	} else {
		return b
	}
}
