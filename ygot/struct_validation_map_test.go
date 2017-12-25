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

package ygot

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/pmezard/go-difflib/difflib"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

const (
	// TestRoot is the path to the directory within which the test runs, appended
	// to any filename that is to be loaded.
	TestRoot string = ""
)

// generateUnifiedDiff takes two strings and generates a diff that can be
// shown to the user in a test error message.
func generateUnifiedDiff(want, got string) (string, error) {
	diffl := difflib.UnifiedDiff{
		A:        difflib.SplitLines(got),
		B:        difflib.SplitLines(want),
		FromFile: "got",
		ToFile:   "want",
		Context:  3,
		Eol:      "\n",
	}
	return difflib.GetUnifiedDiffString(diffl)
}

// errToString returns an error as a string.
func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestStructTagToLibPaths(t *testing.T) {
	tests := []struct {
		name     string
		inField  reflect.StructField
		inParent *gnmiPath
		want     []*gnmiPath
		wantErr  bool
	}{{
		name: "invalid input path",
		inField: reflect.StructField{
			Name: "field",
			Tag:  `path:"foo"`,
		},
		inParent: &gnmiPath{
			pathElemPath:    []*gnmipb.PathElem{},
			stringSlicePath: []string{},
		},
		wantErr: true,
	}, {
		name: "simple single tag example",
		inField: reflect.StructField{
			Name: "field",
			Tag:  `path:"foo"`,
		},
		inParent: &gnmiPath{
			stringSlicePath: []string{},
		},
		want: []*gnmiPath{{
			stringSlicePath: []string{"foo"},
		}},
	}, {
		name: "empty tag example",
		inField: reflect.StructField{
			Name: "field",
			Tag:  `path:"" rootpath:""`,
		},
		inParent: &gnmiPath{
			stringSlicePath: []string{},
		},
		want: []*gnmiPath{{
			stringSlicePath: []string{},
		}},
	}, {
		name: "multiple path",
		inField: reflect.StructField{
			Name: "field",
			Tag:  `path:"foo/bar|bar"`,
		},
		inParent: &gnmiPath{
			stringSlicePath: []string{},
		},
		want: []*gnmiPath{{
			stringSlicePath: []string{"foo", "bar"},
		}, {
			stringSlicePath: []string{"bar"},
		}},
	}, {
		name: "populated parent path",
		inField: reflect.StructField{
			Name: "field",
			Tag:  `path:"baz|foo/baz"`,
		},
		inParent: &gnmiPath{
			stringSlicePath: []string{"existing"},
		},
		want: []*gnmiPath{{
			stringSlicePath: []string{"existing", "baz"},
		}, {
			stringSlicePath: []string{"existing", "foo", "baz"},
		}},
	}, {
		name: "simple pathelem single tag example",
		inField: reflect.StructField{
			Name: "field",
			Tag:  `path:"foo"`,
		},
		inParent: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{},
		},
		want: []*gnmiPath{{
			pathElemPath: []*gnmipb.PathElem{{Name: "foo"}},
		}},
	}, {
		name: "empty tag pathelem example",
		inField: reflect.StructField{
			Name: "field",
			Tag:  `path:"" rootpath:""`,
		},
		inParent: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{},
		},
		want: []*gnmiPath{{
			pathElemPath: []*gnmipb.PathElem{},
		}},
	}, {
		name: "multiple pathelem path",
		inField: reflect.StructField{
			Name: "field",
			Tag:  `path:"foo/bar|bar"`,
		},
		inParent: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{},
		},
		want: []*gnmiPath{{
			pathElemPath: []*gnmipb.PathElem{{Name: "foo"}, {Name: "bar"}},
		}, {
			pathElemPath: []*gnmipb.PathElem{{Name: "bar"}},
		}},
	}, {
		name: "populated pathelem parent path",
		inField: reflect.StructField{
			Name: "field",
			Tag:  `path:"baz|foo/baz"`,
		},
		inParent: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{{Name: "existing"}},
		},
		want: []*gnmiPath{{
			pathElemPath: []*gnmipb.PathElem{{Name: "existing"}, {Name: "baz"}},
		}, {
			pathElemPath: []*gnmipb.PathElem{{Name: "existing"}, {Name: "foo"}, {Name: "baz"}},
		}},
	}}

	for _, tt := range tests {
		got, err := structTagToLibPaths(tt.inField, tt.inParent)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: structTagToLibPaths(%v, %v): did not get expected error status, got: %v, want err: %v", tt.name, tt.inField, tt.inParent, err, tt.wantErr)
		}

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("%s: structTagToLibPaths(%v, %v): did not get expected set of map paths, diff(-got,+want):\n%s", tt.name, tt.inField, tt.inParent, diff)
		}
	}
}

type enumTest int64

func (enumTest) IsYANGGoEnum() {}

const (
	EUNSET enumTest = 0
	EONE   enumTest = 1
	ETWO   enumTest = 2
)

func (enumTest) ΛMap() map[string]map[int64]EnumDefinition {
	return map[string]map[int64]EnumDefinition{
		"enumTest": {
			1: EnumDefinition{Name: "VAL_ONE", DefiningModule: "valone-mod"},
			2: EnumDefinition{Name: "VAL_TWO", DefiningModule: "valtwo-mod"},
		},
	}
}

type badEnumTest int64

func (badEnumTest) IsYANGGoEnum() {}

const (
	BUNSET badEnumTest = 0
	BONE   badEnumTest = 1
)

func (badEnumTest) ΛMap() map[string]map[int64]EnumDefinition {
	return nil
}

func TestEnumFieldToString(t *testing.T) {
	var i interface{}
	i = EONE
	if _, ok := i.(GoEnum); !ok {
		t.Fatalf("TestEnumFieldToString: %T is not a valid GoEnum", i)
	}

	tests := []struct {
		name               string
		inField            reflect.Value
		inAppendModuleName bool
		wantName           string
		wantSet            bool
		wantErr            string
	}{{
		name:     "simple enum",
		inField:  reflect.ValueOf(EONE),
		wantName: "VAL_ONE",
		wantSet:  true,
	}, {
		name:     "unset enum",
		inField:  reflect.ValueOf(EUNSET),
		wantName: "",
		wantSet:  false,
	}, {
		name:               "simple enum with append module name",
		inField:            reflect.ValueOf(ETWO),
		inAppendModuleName: true,
		wantName:           "valtwo-mod:VAL_TWO",
		wantSet:            true,
	}, {
		name:    "bad enum - no mapping",
		inField: reflect.ValueOf(BONE),
		wantErr: "cannot map enumerated value as type badEnumTest was unknown",
	}}

	for _, tt := range tests {
		gotName, gotSet, err := enumFieldToString(tt.inField, tt.inAppendModuleName)
		if err != nil && err.Error() != tt.wantErr {
			t.Errorf("%s: enumFieldToString(%v, %v): did not get expected error, got: %v, want: %v", tt.name, tt.inField, tt.inAppendModuleName, err, tt.wantErr)
		}

		if gotName != tt.wantName {
			t.Errorf("%s: enumFieldToString(%v, %v): did not get expected name, got: %v, want: %v", tt.name, tt.inField, tt.inAppendModuleName, gotName, tt.wantName)
		}

		if gotSet != tt.wantSet {
			t.Errorf("%s: enumFieldToString(%v, %v): did not get expected set status, got: %v, want: %v", tt.name, tt.inField, tt.inAppendModuleName, gotSet, tt.wantSet)
		}
	}
}

// mapStructTestOne is the base struct used for the simple-schema test.
type mapStructTestOne struct {
	Child *mapStructTestOneChild `path:"child" module:"test-one"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*mapStructTestOne) IsYANGGoStruct() {}

func (*mapStructTestOne) Validate(...ValidationOption) error {
	return nil
}

func (*mapStructTestOne) ΛEnumTypeMap() map[string][]reflect.Type { return nil }

// mapStructTestOne_Child is a child structure of the mapStructTestOne test
// case.
type mapStructTestOneChild struct {
	FieldOne   *string  `path:"config/field-one" module:"test-one"`
	FieldTwo   *uint32  `path:"config/field-two" module:"test-one"`
	FieldThree Binary   `path:"config/field-three" module:"test-one"`
	FieldFour  []Binary `path:"config/field-four" module:"test-one"`
	FieldFive  *uint64  `path:"config/field-five" module:"test-five"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*mapStructTestOneChild) IsYANGGoStruct() {}

func (*mapStructTestOneChild) Validate(...ValidationOption) error {
	return nil
}

func (*mapStructTestOneChild) ΛEnumTypeMap() map[string][]reflect.Type { return nil }

// mapStructTestFour is the top-level container used for the
// schema-with-list test.
type mapStructTestFour struct {
	C *mapStructTestFourC `path:"c"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*mapStructTestFour) IsYANGGoStruct() {}

func (*mapStructTestFour) Validate(...ValidationOption) error {
	return nil
}

func (*mapStructTestFour) ΛEnumTypeMap() map[string][]reflect.Type { return nil }

// mapStructTestFourC is the "c" container used for the schema-with-list
// test.
type mapStructTestFourC struct {
	// ACLSet is a YANG list that is keyed with a string.
	ACLSet   map[string]*mapStructTestFourCACLSet   `path:"acl-set"`
	OtherSet map[ECTest]*mapStructTestFourCOtherSet `path:"other-set"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*mapStructTestFourC) IsYANGGoStruct() {}

func (*mapStructTestFourC) Validate(...ValidationOption) error {
	return nil
}

func (*mapStructTestFourC) ΛEnumTypeMap() map[string][]reflect.Type { return nil }

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

func (*mapStructTestFourCACLSet) Validate(...ValidationOption) error {
	return nil
}

func (*mapStructTestFourCACLSet) ΛEnumTypeMap() map[string][]reflect.Type { return nil }

// mapStructTestFourOtherSet is a map entry with a
type mapStructTestFourCOtherSet struct {
	Name ECTest `path:"config/name|name"`
}

// IsYANGGoStruct implements the GoStruct interface.
func (*mapStructTestFourCOtherSet) IsYANGGoStruct() {}

func (*mapStructTestFourCOtherSet) Validate(...ValidationOption) error {
	return nil
}

func (*mapStructTestFourCOtherSet) ΛEnumTypeMap() map[string][]reflect.Type { return nil }

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
func (ECTest) ΛMap() map[string]map[int64]EnumDefinition {
	return map[string]map[int64]EnumDefinition{
		"ECTest": {
			1: EnumDefinition{Name: "VAL_ONE", DefiningModule: "valone-mod"},
			2: EnumDefinition{Name: "VAL_TWO", DefiningModule: "valtwo-mod"},
		},
	}
}

// mapStructInvalid is a valid GoStruct whose Validate() method always returns
// an error.
type mapStructInvalid struct {
	Name *string `path:"name"`
}

// IsYANGGoStruct implements the GoStruct interface.
func (*mapStructInvalid) IsYANGGoStruct() {}

// Validate implements the ValidatedGoStruct interface.
func (*mapStructInvalid) Validate(...ValidationOption) error {
	return fmt.Errorf("invalid")
}

func (*mapStructInvalid) ΛEnumTypeMap() map[string][]reflect.Type { return nil }

// mapStructNoPaths is a valid GoStruct who does not implement path tags.
type mapStructNoPaths struct {
	Name *string
}

// IsYANGGoStruct implements the GoStruct interface.
func (*mapStructNoPaths) IsYANGGoStruct() {}

// Validate implements the ValidatedGoStruct interface.
func (*mapStructNoPaths) Validate(...ValidationOption) error      { return nil }
func (*mapStructNoPaths) ΛEnumTypeMap() map[string][]reflect.Type { return nil }

// TestEmitJSON validates that the EmitJSON function outputs the expected JSON
// for a set of input structs and schema definitions.
func TestEmitJSON(t *testing.T) {
	tests := []struct {
		name         string
		inStruct     ValidatedGoStruct
		inConfig     *EmitJSONConfig
		wantJSONPath string
		wantErr      string
	}{{
		name: "simple schema JSON output",
		inStruct: &mapStructTestOne{
			Child: &mapStructTestOneChild{
				FieldOne: String("hello"),
				FieldTwo: Uint32(42),
			},
		},
		wantJSONPath: filepath.Join(TestRoot, "testdata/emitjson_1.json-txt"),
	}, {
		name: "schema with a list JSON output",
		inStruct: &mapStructTestFour{
			C: &mapStructTestFourC{
				ACLSet: map[string]*mapStructTestFourCACLSet{
					"n42": {Name: String("n42"), SecondValue: String("val")},
				},
			},
		},
		wantJSONPath: filepath.Join(TestRoot, "testdata/emitjson_2.json-txt"),
	}, {
		name: "simple schema IETF JSON output",
		inStruct: &mapStructTestOne{
			Child: &mapStructTestOneChild{
				FieldOne:  String("bar"),
				FieldTwo:  Uint32(84),
				FieldFive: Uint64(42),
			},
		},
		inConfig: &EmitJSONConfig{
			Format: RFC7951,
			RFC7951Config: &RFC7951JSONConfig{
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
					"n42": {Name: String("n42"), SecondValue: String("foo")},
				},
				OtherSet: map[ECTest]*mapStructTestFourCOtherSet{
					ECTestVALONE: {Name: ECTestVALONE},
					ECTestVALTWO: {Name: ECTestVALTWO},
				},
			},
		},
		inConfig: &EmitJSONConfig{
			Format: RFC7951,
			RFC7951Config: &RFC7951JSONConfig{
				AppendModuleName: true,
			},
			Indent: "  ",
		},
		wantJSONPath: filepath.Join(TestRoot, "testdata/emitjson2_ietf.json-txt"),
	}, {
		name:     "invalid struct contents",
		inStruct: &mapStructInvalid{Name: String("aardvark")},
		wantErr:  "validation err: invalid",
	}, {
		name:     "invalid internal JSON",
		inStruct: &mapStructNoPaths{Name: String("honey badger")},
		wantErr:  "ConstructInternalJSON error: Name: field did not specify a path",
	}, {
		name:     "invalid RFC7951 JSON",
		inStruct: &mapStructNoPaths{Name: String("ladybird")},
		inConfig: &EmitJSONConfig{
			Format: RFC7951,
		},
		wantErr: "ConstructIETFJSON error: Name: field did not specify a path",
	}}

	for _, tt := range tests {
		got, err := EmitJSON(tt.inStruct, tt.inConfig)
		if errToString(err) != tt.wantErr {
			t.Errorf("%s: EmitJSON(%v, nil): did not get expected error, got: %v, want: %v", tt.name, tt.inStruct, err, tt.wantErr)
			continue
		}

		if tt.wantErr != "" {
			continue
		}

		wantJSON, ioerr := ioutil.ReadFile(tt.wantJSONPath)
		if ioerr != nil {
			t.Errorf("%s: ioutil.ReadFile(%s): could not open file: %v", tt.name, tt.wantJSONPath, err)
			continue
		}

		if diff := pretty.Compare(got, string(wantJSON)); diff != "" {
			if diffl, err := generateUnifiedDiff(got, string(wantJSON)); err == nil {
				diff = diffl
			}
			t.Errorf("%s: EmitJSON(%v, nil): got invalid JSON, diff(-got,+want):\n%s", tt.name, tt.inStruct, diff)
		}
	}
}

// emptyTreeTestOne is a test case for TestBuildEmptyTree.
type emptyTreeTestOne struct {
	ValOne   *string
	ValTwo   *string
	ValThree *int32
}

// IsYANGGoStruct ensures that emptyTreeTestOne implements the GoStruct interface
func (*emptyTreeTestOne) IsYANGGoStruct() {}

// emptyTreeTestTwo is a test case for TestBuildEmptyTree
type emptyTreeTestTwo struct {
	SliceVal  []*emptyTreeTestTwoChild
	MapVal    map[string]*emptyTreeTestTwoChild
	StructVal *emptyTreeTestTwoChild
}

// IsYANGGoStruct ensures that emptyTreeTestTwo implements the GoStruct interface
func (*emptyTreeTestTwo) IsYANGGoStruct() {}

// emptyTreeTestTwoChild is used in the TestBuildEmptyTree emptyTreeTestTwo structs.
type emptyTreeTestTwoChild struct {
	Val string
}

func TestBuildEmptyTree(t *testing.T) {
	tests := []struct {
		name     string
		inStruct GoStruct
		want     GoStruct
	}{{
		name:     "struct with no children",
		inStruct: &emptyTreeTestOne{},
		want:     &emptyTreeTestOne{},
	}, {
		name:     "struct with children",
		inStruct: &emptyTreeTestTwo{},
		want: &emptyTreeTestTwo{
			SliceVal:  []*emptyTreeTestTwoChild{},
			MapVal:    map[string]*emptyTreeTestTwoChild{},
			StructVal: &emptyTreeTestTwoChild{},
		},
	}}

	for _, tt := range tests {
		BuildEmptyTree(tt.inStruct)
		if diff := pretty.Compare(tt.inStruct, tt.want); diff != "" {
			t.Errorf("%s: did not get expected output, diff(-got,+want):\n%s", tt.name, diff)
		}
	}
}

// initContainerTest is a synthesised GoStruct for use in
// testing InitContainer.
type initContainerTest struct {
	StringVal    *string
	ContainerVal *initContainerTestChild
}

// IsYANGGoStruct ensures that the GoStruct interface is implemented
// for initContainerTest.
func (*initContainerTest) IsYANGGoStruct() {}

// initContainerTestChild is a synthesised GoStruct for use
// as a child of initContainerTest, and used in testing
// InitContainer.
type initContainerTestChild struct {
	Val *string
}

// IsYANGGoStruct ensures that the GoStruct interface is implemented
// for initContainerTestChild.
func (*initContainerTestChild) IsYANGGoStruct() {}

func TestInitContainer(t *testing.T) {
	tests := []struct {
		name            string
		inStruct        GoStruct
		inContainerName string
		want            GoStruct
		wantErr         bool
	}{{
		name:            "initialise existing field",
		inStruct:        &initContainerTest{},
		inContainerName: "ContainerVal",
		want:            &initContainerTest{ContainerVal: &initContainerTestChild{}},
	}, {
		name:            "initialise non-container field",
		inStruct:        &initContainerTest{},
		inContainerName: "StringVal",
		wantErr:         true,
	}, {
		name:            "initialise non-existent field",
		inStruct:        &initContainerTest{},
		inContainerName: "Fish",
		wantErr:         true,
	}}

	for _, tt := range tests {
		if err := InitContainer(tt.inStruct, tt.inContainerName); err != nil {
			if !tt.wantErr {
				t.Errorf("%s: InitContainer(%v): got unexpected error: %v", tt.name, tt.inStruct, err)
			}
			continue
		}

		if diff := pretty.Compare(tt.inStruct, tt.want); diff != "" {
			t.Errorf("%s: InitContainer(...): did not get expected output, diff(-got,+want):\n%s", tt.name, diff)
		}
	}
}

func TestMergeJSON(t *testing.T) {
	tests := []struct {
		name    string
		inA     map[string]interface{}
		inB     map[string]interface{}
		want    map[string]interface{}
		wantErr bool
	}{{
		name: "simple maps",
		inA:  map[string]interface{}{"a": 1},
		inB:  map[string]interface{}{"b": 2},
		want: map[string]interface{}{"a": 1, "b": 2},
	}, {
		name: "non-overlapping multi-layer tree",
		inA: map[string]interface{}{
			"a": map[string]interface{}{
				"a1": 42,
			},
			"aa": map[string]interface{}{
				"aa2": 84,
			},
		},
		inB: map[string]interface{}{
			"b": map[string]interface{}{
				"b1": 42,
			},
			"bb": map[string]interface{}{
				"bb2": 84,
			},
		},
		want: map[string]interface{}{
			"a": map[string]interface{}{
				"a1": 42,
			},
			"aa": map[string]interface{}{
				"aa2": 84,
			},
			"b": map[string]interface{}{
				"b1": 42,
			},
			"bb": map[string]interface{}{
				"bb2": 84,
			},
		},
	}, {
		name: "overlapping trees",
		inA: map[string]interface{}{
			"a": map[string]interface{}{
				"b": "c",
			},
		},
		inB: map[string]interface{}{
			"a": map[string]interface{}{
				"c": "d",
			},
		},
		want: map[string]interface{}{
			"a": map[string]interface{}{
				"b": "c",
				"c": "d",
			},
		},
	}, {
		name: "slice within json",
		inA: map[string]interface{}{
			"a": []interface{}{
				map[string]interface{}{"a": "a"},
			},
		},
		inB: map[string]interface{}{
			"a": []interface{}{
				map[string]interface{}{"b": "b"},
			},
		},
		want: map[string]interface{}{
			"a": []interface{}{
				map[string]interface{}{"a": "a"},
				map[string]interface{}{"b": "b"},
			},
		},
	}, {
		name: "slice value",
		inA: map[string]interface{}{
			"a": []interface{}{"a"},
		},
		inB: map[string]interface{}{
			"a": []interface{}{"b"},
		},
		want: map[string]interface{}{
			"a": []interface{}{"a", "b"},
		},
	}, {
		name: "scalar value",
		inA: map[string]interface{}{
			"a": "a",
		},
		inB: map[string]interface{}{
			"a": "b",
		},
		wantErr: true,
	}, {
		name: "different depth trees",
		inA: map[string]interface{}{
			"a": map[string]interface{}{
				"a1": map[string]interface{}{
					"a2": map[string]interface{}{
						"a3": 42,
					},
				},
			},
			"b": map[string]interface{}{
				"a1": map[string]interface{}{
					"a2": 42,
				},
			},
		},
		inB: map[string]interface{}{
			"a": map[string]interface{}{
				"b1": true,
			},
			"b": map[string]interface{}{
				"b2": 84,
				"b3": map[string]interface{}{
					"b4": map[string]interface{}{
						"b5": true,
					},
				},
			},
		},
		want: map[string]interface{}{
			"a": map[string]interface{}{
				"a1": map[string]interface{}{
					"a2": map[string]interface{}{
						"a3": 42,
					},
				},
				"b1": true,
			},
			"b": map[string]interface{}{
				"a1": map[string]interface{}{
					"a2": 42,
				},
				"b2": 84,
				"b3": map[string]interface{}{
					"b4": map[string]interface{}{
						"b5": true,
					},
				},
			},
		},
	}}

	for _, tt := range tests {
		got, err := MergeJSON(tt.inA, tt.inB)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: MergeJSON(%v, %v): did not get expected error, got: %v, want: %v", tt.name, tt.inA, tt.inB, err, tt.wantErr)
			continue
		}

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("%s: MergeJSON(%v, %v): did not get expected merged JSON, diff(-got,+want):\n%s", tt.name, tt.inA, tt.inB, diff)
		}
	}
}

type mergeTest struct {
	FieldOne    *string                        `path:"field-one" module:"mod"`
	FieldTwo    *uint8                         `path:"field-two" module:"mod"`
	LeafList    []string                       `path:"leaf-list" module:"leaflist"`
	UnkeyedList []*mergeTestListChild          `path:"unkeyed-list" module:"bar"`
	List        map[string]*mergeTestListChild `path:"list" module:"bar"`
}

func (*mergeTest) IsYANGGoStruct() {}

type mergeTestListChild struct {
	Val *string `path:"val" module:"mod"`
}

func (*mergeTestListChild) IsYANGGoStruct() {}

func TestMergeStructJSON(t *testing.T) {
	tests := []struct {
		name     string
		inStruct GoStruct
		inJSON   map[string]interface{}
		inOpts   *EmitJSONConfig
		wantJSON map[string]interface{}
		wantErr  bool
	}{{
		name:     "single field merge test, internal format",
		inStruct: &mergeTest{FieldOne: String("hello")},
		inJSON: map[string]interface{}{
			"field-two": "world",
		},
		wantJSON: map[string]interface{}{
			"field-one": "hello",
			"field-two": "world",
		},
	}, {
		name:     "single field merge test, RFC7951 format",
		inStruct: &mergeTest{FieldOne: String("hello")},
		inJSON: map[string]interface{}{
			"mod:field-two": "world",
		},
		inOpts: &EmitJSONConfig{
			Format: RFC7951,
			RFC7951Config: &RFC7951JSONConfig{
				AppendModuleName: true,
			},
		},
		wantJSON: map[string]interface{}{
			"mod:field-one": "hello",
			"mod:field-two": "world",
		},
	}, {
		name: "leaf-list field, present in only one message, internal JSON",
		inStruct: &mergeTest{
			FieldOne: String("hello"),
			LeafList: []string{"me", "you're", "looking", "for"},
		},
		inJSON: map[string]interface{}{
			"leaf-list": []interface{}{"is", "it"},
		},
		wantJSON: map[string]interface{}{
			"field-one": "hello",
			"leaf-list": []interface{}{"is", "it", "me", "you're", "looking", "for"},
		},
	}, {
		name: "unkeyed list merge",
		inStruct: &mergeTest{
			UnkeyedList: []*mergeTestListChild{{String("in")}, {String("the")}, {String("jar")}},
		},
		inJSON: map[string]interface{}{
			"unkeyed-list": []interface{}{
				map[string]interface{}{"val": "whisky"},
			},
		},
		inOpts: &EmitJSONConfig{
			Format: RFC7951,
		},
		wantJSON: map[string]interface{}{
			"unkeyed-list": []interface{}{
				map[string]interface{}{"val": "whisky"},
				map[string]interface{}{"val": "in"},
				map[string]interface{}{"val": "the"},
				map[string]interface{}{"val": "jar"},
			},
		},
	}, {
		name: "keyed list, RFC7951 JSON",
		inStruct: &mergeTest{
			List: map[string]*mergeTestListChild{
				"anjou":  {String("anjou")},
				"chinon": {String("chinon")},
			},
		},
		inJSON: map[string]interface{}{
			"list": []interface{}{
				map[string]interface{}{"val": "sancerre"},
			},
		},
		inOpts: &EmitJSONConfig{
			Format: RFC7951,
		},
		wantJSON: map[string]interface{}{
			"list": []interface{}{
				map[string]interface{}{"val": "sancerre"},
				map[string]interface{}{"val": "anjou"},
				map[string]interface{}{"val": "chinon"},
			},
		},
	}, {
		name: "keyed list, internal JSON",
		inStruct: &mergeTest{
			List: map[string]*mergeTestListChild{
				"bandol": {String("bandol")},
			},
		},
		inJSON: map[string]interface{}{
			"list": map[string]interface{}{
				"bellet": map[string]interface{}{
					"val": "bellet",
				},
			},
		},
		wantJSON: map[string]interface{}{
			"list": map[string]interface{}{
				"bellet": map[string]interface{}{"val": "bellet"},
				"bandol": map[string]interface{}{"val": "bandol"},
			},
		},
	}, {
		name:     "overlapping trees",
		inStruct: &mergeTest{FieldOne: String("foo")},
		inJSON:   map[string]interface{}{"field-one": "bar"},
		wantErr:  true,
	}}

	for _, tt := range tests {
		got, err := MergeStructJSON(tt.inStruct, tt.inJSON, tt.inOpts)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: MergeStructJSON(%v, %v, %v): did not get expected error status, got: %v, want: %v", tt.name, tt.inStruct, tt.inJSON, tt.inOpts, err, tt.wantErr)
		}

		if diff := pretty.Compare(got, tt.wantJSON); diff != "" {
			t.Errorf("%s: MergeStrucTJSON(%v, %v, %v): did not get expected error status, diff(-got,+want):\n%s", tt.name, tt.inStruct, tt.inJSON, tt.inOpts, diff)
		}
	}
}

// Types for testing copyStruct.
type enumType int64

const (
	EnumTypeValue enumType = 1
)

type copyUnion interface {
	IsUnion()
}

type copyUnionS struct {
	S string
}

func (*copyUnionS) IsUnion() {}

type copyMapKey struct {
	A string
}

type copyTest struct {
	StringField   *string
	Uint32Field   *uint32
	Uint16Field   *uint16
	Float64Field  *float64
	StructPointer *copyTest
	EnumValue     enumType
	UnionField    copyUnion
	StringSlice   []string
	StringMap     map[string]*copyTest
	StructMap     map[copyMapKey]*copyTest
	StructSlice   []*copyTest
}

func (*copyTest) IsYANGGoStruct() {}

type errorCopyTest struct {
	I interface{}
	S *string
	M map[string]errorCopyTest
	N map[string]*errorCopyTest
	E *errorCopyTest
	L []*errorCopyTest
}

func (*errorCopyTest) IsYANGGoStruct() {}

func TestCopyStructError(t *testing.T) {
	// Checks specifically for bad reflect.Values being provided.
	tests := []struct {
		name string
		inA  reflect.Value
		inB  reflect.Value
	}{{
		name: "non-struct pointer",
		inA:  reflect.ValueOf(String("little-creatures-pale-ale")),
		inB:  reflect.ValueOf(String("4-pines-brewing-kolsch")),
	}, {
		name: "non-pointer",
		inA:  reflect.ValueOf("4-pines-indian-summer-ale"),
		inB:  reflect.ValueOf("james-squire-150-lashes"),
	}}

	for _, tt := range tests {
		if err := copyStruct(tt.inA, tt.inB); err == nil {
			t.Errorf("%s: copyStruct(%v, %v): did not get nil error, got: %v, want: nil", tt.name, tt.inA, tt.inB, err)
		}
	}
}

func TestCopyStruct(t *testing.T) {
	tests := []struct {
		name    string
		inSrc   GoStruct
		inDst   GoStruct
		wantDst GoStruct
		wantErr bool
	}{{
		name:    "simple string pointer",
		inSrc:   &copyTest{StringField: String("anchor-steam")},
		inDst:   &copyTest{},
		wantDst: &copyTest{StringField: String("anchor-steam")},
	}, {
		name: "uint and string pointer",
		inSrc: &copyTest{
			StringField: String("fourpure-juicebox"),
			Uint32Field: Uint32(42),
		},
		inDst: &copyTest{},
		wantDst: &copyTest{
			StringField: String("fourpure-juicebox"),
			Uint32Field: Uint32(42),
		},
	}, {
		name: "struct pointer with single field",
		inSrc: &copyTest{
			StringField: String("lagunitas-aunt-sally"),
			StructPointer: &copyTest{
				StringField: String("deschutes-pinedrops"),
			},
		},
		inDst: &copyTest{},
		wantDst: &copyTest{
			StringField: String("lagunitas-aunt-sally"),
			StructPointer: &copyTest{
				StringField: String("deschutes-pinedrops"),
			},
		},
	}, {
		name: "struct pointer with multiple fields",
		inSrc: &copyTest{
			StringField: String("allagash-brett"),
			Uint32Field: Uint32(84),
			StructPointer: &copyTest{
				StringField: String("brooklyn-summer-ale"),
				Uint32Field: Uint32(128),
			},
		},
		inDst: &copyTest{},
		wantDst: &copyTest{
			StringField: String("allagash-brett"),
			Uint32Field: Uint32(84),
			StructPointer: &copyTest{
				StringField: String("brooklyn-summer-ale"),
				Uint32Field: Uint32(128),
			},
		},
	}, {
		name: "enum value",
		inSrc: &copyTest{
			EnumValue: EnumTypeValue,
		},
		inDst: &copyTest{},
		wantDst: &copyTest{
			EnumValue: EnumTypeValue,
		},
	}, {
		name: "union field",
		inSrc: &copyTest{
			UnionField: &copyUnionS{"new-belgium-fat-tire"},
		},
		inDst: &copyTest{},
		wantDst: &copyTest{
			UnionField: &copyUnionS{"new-belgium-fat-tire"},
		},
	}, {
		name: "string slice",
		inSrc: &copyTest{
			StringSlice: []string{"sierra-nevada-pale-ale", "stone-ipa"},
		},
		inDst: &copyTest{},
		wantDst: &copyTest{
			StringSlice: []string{"sierra-nevada-pale-ale", "stone-ipa"},
		},
	}, {
		name: "unimplemented string slice with existing members",
		inSrc: &copyTest{
			StringSlice: []string{"stone-and-wood-pacific", "pirate-life-brewing-iipa"},
		},
		inDst: &copyTest{
			StringSlice: []string{"feral-brewing-co-hop-hog", "balter-brewing-xpa"},
		},
		wantErr: true, // Input combination not supported, destination slice must be nil.
	}, {
		name: "string map",
		inSrc: &copyTest{
			StringMap: map[string]*copyTest{
				"ballast-point": {StringField: String("sculpin")},
				"upslope":       {StringSlice: []string{"amber-ale", "brown"}},
			},
		},
		inDst: &copyTest{},
		wantDst: &copyTest{
			StringMap: map[string]*copyTest{
				"ballast-point": {StringField: String("sculpin")},
				"upslope":       {StringSlice: []string{"amber-ale", "brown"}},
			},
		},
	}, {
		name: "string map with existing members",
		inSrc: &copyTest{
			StringMap: map[string]*copyTest{
				"bentspoke-brewing": {StringField: String("crankshaft")},
			},
		},
		inDst: &copyTest{
			StringMap: map[string]*copyTest{
				"modus-operandi-brewing-co": {StringField: String("former-tenant")},
			},
		},
		wantDst: &copyTest{
			StringMap: map[string]*copyTest{
				"bentspoke-brewing":         {StringField: String("crankshaft")},
				"modus-operandi-brewing-co": {StringField: String("former-tenant")},
			},
		},
	}, {
		name: "unimplemented, string map with overlapping members",
		inSrc: &copyTest{
			StringMap: map[string]*copyTest{
				"wild-beer-co": {StringField: String("wild-goose-chase")},
			},
		},
		inDst: &copyTest{
			StringMap: map[string]*copyTest{
				"wild-beer-co": {StringField: String("wildebeest")},
			},
		},
		wantErr: true, // Maps with matching keys are currently not merged.
	}, {
		name: "struct map",
		inSrc: &copyTest{
			StructMap: map[copyMapKey]*copyTest{
				{"saint-arnold"}: {StringField: String("fancy-lawnmower")},
				{"green-flash"}:  {StringField: String("hop-head-red")},
			},
		},
		inDst: &copyTest{},
		wantDst: &copyTest{
			StructMap: map[copyMapKey]*copyTest{
				{"saint-arnold"}: {StringField: String("fancy-lawnmower")},
				{"green-flash"}:  {StringField: String("hop-head-red")},
			},
		},
	}, {
		name: "struct map with non-overlapping contents",
		inSrc: &copyTest{
			StructMap: map[copyMapKey]*copyTest{
				{"brewdog"}: {StringField: String("kingpin")},
			},
		},
		inDst: &copyTest{
			StructMap: map[copyMapKey]*copyTest{
				{"cheshire-brewhouse"}: {StringField: String("dane'ish")},
			},
		},
		wantDst: &copyTest{
			StructMap: map[copyMapKey]*copyTest{
				{"brewdog"}:            {StringField: String("kingpin")},
				{"cheshire-brewhouse"}: {StringField: String("dane'ish")},
			},
		},
	}, {
		name: "struct map with overlapping contents",
		inSrc: &copyTest{
			StructMap: map[copyMapKey]*copyTest{
				{"fourpure"}: {StringField: String("session-ipa")},
			},
		},
		inDst: &copyTest{
			StructMap: map[copyMapKey]*copyTest{
				{"fourpure"}: {
					Uint32Field:  Uint32(42),
					Uint16Field:  Uint16(16),
					Float64Field: Float64(42.42),
				},
			},
		},
		wantDst: &copyTest{
			StructMap: map[copyMapKey]*copyTest{
				{"fourpure"}: {
					StringField:  String("session-ipa"),
					Uint32Field:  Uint32(42),
					Uint16Field:  Uint16(16),
					Float64Field: Float64(42.42),
				},
			},
		},
	}, {
		name: "struct map with overlapping fields within the same key",
		inSrc: &copyTest{
			StructMap: map[copyMapKey]*copyTest{
				{"new-belgium"}: {StringField: String("voodoo-ranger")},
			},
		},
		inDst: &copyTest{
			StructMap: map[copyMapKey]*copyTest{
				{"new-belgium"}: {StringField: String("fat-tire")},
			},
		},
		wantErr: true,
	}, {
		name: "struct slice",
		inSrc: &copyTest{
			StructSlice: []*copyTest{{
				StringField: String("russian-river-pliny-the-elder"),
			}, {
				StringField: String("lagunitas-brown-shugga"),
			}},
		},
		inDst: &copyTest{},
		wantDst: &copyTest{
			StructSlice: []*copyTest{{
				StringField: String("russian-river-pliny-the-elder"),
			}, {
				StringField: String("lagunitas-brown-shugga"),
			}},
		},
	}, {
		name: "unimplemented: struct slice with overlapping contents",
		inSrc: &copyTest{
			StructSlice: []*copyTest{{
				StringField: String("pirate-life-brewing-ipa"),
			}},
		},
		inDst: &copyTest{
			StructSlice: []*copyTest{{
				StringField: String("gage-roads-little-dove"),
			}},
		},
		wantErr: true, // Input combination unimplemented, destination slice must be nil.
	}, {
		name:    "error, integer in interface",
		inSrc:   &errorCopyTest{I: 42},
		inDst:   &errorCopyTest{},
		wantErr: true,
	}, {
		name:    "error, integer pointer in interface",
		inSrc:   &errorCopyTest{I: Uint32(42)},
		inDst:   &errorCopyTest{},
		wantErr: true,
	}, {
		name:    "error, invalid interface in struct within interface",
		inSrc:   &errorCopyTest{I: &errorCopyTest{I: "founders-kbs"}},
		inDst:   &errorCopyTest{},
		wantErr: true,
	}, {
		name: "error, invalid struct in map",
		inSrc: &errorCopyTest{M: map[string]errorCopyTest{
			"beaver-town-gamma-ray": {S: String("beaver-town-black-betty-ipa")},
		}},
		inDst:   &errorCopyTest{},
		wantErr: true,
	}, {
		name: "error, invalid field in struct in map",
		inSrc: &errorCopyTest{N: map[string]*errorCopyTest{
			"brewdog-punk-ipa": {I: "harbour-amber-ale"},
		}},
		inDst:   &errorCopyTest{},
		wantErr: true,
	}, {
		name:    "error, invalid field in struct in struct ptr",
		inSrc:   &errorCopyTest{E: &errorCopyTest{I: "meantime-wheat"}},
		inDst:   &errorCopyTest{},
		wantErr: true,
	}, {
		name:    "error, invalid struct in struct ptr slice",
		inSrc:   &errorCopyTest{L: []*errorCopyTest{{I: "wild-beer-co-somerset-wild"}}},
		inDst:   &errorCopyTest{},
		wantErr: true,
	}, {
		name:    "error, mismatched types",
		inSrc:   &copyTest{StringField: String("camden-hells")},
		inDst:   &errorCopyTest{S: String("kernel-table-beer")},
		wantErr: true,
	}}

	for _, tt := range tests {
		dst, src := reflect.ValueOf(tt.inDst).Elem(), reflect.ValueOf(tt.inSrc).Elem()
		var wantDst reflect.Value
		if tt.wantDst != nil {
			wantDst = reflect.ValueOf(tt.wantDst).Elem()
		}

		err := copyStruct(dst, src)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: copyStruct(%v, %v): did not get expected error, got: %v, wantErr: %v", tt.name, tt.inSrc, tt.inDst, err, tt.wantErr)
		}

		if err != nil {
			continue
		}

		if diff := pretty.Compare(dst.Interface(), wantDst.Interface()); diff != "" {
			t.Errorf("%s: copyStruct(%v, %v): did not get expected copied struct, diff(-got,+want):\n%s", tt.name, tt.inSrc, tt.inDst, diff)
		}
	}
}

type validatedMergeTest struct {
	String      *string
	StringTwo   *string
	Uint32Field *uint32
}

func (*validatedMergeTest) Validate(...ValidationOption) error      { return nil }
func (*validatedMergeTest) IsYANGGoStruct()                         {}
func (*validatedMergeTest) ΛEnumTypeMap() map[string][]reflect.Type { return nil }

type validatedMergeTestTwo struct {
	String *string
	I      interface{}
}

func (*validatedMergeTestTwo) Validate(...ValidationOption) error      { return nil }
func (*validatedMergeTestTwo) IsYANGGoStruct()                         {}
func (*validatedMergeTestTwo) ΛEnumTypeMap() map[string][]reflect.Type { return nil }

func TestMergeStructs(t *testing.T) {
	tests := []struct {
		name    string
		inA     ValidatedGoStruct
		inB     ValidatedGoStruct
		want    ValidatedGoStruct
		wantErr string
	}{{
		name: "simple struct merge, a empty",
		inA:  &validatedMergeTest{},
		inB:  &validatedMergeTest{String: String("odell-90-shilling")},
		want: &validatedMergeTest{String: String("odell-90-shilling")},
	}, {
		name: "simple struct merge, a populated",
		inA:  &validatedMergeTest{String: String("left-hand-milk-stout-nitro"), Uint32Field: Uint32(42)},
		inB:  &validatedMergeTest{StringTwo: String("new-belgium-lips-of-faith-la-folie")},
		want: &validatedMergeTest{
			String:      String("left-hand-milk-stout-nitro"),
			StringTwo:   String("new-belgium-lips-of-faith-la-folie"),
			Uint32Field: Uint32(42),
		},
	}, {
		name:    "error, differing types",
		inA:     &validatedMergeTest{String: String("great-divide-yeti")},
		inB:     &validatedMergeTestTwo{String: String("north-coast-old-rasputin")},
		wantErr: "cannot merge structs that are not of matching types, *ygot.validatedMergeTest != *ygot.validatedMergeTestTwo",
	}, {
		name:    "error, bad data in A",
		inA:     &validatedMergeTestTwo{I: "belleville-thames-surfer"},
		inB:     &validatedMergeTestTwo{String: String("fourpure-beartooth")},
		wantErr: "cannot DeepCopy struct: invalid interface type received: string",
	}, {
		name:    "error, bad data in B",
		inA:     &validatedMergeTestTwo{String: String("weird-beard-sorachi-faceplant")},
		inB:     &validatedMergeTestTwo{I: "fourpure-southern-latitude"},
		wantErr: "error merging b to new struct: invalid interface type received: string",
	}, {
		name:    "error, field set in both structs",
		inA:     &validatedMergeTest{String: String("karbach-hopadillo")},
		inB:     &validatedMergeTest{String: String("blackwater-draw-brewing-co-border-town")},
		wantErr: "error merging b to new struct: destination value was set when merging, src: blackwater-draw-brewing-co-border-town, dst: karbach-hopadillo",
	}}

	for _, tt := range tests {
		got, err := MergeStructs(tt.inA, tt.inB)
		if err != nil && err.Error() != tt.wantErr {
			t.Errorf("%s: MergeStructs(%v, %v): did not get expected error status, got: %v, want: %v", tt.name, tt.inA, tt.inB, err, tt.wantErr)
		}

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("%s: MergeStructs(%v, %v): did not get expected returned struct, diff(-got,+want):\n%s", tt.name, tt.inA, tt.inB, diff)
		}
	}
}

func TestValidateMap(t *testing.T) {
	tests := []struct {
		name        string
		inSrc       reflect.Value
		inDst       reflect.Value
		wantMapType *mapType
		wantErr     string
	}{{
		name:  "valid maps",
		inSrc: reflect.ValueOf(map[string]*copyTest{}),
		inDst: reflect.ValueOf(map[string]*copyTest{}),
		wantMapType: &mapType{
			key:   reflect.TypeOf(""),
			value: reflect.TypeOf(&copyTest{}),
		},
	}, {
		name:    "invalid src field, not a map",
		inSrc:   reflect.ValueOf(""),
		inDst:   reflect.ValueOf(map[string]*copyTest{}),
		wantErr: "invalid src field, was not a map, was: string",
	}, {
		name:    "invalid dst field, not a map",
		inSrc:   reflect.ValueOf(map[string]*copyTest{}),
		inDst:   reflect.ValueOf(uint32(42)),
		wantErr: "invalid dst field, was not a map, was: uint32",
	}, {
		name:    "invalid src and dst fields, do not have the same value type",
		inSrc:   reflect.ValueOf(map[string]string{}),
		inDst:   reflect.ValueOf(map[string]uint32{}),
		wantErr: "invalid maps, src and dst value types are different, string != uint32",
	}, {
		name:    "invalid src and dst field, not a struct ptr",
		inSrc:   reflect.ValueOf(map[string]copyTest{}),
		inDst:   reflect.ValueOf(map[string]copyTest{}),
		wantErr: "invalid maps, src or dst does not have a struct ptr element, src: struct, dst: struct",
	}, {
		name:    "invalid maps, src and dst key types differ",
		inSrc:   reflect.ValueOf(map[string]*copyTest{}),
		inDst:   reflect.ValueOf(map[uint32]*copyTest{}),
		wantErr: "invalid maps, src and dst key types are different, string != uint32",
	}}

	for _, tt := range tests {
		got, err := validateMap(tt.inSrc, tt.inDst)
		if err != nil {
			if err.Error() != tt.wantErr {
				t.Errorf("%s: validateMap(%v, %v): did not get expected error status, got: %v, wantErr: %v", tt.name, tt.inSrc, tt.inDst, err, tt.wantErr)
			}
			continue
		}

		if diff := pretty.Compare(got, tt.wantMapType); diff != "" {
			t.Errorf("%s: validateMap(%v, %v): did not get expected return mapType, diff(-got,+want):\n%s", tt.name, tt.inSrc, tt.inDst, diff)
		}
	}
}

func TestCopyErrorCases(t *testing.T) {
	type errorTest struct {
		name    string
		inSrc   reflect.Value
		inDst   reflect.Value
		wantErr string
	}

	mapErrs := []errorTest{
		{"bad src", reflect.ValueOf(""), reflect.ValueOf(map[string]string{}), "received a non-map type in src map field: string"},
		{"bad dst", reflect.ValueOf(map[string]string{}), reflect.ValueOf(uint32(42)), "received a non-map type in dst map field: uint32"},
	}
	for _, tt := range mapErrs {
		if err := copyMapField(tt.inDst, tt.inSrc); err == nil || err.Error() != tt.wantErr {
			t.Errorf("%s: copyMapField(%v, %v): did not get expected error, got: %v, want: %v", tt.name, tt.inSrc, tt.inDst, err, tt.wantErr)
		}
	}

	ptrErrs := []errorTest{
		{"non-ptr", reflect.ValueOf(""), reflect.ValueOf(""), "received non-ptr type: string"},
	}
	for _, tt := range ptrErrs {
		if err := copyPtrField(tt.inDst, tt.inSrc); err == nil || err.Error() != tt.wantErr {
			t.Errorf("%s: copyPtrField(%v, %v): did not get expected error, got: %v, want: %v", tt.name, tt.inSrc, tt.inDst, err, tt.wantErr)
		}
	}

	badDeepCopy := &errorCopyTest{I: "foobar"}
	wantBDCErr := "cannot DeepCopy struct: invalid interface type received: string"
	if _, err := DeepCopy(badDeepCopy); err == nil || err.Error() != wantBDCErr {
		t.Errorf("badDeepCopy: DeepCopy(%v): did not get expected error, got: %v, want: %v", badDeepCopy, err, wantBDCErr)
	}
}

func TestDeepCopy(t *testing.T) {
	tests := []struct {
		name    string
		in      *copyTest
		inKey   string
		wantErr bool
	}{{
		name: "simple copy",
		in:   &copyTest{StringField: String("zaphod")},
	}, {
		name: "copy with map",
		in: &copyTest{
			StringMap: map[string]*copyTest{
				"just": {StringField: String("this guy")},
			},
		},
		inKey: "just",
	}, {
		name: "copy with slice",
		in: &copyTest{
			StringSlice: []string{"one"},
		},
	}}

	for _, tt := range tests {
		got, err := DeepCopy(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: DeepCopy(%#v): did not get expected error, got: %v, wantErr: %v", tt.name, tt.in, err, tt.wantErr)
		}
		if diff := pretty.Compare(got, tt.in); diff != "" {
			t.Errorf("%s: DeepCopy(%#v): did not get identical copy, diff(-got,+want):\n%s", tt.name, tt.in, diff)
		}

		// Check we got a copy that doesn't modify the original.
		gotC, ok := got.(*copyTest)
		if !ok {
			t.Errorf("%s: DeepCopy(%#v): did not get back the same type, got: %T, want: %T", tt.name, tt.in, got, tt.in)
		}

		if &gotC == &tt.in {
			t.Errorf("%s: DeepCopy(%#v): after copy, input and copy have same memory address: %v", tt.name, tt.in, &gotC)
		}

		if len(tt.in.StringMap) != 0 && tt.inKey != "" {
			if &tt.in.StringMap == &gotC.StringMap {
				t.Errorf("%s: DeepCopy(%#v): after copy, input map and copied map have the same address: %v", tt.name, tt.in, &gotC.StringMap)
			}

			if v, ok := tt.in.StringMap[tt.inKey]; ok {
				cv, cok := gotC.StringMap[tt.inKey]
				if !cok {
					t.Errorf("%s: DeepCopy(%#v): after copy, received map did not have correct key, want key: %v, got: %v", tt.name, tt.in, tt.inKey, gotC.StringMap)
				}

				if &v == &cv {
					t.Errorf("%s: DeepCopy(%#v): after copy, input map element and copied map element have the same address: %v", tt.name, tt.in, &cv)
				}
			}
		}

		if len(tt.in.StringSlice) != 0 {
			if &tt.in.StringSlice == &gotC.StringSlice {
				t.Errorf("%s: DeepCopy(%#v): after copy, input slice and copied slice have the same address: %v", tt.name, tt.in, &gotC.StringSlice)
			}
		}
	}
}

type buildEmptyTreeMergeTest struct {
	Son      *buildEmptyTreeMergeTestChild
	Daughter *buildEmptyTreeMergeTestChild
	String   *string
}

func (*buildEmptyTreeMergeTest) Validate(...ValidationOption) error      { return nil }
func (*buildEmptyTreeMergeTest) IsYANGGoStruct()                         {}
func (*buildEmptyTreeMergeTest) ΛEnumTypeMap() map[string][]reflect.Type { return nil }

type buildEmptyTreeMergeTestChild struct {
	Grandson      *buildEmptyTreeMergeTestGrandchild
	Granddaughter *buildEmptyTreeMergeTestGrandchild
	String        *string
}

func (*buildEmptyTreeMergeTestChild) Validate(...ValidationOption) error      { return nil }
func (*buildEmptyTreeMergeTestChild) IsYANGGoStruct()                         {}
func (*buildEmptyTreeMergeTestChild) ΛEnumTypeMap() map[string][]reflect.Type { return nil }

type buildEmptyTreeMergeTestGrandchild struct {
	String *string
}

func (*buildEmptyTreeMergeTestGrandchild) Validate(...ValidationOption) error      { return nil }
func (*buildEmptyTreeMergeTestGrandchild) IsYANGGoStruct()                         {}
func (*buildEmptyTreeMergeTestGrandchild) ΛEnumTypeMap() map[string][]reflect.Type { return nil }

func TestBuildEmptyTreeMerge(t *testing.T) {
	tests := []struct {
		name        string
		inStructA   *buildEmptyTreeMergeTest
		inStructB   *buildEmptyTreeMergeTest
		inBuildSonA bool
		inBuildSonB bool
		want        ValidatedGoStruct
		wantErr     bool
	}{{
		name: "check with no build empty",
		inStructA: &buildEmptyTreeMergeTest{
			Son: &buildEmptyTreeMergeTestChild{
				String: String("blackwater-draw-brewing-co-contract-killer"),
			},
		},
		inStructB: &buildEmptyTreeMergeTest{
			Daughter: &buildEmptyTreeMergeTestChild{
				String: String("brazos-valley-brewing-7-spanish-angels"),
			},
		},
		want: &buildEmptyTreeMergeTest{
			Son: &buildEmptyTreeMergeTestChild{
				String: String("blackwater-draw-brewing-co-contract-killer"),
			},
			Daughter: &buildEmptyTreeMergeTestChild{
				String: String("brazos-valley-brewing-7-spanish-angels"),
			},
		},
	}, {
		name: "check with build empty on B",
		inStructA: &buildEmptyTreeMergeTest{
			Son: &buildEmptyTreeMergeTestChild{
				String: String("brazos-valley-brewing-mama-tried-ipa"),
				Grandson: &buildEmptyTreeMergeTestGrandchild{
					String: String("brazos-valley-brewing-killin'-time-blonde"),
				},
			},
		},
		inStructB: &buildEmptyTreeMergeTest{
			Daughter: &buildEmptyTreeMergeTestChild{
				String: String("brazos-valley-brewing-13th-can"),
				Granddaughter: &buildEmptyTreeMergeTestGrandchild{
					String: String("brazos-valley-brewing-silt-brown-ale"),
				},
			},
			Son: &buildEmptyTreeMergeTestChild{},
		},
		inBuildSonB: true,
		want: &buildEmptyTreeMergeTest{
			Son: &buildEmptyTreeMergeTestChild{
				String: String("brazos-valley-brewing-mama-tried-ipa"),
				Grandson: &buildEmptyTreeMergeTestGrandchild{
					String: String("brazos-valley-brewing-killin'-time-blonde"),
				},
				Granddaughter: &buildEmptyTreeMergeTestGrandchild{},
			},
			Daughter: &buildEmptyTreeMergeTestChild{
				String: String("brazos-valley-brewing-13th-can"),
				Granddaughter: &buildEmptyTreeMergeTestGrandchild{
					String: String("brazos-valley-brewing-silt-brown-ale"),
				},
			},
		},
	}, {
		name: "check with build empty on A",
		inStructA: &buildEmptyTreeMergeTest{
			Son: &buildEmptyTreeMergeTestChild{},
			Daughter: &buildEmptyTreeMergeTestChild{
				String: String("huff-brewing-orrange-blossom-saison"),
			},
		},
		inStructB: &buildEmptyTreeMergeTest{
			Son: &buildEmptyTreeMergeTestChild{
				String: String("brazos-valley-brewing-suma-babushka"),
				Grandson: &buildEmptyTreeMergeTestGrandchild{
					String: String("brazos-valley-brewing-big-spoon"),
				},
			},
		},
		inBuildSonA: true,
		want: &buildEmptyTreeMergeTest{
			Son: &buildEmptyTreeMergeTestChild{
				String: String("brazos-valley-brewing-suma-babushka"),
				Grandson: &buildEmptyTreeMergeTestGrandchild{
					String: String("brazos-valley-brewing-big-spoon"),
				},
				Granddaughter: &buildEmptyTreeMergeTestGrandchild{},
			},
			Daughter: &buildEmptyTreeMergeTestChild{
				String: String("huff-brewing-orrange-blossom-saison"),
			},
		},
	}}

	for _, tt := range tests {
		if tt.inBuildSonA {
			BuildEmptyTree(tt.inStructA.Son)
		}

		if tt.inBuildSonB {
			BuildEmptyTree(tt.inStructB.Son)
		}

		got, err := MergeStructs(tt.inStructA, tt.inStructB)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: MergeStructs(%v, %v): got unexpected error status, got: %v, wantErr: %v", tt.name, tt.inStructA, tt.inStructB, err, tt.wantErr)
		}
		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("%s: MergeStructs(%v, %v): did not get expected merge result, diff(-got,+want):\n%s", tt.name, tt.inStructA, tt.inStructB, diff)
		}

	}
}
