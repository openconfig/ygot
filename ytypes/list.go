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

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-7.8.

// validateList validates each of the values in the map, keyed by the list Key
// value, against the given list schema.
func validateList(schema *yang.Entry, value interface{}) util.Errors {
	var errors []error
	if util.IsValueNil(value) {
		return nil
	}

	// Check that the schema itself is valid.
	if err := validateListSchema(schema); err != nil {
		return util.NewErrs(err)
	}

	util.DbgPrint("validateList with value %v, type %T, schema name %s", value, value, schema.Name)

	kind := reflect.TypeOf(value).Kind()
	if kind == reflect.Slice || kind == reflect.Map {
		// Check list attributes: size constraints etc.
		// Skip this check if not a list type - in this case value may be a list
		// element which shares the list schema (excluding ListAttr).
		errors = util.AppendErrs(errors, validateListAttr(schema, value))
	}

	switch kind {
	case reflect.Slice:
		// List without key is a slice in the data tree.
		sv := reflect.ValueOf(value)
		for i := 0; i < sv.Len(); i++ {
			errors = util.AppendErrs(errors, validateStructElems(schema, sv.Index(i).Interface()))
		}
	case reflect.Map:
		// List with key is a map in the data tree, with the key being the value
		// of the key field(s) in the elements.
		for _, key := range reflect.ValueOf(value).MapKeys() {
			cv := reflect.ValueOf(value).MapIndex(key).Interface()
			structElems := reflect.ValueOf(cv).Elem()
			// Check that keys are present and have correct values.
			errors = util.AppendErrs(errors, checkKeys(schema, structElems, key))

			// Verify each elements's fields.
			errors = util.AppendErrs(errors, validateStructElems(schema, cv))
		}
	case reflect.Ptr:
		// Validate was called on a list element rather than the whole list, or
		// on a completely bogus struct. In either case, evaluate just the
		// element against the list schema without considering list attributes.
		errors = util.AppendErrs(errors, validateStructElems(schema, value))

	default:
		errors = util.AppendErr(errors, fmt.Errorf("validateList expected map/slice type for %s, got %T", schema.Name, value))
	}

	return errors
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
func checkKeys(schema *yang.Entry, structElems reflect.Value, keyValue reflect.Value) util.Errors {
	keys := strings.Split(schema.Key, " ")
	if len(keys) == 1 {
		return checkBasicKeyValue(structElems, schema.Key, keyValue)
	}

	return checkStructKeyValues(structElems, keyValue)
}

// checkBasicKeyValue checks if keyValue, which is the value of the map key,
// is equal to the value of the key field with field name keyFieldName in the
// element struct.
func checkBasicKeyValue(structElems reflect.Value, keyFieldSchemaName string, keyValue reflect.Value) util.Errors {
	// Find field name corresponding to keyFieldName in the schema.
	keyFieldName, err := schemaNameToFieldName(structElems, keyFieldSchemaName)
	if err != nil {
		return util.NewErrs(err)
	}
	if util.IsValueNil(keyValue.Interface()) {
		return nil
	}

	if !structElems.FieldByName(keyFieldName).IsValid() {
		return util.NewErrs(fmt.Errorf("missing key field %s in element %v", keyFieldName, structElems))
	}
	var elementKeyValue interface{}
	if structElems.FieldByName(keyFieldName).Kind() == reflect.Ptr && !structElems.FieldByName(keyFieldName).IsNil() {
		elementKeyValue = structElems.FieldByName(keyFieldName).Elem().Interface()

	} else {
		elementKeyValue = structElems.FieldByName(keyFieldName).Interface()
	}
	if elementKeyValue != keyValue.Interface() {
		return util.NewErrs(fmt.Errorf("key field %s: element key %v != map key %v", keyFieldName, elementKeyValue, keyValue))
	}

	return nil
}

// checkStructKeyValues checks that the provided key struct (which is the key
// value of the entry in the data tree map):
//  - has all the fields defined in the schema key definition
//  - has no fields not defined in the schema key definition
//  - has values for each field equal to the corresponding field in the element.
func checkStructKeyValues(structElems reflect.Value, keyStruct reflect.Value) util.Errors {
	var errors []error
	if keyStruct.Type().Kind() != reflect.Struct {
		return util.NewErrs(fmt.Errorf("key value %v is not struct type", keyStruct))
	}
	for i := 0; i < keyStruct.NumField(); i++ {
		keyName := keyStruct.Type().Field(i).Name
		keyValue := keyStruct.Field(i).Interface()
		if !structElems.FieldByName(keyName).IsValid() {
			errors = util.AppendErr(errors, fmt.Errorf("missing key field %s in %v", keyName, keyStruct))
			continue
		}

		elementStructKeyValue := structElems.FieldByName(keyName)
		if structElems.FieldByName(keyName).Kind() == reflect.Ptr && !structElems.FieldByName(keyName).IsNil() {
			elementStructKeyValue = elementStructKeyValue.Elem()
		}

		if elementStructKeyValue.Interface() != keyValue {
			errors = util.AppendErr(errors, fmt.Errorf("element key value %v for key field %s has different value from map key %v",
				elementStructKeyValue, keyName, keyValue))
		}
	}

	return errors
}

// validateStructElems validates each of the struct fields against the schema.
// TODO(mostrowski): choice directly under list is not handled here.
// Also, there's code duplication with a very similar operation in container.
func validateStructElems(schema *yang.Entry, value interface{}) util.Errors {
	var errors []error
	structElems := reflect.ValueOf(value).Elem()
	structTypes := structElems.Type()

	if structElems.Kind() != reflect.Struct {
		return util.NewErrs(fmt.Errorf("expected a struct type for %s: got %s", schema.Name, util.ValueStr(value)))
	}
	// Verify each elements's fields.
	for i := 0; i < structElems.NumField(); i++ {
		fieldName := structElems.Type().Field(i).Name
		fieldValue := structElems.Field(i).Interface()

		cschema, err := childSchema(schema, structTypes.Field(i))
		if err != nil {
			errors = util.AppendErr(errors, err)
			continue
		}
		if cschema == nil {
			errors = util.AppendErr(errors, fmt.Errorf("child schema not found for struct %s field %s", schema.Name, fieldName))
		} else {
			errors = util.AppendErrs(errors, Validate(cschema, fieldValue))
		}
	}

	return errors
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
// containing the field. It returns error if no field is found for the supplied
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
	}
	return false, fmt.Errorf("expected field %s path to have one or two elements, got %v", fieldName, path)
}

// unmarshalList unmarshals a JSON array into a list parent, which must be a
// map or slice ptr.
//   schema is the schema of the schema node corresponding to the struct being
//     unmamshaled into
//   jsonList is a JSON list
func unmarshalList(schema *yang.Entry, parent interface{}, jsonList interface{}) error {
	if util.IsValueNil(jsonList) {
		return nil
	}
	// Check that the schema itself is valid.
	if err := validateListSchema(schema); err != nil {
		return err
	}

	util.DbgPrint("unmarshalList jsonList %v, type %T, into parent type %T, schema name %s", util.ValueStr(jsonList), jsonList, parent, schema.Name)

	// Parent must be a map, slice ptr, or struct ptr.
	t := reflect.TypeOf(parent)

	if util.IsTypeStructPtr(t) {
		// May be trying to unmarshal a single list element rather than the
		// whole list.
		return unmarshalContainerWithListSchema(schema, parent, jsonList)
	}

	// jsonList represents a JSON array, which is a Go slice.
	jl, ok := jsonList.([]interface{})
	if !ok {
		return fmt.Errorf("unmarshalList for schema %s: jsonList %v: got type %T, expect []interface{}",
			schema.Name, util.ValueStr(jsonList), jsonList)
	}

	if !(util.IsTypeMap(t) || util.IsTypeSlicePtr(t)) {
		return fmt.Errorf("unmarshalList for %s got parent type %s, expect map, slice ptr or struct ptr", schema.Name, t.Kind())
	}

	listElementType := t.Elem()
	if util.IsTypeSlicePtr(t) {
		listElementType = t.Elem().Elem()
	}
	if !util.IsTypeStructPtr(listElementType) {
		return fmt.Errorf("unmarshalList for %s parent type %T, has bad field type %v", listElementType, parent, listElementType)
	}

	// Iterate over JSON list. Each JSON list element is a map with the field
	// name as the key. The JSON values must be unmarshaled and inserted into
	// the new struct list element. When all fields of the new element have been
	// filled, the constructed object will be added to listFieldName field in
	// the parent struct, which can be a map or a slice, for keyed/unkeyed list
	// types respectively.
	// For a keyed list, the value(s) of the key are derived from the key fields
	// in the new list element.
	for _, le := range jl {
		var err error
		jt := le.(map[string]interface{})
		newVal := reflect.New(listElementType.Elem())
		util.DbgPrint("creating a new list element val of type %v", newVal.Type())
		if err := unmarshalStruct(schema, newVal.Interface(), jt); err != nil {
			return err
		}

		switch {
		case util.IsTypeMap(t):
			newKey, err := makeKeyForInsert(schema, parent, newVal)
			if err != nil {
				return err
			}
			err = util.InsertIntoMap(parent, newKey.Interface(), newVal.Interface())
		case util.IsTypeSlicePtr(t):
			err = util.InsertIntoSlice(parent, newVal.Interface())
		default:
			return fmt.Errorf("unexpected type %s inserting in unmarshalList for parent type %T", t, parent)
		}
		if err != nil {
			return err
		}
	}
	util.DbgPrint("list after unmarshal:\n%s\n", pretty.Sprint(parent))

	return nil
}

// makeKeyForInsert returns a key for inserting a struct newVal into the parent,
// which must be a map.
func makeKeyForInsert(schema *yang.Entry, parentMap interface{}, newVal reflect.Value) (reflect.Value, error) {
	// Key is always a value type, never a ptr.
	listKeyType := reflect.TypeOf(parentMap).Key()
	newKey := reflect.New(listKeyType).Elem()

	if listKeyType.Kind() == reflect.Struct {
		// For struct key type, copy the key fields from the new list entry
		// struct newVal into the key struct.
		for i := 0; i < newKey.NumField(); i++ {
			kfn := listKeyType.Field(i).Name
			fv := newVal.Elem().FieldByName(kfn)
			if !fv.IsValid() {
				return reflect.ValueOf(nil), fmt.Errorf("element struct type %s does not contain key field %s", newVal.Elem().Type(), kfn)
			}
			nv := fv
			if fv.Type().Kind() == reflect.Ptr {
				// Ptr values are deferenced in key struct.
				nv = nv.Elem()
			}
			util.DbgPrint("Setting value of %v (%T) in key struct (%T)", nv.Interface(), nv.Interface(), newKey.Interface())
			newKey.FieldByName(kfn).Set(nv)
		}
	} else {
		// Simple key type. Get the value from the new value struct,
		// given the key string.
		kv, err := getKeyValue(newVal.Elem(), schema.Key)
		if err != nil {
			return reflect.ValueOf(nil), err
		}
		util.DbgPrint("key value is %v.", kv)
		newKey.Set(reflect.ValueOf(kv))
	}

	return newKey, nil
}

// unmarshalContainerWithListSchema unmarshals a container data tree element
// using a list schema. This can happen because in OC schemas, list elements
// share the list schema so if a user attempts to unmarshal a list element vs.
// the whole list, the supplied schema is the same - the only difference is
// that in the latter case the target is a struct ptr.
func unmarshalContainerWithListSchema(schema *yang.Entry, parent interface{}, value interface{}) error {

	if !util.IsTypeStructPtr(reflect.TypeOf(parent)) {
		return fmt.Errorf("unmarshalContainerWithListSchema value %v, type %T, into parent type %T, schema name %s: parent must be a struct ptr",
			value, value, parent, schema.Name)
	}
	// Create a container equivalent of the list, which is just the list
	// with ListAttrs unset.
	newSchema := *schema
	newSchema.ListAttr = nil
	return Unmarshal(&newSchema, parent, value)
}

// getKeyValue returns the value from the structVal field whose last path
// element is key. The value is dereferenced if it is a ptr type. This function
// is used to create a key value for a keyed list.
// getKeyValue returns an error if no path in any of the fields of structVal has
// key as the last path element.
func getKeyValue(structVal reflect.Value, key string) (interface{}, error) {
	for i := 0; i < structVal.NumField(); i++ {
		f := structVal.Type().Field(i)
		p, err := pathToSchema(f)
		if err != nil {
			return nil, err
		}
		if p[len(p)-1] == key {
			fv := structVal.Field(i)
			if fv.Type().Kind() == reflect.Ptr {
				// The type for the key is the dereferenced type, if the type
				// is a ptr.
				if !fv.Elem().IsValid() {
					return nil, fmt.Errorf("key field %s (%s) has nil value %v", key, fv.Type(), fv)
				}
				return fv.Elem().Interface(), nil
			}
			return fv.Interface(), nil
		}
	}

	return nil, fmt.Errorf("could not find key field %s in struct type %s", key, structVal.Type())
}
