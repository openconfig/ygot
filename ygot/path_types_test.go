// Copyright 2019 Google Inc.
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
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
)

func TestResolvePath(t *testing.T) {
	root := &NodePath{}

	tests := []struct {
		name        string
		in          PathStruct
		wantPathStr string
		wantRoot    PathStruct
		wantErr     bool
	}{{
		name: "simple",
		in: &NodePath{
			relSchemaPath: []string{"child"},
			keys:          map[string]interface{}{},
			p: &NodePath{
				relSchemaPath: []string{"parent"},
				keys:          map[string]interface{}{},
				p:             root,
			},
		},
		wantPathStr: "/parent/child",
		wantRoot:    root,
	}, {
		name:        "root",
		in:          root,
		wantPathStr: "/",
		wantRoot:    root,
	}, {
		name: "list",
		in: &NodePath{
			relSchemaPath: []string{"values", "value"},
			keys:          map[string]interface{}{"ID": 5},
			p: &NodePath{
				relSchemaPath: []string{"parent"},
				keys:          map[string]interface{}{},
				p:             root,
			},
		},
		wantPathStr: "/parent/values/value[ID=5]",
		wantRoot:    root,
	}, {
		name: "list with unconvertible key value",
		in: &NodePath{
			relSchemaPath: []string{"values", "value"},
			keys:          map[string]interface{}{"ID": complex(1, 2)},
			p: &NodePath{
				relSchemaPath: []string{"parent"},
				keys:          map[string]interface{}{},
				p:             root,
			},
		},
		wantPathStr: "",
		wantRoot:    nil,
		wantErr:     true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantP, err := StringToStructuredPath(tt.wantPathStr)
			if err != nil {
				t.Fatal(err)
			}
			wantPath := wantP.Elem

			gotRoot, gotPath, gotErrs := ResolvePath(tt.in)
			if gotErrs != nil && !tt.wantErr {
				t.Fatal(gotErrs)
			} else if gotErrs == nil && tt.wantErr {
				t.Fatal("expected error but did not receive any")
			}

			if gotRoot != tt.wantRoot {
				t.Errorf("root not expected - got: %v, want: %v", gotRoot, tt.wantRoot)
			}

			if diff := cmp.Diff(wantPath, gotPath, cmp.Comparer(proto.Equal)); diff != "" {
				t.Errorf("ResolvePath returned diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveRelPath(t *testing.T) {
	root := &NodePath{}

	tests := []struct {
		name    string
		in      PathStruct
		want    string
		wantErr bool
	}{{
		name: "simple",
		in: &NodePath{
			relSchemaPath: []string{"child"},
			keys:          map[string]interface{}{},
		},
		want: "child",
	}, {
		name: "root",
		in:   root,
		want: "",
	}, {
		name: "list",
		in: &NodePath{
			relSchemaPath: []string{"values", "value"},
			keys:          map[string]interface{}{"ID": 5},
		},
		want: "values/value[ID=5]",
	}, {
		name: "list with unconvertible key value",
		in: &NodePath{
			relSchemaPath: []string{"values", "value"},
			keys:          map[string]interface{}{"ID": complex(1, 2)},
		},
		want:    "",
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantP, err := StringToStructuredPath(tt.want)
			wantPath := wantP.Elem
			if err != nil {
				t.Fatal(err)
			}

			gotPath, gotErrs := ResolveRelPath(tt.in)
			if gotErrs != nil && !tt.wantErr {
				t.Fatal(gotErrs)
			} else if gotErrs == nil && tt.wantErr {
				t.Fatal("expected error but did not receive any")
			}

			if diff := cmp.Diff(wantPath, gotPath, cmp.Comparer(proto.Equal)); diff != "" {
				t.Errorf("ResolveRelPath returned diff (-want +got):\n%s", diff)
			}
		})
	}
}
