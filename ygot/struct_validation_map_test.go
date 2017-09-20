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
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/pmezard/go-difflib/difflib"
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

// mapStructTestOne is the base struct used for the simple-schema test.
type mapStructTestOne struct {
	Child *mapStructTestOneChild `path:"child" module:"test-one"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*mapStructTestOne) IsYANGGoStruct() {}

func (*mapStructTestOne) Validate() error {
	return nil
}

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

func (*mapStructTestOneChild) Validate() error {
	return nil
}

// mapStructTestFour is the top-level container used for the
// schema-with-list test.
type mapStructTestFour struct {
	C *mapStructTestFourC `path:"c"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*mapStructTestFour) IsYANGGoStruct() {}

func (*mapStructTestFour) Validate() error {
	return nil
}

// mapStructTestFourC is the "c" container used for the schema-with-list
// test.
type mapStructTestFourC struct {
	// ACLSet is a YANG list that is keyed with a string.
	ACLSet   map[string]*mapStructTestFourCACLSet   `path:"acl-set"`
	OtherSet map[ECTest]*mapStructTestFourCOtherSet `path:"other-set"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*mapStructTestFourC) IsYANGGoStruct() {}

func (*mapStructTestFourC) Validate() error {
	return nil
}

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

func (*mapStructTestFourCACLSet) Validate() error {
	return nil
}

// mapStructTestFourOtherSet is a map entry with a
type mapStructTestFourCOtherSet struct {
	Name ECTest `path:"config/name|name"`
}

// IsYANGGoStruct implements the GoStruct interface.
func (*mapStructTestFourCOtherSet) IsYANGGoStruct() {}

func (*mapStructTestFourCOtherSet) Validate() error {
	return nil
}

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
func (*mapStructInvalid) Validate() error {
	return fmt.Errorf("invalid")
}

// mapStructNoPaths is a valid GoStruct who does not implement path tags.
type mapStructNoPaths struct {
	Name *string
}

// IsYANGGoStruct implements the GoStruct interface.
func (*mapStructNoPaths) IsYANGGoStruct() {}

// Validate implements the ValidatedGoStruct interface.
func (*mapStructNoPaths) Validate() error { return nil }

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
