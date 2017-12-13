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
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

// TestPathToString validates the functionality provided by the PathToString
// method of the library.
func TestPathToString(t *testing.T) {
	tests := []struct {
		name    string
		in      *gnmipb.Path
		want    string
		wantErr string
	}{{
		name: "root path",
		in:   &gnmipb.Path{Element: []string{}},
		want: "/",
	}, {
		name: "simple path parts",
		in:   &gnmipb.Path{Element: []string{"a", "b", "c", "d"}},
		want: "/a/b/c/d",
	}, {
		name:    "empty path segment",
		in:      &gnmipb.Path{Element: []string{"x", "", "y", "z"}},
		want:    "/x//y/z",
		wantErr: "empty element at index 1 in [x  y z]",
	}, {
		name: "path with attributes",
		in:   &gnmipb.Path{Element: []string{"q", "r[s=t]", "u"}},
		want: "/q/r[s=t]/u",
	}, {
		name: "root path in path elem",
		in:   &gnmipb.Path{Elem: []*gnmipb.PathElem{}},
		want: "/",
	}, {
		name: "simple path parts",
		in: &gnmipb.Path{Elem: []*gnmipb.PathElem{
			{Name: "a"},
			{Name: "b"},
			{Name: "c"},
			{Name: "d"},
		}},
		want: "/a/b/c/d",
	}, {
		name: "path with attributes",
		in: &gnmipb.Path{Elem: []*gnmipb.PathElem{
			{Name: "a", Key: map[string]string{"a": "b"}},
			{Name: "b", Key: map[string]string{"c": "d", "e": "f"}},
			{Name: "g"},
		}},
		want: "/a[a=b]/b[c=d][e=f]/g",
	}, {
		name: "structured path with empty element",
		in: &gnmipb.Path{Elem: []*gnmipb.PathElem{
			{Name: "a", Key: map[string]string{"a": "b"}},
			{Key: map[string]string{"c": "d"}},
		}},
		wantErr: "empty name for PathElem at index 1",
	}, {
		name: "structed path with empty key name",
		in: &gnmipb.Path{Elem: []*gnmipb.PathElem{
			{Name: "a", Key: map[string]string{"": "d"}},
		}},
		wantErr: "empty key name (value: d) in element a",
	}, {
		name: "both path types set",
		in: &gnmipb.Path{
			Element: []string{"one", "two", "three"},
			Elem: []*gnmipb.PathElem{{
				Name: "one",
			}, {
				Name: "three",
			}},
		},
		want: "/one/two/three", // should have the element type, not elem.
	}}

	for _, tt := range tests {
		got, err := PathToString(tt.in)
		if err != nil && !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("%s: PathToString(%v): did not get expected error, got: %v, want: %v", tt.name, tt.in, err, tt.wantErr)
		}

		if err != nil || tt.wantErr != "" {
			continue
		}

		if got != tt.want {
			t.Errorf("%s: PathToString(%v): got: %s, want: %s", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestPathToStrings(t *testing.T) {
	in := &gnmipb.Path{Elem: []*gnmipb.PathElem{
		{Name: "a"},
		{Name: "b"},
	}}
	want := []string{"a", "b"}
	got, err := PathToStrings(in)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PathToStrings(%v): got %q, want %q", in, got, want)
	}
}

func TestStringToPath(t *testing.T) {
	tests := []struct {
		name                string
		in                  string
		wantStringSlicePath *gnmipb.Path
		wantStructuredPath  *gnmipb.Path
		wantSliceErr        string
		wantStructuredErr   string
		wantCombinedErr     string
	}{{
		name:                "simple path",
		in:                  "/a/b/c/d",
		wantStringSlicePath: &gnmipb.Path{Element: []string{"a", "b", "c", "d"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "a"},
				{Name: "b"},
				{Name: "c"},
				{Name: "d"},
			},
		},
	}, {
		name:                "path with simple key",
		in:                  "/a/b[c=d]/e",
		wantStringSlicePath: &gnmipb.Path{Element: []string{"a", "b[c=d]", "e"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "a"},
				{Name: "b", Key: map[string]string{"c": "d"}},
				{Name: "e"},
			},
		},
	}, {
		name:                "path with multiple keys",
		in:                  "/a/b[c=d][e=f]/g",
		wantStringSlicePath: &gnmipb.Path{Element: []string{"a", "b[c=d][e=f]", "g"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "a"},
				{Name: "b", Key: map[string]string{
					"c": "d",
					"e": "f",
				}},
				{Name: "g"},
			},
		},
	}, {
		name:              "path with a key missing an equals sign",
		in:                "/a/b[cd]/e",
		wantSliceErr:      "received null key name for element b",
		wantStructuredErr: "received null key name for element b",
	}, {
		name:                "path with slashes in the key",
		in:                  `/interfaces/interface[name=Ethernet1/2/3]`,
		wantStringSlicePath: &gnmipb.Path{Element: []string{"interfaces", "interface[name=Ethernet1/2/3]"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "interface", Key: map[string]string{"name": "Ethernet1/2/3"}},
			},
		},
	}, {
		name:                "path with escaped equals in the key",
		in:                  `/interfaces/interface[name=Ethernet\=bar]`,
		wantStringSlicePath: &gnmipb.Path{Element: []string{"interfaces", `interface[name=Ethernet\=bar]`}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "interface", Key: map[string]string{"name": `Ethernet=bar`}},
			},
		},
	}, {
		name:                "open square bracket in the key",
		in:                  `/interfaces/interface[name=[foo]/state`,
		wantStringSlicePath: &gnmipb.Path{Element: []string{"interfaces", "interface[name=[foo]", "state"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "interface", Key: map[string]string{"name": "[foo"}},
				{Name: "state"},
			},
		},
	}, {
		name:                `name [name=[\\\]] example from specification`,
		in:                  `/interfaces/interface[name=[\\\]]`,
		wantStringSlicePath: &gnmipb.Path{Element: []string{"interfaces", `interface[name=[\\]]`}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "interface", Key: map[string]string{"name": `[\]`}},
			},
		},
	}, {
		name:                "forward slash in key which does not need to be escaped ",
		in:                  `/interfaces/interface[name=\/foo]/state`,
		wantStringSlicePath: &gnmipb.Path{Element: []string{"interfaces", `interface[name=/foo]`, "state"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "interface", Key: map[string]string{"name": `/foo`}},
				{Name: "state"},
			},
		},
	}, {
		name:                "escaped forward slash in an element name",
		in:                  `/interfaces/inter\/face[name=foo]`,
		wantStringSlicePath: &gnmipb.Path{Element: []string{"interfaces", "inter/face[name=foo]"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "inter/face", Key: map[string]string{"name": "foo"}},
			},
		},
	}, {
		name:                "escaped forward slash in an attribute",
		in:                  `/interfaces/interface[name=foo\/bar]`,
		wantStringSlicePath: &gnmipb.Path{Element: []string{"interfaces", "interface[name=foo/bar]"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "interface", Key: map[string]string{"name": "foo/bar"}},
			},
		},
	}, {
		name:                `single-level wildcard`,
		in:                  `/interfaces/interface/*/state`,
		wantStringSlicePath: &gnmipb.Path{Element: []string{"interfaces", "interface", "*", "state"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "interface"},
				{Name: "*"},
				{Name: "state"},
			},
		},
	}, {
		name:                "multi-level wildcard",
		in:                  `/interfaces/.../state`,
		wantStringSlicePath: &gnmipb.Path{Element: []string{"interfaces", "...", "state"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "..."},
				{Name: "state"},
			},
		},
	}, {
		name:                "path with escaped backslash in an element",
		in:                  `/foo/bar\\\/baz/hat`,
		wantStringSlicePath: &gnmipb.Path{Element: []string{"foo", `bar/baz`, "hat"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "foo"},
				{Name: `bar/baz`},
				{Name: "hat"},
			},
		},
	}, {
		name:                "path with escaped backslash in a key",
		in:                  `/foo/bar[baz\\foo=hat]`,
		wantStringSlicePath: &gnmipb.Path{Element: []string{"foo", `bar[baz\foo=hat]`}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "foo"},
				{Name: "bar", Key: map[string]string{`baz\foo`: "hat"}},
			},
		},
	}, {
		name:                "additional equals within the key, unescaped",
		in:                  `/foo/bar[baz==bat]`,
		wantStringSlicePath: &gnmipb.Path{Element: []string{"foo", "bar[baz=\\=bat]"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "foo"},
				{Name: "bar", Key: map[string]string{"baz": "=bat"}},
			},
		},
	}, {
		name:              "error - unescaped ] within a key value",
		in:                `/foo/bar[baz=]bat]`,
		wantSliceErr:      "received null value for key baz of element bar",
		wantStructuredErr: "received null value for key baz of element bar",
	}, {
		name:                "escaped ] within key value",
		in:                  `/foo/bar[baz=\]bat]`,
		wantStringSlicePath: &gnmipb.Path{Element: []string{"foo", `bar[baz=\]bat]`}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "foo"},
				{Name: "bar", Key: map[string]string{"baz": "]bat"}},
			},
		},
	}, {
		name:              "trailing garbage outside of kv name",
		in:                `/foo/bar[baz=bat]hat`,
		wantSliceErr:      "trailing garbage following keys in element bar, got: hat",
		wantStructuredErr: "trailing garbage following keys in element bar, got: hat",
	}, {
		name:                "relative path",
		in:                  `../foo/bar`,
		wantStringSlicePath: &gnmipb.Path{Element: []string{"..", "foo", "bar"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: ".."},
				{Name: "foo"},
				{Name: "bar"},
			},
		},
	}, {
		name:              "key with null value",
		in:                `/foo/bar[baz=]/hat`,
		wantSliceErr:      "received null value for key baz of element bar",
		wantStructuredErr: "received null value for key baz of element bar",
	}, {
		name:              "key with unescaped [ within key",
		in:                `/foo/bar[[bar=baz]`,
		wantSliceErr:      "received an unescaped [ in key of element bar",
		wantStructuredErr: "received an unescaped [ in key of element bar",
	}, {
		name:              "element with unescaped ]",
		in:                `/foo/bar]`,
		wantSliceErr:      "received an unescaped ] when not in a key for element bar",
		wantStructuredErr: "received an unescaped ] when not in a key for element bar",
	}, {
		name:                "empty string",
		in:                  "",
		wantStringSlicePath: &gnmipb.Path{Element: []string{}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{},
		},
	}, {
		name:                "root element",
		in:                  "/",
		wantStringSlicePath: &gnmipb.Path{Element: []string{}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{},
		},
	}, {
		name:                "trailing /",
		in:                  "/foo/bar/",
		wantStringSlicePath: &gnmipb.Path{Element: []string{"foo", "bar"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "foo"},
				{Name: "bar"},
			},
		},
	}, {
		name:              "whitespace in key",
		in:                "foo[bar =baz]",
		wantSliceErr:      "received an invalid space in element foo key name 'bar '",
		wantStructuredErr: "received an invalid space in element foo key name 'bar '",
	}, {
		name:                "whitespace in value",
		in:                  "foo[bar= baz]",
		wantStringSlicePath: &gnmipb.Path{Element: []string{"foo[bar= baz]"}},
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "foo", Key: map[string]string{"bar": " baz"}},
			},
		},
	}, {
		name:              "whitespace in element name",
		in:                "foo bar/baz",
		wantSliceErr:      "invalid space character included in element name 'foo bar'",
		wantStructuredErr: "invalid space character included in element name 'foo bar'",
	}}

	for _, tt := range tests {
		gotSlicePath, sliceErr := StringToStringSlicePath(tt.in)
		if sliceErr != nil && !strings.Contains(sliceErr.Error(), tt.wantSliceErr) {
			t.Errorf("%s: StringToStringSlicePath(%v): did not get expected error, got:\n%v\nwant:\n%v", tt.name, tt.in, sliceErr, tt.wantSliceErr)
		}

		if sliceErr == nil && !proto.Equal(gotSlicePath, tt.wantStringSlicePath) {
			t.Errorf("%s: StringToStringSlicePath(%v): did not get expected string slice path, got:\n%v\nwant:\n%v", tt.name, tt.in, gotSlicePath, tt.wantStringSlicePath)
		}

		gotStructuredPath, strErr := StringToStructuredPath(tt.in)
		if strErr != nil && !strings.Contains(strErr.Error(), tt.wantStructuredErr) {
			t.Errorf("%s: StringToStructuredPath(%v): did not get expected error, got: %v, want: %v", tt.name, tt.in, strErr, tt.wantStructuredErr)
		}

		if strErr == nil && !proto.Equal(gotStructuredPath, tt.wantStructuredPath) {
			t.Errorf("%s: StringToStructuredPath(%v): did not get expected structured path, got: %v, want: %v", tt.name, tt.in, proto.MarshalTextString(gotStructuredPath), proto.MarshalTextString(tt.wantStructuredPath))
		}

		if strErr != nil || sliceErr != nil {
			continue // If an error is expected for either of the cases, don't test the combined case.
		}

		wantCombined := proto.Clone(tt.wantStructuredPath).(*gnmipb.Path)
		wantCombined.Element = append(wantCombined.Element, tt.wantStringSlicePath.Element...)
		gotCombinedPath, combinedErr := StringToPath(tt.in, StringSlicePath, StructuredPath)
		if combinedErr != nil && combinedErr.Error() != tt.wantCombinedErr {
			t.Errorf("%s: StringToPath(%v, {StringSlicePath, StructuredPath}): did not get expected combined error, got: %v, want: %v", tt.name, tt.in, combinedErr, tt.wantCombinedErr)
		}

		if combinedErr == nil && !proto.Equal(gotCombinedPath, wantCombined) {
			t.Errorf("%s: StringToPath(%v, {StringSlicePath, StructuredPath}): did not get expected combined path message, got: %v, want: %v", tt.name, tt.in, proto.MarshalTextString(gotCombinedPath), proto.MarshalTextString(wantCombined))
		}

	}
}
