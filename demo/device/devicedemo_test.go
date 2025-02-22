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
	"os"
	"path/filepath"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/ygot/testutil"
)

const (
	// TestRoot overrides the root path at which this test is running.
	TestRoot string = ""
)

// TestDeviceDemo is a simple test which compares the output of the device demo
// to a known good configuration.
func TestDeviceDemo(t *testing.T) {

	dev, err := CreateDemoDeviceInstance()
	if err != nil {
		t.Fatalf("TestDeviceDemo: CreateDemoDeviceInstance(): got error: %v, want: nil", err)
	}

	gotjson, err := EmitJSON(dev)
	if err != nil {
		t.Fatalf("TestDeviceDemo: EmitJSON(%#v): got unexpected error, got: %v, want: nil", dev, err)
	}

	gotietfjson, err := EmitRFC7951JSON(dev)
	if err != nil {
		t.Fatalf("TestDeviceDemo: EmitJSON(%#v): got unexpected error, got: %v, want: nil", dev, err)
	}

	tests := []struct {
		name     string
		got      string
		wantFile string
	}{{
		name:     "internal JSON",
		got:      gotjson,
		wantFile: "testdata/device.json",
	}, {
		name:     "rfc7951 JSON",
		got:      gotietfjson,
		wantFile: "testdata/device-ietf.json",
	}}

	for _, tt := range tests {
		want, ioerr := os.ReadFile(filepath.Join(TestRoot, tt.wantFile))
		if ioerr != nil {
			t.Fatalf("%s: TestDeviceDemo: os.ReadFile(%s/%s): could not open file: %v", tt.name, TestRoot, tt.wantFile, ioerr)
		}

		if diff := pretty.Compare(tt.got, string(want)); diff != "" {
			if diffl, err := testutil.GenerateUnifiedDiff(tt.got, string(want)); err == nil {
				diff = diffl
			}
			t.Errorf("%s: TestDeviceDemo: CreateDemoDeviceInstance(): got incorrect output, diff(-got,+want):\n%s", tt.name, diff)
		}
	}
}
