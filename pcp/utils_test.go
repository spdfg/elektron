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
	"container/ring"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAverageNodePowerHistory(t *testing.T) {
	// Creating a ring to hold 5 seconds of node power readings.
	// Note that as we're considering both CPU and DRAM packages, this amounts to 20 entries
	// in the ring.
	nodePowerRecordings := ring.New(20)
	for i := 1; i <= 20; i++ {
		nodePowerRecordings.Value = float64(i)
		nodePowerRecordings = nodePowerRecordings.Next()
	}

	assert.Equal(t, 42.0, AverageNodePowerHistory(nodePowerRecordings))
}

func TestAverageClusterPowerHistory(t *testing.T) {
	// Creating a ring to hold 5 seconds of cluster power consumption.
	clusterPowerRecordings := ring.New(5)
	for i := 1; i <= 5; i++ {
		clusterPowerRecordings.Value = float64(i)
		clusterPowerRecordings = clusterPowerRecordings.Next()
	}

	assert.Equal(t, 3.0, AverageClusterPowerHistory(clusterPowerRecordings))
}
