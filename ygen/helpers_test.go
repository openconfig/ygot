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

package ygen

import (
	"testing"
)

// TestSafeGoEnumeratedValueName tests the safeGoEnumeratedValue function to ensure
// that enumeraton value names are correctly transformed to safe Go names.
func TestSafeGoEnumeratedValueName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"SPEED_2.5G", "SPEED_2_5G"},
		{"IPV4-UNICAST", "IPV4_UNICAST"},
		{"frameRelay", "frameRelay"},
		{"coffee", "coffee"},
		{"ethernetCsmacd", "ethernetCsmacd"},
		{"SFP+", "SFP_PLUS"},
		{"LEVEL1/2", "LEVEL1_2"},
		{"DAYS1-3", "DAYS1_3"},
		{"FISH CHIPS", "FISH_CHIPS"},
		{"FOO*", "FOO_ASTERISK"},
		{"FOO:", "FOO_COLON"},
		{",,FOO:@$,", "_COMMA_COMMAFOO_COLON_AT_DOLLAR_COMMA"},
	}

	for _, tt := range tests {
		got := safeGoEnumeratedValueName(tt.in)
		if got != tt.want {
			t.Errorf("safeGoEnumeratedValueName(%s): got: %s, want: %s", tt.in, got, tt.want)
		}
	}
}

func TestResolveRootName(t *testing.T) {
	tests := []struct {
		name           string
		inName         string
		inDefName      string
		inGenerateRoot bool
		want           string
	}{{
		name:           "generate root false",
		inGenerateRoot: false,
	}, {
		name:           "name specified",
		inName:         "value",
		inDefName:      "invalid",
		inGenerateRoot: true,
		want:           "value",
	}, {
		name:           "name not specified",
		inDefName:      "default",
		inGenerateRoot: true,
		want:           "default",
	}}

	for _, tt := range tests {
		if got := resolveRootName(tt.inName, tt.inDefName, tt.inGenerateRoot); got != tt.want {
			t.Errorf("%s: resolveRootName(%s, %s, %v): did not get expected result, got: %s, want: %s", tt.name, tt.inName, tt.inDefName, tt.inGenerateRoot, got, tt.want)
		}
	}
}
