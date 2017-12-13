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
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

func TestBuildJSONTree(t *testing.T) {
	// Simple YANG hierarchy for test case.
	simpleModule := &yang.Entry{
		Name: "a-module",
		Kind: yang.DirectoryEntry,
	}
	simpleContainer := &yang.Entry{
		Name:   "simple-container",
		Kind:   yang.DirectoryEntry,
		Parent: simpleModule,
	}
	simpleLeaf := &yang.Entry{
		Name:   "simple-leaf",
		Kind:   yang.LeafEntry,
		Parent: simpleContainer,
	}
	simpleContainer.Dir = map[string]*yang.Entry{"simple-leaf": simpleLeaf}
	simpleModule.Dir = map[string]*yang.Entry{"simple-container": simpleContainer}

	// More complex YANG hierarchy with multiple modules, and children.
	moduleTwo := &yang.Entry{
		Name: "a-module",
		Kind: yang.DirectoryEntry,
	}
	moduleTwoContainerOne := &yang.Entry{
		Name:   "container-one",
		Kind:   yang.DirectoryEntry,
		Parent: moduleTwo,
	}
	moduleTwoContainerOne.Dir = map[string]*yang.Entry{
		"ch-one": {
			Name:   "ch-one",
			Kind:   yang.LeafEntry,
			Type:   &yang.YangType{Kind: yang.Ystring},
			Parent: moduleTwoContainerOne,
		},
		"ch-two": {
			Name:   "ch-two",
			Kind:   yang.LeafEntry,
			Type:   &yang.YangType{Kind: yang.Yuint32},
			Parent: moduleTwoContainerOne,
		},
	}
	moduleTwoContainerTwo := &yang.Entry{
		Name:   "container-two",
		Kind:   yang.DirectoryEntry,
		Parent: moduleTwo,
	}
	moduleTwoContainerTwo.Dir = map[string]*yang.Entry{
		"ch2-leafone": {
			Name:   "ch2-leafone",
			Kind:   yang.LeafEntry,
			Type:   &yang.YangType{Kind: yang.Ystring},
			Parent: moduleTwoContainerTwo,
		},
	}
	moduleTwo.Dir = map[string]*yang.Entry{
		"container-one": moduleTwoContainerOne,
		"container-two": moduleTwoContainerTwo,
	}

	tests := []struct {
		name             string
		inEntries        []*yang.Entry
		inDirectoryNames map[string]string
		inFakeRoot       *yang.Entry
		inCompressed     bool
		want             string
		wantErr          string
	}{{
		name:      "simple module entry",
		inEntries: []*yang.Entry{simpleModule},
		inDirectoryNames: map[string]string{
			"/a-module/simple-container": "SimpleContainer",
		},
		inCompressed: true,
		want: `{
    "Name": "",
    "Kind": 0,
    "Config": 0,
    "Dir": {
        "simple-container": {
            "Name": "simple-container",
            "Kind": 1,
            "Config": 0,
            "Dir": {
                "simple-leaf": {
                    "Name": "simple-leaf",
                    "Kind": 0,
                    "Config": 0
                }
            },
            "Annotation": {
                "schemapath": "/a-module/simple-container",
                "structname": "SimpleContainer"
            }
        }
    },
    "Annotation": {
        "isCompressedSchema": true,
        "isFakeRoot": true
    }
}`,
	}, {
		name:      "multiple modules",
		inEntries: []*yang.Entry{simpleModule, moduleTwo},
		inDirectoryNames: map[string]string{
			"/a-module/simple-container": "SimpleContainer",
			"/module-two/container-one":  "C1",
			"/module-two/container-two":  "C2",
		},
		want: `{
    "Name": "",
    "Kind": 0,
    "Config": 0,
    "Dir": {
        "container-one": {
            "Name": "container-one",
            "Kind": 1,
            "Config": 0,
            "Dir": {
                "ch-one": {
                    "Name": "ch-one",
                    "Kind": 0,
                    "Config": 0,
                    "Type": {
                        "Name": "",
                        "Kind": 18
                    }
                },
                "ch-two": {
                    "Name": "ch-two",
                    "Kind": 0,
                    "Config": 0,
                    "Type": {
                        "Name": "",
                        "Kind": 7
                    }
                }
            },
            "Annotation": {
                "schemapath": "/a-module/container-one"
            }
        },
        "container-two": {
            "Name": "container-two",
            "Kind": 1,
            "Config": 0,
            "Dir": {
                "ch2-leafone": {
                    "Name": "ch2-leafone",
                    "Kind": 0,
                    "Config": 0,
                    "Type": {
                        "Name": "",
                        "Kind": 18
                    }
                }
            },
            "Annotation": {
                "schemapath": "/a-module/container-two"
            }
        },
        "simple-container": {
            "Name": "simple-container",
            "Kind": 1,
            "Config": 0,
            "Dir": {
                "simple-leaf": {
                    "Name": "simple-leaf",
                    "Kind": 0,
                    "Config": 0
                }
            },
            "Annotation": {
                "schemapath": "/a-module/simple-container",
                "structname": "SimpleContainer"
            }
        }
    },
    "Annotation": {
        "isFakeRoot": true
    }
}`,
	}, {
		name:      "overlapping root children",
		inEntries: []*yang.Entry{simpleModule, simpleModule},
		inDirectoryNames: map[string]string{
			"/a-module/simple-container": "uniqueName",
		},
		wantErr: "overlapping root children for key simple-container",
	}, {
		name:      "non-nil fake root",
		inEntries: []*yang.Entry{simpleModule},
		inFakeRoot: &yang.Entry{
			Name: "device",
		},
		inDirectoryNames: map[string]string{
			"/a-module/simple-container": "uniqueName",
			"/device":                    "TheFakeRoot",
		},
		want: `{
    "Name": "device",
    "Kind": 1,
    "Config": 0,
    "Dir": {
        "simple-container": {
            "Name": "simple-container",
            "Kind": 1,
            "Config": 0,
            "Dir": {
                "simple-leaf": {
                    "Name": "simple-leaf",
                    "Kind": 0,
                    "Config": 0
                }
            },
            "Annotation": {
                "schemapath": "/a-module/simple-container",
                "structname": "uniqueName"
            }
        }
    },
    "Annotation": {
        "isFakeRoot": true,
        "schemapath": "/",
        "structname": "TheFakeRoot"
    }
}`,
	}}

	for _, tt := range tests {
		gotb, err := buildJSONTree(tt.inEntries, tt.inDirectoryNames, tt.inFakeRoot, tt.inCompressed)
		if err != nil && err.Error() != tt.wantErr {
			t.Errorf("%s: buildJSONTree(%v, %v): did not get expected error, got: %v, want: %v", tt.name, tt.inEntries, tt.inDirectoryNames, err, tt.wantErr)
		}

		got := string(gotb)
		if diff := pretty.Compare(got, tt.want); diff != "" {
			if diffl, err := generateUnifiedDiff(got, tt.want); err == nil {
				diff = diffl
			}
			t.Errorf("%s: buildJSONTree(%v, %v): did not get expected JSON tree, diff(-got,+want):\n%s", tt.name, tt.inEntries, tt.inDirectoryNames, diff)
		}
	}
}

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
		Kind:   yang.DirectoryEntry,
		Parent: moduleEntry,
	}
	leafEntry := &yang.Entry{
		Name:   "leaf",
		Parent: containerEntry,
	}
	containerEntry.Dir = map[string]*yang.Entry{
		"leaf": leafEntry,
	}
	moduleEntry.Dir = map[string]*yang.Entry{
		"container": containerEntry,
	}

	annotatedRootEntry := &yang.Entry{
		Dir:        map[string]*yang.Entry{},
		Annotation: map[string]interface{}{"isFakeRoot": true},
	}
	annotatedContainerEntry := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Annotation: map[string]interface{}{
			"schemapath": "/module/container",
			"structname": "Container",
		},
		Parent: annotatedRootEntry,
	}
	annotatedRootEntry.Dir["container"] = annotatedContainerEntry
	annotatedLeafEntry := &yang.Entry{
		Name:   "leaf",
		Parent: annotatedContainerEntry,
	}
	annotatedContainerEntry.Dir = map[string]*yang.Entry{
		"leaf": annotatedLeafEntry,
	}

	// Test case 2: with fakeroot
	fakeRootEntry := &yang.Entry{
		Name: "device",
		Kind: yang.DirectoryEntry,
	}

	fakeRootModuleEntry := &yang.Entry{
		Name: "module",
		Kind: yang.DirectoryEntry,
		Dir:  map[string]*yang.Entry{},
	}

	fakeRootContainerEntry := &yang.Entry{
		Name:   "container",
		Kind:   yang.DirectoryEntry,
		Parent: fakeRootModuleEntry,
		Dir:    map[string]*yang.Entry{},
	}
	fakeRootModuleEntry.Dir["container"] = fakeRootContainerEntry

	fakeRootLeafEntry := &yang.Entry{
		Name: "leaf",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Ystring,
		},
		Parent: fakeRootContainerEntry,
	}
	fakeRootContainerEntry.Dir["leaf"] = fakeRootLeafEntry

	annotatedFakeRootEntry := &yang.Entry{
		Name: "device",
		Kind: yang.DirectoryEntry,
		Dir:  map[string]*yang.Entry{},
		Annotation: map[string]interface{}{
			"isFakeRoot": true,
			"schemapath": "/",
			"structname": "Device",
		},
	}
	annotatedFakeRootContainerEntry := &yang.Entry{
		Name:   "container",
		Kind:   yang.DirectoryEntry,
		Parent: annotatedFakeRootEntry,
		Annotation: map[string]interface{}{
			"structname": "Container",
			"schemapath": "/module/container",
		},
		Dir: map[string]*yang.Entry{},
	}
	annotatedFakeRootEntry.Dir["container"] = annotatedFakeRootContainerEntry

	annotatedFakeRootLeafEntry := &yang.Entry{
		Name: "leaf",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Ystring,
		},
		Parent: annotatedFakeRootContainerEntry,
	}
	annotatedFakeRootContainerEntry.Dir["leaf"] = annotatedFakeRootLeafEntry

	tests := []struct {
		name             string
		inEntries        []*yang.Entry
		inFakeRoot       *yang.Entry
		inDirectoryNames map[string]string
		inCompressed     bool
		want             map[string]*yang.Entry
		wantJSONErr      string
		wantGzipErr      string
		wantSchemaErr    string
	}{{
		name:      "simple schema",
		inEntries: []*yang.Entry{moduleEntry},
		inDirectoryNames: map[string]string{
			"/module/container": "Container",
		},
		want: map[string]*yang.Entry{
			"Container": annotatedContainerEntry,
		},
	}, {
		name:       "test with fakeroot",
		inEntries:  []*yang.Entry{fakeRootModuleEntry},
		inFakeRoot: fakeRootEntry,
		inDirectoryNames: map[string]string{
			"/module/container": "Container",
			"/device":           "Device",
		},
		want: map[string]*yang.Entry{
			"Container": annotatedFakeRootContainerEntry,
			"Device":    annotatedFakeRootEntry,
		},
	}}

	for _, tt := range tests {
		gotByte, err := buildJSONTree(tt.inEntries, tt.inDirectoryNames, tt.inFakeRoot, tt.inCompressed)
		if err != nil && err.Error() != tt.wantJSONErr {
			t.Errorf("%s: buildJSONTree(%v, %v): did not get expected error, got: %v, want: %v", tt.name, tt.inEntries, tt.inDirectoryNames, err, tt.wantJSONErr)
			continue
		}

		gotGzip, err := WriteGzippedByteSlice(gotByte)
		if err != nil && err.Error() != tt.wantGzipErr {
			t.Errorf("%s: WriteGzippedByteSlice(%v): did not get expected error, got: %v, want: %v", tt.name, gotByte, err, tt.wantGzipErr)
			continue
		}

		got, err := ygot.GzipToSchema(gotGzip)
		if err != nil && err.Error() != tt.wantSchemaErr {
			t.Errorf("%s: ygot.GzipToSchema(%v): did not get expected error, got: %v, want: %v", tt.name, gotGzip, err, tt.wantSchemaErr)
			continue
		}

		if !reflect.DeepEqual(got, tt.want) {
			// Use JSON serialisation for test debugging output.
			gotj, _ := json.MarshalIndent(got, "", strings.Repeat(" ", 4))
			wantj, _ := json.MarshalIndent(tt.want, "", strings.Repeat(" ", 4))
			diff, _ := generateUnifiedDiff(string(gotj), string(wantj))
			t.Errorf("%s: GzipToSchema(...): did not get expected output, diff(-got,+want):\n%s", tt.name, diff)
		}
	}
}
