// Copyright 2018 Google Inc.
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

package ytypes

import (
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

type schemaRoot struct{}

func (s *schemaRoot) Validate(...ygot.ValidationOption) error { return nil }
func (s *schemaRoot) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (s *schemaRoot) IsYANGGoStruct()                         {}

func TestSchema(t *testing.T) {
	tests := []struct {
		desc           string
		in             *Schema
		wantRootSchema *yang.Entry
		wantValid      bool
	}{{
		desc: "simple valid schema",
		in: &Schema{
			Root: &schemaRoot{},
			SchemaTree: map[string]*yang.Entry{
				"schemaRoot": {Name: "test"},
			},
			Unmarshal: func([]byte, ygot.GoStruct, ...UnmarshalOpt) error { return nil },
		},
		wantRootSchema: &yang.Entry{Name: "test"},
		wantValid:      true,
	}, {
		desc:      "invalid schema",
		in:        &Schema{},
		wantValid: false,
	}, {
		desc: "no such schema root",
		in: &Schema{
			Root:       &schemaRoot{},
			SchemaTree: map[string]*yang.Entry{},
			Unmarshal:  func([]byte, ygot.GoStruct, ...UnmarshalOpt) error { return nil },
		},
		wantRootSchema: nil,
		wantValid:      true,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := tt.in.IsValid(); got != tt.wantValid {
				t.Errorf("did not get expected valid status, got: %v, want: %v", got, tt.wantValid)
			}

			if !tt.wantValid {
				return
			}

			if diff := pretty.Compare(tt.in.RootSchema(), tt.wantRootSchema); diff != "" {
				t.Errorf("did not get expected root schema, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
