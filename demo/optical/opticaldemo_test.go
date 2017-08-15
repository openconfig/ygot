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
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/pmezard/go-difflib/difflib"
)

const (
	// TestRoot indicates the base directory within which this test is running
	// such that local files can be loaded.
	TestRoot string = ""
)

// generateUnifiedDiff takes two strings and generates a diff that can be
// shown to the user in a test error message.
func generateUnifiedDiff(want, got string) (string, error) {
	diffl := difflib.UnifiedDiff{
		A:        difflib.SplitLines(got),
		B:        difflib.SplitLines(want),
		FromFile: "got",
		ToFile:   "want",
		Context:  3,
		Eol:      "\n",
	}
	return difflib.GetUnifiedDiffString(diffl)
}

// TestOpticalDemo is a simple test which compares the output of the device demo
// to a known good configuration.
func TestOpticalDemoJSON(t *testing.T) {

	inst, err := CreateDemoOpticalInstance()
	if err != nil {
		t.Fatalf("TestOpticalDemo: CreateDemoOpticalInstance(): got error: %v, want: nil", err)
	}

	got, err := OutputJSON(inst)
	if err != nil {
		t.Fatalf("TestOpticalDemo: CreateDemoOpticalInstance(): got error: %v, want: nil", err)
	}

	gotietf, err := OutputIETFJSON(inst)
	if err != nil {
		t.Fatalf("TestOpticalDemo: CreateDemoOpticalInstance(): got error in IETF JSON output: %v, want: nil", err)
	}

	tests := []struct {
		name, wantFile string
		got            string
	}{{
		name:     "ietf-json",
		wantFile: "optical-ietf.json",
		got:      gotietf,
	}, {
		name:     "json",
		wantFile: "optical.json",
		got:      got,
	}}

	for _, tt := range tests {
		want, err := ioutil.ReadFile(filepath.Join(TestRoot, "testdata", tt.wantFile))
		if err != nil {
			t.Errorf("ioutil.ReadFile(%s/testdata/%s): could not open file: %v", TestRoot, tt.wantFile, err)
			continue
		}
		if diff := pretty.Compare(tt.got, string(want)); diff != "" {
			if diffl, err := generateUnifiedDiff(tt.got, string(want)); err == nil {
				diff = diffl
			}
			t.Errorf("TestOpticalDemo %s: CreateDemoOpticalInstance(): got incorrect output, diff(-got,+want):\n%s", tt.name, diff)
		}
	}
}
