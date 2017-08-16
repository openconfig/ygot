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
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-7.8.

// validateList validates each of the values in the map, keyed by the list Key
// value, against the given list schema.
func validateList(schema *yang.Entry, value interface{}) (errors []error) {
	if isNil(value) {
		return nil
	}

	// Check that the schema itself is valid.
	if err := validateListSchema(schema); err != nil {
		return appendErr(errors, err)
	}

	dbgPrint("validateList with value %v, type %T, schema name %s", value, value, schema.Name)

	kind := reflect.TypeOf(value).Kind()
	if kind == reflect.Slice || kind == reflect.Map {
		// Check list attributes: size constraints etc.
		// Skip this check if not a list type - in this case value may be a list
		// element which shares the list schema (excluding ListAttr).
		errors = appendErrs(errors, validateListAttr(schema, value))
	}

	switch kind {
	case reflect.Slice:
		// List without key is a slice in the data tree.
		sv := reflect.ValueOf(value)
		for i := 0; i < sv.Len(); i++ {
			errors = appendErrs(errors, validateStructElems(schema, sv.Index(i).Interface()))
		}
	case reflect.Map:
		// List with key is a map in the data tree, with the key being the value
		// of the key field(s) in the elements.
		for _, key := range reflect.ValueOf(value).MapKeys() {
			cv := reflect.ValueOf(value).MapIndex(key).Interface()
			structElems := reflect.ValueOf(cv).Elem()
			// Check that keys are present and have correct values.
			errors = appendErrs(errors, checkKeys(schema, structElems, key))

			// Verify each elements's fields.
			errors = appendErrs(errors, validateStructElems(schema, cv))
		}
	case reflect.Ptr:
		// Validate was called on a list element rather than the whole list, or
		// on a completely bogus struct. In either case, evaluate just the
		// element against the list schema without considering list attributes.
		errors = appendErrs(errors, validateStructElems(schema, value))

	default:
		errors = appendErr(errors, fmt.Errorf("validateList expected map/slice type for %s, got %T", schema.Name, value))
	}
	return
}

// checkKeys checks that the map key value for the list equals the value of the
// key field(s) in the elements for the map value.
//   entry is the schema for the list.
//   structElems is the structure representing the element in the data tree.
//   keyElems is the structure representing the map key in the data tree.
// For a list schema that has a struct key, it's expected that:
//    1. The schema contains leaves with the struct field names (checked before
//       calling this function).
//    2. Each element in the list has key fields defined by the leaves in 1.
//    3. For each such key field, the field value in the element equals the
//       value of the map key of the containing map in the data tree.
func checkKeys(schema *yang.Entry, structElems reflect.Value, keyValue reflect.Value) (errors []error) {
	keys := strings.Split(schema.Key, " ")
	if len(keys) == 1 {
		errors = appendErrs(errors, checkBasicKeyValue(structElems, schema.Key, keyValue))
	} else {
		errors = appendErrs(errors, checkStructKeyValues(structElems, keyValue))
	}
	return
}

// checkBasicKeyValue checks if keyValue, which is the value of the map key,
// is equal to the value of the key field with field name keyFieldName in the
// element struct.
func checkBasicKeyValue(structElems reflect.Value, keyFieldSchemaName string, keyValue reflect.Value) (errors []error) {
	// Find field name corresponding to keyFieldName in the schema.
	keyFieldName, err := schemaNameToFieldName(structElems, keyFieldSchemaName)
	if err != nil {
		return appendErr(errors, err)
	}
	if isNil(keyValue.Interface()) {
		return nil
	}

	if !structElems.FieldByName(keyFieldName).IsValid() {
		return []error{fmt.Errorf("missing key field %s in element %v", keyFieldName, structElems)}
	}
	var elementKeyValue interface{}
	if structElems.FieldByName(keyFieldName).Kind() == reflect.Ptr && !structElems.FieldByName(keyFieldName).IsNil() {
		elementKeyValue = structElems.FieldByName(keyFieldName).Elem().Interface()

	} else {
		elementKeyValue = structElems.FieldByName(keyFieldName).Interface()
	}
	if elementKeyValue != keyValue.Interface() {
		errors = appendErr(errors, fmt.Errorf("key value for field %s in list member (%v) is not equal to the key used in the map (%v)",
			keyFieldName, elementKeyValue, keyValue))
	}

	return
}

// checkStructKeyValues checks that the provided key struct (which is the key
// value of the entry in the data tree map):
//  - has all the fields defined in the schema key definition
//  - has no fields not defined in the schema key definition
//  - has values for each field equal to the corresponding field in the element.
func checkStructKeyValues(structElems reflect.Value, keyStruct reflect.Value) (errors []error) {
	//dbgPrint("checkStructKeyValues structElems=%v, keyStruct=%v", valueStr(structElems.Interface()), keyStruct)
	switch keyStruct.Type().Kind() {
	case reflect.Struct:
		for i := 0; i < keyStruct.NumField(); i++ {
			keyName := keyStruct.Type().Field(i).Name
			keyValue := keyStruct.Field(i).Interface()
			if !structElems.FieldByName(keyName).IsValid() {
				errors = appendErr(errors, fmt.Errorf("missing key field %s in %v", keyName, keyStruct))
				continue
			}

			var elementStructKeyValue interface{}
			if structElems.FieldByName(keyName).Kind() == reflect.Ptr && !structElems.FieldByName(keyName).IsNil() {
				elementStructKeyValue = structElems.FieldByName(keyName).Elem().Interface()

			} else {
				elementStructKeyValue = structElems.FieldByName(keyName).Interface()
			}
			if elementStructKeyValue != keyValue {
				errors = appendErr(errors, fmt.Errorf("element key value %v for key field %s has different value from map key %v",
					elementStructKeyValue, keyName, keyValue))
			}
		}

	default:
		errors = appendErr(errors, fmt.Errorf("key value %v is not struct type", keyStruct))
	}

	return
}

// validateStructElems validates each of the struct fields against the schema.
// TODO(mostrowski): choice directly under list is not handled here.
// Also, there's code duplication with a very similar operation in container.
func validateStructElems(schema *yang.Entry, value interface{}) (errors []error) {
	structElems := reflect.ValueOf(value).Elem()
	structTypes := structElems.Type()

	if structElems.Kind() != reflect.Struct {
		return appendErr(errors, fmt.Errorf("expected a struct type for %s: got %s", schema.Name, valueStr(value)))
	}
	// Verify each elements's fields.
	for i := 0; i < structElems.NumField(); i++ {
		fieldName := structElems.Type().Field(i).Name
		fieldValue := structElems.Field(i).Interface()

		cschema, err := childSchema(schema, structTypes.Field(i))
		if err != nil {
			errors = appendErr(errors, err)
			continue
		}
		if cschema == nil {
			errors = appendErr(errors, fmt.Errorf("child schema not found for struct %s field %s", schema.Name, fieldName))
		} else {
			errors = appendErrs(errors, Validate(cschema, fieldValue))
		}
	}
	return
}

// validateListSchema validates the given list type schema. This is a sanity
// check validation rather than a comprehensive validation against the RFC.
// It is assumed that such a validation is done when the schema is parsed from
// source YANG.
func validateListSchema(schema *yang.Entry) error {
	if schema == nil {
		return fmt.Errorf("list schema is nil")
	}
	if !schema.IsList() {
		return fmt.Errorf("schema %s is not list type", schema.Name)
	}
	if schema.IsList() && schema.Config.Value() {
		if len(schema.Key) == 0 {
			return fmt.Errorf("list %s with config set must have a key", schema.Name)
		}
		keys := strings.Split(schema.Key, " ")
		keysMissing := make(map[string]bool)
		for _, v := range keys {
			keysMissing[v] = true
		}
		for _, v := range schema.Dir {
			if _, ok := keysMissing[v.Name]; ok {
				delete(keysMissing, v.Name)
			}
		}
		if len(keysMissing) != 0 {
			return fmt.Errorf("list %s has keys %v missing from required list of %v", schema.Name, keysMissing, keys)
		}
	}

	return nil
}

// schemaNameToFieldName returns the name of the struct field that corresponds
// to the name in the schema Key field, given structElems which is the stuct
// containing the field. Returns error if no field is found for the supplied
// key field name.
func schemaNameToFieldName(structElems reflect.Value, schemaKeyFieldName string) (string, error) {
	for i := 0; i < structElems.NumField(); i++ {
		ps, err := pathToSchema(structElems.Type().Field(i))
		if err != nil {
			return "", err
		}
		matches, err := nameMatchesPath(schemaKeyFieldName, ps)
		if err != nil {
			return "", err
		}
		if matches {
			return structElems.Type().Field(i).Name, nil
		}
	}

	return "", fmt.Errorf("struct %v does not contain a field with tag %s", structElems, schemaKeyFieldName)
}

// nameMatchesPath returns true if the supplied path matches the given field
// name in the schema.
// For MyStructFieldName, the path is expected to follow the pattern of either
// {"my-struct-field-name"} or {"my-struct-name", "my-struct-field-name"}
func nameMatchesPath(fieldName string, path []string) (bool, error) {
	switch len(path) {
	case 1:
		return fieldName == path[0], nil
	case 2:
		return fieldName == path[1], nil
	default:
	}
	return false, fmt.Errorf("expected field %s path to have one or two elements, got %v", fieldName, path)
}
