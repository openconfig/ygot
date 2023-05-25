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

package testutil

import (
	"fmt"
	"reflect"
	"testing"
)

// TODO: This package should be auto-generated rather than copied and then
// handcrafted to avoid circular dependency with the ygot package when being
// used in packages that ygot imports, or the ygot package itself.

// MapStructTestOne is the base struct used for the simple-schema test.
type MapStructTestOne struct {
	Child       *MapStructTestOneChild `path:"child" module:"test-one"`
	OrderedList *OrderedMap            `path:"ordered-lists/ordered-list" module:"ctestschema/ctestschema"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*MapStructTestOne) IsYANGGoStruct() {}

func (*MapStructTestOne) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*MapStructTestOne) ΛBelongingModule() string                { return "" }

// MapStructTestOneChild is a child structure of the MapStructTestOne test
// case.
type MapStructTestOneChild struct {
	FieldOne    *string     `path:"config/field-one" module:"test-one/test-one"`
	FieldTwo    *uint32     `path:"config/field-two" module:"test-one/test-one"`
	FieldThree  Binary      `path:"config/field-three" module:"test-one/test-one"`
	FieldFour   []Binary    `path:"config/field-four" module:"test-one/test-one"`
	FieldFive   *uint64     `path:"config/field-five" module:"test-five/test-five"`
	OrderedList *OrderedMap `path:"ordered-lists/ordered-list" module:"ctestschema/ctestschema"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*MapStructTestOneChild) IsYANGGoStruct() {}

func (*MapStructTestOneChild) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*MapStructTestOneChild) ΛBelongingModule() string                { return "test-one" }

type OrderedMap struct {
	keys     []string
	valueMap map[string]*OrderedList
}

// IsYANGOrderedList ensures that OrderedList_OrderedMap implements the
// ygot.GoOrderedList interface.
func (*OrderedMap) IsYANGOrderedList() {}

// ΛListKeyMap ensures that OrderedList implements the KeyHelperGoStruct
// helper.
func (p *OrderedList) ΛListKeyMap() (map[string]interface{}, error) {
	if p.Key == nil {
		return nil, fmt.Errorf("invalid input, key Val was nil")
	}
	return map[string]interface{}{
		"key": *p.Key,
	}, nil
}

// Keys returns a copy of the list's keys.
func (o *OrderedMap) Keys() []string {
	if o == nil {
		return nil
	}
	return append([]string{}, o.keys...)
}

// Values returns the current set of the list's values in order.
func (o *OrderedMap) Values() []*OrderedList {
	if o == nil {
		return nil
	}
	var values []*OrderedList
	for _, key := range o.keys {
		values = append(values, o.valueMap[key])
	}
	return values
}

// Get returns the value corresponding to the key. If the key is not found, nil
// is returned.
func (o *OrderedMap) Get(key string) *OrderedList {
	if o == nil {
		return nil
	}
	return o.valueMap[key]
}

// init initializes any uninitialized values.
func (o *OrderedMap) init() {
	if o == nil {
		return
	}
	if o.valueMap == nil {
		o.valueMap = map[string]*OrderedList{}
	}
}

// Append appends a OrderedList, returning an error if the key
// already exists in the ordered list or if the key is unspecified.
func (o *OrderedMap) Append(v *OrderedList) error {
	if o == nil {
		return fmt.Errorf("nil ordered map, cannot append OrderedList")
	}
	if v == nil {
		return fmt.Errorf("nil OrderedList")
	}
	if v.Key == nil {
		return fmt.Errorf("invalid nil key received for Key")
	}

	key := *v.Key

	if _, ok := o.valueMap[key]; ok {
		return fmt.Errorf("duplicate key for list Statement %v", key)
	}
	o.keys = append(o.keys, key)
	o.init()
	o.valueMap[key] = v
	return nil
}

// AppendNew creates and appends a new OrderedList, returning the
// newly-initialized v. It returns an error if the v already exists.
func (o *OrderedMap) AppendNew(Key string) (*OrderedList, error) {
	if o == nil {
		return nil, fmt.Errorf("nil ordered map, cannot append OrderedList")
	}
	key := Key

	if _, ok := o.valueMap[key]; ok {
		return nil, fmt.Errorf("duplicate key for list Statement %v", key)
	}
	o.keys = append(o.keys, key)
	newElement := &OrderedList{
		Key: &Key,
	}
	o.init()
	o.valueMap[key] = newElement
	return newElement, nil
}

// OrderedList represents the /ctestschema/ordered-lists/ordered-list YANG schema element.
type OrderedList struct {
	Key         *string     `path:"config/key|key" module:"ctestschema/ctestschema|ctestschema"`
	Value       *string     `path:"config/value" module:"ctestschema/ctestschema"`
	OrderedList *OrderedMap `path:"ordered-lists/ordered-list" module:"ctestschema/ctestschema"`
}

// IsYANGGoStruct ensures that OrderedList implements the yang.GoStruct
// interface. This allows functions that need to handle this struct to
// identify it as being generated by ygen.
func (*OrderedList) IsYANGGoStruct() {}

// stringHelper takes a string argument and returns a pointer to it. It is
// meant to avoid a cyclic dependency with ygot. In the future we can consider
// moving dependees to their own test package so that these cyclic dependencies
// can be broken.
func stringHelper(s string) *string { return &s }

func GetOrderedMap(t *testing.T) *OrderedMap {
	orderedMap := &OrderedMap{}
	v, err := orderedMap.AppendNew("foo")
	if err != nil {
		t.Error(err)
	}
	v.Value = stringHelper("foo-val")
	v, err = orderedMap.AppendNew("bar")
	if err != nil {
		t.Error(err)
	}
	v.Value = stringHelper("bar-val")
	return orderedMap
}

func GetOrderedMap2(t *testing.T) *OrderedMap {
	orderedMap := &OrderedMap{}
	v, err := orderedMap.AppendNew("wee")
	if err != nil {
		t.Error(err)
	}
	v.Value = stringHelper("wee-val")
	v, err = orderedMap.AppendNew("woo")
	if err != nil {
		t.Error(err)
	}
	v.Value = stringHelper("woo-val")
	return orderedMap
}

func GetNestedOrderedMap(t *testing.T) *OrderedMap {
	om := GetOrderedMap(t)
	om.Get("foo").OrderedList = GetOrderedMap(t)
	return om
}
