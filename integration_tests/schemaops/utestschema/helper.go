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

package utestschema

import (
	"testing"

	"github.com/openconfig/ygot/ygot"
)

// GetOrderedMap returns a populated ordered map with dummy values.
//
// - foo: foo-val
// - bar: bar-val
func GetOrderedMap(t *testing.T) *Ctestschema_OrderedLists_OrderedList_OrderedMap {
	orderedMap := &Ctestschema_OrderedLists_OrderedList_OrderedMap{}
	v, err := orderedMap.AppendNew("foo")
	if err != nil {
		t.Error(err)
	}
	// Config value
	v.GetOrCreateConfig().Value = ygot.String("foo-val")
	v, err = orderedMap.AppendNew("bar")
	if err != nil {
		t.Error(err)
	}
	// State value
	v.GetOrCreateState().Value = ygot.String("bar-val")
	return orderedMap
}

// GetDeviceWithOrderedMap returns a Device object with a populated ordered map
// field.
func GetDeviceWithOrderedMap(t *testing.T) *Device {
	return &Device{
		OrderedLists: &Ctestschema_OrderedLists{
			OrderedList: GetOrderedMap(t),
		},
	}
}

func GetOrderedMap2(t *testing.T) *Ctestschema_OrderedLists_OrderedList_OrderedMap {
	orderedMap := &Ctestschema_OrderedLists_OrderedList_OrderedMap{}
	v, err := orderedMap.AppendNew("wee")
	if err != nil {
		t.Error(err)
	}
	// Config value
	v.GetOrCreateConfig().Value = ygot.String("wee-val")
	v, err = orderedMap.AppendNew("woo")
	if err != nil {
		t.Error(err)
	}
	// State value
	v.GetOrCreateState().Value = ygot.String("woo-val")
	return orderedMap
}

func GetDeviceWithOrderedMap2(t *testing.T) *Device {
	return &Device{
		OrderedLists: &Ctestschema_OrderedLists{
			OrderedList: GetOrderedMap2(t),
		},
	}
}

func GetOrderedMultikeyedMap(t *testing.T) *CtestschemaRootmod_OrderedMultikeyedLists_OrderedMultikeyedList_OrderedMap {
	orderedMap := &CtestschemaRootmod_OrderedMultikeyedLists_OrderedMultikeyedList_OrderedMap{}
	v, err := orderedMap.AppendNew("foo", 42)
	if err != nil {
		t.Error(err)
	}
	// Config value
	v.GetOrCreateConfig().Value = ygot.String("foo-val")
	v, err = orderedMap.AppendNew("bar", 42)
	if err != nil {
		t.Error(err)
	}
	// State value
	v.GetOrCreateState().Value = ygot.String("bar-val")
	v, err = orderedMap.AppendNew("baz", 84)
	if err != nil {
		t.Error(err)
	}
	// Config & State value
	v.GetOrCreateConfig().Value = ygot.String("baz-val")
	v.GetOrCreateState().Value = ygot.String("baz-val")
	return orderedMap
}
