package pcp

type Victim struct {
	Watts float64
	Host string
}

type VictimSorter []Victim

func (slice VictimSorter) Len() int {
	return len(slice)
}

func (slice VictimSorter) Less(i, j int) bool {
	return slice[i].Watts >= slice[j].Watts
}

func (slice VictimSorter) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}
