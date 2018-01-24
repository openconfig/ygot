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

package diff

import (
	"testing"

	"github.com/openconfig/ygot/ygot"
)

type basicStruct struct {
	StringValue *string         `path:"string-value"`
	StructValue *basicStructTwo `path:"struct-value"`
}

type basicStructTwo struct {
	StringValue *string `path:"string-value"`
}

func (*basicStruct) IsYANGGoStruct() {}

func TestDiff(t *testing.T) {
	tests := []struct {
		desc       string
		inOriginal ygot.GoStruct
		inModified ygot.GoStruct
		inIn       interface{}
		inOut      interface{}
		wantErr    string
	}{{
		desc:       "test",
		inOriginal: &basicStruct{StringValue: ygot.String("value")},
	}}

	for _, tt := range tests {
		_, err := Diff(tt.inOriginal, tt.inModified)
		if err != nil && (err.Error() != tt.wantErr) {
			t.Errorf("%s: Diff(%v, %v): did not get expected error: %v", tt.desc, tt.inOriginal, tt.inModified, err)
			continue
		}

	}
}
