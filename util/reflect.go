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
	"reflect"

	"github.com/kylelemons/godebug/pretty"
)

// IsTypeStructPtr reports whether v is a struct ptr type.
func IsTypeStructPtr(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct
}

// IsTypeSlicePtr reports whether v is a slice ptr type.
func IsTypeSlicePtr(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Slice
}

// IsTypeMap reports whether v is a map type.
func IsTypeMap(t reflect.Type) bool {
	return t.Kind() == reflect.Map
}

// IsTypeInterface reports whether v is an interface.
func IsTypeInterface(t reflect.Type) bool {
	return t.Kind() == reflect.Interface
}

// IsNilOrInvalidValue reports whether v is nil or reflect.Zero.
func IsNilOrInvalidValue(v reflect.Value) bool {
	return !v.IsValid() || (v.Kind() == reflect.Ptr && v.IsNil()) || IsValueNil(v.Interface())
}

// IsValuePtr reports whether v is a ptr type.
func IsValuePtr(v reflect.Value) bool {
	return v.Kind() == reflect.Ptr
}

// IsValueInterface reports whether v is an interface type.
func IsValueInterface(v reflect.Value) bool {
	return v.Kind() == reflect.Interface
}

// IsValueStruct reports whether v is a struct type.
func IsValueStruct(v reflect.Value) bool {
	return v.Kind() == reflect.Struct
}

// IsValueStructPtr reports whether v is a struct ptr type.
func IsValueStructPtr(v reflect.Value) bool {
	return v.Kind() == reflect.Ptr && IsValueStruct(v.Elem())
}

// IsValueMap reports whether v is a map type.
func IsValueMap(v reflect.Value) bool {
	return v.Kind() == reflect.Map
}

// IsValueSlice reports whether v is a slice type.
func IsValueSlice(v reflect.Value) bool {
	return v.Kind() == reflect.Slice
}

// IsValueScalar reports whether v is a scalar type.
func IsValueScalar(v reflect.Value) bool {
	return !IsValueStruct(v) && !IsValueStructPtr(v) && !IsValueMap(v) && !IsValueSlice(v)
}

// PtrToValue returns the dereferenced reflect.Value of value if it is a ptr, or
// value if it is not.
func PtrToValue(value reflect.Value) reflect.Value {
	if IsValueStructPtr(value) {
		return value.Elem()
	}
	return value
}

// GetFieldType returns the type of the field with fieldName in the containing
// parent, which must be a ptr to a struct.
// It returns an error if the parent is the wrong type or has no field called
// fieldName.
func GetFieldType(parent interface{}, fieldName string) (reflect.Type, error) {
	pt := reflect.TypeOf(parent)
	if !IsValueStructPtr(reflect.ValueOf(parent)) {
		return reflect.TypeOf(nil), fmt.Errorf("parent is type %T, must be struct ptr in GetFieldType with fieldName %s", parent, fieldName)
	}

	pt = pt.Elem()
	ft, ok := pt.FieldByName(fieldName)
	if !ok {
		return reflect.TypeOf(nil), fmt.Errorf("field name %s not a part of %T in GetFieldType", fieldName, parent)
	}

	switch ft.Type.Kind() {
	case reflect.Slice, reflect.Map:
		return ft.Type.Elem(), nil
	}
	return ft.Type, nil
}

// InsertIntoSlice inserts value into parent which must be a slice.
func InsertIntoSlice(parentSlice interface{}, value interface{}) error {
	DbgPrint("InsertIntoSlice into parent type %T with value %v, type %T", parentSlice, ValueStr(value), value)

	pv := reflect.ValueOf(parentSlice)
	t := reflect.TypeOf(parentSlice)
	v := reflect.ValueOf(value)

	if !IsTypeSlicePtr(t) {
		return fmt.Errorf("InsertIntoSlice parent type is %s, must be slice ptr", t)
	}

	pv.Elem().Set(reflect.Append(pv.Elem(), v))
	DbgPrint("new list: %v\n", pv.Elem().Interface())

	return nil
}

// InsertIntoMap inserts value with key into parent which must be a map.
func InsertIntoMap(parentMap interface{}, key interface{}, value interface{}) error {
	DbgPrint("InsertIntoMap into parent type %T with key %v(%T) value \n%s\n (%T)",
		parentMap, ValueStr(key), key, pretty.Sprint(value), value)

	v := reflect.ValueOf(parentMap)
	t := reflect.TypeOf(parentMap)
	kv := reflect.ValueOf(key)
	vv := reflect.ValueOf(value)

	if t.Kind() != reflect.Map {
		return fmt.Errorf("InsertIntoMap parent type is %s, must be map", t)
	}

	v.SetMapIndex(kv, vv)

	return nil
}

// UpdateField updates a field called fieldName (which must exist, but may be
// nil) in parentStruct, with value fieldValue. If the field is a slice,
// fieldValue is appended.
func UpdateField(parentStruct interface{}, fieldName string, fieldValue interface{}) error {
	DbgPrint("UpdateField field %s of parent type %T with value %v", fieldName, parentStruct, ValueStr(fieldValue))

	if IsValueNil(parentStruct) {
		return fmt.Errorf("parent is nil in UpdateField for field %s", fieldName)
	}

	pt := reflect.TypeOf(parentStruct)

	if !IsTypeStructPtr(pt) {
		return fmt.Errorf("parent type %T must be a struct ptr", parentStruct)
	}
	ft, ok := pt.Elem().FieldByName(fieldName)
	if !ok {
		return fmt.Errorf("parent type %T does not have a field name %s", parentStruct, fieldName)
	}

	if ft.Type.Kind() == reflect.Slice {
		return InsertIntoSliceStructField(parentStruct, fieldName, fieldValue)
	}
	return InsertIntoStruct(parentStruct, fieldName, fieldValue)
}

// InsertIntoStruct updates a field called fieldName (which must exist, but may
// be nil) in parentStruct, with value fieldValue.
// If the struct field type is a ptr and the value is non-ptr, the field is
// populated with the corresponding ptr type.
func InsertIntoStruct(parentStruct interface{}, fieldName string, fieldValue interface{}) error {
	DbgPrint("InsertIntoStruct field %s of parent type %T with value %v", fieldName, parentStruct, ValueStr(fieldValue))

	v, t := reflect.ValueOf(fieldValue), reflect.TypeOf(fieldValue)
	pv, pt := reflect.ValueOf(parentStruct), reflect.TypeOf(parentStruct)

	if !IsTypeStructPtr(pt) {
		return fmt.Errorf("parent type %T must be a struct ptr", parentStruct)
	}
	ft, ok := pt.Elem().FieldByName(fieldName)
	if !ok {
		return fmt.Errorf("parent type %T does not have a field name %s", parentStruct, fieldName)
	}

	n := v
	if n.IsValid() && (ft.Type.Kind() == reflect.Ptr && t.Kind() != reflect.Ptr) {
		n = reflect.New(t)
		n.Elem().Set(v)
	}

	if !n.IsValid() {
		if ft.Type.Kind() != reflect.Ptr {
			return fmt.Errorf("cannot assign value %v (type %T) to struct field %s (type %v) in struct %T", fieldValue, fieldValue, fieldName, ft.Type, parentStruct)
		}
		n = reflect.Zero(ft.Type)
	}

	if !isFieldTypeCompatible(ft, n) {
		return fmt.Errorf("cannot assign value %v (type %T) to struct field %s (type %v) in struct %T", fieldValue, fieldValue, fieldName, ft.Type, parentStruct)
	}

	pv.Elem().FieldByName(fieldName).Set(n)

	return nil
}

// InsertIntoSliceStructField inserts fieldValue into a field of type slice in parentStruct
// called fieldName (which must exist, but may be nil).
func InsertIntoSliceStructField(parentStruct interface{}, fieldName string, fieldValue interface{}) error {
	DbgPrint("InsertIntoSliceStructField field %s of parent type %T with value %v", fieldName, parentStruct, ValueStr(fieldValue))

	v, t := reflect.ValueOf(fieldValue), reflect.TypeOf(fieldValue)
	pv, pt := reflect.ValueOf(parentStruct), reflect.TypeOf(parentStruct)

	if !IsTypeStructPtr(pt) {
		return fmt.Errorf("parent type %T must be a struct ptr", parentStruct)
	}
	ft, ok := pt.Elem().FieldByName(fieldName)
	if !ok {
		return fmt.Errorf("parent type %T does not have a field name %s", parentStruct, fieldName)
	}
	if ft.Type.Kind() != reflect.Slice {
		return fmt.Errorf("parent type %T, field name %s in type %s, must be a slice", parentStruct, fieldName, ft.Type)
	}
	et := ft.Type.Elem()

	n := v
	if n.IsValid() && (et.Kind() == reflect.Ptr && t.Kind() != reflect.Ptr) {
		n = reflect.New(t)
		n.Elem().Set(v)
	}
	if !n.IsValid() {
		n = reflect.Zero(et)
	}
	if !isValueTypeCompatible(et, n) {
		return fmt.Errorf("cannot assign value %v (type %T) to struct field %s (type %v) in struct %T", fieldValue, fieldValue, fieldName, et, parentStruct)
	}

	nl := reflect.Append(pv.Elem().FieldByName(fieldName), n)
	pv.Elem().FieldByName(fieldName).Set(nl)

	return nil
}

// InsertIntoMapStructField inserts fieldValue into a field of type map in parentStruct
// called fieldName (which must exist, but may be nil), using the given key.
// If the key already exists in the map, the corresponding value is updated.
func InsertIntoMapStructField(parentStruct interface{}, fieldName string, key, fieldValue interface{}) error {
	DbgPrint("InsertIntoMapStructField field %s of parent type %T with key %v, value %v", fieldName, parentStruct, key, ValueStr(fieldValue))

	v := reflect.ValueOf(parentStruct)
	t := reflect.TypeOf(parentStruct)
	if v.Kind() == reflect.Ptr {
		t = reflect.TypeOf(v.Elem().Interface())
	}
	ft, ok := t.FieldByName(fieldName)
	if !ok {
		return fmt.Errorf("field %s not found in parent type %T", fieldName, parentStruct)
	}

	if ft.Type.Kind() != reflect.Map {
		return fmt.Errorf("field %s to insert into must be a map, type is %v", fieldName, ft.Type.Kind())
	}
	vv := v
	if v.Kind() == reflect.Ptr {
		vv = v.Elem()
	}
	fvn := reflect.TypeOf(vv.FieldByName(fieldName).Interface()).Elem()
	if fvn.Kind() != reflect.ValueOf(fieldValue).Kind() && !(fieldValue == nil && fvn.Kind() == reflect.Ptr) {
		return fmt.Errorf("cannot assign value %v (type %T) to field %s (type %v) in struct %s",
			fieldValue, fieldValue, fieldName, fvn.Kind(), t.Name())
	}

	n := reflect.New(fvn)
	if fieldValue != nil {
		n.Elem().Set(reflect.ValueOf(fieldValue))
	}
	fv := v.Elem().FieldByName(fieldName)
	if fv.IsNil() {
		fv.Set(reflect.MakeMap(fv.Type()))
	}
	fv.SetMapIndex(reflect.ValueOf(key), n.Elem())

	return nil
}

func isFieldTypeCompatible(ft reflect.StructField, v reflect.Value) bool {
	if ft.Type.Kind() == reflect.Ptr {
		if !v.IsValid() {
			return true
		}
		return v.Type() == ft.Type
	}
	if !v.IsValid() || IsValueNil(v.Interface()) {
		return false
	}
	return v.Type() == ft.Type
}

func isValueTypeCompatible(t reflect.Type, v reflect.Value) bool {
	if !v.IsValid() {
		return t.Kind() == reflect.Ptr
	}

	return v.Type().Kind() == t.Kind()
}

// NodeInfo describes a node in a tree being traversed. It is passed to the
// iterator function supplied to a traversal driver function like ForEach.
type NodeInfo struct {
	// ParentStruct is a ptr to the containing struct (if any).
	ParentStruct interface{}
	// FieldType is the StructField for the field being traversed.
	FieldType reflect.StructField
	// FieldValue is the Value for the field being traversed.
	FieldValue reflect.Value
	// FieldKeys is the slice of keys in the map being traversed. nil if type
	// being traversed is not a map.
	FieldKeys []reflect.Value
	// FieldKey is the key of the map element being traversed. ValueOf(nil) if
	// type being traversed is not a map.
	FieldKey reflect.Value
}

// FieldIteratorFunc is an iteration function for arbitrary field traversals.
// in, out are passed through from the caller to the iteration and can be used
// to pass state in and out.
// It returns a slice of errors encountered while processing the field.
type FieldIteratorFunc func(ni *NodeInfo, in, out interface{}) []error

// ForEachField recursively iterates through the fields of value (which may be
// any Go type) and executes iterFunction on each field.
//   in, out are passed to the iterator function and can be used to carry state
//     and return results from the iterator.
//   iterFunction is executed on each scalar field.
// It returns a slice of errors encountered while processing the struct.
func ForEachField(value interface{}, in, out interface{}, iterFunction FieldIteratorFunc) (errs []error) {
	if IsValueNil(value) {
		return nil
	}
	return forEachFieldInternal(&NodeInfo{FieldValue: reflect.ValueOf(value)}, in, out, iterFunction)
}

// forEachFieldInternal recursively iterates through the fields of value (which
// may be any Go type) and executes iterFunction on each field.
//   in, out are passed through from the caller to the iteration and can be used
//     arbitrarily in the iteration function to carry state and results.
func forEachFieldInternal(ni *NodeInfo, in, out interface{}, iterFunction FieldIteratorFunc) (errs []error) {
	if IsNilOrInvalidValue(ni.FieldValue) {
		return nil
	}

	errs = AppendErrs(errs, iterFunction(ni, in, out))

	switch {
	case IsValueStruct(ni.FieldValue) || IsValueStructPtr(ni.FieldValue):
		structElems := PtrToValue(ni.FieldValue)
		for i := 0; i < structElems.NumField(); i++ {
			nn := *ni
			nn.ParentStruct = ni.FieldValue.Interface()
			nn.FieldType = structElems.Type().Field(i)
			nn.FieldValue = structElems.Field(i)
			errs = AppendErrs(errs, forEachFieldInternal(&nn, in, out, iterFunction))
		}

	case IsValueSlice(ni.FieldValue):
		for i := 0; i < ni.FieldValue.Len(); i++ {
			nn := *ni
			nn.FieldValue = ni.FieldValue.Index(i)
			errs = AppendErrs(errs, forEachFieldInternal(&nn, in, out, iterFunction))
		}

	case IsValueMap(ni.FieldValue):
		for _, key := range ni.FieldValue.MapKeys() {
			nn := *ni
			nn.FieldValue = ni.FieldValue.MapIndex(key)
			nn.FieldKey = key
			nn.FieldKeys = ni.FieldValue.MapKeys()
			errs = AppendErrs(errs, forEachFieldInternal(&nn, in, out, iterFunction))
		}
	}

	return nil
}
