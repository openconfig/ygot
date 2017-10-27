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
	// TestRoot is the path at which this test is running such that it
	// is possible to determine where to load test files from.
	TestRoot string = ""
)

// generateUnifiedDiff takes two strings and generates a diff that can be
// shown to the user in a test error message.
func generateUnifiedDiff(want, got string) (string, error) {
	diffl := difflib.UnifiedDiff{
		A:        difflib.SplitLines(want),
		B:        difflib.SplitLines(got),
		FromFile: "got",
		ToFile:   "want",
		Context:  3,
		Eol:      "\n",
	}
	return difflib.GetUnifiedDiffString(diffl)
}

// TestBGPDemo is a simple test which compares the output of the BGP demo
// to a known good configuration. It is intended as an integration test
// for the code generation pipeline used for the OpenConfig models, and
// to detect regression bugs prior to generator code directly utilising
// the set of libraries making up the OpenConfig struct code base.
func TestBGPDemo(t *testing.T) {

	bgp, err := CreateDemoBGPInstance()
	if err != nil {
		t.Fatalf("TestBGPDemo: CreateDemoBGPInstance(): got error: %v, want: nil", err)
	}

	got, err := EmitBGPJSON(bgp)
	if err != nil {
		t.Fatalf("TestBGPDemo: EmitBGPJSON(%v): got error: %v, want: nil", bgp, err)
	}

	gotietf, err := EmitRFC7951JSON(bgp)
	if err != nil {
		t.Fatalf("TestBGPDemo: EmitRFC7951JSON(%v): got error: %v, want: nil", bgp, err)
	}

	tests := []struct {
		name     string
		got      string
		wantFile string
	}{{
		name:     "internal JSON",
		got:      got,
		wantFile: "testdata/bgp.json",
	}, {
		name:     "RFC7951 JSON",
		got:      gotietf,
		wantFile: "testdata/bgp-ietf.json",
	}}

	for _, tt := range tests {
		want, ioerr := ioutil.ReadFile(filepath.Join(TestRoot, tt.wantFile))
		if ioerr != nil {
			t.Fatalf("TestBGPDemo %s: ioutil.ReadFile(%s/%s): could not open file: %v", tt.name, TestRoot, tt.wantFile, ioerr)
		}

		if diff := pretty.Compare(tt.got, string(want)); diff != "" {
			if diffl, err := generateUnifiedDiff(tt.got, string(want)); err == nil {
				diff = diffl
			}
			t.Errorf("TestBGPDemo %s: CreateDemoBGPInstance(): got incorrect output using structs lib, diff(-got,+want):\n%s", tt.name, diff)
		}
	}
}
