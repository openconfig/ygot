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
package ytypes

// Errors is a slice of error.
type Errors []error

// Error implements the error#Error method.
func (e Errors) Error() string {
	return errStr([]error(e))
}

// String implements the stringer#String method.
func (e Errors) String() string {
	return e.Error()
}

// appendErr appends err to errors if it is not nil and returns the result.
func appendErr(errors []error, err error) []error {
	if len(errors) == 0 && err == nil {
		return nil
	}
	return append(errors, err)
}

// appendErrs appends newErrs to errors and returns the result.
func appendErrs(errors []error, newErrs []error) []error {
	if len(errors) == 0 && len(newErrs) == 0 {
		return nil
	}
	return append(errors, newErrs...)
}

// errStr returns a string representation of errors.
func errStr(errors []error) string {
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
