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

package ygot_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/ygot/integration_tests/schemaops/ctestschema"
	"github.com/openconfig/ygot/internal/ytestutil"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygot"
)

const (
	// TestRoot is the path to the directory within which the test runs, appended
	// to any filename that is to be loaded.
	TestRoot string = ""
)

// errToString returns an error as a string.
func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// mapStructTestFour is the top-level container used for the
// schema-with-list test.
type mapStructTestFour struct {
	C *mapStructTestFourC `path:"c"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*mapStructTestFour) IsYANGGoStruct() {}

func (*mapStructTestFour) ΛValidate(...ygot.ValidationOption) error {
	return nil
}

func (*mapStructTestFour) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*mapStructTestFour) ΛBelongingModule() string                { return "" }

// mapStructTestFourC is the "c" container used for the schema-with-list
// test.
type mapStructTestFourC struct {
	// ACLSet is a YANG list that is keyed with a string.
	ACLSet   map[string]*mapStructTestFourCACLSet   `path:"acl-set"`
	OtherSet map[ECTest]*mapStructTestFourCOtherSet `path:"other-set"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*mapStructTestFourC) IsYANGGoStruct() {}

func (*mapStructTestFourC) ΛValidate(...ygot.ValidationOption) error {
	return nil
}

func (*mapStructTestFourC) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*mapStructTestFourC) ΛBelongingModule() string                { return "" }

// mapStructTestFourCACLSet is the struct which represents each entry in
// the ACLSet list in the schema-with-list test.
type mapStructTestFourCACLSet struct {
	// Name explicitly maps to two leaves, as shown with the two values
	// that are pipe separated.
	Name        *string `path:"config/name|name"`
	SecondValue *string `path:"config/second-value"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*mapStructTestFourCACLSet) IsYANGGoStruct() {}

func (*mapStructTestFourCACLSet) ΛValidate(...ygot.ValidationOption) error {
	return nil
}

func (*mapStructTestFourCACLSet) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*mapStructTestFourCACLSet) ΛBelongingModule() string                { return "" }

// mapStructTestFourOtherSet is a map entry with a
type mapStructTestFourCOtherSet struct {
	Name ECTest `path:"config/name|name"`
}

// IsYANGGoStruct implements the GoStruct interface.
func (*mapStructTestFourCOtherSet) IsYANGGoStruct() {}

func (*mapStructTestFourCOtherSet) ΛValidate(...ygot.ValidationOption) error {
	return nil
}

func (*mapStructTestFourCOtherSet) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*mapStructTestFourCOtherSet) ΛBelongingModule() string                { return "" }

// ECTest is a synthesised derived type which is used to represent
// an enumeration in the YANG schema.
type ECTest int64

// IsYANGEnumeration ensures that the ECTest derived enum type implemnts
// the GoEnum interface.
func (ECTest) IsYANGGoEnum() {}

const (
	ECTestUNSET  = 0
	ECTestVALONE = 1
	ECTestVALTWO = 2
)

// ΛMap returns the enumeration dictionary associated with the mapStructTestFiveC
// struct.
func (ECTest) ΛMap() map[string]map[int64]ygot.EnumDefinition {
	return map[string]map[int64]ygot.EnumDefinition{
		"ECTest": {
			1: ygot.EnumDefinition{Name: "VAL_ONE", DefiningModule: "valone-mod"},
			2: ygot.EnumDefinition{Name: "VAL_TWO", DefiningModule: "valtwo-mod"},
		},
	}
}

func (e ECTest) String() string {
	return ygot.EnumLogString(e, int64(e), "ECTest")
}

// mapStructInvalid is a valid GoStruct whose ΛValidate() method always returns
// an error.
type mapStructInvalid struct {
	Name *string `path:"name"`
}

// IsYANGGoStruct implements the GoStruct interface.
func (*mapStructInvalid) IsYANGGoStruct() {}

// Validate implements the GoStruct interface.
func (*mapStructInvalid) ΛValidate(...ygot.ValidationOption) error {
	return fmt.Errorf("invalid")
}

func (*mapStructInvalid) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*mapStructInvalid) ΛBelongingModule() string                { return "" }

// mapStructNoPaths is a valid GoStruct who does not implement path tags.
type mapStructNoPaths struct {
	Name *string
}

// IsYANGGoStruct implements the GoStruct interface.
func (*mapStructNoPaths) IsYANGGoStruct() {}

// Validate implements the GoStruct interface.
func (*mapStructNoPaths) ΛValidate(...ygot.ValidationOption) error { return nil }
func (*mapStructNoPaths) ΛEnumTypeMap() map[string][]reflect.Type  { return nil }
func (*mapStructNoPaths) ΛBelongingModule() string                 { return "" }

// TestEmitJSON validates that the EmitJSON function outputs the expected JSON
// for a set of input structs and schema definitions.
func TestEmitJSON(t *testing.T) {
	tests := []struct {
		name         string
		inStruct     ygot.GoStruct
		inConfig     *ygot.EmitJSONConfig
		wantJSONPath string
		wantErr      string
	}{{
		name: "simple schema JSON output",
		inStruct: &ctestschema.MapStructTestOne{
			Child: &ctestschema.MapStructTestOneChild{
				FieldOne: ygot.String("abc -> def"),
				FieldTwo: ygot.Uint32(42),
			},
		},
		wantJSONPath: filepath.Join(TestRoot, "testdata/emitjson_1.json-txt"),
	}, {
		name: "simple schema JSON output with safe HTML",
		inStruct: &ctestschema.MapStructTestOne{
			Child: &ctestschema.MapStructTestOneChild{
				FieldOne: ygot.String("abc -> def"),
				FieldTwo: ygot.Uint32(42),
			},
		},
		inConfig: &ygot.EmitJSONConfig{
			EscapeHTML: true,
		},
		wantJSONPath: filepath.Join(TestRoot, "testdata/emitjson_1_html_safe.json-txt"),
	}, {
		name: "schema with a list JSON output",
		inStruct: &mapStructTestFour{
			C: &mapStructTestFourC{
				ACLSet: map[string]*mapStructTestFourCACLSet{
					"n42": {Name: ygot.String("n42"), SecondValue: ygot.String("val")},
				},
			},
		},
		wantJSONPath: filepath.Join(TestRoot, "testdata/emitjson_2.json-txt"),
	}, {
		name: "simple schema IETF JSON output",
		inStruct: &ctestschema.MapStructTestOne{
			Child: &ctestschema.MapStructTestOneChild{
				FieldOne:  ygot.String("bar"),
				FieldTwo:  ygot.Uint32(84),
				FieldFive: ygot.Uint64(42),
			},
		},
		inConfig: &ygot.EmitJSONConfig{
			Format: ygot.RFC7951,
			RFC7951Config: &ygot.RFC7951JSONConfig{
				AppendModuleName: true,
			},
			Indent: "  ",
		},
		wantJSONPath: filepath.Join(TestRoot, "testdata/emitjson1_ietf.json-txt"),
	}, {
		name: "schema with list and enum IETF JSON",
		inStruct: &mapStructTestFour{
			C: &mapStructTestFourC{
				ACLSet: map[string]*mapStructTestFourCACLSet{
					"n42": {Name: ygot.String("n42"), SecondValue: ygot.String("foo")},
				},
				OtherSet: map[ECTest]*mapStructTestFourCOtherSet{
					ECTestVALONE: {Name: ECTestVALONE},
					ECTestVALTWO: {Name: ECTestVALTWO},
				},
			},
		},
		inConfig: &ygot.EmitJSONConfig{
			Format: ygot.RFC7951,
			RFC7951Config: &ygot.RFC7951JSONConfig{
				AppendModuleName: true,
			},
			Indent: "  ",
		},
		wantJSONPath: filepath.Join(TestRoot, "testdata/emitjson2_ietf.json-txt"),
	}, {
		name: "schema with container around a ordered list JSON output",
		inStruct: &ctestschema.MapStructTestOne{
			OrderedList: ctestschema.GetOrderedMapLonger(t),
		},
		wantJSONPath: filepath.Join(TestRoot, "testdata/emitjson_orderedmap_container_internal.json-txt"),
	}, {
		name:     "invalid struct contents",
		inStruct: &mapStructInvalid{Name: ygot.String("aardvark")},
		wantErr:  "validation err: invalid",
	}, {
		name:     "invalid with skip validation",
		inStruct: &mapStructInvalid{Name: ygot.String("aardwolf")},
		inConfig: &ygot.EmitJSONConfig{
			SkipValidation: true,
		},
		wantJSONPath: filepath.Join(TestRoot, "testdata", "invalid-struct.json-txt"),
	}, {
		name:     "invalid internal JSON",
		inStruct: &mapStructNoPaths{Name: ygot.String("honey badger")},
		wantErr:  "ConstructInternalJSON error: Name: field did not specify a path",
	}, {
		name:     "invalid RFC7951 JSON",
		inStruct: &mapStructNoPaths{Name: ygot.String("ladybird")},
		inConfig: &ygot.EmitJSONConfig{
			Format: ygot.RFC7951,
		},
		wantErr: "ConstructIETFJSON error: Name: field did not specify a path",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ygot.EmitJSON(tt.inStruct, tt.inConfig)
			if errToString(err) != tt.wantErr {
				t.Fatalf("%s: EmitJSON(%v, nil): did not get expected error, got: %v, want (\"\" means no error expected): %q", tt.name, tt.inStruct, err, tt.wantErr)
			}

			if tt.wantErr != "" {
				return
			}

			wantJSON, ioerr := ioutil.ReadFile(tt.wantJSONPath)
			if ioerr != nil {
				t.Fatalf("%s: ioutil.ReadFile(%s): could not open file: %v", tt.name, tt.wantJSONPath, ioerr)
			}
			strJSON := strings.TrimRight(string(wantJSON), "\n")

			if diff := pretty.Compare(got, strJSON); diff != "" {
				if diffl, err := testutil.GenerateUnifiedDiff(string(wantJSON), got); err == nil {
					diff = diffl
				}
				t.Errorf("%s: EmitJSON(%v, nil): got invalid JSON, diff(-want, +got):\n%s", tt.name, tt.inStruct, diff)
			}
		})
	}
}

func TestDeepCopyOrderedMap(t *testing.T) {
	tests := []struct {
		name             string
		in               func() *ctestschema.Device
		inKey            string
		wantErrSubstring string
	}{{
		name: "single-keyed",
		in:   func() *ctestschema.Device { return &ctestschema.Device{OrderedList: ctestschema.GetOrderedMap(t)} },
	}, {
		name: "multi-keyed",
		in: func() *ctestschema.Device {
			return &ctestschema.Device{OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t)}
		},
	}, {
		name: "nested",
		in: func() *ctestschema.Device {
			return &ctestschema.Device{OrderedList: ctestschema.GetNestedOrderedMap(t)}
		},
	}}

	for _, tt := range tests {
		got, err := ygot.DeepCopy(tt.in())
		gotRoot, ok := got.(*ctestschema.Device)
		if !ok {
			t.Fatalf("Got object that's not root device: %T", got)
		}

		in := tt.in()
		if err != nil {
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Errorf("%s: DeepCopy(%#v): did not get expected error, %s", tt.name, in, diff)
			}
			continue
		}

		if diff := cmp.Diff(got, in, ytestutil.OrderedMapCmpOptions...); diff != "" {
			t.Errorf("did not get identical copy, diff(-got,+want):\n%s", diff)
		}

		gotValues := gotRoot.OrderedList.Values()
		for i, inV := range in.OrderedList.Values() {
			gotV := gotValues[i]
			if inV == gotV {
				t.Errorf("%s: DeepCopy: after copy, input and copy have same memory address: %v", tt.name, inV)
			}
			if gotV == nil {
				continue
			}
			if inV.Key != nil && inV.Key == gotV.Key {
				t.Errorf("%s: DeepCopy: key have same address", tt.name)
			}
			if inV.ParentKey != nil && inV.ParentKey == gotV.ParentKey {
				t.Errorf("%s: DeepCopy: ParentKey have same address", tt.name)
			}
			if inV.RoValue != nil && inV.RoValue == gotV.RoValue {
				t.Errorf("%s: DeepCopy: RoValue have same address", tt.name)
			}
			if inV.Value != nil && inV.Value == gotV.Value {
				t.Errorf("%s: DeepCopy: Value have same address", tt.name)
			}
			for j, inV := range inV.OrderedList.Values() {
				gotV := gotValues[i].OrderedList.Values()[j]
				if inV == gotV {
					t.Errorf("%s: DeepCopy: after copy, input and copy have same memory address: %v", tt.name, inV)
				}
				if gotV == nil {
					continue
				}
				if inV.Key != nil && inV.Key == gotV.Key {
					t.Errorf("%s: DeepCopy: key have same address", tt.name)
				}
				if inV.ParentKey != nil && inV.ParentKey == gotV.ParentKey {
					t.Errorf("%s: DeepCopy: ParentKey have same address", tt.name)
				}
				if inV.Value != nil && inV.Value == gotV.Value {
					t.Errorf("%s: DeepCopy: Value have same address", tt.name)
				}
			}
		}

		gotMultikeyedValues := gotRoot.OrderedMultikeyedList.Values()
		for i, inV := range in.OrderedMultikeyedList.Values() {
			gotV := gotMultikeyedValues[i]
			if inV == gotV {
				t.Errorf("%s: DeepCopy: after copy, input and copy have same memory address: %v", tt.name, inV)
			}
			if gotV == nil {
				continue
			}
			if inV.Key1 != nil && inV.Key1 == gotV.Key1 {
				t.Errorf("%s: DeepCopy: key have same address", tt.name)
			}
			if inV.Key2 != nil && inV.Key2 == gotV.Key2 {
				t.Errorf("%s: DeepCopy: RoValue have same address", tt.name)
			}
			if inV.Value != nil && inV.Value == gotV.Value {
				t.Errorf("%s: DeepCopy: Value have same address", tt.name)
			}
		}
	}
}
