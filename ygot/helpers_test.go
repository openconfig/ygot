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

package ygot

import (
	"reflect"
	"testing"
)

func TestToPtr(t *testing.T) {
	s := "foo"
	i := uint32(42)

	tests := []struct {
		name string
		in   interface{}
		want interface{}
	}{{
		name: "string",
		in:   s,
		want: &s,
	}, {
		name: "uint32",
		in:   i,
		want: &i,
	}}

	for _, tt := range tests {
		if got := ToPtr(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: ToPtr(%v): did not get expected ptr, got: %v, want: %v", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestBinaryToFloat32(t *testing.T) {
	tests := []struct {
		name string
		in   Binary
		want float32
	}{{
		name: "basic",
		// 01010000100101010000001011111001
		in:   Binary{80, 149, 2, 249},
		want: 2e+10,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BinaryToFloat32(tt.in); got != tt.want {
				t.Errorf("BinaryToFloat32(%v): got %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
