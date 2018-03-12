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

package validate

import (
	"testing"

	oc "github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/ygot"
)

// TestRace performs operations on generated code which may be executed
// concurrently, it checks for race conditions when run with go test -race.
func TestRace(t *testing.T) {
	d1 := &oc.Device{}
	d1.GetOrCreateSystem().Hostname = ygot.String("dev1")

	d2 := &oc.Device{}
	d2.GetOrCreateSystem().Hostname = ygot.String("dev2")

	tests := []struct {
		name string
		fn   func(*oc.Device, *testing.T)
	}{{
		name: "EmitJSON",
		fn: func(d *oc.Device, t *testing.T) {
			if _, err := ygot.EmitJSON(d, nil); err != nil {
				t.Errorf("could not emit JSON - unexpected err, got: %v, want: nil", err)
			}
		},
	}, {
		name: "Validate",
		fn: func(d *oc.Device, t *testing.T) {
			if err := d.Validate(); err != nil {
				t.Errorf("could not validate device - unexpected err, got: %v, want: nil", err)
			}
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go tt.fn(d1, t)
			go tt.fn(d2, t)
		})
	}
}
