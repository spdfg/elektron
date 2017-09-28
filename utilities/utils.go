package utilities

/*
The Pair and PairList have been taken from google groups forum,
https://groups.google.com/forum/#!topic/golang-nuts/FT7cjmcL7gw.
*/

// Utility struct that helps in sorting map[string]float64 by value.
type Pair struct {
	Key   string
	Value float64
}

// A slice of pairs that implements the sort.Interface to sort by value.
type PairList []Pair

// Convert map[string]float64 to PairList.
func GetPairList(m map[string]float64) PairList {
	pl := PairList{}
	for k, v := range m {
		pl = append(pl, Pair{Key: k, Value: v})
	}
	return pl
}

// Swap pairs in the PairList.
func (plist PairList) Swap(i, j int) {
	plist[i], plist[j] = plist[j], plist[i]
}

// Get the length of the pairlist.
func (plist PairList) Len() int {
	return len(plist)
}

// Compare two elements in pairlist.
func (plist PairList) Less(i, j int) bool {
	return plist[i].Value < plist[j].Value
}
