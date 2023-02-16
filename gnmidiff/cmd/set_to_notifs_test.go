// Copyright 2023 Google Inc.
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

package cmd

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSplitByEmptyline(t *testing.T) {
	tests := []struct {
		desc      string
		inBytes   []byte
		wantSplit [][]byte
	}{{
		desc:    "single line with newlines",
		inBytes: []byte("foo\nbar\nbaz"),
		wantSplit: [][]byte{
			[]byte("foo\nbar\nbaz"),
		},
	}, {
		desc:    "empty newlines",
		inBytes: []byte("foo\n\nbar\nbaz\n\n\nboo"),
		wantSplit: [][]byte{
			[]byte("foo"),
			[]byte("bar\nbaz"),
			[]byte("boo"),
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := splitByEmptyline(tt.inBytes)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.wantSplit, got); diff != "" {
				t.Errorf("splitByEmptyline (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestNotifsFromFile(t *testing.T) {
	tests := []struct {
		desc       string
		inFile     string
		wantNotifs int
	}{{
		desc:       "subscribeResponses",
		inFile:     "notifs.textproto",
		wantNotifs: 12,
	}, {
		desc:       "GetResponse",
		inFile:     "getresponse.textproto",
		wantNotifs: 12,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := notifsFromFile(tt.inFile)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.wantNotifs, len(got)); diff != "" {
				t.Errorf("notifsFromFile (-want, +got):\n%s", diff)
			}
		})
	}
}
