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
	appendMethod, err := MethodByName(reflect.ValueOf(orderedMap), "Append")
	if err != nil {
		return err
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
	getMethod, err := MethodByName(reflect.ValueOf(orderedMap), "Get")
	if err != nil {
		return err
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
	return UnaryMethodArgType(reflect.TypeOf(om), "Append")
}

// OrderedMapKeyType returns the key type of the ordered map, which will be a
// struct type for a multi-keyed list.
func OrderedMapKeyType(om goOrderedMap) (reflect.Type, error) {
	return UnaryMethodArgType(reflect.TypeOf(om), "Get")
}

// OrderedMapKeys returns the keys of the ordered map in a slice analogous to
// reflect's Value.MapKeys() method although it returns an error.
func OrderedMapKeys(om goOrderedMap) ([]reflect.Value, error) {
	// First get the ordered keys, and then index into each of the values associated with it.
	keysMethod, err := MethodByName(reflect.ValueOf(om), "Keys")
	if err != nil {
		return nil, err
	}
	ret := keysMethod.Call(nil)
	if got, wantReturnN := len(ret), 1; got != wantReturnN {
		return nil, fmt.Errorf("method Keys() doesn't have expected number of return values, got %v, want %v", got, wantReturnN)
	}
	keys := ret[0]
	if gotKind := keys.Type().Kind(); gotKind != reflect.Slice {
		return nil, fmt.Errorf("method Keys() did not return a slice value, got %v", gotKind)
	}

	var keySlice []reflect.Value
	for i := 0; i != keys.Len(); i++ {
		keySlice = append(keySlice, keys.Index(i))
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
	getMethod, err := MethodByName(reflect.ValueOf(om), "Get")
	if err != nil {
		return reflect.Value{}, false, err
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
