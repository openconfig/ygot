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
		ft := structElems.Type().Field(i)

		// If this is an annotation field, then skip it since it does not have
		// a schema.
		if util.IsYgotAnnotation(ft) {
			continue
		}

		fieldName := ft.Name
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
//   opts... are a set of ytypes.UnmarshalOptionst that are used to control
//     the behaviour of the unmarshal function.
func unmarshalList(schema *yang.Entry, parent interface{}, jsonList interface{}, enc Encoding, opts ...UnmarshalOpt) error {
	if util.IsValueNil(jsonList) {
		return nil
	}
	// Check that the schema itself is valid.
	if err := validateListSchema(schema); err != nil {
		return err
	}

	util.DbgPrint("unmarshalList jsonList %v, type %T, into parent type %T, schema name %s", util.ValueStrDebug(jsonList), jsonList, parent, schema.Name)

	// Parent must be a map, slice ptr, or struct ptr.
	t := reflect.TypeOf(parent)

	if util.IsTypeStructPtr(t) {
		// May be trying to unmarshal a single list element rather than the
		// whole list.
		return unmarshalContainerWithListSchema(schema, parent, jsonList, opts...)
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
		if err := unmarshalStruct(schema, newVal.Interface(), jt, enc, opts...); err != nil {
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

// makeValForInsert is used to create a value with the type extracted from
// given map. The returned value is populated according to the supplied "keys"
// map, which is assumed to be the map[string]string keys field from a gNMI
// PathElem protobuf message. Output of this function can be passed to
// makeKeyForInsert to produce a key to use while inserting into map. The
// function returns an error if a key name is not a valid schema tag in the
// supplied schema. Also, function uses the last schema tag if there is more
// than one by assuming it is direct descendant.
// - schema: schema of the map.
// - parent: value of the map.
// - keys: dictionary received as part of Key field of gNMI PathElem.
func makeValForInsert(schema *yang.Entry, parent interface{}, keys map[string]string) (reflect.Value, error) {
	rv, rt := reflect.ValueOf(parent), reflect.TypeOf(parent)
	if !util.IsValueMap(rv) {
		return reflect.ValueOf(nil), fmt.Errorf("%T is not a reflect.Map kind", parent)
	}
	// key is a non-pointer type
	keyT := rt.Key()
	// element is pointer type
	elmT := rt.Elem()

	if !util.IsTypeStructPtr(elmT) {
		return reflect.ValueOf(nil), fmt.Errorf("%v is not a pointer to a struct", elmT)
	}

	// Create an instance of map value type. Element is dereferenced as it is a pointer.
	val := reflect.New(elmT.Elem())
	// Helper to update the field corresponding to schema key.
	setKey := func(schemaKey string, fieldVal string) error {
		fn, err := schemaNameToFieldName(val.Elem(), schemaKey)
		if err != nil {
			return err
		}

		fv := val.Elem().FieldByName(fn)
		ft := fv.Type()
		if util.IsValuePtr(fv) {
			ft = ft.Elem()
		}

		nv, err := StringToType(ft, fieldVal)
		if err != nil {
			return err
		}
		return util.InsertIntoStruct(val.Interface(), fn, nv.Interface())
	}

	if util.IsTypeStruct(keyT) {
		for i := 0; i < keyT.NumField(); i++ {
			schKey, err := directDescendantSchema(keyT.Field(i))
			if err != nil {
				return reflect.ValueOf(nil), err
			}
			schVal, ok := keys[schKey]
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("missing %v key in %v", schKey, keys)
			}
			if err := setKey(schKey, schVal); err != nil {
				return reflect.ValueOf(nil), err
			}
		}
		return val, nil
	}
	v, ok := keys[schema.Key]
	if !ok {
		return reflect.ValueOf(nil), fmt.Errorf("missing %v key in %v", schema.Key, keys)
	}
	if err := setKey(schema.Key, v); err != nil {
		return reflect.ValueOf(nil), err
	}
	return val, nil
}

// makeKeyForInsert returns a key for inserting a struct newVal into the parent,
// which must be a map.
func makeKeyForInsert(schema *yang.Entry, parentMap interface{}, newVal reflect.Value) (reflect.Value, error) {
	// Key is always a value type, never a ptr.
	listKeyType := reflect.TypeOf(parentMap).Key()
	newKey := reflect.New(listKeyType).Elem()

	if util.IsTypeStruct(listKeyType) {
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
			if !nv.IsValid() {
				return reflect.ValueOf(nil), fmt.Errorf("%v field doesn't have a valid value", kfn)
			}
			util.DbgPrint("Setting value of %v (%T) in key struct (%T)", nv.Interface(), nv.Interface(), newKey.Interface())
			newKeyField := newKey.FieldByName(kfn)
			if !util.ValuesAreSameType(newKeyField, nv) {
				return reflect.ValueOf(nil), fmt.Errorf("multi-key %v is not assignable to %v", nv.Type(), newKeyField.Type())
			}
			newKeyField.Set(nv)
		}

		return newKey, nil
	}

	// Simple key type. Get the value from the new value struct,
	// given the key string.
	kv, err := getKeyValue(newVal.Elem(), schema.Key)
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	util.DbgPrint("key value is %v.", kv)

	rvKey := reflect.ValueOf(kv)

	switch {
	case util.IsTypeInterface(listKeyType) && util.IsValueTypeCompatible(listKeyType, newKey), util.ValuesAreSameType(newKey, rvKey):
	default:
		return reflect.ValueOf(nil), fmt.Errorf("single-key %v is not assignable to %v", rvKey.Type(), newKey.Type())
	}
	newKey.Set(rvKey)

	return newKey, nil
}

// insertAndGetKey creates key and value from the supplied keys map. It inserts
// key and value into the given root which must be a map with the supplied schema.
func insertAndGetKey(schema *yang.Entry, root interface{}, keys map[string]string) (interface{}, error) {
	switch {
	case schema.Key == "":
		return nil, fmt.Errorf("unkeyed list can't be traversed, type %T, keys %v", root, keys)
	case !util.IsValueMap(reflect.ValueOf(root)):
		return nil, fmt.Errorf("root has type %T, want map", root)
	}

	// TODO(yusufsn): When the key is a leafref, its target should be filled out.
	mapVal, err := makeValForInsert(schema, root, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to create map value for insert, root %T, keys %v: %v", root, keys, err)
	}
	mapKey, err := makeKeyForInsert(schema, root, mapVal)
	if err != nil {
		return nil, fmt.Errorf("failed to create map key for insert, root %T, keys %v: %v", root, keys, err)
	}
	err = util.InsertIntoMap(root, mapKey.Interface(), mapVal.Interface())
	if err != nil {
		return nil, fmt.Errorf("failed to insert into map %T, keys %v: %v", root, keys, err)
	}

	return mapKey.Interface(), nil
}

// unmarshalContainerWithListSchema unmarshals a container data tree element
// using a list schema. This can happen because in OC schemas, list elements
// share the list schema so if a user attempts to unmarshal a list element vs.
// the whole list, the supplied schema is the same - the only difference is
// that in the latter case the target is a struct ptr. The supplied opts control
// the behaviour of the unmarshal function.
func unmarshalContainerWithListSchema(schema *yang.Entry, parent interface{}, value interface{}, opts ...UnmarshalOpt) error {

	if !util.IsTypeStructPtr(reflect.TypeOf(parent)) {
		return fmt.Errorf("unmarshalContainerWithListSchema value %v, type %T, into parent type %T, schema name %s: parent must be a struct ptr",
			value, value, parent, schema.Name)
	}
	// Create a container equivalent of the list, which is just the list
	// with ListAttrs unset.
	newSchema := *schema
	newSchema.ListAttr = nil
	return Unmarshal(&newSchema, parent, value, opts...)
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
