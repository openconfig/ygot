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

// mapStructTestOne is the base struct used for the simple-schema test.
type mapStructTestOne struct {
	Child *mapStructTestOneChild `path:"child"`
}

// IsYANGGoStruct makes sure that we implement the GoStruct interface.
func (*mapStructTestOne) IsYANGGoStruct() {}

func (*mapStructTestOne) Validate() error {
	return nil
}

// mapStructTestOne_Child is a child structure of the mapStructTestOne test
// case.
type mapStructTestOneChild struct {
	FieldOne   *string  `path:"config/field-one"`
	FieldTwo   *uint32  `path:"config/field-two"`
	FieldThree Binary   `path:"config/field-three"`
	FieldFour  []Binary `path:"config/field-four"`
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

// ΛMap returns the enumeration dictionary associated with the mapStructTestFiveC
// struct.
func (ECTest) ΛMap() map[string]map[int64]string {
	return map[string]map[int64]string{
		"ECTest": {
			1: "VAL_ONE",
			2: "VAL_TWO",
		},
	}
}

// TestEmitJSON validates that the EmitJSON function outputs the expected JSON
// for a set of input structs and schema definitions.
func TestEmitJSON(t *testing.T) {
	tests := []struct {
		name         string
		inStruct     ValidatedGoStruct
		wantJSONPath string
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
	}}
	// TODO(robjs): Add test cases for RFC7951 JSON output here.

	for _, tt := range tests {
		got, err := EmitJSON(tt.inStruct, nil)
		if err != nil {
			t.Errorf("%s: EmitJSON(%v, nil): got unexpected error: %v", tt.name, tt.inStruct, err)
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

type validKeyStruct struct {
	KeyOne string `path:"k1"`
	KeyTwo int64  `path:"k2"`
}

type emptyKeyStruct struct{}

type invalidKeyStruct struct {
	KeyOne, KeyTwo string
}

type longKeyStruct struct {
	KeyOne string `path:"foobar|foo"`
}

type multipartKeyStruct struct {
	KeyOne string `path:"config/foo"`
}

func TestKeyStructMap(t *testing.T) {
	tests := []struct {
		name    string
		in      interface{}
		want    map[string]interface{}
		wantErr bool
	}{{
		name: "simple key struct test",
		in: validKeyStruct{
			KeyOne: "one",
			KeyTwo: 42,
		},
		want: map[string]interface{}{
			"k1": "one",
			"k2": 42,
		},
	}, {
		name:    "empty key struct",
		in:      emptyKeyStruct{},
		wantErr: true,
	}, {
		name: "invalid key struct",
		in: invalidKeyStruct{
			KeyOne: "hello",
			KeyTwo: "world",
		},
		wantErr: true,
	}, {
		name:    "multi-path invalid key struct",
		in:      longKeyStruct{"foo"},
		wantErr: true,
	}, {
		name:    "multi-part single path invalid key",
		in:      multipartKeyStruct{"foo"},
		wantErr: true,
	}}

	for _, tt := range tests {
		got, err := keyStructMap(tt.in)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: keyStructMap(%v): got unexpected error: %v", tt.name, tt.in, err)
			}
			continue
		}

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("%s: keyStructMap(%v): did not get expected output, diff(-got,+want):\n%s", tt.name, tt.in, diff)
		}
	}
}
