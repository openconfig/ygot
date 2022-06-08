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
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/ygot/gogen"
	"github.com/openconfig/ygot/ypathgen"
)

func TestWriteGoCode(t *testing.T) {
	tests := []struct {
		name     string
		inGoCode *gogen.GeneratedCode
		wantCode string
	}{{
		name: "single element structs and enums",
		inGoCode: &gogen.GeneratedCode{
			Structs: []gogen.GoStructCodeSnippet{{
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
		inGoCode: &gogen.GeneratedCode{
			Structs: []gogen.GoStructCodeSnippet{{
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
		inGoCode: &gogen.GeneratedCode{
			JSONSchemaCode: "foo",
		},
		wantCode: `
foo
`,
	}, {
		name: "enum type map",
		inGoCode: &gogen.GeneratedCode{
			EnumTypeMap: "map",
		},
		wantCode: `
map
`,
	}}

	for _, tt := range tests {
		var got strings.Builder
		if err := writeGoCodeSingleFile(&got, tt.inGoCode); err != nil {
			t.Errorf("%s: writeGoCode(%v): got unexpected error: %v", tt.name, tt.inGoCode, err)
			continue
		}

		if diff := pretty.Compare(tt.wantCode, got.String()); diff != "" {
			t.Errorf("%s: writeGoCode(%v): got invalid output, diff(-got,+want):\n%s", tt.name, tt.inGoCode, diff)
		}
	}
}

func TestSplitCodeByFileN(t *testing.T) {
	tests := []struct {
		name             string
		in               *gogen.GeneratedCode
		inFileN          int
		want             map[string]string
		wantErrSubstring string
	}{{
		name: "simple struct with all only structs populated",
		in: &gogen.GeneratedCode{
			CommonHeader: "common_header\n",
			OneOffHeader: "oneoff_header\n",
			Structs: []gogen.GoStructCodeSnippet{{
				StructName: "name",
				StructDef:  "def\n",
				ListKeys:   "name_key",
				Methods:    "methods",
				Interfaces: "interfaces",
			}},
		},
		inFileN: 1,
		want: map[string]string{
			enumMapFn:                      "common_header\n",
			enumFn:                         "common_header\n",
			schemaFn:                       "common_header\n",
			interfaceFn:                    "common_header\ninterfaces\n",
			fmt.Sprintf(structsFileFmt, 0): "common_header\noneoff_header\ndef\nname_key\nmethods\n",
		},
	}, {
		name: "less than 1 file requested for splitting",
		in: &gogen.GeneratedCode{
			CommonHeader: "common_header\n",
			OneOffHeader: "oneoff_header\n",
			Structs: []gogen.GoStructCodeSnippet{{
				StructName: "name",
				StructDef:  "def\n",
				ListKeys:   "name_key",
				Methods:    "methods",
				Interfaces: "interfaces",
			}},
		},
		inFileN:          0,
		wantErrSubstring: "requested 0 files",
	}, {
		name: "more than # of structs files requested for splitting",
		in: &gogen.GeneratedCode{
			CommonHeader: "common_header\n",
			OneOffHeader: "oneoff_header\n",
			Structs: []gogen.GoStructCodeSnippet{{
				StructName: "name",
				StructDef:  "def\n",
				ListKeys:   "name_key",
				Methods:    "methods",
				Interfaces: "interfaces",
			}},
		},
		inFileN:          2,
		wantErrSubstring: "requested 2 files",
	}, {
		name: "two structs with enums populated",
		in: &gogen.GeneratedCode{
			CommonHeader: "common_header\n",
			OneOffHeader: "oneoff_header\n",
			Structs: []gogen.GoStructCodeSnippet{{
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
		inFileN: 1,
		want: map[string]string{
			enumMapFn:                      "common_header\nenummap\n",
			enumFn:                         "common_header\nenum1\nenum2",
			schemaFn:                       "common_header\n",
			interfaceFn:                    "common_header\ns1interfaces\ns2interfaces\n",
			fmt.Sprintf(structsFileFmt, 0): "common_header\noneoff_header\ns1def\ns1key\ns1methods\ns2def\ns2key\ns2methods\n",
		},
	}, {
		name: "two structs, separated into two files",
		in: &gogen.GeneratedCode{
			CommonHeader: "common_header\n",
			OneOffHeader: "oneoff_header\n",
			Structs: []gogen.GoStructCodeSnippet{{
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
		inFileN: 2,
		want: map[string]string{
			enumMapFn:                      "common_header\n",
			enumFn:                         "common_header\n",
			schemaFn:                       "common_header\nschema",
			interfaceFn:                    "common_header\ns1interfaces\nq2interfaces\n",
			fmt.Sprintf(structsFileFmt, 0): "common_header\noneoff_header\ns1def\ns1key\ns1methods\n",
			fmt.Sprintf(structsFileFmt, 1): "common_header\nq2def\nq2key\nq2methods\n",
		},
	}, {
		name: "five structs, separated into four files",
		in: &gogen.GeneratedCode{
			CommonHeader: "common_header\n",
			OneOffHeader: "oneoff_header\n",
			Structs: []gogen.GoStructCodeSnippet{{
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
			}, {
				StructName: "s3",
				StructDef:  "s3def\n",
				ListKeys:   "s3key",
			}, {
				StructName: "s4",
				StructDef:  "s4def\n",
				ListKeys:   "s4key",
			}, {
				StructName: "s5",
				StructDef:  "s5def\n",
				ListKeys:   "s5key",
			}},
			JSONSchemaCode: "schema",
		},
		inFileN: 4,
		want: map[string]string{
			enumMapFn:                      "common_header\n",
			enumFn:                         "common_header\n",
			schemaFn:                       "common_header\nschema",
			interfaceFn:                    "common_header\ns1interfaces\ns2interfaces\n",
			fmt.Sprintf(structsFileFmt, 0): "common_header\noneoff_header\ns1def\ns1key\ns1methods\ns2def\ns2key\ns2methods\n",
			fmt.Sprintf(structsFileFmt, 1): "common_header\ns3def\ns3key\ns4def\ns4key\n",
			fmt.Sprintf(structsFileFmt, 2): "common_header\ns5def\ns5key\n",
			fmt.Sprintf(structsFileFmt, 3): "common_header\n",
		},
	}, {
		name: "five structs, separated into three files",
		in: &gogen.GeneratedCode{
			CommonHeader: "common_header\n",
			OneOffHeader: "oneoff_header\n",
			Structs: []gogen.GoStructCodeSnippet{{
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
			}, {
				StructName: "s3",
				StructDef:  "s3def\n",
				ListKeys:   "s3key",
			}, {
				StructName: "s4",
				StructDef:  "s4def\n",
				ListKeys:   "s4key",
			}, {
				StructName: "s5",
				StructDef:  "s5def\n",
				ListKeys:   "s5key",
			}},
			JSONSchemaCode: "schema",
		},
		inFileN: 3,
		want: map[string]string{
			enumMapFn:                      "common_header\n",
			enumFn:                         "common_header\n",
			schemaFn:                       "common_header\nschema",
			interfaceFn:                    "common_header\ns1interfaces\ns2interfaces\n",
			fmt.Sprintf(structsFileFmt, 0): "common_header\noneoff_header\ns1def\ns1key\ns1methods\ns2def\ns2key\ns2methods\n",
			fmt.Sprintf(structsFileFmt, 1): "common_header\ns3def\ns3key\ns4def\ns4key\n",
			fmt.Sprintf(structsFileFmt, 2): "common_header\ns5def\ns5key\n",
		},
	}, {
		name: "five structs, separated into two files",
		in: &gogen.GeneratedCode{
			CommonHeader: "common_header\n",
			OneOffHeader: "oneoff_header\n",
			Structs: []gogen.GoStructCodeSnippet{{
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
			}, {
				StructName: "s3",
				StructDef:  "s3def\n",
				ListKeys:   "s3key",
			}, {
				StructName: "s4",
				StructDef:  "s4def\n",
				ListKeys:   "s4key",
			}, {
				StructName: "s5",
				StructDef:  "s5def\n",
				ListKeys:   "s5key",
			}},
			JSONSchemaCode: "schema",
		},
		inFileN: 2,
		want: map[string]string{
			enumMapFn:                      "common_header\n",
			enumFn:                         "common_header\n",
			schemaFn:                       "common_header\nschema",
			interfaceFn:                    "common_header\ns1interfaces\ns2interfaces\n",
			fmt.Sprintf(structsFileFmt, 0): "common_header\noneoff_header\ns1def\ns1key\ns1methods\ns2def\ns2key\ns2methods\ns3def\ns3key\n",
			fmt.Sprintf(structsFileFmt, 1): "common_header\ns4def\ns4key\ns5def\ns5key\n",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitCodeByFileN(tt.in, tt.inFileN)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %v", diff)
			}
			if diff := pretty.Compare(got, tt.want); diff != "" {
				t.Errorf("splitCodeByFileN(%v): did not get expected output, diff (-got,+want):\n%s", tt.in, diff)
			}
		})
	}
}

func TestWritePathCode(t *testing.T) {
	tests := []struct {
		name string
		in   *ypathgen.GeneratedPathCode
		want string
	}{{
		name: "simple",
		in: &ypathgen.GeneratedPathCode{
			CommonHeader: "path common header\n",
			Structs: []ypathgen.GoPathStructCodeSnippet{{
				PathStructName:    "PathStructName",
				StructBase:        "\nStructDef\n",
				ChildConstructors: "\nChildConstructor\n",
			}},
		},
		want: `path common header

StructDef

ChildConstructor
`,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			if err := writeGoPathCodeSingleFile(&b, tt.in); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, b.String()); diff != "" {
				t.Errorf("diff (-want,+got):\n%s", diff)
			}
		})
	}
}
