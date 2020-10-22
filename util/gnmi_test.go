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

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

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
			if got := PathElemsEqual(tt.lhs, tt.rhs); got != tt.want {
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
		if got := FindPathElemPrefix(tt.inPaths); !proto.Equal(got, tt.want) {
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
		got, err := FindModelData(tt.in)

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
