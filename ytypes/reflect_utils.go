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

// TODO(mostrowski): move to more common package.

import (
	"fmt"
	"reflect"
)

// UpdateField updates a field called fieldName (which must already exist) in
// parentStruct, with value fieldValue, which must be a pointer to a struct.
func UpdateField(parentStruct interface{}, fieldName string, fieldValue interface{}) (interface{}, error) {
	v := reflect.ValueOf(parentStruct)
	if isNil(parentStruct) {
		return nil, fmt.Errorf("parentStruct is nil in UpdateField for field %s, value %v", fieldName, fieldValue)
	}
	t := reflect.TypeOf(parentStruct)
	if v.Kind() == reflect.Ptr {
		t = reflect.TypeOf(v.Elem().Interface())
	}
	if !isValueStructPtr(v) {
		return nil, fmt.Errorf("type for %s must be Ptr to Struct, is %s", t.Name(), v.Kind())
	}
	ft, ok := t.FieldByName(fieldName)
	if !ok {
		return nil, fmt.Errorf("no field named %s in struct %s", fieldName, t.Name())
	}
	// Allow fieldValue to be nil type if field to set is a pointer.
	if ft.Type.Kind() != reflect.ValueOf(fieldValue).Kind() && !(fieldValue == nil && ft.Type.Kind() == reflect.Ptr) {
		return nil, fmt.Errorf("cannot assign value %v (type %T) to field %s (type %v) in struct %s",
			fieldValue, fieldValue, fieldName, ft.Type.Kind(), t.Name())
	}

	n := reflect.New(ft.Type)
	if fieldValue != nil {
		n.Elem().Set(reflect.ValueOf(fieldValue))
	}
	fv := v.Elem().FieldByName(fieldName)
	fv.Set(n.Elem())

	return n.Elem().Interface(), nil
}

// FieldIteratorFunc is an iteration function for arbitrary field traversals.
//   fieldType and fieldValue are passed in with the struct field information
//     for every field traversed if it is part of a struct.
//   in, out are passed through from the caller to the iteration and can be used
//     arbitrarily in the iteration function to carry state and results.
//   fieldKeys is a slice of keys if the traversed element is in a map.
//   fieldKey is the key value of the element if it's part of a map.
// Returns a slice of errors encountered while processing the field.
type FieldIteratorFunc func(parentStruct interface{},
	fieldType reflect.StructField, fieldValue reflect.Value,
	fieldKeys []reflect.Value, fieldKey reflect.Value,
	in, out interface{}) []error

// ForEachField recursively iterates through the fields of value (which may be
// any Go type) and executes iterFunction on each field.
//   in, out are passed to the iterator function and can be used to carry state
//     and return results from the iterator.
//   iterFunction is executed on each scalar field.
// Returns a slice of errors encountered while processing the struct.
func ForEachField(value interface{}, in, out interface{}, iterFunction FieldIteratorFunc) (errs []error) {
	if isNil(value) {
		return nil
	}
	return forEachFieldInternal(nil, reflect.StructField{}, reflect.ValueOf(value), nil, reflect.ValueOf(nil), in, out, iterFunction)
}

// forEachFieldInternal recursively iterates through the fields of value (which
// may be any Go type) and executes iterFunction on each field.
//   parentStruct is a ptr to the containing struct (if any).
//   fieldType is struct field info if value is a struct or struct ptr type.
//   value is the value of the root element to traverse.
//   fieldKeys is a slice of keys if the traversed element is in a map.
//   fieldKey is the key value of the element if it's part of a map.
//   in, out are passed through from the caller to the iteration and can be used
//     arbitrarily in the iteration function to carry state and results.
func forEachFieldInternal(parentStruct interface{}, fieldType reflect.StructField, value reflect.Value, fieldKeys []reflect.Value, fieldKey reflect.Value, in, out interface{}, iterFunction FieldIteratorFunc) (errs []error) {
	if isNilOrInvalidValue(value) {
		return nil
	}

	errs = appendErrs(errs, iterFunction(parentStruct, fieldType, value, fieldKeys, fieldKey, in, out))

	switch {
	case isValueStruct(value) || isValueStructPtr(value):
		structElems := ptrToValue(value)
		for i := 0; i < structElems.NumField(); i++ {
			fieldValue := structElems.Field(i)
			errs = appendErrs(errs, forEachFieldInternal(value.Interface(), structElems.Type().Field(i), fieldValue, fieldKeys, fieldKey, in, out, iterFunction))
		}
	case isValueSlice(value):
		for i := 0; i < value.Len(); i++ {
			errs = appendErrs(errs, forEachFieldInternal(parentStruct, reflect.StructField{}, value.Index(i), fieldKeys, fieldKey, in, out, iterFunction))
		}
	case isValueMap(value):
		for _, key := range value.MapKeys() {
			cv := value.MapIndex(key)
			errs = appendErrs(errs, forEachFieldInternal(parentStruct, reflect.StructField{}, cv, value.MapKeys(), key, in, out, iterFunction))
		}
	}

	return nil
}

// ptrToValue returns the dereferenced reflect.Value of value if it is a ptr, or
// value if it is not.
func ptrToValue(value reflect.Value) reflect.Value {
	if isValueStructPtr(value) {
		return value.Elem()
	}
	return value
}

func isNilOrInvalidValue(v reflect.Value) bool {
	return !v.IsValid() || (v.Kind() == reflect.Ptr && v.IsNil())
}

func isValueStruct(v reflect.Value) bool {
	return v.Kind() == reflect.Struct
}

func isValueStructPtr(v reflect.Value) bool {
	return v.Kind() == reflect.Ptr && isValueStruct(v.Elem())
}

func isValueMap(v reflect.Value) bool {
	return v.Kind() == reflect.Map
}

func isValueSlice(v reflect.Value) bool {
	return v.Kind() == reflect.Slice
}

func isValueScalar(v reflect.Value) bool {
	return !isValueStruct(v) && !isValueStructPtr(v) && !isValueMap(v) && !isValueSlice(v)
}
