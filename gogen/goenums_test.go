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

package gogen

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygen"
	"github.com/openconfig/ygot/ygot"
)

// TestGenGoEnumeratedTypes validates the enumerated type code generation from a YANG
// module.
func TestGenGoEnumeratedTypes(t *testing.T) {
	// In order to create a mock enum within goyang, we must construct it using the
	// relevant methods, since the field of the EnumType struct (toString) that we
	// need to set is not publicly exported.
	testEnumerations := map[string][]string{
		"enumOne": {"SPEED_2.5G", "SPEED-40G"},
		"enumTwo": {"VALUE_1", "VALUE_2", "VALUE_3", "VALUE_4"},
	}
	testYangEnums := make(map[string]*yang.EnumType)

	for name, values := range testEnumerations {
		enum := yang.NewEnumType()
		for i, enumValue := range values {
			enum.Set(enumValue, int64(i))
		}
		testYangEnums[name] = enum
	}

	tests := []struct {
		name string
		in   map[string]*ygen.EnumeratedYANGType
		want map[string]*goEnumeratedType
	}{{
		name: "enum",
		in: map[string]*ygen.EnumeratedYANGType{
			"foo": {
				Name:     "EnumeratedValue",
				Kind:     ygen.SimpleEnumerationType,
				TypeName: "enumerated-value",
				ValToYANGDetails: []ygot.EnumDefinition{
					{
						Name:           "VALUE_A",
						DefiningModule: "",
						Value:          0,
					},
					{
						Name:           "VALUE_B",
						DefiningModule: "",
						Value:          1,
					},
					{
						Name:           "VALUE_C",
						DefiningModule: "",
						Value:          2,
					},
				},
			},
		},
		want: map[string]*goEnumeratedType{
			"EnumeratedValue": {
				Name: "EnumeratedValue",
				CodeValues: map[int64]string{
					0: "UNSET",
					1: "VALUE_A",
					2: "VALUE_B",
					3: "VALUE_C",
				},
				YANGValues: map[int64]ygot.EnumDefinition{
					1: {
						Name:           "VALUE_A",
						DefiningModule: "",
						Value:          0,
					},
					2: {
						Name:           "VALUE_B",
						DefiningModule: "",
						Value:          1,
					},
					3: {
						Name:           "VALUE_C",
						DefiningModule: "",
						Value:          2,
					},
				},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := genGoEnumeratedTypes(tt.in)
			if err != nil {
				t.Errorf("%s: genGoEnumeratedTypes(%v): got unexpected error: %v",
					tt.name, tt.in, err)
				return
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("%s: genGoEnumeratedTypes(%v): got incorrect output, diff(-want, +got):\n%s",
					tt.name, tt.in, diff)
			}
		})
	}
}

// TestWriteGoEnum validates the enumerated type code generation from a parsed enum.
func TestWriteGoEnum(t *testing.T) {
	tests := []struct {
		name string
		in   *goEnumeratedType
		want string
	}{{
		name: "enum from identityref",
		in: &goEnumeratedType{
			Name: "EnumeratedValue",
			CodeValues: map[int64]string{
				0: "UNSET",
				1: "VALUE_A",
				2: "VALUE_B",
				3: "VALUE_C",
			},
		},
		want: `
// E_EnumeratedValue is a derived int64 type which is used to represent
// the enumerated node EnumeratedValue. An additional value named
// EnumeratedValue_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_EnumeratedValue int64

// IsYANGGoEnum ensures that EnumeratedValue implements the yang.GoEnum
// interface. This ensures that EnumeratedValue can be identified as a
// mapped type for a YANG enumeration.
func (E_EnumeratedValue) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  EnumeratedValue.
func (E_EnumeratedValue) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum; }

// String returns a logging-friendly string for E_EnumeratedValue.
func (e E_EnumeratedValue) String() string {
	return ygot.EnumLogString(e, int64(e), "E_EnumeratedValue")
}

const (
	// EnumeratedValue_UNSET corresponds to the value UNSET of EnumeratedValue
	EnumeratedValue_UNSET E_EnumeratedValue = 0
	// EnumeratedValue_VALUE_A corresponds to the value VALUE_A of EnumeratedValue
	EnumeratedValue_VALUE_A E_EnumeratedValue = 1
	// EnumeratedValue_VALUE_B corresponds to the value VALUE_B of EnumeratedValue
	EnumeratedValue_VALUE_B E_EnumeratedValue = 2
	// EnumeratedValue_VALUE_C corresponds to the value VALUE_C of EnumeratedValue
	EnumeratedValue_VALUE_C E_EnumeratedValue = 3
)
`,
	}}

	for _, tt := range tests {
		got, err := writeGoEnum(tt.in)
		if err != nil {
			t.Errorf("%s: writeGoEnum(%v): got unexpected error: %v",
				tt.name, tt.in, err)
			continue
		}

		if diff := cmp.Diff(tt.want, got); diff != "" {
			fmt.Println(diff)
			if diffl, err := testutil.GenerateUnifiedDiff(tt.want, got); err == nil {
				diff = diffl
			}
			t.Errorf("%s: writeGoEnum(%v): got incorrect output, diff(-want, +got):\n%s",
				tt.name, tt.in, diff)
		}
	}
}

func TestWriteGoEnumMap(t *testing.T) {
	tests := []struct {
		name    string
		inMap   map[string]map[int64]ygot.EnumDefinition
		wantErr bool
		wantMap string
	}{{
		name: "simple map input",
		inMap: map[string]map[int64]ygot.EnumDefinition{
			"EnumOne": {
				1: {Name: "VAL1"},
				2: {Name: "VAL2"},
			},
		},
		wantMap: `
// ΛEnum is a map, keyed by the name of the type defined for each enum in the
// generated Go code, which provides a mapping between the constant int64 value
// of each value of the enumeration, and the string that is used to represent it
// in the YANG schema. The map is named ΛEnum in order to avoid clash with any
// valid YANG identifier.
var ΛEnum = map[string]map[int64]ygot.EnumDefinition{
	"E_EnumOne": {
		1: {Name: "VAL1"},
		2: {Name: "VAL2"},
	},
}
`,
	}, {
		name: "multiple enum input",
		inMap: map[string]map[int64]ygot.EnumDefinition{
			"EnumOne": {
				1: {Name: "VAL1"},
				2: {Name: "VAL2"},
			},
			"EnumTwo": {
				1: {Name: "VAL42"},
				2: {Name: "VAL43"},
			},
		},
		wantMap: `
// ΛEnum is a map, keyed by the name of the type defined for each enum in the
// generated Go code, which provides a mapping between the constant int64 value
// of each value of the enumeration, and the string that is used to represent it
// in the YANG schema. The map is named ΛEnum in order to avoid clash with any
// valid YANG identifier.
var ΛEnum = map[string]map[int64]ygot.EnumDefinition{
	"E_EnumOne": {
		1: {Name: "VAL1"},
		2: {Name: "VAL2"},
	},
	"E_EnumTwo": {
		1: {Name: "VAL42"},
		2: {Name: "VAL43"},
	},
}
`,
	}}

	for _, tt := range tests {
		got, err := writeGoEnumMap(tt.inMap)

		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: got unexpected error when generating map: %v", tt.name, err)
			}
			continue
		}

		if tt.wantMap != got {
			diff := fmt.Sprintf("got: %s, want %s", got, tt.wantMap)
			if diffl, err := testutil.GenerateUnifiedDiff(tt.wantMap, got); err == nil {
				diff = "diff (-want, +got):\n" + diffl
			}
			t.Errorf("%s: did not get expected generated enum, %s", tt.name, diff)
		}
	}
}
