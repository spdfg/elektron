// Copyright (C) 2018 spdf
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

type Victim struct {
	Watts float64
	Host  string
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
