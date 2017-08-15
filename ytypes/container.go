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

	"github.com/openconfig/ygot/ygot"
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-7.5.

// validateContainer validates each of the values in the map, keyed by the list
// Key value, against the given list schema.
func validateContainer(schema *yang.Entry, value ygot.GoStruct) (errors []error) {
	if isNil(value) {
		return nil
	}
	// Check that the schema itself is valid.
	if err := validateContainerSchema(schema); err != nil {
		return appendErr(errors, err)
	}
	dbgPrint("validateContainer with value %v, type %T, schema name %s", valueStr(value), value, schema.Name)

	extraFields := make(map[string]bool)

	switch reflect.TypeOf(value).Kind() {
	case reflect.Ptr:
		// Field exists in a struct but is unset.
		if reflect.ValueOf(value).IsNil() {
			return nil
		}
		structElems := reflect.ValueOf(value).Elem()
		structTypes := structElems.Type()

		for i := 0; i < structElems.NumField(); i++ {
			fieldName := structElems.Type().Field(i).Name
			fieldValue := structElems.Field(i).Interface()

			cschema, err := childSchema(schema, structTypes.Field(i))
			switch {
			case err != nil:
				errors = appendErr(errors, fmt.Errorf("%s: %v", fieldName, err))
				continue
			case cschema != nil:
				// Regular named child.
				if errs := Validate(cschema, fieldValue); len(errs) != 0 {
					errors = appendErrs(appendErr(errors, fmt.Errorf("%s/", fieldName)), errs)
				}
			case !structElems.Field(i).IsNil():
				// Either an element in choice schema subtree, or bad field.
				// If the former, it will be found in the choice check below.
				extraFields[fieldName] = true
			}
		}

		// Field names in the data tree belonging to Choice have the schema of
		// the elements of that choice. Hence, choice schemas must be checked
		// separately.
		for _, choiceSchema := range schema.Dir {
			if choiceSchema.IsChoice() {
				selected, errs := validateChoice(choiceSchema, value)
				for _, s := range selected {
					delete(extraFields, s)
				}
				if errs != nil {
					errors = appendErrs(appendErr(errors, fmt.Errorf("%s/", choiceSchema.Name)), errs)
				}
			}
		}

	default:
		errors = appendErr(errors, fmt.Errorf("validateContainer expected struct type for %s (type %T), got %v", schema.Name, value, reflect.TypeOf(value).Kind()))
	}

	if len(extraFields) > 0 {
		errors = appendErr(errors, fmt.Errorf("fields %v are not found in the container schema %s", mapToStrSlice(extraFields), schema.Name))
	}

	return
}

// validateContainerSchema validates the given container type schema. This is a
// sanity check validation rather than a comprehensive validation against the
// RFC. It is assumed that such a validation is done when the schema is parsed
// from source YANG.
func validateContainerSchema(schema *yang.Entry) error {
	if schema == nil {
		return fmt.Errorf("container schema is nil")
	}
	if !schema.IsContainer() {
		return fmt.Errorf("container schema %s is not a container type", schema.Name)
	}

	return nil
}
