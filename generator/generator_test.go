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

package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/ygot/ygen"
)

func TestWriteGoCode(t *testing.T) {
	tests := []struct {
		name     string
		inGoCode *ygen.GeneratedGoCode
		wantCode string
	}{{
		name: "single element structs and enums",
		inGoCode: &ygen.GeneratedGoCode{
			Structs: []ygen.GoStructCodeSnippet{{
				StructDef: `structOne`,
			}},
			Enums:   []string{`enumOne`},
			EnumMap: "ΛMap",
		},
		wantCode: `structOne
enumOne
ΛMap
`,
	}, {
		name: "multi-element structs and enums",
		inGoCode: &ygen.GeneratedGoCode{
			Structs: []ygen.GoStructCodeSnippet{{
				StructDef: "structOne",
			}, {
				StructDef: "structTwo",
			}},
			Enums:   []string{"enumOne", "enumTwo"},
			EnumMap: "ΛMap",
		},
		wantCode: `structOne
structTwo
enumOne
enumTwo
ΛMap
`,
	}, {
		name: "json string code",
		inGoCode: &ygen.GeneratedGoCode{
			JSONSchemaCode: "foo",
		},
		wantCode: `
foo
`,
	}, {
		name: "enum type map",
		inGoCode: &ygen.GeneratedGoCode{
			EnumTypeMap: "map",
		},
		wantCode: `
map
`,
	}}

	for _, tt := range tests {
		var got bytes.Buffer
		if err := writeGoCodeSingleFile(&got, tt.inGoCode); err != nil {
			t.Errorf("%s: writeGoCode(%v): got unexpected error: %v", tt.name, tt.inGoCode, err)
			continue
		}

		if diff := pretty.Compare(tt.wantCode, got.String()); diff != "" {
			t.Errorf("%s: writeGoCode(%v): got invalid output, diff(-got,+want):\n%s", tt.name, tt.inGoCode, diff)
		}
	}
}

func TestMakeOutputSpec(t *testing.T) {
	tests := []struct {
		name string
		in   *ygen.GeneratedGoCode
		want map[string]codeOut
	}{{
		name: "simple struct with all only structs populated",
		in: &ygen.GeneratedGoCode{
			Structs: []ygen.GoStructCodeSnippet{{
				StructName: "name",
				StructDef:  "def\n",
				ListKeys:   "name_key",
				Methods:    "methods",
				Interfaces: "interfaces",
			}},
		},
		want: map[string]codeOut{
			enumMapFn:   {},
			enumFn:      {},
			schemaFn:    {},
			interfaceFn: {contents: "interfaces\n"},
			methodFn:    {contents: "methods\n", oneoffHeader: true},
			fmt.Sprintf("%sn.go", structBaseFn): {contents: "def\nname_key\n"},
		},
	}, {
		name: "two structs with enums populated",
		in: &ygen.GeneratedGoCode{
			Structs: []ygen.GoStructCodeSnippet{{
				StructName: "s1",
				StructDef:  "s1def\n",
				ListKeys:   "s1key",
				Methods:    "s1methods",
				Interfaces: "s1interfaces",
			}, {
				StructName: "s2",
				StructDef:  "s2def\n",
				ListKeys:   "s2key",
				Methods:    "s2methods",
				Interfaces: "s2interfaces",
			}},
			Enums:   []string{"enum1", "enum2"},
			EnumMap: "enummap",
		},
		want: map[string]codeOut{
			enumMapFn:                           {contents: "enummap\n"},
			enumFn:                              {contents: "enum1\nenum2"},
			schemaFn:                            {},
			interfaceFn:                         {contents: "s1interfaces\ns2interfaces\n"},
			fmt.Sprintf("%ss.go", structBaseFn): {contents: "s1def\ns1key\ns2def\ns2key\n"},
			methodFn: {contents: "s1methods\ns2methods\n", oneoffHeader: true},
		},
	}, {
		name: "two structs, different starting letters",
		in: &ygen.GeneratedGoCode{
			Structs: []ygen.GoStructCodeSnippet{{
				StructName: "s1",
				StructDef:  "s1def\n",
				ListKeys:   "s1key",
				Methods:    "s1methods",
				Interfaces: "s1interfaces",
			}, {
				StructName: "q2",
				StructDef:  "q2def\n",
				ListKeys:   "q2key",
				Methods:    "q2methods",
				Interfaces: "q2interfaces",
			}},
			JSONSchemaCode: "schema",
		},
		want: map[string]codeOut{
			enumMapFn:                           {},
			enumFn:                              {},
			schemaFn:                            {contents: "schema"},
			interfaceFn:                         {contents: "s1interfaces\nq2interfaces\n"},
			fmt.Sprintf("%ss.go", structBaseFn): {contents: "s1def\ns1key\n"},
			fmt.Sprintf("%sq.go", structBaseFn): {contents: "q2def\nq2key\n"},
			methodFn: {contents: "s1methods\nq2methods\n", oneoffHeader: true},
		},
	}}

	for _, tt := range tests {
		got := makeOutputSpec(tt.in)
		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("%s: makeOutputSpec(%v): did not get expected output, diff (-got,+want):\n%s", tt.name, tt.in, diff)
		}
	}
}
