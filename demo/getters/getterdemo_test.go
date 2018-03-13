// Copyright 2018 Google Inc.
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

package getterdemo

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/ygot/demo/getters/pkg/getteroc"
	"github.com/openconfig/ygot/ygot"
)

func TestGettersNotPopulated(t *testing.T) {
	e := getteroc.Root{}

	if e.GetTop() != nil {
		t.Errorf("GetTop(): returned non-nil when not initialised")
	}

	if got, want := e.GetOrCreateTop().GetOrCreateConfig().GetA(), "A VALUE"; !cmp.Equal(got, ygot.String(want)) {
		t.Errorf("GetA(): did not get default value for leaf /top/config/a, got: %v, want: %v", got, want)
	}

	if got, want := e.GetOrCreateTop().GetChildLists(), (*getteroc.GettersDemo_Top_ChildLists)(nil); !cmp.Equal(got, want) {
		t.Errorf("GetChildLists(): did not get empty container for /top/child-lists, got: %#v, want: %#v", got, want)
	}

	kv := "key_one"
	e.GetTop().GetOrCreateChildLists().GetOrCreateChildList(kv)
	if _, ok := e.Top.ChildLists.ChildList[kv]; !ok {
		t.Errorf("GetOrCreateChildLists('%s'): did not create key correctly, got: %v, want: true", kv, ok)
	}

	v := e.GetTop().GetChildLists().GetChildList(kv)
	if got, want := v.GetOrCreateConfig().GetValueWithDefault(), uint32(42); !cmp.Equal(got, ygot.Uint32(want)) {
		t.Errorf("GetValueWithDefault(): did not get expected value, got: %v, want: %v", got, want)
	}

	// TODO(https://github.com/openconfig/ygot/issues/172): Ensure that this test passes by mapping
	// the UNSET value of an enum to its default.
	//if got, want := e.GetTop().GetConfig().GetB(), getteroc.GettersDemo_Top_Config_B_C; got != want {
	//	t.Errorf("GetB(): did not get expected default value for enumeration for /top/config/b, got: %v, want: %v", got, want)
	//}

	if got, want := e.GetTop().GetConfig().GetC(), getteroc.GettersDemo_Top_Config_C_UNSET; got != want {
		t.Errorf("GetC(): did not get expected default value for enumeration for /top/config/c, got: %v, want: %v", got, want)
	}
}

func GetterTestPopulated(t *testing.T) {
	e := getteroc.Root{
		Top: &getteroc.GettersDemo_Top{
			Config: &getteroc.GettersDemo_Top_Config{
				A: ygot.String("SET VALUE"),
				B: getteroc.GettersDemo_Top_Config_B_A,
				C: getteroc.GettersDemo_Top_Config_C_ONE,
				D: []string{"one", "two"},
			},
			ChildLists: &getteroc.GettersDemo_Top_ChildLists{
				ChildList: map[string]*getteroc.GettersDemo_Top_ChildLists_ChildList{
					"val_one": {
						K: ygot.String("val_one"),
						Config: &getteroc.GettersDemo_Top_ChildLists_ChildList_Config{
							K:                ygot.String("val_one"),
							ValueWithDefault: ygot.Uint32(84),
						},
					},
				},
			},
		},
	}

	if got, want := e.GetTop().GetConfig().GetA(), "SET VALUE"; !cmp.Equal(got, ygot.String(want)) {
		t.Errorf("GetA(): did not get expected value, got: %v, want: %v", got, want)
	}

	if got, want := e.GetTop().GetConfig().GetB(), getteroc.GettersDemo_Top_Config_B_A; got != want {
		t.Errorf("GetB(): did not get expected value, got: %v, want: %v", got, want)
	}

	if got, want := e.GetTop().GetConfig().GetC(), getteroc.GettersDemo_Top_Config_C_ONE; got != want {
		t.Errorf("GetC(): did not get expected value, got: %v, want: %v", got, want)
	}

	kv := "val_one"
	le := e.GetTop().GetChildLists().GetChildList(kv)
	if got, want := le.GetConfig().GetK(), kv; !cmp.Equal(got, ygot.String(want)) {
		t.Errorf("GetK(): did not get expected value, got: %v, want: %v", got, want)
	}

	if got, want := le.GetConfig().GetValueWithDefault(), uint32(84); !cmp.Equal(got, ygot.Uint32(want)) {
		t.Errorf("GetValueWithDefault(): did not get expected value, got: %v, want: %v", got, want)
	}
}
