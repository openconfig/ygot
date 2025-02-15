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

package util_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

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
			if got, want := util.PathMatchesPrefix(pathNoKeysToGNMIPath(tt.path), strings.Split(tt.prefix, "/")), tt.want; got != want {
				t.Errorf("%s: got: %v want: %v", tt.desc, got, want)
			}
		})
	}
}

func TestPathPartiallyMatchesPrefix(t *testing.T) {
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
			want:   false,
		},
		{
			desc:   "relative vs absolute 2",
			path:   "a/b/c/",
			prefix: "/a/b/",
			want:   false,
		},
		{
			desc:   "prefix longer",
			path:   "a/b",
			prefix: "a/b/c",
			want:   true,
		},
		{
			desc:   "prefix longer 2",
			path:   "/a",
			prefix: "/a/b/c",
			want:   true,
		},
		{
			desc:   "not equal",
			path:   "a/b/c",
			prefix: "a/d",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := util.PathPartiallyMatchesPrefix(pathNoKeysToGNMIPath(tt.path), strings.Split(tt.prefix, "/")), tt.want; got != want {
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
			got := gnmiPathNoKeysToPath(util.TrimGNMIPathPrefix(path, prefix))
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
			if got, want := gnmiPathNoKeysToPath(util.PopGNMIPath(pathNoKeysToGNMIPath(tt.path))), tt.want; got != want {
				t.Errorf("%s: got: %s want: %s", tt.desc, got, want)
			}
		})
	}
}

func TestPathElemsEqual(t *testing.T) {
	tests := []struct {
		desc string
		lhs  *gpb.PathElem
		rhs  *gpb.PathElem
		want bool
	}{{
		desc: "equal names with no keys",
		lhs: &gpb.PathElem{
			Name: "one",
		},
		rhs: &gpb.PathElem{
			Name: "one",
		},
		want: true,
	}, {
		desc: "equal names and keys",
		lhs: &gpb.PathElem{
			Name: "one",
			Key:  map[string]string{"two": "three", "four": "five"},
		},
		rhs: &gpb.PathElem{
			Name: "one",
			Key:  map[string]string{"two": "three", "four": "five"},
		},
		want: true,
	}, {
		desc: "names don't match",
		lhs: &gpb.PathElem{
			Name: "one",
			Key:  map[string]string{"two": "three", "four": "five"},
		},
		rhs: &gpb.PathElem{
			Name: "two",
			Key:  map[string]string{"two": "three", "four": "five"},
		},
	}, {
		desc: "keys don't match",
		lhs: &gpb.PathElem{
			Name: "one",
			Key:  map[string]string{"two": "three", "four": "five"},
		},
		rhs: &gpb.PathElem{
			Name: "one",
			Key:  map[string]string{"two": "three", "four": "six"},
		},
	}, {
		desc: "keys don't have same length",
		lhs: &gpb.PathElem{
			Name: "one",
			Key:  map[string]string{"two": "three"},
		},
		rhs: &gpb.PathElem{
			Name: "one",
			Key:  map[string]string{"two": "three", "four": "five"},
		},
	}, {
		desc: "keys don't have same length the other way",
		lhs: &gpb.PathElem{
			Name: "one",
			Key:  map[string]string{"two": "three", "four": "five"},
		},
		rhs: &gpb.PathElem{
			Name: "one",
			Key:  map[string]string{"two": "three"},
		},
	}, {
		desc: "lhs PathElem is nil",
		lhs:  nil,
		rhs: &gpb.PathElem{
			Name: "one",
			Key:  map[string]string{"two": "three", "four": "five"},
		},
	}, {
		desc: "rhs PathElem is nil",
		lhs: &gpb.PathElem{
			Name: "one",
			Key:  map[string]string{"two": "three", "four": "five"},
		},
		rhs: nil,
	}, {
		desc: "both PathElems are nil",
		lhs:  nil,
		rhs:  nil,
		want: true,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := util.PathElemsEqual(tt.lhs, tt.rhs); got != tt.want {
				t.Fatalf("did not get expected result, got: %v, want: %v", got, tt.want)
			}
		})
	}
}

func TestPathElemSlicesEqual(t *testing.T) {
	tests := []struct {
		desc     string
		inElemsA []*gpb.PathElem
		inElemsB []*gpb.PathElem
		want     bool
	}{{
		desc: "equal elems with no keys",
		inElemsA: []*gpb.PathElem{{
			Name: "one",
		}, {
			Name: "two",
		}},
		inElemsB: []*gpb.PathElem{{
			Name: "one",
		}, {
			Name: "two",
		}},
		want: true,
	}, {
		desc: "equal elems with keys",
		inElemsA: []*gpb.PathElem{{
			Name: "one",
			Key:  map[string]string{"two": "three"},
		}, {
			Name: "four",
		}},
		inElemsB: []*gpb.PathElem{{
			Name: "one",
			Key:  map[string]string{"two": "three"},
		}, {
			Name: "four",
		}},
		want: true,
	}, {
		desc: "unequal elems",
		inElemsA: []*gpb.PathElem{{
			Name: "fourteen",
		}, {
			Name: "twelve",
		}},
		inElemsB: []*gpb.PathElem{{
			Name: "three",
		}},
		want: false,
	}, {
		desc: "unequal elems with keys",
		inElemsA: []*gpb.PathElem{{
			Name: "one",
			Key:  map[string]string{"two": "three"},
		}, {
			Name: "four",
			Key:  map[string]string{"five": "six"},
		}},
		inElemsB: []*gpb.PathElem{{
			Name: "one",
			Key:  map[string]string{"two": "three"},
		}, {
			Name: "eight",
			Key:  map[string]string{"five": "six"},
		}},
		want: false,
	}, {
		desc: "unequal elem length",
		inElemsA: []*gpb.PathElem{{
			Name: "one",
		}, {
			Name: "two",
		}},
		inElemsB: []*gpb.PathElem{{
			Name: "one",
		}},
		want: false,
	}, {
		desc: "unequal elems due to keys",
		inElemsA: []*gpb.PathElem{{
			Name: "three",
			Key:  map[string]string{"four": "five"},
		}, {
			Name: "six",
		}},
		inElemsB: []*gpb.PathElem{{
			Name: "three",
			Key:  map[string]string{"seven": "eight"},
		}, {
			Name: "six",
		}},
		want: false,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := util.PathElemSlicesEqual(tt.inElemsA, tt.inElemsB); got != tt.want {
				t.Fatalf("did not get expected result, got: %v, want: %v", got, tt.want)
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
			if got := util.PathMatchesPathElemPrefix(tt.inPath, tt.inPrefix); got != tt.want {
				t.Fatalf("did not get expected result, got: %v, want: %v", got, tt.want)
			}
		})
	}
}

func TestPathMatchesQuery(t *testing.T) {
	tests := []struct {
		desc    string
		inPath  *gpb.Path
		inQuery *gpb.Path
		want    bool
	}{{
		desc: "valid query with no keys",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
			}, {
				Name: "two",
			}},
		},
		inQuery: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
			}},
		},
		want: true,
	}, {
		desc: "valid query with implied openconfig origin path",
		inPath: &gpb.Path{
			Origin: "",
		},
		inQuery: &gpb.Path{
			Origin: "openconfig",
		},
		want: true,
	}, {
		desc: "valid query with implied openconfig origin query",
		inPath: &gpb.Path{
			Origin: "openconfig",
		},
		inQuery: &gpb.Path{
			Origin: "",
		},
		want: true,
	}, {
		desc: "valid query with wildcard name",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
			}, {
				Name: "two",
			}},
		},
		inQuery: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "*",
			}, {
				Name: "two",
			}},
		},
		want: true,
	}, {
		desc: "valid query with exact key match",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
				Key:  map[string]string{"two": "three"},
			}, {
				Name: "four",
			}},
		},
		inQuery: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
				Key:  map[string]string{"two": "three"},
			}},
		},
		want: true,
	}, {
		desc: "valid query with wildcard keys",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
				Key:  map[string]string{"two": "three"},
			}, {
				Name: "four",
			}},
		},
		inQuery: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
				Key:  map[string]string{"two": "*"},
			}},
		},
		want: true,
	}, {
		desc: "valid query with no keys and path with keys",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
				Key:  map[string]string{"two": "three"},
			}, {
				Name: "four",
			}},
		},
		inQuery: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
			}},
		},
		want: true,
	}, {
		desc: "valid query with both missing and wildcard keys",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
				Key: map[string]string{
					"two":  "three",
					"four": "five",
				},
			}, {
				Name: "four",
			}},
		},
		inQuery: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
				Key:  map[string]string{"four": "*"},
			}},
		},
		want: true,
	}, {
		desc: "invalid nil elements",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{
				nil,
				{
					Name: "twelve",
				}},
		},
		inQuery: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "three",
			}},
		},
	}, {
		desc: "invalid longer query",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{
				{
					Name: "twelve",
				}},
		},
		inQuery: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "one",
			}, {
				Name: "two",
			}},
		},
	}, {
		desc: "invalid names not equal",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "fourteen",
			}, {
				Name: "twelve",
			}},
		},
		inQuery: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "three",
			}},
		},
	}, {
		desc: "invalid origin",
		inPath: &gpb.Path{
			Origin: "openconfig",
			Elem: []*gpb.PathElem{{
				Name: "one",
			}, {
				Name: "two",
			}},
		},
		inQuery: &gpb.Path{
			Origin: "google",
			Elem: []*gpb.PathElem{{
				Name: "one",
			}},
		},
	}, {
		desc: "invalid keys",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "three",
				Key:  map[string]string{"four": "five"},
			}, {
				Name: "six",
			}},
		},
		inQuery: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "three",
				Key:  map[string]string{"seven": "eight"},
			}},
		},
	}, {
		desc: "invalid missing wildcard keys",
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "three",
				Key:  map[string]string{"four": "five"},
			}, {
				Name: "six",
			}},
		},
		inQuery: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "three",
				Key:  map[string]string{"seven": "*"},
			}},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := util.PathMatchesQuery(tt.inPath, tt.inQuery); got != tt.want {
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
			if got := util.TrimGNMIPathElemPrefix(tt.inPath, tt.inPrefix); !proto.Equal(got, tt.want) {
				t.Fatalf("did not get expected path, got: %s, want: %s", prototext.Format(got), prototext.Format(tt.want))
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
		if got := util.FindPathElemPrefix(tt.inPaths); !proto.Equal(got, tt.want) {
			t.Errorf("%s: FindPathElemPrefix(%v): did not get expected prefix, got: %s, want: %s", tt.name, tt.inPaths, prototext.Format(got), prototext.Format(tt.want))
		}
	}
}

func mustPathElem(s string) []*gpb.PathElem {
	p, err := stringToStructuredPath(s)
	if err != nil {
		panic(err)
	}
	return p.Elem
}

func TestPathElemsMatchQuery(t *testing.T) {
	tests := []struct {
		desc               string
		inRefElems         []*gpb.PathElem
		inMatchingElems    [][]*gpb.PathElem
		inNonMatchingElems [][]*gpb.PathElem
	}{{
		desc:       "no-wildcard, non-list path",
		inRefElems: mustPathElem("/alpha/bravo/charlie"),
		inMatchingElems: [][]*gpb.PathElem{
			mustPathElem("/alpha/bravo/charlie"),
			mustPathElem("/alpha/bravo/charlie/delta"),
			mustPathElem("/alpha/bravo/charlie/echo"),
		},
		inNonMatchingElems: [][]*gpb.PathElem{
			mustPathElem("/alpha/bravo/delta"),
			mustPathElem("/alpha/bravo/delta/charlie"),
			mustPathElem("/alpha/bravo/delta/echo"),
		},
	}, {
		desc:       "wildcard, non-list path",
		inRefElems: mustPathElem("/alpha/*/charlie"),
		inMatchingElems: [][]*gpb.PathElem{
			mustPathElem("/alpha/bravo/charlie"),
			mustPathElem("/alpha/zulu/charlie/delta"),
			mustPathElem("/alpha/yankee/charlie/echo"),
		},
		inNonMatchingElems: [][]*gpb.PathElem{
			mustPathElem("/alpha/bravo/delta"),
			mustPathElem("/alpha/zulu/delta/charlie"),
			mustPathElem("/bravo/yankee/charlie/echo"),
		},
	}, {
		desc:       "no-wildcard, list path",
		inRefElems: mustPathElem("/alpha/bravo[key=value]/charlie"),
		inMatchingElems: [][]*gpb.PathElem{
			mustPathElem("/alpha/bravo[key=value]/charlie"),
			mustPathElem("/alpha/bravo[key=value]/charlie/delta"),
		},
		inNonMatchingElems: [][]*gpb.PathElem{
			mustPathElem("/alpha/bravo[key=value2]/charlie"),
			mustPathElem("/alpha/bravo[key=value2]/charlie/echo"),
			mustPathElem("/alpha/bravo/charlie"),
			mustPathElem("/alpha/bravo/charlie/echo"),
		},
	}, {
		desc:       "wildcard, list path",
		inRefElems: mustPathElem("/alpha/bravo[key=*]/charlie"),
		inMatchingElems: [][]*gpb.PathElem{
			mustPathElem("/alpha/bravo[key=value]/charlie"),
			mustPathElem("/alpha/bravo[key=value]/charlie/delta"),
			mustPathElem("/alpha/bravo[key=value2]/charlie"),
			mustPathElem("/alpha/bravo[key=value2]/charlie/echo"),
		},
		inNonMatchingElems: [][]*gpb.PathElem{
			mustPathElem("/alpha/bravo/charlie"),
			mustPathElem("/alpha/bravo/charlie/foxtrot"),
			mustPathElem("/alpha/bravo/charlie"),
			mustPathElem("/alpha/bravo/charlie/echo"),
		},
	}, {
		desc:       "multi-wildcard, list path",
		inRefElems: mustPathElem("/alpha[asn=15169]/bravo[key=*]/*/delta[name=*]/echo"),
		inMatchingElems: [][]*gpb.PathElem{
			mustPathElem("/alpha[asn=15169]/bravo[key=tincan][key2=kale]/charlie[k=v]/delta[name=lamp]/echo[a=b]/"),
			mustPathElem("/alpha[asn=15169]/bravo[key=tincan]/charlie/delta[name=lamp]/echo/"),
			mustPathElem("/alpha[asn=15169]/bravo[key=tincan]/whiskey/delta[name=lamp]/echo/"),
			mustPathElem("/alpha[asn=15169]/bravo[key=tincan]/charlie/delta[name=lamp]/echo/a[name=bulb]/b/c"),
			mustPathElem("/alpha[asn=15169]/bravo[key=tincan]/charlie/delta[name=lamp]/echo/f[name=bulb]"),
		},
		inNonMatchingElems: [][]*gpb.PathElem{
			mustPathElem("/alpha[asn=30]/bravo[key=tincan]/charlie/delta[name=lamp]/echo/b/c[name=bulb]/d"),
			mustPathElem("/alpha[asn=15169]/bravo/charlie/delta[name=lamp]/echo/f[name=bulb]"),
			mustPathElem("/quebec[asn=15169]/bravo/charlie/delta[name=lamp]/echo/f[name=bulb]"),
			mustPathElem("/alpha[password=15169]/bravo[key=tincan]/charlie/delta[name=lamp]/echo/"),
			mustPathElem("/alpha/bravo[key=tincan]/charlie/delta[name=lamp]/echo/f[name=bulb]"),
			mustPathElem("/alpha/bravo[key=tincan]/charlie/delta[name=lamp]/echo/f[name=bulb]"),
			mustPathElem("/alpha[asn=15169]/bravo[key=tincan]/charlie/delta/echo/f[name=bulb]"),
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for _, matchElems := range tt.inMatchingElems {
				if !PathElemsMatchQuery(matchElems, tt.inRefElems) {
					t.Errorf("unexpected non-matching result for %v\nreference path elems: %v", matchElems, tt.inRefElems)
				}
			}
			for _, nonMatchElems := range tt.inNonMatchingElems {
				if PathElemsMatchQuery(nonMatchElems, tt.inRefElems) {
					t.Errorf("unexpected matching result for %v\nreference path elems: %v", nonMatchElems, tt.inRefElems)
				}
			}
		})
	}
}

func TestFindModelData(t *testing.T) {
	tests := []struct {
		name             string
		in               []*yang.Entry
		want             []*gpb.ModelData
		wantErrSubstring string
	}{{
		name: "single model with organization and version",
		in: []*yang.Entry{{
			Name: "module-one",
			Node: &yang.Module{
				Name: "module-one",
				Organization: &yang.Value{
					Source: &yang.Statement{
						Keyword:     "organization",
						HasArgument: true,
						Argument:    "openconfig",
					},
				},
				Extensions: []*yang.Statement{{
					Keyword:  "oc-ext:openconfig-version",
					Argument: "0.1.0",
				}},
			},
		}},
		want: []*gpb.ModelData{{
			Name:         "module-one",
			Organization: "openconfig",
			Version:      "0.1.0",
		}},
	}, {
		name: "multiple models with organization and version",
		in: []*yang.Entry{{
			Name: "module-one",
			Node: &yang.Module{
				Name: "module-one",
				Organization: &yang.Value{
					Source: &yang.Statement{
						Keyword:     "organization",
						HasArgument: true,
						Argument:    "openconfig",
					},
				},
				Extensions: []*yang.Statement{{
					Keyword:  "oc-foo:openconfig-version", // different import prefix
					Argument: "0.1.0",
				}},
			},
		}, {
			Name: "module-two",
			Node: &yang.Module{
				Name: "module-two",
				Organization: &yang.Value{
					Source: &yang.Statement{
						Keyword:     "organization",
						HasArgument: true,
						Argument:    "closedconfig",
					},
				},
				Extensions: []*yang.Statement{{
					Keyword:  "oc-ext:openconfig-version",
					Argument: "0.4.0",
				}},
			},
		}},
		want: []*gpb.ModelData{{
			Name:         "module-one",
			Organization: "openconfig",
			Version:      "0.1.0",
		}, {
			Name:         "module-two",
			Organization: "closedconfig",
			Version:      "0.4.0",
		}},
	}, {
		name: "nil organization and extension",
		in: []*yang.Entry{{
			Name: "module-one",
			Node: &yang.Module{
				Name: "module-one",
			},
		}},
		want: []*gpb.ModelData{{
			Name: "module-one",
		}},
	}, {
		name: "non-module in node",
		in: []*yang.Entry{{
			Name: "module-one",
			Node: &yang.Leaf{},
		}},
		wantErrSubstring: "nil node, or not a module",
	}, {
		name: "nil node",
		in: []*yang.Entry{{
			Name: "badmod",
		}},
		wantErrSubstring: "nil node, or not a module",
	}}

	for _, tt := range tests {
		got, err := util.FindModelData(tt.in)

		if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
			t.Errorf("%s: FindModelData(%v): did not get expected error, %s", tt.name, tt.in, diff)
		}

		if err != nil {
			continue
		}

		if diff := cmp.Diff(tt.want, got, cmp.Comparer(proto.Equal)); diff != "" {
			t.Errorf("%s: FindModelData(%v): did not get expected result, diff(-want, +got):\n%s", tt.name, tt.in, diff)
		}
	}
}

func TestJoinPaths(t *testing.T) {
	tests := []struct {
		desc                 string
		prefix, suffix, want *gpb.Path
		wantErrSubstring     string
	}{{
		desc:   "all empty",
		prefix: &gpb.Path{},
		suffix: &gpb.Path{},
		want:   &gpb.Path{},
	}, {
		desc:   "prefix only",
		prefix: &gpb.Path{Origin: "o", Target: "t", Elem: []*gpb.PathElem{{Name: "p"}}},
		suffix: &gpb.Path{},
		want:   &gpb.Path{Origin: "o", Target: "t", Elem: []*gpb.PathElem{{Name: "p"}}},
	}, {
		desc:   "suffix only",
		prefix: &gpb.Path{},
		suffix: &gpb.Path{Origin: "o", Target: "t", Elem: []*gpb.PathElem{{Name: "s"}}},
		want:   &gpb.Path{Origin: "o", Target: "t", Elem: []*gpb.PathElem{{Name: "s"}}},
	}, {
		desc:   "elements joined",
		prefix: &gpb.Path{Elem: []*gpb.PathElem{{Name: "p"}}},
		suffix: &gpb.Path{Elem: []*gpb.PathElem{{Name: "s"}}},
		want:   &gpb.Path{Elem: []*gpb.PathElem{{Name: "p"}, {Name: "s"}}},
	}, {
		desc:   "same origin and target",
		prefix: &gpb.Path{Origin: "o", Target: "t"},
		suffix: &gpb.Path{Origin: "o", Target: "t"},
		want:   &gpb.Path{Origin: "o", Target: "t"},
	}, {
		desc:             "mismatch origins",
		prefix:           &gpb.Path{Origin: "o1"},
		suffix:           &gpb.Path{Origin: "o2"},
		wantErrSubstring: "different origins",
	}, {
		desc:             "mismatch targets",
		prefix:           &gpb.Path{Target: "t1"},
		suffix:           &gpb.Path{Target: "t2"},
		wantErrSubstring: "different targets",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := util.JoinPaths(tt.prefix, tt.suffix)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Errorf("JoinPaths(%v, %v) got unexpected error diff: %s", tt.prefix, tt.suffix, diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("JoinPaths(%v, %v) got unexpected result diff(-want, +got): %s", tt.prefix, tt.suffix, diff)
			}
		})
	}
}

func mustStringToPath(t testing.TB, path string) *gpb.Path {
	p, err := ygot.StringToStructuredPath(path)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestComparePaths(t *testing.T) {
	tests := []struct {
		desc string
		a, b *gpb.Path
		want util.CompareRelation
	}{{
		desc: "different origins",
		a:    &gpb.Path{Origin: "foo"},
		b:    &gpb.Path{Origin: "bar"},
		want: util.Disjoint,
	}, {
		desc: "disjoint paths",
		a:    mustStringToPath(t, "/foo"),
		b:    mustStringToPath(t, "/bar"),
		want: util.Disjoint,
	}, {
		desc: "disjoint paths by list keys",
		a:    mustStringToPath(t, "/foo[a=1][b=2]"),
		b:    mustStringToPath(t, "/foo[a=1][b=3]"),
		want: util.Disjoint,
	}, {
		desc: "origin a openconfig and origin b blank",
		a:    &gpb.Path{Origin: "openconfig"},
		b:    &gpb.Path{Origin: ""},
		want: util.Equal,
	}, {
		desc: "origin a blank and origin b openconfig",
		a:    &gpb.Path{Origin: ""},
		b:    &gpb.Path{Origin: "openconfig"},
		want: util.Equal,
	}, {
		desc: "equal paths",
		a:    mustStringToPath(t, "/foo"),
		b:    mustStringToPath(t, "/foo"),
		want: util.Equal,
	}, {
		desc: "equal paths with list keys",
		a:    mustStringToPath(t, "/foo[a=1]"),
		b:    mustStringToPath(t, "/foo[a=1]"),
		want: util.Equal,
	}, {
		desc: "equal paths with implicit wildcards",
		a:    mustStringToPath(t, "/foo[a=*]"),
		b:    mustStringToPath(t, "/foo"),
		want: util.Equal,
	}, {
		desc: "equal paths with implicit wildcards",
		a:    mustStringToPath(t, "/foo"),
		b:    mustStringToPath(t, "/foo[a=*]"),
		want: util.Equal,
	}, {
		desc: "superset by length",
		a:    mustStringToPath(t, "/foo"),
		b:    mustStringToPath(t, "/foo/bar"),
		want: util.Superset,
	}, {
		desc: "superset by length and keys",
		a:    mustStringToPath(t, "/foo[a=*]"),
		b:    mustStringToPath(t, "/foo[b=1]/bar"),
		want: util.Superset,
	}, {
		desc: "superset by list keys",
		a:    mustStringToPath(t, "/foo[a=*]"),
		b:    mustStringToPath(t, "/foo[a=1]"),
		want: util.Superset,
	}, {
		desc: "superset by list keys implicit wildcard",
		a:    mustStringToPath(t, "/foo"),
		b:    mustStringToPath(t, "/foo[a=1]"),
		want: util.Superset,
	}, {
		desc: "subset by length",
		a:    mustStringToPath(t, "/foo/bar"),
		b:    mustStringToPath(t, "/foo"),
		want: util.Subset,
	}, {
		desc: "subset by length and keys",
		a:    mustStringToPath(t, "/foo[a=1]/bar"),
		b:    mustStringToPath(t, "/foo[b=*]"),
		want: util.Subset,
	}, {
		desc: "subset by list keys",
		a:    mustStringToPath(t, "/foo[a=1]"),
		b:    mustStringToPath(t, "/foo[a=*]"),
		want: util.Subset,
	}, {
		desc: "subset by list keys implicit wildcard",
		a:    mustStringToPath(t, "/foo[a=1]"),
		b:    mustStringToPath(t, "/foo"),
		want: util.Subset,
	}, {
		desc: "error single elem is both subset and superset",
		a:    mustStringToPath(t, "/foo[a=1][b=*]"),
		b:    mustStringToPath(t, "/foo[a=*][b=1]"),
		want: util.PartialIntersect,
	}, {
		desc: "error single elem is both subset and superset implicit wildcards",
		a:    mustStringToPath(t, "/foo[a=1]"),
		b:    mustStringToPath(t, "/foo[b=1]"),
		want: util.PartialIntersect,
	}, {
		desc: "error path is both subset and superset",
		a:    mustStringToPath(t, "/foo[a=*]/bar[b=1]"),
		b:    mustStringToPath(t, "/foo[b=1]/bar[b=*]"),
		want: util.PartialIntersect,
	}, {
		desc: "error path elem is both subset and superset",
		a:    mustStringToPath(t, "/foo[a=1]/bar[b=*]"),
		b:    mustStringToPath(t, "/foo[b=*]/bar[b=1]"),
		want: util.PartialIntersect,
	}, {
		desc: "error shorter path is both subset and superset",
		a:    mustStringToPath(t, "/foo[a=*]/bar"),
		b:    mustStringToPath(t, "/foo[b=1]"),
		want: util.PartialIntersect,
	}, {
		desc: "error path elem is both subset and superset",
		a:    mustStringToPath(t, "/foo[a=1]"),
		b:    mustStringToPath(t, "/foo[b=*]/bar"),
		want: util.PartialIntersect,
	}}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := util.ComparePaths(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("ComparePaths(%v, %v) got unexpected result: got %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
