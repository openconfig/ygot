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

	"github.com/openconfig/ygot/ygot"
)

type ChoiceStruct struct {
	Case1Leaf1  *string `path:"case1-leaf1"`
	Case21Leaf1 *string `path:"case21-leaf"`
}

func (c *ChoiceStruct) IsYANGGoStruct() {}

func TestValidateChoice(t *testing.T) {
	containerWithChoiceSchema := &yang.Entry{
		Name: "container-with-choice-schema",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"choice1": {
				Kind: yang.ChoiceEntry,
				Name: "choice1",
				Dir: map[string]*yang.Entry{
					"case1": {
						Kind: yang.CaseEntry,
						Name: "case1",
						Dir: map[string]*yang.Entry{
							"case1-leaf1": {
								Kind: yang.LeafEntry,
								Name: "Case1Leaf1",
								Type: &yang.YangType{Kind: yang.Ystring},
							},
						},
					},
					"case2": {
						Kind: yang.CaseEntry,
						Name: "case2",
						Dir: map[string]*yang.Entry{
							"case2_choice1": {
								Kind: yang.ChoiceEntry,
								Name: "case2_choice1",
								Dir: map[string]*yang.Entry{
									"case21": {
										Kind: yang.CaseEntry,
										Name: "case21",
										Dir: map[string]*yang.Entry{
											"case21-leaf": {
												Kind: yang.LeafEntry,
												Name: "case21-leaf",
												Type: &yang.YangType{Kind: yang.Ystring},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		desc    string
		val     interface{}
		wantErr bool
	}{
		{
			desc: "success",
			val:  &ChoiceStruct{Case1Leaf1: ygot.String("Case1Leaf1Value")},
		},
		{
			desc:    "multiple cases selected",
			val:     &ChoiceStruct{Case1Leaf1: ygot.String("Case1Leaf1Value"), Case21Leaf1: ygot.String("Case21Leaf1Value")},
			wantErr: true,
		},
	}

	for _, test := range tests {
		errs := Validate(containerWithChoiceSchema, test.val)
		if got, want := (errs != nil), test.wantErr; got != want {
			t.Errorf("%s: Validate got error: %s, wanted error? %v", test.desc, errs, test.wantErr)
		}
		testErrLog(t, test.desc, errs)
	}
}
