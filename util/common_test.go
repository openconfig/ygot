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

	"github.com/golang/protobuf/proto"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// pathNoKeysToGNMIPath converts the supplied path, which may not contain any
// keys, into a GNMI path. We cannot use the ygot helpers such that we avoid
// a circular dependency with the util package.
func pathNoKeysToGNMIPath(path string) *gpb.Path {
	out := &gpb.Path{}
	for _, p := range strings.Split(path, "/") {
		out.Elem = append(out.Elem, &gpb.PathElem{Name: p})
	}
	return out
}

// gnmiPathNoKeysToPath converts the supplied GNMI path, which may not contain
// any keys, into a string slice path. We cannot use the ygot helpers such that
// we avoid a circular dependency with the util package.
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
			if got, want := PathMatchesPrefix(pathNoKeysToGNMIPath(tt.path), strings.Split(tt.prefix, "/")), tt.want; got != want {
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
			got := gnmiPathNoKeysToPath(TrimGNMIPathPrefix(path, prefix))
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
			if got, want := gnmiPathNoKeysToPath(PopGNMIPath(pathNoKeysToGNMIPath(tt.path))), tt.want; got != want {
				t.Errorf("%s: got: %s want: %s", tt.desc, got, want)
			}
		})
	}
}

func TestPathMatchesPathElemPrefix(t *testing.T) {
	tests := []struct {
		desc     string
		inPath   *gpb.Path
		inPrefix *gpb.Path
		want     bool
	}{{
		desc: "valid prefix with no keys",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
			}, {
				Name: "two",
			}},
		},
		inPrefix: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
			}},
		},
		want: true,
	}, {
		desc: "valid prefix with keys",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
				Key:  map[string]string{"two": "three"},
			}, {
				Name: "four",
			}},
		},
		inPrefix: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
				Key:  map[string]string{"two": "three"},
			}},
		},
		want: true,
	}, {
		desc: "not a prefix",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "fourteen",
			}, {
				Name: "twelve",
			}},
		},
		inPrefix: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "three",
			}},
		},
	}, {
		desc: "not a prefix due to origin",
		inPath: &gpb.Path{
			Origin: "openconfig",
			Elem: []*gpb.PathElem{{
				Name: "one",
			}, {
				Name: "two",
			}},
		},
		inPrefix: &gpb.Path{
			Origin: "google",
			Elem: []*gpb.PathElem{{
				Name: "one",
			}},
		},
	}, {
		desc: "not a prefix due to keys",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "three",
				Key:  map[string]string{"four": "five"},
			}, {
				Name: "six",
			}},
		},
		inPrefix: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "three",
				Key:  map[string]string{"seven": "eight"},
			}},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := PathMatchesPathElemPrefix(tt.inPath, tt.inPrefix); got != tt.want {
				t.Fatalf("did not get expected result, got: %v, want: %v", got, tt.want)
			}
		})
	}
}

func TestTrimGNMIPathElemPrefix(t *testing.T) {
	tests := []struct {
		desc     string
		inPath   *gpb.Path
		inPrefix *gpb.Path
		want     *gpb.Path
	}{{
		desc: "not a prefix",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
			}, {
				Name: "two",
			}},
		},
		inPrefix: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "three",
			}},
		},
		want: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
			}, {
				Name: "two",
			}},
		},
	}, {
		desc: "prefix with keys",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
				Key:  map[string]string{"two": "three"},
			}, {
				Name: "four",
			}},
		},
		inPrefix: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
				Key:  map[string]string{"two": "three"},
			}},
		},
		want: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "four",
			}},
		},
	}, {
		desc: "prefix longer",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "short",
			}},
		},
		inPrefix: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "short",
			}, {
				Name: "long",
			}},
		},
		want: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "short",
			}},
		},
	}, {
		desc: "nil prefix",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "foo",
			}},
		},
		want: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "foo",
			}},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := TrimGNMIPathElemPrefix(tt.inPath, tt.inPrefix); !proto.Equal(got, tt.want) {
				t.Fatalf("did not get expected path, got: %s, want: %s", proto.MarshalTextString(got), proto.MarshalTextString(tt.want))
			}
		})
	}
}

func TestFindPathElemPrefix(t *testing.T) {
	tests := []struct {
		name    string
		inPaths []*gpb.Path
		want    *gpb.Path
	}{{
		name: "no common prefix",
		inPaths: []*gpb.Path{
			pathNoKeysToGNMIPath("one/two"),
			pathNoKeysToGNMIPath("three/four"),
		},
	}, {
		name: "common prefix two paths",
		inPaths: []*gpb.Path{
			pathNoKeysToGNMIPath("one/two"),
			pathNoKeysToGNMIPath("one/two/three/four/five"),
		},
		want: pathNoKeysToGNMIPath("one/two"),
	}, {
		name: "common prefix three paths, none match",
		inPaths: []*gpb.Path{
			pathNoKeysToGNMIPath("one/two/three"),
			pathNoKeysToGNMIPath("one/two/four"),
			pathNoKeysToGNMIPath("one/two/five"),
		},
		want: pathNoKeysToGNMIPath("one/two"),
	}}

	for _, tt := range tests {
		if got := FindPathElemPrefix(tt.inPaths); !proto.Equal(got, tt.want) {
			t.Errorf("%s: FindPathElemPrefix(%v): did not get expected prefix, got: %s, want: %s", tt.name, tt.inPaths, proto.MarshalTextString(got), proto.MarshalTextString(tt.want))
		}
	}
}
