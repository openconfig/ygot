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

package util

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

// XXX: This is copied code from ygot package. ygot's code should probably
// live in this package instead.
func TestStringToPath(t *testing.T) {
	tests := []struct {
		name               string
		in                 string
		wantStructuredPath *gnmipb.Path
		wantStructuredErr  string
	}{{
		name: "simple path",
		in:   "/a/b/c/d",
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "a"},
				{Name: "b"},
				{Name: "c"},
				{Name: "d"},
			},
		},
	}, {
		name: "path with simple key",
		in:   "/a/b[c=d]/e",
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "a"},
				{Name: "b", Key: map[string]string{"c": "d"}},
				{Name: "e"},
			},
		},
	}, {
		name: "path with multiple keys",
		in:   "/a/b[c=d][e=f]/g",
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
		wantStructuredErr: "received null key name for element b",
	}, {
		name: "path with slashes in the key",
		in:   `/interfaces/interface[name=Ethernet1/2/3]`,
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "interface", Key: map[string]string{"name": "Ethernet1/2/3"}},
			},
		},
	}, {
		name: "path with escaped equals in the key",
		in:   `/interfaces/interface[name=Ethernet\=bar]`,
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "interface", Key: map[string]string{"name": `Ethernet=bar`}},
			},
		},
	}, {
		name: "open square bracket in the key",
		in:   `/interfaces/interface[name=[foo]/state`,
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "interface", Key: map[string]string{"name": "[foo"}},
				{Name: "state"},
			},
		},
	}, {
		name: `name [name=[\\\]] example from specification`,
		in:   `/interfaces/interface[name=[\\\]]`,
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "interface", Key: map[string]string{"name": `[\]`}},
			},
		},
	}, {
		name: "forward slash in key which does not need to be escaped ",
		in:   `/interfaces/interface[name=\/foo]/state`,
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "interface", Key: map[string]string{"name": `/foo`}},
				{Name: "state"},
			},
		},
	}, {
		name: "escaped forward slash in an element name",
		in:   `/interfaces/inter\/face[name=foo]`,
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "inter/face", Key: map[string]string{"name": "foo"}},
			},
		},
	}, {
		name: "escaped forward slash in an attribute",
		in:   `/interfaces/interface[name=foo\/bar]`,
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "interface", Key: map[string]string{"name": "foo/bar"}},
			},
		},
	}, {
		name: `single-level wildcard`,
		in:   `/interfaces/interface/*/state`,
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "interface"},
				{Name: "*"},
				{Name: "state"},
			},
		},
	}, {
		name: "multi-level wildcard",
		in:   `/interfaces/.../state`,
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "interfaces"},
				{Name: "..."},
				{Name: "state"},
			},
		},
	}, {
		name: "path with escaped backslash in an element",
		in:   `/foo/bar\\\/baz/hat`,
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "foo"},
				{Name: `bar/baz`},
				{Name: "hat"},
			},
		},
	}, {
		name: "path with escaped backslash in a key",
		in:   `/foo/bar[baz\\foo=hat]`,
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "foo"},
				{Name: "bar", Key: map[string]string{`baz\foo`: "hat"}},
			},
		},
	}, {
		name: "additional equals within the key, unescaped",
		in:   `/foo/bar[baz==bat]`,
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "foo"},
				{Name: "bar", Key: map[string]string{"baz": "=bat"}},
			},
		},
	}, {
		name:              "error - unescaped ] within a key value",
		in:                `/foo/bar[baz=]bat]`,
		wantStructuredErr: "received null value for key baz of element bar",
	}, {
		name: "escaped ] within key value",
		in:   `/foo/bar[baz=\]bat]`,
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "foo"},
				{Name: "bar", Key: map[string]string{"baz": "]bat"}},
			},
		},
	}, {
		name:              "trailing garbage outside of kv name",
		in:                `/foo/bar[baz=bat]hat`,
		wantStructuredErr: "trailing garbage following keys in element bar, got: hat",
	}, {
		name: "relative path",
		in:   `../foo/bar`,
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
		wantStructuredErr: "received null value for key baz of element bar",
	}, {
		name:              "key with unescaped [ within key",
		in:                `/foo/bar[[bar=baz]`,
		wantStructuredErr: "received an unescaped [ in key of element bar",
	}, {
		name:              "element with unescaped ]",
		in:                `/foo/bar]`,
		wantStructuredErr: "received an unescaped ] when not in a key for element bar",
	}, {
		name: "empty string",
		in:   "",
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{},
		},
	}, {
		name: "root element",
		in:   "/",
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{},
		},
	}, {
		name: "trailing /",
		in:   "/foo/bar/",
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "foo"},
				{Name: "bar"},
			},
		},
	}, {
		name:              "whitespace in key",
		in:                "foo[bar =baz]",
		wantStructuredErr: "received an invalid space in element foo key name 'bar '",
	}, {
		name: "whitespace in value",
		in:   "foo[bar= baz]",
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{
				{Name: "foo", Key: map[string]string{"bar": " baz"}},
			},
		},
	}, {
		name:              "whitespace in element name",
		in:                "foo bar/baz",
		wantStructuredErr: "invalid space character included in element name 'foo bar'",
	}, {
		name: "bgp example",
		in:   "neighbors/neighbor[neighbor-address=192.0.2.1]/config/neighbor-address",
		wantStructuredPath: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "neighbors",
			}, {
				Name: "neighbor",
				Key:  map[string]string{"neighbor-address": "192.0.2.1"},
			}, {
				Name: "config",
			}, {
				Name: "neighbor-address",
			}},
		},
	}}

	for _, tt := range tests {
		gotStructuredPath, strErr := stringToStructuredPath(tt.in)
		if strErr != nil && !strings.Contains(strErr.Error(), tt.wantStructuredErr) {
			t.Errorf("%s: stringToStructuredPath(%v): did not get expected error, got: %v, want: %v", tt.name, tt.in, strErr, tt.wantStructuredErr)
		}

		if strErr == nil && !proto.Equal(gotStructuredPath, tt.wantStructuredPath) {
			t.Errorf("%s: stringToStructuredPath(%v): did not get expected structured path, got: %v, want: %v", tt.name, tt.in, prototext.Format(gotStructuredPath), prototext.Format(tt.wantStructuredPath))
		}
	}
}
