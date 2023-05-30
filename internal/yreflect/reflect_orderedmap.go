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

	"github.com/openconfig/ygot/util"
)

// goOrderedList is a convenience interface for ygot.GoOrderedList. It is here
// to avoid a circular dependency.
type goOrderedList interface {
	// IsYANGOrderedList is a marker method that indicates that the struct
	// implements the goOrderedList interface.
	IsYANGOrderedList()
}

// AppendIntoOrderedMap appends a populated value into the ordered map.
//
// There must not exist an existing element with the same key.
func AppendIntoOrderedMap(orderedMap goOrderedList, value interface{}) error {
	appendMethod, err := util.MethodByName(reflect.ValueOf(orderedMap), "Append")
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
// If the visit function returns false, the for loop breaks.
// An erorr is returned if the ordered map is not well-formed.
func RangeOrderedMap(orderedMap goOrderedList, visit func(k reflect.Value, v reflect.Value) bool) error {
	omv := reflect.ValueOf(orderedMap)
	// First get the ordered keys, and then index into each of the values associated with it.
	keysMethod, err := util.MethodByName(omv, "Keys")
	if err != nil {
		return err
	}
	ret := keysMethod.Call(nil)
	if got, wantReturnN := len(ret), 1; got != wantReturnN {
		return fmt.Errorf("method Keys() doesn't have expected number of return values, got %v, want %v", got, wantReturnN)
	}
	keys := ret[0]
	if gotKind := keys.Type().Kind(); gotKind != reflect.Slice {
		return fmt.Errorf("method Keys() did not return a slice value, got %v", gotKind)
	}

	getMethod, err := util.MethodByName(omv, "Get")
	if err != nil {
		return err
	}

	for i := 0; i != keys.Len(); i++ {
		k := keys.Index(i)
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
