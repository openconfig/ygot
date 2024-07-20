// Copyright 2020 Google Inc.
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

package validate

import (
	"os"
	"path/filepath"
	"testing"

	oc "github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/ygot"
)

func BenchmarkDiff(b *testing.B) {
	jsonFileA := "interfaceBenchmarkA.json"
	jsonFileB := "interfaceBenchmarkB.json"

	jsonA, err := os.ReadFile(filepath.Join(testRoot, "testdata", jsonFileA))
	if err != nil {
		b.Errorf("os.ReadFile(%s): could not open file: %v", jsonFileA, err)
		return
	}
	deviceA := &oc.Device{}
	if err := oc.Unmarshal(jsonA, deviceA); err != nil {
		b.Errorf("os.ReadFile(%s): could unmarschal: %v", jsonFileA, err)
		return
	}

	jsonB, err := os.ReadFile(filepath.Join(testRoot, "testdata", jsonFileB))
	if err != nil {
		b.Errorf("os.ReadFile(%s): could not open file: %v", jsonFileB, err)
		return
	}
	deviceB := &oc.Device{}
	if err := oc.Unmarshal(jsonB, deviceB); err != nil {
		b.Errorf("os.ReadFile(%s): could unmarschal: %v", jsonFileB, err)
		return
	}
	_, err = ygot.Diff(deviceA, deviceB)
	if err != nil {
		b.Errorf("Error in diff %v", err)
		return
	}
}
