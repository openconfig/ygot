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
	"strings"
	"testing"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// pathNoKeysToGNMIPath converts the supplied path, which may not contain any
// keys, into a GNMI path.
func pathNoKeysToGNMIPath(path string) *gpb.Path {
	out := &gpb.Path{}
	for _, p := range strings.Split(path, "/") {
		out.Elem = append(out.Elem, &gpb.PathElem{Name: p})
	}
	return out
}

// gnmiPathNoKeysToPath converts the supplied GNMI path, which may not contain
// any keys, into a string slice path.
func gnmiPathNoKeysToPath(path *gpb.Path) string {
	if path == nil {
		return ""
	}
	out := ""
	for _, p := range path.Elem {
		out += p.Name + "/"
	}
	for strings.HasSuffix(out, "/") {
		out = strings.TrimSuffix(out, "/")
	}
	return out
}

func TestPathMatchesPrefix(t *testing.T) {
	tests := []struct {
		desc   string
		path   string
		prefix string
		want   bool
	}{
		{
			desc:   "empty",
			path:   "",
			prefix: "",
			want:   true,
		},
		{
			desc:   "root",
			path:   "/",
			prefix: "/",
			want:   true,
		},
		{
			desc:   "absolute",
			path:   "/a/b/c",
			prefix: "/a/b",
			want:   true,
		},
		{
			desc:   "relative",
			path:   "a/b/c/",
			prefix: "a/b/",
			want:   true,
		},
		{
			desc:   "relative, different trailing slash 1",
			path:   "a/b/c/",
			prefix: "a/b",
			want:   true,
		},
		{
			desc:   "relative, different trailing slash 2",
			path:   "a/b/c",
			prefix: "a/b/",
			want:   true,
		},
		{
			desc:   "relative vs absolute 1",
			path:   "/a/b/c/",
			prefix: "a/b/",
		},
		{
			desc:   "relative vs absolute 2",
			path:   "a/b/c/",
			prefix: "/a/b/",
		},
		{
			desc:   "prefix longer",
			path:   "a/b",
			prefix: "a/b/c",
		},
		{
			desc:   "not equal",
			path:   "a/b/c",
			prefix: "a/d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := pathMatchesPrefix(pathNoKeysToGNMIPath(tt.path), strings.Split(tt.prefix, "/")), tt.want; got != want {
				t.Errorf("%s: got: %v want: %v", tt.desc, got, want)
			}
		})
	}
}

func TestTrimGNMIPathPrefix(t *testing.T) {
	tests := []struct {
		desc   string
		path   string
		prefix string
		want   string
	}{
		{
			desc:   "empty",
			path:   "",
			prefix: "",
			want:   "",
		},
		{
			desc:   "root",
			path:   "/",
			prefix: "/",
			want:   "",
		},
		{
			desc:   "absolute",
			path:   "/a/b/c",
			prefix: "/a/b",
			want:   "c",
		},
		{
			desc:   "relative",
			path:   "a/b/c/",
			prefix: "a/b/",
			want:   "c",
		},
		{
			desc:   "relative, different trailing slash 1",
			path:   "a/b/c/",
			prefix: "a/b",
			want:   "c",
		},
		{
			desc:   "relative, different trailing slash 2",
			path:   "a/b/c",
			prefix: "a/b/",
			want:   "c",
		},
		{
			desc:   "prefix longer",
			path:   "a/b",
			prefix: "a/b/c",
			want:   "a/b",
		},
		{
			desc:   "not equal",
			path:   "a/b/c",
			prefix: "a/d",
			want:   "a/b/c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			path := pathNoKeysToGNMIPath(tt.path)
			prefix := strings.Split(tt.prefix, "/")
			got := gnmiPathNoKeysToPath(trimGNMIPathPrefix(path, prefix))
			if got != tt.want {
				t.Errorf("%s: got: %s want: %s", tt.desc, got, tt.want)
			}
		})
	}
}

func TestPopGNMIPath(t *testing.T) {
	tests := []struct {
		desc string
		path string
		want string
	}{
		{
			desc: "empty",
			path: "",
			want: "",
		},
		{
			desc: "root",
			path: "/",
			want: "",
		},
		{
			desc: "absolute",
			path: "/a/b/c",
			want: "a/b/c",
		},
		{
			desc: "relative",
			path: "a/b/c/",
			want: "b/c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := gnmiPathNoKeysToPath(popGNMIPath(pathNoKeysToGNMIPath(tt.path))), tt.want; got != want {
				t.Errorf("%s: got: %s want: %s", tt.desc, got, want)
			}
		})
	}
}
