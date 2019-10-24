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

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

// TestVictimSorter tests whether victim nodes are sorted in the non-increasing
// order of Watts consumption.
func TestVictimSorter(t *testing.T) {
	victims := []Victim{
		{Watts: 10.0, Host: "host1"},
		{Watts: 20.0, Host: "host2"},
		{Watts: 30.0, Host: "host3"},
		{Watts: 40.0, Host: "host4"},
	}

	sort.Sort(VictimSorter(victims))

	expectedVictimsOrder := []Victim{
		{Watts: 40.0, Host: "host4"},
		{Watts: 30.0, Host: "host3"},
		{Watts: 20.0, Host: "host2"},
		{Watts: 10.0, Host: "host1"},
	}

	for i, v := range expectedVictimsOrder {
		assert.Equal(t, v.Watts, victims[i].Watts, "failed to sorted victims")
		assert.Equal(t, v.Host, victims[i].Host, "failed to sorted victims")
	}
}
