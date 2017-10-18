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

// Package ytypes implements YANG type validation logic.
package ytypes

import (
	"reflect"

	"github.com/openconfig/goyang/pkg/yang"
)

// stringMapKeys returns the keys for map m.
func stringMapKeys(m map[string]*yang.Entry) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// stringMapSetToSlice converts a string set expressed as a map m, into a slice
// of strings.
func stringMapSetToSlice(m map[string]interface{}) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// makeField sets field f in parentStruct to a default newly constructed value
// with the type of the given field.
func makeField(parentStruct reflect.Value, f reflect.StructField) {
	switch f.Type.Kind() {
	case reflect.Map:
		parentStruct.FieldByName(f.Name).Set(reflect.MakeMap(f.Type))
	case reflect.Slice:
		parentStruct.FieldByName(f.Name).Set(reflect.MakeSlice(f.Type, 0, 0))
	case reflect.Interface:
		// This is a union field type, which can only be created once its type
		// is known.
	default:
		parentStruct.FieldByName(f.Name).Set(reflect.New(f.Type.Elem()))
	}
}
