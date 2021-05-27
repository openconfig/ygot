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

package testutil

import (
	"strings"
	"testing"
)

func TestGenerateUnifiedDiff(t *testing.T) {
	tests := []struct {
		name           string
		inWant         string
		inGot          string
		wantDiffSubstr string
	}{{
		name:           "basic",
		inWant:         "hello, world!",
		inGot:          "Hello, world",
		wantDiffSubstr: "-hello, world!\n+Hello, world",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff, _ := GenerateUnifiedDiff(tt.inWant, tt.inGot); !strings.Contains(diff, tt.wantDiffSubstr) {
				t.Errorf("expected diff to contain %q\nbut got %q", tt.wantDiffSubstr, diff)
			}
		})
	}
}
