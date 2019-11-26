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
	"github.com/spdfg/elektron/utilities/validation"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidatorForTask(t *testing.T) {
	getValidator := func(t Task) validation.Validator {
		return ValidatorForTask(t,
			withNameValidator(),
			withImageValidator(),
			withResourceValidator())
	}

	test := func(task Task, expectErr bool, message string) {
		validator := getValidator(task)
		if expectErr {
			assert.Error(t, validation.Validate(message, validator))
		} else {
			assert.NoError(t, validation.Validate(message, validator))
		}
	}

	inst := 10
	validTask := Task{
		Name:      "minife",
		CPU:       3.0,
		RAM:       4096,
		Watts:     50,
		Image:     "rdelvalle/minife:electron1",
		CMD:       "cd src && mpirun -np 1 miniFE.x -nx 100 -ny 100 -nz 100",
		Instances: &inst,
	}
	test(validTask, false, "invalid task definition")
	// Task with empty name.
	invalidTaskEmptyName := validTask
	invalidTaskEmptyName.Name = ""
	test(invalidTaskEmptyName, true, "invalid task definition")
	// Task with name that contains spaces.
	invalidTaskNameWithSpaces := validTask
	invalidTaskNameWithSpaces.Name = "my task"
	test(invalidTaskNameWithSpaces, true, "invalid task definition")
	// Task with invalid image.
	invalidTaskImage := validTask
	invalidTaskImage.Image = ""
	test(invalidTaskImage, true, "invalid task definition")
	// Task with invalid cpu resources.
	invalidTaskResourcesCPU := validTask
	invalidTaskResourcesCPU.CPU = 0
	test(invalidTaskResourcesCPU, true, "invalid task definition")
	// Task with invalid memory resources.
	invalidTaskResourcesRAM := validTask
	invalidTaskResourcesRAM.RAM = 0
	test(invalidTaskResourcesRAM, true, "invalid task definition")
}
