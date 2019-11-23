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
	"github.com/pkg/errors"
	"github.com/spdfg/elektron/utilities/validation"
	"regexp"
)

// taskValidator is a validator that validates one or more attributes of a task.
type taskValidator func(Task) error

// ValidatorForTask returns a validator that runs all provided taskValidators and
// returns an error corresponding to the first taskValidator that failed.
func ValidatorForTask(t Task, taskValidators ...taskValidator) validation.Validator {
	return func() error {
		for _, tv := range taskValidators {
			if err := tv(t); err != nil {
				return err
			}
		}

		return nil
	}
}

// withNameValidator returns a taskValidator that checks whether the task name is valid.
func withNameValidator() taskValidator {
	return func(t Task) error {
		// Task name cannot be empty string.
		if t.Name == "" {
			return errors.New("task name cannot be empty string")
		}

		// Task name cannot contain tabs or spaces.
		matched, _ := regexp.MatchString("\\t+|\\s+", t.Name)
		if matched {
			return errors.New("task name cannot contain tabs or spaces")
		}

		return nil
	}
}

// withResourceValidator returns a taskValidator that checks whether the resource requirements are valid.
// Currently, only requirements for traditional resources (cpu and memory) are validated.
func withResourceValidator() taskValidator {
	return func(t Task) error {
		// CPU value cannot be 0.
		if t.CPU == 0.0 {
			return errors.New("cpu resource for task cannot be 0")
		}

		// RAM value cannot be 0.
		if t.RAM == 0.0 {
			return errors.New("memory resource for task cannot be 0")
		}

		return nil
	}
}

// withImageValidator returns a taskValidator that checks whether a valid image has been
// provided for the task.
func withImageValidator() taskValidator {
	return func(t Task) error {
		// Image cannot be empty.
		if t.Image == "" {
			return errors.New("valid image needs to be provided for task")
		}

		return nil
	}
}
