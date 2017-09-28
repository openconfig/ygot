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

package util

import (
	"testing"
)

func TestIsValueNil(t *testing.T) {
	if !IsValueNil(nil) {
		t.Error("got IsValueNil(nil) false, want true")
	}
	if !IsValueNil((*int)(nil)) {
		t.Error("got IsValueNil(ptr) false, want true")
	}
	if !IsValueNil((map[int]int)(nil)) {
		t.Error("got IsValueNil(map) false, want true")
	}
	if !IsValueNil(([]int)(nil)) {
		t.Error("got IsValueNil(slice) false, want true")
	}
	if !IsValueNil((interface{})(nil)) {
		t.Error("got IsValueNil(interface) false, want true")
	}

	if IsValueNil(toInt8Ptr(42)) {
		t.Error("got IsValueNil(ptr) true, want false")
	}
	if IsValueNil(map[int]int{42: 42}) {
		t.Error("got IsValueNil(map) true, want false")
	}
	if IsValueNil([]int{1, 2, 3}) {
		t.Error("got IsValueNil(slice) true, want false")
	}
	if IsValueNil((interface{})(42)) {
		t.Error("got IsValueNil(interface) true, want false")
	}
}
