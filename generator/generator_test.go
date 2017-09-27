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
			Structs: []string{`structOne`},
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
			Structs: []string{"structOne", "structTwo"},
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
		if err := writeGoCode(&got, tt.inGoCode); err != nil {
			t.Errorf("%s: writeGoCode(%v): got unexpected error: %v", tt.name, tt.inGoCode, err)
			continue
		}

		if diff := pretty.Compare(tt.wantCode, got.String()); diff != "" {
			t.Errorf("%s: writeGoCode(%v): got invalid output, diff(-got,+want):\n%s", tt.name, tt.inGoCode, diff)
		}
	}
}
