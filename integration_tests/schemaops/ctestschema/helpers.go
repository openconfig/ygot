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

package ctestschema

import (
	"reflect"
	"testing"

	"github.com/openconfig/ygot/ygot"
)

// MapStructTestOne is the base struct used for the simple-schema test.
type MapStructTestOne struct {
	Child       *MapStructTestOneChild  `path:"child" module:"test-one"`
	OrderedList *OrderedList_OrderedMap `path:"ordered-lists/ordered-list" module:"ctestschema/ctestschema"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*MapStructTestOne) IsYANGGoStruct() {}

func (*MapStructTestOne) ΛValidate(...ygot.ValidationOption) error {
	return nil
}

func (*MapStructTestOne) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*MapStructTestOne) ΛBelongingModule() string                { return "" }

// MapStructTestOneChild is a child structure of the MapStructTestOne test
// case.
type MapStructTestOneChild struct {
	FieldOne    *string                 `path:"config/field-one" module:"test-one/test-one"`
	FieldTwo    *uint32                 `path:"config/field-two" module:"test-one/test-one"`
	FieldThree  Binary                  `path:"config/field-three" module:"test-one/test-one"`
	FieldFour   []Binary                `path:"config/field-four" module:"test-one/test-one"`
	FieldFive   *uint64                 `path:"config/field-five" module:"test-five/test-five"`
	OrderedList *OrderedList_OrderedMap `path:"ordered-lists/ordered-list" module:"ctestschema/ctestschema"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*MapStructTestOneChild) IsYANGGoStruct() {}

func (*MapStructTestOneChild) ΛValidate(...ygot.ValidationOption) error {
	return nil
}

func (*MapStructTestOneChild) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*MapStructTestOneChild) ΛBelongingModule() string                { return "test-one" }

// GetOrderedMap returns a populated ordered map with dummy values.
//
// - foo: foo-val
// - bar: bar-val
func GetOrderedMap(t *testing.T) *OrderedList_OrderedMap {
	orderedMap := &OrderedList_OrderedMap{}
	v, err := orderedMap.AppendNew("foo")
	if err != nil {
		t.Error(err)
	}
	v.Value = ygot.String("foo-val")
	v, err = orderedMap.AppendNew("bar")
	if err != nil {
		t.Error(err)
	}
	v.Value = ygot.String("bar-val")
	return orderedMap
}

// GetOrderedMapLonger returns a populated ordered map with more dummy values.
//
// - foo: foo-val
// - bar: bar-val
// - baz: baz-val
func GetOrderedMapLonger(t *testing.T) *OrderedList_OrderedMap {
	orderedMap := &OrderedList_OrderedMap{}
	v, err := orderedMap.AppendNew("foo")
	if err != nil {
		t.Error(err)
	}
	v.Value = ygot.String("foo-val")
	v, err = orderedMap.AppendNew("bar")
	if err != nil {
		t.Error(err)
	}
	v.Value = ygot.String("bar-val")
	v, err = orderedMap.AppendNew("baz")
	if err != nil {
		t.Error(err)
	}
	v.Value = ygot.String("baz-val")
	return orderedMap
}

// GetOrderedMap2 returns a populated ordered map with different dummy values.
//
// - wee: wee-val
// - woo: woo-val
func GetOrderedMap2(t *testing.T) *OrderedList_OrderedMap {
	orderedMap := &OrderedList_OrderedMap{}
	v, err := orderedMap.AppendNew("wee")
	if err != nil {
		t.Error(err)
	}
	v.Value = ygot.String("wee-val")
	v, err = orderedMap.AppendNew("woo")
	if err != nil {
		t.Error(err)
	}
	v.Value = ygot.String("woo-val")
	return orderedMap
}

// GetNestedOrderedMap returns a populated nested ordered map with dummy
// values.
//
// - foo: foo-val
//   - foo: foo-val
//   - bar: bar-val
func GetNestedOrderedMap(t *testing.T) *OrderedList_OrderedMap {
	om := GetOrderedMap(t)

	nestedOrderedMap := &OrderedList_OrderedList_OrderedMap{}
	v, err := nestedOrderedMap.AppendNew("foo")
	if err != nil {
		t.Error(err)
	}
	v.Value = ygot.String("foo-val")
	v, err = nestedOrderedMap.AppendNew("bar")
	if err != nil {
		t.Error(err)
	}
	v.Value = ygot.String("bar-val")

	om.Get("foo").OrderedList = nestedOrderedMap
	return om
}

// GetOrderedMapMultikeyed returns a populated multi-keyed ordered map with
// dummy values.
//
// - foo, 42: foo-val
// - bar, 42: bar-val
// - baz, 84: baz-val
func GetOrderedMapMultikeyed(t *testing.T) *OrderedMultikeyedList_OrderedMap {
	orderedMap := &OrderedMultikeyedList_OrderedMap{}
	v, err := orderedMap.AppendNew("foo", 42)
	if err != nil {
		t.Error(err)
	}
	v.Value = ygot.String("foo-val")
	v, err = orderedMap.AppendNew("bar", 42)
	if err != nil {
		t.Error(err)
	}
	v.Value = ygot.String("bar-val")
	v, err = orderedMap.AppendNew("baz", 84)
	if err != nil {
		t.Error(err)
	}
	v.Value = ygot.String("baz-val")
	return orderedMap
}
