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

// Package validation contains utilities to help run validators.
package validation

import "github.com/pkg/errors"

// Validator is a function that performs some sort of validation.
// To keep things generic, this function does not accept any arguments.
// In practice, a validator could be a closure.
// Assume we are validating the below struct.
// 	type A struct { value string }
// One could then create a validator for the above struct like this:
// 	func AValidator(a A) Validator {
// 		return func() error {
// 			if a.value == "" {
// 				return errors.New("invalid value")
//			}
//			return nil
// 		}
//	}
type Validator func() error

// Validate a list of validators.
// If validation fails, then wrap the returned error with the given base
// error message.
func Validate(baseErrMsg string, validators ...Validator) error {
	for _, v := range validators {
		if err := v(); err != nil {
			return errors.Wrap(err, baseErrMsg)
		}
	}

	return nil
}
