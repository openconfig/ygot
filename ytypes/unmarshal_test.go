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

package ytypes

import (
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
)

func TestUnmarshal(t *testing.T) {
	type ParentStruct struct {
		Leaf *string `path:"leaf"`
	}
	validSchema := &yang.Entry{
		Name: "leaf",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Ystring,
		},
	}
	choiceSchema := &yang.Entry{
		Name: "choice",
		Kind: yang.ChoiceEntry,
	}
	tests := []struct {
		desc    string
		schema  *yang.Entry
		value   interface{}
		wantErr string
	}{
		{
			desc:   "success nil field",
			schema: validSchema,
			value:  nil,
		},
		{
			desc:    "error nil schema",
			schema:  nil,
			value:   "{}",
			wantErr: `nil schema for parent type *ytypes.ParentStruct, value {} (string)`,
		},
		{
			desc:    "error choice schema",
			schema:  choiceSchema,
			value:   "{}",
			wantErr: `cannot pass choice schema choice to Unmarshal`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var parent ParentStruct

			err := Unmarshal(tt.schema, &parent, tt.value)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: got error: %v, want error: %v", tt.desc, got, want)
			}
		})
	}
}
