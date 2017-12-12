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

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-7.5.

// validateContainer validates each of the values in the map, keyed by the list
// Key value, against the given list schema.
func validateContainer(schema *yang.Entry, value ygot.GoStruct) util.Errors {
	var errors []error
	if util.IsValueNil(value) {
		return nil
	}
	// Check that the schema itself is valid.
	if err := validateContainerSchema(schema); err != nil {
		return util.NewErrs(err)
	}

	util.DbgPrint("validateContainer with value %v, type %T, schema name %s", util.ValueStr(value), value, schema.Name)

	extraFields := make(map[string]interface{})

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
				errors = util.AppendErr(errors, fmt.Errorf("%s: %v", fieldName, err))
				continue
			case cschema != nil:
				// Regular named child.
				if errs := Validate(cschema, fieldValue); errs != nil {
					errors = util.AppendErrs(util.AppendErr(errors, fmt.Errorf("%s/", fieldName)), errs)
				}
			case !structElems.Field(i).IsNil():
				// Either an element in choice schema subtree, or bad field.
				// If the former, it will be found in the choice check below.
				extraFields[fieldName] = nil
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
					errors = util.AppendErrs(util.AppendErr(errors, fmt.Errorf("%s/", choiceSchema.Name)), errs)
				}
			}
		}

	default:
		errors = util.AppendErr(errors, fmt.Errorf("validateContainer expected struct type for %s (type %T), got %v", schema.Name, value, reflect.TypeOf(value).Kind()))
	}

	if len(extraFields) > 0 {
		errors = util.AppendErr(errors, fmt.Errorf("fields %v are not found in the container schema %s", stringMapSetToSlice(extraFields), schema.Name))
	}

	return errors
}

// unmarshalContainer unmarshals a JSON tree into a struct.
//   schema is the schema of the schema node corresponding to the struct being
//     unmamshaled into.
//   parent is the parent struct, which must be a struct ptr.
//   jsonTree is a JSON data tree which must be a map[string]interface{}.
func unmarshalContainer(schema *yang.Entry, parent interface{}, jsonTree interface{}) error {
	if util.IsValueNil(jsonTree) {
		return nil
	}

	// Check that the schema itself is valid.
	if err := validateContainerSchema(schema); err != nil {
		return err
	}

	util.DbgPrint("unmarshalContainer jsonTree %v, type %T, into parent type %T, schema name %s", util.ValueStr(jsonTree), jsonTree, parent, schema.Name)

	// Since this is a container, the JSON data tree is a map.
	jt, ok := jsonTree.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unmarshalContainer for schema %s: jsonTree %v: got type %T inside container, expect map[string]interface{}",
			schema.Name, util.ValueStr(jsonTree), jsonTree)
	}

	pvp := reflect.ValueOf(parent)
	if !util.IsValueStructPtr(pvp) {
		return fmt.Errorf("unmarshalContainer got parent type %T, expect struct ptr", parent)
	}

	return unmarshalStruct(schema, parent, jt)
}

// unmarshalStruct unmarshals a JSON tree into a struct.
//   schema is the YANG schema of the node corresponding to the struct being
//     unmarshalled into.
//   parent is the parent struct, which must be a struct ptr.
//   jsonTree is a JSON data tree which must be a map[string]interface{}.
func unmarshalStruct(schema *yang.Entry, parent interface{}, jsonTree map[string]interface{}) error {
	destv := reflect.ValueOf(parent).Elem()
	var allSchemaPaths [][]string
	// Range over the parent struct fields. For each field, check if the data
	// is present in the JSON tree and if so unmarshal it into the field.
	for i := 0; i < destv.NumField(); i++ {
		f := destv.Field(i)
		ft := destv.Type().Field(i)
		cschema, err := childSchema(schema, ft)
		if err != nil {
			return err
		}
		if cschema == nil {
			return fmt.Errorf("unmarshalContainer could not find schema for type %T, field name %s", parent, ft.Name)
		}
		jsonValue, err := getJSONTreeValForField(schema, cschema, ft, jsonTree)
		if err != nil {
			return err
		}
		// Store the data tree path of the current field. These will be used
		// at the end to ensure that there are no excess elements in the JSON
		// tree not covered by any data path.
		sp, err := dataTreePaths(schema, cschema, ft)
		if err != nil {
			return err
		}

		allSchemaPaths = append(allSchemaPaths, sp...)
		if jsonValue == nil {
			util.DbgPrint("field %s paths %v not present in tree", ft.Name, sp)
			continue
		}

		util.DbgPrint("populating field %s type %s with paths %v.", ft.Name, ft.Type, sp)
		// Only create a new field if it is nil, otherwise update just the
		// fields that are in the data tree being passed to unmarshal, and
		// preserve all other existing values.
		if util.IsNilOrInvalidValue(f) {
			makeField(destv, ft)
		}

		p := parent
		switch {
		case util.IsUnkeyedList(cschema):
			// For unkeyed list, we must pass in the addr of the slice to be
			// able to append to it.
			p = f.Addr().Interface()
		case cschema.IsContainer() || cschema.IsList():
			// For list and container, the new parent is the field we just
			// created. For leaf and leaf-list, the parent is still the
			// current container.
			p = f.Interface()
		}
		if err := Unmarshal(cschema, p, jsonValue); err != nil {
			return err
		}
	}

	// Go over all JSON fields to make sure that each one is covered
	// by a data path in the struct.
	if err := checkDataTreeAgainstPaths(jsonTree, allSchemaPaths); err != nil {
		return fmt.Errorf("parent container %s (type %T): %s", schema.Name, parent, err)
	}

	util.DbgPrint("container after unmarshal:\n%s\n", pretty.Sprint(destv.Interface()))
	return nil
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
