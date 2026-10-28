// Copyright 2023 Google Inc.
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

package yreflect

import (
	"fmt"
	"reflect"
	"sync"
)

// goOrderedMap is a convenience interface for ygot.GoOrderedMap. It is here
// to avoid a circular dependency.
type goOrderedMap interface {
	// IsYANGOrderedList is a marker method that indicates that the struct
	// implements the goOrderedMap interface.
	IsYANGOrderedList()
	// Len returns the size of the ordered list.
	Len() int
}

// orderedMapTypeInfo caches reflection method indices and parameter types for a
// concrete GoOrderedMap implementation type.
type orderedMapTypeInfo struct {
	// keysIdx is the method index of Keys(), or -1 if absent.
	keysIdx int
	// valuesIdx is the method index of Values(), or -1 if absent.
	valuesIdx int
	// getIdx is the method index of Get(), or -1 if absent.
	getIdx int
	// appendIdx is the method index of Append(), or -1 if absent.
	appendIdx int
	// keyType is the argument type of Get() when valid.
	keyType reflect.Type
	// keyTypeErr is the error encountered when resolving keyType.
	keyTypeErr error
	// elemType is the argument type of Append() when valid.
	elemType reflect.Type
	// elemTypeErr is the error encountered when resolving elemType.
	elemTypeErr error
}

// orderedMapTypes caches *orderedMapTypeInfo keyed by reflect.Type.
var orderedMapTypes sync.Map

// getOrderedMapTypeInfo returns the cached reflection metadata for t.
func getOrderedMapTypeInfo(t reflect.Type) *orderedMapTypeInfo {
	if cached, ok := orderedMapTypes.Load(t); ok {
		return cached.(*orderedMapTypeInfo)
	}
	info := &orderedMapTypeInfo{
		keysIdx:   -1,
		valuesIdx: -1,
		getIdx:    -1,
		appendIdx: -1,
	}
	if m, ok := t.MethodByName("Keys"); ok {
		info.keysIdx = m.Index
	}
	if m, ok := t.MethodByName("Values"); ok {
		info.valuesIdx = m.Index
	}
	if m, ok := t.MethodByName("Get"); ok {
		info.getIdx = m.Index
	}
	if m, ok := t.MethodByName("Append"); ok {
		info.appendIdx = m.Index
	}
	info.keyType, info.keyTypeErr = computeUnaryMethodArgType(t, "Get")
	info.elemType, info.elemTypeErr = computeUnaryMethodArgType(t, "Append")
	actual, _ := orderedMapTypes.LoadOrStore(t, info)
	return actual.(*orderedMapTypeInfo)
}

// MethodByName returns a valid method for the given value, or an error if the
// method is not valid for use.
func MethodByName(v reflect.Value, name string) (reflect.Value, error) {
	method := v.MethodByName(name)
	if !method.IsValid() || method.IsZero() {
		return method, fmt.Errorf("did not find %s() method on type: %s", name, v.Type().Name())
	}
	return method, nil
}

// AppendIntoOrderedMap appends a populated value into the ordered map.
//
// There must not exist an existing element with the same key.
func AppendIntoOrderedMap(orderedMap goOrderedMap, value any) error {
	omVal := reflect.ValueOf(orderedMap)
	info := getOrderedMapTypeInfo(omVal.Type())
	if info.appendIdx < 0 {
		return fmt.Errorf("did not find Append() method on type: %s", omVal.Type().Name())
	}
	appendMethod := omVal.Method(info.appendIdx)
	if !appendMethod.IsValid() || appendMethod.IsZero() {
		return fmt.Errorf("did not find Append() method on type: %s", omVal.Type().Name())
	}
	ret := appendMethod.Call([]reflect.Value{reflect.ValueOf(value)})
	if got, wantReturnN := len(ret), 1; got != wantReturnN {
		return fmt.Errorf("method Append() doesn't have expected number of return values, got %v, want %v", got, wantReturnN)
	}
	if err := ret[0].Interface(); err != nil {
		return fmt.Errorf("unable to append new ordered map element (it is expected that YANG `ordered-by user` lists are always unmarshalled as a whole instead of individually): %v", err)
	}

	return nil
}

// RangeOrderedMap calls a visitor function over each key-value pair in order.
//
// The for loop break when either the visit function returns false or an error
// is encountered due to the ordered map not being well-formed.
func RangeOrderedMap(orderedMap goOrderedMap, visit func(k reflect.Value, v reflect.Value) bool) error {
	omVal := reflect.ValueOf(orderedMap)
	info := getOrderedMapTypeInfo(omVal.Type())
	if info.getIdx < 0 {
		return fmt.Errorf("did not find Get() method on type: %s", omVal.Type().Name())
	}
	if info.keysIdx < 0 {
		return fmt.Errorf("did not find Keys() method on type: %s", omVal.Type().Name())
	}

	if info.valuesIdx >= 0 {
		if orderedMap == nil || (omVal.Kind() == reflect.Ptr && omVal.IsNil()) || orderedMap.Len() == 0 {
			return nil
		}
		keysRet := omVal.Method(info.keysIdx).Call(nil)
		if got, wantReturnN := len(keysRet), 1; got != wantReturnN {
			return fmt.Errorf("method Keys() doesn't have expected number of return values, got %v, want %v", got, wantReturnN)
		}
		keys := keysRet[0]
		if gotKind := keys.Type().Kind(); gotKind != reflect.Slice {
			return fmt.Errorf("method Keys() did not return a slice value, got %v", gotKind)
		}

		valsRet := omVal.Method(info.valuesIdx).Call(nil)
		if got, wantReturnN := len(valsRet), 1; got != wantReturnN {
			return fmt.Errorf("method Values() doesn't have expected number of return values, got %v, want %v", got, wantReturnN)
		}
		vals := valsRet[0]
		if gotKind := vals.Type().Kind(); gotKind != reflect.Slice {
			return fmt.Errorf("method Values() did not return a slice value, got %v", gotKind)
		}
		if gotKind := vals.Type().Elem().Kind(); gotKind != reflect.Ptr {
			return fmt.Errorf("method Keys() did not return a ptr value, got %v", gotKind)
		}

		n := keys.Len()
		for i := 0; i < n; i++ {
			if !visit(keys.Index(i), vals.Index(i)) {
				return nil
			}
		}
		return nil
	}

	getMethod := omVal.Method(info.getIdx)
	if !getMethod.IsValid() || getMethod.IsZero() {
		return fmt.Errorf("did not find Get() method on type: %s", omVal.Type().Name())
	}

	keys, err := OrderedMapKeys(orderedMap)
	if err != nil {
		return err
	}

	for _, k := range keys {
		ret := getMethod.Call([]reflect.Value{k})
		if got, wantReturnN := len(ret), 1; got != wantReturnN {
			return fmt.Errorf("method Get() doesn't have expected number of return values, got %v, want %v", got, wantReturnN)
		}
		v := ret[0]
		if gotKind := v.Type().Kind(); gotKind != reflect.Ptr {
			return fmt.Errorf("method Keys() did not return a ptr value, got %v", gotKind)
		}

		if !visit(k, v) {
			return nil
		}
	}

	return nil
}

// UnaryMethodArgType returns the argument type of the input type's specified
// unary method.
func UnaryMethodArgType(t reflect.Type, methodName string) (reflect.Type, error) {
	info := getOrderedMapTypeInfo(t)
	switch methodName {
	case "Append":
		return info.elemType, info.elemTypeErr
	case "Get":
		return info.keyType, info.keyTypeErr
	default:
		return computeUnaryMethodArgType(t, methodName)
	}
}

// computeUnaryMethodArgType resolves the argument type of t's unary method
// named methodName via reflection.
func computeUnaryMethodArgType(t reflect.Type, methodName string) (reflect.Type, error) {
	appendMethod, ok := t.MethodByName(methodName)
	if !ok {
		return nil, fmt.Errorf("did not find %s() method on type: %s", methodName, t.Name())
	}
	methodSpec := appendMethod.Func.Type()
	// The receiver is the first arg.
	if gotIn, wantIn := methodSpec.NumIn(), 2; gotIn != wantIn {
		return nil, fmt.Errorf("method %s() doesn't have expected number of input parameters, got %v, want %v", methodName, gotIn, wantIn)
	}
	return methodSpec.In(1), nil
}

// OrderedMapElementType returns the list element type of the ordered map.
func OrderedMapElementType(om goOrderedMap) (reflect.Type, error) {
	info := getOrderedMapTypeInfo(reflect.TypeOf(om))
	return info.elemType, info.elemTypeErr
}

// OrderedMapKeyType returns the key type of the ordered map, which will be a
// struct type for a multi-keyed list.
func OrderedMapKeyType(om goOrderedMap) (reflect.Type, error) {
	info := getOrderedMapTypeInfo(reflect.TypeOf(om))
	return info.keyType, info.keyTypeErr
}

// OrderedMapKeys returns the keys of the ordered map in a slice analogous to
// reflect's Value.MapKeys() method although it returns an error.
func OrderedMapKeys(om goOrderedMap) ([]reflect.Value, error) {
	omVal := reflect.ValueOf(om)
	info := getOrderedMapTypeInfo(omVal.Type())
	if info.keysIdx < 0 {
		return nil, fmt.Errorf("did not find Keys() method on type: %s", omVal.Type().Name())
	}
	keysMethod := omVal.Method(info.keysIdx)
	if !keysMethod.IsValid() || keysMethod.IsZero() {
		return nil, fmt.Errorf("did not find Keys() method on type: %s", omVal.Type().Name())
	}
	if om == nil || (omVal.Kind() == reflect.Ptr && omVal.IsNil()) || om.Len() == 0 {
		return nil, nil
	}
	ret := keysMethod.Call(nil)
	if got, wantReturnN := len(ret), 1; got != wantReturnN {
		return nil, fmt.Errorf("method Keys() doesn't have expected number of return values, got %v, want %v", got, wantReturnN)
	}
	keys := ret[0]
	if gotKind := keys.Type().Kind(); gotKind != reflect.Slice {
		return nil, fmt.Errorf("method Keys() did not return a slice value, got %v", gotKind)
	}

	n := keys.Len()
	if n == 0 {
		return nil, nil
	}
	keySlice := make([]reflect.Value, n)
	for i := 0; i < n; i++ {
		keySlice[i] = keys.Index(i)
	}

	return keySlice, nil
}

// GetOrderedMapElement calls the given ordered map's Get function given the
// key value.
//
// - reflect.Value is the retrieved value at the key.
// - bool is whether the value exists.
// - error is whether an unexpected condition was detected.
func GetOrderedMapElement(om goOrderedMap, k reflect.Value) (reflect.Value, bool, error) {
	omVal := reflect.ValueOf(om)
	info := getOrderedMapTypeInfo(omVal.Type())
	if info.getIdx < 0 {
		return reflect.Value{}, false, fmt.Errorf("did not find Get() method on type: %s", omVal.Type().Name())
	}
	getMethod := omVal.Method(info.getIdx)
	if !getMethod.IsValid() || getMethod.IsZero() {
		return reflect.Value{}, false, fmt.Errorf("did not find Get() method on type: %s", omVal.Type().Name())
	}

	ret := getMethod.Call([]reflect.Value{k})
	if got, wantReturnN := len(ret), 1; got != wantReturnN {
		return reflect.Value{}, false, fmt.Errorf("method Get() doesn't have expected number of return values, got %v, want %v", got, wantReturnN)
	}
	v := ret[0]
	if gotKind := v.Type().Kind(); gotKind != reflect.Ptr {
		return reflect.Value{}, false, fmt.Errorf("method Keys() did not return a ptr value, got %v", gotKind)
	}

	return v, !v.IsZero(), nil
}
