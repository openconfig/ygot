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

package ygen

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/gnmi/errdiff"
)

func TestGenerateIR(t *testing.T) {
	// TODO(robjs): Add test coverage.
	tests := []struct {
		desc             string
		inYANGFiles      []string
		inIncludePaths   []string
		inLangMapperFn   NewLangMapperFn
		inOpts           IROptions
		wantIR           *IR
		wantErrSubstring string
	}{{
		desc: "no error",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := GenerateIR(tt.inYANGFiles, tt.inIncludePaths, tt.inLangMapperFn, tt.inOpts)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}
			if diff := cmp.Diff(got, tt.wantIR, cmpopts.IgnoreUnexported()); diff != "" {
				t.Fatalf("did not get expected IR, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
