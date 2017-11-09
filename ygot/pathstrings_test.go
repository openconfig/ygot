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
	"testing"

	"github.com/kylelemons/godebug/pretty"

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
		wantErr: "nil element at index 1 in [x  y z]",
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
		name: "structured path with nil element",
		in: &gnmipb.Path{Elem: []*gnmipb.PathElem{
			{Name: "a", Key: map[string]string{"a": "b"}},
			{Key: map[string]string{"c": "d"}},
		}},
		wantErr: "nil name for PathElem at index 1",
	}, {
		name: "structed path with nil key name",
		in: &gnmipb.Path{Elem: []*gnmipb.PathElem{
			{Name: "a", Key: map[string]string{"": "d"}},
		}},
		wantErr: "nil key name (value: d) in element a",
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
		if err != nil && err.Error() != tt.wantErr {
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

func TestStringtoPath(t *testing.T) {
	tests := []struct {
		name       string
		inString   string
		inPathType []PathType
		wantPath   *gnmipb.Path
		wantErr    string
	}{{
		name:       "path populating structured path only",
		inString:   "/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/name",
		inPathType: []PathType{StructuredPath},
		wantPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "interfaces",
			}, {
				Name: "interface",
				Key:  map[string]string{"name": "eth0"},
			}, {
				Name: "subinterfaces",
			}, {
				Name: "subinterface",
				Key:  map[string]string{"index": "0"},
			}, {
				Name: "config",
			}, {
				Name: "name",
			}},
		},
	}, {
		name:       "path populating string slice path only",
		inString:   "/acl/acl-sets/acl-set[name=foo][type=IPV4]/config/name",
		inPathType: []PathType{StringSlicePath},
		wantPath: &gnmipb.Path{
			Element: []string{"acl", "acl-sets", "acl-set[name=foo][type=IPV4]", "config", "name"},
		},
	}, {
		name:       "both path types",
		inString:   "/wavelength-router/media-channels/channel[index=42]",
		inPathType: []PathType{StringSlicePath, StructuredPath},
		wantPath: &gnmipb.Path{
			Element: []string{"wavelength-router", "media-channels", "channel[index=42]"},
			Elem: []*gnmipb.PathElem{{
				Name: "wavelength-router",
			}, {
				Name: "media-channels",
			}, {
				Name: "channel",
				Key:  map[string]string{"index": "42"},
			}},
		},
	}, {
		name:     "zero path types requested",
		inString: "/acl/state/counter-capability",
		wantErr:  "no path types specified",
	}, {
		name:       "bad path - element",
		inString:   "/foo/bar[42]",
		inPathType: []PathType{StringSlicePath},
		wantErr:    "error building string slice path: key value bar[42 does not contain a key=value pair",
	}, {
		name:       "bad path - structured",
		inString:   "/foo/bar[42]",
		inPathType: []PathType{StructuredPath},
		wantErr:    "error building structured path: received a key with no equals sign in it, key name: , key value: 42",
	}}

	for _, tt := range tests {
		got, err := StringToPath(tt.inString, tt.inPathType...)
		if err != nil && err.Error() != tt.wantErr {
			t.Errorf("%s: StringToPath(%v, %v): did not get expected error, got: %v, want: %v", tt.name, tt.inString, tt.inPathType, err, tt.wantErr)
		}

		if err != nil || tt.wantErr != "" {
			continue
		}

		if diff := pretty.Compare(got, tt.wantPath); diff != "" {
			t.Errorf("%s: StringToPath(%v, %v): did not get expected path, diff(-got,+want):\n%s", tt.name, tt.inString, tt.inPathType, diff)
		}
	}
}

func TestStringToStringSlicePath(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    *gnmipb.Path
		wantErr string
	}{{
		name: `simple path`,
		in:   `/a/b/c/d`,
		want: &gnmipb.Path{Element: []string{`a`, `b`, `c`, `d`}},
	}, {
		name: `slashes in key`,
		in:   `/interfaces/interface[name=Ethernet1/2/3]/state`,
		want: &gnmipb.Path{Element: []string{`interfaces`, `interface[name=Ethernet1/2/3]`, `state`}},
	}, {
		name: `open square bracket in key`,
		in:   `/interfaces/interface[name=[foo]/state`,
		want: &gnmipb.Path{Element: []string{`interfaces`, `interface[name=[foo]`, `state`}},
	}, {
		name: `escaped forward slash in key`,
		in:   `/element/list[key=\/foo]/bar`,
		want: &gnmipb.Path{Element: []string{`element`, `list[key=/foo]`, `bar`}},
	}, {
		name:    `invalid key`,
		in:      `/element/list[keynoequals]`,
		wantErr: "key value list[keynoequals does not contain a key=value pair",
	}, {
		name: `multiple keys`,
		in:   `/network-instances/network-instance/tables/table[protocol=BGP][address-family=*]`,
		want: &gnmipb.Path{Element: []string{`network-instances`, `network-instance`, `tables`, `table[protocol=BGP][address-family=*]`}},
	}, {
		name: `single-level wildcard`,
		in:   `../*/*/*/state`,
		want: &gnmipb.Path{Element: []string{`..`, `*`, `*`, `*`, `state`}},
	}, {
		name: `multi-level wildcard`,
		in:   `//config`,
		want: &gnmipb.Path{Element: []string{``, `config`}},
	}}

	for _, tt := range tests {
		got, err := StringToStringSlicePath(tt.in)
		if err != nil && err.Error() != tt.wantErr {
			t.Errorf("%s: StringToPath(%s): got err: %v, want: %v", tt.name, tt.in, err, tt.wantErr)
		}

		if tt.wantErr != "" || err != nil {
			continue
		}

		if diff := pretty.Compare(tt.want, got); diff != "" {
			t.Errorf("%s: StringtoPath(%s): diff (-want, +got):\n%v", tt.name, tt.in, diff)
		}
	}
}

func TestStringToStructuredPath(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    *gnmipb.Path
		wantErr string
	}{{
		name: "simple path",
		in:   `/a/b/c/d`,
		want: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "a"},
				{Name: "b"},
				{Name: "c"},
				{Name: "d"},
			},
		},
	}, {
		name: "path with key",
		in:   `/a[b=c]/d`,
		want: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key: map[string]string{
					"b": "c",
				},
			}, {
				Name: "d",
			}},
		},
	}, {
		name: "path with multiple keys",
		in:   `/a[b=c][d=e]/f`,
		want: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key: map[string]string{
					"b": "c",
					"d": "e",
				},
			}, {
				Name: "f",
			}},
		},
	}, {
		name:    "path with a key with no equals",
		in:      `/a[badkey]/c`,
		wantErr: "received a key with no equals sign in it, key name: , key value: badkey",
	}, {
		name: "slashes in key",
		in:   `/interfaces/interface[name=Ethernet1/2/3]`,
		want: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "interfaces",
			}, {
				Name: "interface",
				Key: map[string]string{
					"name": "Ethernet1/2/3",
				},
			}},
		},
	}, {
		name: "escaped equals in key",
		in:   `/is/i[name=one\=two]`,
		want: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "is",
			}, {
				Name: "i",
				Key: map[string]string{
					"name": `one=two`,
				},
			}},
		},
	}, {
		name:    "invalid length key name",
		in:      `/a[=c]`,
		wantErr: "received a key with no equals sign in it, key name: , key value: c",
	}}

	for _, tt := range tests {
		got, err := StringToStructuredPath(tt.in)
		if (err != nil) && err.Error() != tt.wantErr {
			t.Errorf("%s: StringToPathElemPath(%v): did not get expected error status, got: %v, wantErr: %v", tt.name, tt.in, err, tt.wantErr)
		}

		if tt.wantErr != "" || err != nil {
			continue
		}

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("%s: StringToPathElemPath(%v): did not get expected return value, diff(-got,+want):\n%v", tt.name, tt.in, diff)
		}
	}
}
