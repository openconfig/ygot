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

package ygen

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

func TestWriteGzippedByteSlice(t *testing.T) {
	tests := []struct {
		name    string
		inBytes []byte
		want    []byte
		wantErr bool
	}{{
		name:    "simple string test",
		inBytes: []byte("test"),
		want:    []byte{31, 139, 8, 0, 0, 0, 0, 0, 2, 255, 42, 73, 45, 46, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 12, 126, 127, 216, 4, 0, 0, 0},
	}, {
		name:    "mixed input test",
		inBytes: []byte{0x42, 0x32, 0x26},
		want:    []byte{31, 139, 8, 0, 0, 0, 0, 0, 2, 255, 114, 50, 82, 3, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 48, 81, 34, 179, 3, 0, 0, 0},
	}}

	for _, tt := range tests {
		got, err := WriteGzippedByteSlice(tt.inBytes)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: WriteGzippedByteSlice(%v): got unexpected error: %v", tt.name, tt.inBytes, err)
			}
			continue
		}

		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: WriteGzippedByteSlice(%v): did not get expected output, got: %v, want: %v", tt.name, tt.inBytes, got, tt.want)
		}
	}

}

func TestBytesToGoByteSlice(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []string
	}{{
		name: "short string",
		in:   []byte{0x0, 0x1, 0x2, 0x3},
		want: []string{"0x00, 0x01, 0x02, 0x03,"},
	}, {
		name: "longer string",
		in:   []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF, 0x10},
		want: []string{"0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,",
			"0x10,"},
	}}

	for _, tt := range tests {
		got := BytesToGoByteSlice(tt.in)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: BytesToGoByteSlice(%v): did not get expected output, got: %v, want: %v", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestSchemaRoundtrip(t *testing.T) {
	// Test case 1 input, annotations are added during the schema generation,
	// therefore we must annotate the schema after the fact.
	moduleEntry := &yang.Entry{
		Name: "module",
	}
	containerEntry := &yang.Entry{
		Name:   "container",
		Parent: moduleEntry,
	}
	leafEntry := &yang.Entry{
		Name:   "leaf",
		Parent: containerEntry,
	}
	containerEntry.Dir = map[string]*yang.Entry{
		"leaf": leafEntry,
	}

	annotatedContainerEntry := &yang.Entry{
		Name: "container",
		Annotation: map[string]interface{}{
			"schemapath": "/module/container",
			"structname": "Container",
		},
	}
	annotatedLeafEntry := &yang.Entry{
		Name:   "leaf",
		Parent: annotatedContainerEntry,
	}
	annotatedContainerEntry.Dir = map[string]*yang.Entry{
		"leaf": annotatedLeafEntry,
	}

	// Test case 2: fake root entry.
	fakeRootEntry := &yang.Entry{
		Name: "device",
		Kind: yang.DirectoryEntry,
		Node: &yang.Value{
			Name: rootElementNodeName,
		},
	}
	childEntry := &yang.Entry{
		Name:   "child",
		Parent: fakeRootEntry,
	}
	fakeRootEntry.Dir = map[string]*yang.Entry{
		"child": childEntry,
	}

	annotatedFakeRootEntry := &yang.Entry{
		Name: "device",
		Kind: yang.DirectoryEntry,
		Annotation: map[string]interface{}{
			"structname": "Device",
			"schemapath": "/device",
			"isFakeRoot": true,
		},
	}
	annotatedChildEntry := &yang.Entry{
		Name:   "child",
		Parent: annotatedFakeRootEntry,
	}
	annotatedFakeRootEntry.Dir = map[string]*yang.Entry{
		"child": annotatedChildEntry,
	}

	tests := []struct {
		name               string
		inMap              map[string]*yangStruct
		inGenerateFakeRoot bool
		want               map[string]*yang.Entry
		wantErr            bool
	}{{
		name: "simple schema",
		inMap: map[string]*yangStruct{
			"Container": {
				name:  "Container",
				entry: containerEntry,
			},
		},
		want: map[string]*yang.Entry{
			"Container": annotatedContainerEntry,
		},
	}, {
		name: "fakeroot",
		inMap: map[string]*yangStruct{
			"Container": {
				name:  "Container",
				entry: containerEntry,
			},
			"Device": {
				name:       "Device",
				entry:      fakeRootEntry,
				isFakeRoot: true,
			},
		},
		inGenerateFakeRoot: true,
		want: map[string]*yang.Entry{
			"Device": annotatedFakeRootEntry,
		},
	}}

	for _, tt := range tests {
		cg := NewYANGCodeGenerator(&GeneratorConfig{
			GenerateFakeRoot: tt.inGenerateFakeRoot,
		})
		gotByte, err := cg.serialiseStructDefinitions(tt.inMap)

		if (err != nil) != tt.wantErr {
			t.Errorf("%s: cg.SerialiseStructDefinitions(%v), got unexpected error, err: %v", tt.name, tt.inMap, err)
			continue
		}

		gotGzip, err := WriteGzippedByteSlice(gotByte)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: WriteGzippedByteSlice(...), got unexpected error:, err: %v", tt.name, err)
		}

		got, err := ygot.GzipToSchema(gotGzip)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: GzipToSchema(...), got unexpected error, err: %v", tt.name, err)
			continue
		}

		if !reflect.DeepEqual(got, tt.want) {
			// Use the JSON serialisation for test debugging output.
			gotj, _ := json.MarshalIndent(got, "", "  ")
			wantj, _ := json.MarshalIndent(tt.want, "", "  ")
			diff, _ := generateUnifiedDiff(string(wantj), string(gotj))
			t.Errorf("%s: GzipToSchema(...), did not get expected output, diff(-got,+want):\n%s", tt.name, diff)
		}
	}
}
