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

package def

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSortingCriteria(t *testing.T) {
	task := &Task{CPU: 1.0, RAM: 1024.0, Watts: 50.0}
	assert.Equal(t, 1.0, SortByCPU(task))
	assert.Equal(t, 1024.0, SortByRAM(task))
	assert.Equal(t, 50.0, SortByWatts(task))
}
