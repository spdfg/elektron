package utilities

import "errors"

// Utility struct that helps in sorting the available power by value.
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
	ordered_keys := make([]string, len(plist))
	for _, pair := range plist {
		ordered_keys = append(ordered_keys, pair.Key)
	}
	return ordered_keys, nil
}

// determine the max value
func Max(a, b float64) float64 {
	if a > b {
		return a
	} else {
		return b
	}
}
