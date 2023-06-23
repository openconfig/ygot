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
	"testing"

	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/ygot/ygen"
)

func TestUnOrderedKeyedMapTypeName(t *testing.T) {
	tests := []struct {
		desc               string
		inListYANGPath     string
		inListFieldName    string
		inParentName       string
		inGoStructElements map[string]*ygen.ParsedDirectory
		wantMapName        string
		wantKeyName        string
		wantIsDefined      bool
		wantErrSubstr      string
	}{{
		desc:            "single-key",
		inListYANGPath:  "/foo/bar",
		inListFieldName: "Bar",
		inParentName:    "Foo",
		inGoStructElements: map[string]*ygen.ParsedDirectory{
			"/foo/bar": {
				Name: "Foo_Bar",
				ListKeys: map[string]*ygen.ListKey{
					"name": {
						LangType: &ygen.MappedType{
							NativeType: "string",
						},
					},
				},
			},
		},
		wantMapName:   "map[string]*Foo_Bar",
		wantKeyName:   "string",
		wantIsDefined: false,
	}, {
		desc:            "multi-key",
		inListYANGPath:  "/foo/bar",
		inListFieldName: "Bar",
		inParentName:    "Foo",
		inGoStructElements: map[string]*ygen.ParsedDirectory{
			"/foo/bar": {
				Name: "Foo_Bar",
				ListKeys: map[string]*ygen.ListKey{
					"name": {
						LangType: &ygen.MappedType{
							NativeType: "string",
						},
					},
					"place": {
						LangType: &ygen.MappedType{
							NativeType: "string",
						},
					},
				},
			},
		},
		wantMapName:   "map[Foo_Bar_Key]*Foo_Bar",
		wantKeyName:   "Foo_Bar_Key",
		wantIsDefined: true,
	}, {
		desc:            "multi-key-with-conflict",
		inListYANGPath:  "/foo/bar",
		inListFieldName: "Bar",
		inParentName:    "Foo",
		inGoStructElements: map[string]*ygen.ParsedDirectory{
			"/foo/bar": {
				Name: "Foo_Bar",
				ListKeys: map[string]*ygen.ListKey{
					"name": {
						LangType: &ygen.MappedType{
							NativeType: "string",
						},
					},
					"place": {
						LangType: &ygen.MappedType{
							NativeType: "string",
						},
					},
				},
			},
			"/foo/bar/key": {
				Name: "Foo_Bar_Key",
			},
		},
		wantMapName:   "map[Foo_Bar_YANGListKey]*Foo_Bar",
		wantKeyName:   "Foo_Bar_YANGListKey",
		wantIsDefined: true,
	}, {
		desc:            "multi-key-with-unresolvable-conflict",
		inListYANGPath:  "/foo/bar",
		inListFieldName: "Bar",
		inParentName:    "Foo",
		inGoStructElements: map[string]*ygen.ParsedDirectory{
			"/foo/bar": {
				Name: "Foo_Bar",
				ListKeys: map[string]*ygen.ListKey{
					"name": {
						LangType: &ygen.MappedType{
							NativeType: "string",
						},
					},
					"place": {
						LangType: &ygen.MappedType{
							NativeType: "string",
						},
					},
				},
			},
			"/foo/bar/key": {
				Name: "Foo_Bar_Key",
			},
			"/foo/bar/key/YANGListKey": {
				Name: "Foo_Bar_YANGListKey",
			},
		},
		wantErrSubstr: "unexpected generated list key name conflict",
	}, {
		desc:               "error-list-not-found",
		inListYANGPath:     "/foo/bar",
		inListFieldName:    "Bar",
		inParentName:       "Foo",
		inGoStructElements: map[string]*ygen.ParsedDirectory{},
		wantErrSubstr:      "did not exist",
	}, {
		desc:            "error-unkeyed-list",
		inListYANGPath:  "/foo/bar",
		inListFieldName: "Bar",
		inParentName:    "Foo",
		inGoStructElements: map[string]*ygen.ParsedDirectory{
			"/foo/bar": {
				Name: "Foo_Bar",
			},
		},
		wantErrSubstr: "list does not contain any keys",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			gotMapName, gotKeyName, gotIsDefined, err := UnorderedMapTypeName(tt.inListYANGPath, tt.inListFieldName, tt.inParentName, tt.inGoStructElements)
			if diff := errdiff.Substring(err, tt.wantErrSubstr); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}
			if gotMapName != tt.wantMapName {
				t.Errorf("map name: got %q, want %q", gotMapName, tt.wantMapName)
			}
			if gotKeyName != tt.wantKeyName {
				t.Errorf("key name: got %q, want %q", gotKeyName, tt.wantKeyName)
			}
			if gotIsDefined != tt.wantIsDefined {
				t.Errorf("map name: got %v, want %v", gotIsDefined, tt.wantIsDefined)
			}
		})
	}
}
