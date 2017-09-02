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

var (
	// testErrOutput controls whether expect error test cases log the error
	// values.
	testErrOutput = false
)

// testErrLog logs err to t if err != nil and global value testErrOutput is set.
func testErrLog(t *testing.T, desc string, err error) {
	if err != nil {
		if testErrOutput {
			t.Logf("%s: %v", desc, err)
		}
	}
}

func TestValidateListAttr(t *testing.T) {
	validLeafListSchemaMin1 := &yang.Entry{
		Name:     "min1",
		Kind:     yang.LeafEntry,
		Type:     &yang.YangType{Kind: yang.Ystring},
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "1"}},
	}
	validLeafListSchemaMax3 := &yang.Entry{
		Name:     "max3",
		Kind:     yang.LeafEntry,
		Type:     &yang.YangType{Kind: yang.Ystring},
		ListAttr: &yang.ListAttr{MaxElements: &yang.Value{Name: "3"}},
	}
	validLeafListSchemaMin1Max3 := &yang.Entry{
		Name:     "min1max3",
		Kind:     yang.LeafEntry,
		Type:     &yang.YangType{Kind: yang.Ystring},
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "1"}, MaxElements: &yang.Value{Name: "3"}},
	}
	invalidLeafListSchemaNoAttr := &yang.Entry{
		Name: "no_attr",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{Kind: yang.Ystring},
	}
	invalidLeafListSchemaBadRange := &yang.Entry{
		Name:     "bad_range",
		Kind:     yang.LeafEntry,
		Type:     &yang.YangType{Kind: yang.Ystring},
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "bad"}},
	}
	invalidLeafListSchemaNegativeMinRange := &yang.Entry{
		Name:     "negative_min_range",
		Kind:     yang.LeafEntry,
		Type:     &yang.YangType{Kind: yang.Ystring},
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "-1"}},
	}
	invalidLeafListSchemaNegativeMaxRange := &yang.Entry{
		Name:     "negative_min_range",
		Kind:     yang.LeafEntry,
		Type:     &yang.YangType{Kind: yang.Ystring},
		ListAttr: &yang.ListAttr{MaxElements: &yang.Value{Name: "-1"}},
	}

	tests := []struct {
		desc    string
		schema  *yang.Entry
		value   interface{}
		wantErr bool
	}{
		{
			desc:    "nil schema",
			schema:  nil,
			wantErr: true,
		},
		{
			desc:    "missing ListAttr",
			schema:  invalidLeafListSchemaNoAttr,
			wantErr: true,
		},
		{
			desc:    "bad range value",
			schema:  invalidLeafListSchemaBadRange,
			wantErr: true,
		},
		{
			desc:    "negative min range value",
			schema:  invalidLeafListSchemaNegativeMinRange,
			wantErr: true,
		},
		{
			desc:    "negative max range value",
			schema:  invalidLeafListSchemaNegativeMaxRange,
			wantErr: true,
		},
		{
			desc:    "bad value type",
			schema:  validLeafListSchemaMin1,
			value:   int(1),
			wantErr: true,
		},
		{
			desc:   "min elements success",
			schema: validLeafListSchemaMin1,
			value:  []string{"a"},
		},
		{
			desc:    "min elements too few",
			schema:  validLeafListSchemaMin1,
			value:   []string{},
			wantErr: true,
		},
		{
			desc:    "min elements too few, nil value",
			schema:  validLeafListSchemaMin1,
			value:   nil,
			wantErr: true,
		},
		{
			desc:   "max elements success",
			schema: validLeafListSchemaMax3,
			value:  []string{"a"},
		},
		{
			desc:    "max elements too many",
			schema:  validLeafListSchemaMax3,
			value:   []string{"a", "b", "c", "d"},
			wantErr: true,
		},
		{
			desc:   "min/max elements success",
			schema: validLeafListSchemaMin1Max3,
			value:  []string{"a"},
		},
		{
			desc:    "min/max elements too few",
			schema:  validLeafListSchemaMin1Max3,
			value:   []string{},
			wantErr: true,
		},
		{
			desc:    "min/max elements too many",
			schema:  validLeafListSchemaMax3,
			value:   []string{"a", "b", "c", "d"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		err := validateListAttr(test.schema, test.value)
		// TODO(mostrowski): make consistent with rest of structs library.
		if got, want := (err != nil), test.wantErr; got != want {
			t.Errorf("%s: TestValidateListAttr(%v) got error: %v, wanted error? %v", test.desc, test.schema, err, test.wantErr)
		}
		if err != nil {
			if testErrOutput {
				t.Logf("%s: %v", test.desc, err)
			}
		}
	}
}

func TestIsFakeRoot(t *testing.T) {
	tests := []struct {
		name string
		in   *yang.Entry
		want bool
	}{
		{
			name: "explicitly true",
			in: &yang.Entry{
				Name: "entry",
				Annotation: map[string]interface{}{
					"isFakeRoot": true,
				},
			},
			want: true,
		},
		{
			name: "unspecified",
			in: &yang.Entry{
				Name: "entry",
			},
		},
	}

	for _, tt := range tests {
		if got := isFakeRoot(tt.in); got != tt.want {
			t.Errorf("%v: isFakeRoot(%v): did not get expected return value, got: %v, want: %v", tt.name, tt.in, got, tt.want)
		}
	}
}
