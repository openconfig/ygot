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

import (
	"fmt"
	"reflect"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-7.9.

// validateChoice validates a choice struct in the structValue data tree against
// the given schema. It ensures that fields from only one case are selected from
// the available set. Since a case may contain choice elements that are not
// named in the data tree, the function recurses until it reaches a named
// element in such cases. It returns all the field names that were selected in
// the data tree from the Choice schema.
func validateChoice(schema *yang.Entry, structValue ygot.GoStruct) (selected []string, errors []error) {
	util.DbgPrint("validateChoice with value %s, schema name %s\n", util.ValueStrDebug(structValue), schema.Name)
	// Validate that multiple cases are not selected. Since choice is always
	// inside a container, there's no need to validate each individual field
	// since that is part of container validation.
	var selectedCases []string
	for _, caseSchema := range schema.Dir {
		sel, errs := IsCaseSelected(caseSchema, structValue)
		selected = append(selected, sel...)
		errors = util.AppendErrs(errors, errs)
		if len(sel) > 0 {
			selectedCases = append(selectedCases, caseSchema.Name)
		}
	}

	if len(selectedCases) > 1 {
		errors = util.AppendErr(errors, fmt.Errorf("multiple cases %v selected for choice %s", selectedCases, schema.Name))
	}

	return
}

// IsCaseSelected reports whether a case with the given schema has been selected
// in the given value struct. The top level of the struct is checked, and any
// choices present in the schema are recursively followed to determine whether
// any case is selected for that choice schema subtree. It returns a slice with
// the names of all fields in the case that were selected.
func IsCaseSelected(schema *yang.Entry, value interface{}) (selected []string, errors []error) {
	v := reflect.ValueOf(value).Elem()
	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).IsNil() {
			fieldType := v.Type().Field(i)
			cs, err := childSchema(schema, fieldType)
			if err != nil {
				errors = util.AppendErr(errors, err)
				continue
			}
			if cs != nil {
				// Since the field is non-nil and matching schema is found for
				// the field under this case, this means that this case is
				// selected.
				selected = append(selected, fieldType.Name)
			}
		}
	}

	for _, elemSchema := range schema.Dir {
		// If element is a choice, recurse down to the next named element.
		if elemSchema.IsChoice() {
			return validateChoice(elemSchema, value.(ygot.GoStruct))
		}
	}

	return
}
