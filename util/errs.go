// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"fmt"
)

// Errors is a slice of error.
type Errors []error

// Error implements the error#Error method.
func (e Errors) Error() string {
	return ToString([]error(e))
}

// String implements the stringer#String method.
func (e Errors) String() string {
	return e.Error()
}

// NewErrs returns a slice of error with a single element err.
// If err is nil, returns nil.
func NewErrs(err error) Errors {
	if err == nil {
		return nil
	}
	return []error{err}
}

// AppendErr appends err to errors if it is not nil and returns the result.
// If err is nil, it is not appended.
func AppendErr(errors []error, err error) Errors {
	if err == nil {
		if len(errors) == 0 {
			return nil
		}
		return errors
	}
	return append(errors, err)
}

// AppendErrs appends newErrs to errors and returns the result.
// If newErrs is empty, nothing is appended.
func AppendErrs(errors []error, newErrs []error) Errors {
	if len(newErrs) == 0 {
		return errors
	}
	for _, e := range newErrs {
		errors = AppendErr(errors, e)
	}
	if len(errors) == 0 {
		return nil
	}
	return errors
}

// ToString returns a string representation of errors. Any nil errors in the
// slice are skipped.
func ToString(errors []error) string {
	var out string
	for i, e := range errors {
		if e == nil {
			continue
		}
		if i != 0 {
			out += ", "
		}
		out += e.Error()
	}
	return out
}

// PrefixErrors prefixes each error within the supplied Errors slice with the
// string pfx.
func PrefixErrors(errs Errors, pfx string) Errors {
	var nerr Errors
	for _, err := range errs {
		nerr = append(nerr, fmt.Errorf("%s: %s", pfx, err))
	}
	return nerr
}

// UniqueErrors returns the unique errors from the supplied Errors slice. Errors
// are considered equal if they have equal stringified values.
func UniqueErrors(errs Errors) Errors {
	u := map[string]error{}
	for _, err := range errs {
		u[fmt.Sprintf("%v", err)] = err
	}

	var ne Errors
	for _, err := range u {
		ne = append(ne, err)
	}
	return ne
}
