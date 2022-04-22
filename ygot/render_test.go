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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/ygot/testutil"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

var (
	base64testString        = "forty two"
	base64testStringEncoded = base64.StdEncoding.EncodeToString([]byte(base64testString))
	testBinary              = testutil.Binary(base64testString)
)

func TestPathElemBasics(t *testing.T) {
	tests := []struct {
		name               string
		inGNMIPath         *gnmiPath
		wantValid          bool
		wantIsStringPath   bool
		wantIsPathElemPath bool
		wantLen            int
	}{{
		name: "string path only",
		inGNMIPath: &gnmiPath{
			stringSlicePath: []string{"foo", "bar"},
		},
		wantValid:          true,
		wantIsStringPath:   true,
		wantIsPathElemPath: false,
		wantLen:            2,
	}, {
		name: "path elem path only",
		inGNMIPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{{
				Name: "foo",
			}},
		},
		wantValid:          true,
		wantIsStringPath:   false,
		wantIsPathElemPath: true,
		wantLen:            1,
	}, {
		name:       "invalid, both nil",
		inGNMIPath: &gnmiPath{},
		wantValid:  false,
	}, {
		name: "invalid, both set",
		inGNMIPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{{
				Name: "bar",
			}},
			stringSlicePath: []string{"foo"},
		},
		wantValid: false,
	}}

	for _, tt := range tests {
		if got := tt.inGNMIPath.isValid(); got != tt.wantValid {
			t.Errorf("%s: (gnmiPath)(%#v).isValid(): did not get expected result, got: %v, want: %v", tt.name, tt.inGNMIPath, got, tt.wantValid)
		}

		if !tt.inGNMIPath.isValid() {
			continue
		}

		if got := tt.inGNMIPath.isStringSlicePath(); got != tt.wantIsStringPath {
			t.Errorf("%s: (gnmiPath)(%#v).isStringSlicePath(): did not get expeted result, got: %v, want: %v", tt.name, tt.inGNMIPath, got, tt.wantIsStringPath)
		}

		if got := tt.inGNMIPath.isPathElemPath(); got != tt.wantIsPathElemPath {
			t.Errorf("%s: (gnmiPath)(%#v).isPathElemPath(): did not get expected result, got: %v, want: %v", tt.name, tt.inGNMIPath, got, tt.wantIsPathElemPath)
		}

		if got := tt.inGNMIPath.Len(); got != tt.wantLen {
			t.Errorf("%s: (gnmiPath)(%#v).Len(): did not get expected result, got: %v, want: %v", tt.name, tt.inGNMIPath, got, tt.wantLen)
		}
	}
}

func TestAppendName(t *testing.T) {
	tests := []struct {
		name    string
		inPath  *gnmiPath
		inName  string
		want    *gnmiPath
		wantErr bool
	}{{
		name:   "string slice append",
		inPath: &gnmiPath{stringSlicePath: []string{}},
		inName: "foo",
		want:   &gnmiPath{stringSlicePath: []string{"foo"}},
	}, {
		name:   "pathElem slice append",
		inPath: &gnmiPath{pathElemPath: []*gnmipb.PathElem{}},
		inName: "bar",
		want:   &gnmiPath{pathElemPath: []*gnmipb.PathElem{{Name: "bar"}}},
	}, {
		name:    "invalid input",
		inPath:  &gnmiPath{},
		inName:  "foo",
		wantErr: true,
	}, {
		name:   "existing string slice",
		inPath: &gnmiPath{stringSlicePath: []string{"bar"}},
		inName: "foo",
		want:   &gnmiPath{stringSlicePath: []string{"bar", "foo"}},
	}, {
		name: "existing pathElem",
		inPath: &gnmiPath{pathElemPath: []*gnmipb.PathElem{{
			Name: "zaphod",
			Key:  map[string]string{"just": "this-guy"},
		}}},
		inName: "beeblebrox",
		want: &gnmiPath{pathElemPath: []*gnmipb.PathElem{{
			Name: "zaphod",
			Key:  map[string]string{"just": "this-guy"},
		}, {
			Name: "beeblebrox",
		}}},
	}}

	for _, tt := range tests {
		if err := tt.inPath.AppendName(tt.inName); (err != nil) != tt.wantErr {
			t.Errorf("%s: (gnmiPath)(%#v).AppendName(%s): did not get expected error status, got: %v, want error: %v", tt.name, tt.inPath, tt.inName, err, tt.wantErr)
		}

		if tt.wantErr {
			continue
		}

		if diff := cmp.Diff(tt.want, tt.inPath, cmp.AllowUnexported(gnmiPath{}), cmp.Comparer(proto.Equal)); diff != "" {
			t.Errorf("%s: (gnmiPath)(%#v).AppendName(%s): did not get expected path, diff(-want,+got):\n%s", tt.name, tt.inPath, tt.inName, diff)
		}
	}
}

func TestGNMIPathCopy(t *testing.T) {
	tests := []struct {
		name   string
		inPath *gnmiPath
	}{{
		name:   "string element path",
		inPath: &gnmiPath{stringSlicePath: []string{"one", "two"}},
	}, {
		name: "path element path",
		inPath: &gnmiPath{pathElemPath: []*gnmipb.PathElem{
			{Name: "one"},
			{Name: "two", Key: map[string]string{"three": "four"}},
		}},
	}}

	for _, tt := range tests {
		if got := tt.inPath.Copy(); !cmp.Equal(got, tt.inPath, cmp.AllowUnexported(gnmiPath{}), protocmp.Transform()) {
			t.Errorf("%s: (gnmiPath).Copy(): did not get expected result, got: %v, want: %v", tt.name, got, tt.inPath)
		}
	}
}

func TestGNMIPathOps(t *testing.T) {
	tests := []struct {
		name                string
		inPath              *gnmiPath
		inIndex             int
		inValue             interface{}
		wantLastPathElem    *gnmipb.PathElem
		wantLastPathElemErr bool
		wantPath            *gnmiPath
		wantSetIndexErr     bool
	}{{
		name:                "string slice path",
		inPath:              newStringSliceGNMIPath([]string{"one", "two"}),
		wantLastPathElemErr: true,
		inIndex:             1,
		inValue:             "three",
		wantPath:            newStringSliceGNMIPath([]string{"one", "three"}),
	}, {
		name:             "pathElem path",
		inPath:           newPathElemGNMIPath([]*gnmipb.PathElem{{Name: "foo"}, {Name: "bar"}}),
		inIndex:          0,
		inValue:          &gnmipb.PathElem{Name: "baz", Key: map[string]string{"formerly": "foo"}},
		wantLastPathElem: &gnmipb.PathElem{Name: "bar"},
		wantPath:         &gnmiPath{pathElemPath: []*gnmipb.PathElem{{Name: "baz", Key: map[string]string{"formerly": "foo"}}, {Name: "bar"}}},
	}, {
		name:                "invalid set index - path elem into string",
		inPath:              newStringSliceGNMIPath([]string{"one", "two"}),
		inIndex:             1,
		inValue:             &gnmipb.PathElem{Name: "bar"},
		wantLastPathElemErr: true,
		wantSetIndexErr:     true,
	}, {
		name:             "invalid set index - string into path elem",
		inPath:           newPathElemGNMIPath([]*gnmipb.PathElem{{Name: "one"}}),
		inIndex:          0,
		inValue:          "foo",
		wantLastPathElem: &gnmipb.PathElem{Name: "one"},
		wantSetIndexErr:  true,
	}, {
		name:                "invalid set index - no known type",
		inPath:              newStringSliceGNMIPath([]string{"foo"}),
		inIndex:             0,
		inValue:             32,
		wantLastPathElemErr: true,
		wantSetIndexErr:     true,
	}, {
		name:                "invalid set index - index out of range",
		inPath:              newStringSliceGNMIPath([]string{"bar"}),
		inIndex:             422,
		inValue:             "hello buffer overflow!",
		wantLastPathElemErr: true,
		wantSetIndexErr:     true,
	}}

	for _, tt := range tests {
		gotLast, err := tt.inPath.LastPathElem()
		if (err != nil) != tt.wantLastPathElemErr {
			t.Errorf("%s: %v.LastPathElem(): did not get expected error, got: %v, wantErr: %v", tt.name, tt.inPath, err, tt.wantLastPathElemErr)
		}

		if err == nil && !proto.Equal(gotLast, tt.wantLastPathElem) {
			t.Errorf("%s: %v.LastPathElem(), did not get expected last element, got: %v, want: %v", tt.name, tt.inPath, gotLast, tt.wantLastPathElem)
		}

		np := tt.inPath.Copy()
		err = np.SetIndex(tt.inIndex, tt.inValue)
		if (err != nil) != tt.wantSetIndexErr {
			t.Errorf("%s: %v.SetIndex(%d, %v): did not get expected error, got: %v, wantErr: %v", tt.name, tt.inPath, tt.inIndex, tt.inValue, err, tt.wantSetIndexErr)
		}

		if err == nil && !cmp.Equal(np, tt.wantPath, cmp.AllowUnexported(gnmiPath{}), cmp.Comparer(proto.Equal)) {
			t.Errorf("%s: %v.SetIndex(%d, %v): did not get expected path, got: %v, want: %v", tt.name, tt.inPath, tt.inIndex, tt.inValue, np, tt.wantPath)
		}
	}
}

func TestGNMIPathToProto(t *testing.T) {
	tests := []struct {
		name      string
		inPath    *gnmiPath
		wantProto *gnmipb.Path
		wantErr   bool
	}{{
		name:      "string slice path",
		inPath:    newStringSliceGNMIPath([]string{"one", "two"}),
		wantProto: &gnmipb.Path{Element: []string{"one", "two"}},
	}, {
		name:      "empty string slice path",
		inPath:    newStringSliceGNMIPath([]string{}),
		wantProto: nil,
	}, {
		name:      "path elem path",
		inPath:    newPathElemGNMIPath([]*gnmipb.PathElem{{Name: "one"}}),
		wantProto: &gnmipb.Path{Elem: []*gnmipb.PathElem{{Name: "one"}}},
	}, {
		name:      "empty path elem path",
		inPath:    newPathElemGNMIPath([]*gnmipb.PathElem{}),
		wantProto: nil,
	}, {
		name:    "invalid path",
		inPath:  &gnmiPath{stringSlicePath: []string{"one"}, pathElemPath: []*gnmipb.PathElem{{Name: "bar"}}},
		wantErr: true,
	}}

	for _, tt := range tests {
		got, err := tt.inPath.ToProto()

		if (err != nil) != tt.wantErr {
			t.Errorf("%s: %v.ToProto(), did not get expected error, got: %v, wantErr: %v", tt.name, tt.inPath, err, tt.wantErr)
		}

		if !proto.Equal(got, tt.wantProto) {
			t.Errorf("%s: %v.ToProto, did not get expected return value, got: %s, want: %s", tt.name, tt.inPath, prototext.Format(got), prototext.Format(tt.wantProto))
		}
	}
}

func TestStripPrefix(t *testing.T) {
	tests := []struct {
		name     string
		inPath   *gnmiPath
		inPrefix *gnmiPath
		want     *gnmiPath
		wantErr  bool
	}{{
		name: "mismatched types",
		inPath: &gnmiPath{
			stringSlicePath: []string{},
		},
		inPrefix: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{},
		},
		wantErr: true,
	}, {
		name: "simple element prefix",
		inPath: &gnmiPath{
			stringSlicePath: []string{"one", "two", "three"},
		},
		inPrefix: &gnmiPath{
			stringSlicePath: []string{"one"},
		},
		want: &gnmiPath{
			stringSlicePath: []string{"two", "three"},
		},
	}, {
		name: "two element prefix",
		inPath: &gnmiPath{
			stringSlicePath: []string{"one", "two", "three"},
		},
		inPrefix: &gnmiPath{
			stringSlicePath: []string{"one", "two"},
		},
		want: &gnmiPath{
			stringSlicePath: []string{"three"},
		},
	}, {
		name: "invalid prefix",
		inPath: &gnmiPath{
			stringSlicePath: []string{"four", "five", "six"},
		},
		inPrefix: &gnmiPath{
			stringSlicePath: []string{"one", "two"},
		},
		wantErr: true,
	}, {
		name: "simple pathelem prefix",
		inPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{{
				Name: "one",
			}, {
				Name: "two",
			}, {
				Name: "three",
			}},
		},
		inPrefix: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{{
				Name: "one",
			}},
		},
		want: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{{
				Name: "two",
			}, {
				Name: "three",
			}},
		},
	}, {
		name: "two element pathelem prefix",
		inPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{{
				Name: "one",
			}, {
				Name: "two",
			}, {
				Name: "three",
			}},
		},
		inPrefix: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{{
				Name: "one",
			}, {
				Name: "two",
			}},
		},
		want: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{{
				Name: "three",
			}},
		},
	}, {
		name: "pathelem with a key",
		inPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{{
				Name: "one",
				Key:  map[string]string{"key": "value"},
			}, {
				Name: "two",
			}},
		},
		inPrefix: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{{
				Name: "one",
				Key:  map[string]string{"key": "value"},
			}},
		},
		want: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{{
				Name: "two",
			}},
		},
	}, {
		name: "invalid prefix for pathelem path",
		inPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{{
				Name: "one",
			}, {
				Name: "two",
			}},
		},
		inPrefix: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{{
				Name: "four",
			}},
		},
		wantErr: true,
	}, {
		name: "invalid inputs",
		inPath: &gnmiPath{
			pathElemPath:    []*gnmipb.PathElem{},
			stringSlicePath: []string{},
		},
		inPrefix: &gnmiPath{stringSlicePath: []string{"foo"}},
		wantErr:  true,
	}}

	for _, tt := range tests {
		got, err := tt.inPath.StripPrefix(tt.inPrefix)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: stripPrefix(%v, %v): got unexpected error: %v", tt.name, tt.inPath, tt.inPrefix, err)
			}
			continue
		}

		if !cmp.Equal(got, tt.want, cmp.AllowUnexported(gnmiPath{}), cmp.Comparer(proto.Equal)) {
			t.Errorf("%s: stripPrefix(%v, %v): did not get expected path, got: %v, want: %v", tt.name, tt.inPath, tt.inPrefix, got, tt.want)
		}
	}
}

type pathElemMultiKey struct {
	I *int8              `path:"i"`
	J *uint8             `path:"j"`
	S *string            `path:"s"`
	E EnumTest           `path:"e"`
	X renderExampleUnion `path:"x"`
	Y exampleUnion       `path:"y"`
}

func (*pathElemMultiKey) IsYANGGoStruct()                         {}
func (*pathElemMultiKey) Validate(...ValidationOption) error      { return nil }
func (*pathElemMultiKey) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*pathElemMultiKey) ΛBelongingModule() string                { return "" }

func (e *pathElemMultiKey) ΛListKeyMap() (map[string]interface{}, error) {
	if e.I == nil || e.J == nil || e.S == nil || e.E == (EnumTest)(0) || e.X == nil || e.Y == nil {
		return nil, fmt.Errorf("unset keys")
	}
	return map[string]interface{}{
		"i": *e.I,
		"j": *e.J,
		"s": *e.S,
		"e": e.E,
		"x": e.X,
		"y": e.Y,
	}, nil
}

func TestAppendGNMIPathElemKey(t *testing.T) {
	c := complex(30, 4)

	tests := []struct {
		name     string
		inValue  reflect.Value
		inPath   *gnmiPath
		wantPath *gnmiPath
		wantErr  bool
	}{{
		name: "invalid path",
		inValue: reflect.ValueOf(&pathElemExampleChild{
			Val: String("foo"),
		}),
		inPath:  &gnmiPath{},
		wantErr: true,
	}, {
		name: "invalid path - both specified",
		inValue: reflect.ValueOf(&pathElemExampleChild{
			Val: String("bar"),
		}),
		inPath: &gnmiPath{
			stringSlicePath: []string{"fish"},
			pathElemPath:    []*gnmipb.PathElem{{Name: "bar"}},
		},
		wantErr: true,
	}, {
		name: "zero length input path",
		inValue: reflect.ValueOf(&pathElemExampleChild{
			Val: String("bar"),
		}),
		inPath:  &gnmiPath{pathElemPath: []*gnmipb.PathElem{}},
		wantErr: true,
	}, {
		name:    "invalid struct input",
		inValue: reflect.ValueOf(&struct{ Fish string }{"haddock"}),
		inPath:  &gnmiPath{pathElemPath: []*gnmipb.PathElem{{Name: "bar"}}},
		wantErr: true,
	}, {
		name:    "unserialisable input",
		inValue: reflect.ValueOf(&pathElemUnserialisable{&c}),
		inPath:  &gnmiPath{pathElemPath: []*gnmipb.PathElem{{Name: "bar"}}},
		wantErr: true,
	}, {
		name: "simple append",
		inValue: reflect.ValueOf(&pathElemExampleChild{
			Val: String("foo"),
		}),
		inPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{
				{Name: "foo"},
				{Name: "bar"},
			},
		},
		wantPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{
				{Name: "foo"},
				{Name: "bar", Key: map[string]string{"val": "foo"}},
			},
		},
	}, {
		name: "append with multiple value key, diverse values",
		inValue: reflect.ValueOf(&pathElemMultiKey{
			I: Int8(-42),
			J: Uint8(42),
			S: String("foo"),
			E: EnumTestVALTWO,
			X: &renderExampleUnionString{"hello"},
			Y: testutil.UnionFloat64(3.14),
		}),
		inPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{
				{Name: "foo"},
			},
		},
		wantPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{
				{
					Name: "foo",
					Key: map[string]string{
						"i": "-42",
						"j": "42",
						"s": "foo",
						"e": "VAL_TWO",
						"x": "hello",
						"y": "3.14",
					},
				},
			},
		},
	}, {
		name: "append with multiple value key, diverse values -- binary union value",
		inValue: reflect.ValueOf(&pathElemMultiKey{
			I: Int8(-42),
			J: Uint8(42),
			S: String("foo"),
			E: EnumTestVALTWO,
			X: &renderExampleUnionString{"hello"},
			Y: testBinary,
		}),
		inPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{
				{Name: "foo"},
			},
		},
		wantPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{
				{
					Name: "foo",
					Key: map[string]string{
						"i": "-42",
						"j": "42",
						"s": "foo",
						"e": "VAL_TWO",
						"x": "hello",
						"y": base64testStringEncoded,
					},
				},
			},
		},
	}, {
		name: "append with multiple value key, diverse values -- enum union value",
		inValue: reflect.ValueOf(&pathElemMultiKey{
			I: Int8(-42),
			J: Uint8(42),
			S: String("foo"),
			E: EnumTestVALTWO,
			X: &renderExampleUnionString{"hello"},
			Y: EnumTestVALTWO,
		}),
		inPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{
				{Name: "foo"},
			},
		},
		wantPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{
				{
					Name: "foo",
					Key: map[string]string{
						"i": "-42",
						"j": "42",
						"s": "foo",
						"e": "VAL_TWO",
						"x": "hello",
						"y": "VAL_TWO",
					},
				},
			},
		},
	}, {
		name: "append with multiple value key, invalid enum value",
		inValue: reflect.ValueOf(&pathElemMultiKey{
			I: Int8(-42),
			J: Uint8(42),
			S: String("foo"),
			E: EnumTestVALTHREE,
			X: &renderExampleUnionString{"hello"},
			Y: testutil.UnionInt64(314),
		}),
		inPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{
				{Name: "foo"},
			},
		},
		wantErr: true,
	}, {
		name: "append with multiple value key, invalid union value",
		inValue: reflect.ValueOf(&pathElemMultiKey{
			I: Int8(-42),
			J: Uint8(42),
			S: String("foo"),
			E: EnumTestVALTWO,
			X: &renderExampleUnionInvalid{String: "test"},
			Y: EnumTestVALTWO,
		}),
		inPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{
				{Name: "foo"},
			},
		},
		wantErr: true,
	}, {
		name:    "append with nil key",
		inValue: reflect.ValueOf(&pathElemMultiKey{}),
		inPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{
				{Name: "foo"},
			},
		},
		wantErr: true,
	}, {
		name: "append with path that does not have a pathelem in it",
		inValue: reflect.ValueOf(&pathElemExampleChild{
			Val: String("foo"),
		}),
		inPath: &gnmiPath{
			stringSlicePath: []string{},
		},
		wantErr: true,
	}, {
		name:    "nil input",
		inValue: reflect.ValueOf(nil),
		inPath: &gnmiPath{
			pathElemPath: []*gnmipb.PathElem{
				{Name: "foo"},
			},
		},
		wantErr: true,
	}, {
		name: "nil path input",
		inValue: reflect.ValueOf(&pathElemExampleChild{
			Val: String("bar"),
		}),
		inPath:  nil,
		wantErr: true,
	}}

	for _, tt := range tests {
		got, err := appendgNMIPathElemKey(tt.inValue, tt.inPath)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: appendgNMIPathElemKey(%v, %v): did not get expected error status, got: %v, want error: %v", tt.name, tt.inValue, tt.inPath, err, tt.wantErr)
		}

		if diff := cmp.Diff(tt.wantPath, got, cmp.AllowUnexported(gnmiPath{}), cmp.Comparer(proto.Equal)); diff != "" {
			t.Errorf("%s: appendgNMIPathElemKey(%v, %v): did not get expected return path, diff(-want,+got):\n%s", tt.name, tt.inValue, tt.inPath, diff)
		}
	}
}

func TestSliceToScalarArray(t *testing.T) {
	tests := []struct {
		name    string
		in      []interface{}
		want    *gnmipb.ScalarArray
		wantErr bool
	}{{
		name: "simple scalar array with only strings",
		in:   []interface{}{"forty", "two"},
		want: &gnmipb.ScalarArray{
			Element: []*gnmipb.TypedValue{
				{Value: &gnmipb.TypedValue_StringVal{"forty"}},
				{Value: &gnmipb.TypedValue_StringVal{"two"}},
			},
		},
	}, {
		name: "mixed scalar array with strings and integers",
		in:   []interface{}{uint8(42), uint16(1642), uint32(3242), "towel"},
		want: &gnmipb.ScalarArray{
			Element: []*gnmipb.TypedValue{
				{Value: &gnmipb.TypedValue_UintVal{42}},
				{Value: &gnmipb.TypedValue_UintVal{1642}},
				{Value: &gnmipb.TypedValue_UintVal{3242}},
				{Value: &gnmipb.TypedValue_StringVal{"towel"}},
			},
		},
	}, {
		name:    "scalar array with an unmappable type",
		in:      []interface{}{uint8(1), struct{ val string }{"hello"}},
		wantErr: true,
	}}

	for _, tt := range tests {
		got, err := sliceToScalarArray(tt.in)

		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: sliceToScalarArray(%v): got unexpected error: %v", tt.name, tt.in, err)
			}
			continue
		}

		if !proto.Equal(got, tt.want) {
			t.Errorf("%s: sliceToScalarArray(%v): did not get expected protobuf, got: %v, want: %v", tt.name, tt.in, got, tt.want)
		}
	}
}

// Binary is the name used for binary encoding in the Go structures.
type Binary []byte

// YANGEmpty is the name used for a leaf of type empty in the Go structures.
type YANGEmpty bool

// renderExample is used within TestTogNMINotifications as a GoStruct.
type renderExample struct {
	Str                 *string                             `path:"str" shadow-path:"srt"`
	IntVal              *int32                              `path:"int-val"`
	Int64Val            *int64                              `path:"int64-val"`
	FloatVal            *float32                            `path:"floatval"`
	EnumField           EnumTest                            `path:"enum"`
	Ch                  *renderExampleChild                 `path:"ch"`
	LeafList            []string                            `path:"leaf-list"`
	MixedList           []interface{}                       `path:"mixed-list"`
	List                map[uint32]*renderExampleList       `path:"list"`
	EnumList            map[EnumTest]*renderExampleEnumList `path:"enum-list"`
	UnionVal            renderExampleUnion                  `path:"union-val"`
	UnionLeafList       []renderExampleUnion                `path:"union-list"`
	UnionValSimple      exampleUnion                        `path:"union-val-simple"`
	UnionLeafListSimple []exampleUnion                      `path:"union-list-simple"`
	Binary              Binary                              `path:"binary"`
	KeylessList         []*renderExampleList                `path:"keyless-list"`
	InvalidMap          map[string]*invalidGoStruct         `path:"invalid-gostruct-map"`
	InvalidPtr          *invalidGoStruct                    `path:"invalid-gostruct"`
	Empty               YANGEmpty                           `path:"empty"`
	EnumLeafList        []EnumTest                          `path:"enum-leaflist"`
}

// IsYANGGoStruct ensures that the renderExample type implements the ValidatedGoStruct
// interface.
func (*renderExample) IsYANGGoStruct()                         {}
func (*renderExample) Validate(...ValidationOption) error      { return nil }
func (*renderExample) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*renderExample) ΛBelongingModule() string                { return "" }

// exampleUnion is an interface that is used to represent a mixed type
// union.
type exampleUnion interface {
	IsExampleUnion()
}

// renderExampleUnion is an interface that is used to represent a mixed type
// union.
type renderExampleUnion interface {
	IsRenderUnionExample()
}

type renderExampleUnionString struct {
	String string
}

func (*renderExampleUnionString) IsRenderUnionExample() {}

type renderExampleUnionInt64 struct {
	Int64 int64
}

func (*renderExampleUnionInt64) IsRenderUnionExample() {}

type renderExampleUnionBinary struct {
	Binary Binary
}

func (*renderExampleUnionBinary) IsRenderUnionExample() {}

// renderExampleUnionInvalid is an invalid union struct.
type renderExampleUnionInvalid struct {
	String string
	Int8   int8
}

func (*renderExampleUnionInvalid) IsRenderUnionExample() {}

type renderExampleUnionEnum struct {
	Enum EnumTest
}

func (*renderExampleUnionEnum) IsRenderUnionExample() {}

// renderExampleChild is a child of the renderExample struct.
type renderExampleChild struct {
	Val   *uint64   `path:"val"`
	Enum  EnumTest  `path:"enum"`
	Empty YANGEmpty `path:"empty"`
}

// IsYANGGoStruct implements the ValidatedGoStruct interface.
func (*renderExampleChild) IsYANGGoStruct()                         {}
func (*renderExampleChild) Validate(...ValidationOption) error      { return nil }
func (*renderExampleChild) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*renderExampleChild) ΛBelongingModule() string                { return "" }

// renderExampleList is a list entry in the renderExample struct.
type renderExampleList struct {
	Val *string `path:"val|state/val"`
}

// IsYANGGoStruct implements the ValidatedGoStruct interface.
func (*renderExampleList) IsYANGGoStruct()                         {}
func (*renderExampleList) Validate(...ValidationOption) error      { return nil }
func (*renderExampleList) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*renderExampleList) ΛBelongingModule() string                { return "" }

func (r *renderExampleList) ΛListKeyMap() (map[string]interface{}, error) {
	return map[string]interface{}{"val": *r.Val}, nil
}

// renderExampleEnumList is a list entry that is keyed on an enum
// in renderExample.
type renderExampleEnumList struct {
	Key EnumTest `path:"config/key|key"`
}

// IsYANGGoStruct implements the ValidatedGoStruct interface.
func (*renderExampleEnumList) IsYANGGoStruct()                         {}
func (*renderExampleEnumList) Validate(...ValidationOption) error      { return nil }
func (*renderExampleEnumList) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*renderExampleEnumList) ΛBelongingModule() string                { return "" }

// EnumTest is a synthesised derived type which is used to represent
// an enumeration in the YANG schema.
type EnumTest int64

// IsYANGEnumeration ensures that the EnumTest derived enum type implemnts
// the GoEnum interface.
func (EnumTest) IsYANGGoEnum() {}

func (EnumTest) IsExampleUnion() {}

// ΛMap returns the enumeration dictionary associated with the mapStructTestFiveC
// struct.
func (EnumTest) ΛMap() map[string]map[int64]EnumDefinition {
	return map[string]map[int64]EnumDefinition{
		"EnumTest": {
			1: EnumDefinition{Name: "VAL_ONE", DefiningModule: "foo"},
			2: EnumDefinition{Name: "VAL_TWO", DefiningModule: "bar"},
		},
	}
}

func (e EnumTest) String() string {
	return EnumLogString(e, int64(e), "EnumTest")
}

const (
	// EnumTestUNSET is used to represent the unset value of the
	// /c/test enumerated value across a number of tests.
	EnumTestUNSET EnumTest = 0
	// EnumTestVALONE is used to represent VAL_ONE of the /c/test
	// enumerated leaf in the schema-with-list test.
	EnumTestVALONE EnumTest = 1
	// EnumTestVALTWO is used to represent VAL_TWO of the /c/test
	// enumerated leaf in the schema-with-list test.
	EnumTestVALTWO EnumTest = 2
	// EnumTestVALTHREE is an an enum value that does not have
	// a corresponding string mapping.
	EnumTestVALTHREE EnumTest = 3
)

// pathElemExample is an example struct used for rendering using gNMI PathElems.
type pathElemExample struct {
	List        map[string]*pathElemExampleChild                                  `path:"list"`
	StringField *string                                                           `path:"string-field"`
	MKey        map[pathElemExampleMultiKeyChildKey]*pathElemExampleMultiKeyChild `path:"m-key"`
}

// IsYANGGoStruct ensures that pathElemExample implements GoStruct.
func (*pathElemExample) IsYANGGoStruct()                         {}
func (*pathElemExample) Validate(...ValidationOption) error      { return nil }
func (*pathElemExample) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*pathElemExample) ΛBelongingModule() string                { return "" }

// pathElemExampleChild is an example struct that is used as a list child struct.
type pathElemExampleChild struct {
	Val        *string `path:"val|config/val" shadow-path:"val|state/val"`
	OtherField *uint8  `path:"other-field"`
}

// IsYANGGoStruct ensures that pathElemExampleChild implements GoStruct.
func (*pathElemExampleChild) IsYANGGoStruct()                         {}
func (*pathElemExampleChild) Validate(...ValidationOption) error      { return nil }
func (*pathElemExampleChild) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*pathElemExampleChild) ΛBelongingModule() string                { return "" }

// ΛListKeyMap ensures that pathElemExampleChild implements the KeyHelperGoStruct
// helper.
func (p *pathElemExampleChild) ΛListKeyMap() (map[string]interface{}, error) {
	if p.Val == nil {
		return nil, fmt.Errorf("invalid input, key Val was nil")
	}
	return map[string]interface{}{
		"val": *p.Val,
	}, nil
}

// pathElemUnserialisable is an example struct that is used as a list child struct.
type pathElemUnserialisable struct {
	Complex *complex128 `path:"complex"`
}

// IsYANGGoStruct ensures that pathElemUnserialisable implements GoStruct.
func (*pathElemUnserialisable) IsYANGGoStruct()                         {}
func (*pathElemUnserialisable) Validate(...ValidationOption) error      { return nil }
func (*pathElemUnserialisable) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*pathElemUnserialisable) ΛBelongingModule() string                { return "" }

// ΛListKeyMap ensures that pathElemUnserialisable implements the KeyHelperGoStruct
// helper.
func (p *pathElemUnserialisable) ΛListKeyMap() (map[string]interface{}, error) {
	if p.Complex == nil {
		return nil, fmt.Errorf("invalid input, key Val was nil")
	}
	return map[string]interface{}{
		"complex": *p.Complex,
	}, nil
}

// pathElemExampleMultiKeyChild is an example struct that is used as a list child
// struct where there are multiple keys.
type pathElemExampleMultiKeyChild struct {
	Foo *string `path:"foo"`
	Bar *uint16 `path:"bar"`
	Baz *uint8  `path:"baz"`
}

// IsYANGGoStruct ensures that pathElemExampleMultiKeyChild implements the ValidatedGoStruct
// interface.
func (*pathElemExampleMultiKeyChild) IsYANGGoStruct()                         {}
func (*pathElemExampleMultiKeyChild) Validate(...ValidationOption) error      { return nil }
func (*pathElemExampleMultiKeyChild) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*pathElemExampleMultiKeyChild) ΛBelongingModule() string                { return "" }

// ΛListKeyMap ensurs that pathElemExampleMultiKeyChild implements the KeyHelperGoStruct
// interface.
func (p *pathElemExampleMultiKeyChild) ΛListKeyMap() (map[string]interface{}, error) {
	if p.Foo == nil {
		return nil, fmt.Errorf("invalid input, key Foo was nil")
	}

	if p.Bar == nil {
		return nil, fmt.Errorf("invalid input, key Bar was nil")
	}
	return map[string]interface{}{
		"foo": *p.Foo,
		"bar": *p.Bar,
	}, nil
}

// pathElemExampleMultiKeyChildKey is the key type used for the MultiKeyChild list.
type pathElemExampleMultiKeyChildKey struct {
	Foo string `path:"foo"`
	Bar uint16 `path:"bar"`
}

func TestTogNMINotifications(t *testing.T) {
	tests := []struct {
		name        string
		inTimestamp int64
		inStruct    GoStruct
		inConfig    GNMINotificationsConfig
		want        []*gnmipb.Notification
		wantErr     bool
	}{{
		name:        "simple single leaf example",
		inTimestamp: 42,
		inStruct:    &renderExample{Str: String("hello")},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"str"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"hello"}},
			}},
		}},
	}, {
		name:        "simple float value leaf example",
		inTimestamp: 42,
		inStruct:    &renderExample{FloatVal: Float32(42.0)},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"floatval"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_FloatVal{42.0}},
			}},
		}},
	}, {
		name:        "struct with invalid GoStruct map",
		inTimestamp: 42,
		inStruct: &renderExample{
			InvalidMap: map[string]*invalidGoStruct{
				"test": {Value: String("test")},
			},
		},
		wantErr: true,
	}, {
		name:        "nil value",
		inTimestamp: 42,
		inStruct:    nil,
		wantErr:     true,
	}, {
		name:        "no path tags on struct",
		inTimestamp: 42,
		inStruct:    &invalidGoStructEntity{NoPath: String("foo")},
		wantErr:     true,
	}, {
		name:        "struct with invalid pointer",
		inTimestamp: 42,
		inStruct: &renderExample{
			InvalidPtr: &invalidGoStruct{Value: String("fish")},
		},
		wantErr: true,
	}, {
		name:        "simple binary single leaf example",
		inTimestamp: 42,
		inStruct: &renderExample{
			Binary: Binary([]byte{42}),
		},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"binary"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_BytesVal{[]byte{42}}},
			}},
		}},
	}, {
		name:        "struct with enum",
		inTimestamp: 84,
		inStruct:    &renderExample{EnumField: EnumTestVALONE},
		want: []*gnmipb.Notification{{
			Timestamp: 84,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"enum"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_ONE"}},
			}},
		}},
	}, {
		name:        "struct with invalid enum",
		inTimestamp: 42,
		inStruct:    &renderExample{EnumField: EnumTestVALTHREE},
		wantErr:     true,
	}, {
		name:        "struct with leaflist",
		inTimestamp: 42,
		inStruct:    &renderExample{LeafList: []string{"one", "two"}},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"leaf-list"}},
				Val: &gnmipb.TypedValue{
					Value: &gnmipb.TypedValue_LeaflistVal{
						&gnmipb.ScalarArray{
							Element: []*gnmipb.TypedValue{{
								Value: &gnmipb.TypedValue_StringVal{"one"},
							}, {
								Value: &gnmipb.TypedValue_StringVal{"two"},
							}},
						},
					},
				},
			}},
		}},
	}, {
		name:        "struct with enum union",
		inTimestamp: 42,
		inStruct:    &renderExample{UnionValSimple: EnumTestVALONE},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"union-val-simple"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_ONE"}},
			}},
		}},
	}, {
		name:        "struct with int64 union",
		inTimestamp: 42,
		inStruct:    &renderExample{UnionValSimple: testutil.UnionInt64(42)},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"union-val-simple"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_IntVal{42}},
			}},
		}},
	}, {
		name:        "struct with float64 union",
		inTimestamp: 42,
		inStruct:    &renderExample{UnionValSimple: testutil.UnionFloat64(3.14)},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"union-val-simple"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_FloatVal{3.14}},
			}},
		}},
	}, {
		name:        "struct with binary union",
		inTimestamp: 42,
		inStruct:    &renderExample{UnionValSimple: testBinary},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"union-val-simple"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_BytesVal{[]byte(base64testString)}},
			}},
		}},
	}, {
		name:        "string with leaf-list of union",
		inTimestamp: 42,
		inStruct: &renderExample{
			UnionLeafListSimple: []exampleUnion{
				testBinary,
				EnumTestVALTWO,
				testutil.UnionInt64(42),
				testutil.UnionFloat64(3.14),
			},
		},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"union-list-simple"}},
				Val: &gnmipb.TypedValue{
					Value: &gnmipb.TypedValue_LeaflistVal{
						&gnmipb.ScalarArray{
							Element: []*gnmipb.TypedValue{{
								Value: &gnmipb.TypedValue_BytesVal{[]byte(base64testString)},
							}, {
								Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"},
							}, {
								Value: &gnmipb.TypedValue_IntVal{42},
							}, {
								Value: &gnmipb.TypedValue_FloatVal{3.14},
							}},
						},
					},
				},
			}},
		}},
	}, {
		name:        "struct with string union (wrapper union)",
		inTimestamp: 42,
		inStruct:    &renderExample{UnionVal: &renderExampleUnionString{"hello"}},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"union-val"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"hello"}},
			}},
		}},
	}, {
		name:        "struct with int64 union (wrapper union)",
		inTimestamp: 42,
		inStruct:    &renderExample{UnionVal: &renderExampleUnionInt64{42}},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"union-val"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_IntVal{42}},
			}},
		}},
	}, {
		name:        "struct with binary union (wrapper union)",
		inTimestamp: 42,
		inStruct:    &renderExample{UnionVal: &renderExampleUnionBinary{Binary(base64testString)}},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"union-val"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_BytesVal{[]byte(base64testString)}},
			}},
		}},
	}, {
		name:        "invalid union (wrapper union)",
		inTimestamp: 42,
		inStruct:    &renderExample{UnionVal: &renderExampleUnionInvalid{String: "hello", Int8: 42}},
		wantErr:     true,
	}, {
		name:        "string with leaf-list of union (wrapper union)",
		inTimestamp: 42,
		inStruct: &renderExample{
			UnionLeafList: []renderExampleUnion{
				&renderExampleUnionString{"frog"},
			},
		},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"union-list"}},
				Val: &gnmipb.TypedValue{
					Value: &gnmipb.TypedValue_LeaflistVal{
						&gnmipb.ScalarArray{
							Element: []*gnmipb.TypedValue{{
								Value: &gnmipb.TypedValue_StringVal{"frog"},
							}},
						},
					},
				},
			}},
		}},
	}, {
		name:        "struct with mixed leaflist",
		inTimestamp: 720,
		inStruct: &renderExample{MixedList: []interface{}{
			42.42, int8(-42), int16(-84), int32(-168), int64(-336),
			uint8(12), uint16(144), uint32(20736), uint64(429981696),
			true, EnumTestVALTWO, float32(42.0),
		}},
		want: []*gnmipb.Notification{{
			Timestamp: 720,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"mixed-list"}},
				Val: &gnmipb.TypedValue{
					Value: &gnmipb.TypedValue_LeaflistVal{
						&gnmipb.ScalarArray{
							Element: []*gnmipb.TypedValue{{
								Value: &gnmipb.TypedValue_FloatVal{42.42},
							}, {
								Value: &gnmipb.TypedValue_IntVal{-42},
							}, {
								Value: &gnmipb.TypedValue_IntVal{-84},
							}, {
								Value: &gnmipb.TypedValue_IntVal{-168},
							}, {
								Value: &gnmipb.TypedValue_IntVal{-336},
							}, {
								Value: &gnmipb.TypedValue_UintVal{12},
							}, {
								Value: &gnmipb.TypedValue_UintVal{144},
							}, {
								Value: &gnmipb.TypedValue_UintVal{20736},
							}, {
								Value: &gnmipb.TypedValue_UintVal{429981696},
							}, {
								Value: &gnmipb.TypedValue_BoolVal{true},
							}, {
								Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"},
							}, {
								Value: &gnmipb.TypedValue_FloatVal{42.0},
							}},
						},
					},
				},
			}},
		}},
	}, {
		name:        "struct with child struct",
		inTimestamp: 420042,
		inStruct: &renderExample{
			Str:    String("beeblebrox"),
			IntVal: Int32(42),
			Ch:     &renderExampleChild{Val: Uint64(42)},
		},
		inConfig: GNMINotificationsConfig{
			StringSlicePrefix: []string{"base"},
		},
		want: []*gnmipb.Notification{{
			Timestamp: 420042,
			Prefix: &gnmipb.Path{
				Element: []string{"base"},
			},
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"str"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"beeblebrox"}},
			}, {
				Path: &gnmipb.Path{Element: []string{"int-val"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_IntVal{42}},
			}, {
				Path: &gnmipb.Path{Element: []string{"ch", "val"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{42}},
			}},
		}},
	}, {
		name:        "struct with list",
		inTimestamp: 42,
		inStruct: &renderExample{
			List: map[uint32]*renderExampleList{
				42: {String("hello")},
				84: {String("zaphod")},
			},
		},
		inConfig: GNMINotificationsConfig{
			StringSlicePrefix: []string{"heart", "of", "gold"},
		},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Prefix:    &gnmipb.Path{Element: []string{"heart", "of", "gold"}},
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"list", "42", "val"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"hello"}},
			}, {
				Path: &gnmipb.Path{Element: []string{"list", "42", "state", "val"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"hello"}},
			}, {
				Path: &gnmipb.Path{Element: []string{"list", "84", "val"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"zaphod"}},
			}, {
				Path: &gnmipb.Path{Element: []string{"list", "84", "state", "val"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"zaphod"}},
			}},
		}},
	}, {
		name:        "struct with enum keyed list",
		inTimestamp: 42,
		inStruct: &renderExample{
			EnumList: map[EnumTest]*renderExampleEnumList{
				EnumTestVALTWO: {EnumTestVALTWO},
			},
		},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "key"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
			}, {
				Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "config", "key"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
			}},
		}},
	}, {
		name:        "keyless list",
		inTimestamp: 42,
		inStruct: &renderExample{
			KeylessList: []*renderExampleList{
				{String("trillian")},
				{String("arthur")},
			},
		},
		wantErr: true, //unimplemented.
	}, {
		name:        "invalid element in leaf-list",
		inTimestamp: 42,
		inStruct: &renderExample{
			MixedList: []interface{}{struct{ Foo string }{"bar"}},
		},
		wantErr: true,
	}, {
		name:        "invalid slice within a slice",
		inTimestamp: 42,
		inStruct: &renderExample{
			MixedList: []interface{}{[]string{"foo"}},
		},
		wantErr: true,
	}, {
		name:        "simple pathElemExample",
		inTimestamp: 42,
		inStruct: &pathElemExample{
			StringField: String("foo"),
			List: map[string]*pathElemExampleChild{
				"p1": {Val: String("p1"), OtherField: Uint8(42)},
				"p2": {Val: String("p2"), OtherField: Uint8(84)},
			},
		},
		inConfig: GNMINotificationsConfig{UsePathElem: true},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "string-field",
					}},
				},
				Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"foo"}},
			}, {
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "list",
						Key:  map[string]string{"val": "p1"},
					}, {
						Name: "val",
					}},
				},
				Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"p1"}},
			}, {
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "list",
						Key:  map[string]string{"val": "p1"},
					}, {
						Name: "config",
					}, {
						Name: "val",
					}},
				},
				Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"p1"}},
			}, {
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "list",
						Key:  map[string]string{"val": "p1"},
					}, {
						Name: "other-field",
					}},
				},
				Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{42}},
			}, {
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "list",
						Key:  map[string]string{"val": "p2"},
					}, {
						Name: "val",
					}},
				},
				Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"p2"}},
			}, {
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "list",
						Key:  map[string]string{"val": "p2"},
					}, {
						Name: "config",
					}, {
						Name: "val",
					}},
				},
				Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"p2"}},
			}, {
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "list",
						Key:  map[string]string{"val": "p2"},
					}, {
						Name: "other-field",
					}},
				},
				Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{84}},
			}},
		}},
	}, {
		name:        "multi key example with path elements",
		inTimestamp: 42,
		inStruct: &pathElemExample{
			MKey: map[pathElemExampleMultiKeyChildKey]*pathElemExampleMultiKeyChild{
				{Foo: "foo", Bar: 16}: {Foo: String("foo"), Bar: Uint16(16)},
			},
		},
		inConfig: GNMINotificationsConfig{UsePathElem: true},
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "m-key",
						Key: map[string]string{
							"foo": "foo",
							"bar": "16",
						},
					}, {
						Name: "foo",
					}},
				},
				Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"foo"}},
			}, {
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "m-key",
						Key: map[string]string{
							"foo": "foo",
							"bar": "16",
						},
					}, {
						Name: "bar",
					}},
				},
				Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{16}},
			}},
		}},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TogNMINotifications(tt.inStruct, tt.inTimestamp, tt.inConfig)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("%s: TogNMINotifications(%v, %v, %v): got unexpected error: %v", tt.name, tt.inStruct, tt.inTimestamp, tt.inConfig, err)
				}
				return
			}

			// Avoid test flakiness by ignoring the update ordering. Required because
			// there is no order to the map of fields that are returned by the struct
			// output.

			if !testutil.NotificationSetEqual(got, tt.want) {
				diff := cmp.Diff(got, tt.want, protocmp.Transform())
				t.Errorf("%s: TogNMINotifications(%v, %v): did not get expected Notification, diff(-got,+want):%s\n", tt.name, tt.inStruct, tt.inTimestamp, diff)
			}
		})
	}
}

// exampleDevice and the following structs are a set of structs used for more
// complex testing in TestConstructIETFJSON
type exampleDevice struct {
	Bgp *exampleBgp `path:""`
}

func (*exampleDevice) IsYANGGoStruct()                         {}
func (*exampleDevice) Validate(...ValidationOption) error      { return nil }
func (*exampleDevice) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*exampleDevice) ΛBelongingModule() string                { return "" }

type exampleBgp struct {
	Global   *exampleBgpGlobal              `path:"bgp/global"`
	Neighbor map[string]*exampleBgpNeighbor `path:"bgp/neighbors/neighbor"`
}

func (*exampleBgp) IsYANGGoStruct()                         {}
func (*exampleBgp) Validate(...ValidationOption) error      { return nil }
func (*exampleBgp) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*exampleBgp) ΛBelongingModule() string                { return "" }

type exampleBgpGlobal struct {
	As       *uint32 `path:"config/as"`
	RouterID *string `path:"config/router-id"`
}

func (*exampleBgpGlobal) IsYANGGoStruct()                         {}
func (*exampleBgpGlobal) Validate(...ValidationOption) error      { return nil }
func (*exampleBgpGlobal) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*exampleBgpGlobal) ΛBelongingModule() string                { return "" }

type exampleBgpNeighbor struct {
	Description                  *string                                         `path:"config/description"`
	Enabled                      *bool                                           `path:"config/enabled"`
	NeighborAddress              *string                                         `path:"config/neighbor-address|neighbor-address"`
	PeerAs                       *uint32                                         `path:"config/peer-as"`
	TransportAddress             exampleTransportAddress                         `path:"state/transport-address"`
	TransportAddressSimple       exampleUnion                                    `path:"state/transport-address-simple"`
	EnabledAddressFamilies       []exampleBgpNeighborEnabledAddressFamiliesUnion `path:"state/enabled-address-families"`
	EnabledAddressFamiliesSimple []exampleUnion                                  `path:"state/enabled-address-families-simple"`
	MessageDump                  Binary                                          `path:"state/message-dump"`
	Updates                      []Binary                                        `path:"state/updates"`
}

func (*exampleBgpNeighbor) IsYANGGoStruct()                         {}
func (*exampleBgpNeighbor) Validate(...ValidationOption) error      { return nil }
func (*exampleBgpNeighbor) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*exampleBgpNeighbor) ΛBelongingModule() string                { return "" }

// exampleBgpNeighborEnabledAddressFamiliesUnion is an interface that is implemented by
// valid types of the EnabledAddressFamilies field of the exampleBgpNeighbor struct.
type exampleBgpNeighborEnabledAddressFamiliesUnion interface {
	IsExampleBgpNeighborEnabledAddressFamiliesUnion()
}

type exampleBgpNeighborEnabledAddressFamiliesUnionString struct {
	String string
}

func (*exampleBgpNeighborEnabledAddressFamiliesUnionString) IsExampleBgpNeighborEnabledAddressFamiliesUnion() {
}

type exampleBgpNeighborEnabledAddressFamiliesUnionUint64 struct {
	Uint64 uint64
}

func (*exampleBgpNeighborEnabledAddressFamiliesUnionUint64) IsExampleBgpNeighborEnabledAddressFamiliesUnion() {
}

type exampleBgpNeighborEnabledAddressFamiliesUnionBinary struct {
	Binary Binary
}

func (*exampleBgpNeighborEnabledAddressFamiliesUnionBinary) IsExampleBgpNeighborEnabledAddressFamiliesUnion() {
}

// exampleTransportAddress is an interface implemnented by valid types of the
// TransportAddress union.
type exampleTransportAddress interface {
	IsExampleTransportAddress()
}

type exampleTransportAddressString struct {
	String string
}

func (*exampleTransportAddressString) IsExampleTransportAddress() {}

type exampleTransportAddressUint64 struct {
	Uint64 uint64
}

func (*exampleTransportAddressUint64) IsExampleTransportAddress() {}

type exampleTransportAddressEnum struct {
	E EnumTest
}

func (*exampleTransportAddressEnum) IsExampleTransportAddress() {}

type exampleTransportAddressBinary struct {
	Binary Binary
}

func (*exampleTransportAddressBinary) IsExampleTransportAddress() {}

// invalidGoStruct explicitly does not implement the ValidatedGoStruct interface.
type invalidGoStruct struct {
	Value *string
}

type invalidGoStructChild struct {
	Child *invalidGoStruct `path:"child"`
}

func (*invalidGoStructChild) IsYANGGoStruct()                         {}
func (*invalidGoStructChild) Validate(...ValidationOption) error      { return nil }
func (*invalidGoStructChild) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*invalidGoStructChild) ΛBelongingModule() string                { return "" }

type invalidGoStructField struct {
	// A string is not directly allowed inside a GoStruct
	Value string `path:"value"`
}

func (*invalidGoStructField) IsYANGGoStruct()                         {}
func (*invalidGoStructField) Validate(...ValidationOption) error      { return nil }
func (*invalidGoStructField) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*invalidGoStructField) ΛBelongingModule() string                { return "" }

// invalidGoStructEntity is a GoStruct that contains invalid path data.
type invalidGoStructEntity struct {
	EmptyPath   *string `path:""`
	NoPath      *string
	InvalidEnum int64 `path:"an-enum"`
}

func (*invalidGoStructEntity) IsYANGGoStruct()                         {}
func (*invalidGoStructEntity) Validate(...ValidationOption) error      { return nil }
func (*invalidGoStructEntity) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*invalidGoStructEntity) ΛBelongingModule() string                { return "" }

type invalidGoStructMapChild struct {
	InvalidField string
}

func (*invalidGoStructMapChild) IsYANGGoStruct()                         {}
func (*invalidGoStructMapChild) Validate(...ValidationOption) error      { return nil }
func (*invalidGoStructMapChild) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*invalidGoStructMapChild) ΛBelongingModule() string                { return "" }

type invalidGoStructMap struct {
	Map    map[string]*invalidGoStructMapChild `path:"foobar"`
	FooBar map[string]*invalidGoStruct         `path:"baz"`
}

func (*invalidGoStructMap) IsYANGGoStruct()                         {}
func (*invalidGoStructMap) Validate(...ValidationOption) error      { return nil }
func (*invalidGoStructMap) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*invalidGoStructMap) ΛBelongingModule() string                { return "" }

type structWithMultiKey struct {
	Map map[mapKey]*structMultiKeyChild `path:"foo" module:"rootmod"`
}

func (*structWithMultiKey) IsYANGGoStruct()                         {}
func (*structWithMultiKey) Validate(...ValidationOption) error      { return nil }
func (*structWithMultiKey) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*structWithMultiKey) ΛBelongingModule() string                { return "" }

type structWithMultiKeyInvalidModuleTag struct {
	Map map[mapKey]*structMultiKeyChild `path:"foo/bar" module:"rootmod"`
}

func (*structWithMultiKeyInvalidModuleTag) IsYANGGoStruct()                         {}
func (*structWithMultiKeyInvalidModuleTag) Validate(...ValidationOption) error      { return nil }
func (*structWithMultiKeyInvalidModuleTag) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*structWithMultiKeyInvalidModuleTag) ΛBelongingModule() string                { return "" }

type structWithMultiKeyInvalidModuleTag2 struct {
	Map map[mapKey]*structMultiKeyChild `path:"foo" module:"rootmod/rootmod"`
}

func (*structWithMultiKeyInvalidModuleTag2) IsYANGGoStruct()                         {}
func (*structWithMultiKeyInvalidModuleTag2) Validate(...ValidationOption) error      { return nil }
func (*structWithMultiKeyInvalidModuleTag2) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*structWithMultiKeyInvalidModuleTag2) ΛBelongingModule() string                { return "" }

type structWithMultiKeyInvalidModuleTag3 struct {
	Map map[mapKey]*structMultiKeyChild `path:"foo/bar" module:"rootmod/rootmod|rootmod"`
}

func (*structWithMultiKeyInvalidModuleTag3) IsYANGGoStruct()                         {}
func (*structWithMultiKeyInvalidModuleTag3) Validate(...ValidationOption) error      { return nil }
func (*structWithMultiKeyInvalidModuleTag3) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*structWithMultiKeyInvalidModuleTag3) ΛBelongingModule() string                { return "" }

type structWithMultiKeyInvalidModuleTag4 struct {
	Map map[mapKey]*structMultiKeyChild `path:"foo/bar" module:""`
}

func (*structWithMultiKeyInvalidModuleTag4) IsYANGGoStruct()                         {}
func (*structWithMultiKeyInvalidModuleTag4) Validate(...ValidationOption) error      { return nil }
func (*structWithMultiKeyInvalidModuleTag4) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*structWithMultiKeyInvalidModuleTag4) ΛBelongingModule() string                { return "" }

type structWithMultiKeyInvalidModuleTag5 struct {
	Map map[mapKey]*structMultiKeyChild `path:"foo/bar" module:"rootmod/rootmod2|rootmod"`
}

func (*structWithMultiKeyInvalidModuleTag5) IsYANGGoStruct()                         {}
func (*structWithMultiKeyInvalidModuleTag5) Validate(...ValidationOption) error      { return nil }
func (*structWithMultiKeyInvalidModuleTag5) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*structWithMultiKeyInvalidModuleTag5) ΛBelongingModule() string                { return "" }

type mapKey struct {
	F1 string `path:"fOne"`
	F2 string `path:"fTwo"`
}

type structMultiKeyChild struct {
	F1 *string `path:"config/fOne|fOne" module:"fmod/f1mod|f1mod" shadow-path:"state/fOne|fOne" shadow-module:"fmod/f1mod|f1mod"`
	F2 *string `path:"config/fTwo|fTwo" module:"fmod/f2mod|f2mod" shadow-path:"state/fTwo|fTwo" shadow-module:"fmod/f2mod-shadow|f2mod-shadow"`
}

func (*structMultiKeyChild) IsYANGGoStruct()                         {}
func (*structMultiKeyChild) Validate(...ValidationOption) error      { return nil }
func (*structMultiKeyChild) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*structMultiKeyChild) ΛBelongingModule() string                { return "" }

// ietfRenderExampleEnumList is a list entry that is keyed on an enum
// in ietfRenderExample.
type ietfRenderExampleEnumList struct {
	Key EnumTest `path:"config/key|key" module:"f1mod/f1mod|f1mod"`
}

// IsYANGGoStruct implements the ValidatedGoStruct interface.
func (*ietfRenderExampleEnumList) IsYANGGoStruct()                         {}
func (*ietfRenderExampleEnumList) Validate(...ValidationOption) error      { return nil }
func (*ietfRenderExampleEnumList) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*ietfRenderExampleEnumList) ΛBelongingModule() string                { return "f1mod" }

type ietfRenderExample struct {
	F1        *string                                 `path:"f1" module:"f1mod"`
	F2        *string                                 `path:"config/f2" module:"f2mod/f2mod"`
	F3        *ietfRenderExampleChild                 `path:"f3" module:"f1mod"`
	F6        *string                                 `path:"config/f6" module:"f1mod/f2mod"`
	F7        *string                                 `path:"config/f7" module:"f2mod/f3mod"`
	MixedList []interface{}                           `path:"mixed-list" module:"f1mod"`
	EnumList  map[EnumTest]*ietfRenderExampleEnumList `path:"enum-list" module:"f1mod"`
}

func (*ietfRenderExample) IsYANGGoStruct()                         {}
func (*ietfRenderExample) Validate(...ValidationOption) error      { return nil }
func (*ietfRenderExample) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*ietfRenderExample) ΛBelongingModule() string                { return "f1mod" }

type ietfRenderExampleChild struct {
	F4 *string `path:"config/f4" module:"f42mod/f42mod"`
	F5 *string `path:"f5" module:"f1mod"`
}

func (*ietfRenderExampleChild) IsYANGGoStruct()                         {}
func (*ietfRenderExampleChild) Validate(...ValidationOption) error      { return nil }
func (*ietfRenderExampleChild) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*ietfRenderExampleChild) ΛBelongingModule() string                { return "" }

type listAtRoot struct {
	Foo map[string]*listAtRootChild `path:"foo" rootname:"foo" module:"m1"`
}

func (*listAtRoot) IsYANGGoStruct()                         {}
func (*listAtRoot) Validate(...ValidationOption) error      { return nil }
func (*listAtRoot) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*listAtRoot) ΛBelongingModule() string                { return "" }

type listAtRootChild struct {
	Bar *string `path:"bar" module:"m1"`
}

func (*listAtRootChild) IsYANGGoStruct()                         {}
func (*listAtRootChild) Validate(...ValidationOption) error      { return nil }
func (*listAtRootChild) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*listAtRootChild) ΛBelongingModule() string                { return "m1" }

type listAtRootEnumKeyed struct {
	Foo map[EnumTest]*listAtRootChildEnumKeyed `path:"foo" rootname:"foo" module:"m1"`
}

func (*listAtRootEnumKeyed) IsYANGGoStruct()                         {}
func (*listAtRootEnumKeyed) Validate(...ValidationOption) error      { return nil }
func (*listAtRootEnumKeyed) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*listAtRootEnumKeyed) ΛBelongingModule() string                { return "" }

type listAtRootChildEnumKeyed struct {
	Bar EnumTest `path:"bar" module:"m1"`
}

func (*listAtRootChildEnumKeyed) IsYANGGoStruct()                         {}
func (*listAtRootChildEnumKeyed) Validate(...ValidationOption) error      { return nil }
func (*listAtRootChildEnumKeyed) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*listAtRootChildEnumKeyed) ΛBelongingModule() string                { return "m1" }

// Types to ensure correct serialisation of elements with different
// modules at the root.
type diffModAtRoot struct {
	Child *diffModAtRootChild `path:"" module:"m1"`
	Elem  *diffModAtRootElem  `path:"" module:"m1"`
}

func (*diffModAtRoot) IsYANGGoStruct()                         {}
func (*diffModAtRoot) Validate(...ValidationOption) error      { return nil }
func (*diffModAtRoot) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*diffModAtRoot) ΛBelongingModule() string                { return "" }

type diffModAtRootChild struct {
	ValueOne   *string `path:"/foo/value-one" module:"/m1/m2"`
	ValueTwo   *string `path:"/foo/value-two" module:"/m1/m3"`
	ValueThree *string `path:"/foo/value-three" module:"/m1/m1"`
}

func (*diffModAtRootChild) IsYANGGoStruct()                         {}
func (*diffModAtRootChild) Validate(...ValidationOption) error      { return nil }
func (*diffModAtRootChild) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*diffModAtRootChild) ΛBelongingModule() string                { return "m1" }

type diffModAtRootElem struct {
	C *diffModAtRootElemTwo `path:"/baz/c" module:"/m1/m1"`
}

func (*diffModAtRootElem) IsYANGGoStruct()                         {}
func (*diffModAtRootElem) Validate(...ValidationOption) error      { return nil }
func (*diffModAtRootElem) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*diffModAtRootElem) ΛBelongingModule() string                { return "m1" }

type diffModAtRootElemTwo struct {
	Name *string `path:"name" module:"m1"`
}

func (*diffModAtRootElemTwo) IsYANGGoStruct()                         {}
func (*diffModAtRootElemTwo) Validate(...ValidationOption) error      { return nil }
func (*diffModAtRootElemTwo) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*diffModAtRootElemTwo) ΛBelongingModule() string                { return "m1" }

type annotatedJSONTestStruct struct {
	Field       *string      `path:"field" module:"bar"`
	ΛField      []Annotation `path:"@field" ygotAnnotation:"true"`
	ΛFieldTwo   []Annotation `path:"@emptyannotation" ygotAnnotation:"true"`
	ΛFieldThree []Annotation `path:"@one|config/@two" ygotAnnotation:"true"`
}

func (*annotatedJSONTestStruct) IsYANGGoStruct()                         {}
func (*annotatedJSONTestStruct) Validate(...ValidationOption) error      { return nil }
func (*annotatedJSONTestStruct) ΛEnumTypeMap() map[string][]reflect.Type { return nil }
func (*annotatedJSONTestStruct) ΛBelongingModule() string                { return "" }

type testAnnotation struct {
	AnnotationFieldOne string `json:"field"`
}

// MarshalJSON repeats the string in the JSON representation.
func (t *testAnnotation) MarshalJSON() ([]byte, error) {
	return json.Marshal(*t)
}

// UnmarshalJSON halves the string from the JSON representation.
func (t *testAnnotation) UnmarshalJSON(d []byte) error {
	return json.Unmarshal(d, t)
}

type errorAnnotation struct {
	AnnotationField string `json:"field"`
}

func (t *errorAnnotation) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("injected error")
}

func (t *errorAnnotation) UnmarshalJSON(d []byte) error {
	return fmt.Errorf("unimplemented")
}

type unmarshalableJSON struct {
	AnnotationField string `json:"field"`
}

func (t *unmarshalableJSON) MarshalJSON() ([]byte, error) {
	return []byte("{{"), nil
}

func (t *unmarshalableJSON) UnmarshalJSON(d []byte) error {
	return fmt.Errorf("unimplemented")
}

func TestConstructJSON(t *testing.T) {
	tests := []struct {
		name                     string
		in                       ValidatedGoStruct
		inAppendMod              bool
		inPrependModIref         bool
		inRewriteModuleNameRules map[string]string
		inPreferShadowPath       bool
		wantIETF                 map[string]interface{}
		wantInternal             map[string]interface{}
		wantSame                 bool
		wantErr                  bool
		wantJSONErr              bool
	}{{
		name: "invalidGoStruct",
		in: &invalidGoStructChild{
			Child: &invalidGoStruct{
				Value: String("foo"),
			},
		},
		wantErr: true,
	}, {
		name: "invalid go struct field",
		in: &invalidGoStructField{
			Value: "invalid",
		},
		wantErr: true,
	}, {
		name: "field with empty path",
		in: &invalidGoStructEntity{
			EmptyPath: String("some string"),
		},
		wantErr: true,
	}, {
		name: "field with no path",
		in: &invalidGoStructEntity{
			NoPath: String("other string"),
		},
		wantErr: true,
	}, {
		name: "field with invalid enum",
		in: &invalidGoStructEntity{
			InvalidEnum: int64(42),
		},
		wantErr: true,
	}, {
		name: "different modules at root",
		in: &diffModAtRoot{
			Child: &diffModAtRootChild{
				ValueOne:   String("one"),
				ValueTwo:   String("two"),
				ValueThree: String("three"),
			},
			Elem: &diffModAtRootElem{
				C: &diffModAtRootElemTwo{
					Name: String("baz"),
				},
			},
		},
		inAppendMod: true,
		wantIETF: map[string]interface{}{
			"m1:foo": map[string]interface{}{
				"m2:value-one": "one",
				"m3:value-two": "two",
				"value-three":  "three",
			},
			"m1:baz": map[string]interface{}{
				"c": map[string]interface{}{
					"name": "baz",
				},
			},
		},
		wantInternal: map[string]interface{}{
			"foo": map[string]interface{}{
				"value-one":   "one",
				"value-two":   "two",
				"value-three": "three",
			},
			"baz": map[string]interface{}{
				"c": map[string]interface{}{
					"name": "baz",
				},
			},
		},
	}, {
		name: "rewrite module name for an element with children",
		in: &diffModAtRoot{
			Child: &diffModAtRootChild{
				ValueOne:   String("one"),
				ValueTwo:   String("two"),
				ValueThree: String("three"),
			},
			Elem: &diffModAtRootElem{
				C: &diffModAtRootElemTwo{
					Name: String("baz"),
				},
			},
		},
		inAppendMod: true,
		inRewriteModuleNameRules: map[string]string{
			// rewrite m1 to m2
			"m1": "m2",
		},
		wantIETF: map[string]interface{}{
			"m2:foo": map[string]interface{}{
				"value-one":    "one",
				"m3:value-two": "two",
				"value-three":  "three",
			},
			"m2:baz": map[string]interface{}{
				"c": map[string]interface{}{
					"name": "baz",
				},
			},
		},
	}, {
		name: "rewrite leaf node module",
		in: &diffModAtRoot{
			Child: &diffModAtRootChild{
				ValueOne:   String("one"),
				ValueTwo:   String("two"),
				ValueThree: String("three"),
			},
			Elem: &diffModAtRootElem{
				C: &diffModAtRootElemTwo{
					Name: String("baz"),
				},
			},
		},
		inAppendMod: true,
		inRewriteModuleNameRules: map[string]string{
			"m3": "fish",
		},
		wantIETF: map[string]interface{}{
			"m1:foo": map[string]interface{}{
				"m2:value-one":   "one",
				"fish:value-two": "two",
				"value-three":    "three",
			},
			"m1:baz": map[string]interface{}{
				"c": map[string]interface{}{
					"name": "baz",
				},
			},
		},
	}, {
		name: "simple render",
		in: &renderExample{
			Str: String("hello"),
		},
		wantIETF: map[string]interface{}{
			"str": "hello",
		},
		wantSame: true,
	}, {
		name: "empty value",
		in: &renderExample{
			Empty: true,
		},
		wantIETF: map[string]interface{}{
			"empty": []interface{}{nil},
		},
		wantInternal: map[string]interface{}{
			"empty": true,
		},
	}, {
		name: "multi-keyed list",
		in: &structWithMultiKey{
			Map: map[mapKey]*structMultiKeyChild{
				{F1: "one", F2: "two"}: {F1: String("one"), F2: String("two")},
			},
		},
		wantIETF: map[string]interface{}{
			"foo": []interface{}{
				map[string]interface{}{
					"fOne": "one",
					"fTwo": "two",
					"config": map[string]interface{}{
						"fOne": "one",
						"fTwo": "two",
					},
				},
			},
		},
		wantInternal: map[string]interface{}{
			"foo": map[string]interface{}{
				"one two": map[string]interface{}{
					"fOne": "one",
					"fTwo": "two",
					"config": map[string]interface{}{
						"fOne": "one",
						"fTwo": "two",
					},
				},
			},
		},
	}, {
		name: "multi-keyed list with PreferShadowPath=true",
		in: &structWithMultiKey{
			Map: map[mapKey]*structMultiKeyChild{
				{F1: "one", F2: "two"}: {F1: String("one"), F2: String("two")},
			},
		},
		inPreferShadowPath: true,
		wantIETF: map[string]interface{}{
			"foo": []interface{}{
				map[string]interface{}{
					"fOne": "one",
					"fTwo": "two",
					"state": map[string]interface{}{
						"fOne": "one",
						"fTwo": "two",
					},
				},
			},
		},
		wantInternal: map[string]interface{}{
			"foo": map[string]interface{}{
				"one two": map[string]interface{}{
					"fOne": "one",
					"fTwo": "two",
					// NOTE: internal JSON generation doesn't have the
					// preferShadowPath option, so its results are unchanged.
					"config": map[string]interface{}{
						"fOne": "one",
						"fTwo": "two",
					},
				},
			},
		},
	}, {
		name: "multi-keyed list with PreferShadowPath=true and appendModules=true",
		in: &structWithMultiKey{
			Map: map[mapKey]*structMultiKeyChild{
				{F1: "one", F2: "two"}: {F1: String("one"), F2: String("two")},
			},
		},
		inPreferShadowPath: true,
		inAppendMod:        true,
		wantIETF: map[string]interface{}{
			"rootmod:foo": []interface{}{
				map[string]interface{}{
					"f1mod:fOne":        "one",
					"f2mod-shadow:fTwo": "two",
					"fmod:state": map[string]interface{}{
						"f1mod:fOne":        "one",
						"f2mod-shadow:fTwo": "two",
					},
				},
			},
		},
		wantInternal: map[string]interface{}{
			"foo": map[string]interface{}{
				"one two": map[string]interface{}{
					"fOne": "one",
					"fTwo": "two",
					// NOTE: internal JSON generation doesn't have the
					// preferShadowPath option, so its results are unchanged.
					"config": map[string]interface{}{
						"fOne": "one",
						"fTwo": "two",
					},
				},
			},
		},
	}, {
		name: "not enough module elements",
		in: &structWithMultiKeyInvalidModuleTag{
			Map: map[mapKey]*structMultiKeyChild{
				{F1: "one", F2: "two"}: {F1: String("one"), F2: String("two")},
			},
		},
		inAppendMod: true,
		wantErr:     true,
	}, {
		name: "too many module elements",
		in: &structWithMultiKeyInvalidModuleTag2{
			Map: map[mapKey]*structMultiKeyChild{
				{F1: "one", F2: "two"}: {F1: String("one"), F2: String("two")},
			},
		},
		inAppendMod: true,
		wantErr:     true,
	}, {
		name: "too many module paths",
		in: &structWithMultiKeyInvalidModuleTag3{
			Map: map[mapKey]*structMultiKeyChild{
				{F1: "one", F2: "two"}: {F1: String("one"), F2: String("two")},
			},
		},
		inAppendMod: true,
		wantErr:     true,
	}, {
		name: "empty modules tag",
		in: &structWithMultiKeyInvalidModuleTag4{
			Map: map[mapKey]*structMultiKeyChild{
				{F1: "one", F2: "two"}: {F1: String("one"), F2: String("two")},
			},
		},
		inAppendMod: true,
		wantErr:     true,
	}, {
		name: "module paths with inconsistent child modules",
		in: &structWithMultiKeyInvalidModuleTag5{
			Map: map[mapKey]*structMultiKeyChild{
				{F1: "one", F2: "two"}: {F1: String("one"), F2: String("two")},
			},
		},
		inAppendMod: true,
		wantErr:     true,
	}, {
		name: "multi-element render",
		in: &renderExample{
			Str:       String("hello"),
			IntVal:    Int32(42),
			EnumField: EnumTestVALTWO,
			LeafList:  []string{"hello", "world"},
			MixedList: []interface{}{uint64(42)},
			KeylessList: []*renderExampleList{
				{Val: String("21st Amendment")},
				{Val: String("Anchor")},
			},
		},
		inAppendMod: true,
		wantIETF: map[string]interface{}{
			"str":        "hello",
			"leaf-list":  []string{"hello", "world"},
			"int-val":    42,
			"enum":       "bar:VAL_TWO",
			"mixed-list": []interface{}{"42"},
			"keyless-list": []interface{}{
				map[string]interface{}{
					"val": "21st Amendment",
					"state": map[string]interface{}{
						"val": "21st Amendment",
					},
				},
				map[string]interface{}{
					"val": "Anchor",
					"state": map[string]interface{}{
						"val": "Anchor",
					},
				},
			},
		},
		wantInternal: map[string]interface{}{
			"str":        "hello",
			"leaf-list":  []string{"hello", "world"},
			"int-val":    42,
			"enum":       "VAL_TWO",
			"mixed-list": []interface{}{42},
			"keyless-list": []interface{}{
				map[string]interface{}{
					"val": "21st Amendment",
					"state": map[string]interface{}{
						"val": "21st Amendment",
					},
				},
				map[string]interface{}{
					"val": "Anchor",
					"state": map[string]interface{}{
						"val": "Anchor",
					},
				},
			},
		},
	}, {
		name: "empty map",
		in: &renderExample{
			Ch: &renderExampleChild{
				Val: Uint64(42),
			},
			List: map[uint32]*renderExampleList{},
		},
		wantIETF: map[string]interface{}{
			"ch": map[string]interface{}{"val": "42"},
			/// RFC7951 Section 5.4 defines a YANG list as an JSON array. Per RFC 8259 Section 5 an empty array should be [] rather than 'null'.
			"list": []interface{}{},
		},
		wantInternal: map[string]interface{}{
			"ch": map[string]interface{}{"val": 42},
		},
	}, {
		name: "empty map nil",
		in: &renderExample{
			Ch: &renderExampleChild{
				Val: Uint64(42),
			},
			List: nil,
		},
		wantIETF: map[string]interface{}{
			"ch": map[string]interface{}{"val": "42"},
		},
		wantInternal: map[string]interface{}{
			"ch": map[string]interface{}{"val": 42},
		},
	}, {
		name:     "empty child",
		in:       &renderExample{Ch: &renderExampleChild{}},
		wantIETF: map[string]interface{}{},
	}, {
		name:    "child with invalid map contents",
		in:      &invalidGoStructMap{Map: map[string]*invalidGoStructMapChild{"foobar": {InvalidField: "foobar"}}},
		wantErr: true,
	}, {
		name:    "child that is not a GoStruct",
		in:      &invalidGoStructMap{FooBar: map[string]*invalidGoStruct{"foobar": {Value: String("fooBar")}}},
		wantErr: true,
	}, {
		name: "json test with complex children",
		in: &renderExample{
			Ch: &renderExampleChild{
				Val: Uint64(42),
			},
			MixedList: []interface{}{EnumTestVALONE, "test", 42},
			List: map[uint32]*renderExampleList{
				42: {Val: String("forty two")},
				84: {Val: String("eighty four")},
			},
			EnumList: map[EnumTest]*renderExampleEnumList{
				EnumTestVALONE: {Key: EnumTestVALONE},
			},
		},
		inAppendMod: true,
		wantIETF: map[string]interface{}{
			"ch": map[string]interface{}{"val": "42"},
			"enum-list": []interface{}{
				map[string]interface{}{
					"config": map[string]interface{}{
						"key": "foo:VAL_ONE",
					},
					"key": "foo:VAL_ONE",
				},
			},
			"list": []interface{}{
				map[string]interface{}{
					"state": map[string]interface{}{
						"val": "forty two",
					},
					"val": "forty two",
				},
				map[string]interface{}{
					"state": map[string]interface{}{
						"val": "eighty four",
					},
					"val": "eighty four",
				},
			},
			"mixed-list": []interface{}{"foo:VAL_ONE", "test", uint32(42)},
		},
		wantInternal: map[string]interface{}{
			"ch": map[string]interface{}{"val": 42},
			"enum-list": map[string]interface{}{
				"VAL_ONE": map[string]interface{}{
					"config": map[string]interface{}{
						"key": "VAL_ONE",
					},
					"key": "VAL_ONE",
				},
			},
			"list": map[string]interface{}{
				"42": map[string]interface{}{
					"state": map[string]interface{}{
						"val": "forty two",
					},
					"val": "forty two",
				},
				"84": map[string]interface{}{
					"state": map[string]interface{}{
						"val": "eighty four",
					},
					"val": "eighty four",
				},
			},
			"mixed-list": []interface{}{"VAL_ONE", "test", uint32(42)},
		},
	}, {
		name: "json test with complex children with PrependModuleNameIdentityref=true",
		in: &renderExample{
			Ch: &renderExampleChild{
				Val: Uint64(42),
			},
			MixedList: []interface{}{EnumTestVALONE, "test", 42},
			List: map[uint32]*renderExampleList{
				42: {Val: String("forty two")},
				84: {Val: String("eighty four")},
			},
			EnumList: map[EnumTest]*renderExampleEnumList{
				EnumTestVALONE: {Key: EnumTestVALONE},
			},
		},
		inPrependModIref: true,
		wantIETF: map[string]interface{}{
			"ch": map[string]interface{}{"val": "42"},
			"enum-list": []interface{}{
				map[string]interface{}{
					"config": map[string]interface{}{
						"key": "foo:VAL_ONE",
					},
					"key": "foo:VAL_ONE",
				},
			},
			"list": []interface{}{
				map[string]interface{}{
					"state": map[string]interface{}{
						"val": "forty two",
					},
					"val": "forty two",
				},
				map[string]interface{}{
					"state": map[string]interface{}{
						"val": "eighty four",
					},
					"val": "eighty four",
				},
			},
			"mixed-list": []interface{}{"foo:VAL_ONE", "test", uint32(42)},
		},
		wantInternal: map[string]interface{}{
			"ch": map[string]interface{}{"val": 42},
			"enum-list": map[string]interface{}{
				"VAL_ONE": map[string]interface{}{
					"config": map[string]interface{}{
						"key": "VAL_ONE",
					},
					"key": "VAL_ONE",
				},
			},
			"list": map[string]interface{}{
				"42": map[string]interface{}{
					"state": map[string]interface{}{
						"val": "forty two",
					},
					"val": "forty two",
				},
				"84": map[string]interface{}{
					"state": map[string]interface{}{
						"val": "eighty four",
					},
					"val": "eighty four",
				},
			},
			"mixed-list": []interface{}{"VAL_ONE", "test", uint32(42)},
		},
	}, {
		name: "device example #1",
		in: &exampleDevice{
			Bgp: &exampleBgp{
				Global: &exampleBgpGlobal{
					As:       Uint32(15169),
					RouterID: String("192.0.2.1"),
				},
			},
		},
		wantIETF: map[string]interface{}{
			"bgp": map[string]interface{}{
				"global": map[string]interface{}{
					"config": map[string]interface{}{
						"as":        15169,
						"router-id": "192.0.2.1",
					},
				},
			},
		},
		wantSame: true,
	}, {
		name: "device example #2",
		in: &exampleDevice{
			Bgp: &exampleBgp{
				Neighbor: map[string]*exampleBgpNeighbor{
					"192.0.2.1": {
						Description:     String("a neighbor"),
						Enabled:         Bool(true),
						NeighborAddress: String("192.0.2.1"),
						PeerAs:          Uint32(29636),
					},
					"100.64.32.96": {
						Description:     String("a second neighbor"),
						Enabled:         Bool(false),
						NeighborAddress: String("100.64.32.96"),
						PeerAs:          Uint32(5413),
					},
				},
			},
		},
		wantIETF: map[string]interface{}{
			"bgp": map[string]interface{}{
				"neighbors": map[string]interface{}{
					"neighbor": []interface{}{
						map[string]interface{}{
							"config": map[string]interface{}{
								"description":      "a second neighbor",
								"enabled":          false,
								"neighbor-address": "100.64.32.96",
								"peer-as":          5413,
							},
							"neighbor-address": "100.64.32.96",
						},
						map[string]interface{}{
							"config": map[string]interface{}{
								"description":      "a neighbor",
								"enabled":          true,
								"neighbor-address": "192.0.2.1",
								"peer-as":          29636,
							},
							"neighbor-address": "192.0.2.1",
						},
					},
				},
			},
		},
		wantInternal: map[string]interface{}{
			"bgp": map[string]interface{}{
				"neighbors": map[string]interface{}{
					"neighbor": map[string]interface{}{
						"192.0.2.1": map[string]interface{}{
							"config": map[string]interface{}{
								"description":      "a neighbor",
								"enabled":          true,
								"neighbor-address": "192.0.2.1",
								"peer-as":          29636,
							},
							"neighbor-address": "192.0.2.1",
						},
						"100.64.32.96": map[string]interface{}{
							"config": map[string]interface{}{
								"description":      "a second neighbor",
								"enabled":          false,
								"neighbor-address": "100.64.32.96",
								"peer-as":          5413,
							},
							"neighbor-address": "100.64.32.96",
						},
					},
				},
			},
		},
	}, {
		name: "union leaf-list example",
		in: &exampleBgpNeighbor{
			EnabledAddressFamiliesSimple: []exampleUnion{
				testutil.UnionFloat64(3.14),
				testutil.UnionInt64(42),
				testBinary,
				EnumTestVALONE,
			},
		},
		wantIETF: map[string]interface{}{
			"state": map[string]interface{}{
				"enabled-address-families-simple": []interface{}{"3.14", "42", base64testStringEncoded, "VAL_ONE"},
			},
		},
		wantInternal: map[string]interface{}{
			"state": map[string]interface{}{
				"enabled-address-families-simple": []interface{}{3.14, 42, base64testStringEncoded, "VAL_ONE"},
			},
		},
	}, {
		name: "union example - string",
		in: &exampleBgpNeighbor{
			TransportAddressSimple: testutil.UnionString("42.42.42.42"),
		},
		wantIETF: map[string]interface{}{
			"state": map[string]interface{}{
				"transport-address-simple": "42.42.42.42",
			},
		},
		wantSame: true,
	}, {
		name: "union example - enum",
		in: &exampleBgpNeighbor{
			TransportAddressSimple: EnumTestVALONE,
		},
		wantIETF: map[string]interface{}{
			"state": map[string]interface{}{
				"transport-address-simple": "VAL_ONE",
			},
		},
		wantSame: true,
	}, {
		name: "union example - binary",
		in: &exampleBgpNeighbor{
			TransportAddressSimple: testBinary,
		},
		wantIETF: map[string]interface{}{
			"state": map[string]interface{}{
				"transport-address-simple": base64testStringEncoded,
			},
		},
		wantSame: true,
	}, {
		name: "union with IETF content",
		in: &exampleBgpNeighbor{
			TransportAddressSimple: testutil.UnionFloat64(3.14),
		},
		wantIETF: map[string]interface{}{
			"state": map[string]interface{}{
				"transport-address-simple": "3.14",
			},
		},
		wantInternal: map[string]interface{}{
			"state": map[string]interface{}{
				"transport-address-simple": 3.14,
			},
		},
	}, {
		name: "union leaf-list example (wrapper union)",
		in: &exampleBgpNeighbor{
			EnabledAddressFamilies: []exampleBgpNeighborEnabledAddressFamiliesUnion{
				&exampleBgpNeighborEnabledAddressFamiliesUnionString{"IPV4"},
				&exampleBgpNeighborEnabledAddressFamiliesUnionBinary{[]byte{42}},
			},
		},
		wantIETF: map[string]interface{}{
			"state": map[string]interface{}{
				"enabled-address-families": []interface{}{"IPV4", "Kg=="},
			},
		},
		wantSame: true,
	}, {
		name: "union example (wrapper union)",
		in: &exampleBgpNeighbor{
			TransportAddress: &exampleTransportAddressString{"42.42.42.42"},
		},
		wantIETF: map[string]interface{}{
			"state": map[string]interface{}{
				"transport-address": "42.42.42.42",
			},
		},
		wantSame: true,
	}, {
		name: "union enum example (wrapper union)",
		in: &exampleBgpNeighbor{
			TransportAddress: &exampleTransportAddressEnum{EnumTestVALONE},
		},
		wantIETF: map[string]interface{}{
			"state": map[string]interface{}{
				"transport-address": "VAL_ONE",
			},
		},
		wantSame: true,
	}, {
		name: "union binary example (wrapper union)",
		in: &exampleBgpNeighbor{
			TransportAddress: &exampleTransportAddressBinary{Binary(base64testString)},
		},
		wantIETF: map[string]interface{}{
			"state": map[string]interface{}{
				"transport-address": base64testStringEncoded,
			},
		},
		wantSame: true,
	}, {
		name: "union with IETF content (wrapper union)",
		in: &exampleBgpNeighbor{
			TransportAddress: &exampleTransportAddressUint64{42},
		},
		wantIETF: map[string]interface{}{
			"state": map[string]interface{}{
				"transport-address": "42",
			},
		},
		wantInternal: map[string]interface{}{
			"state": map[string]interface{}{
				"transport-address": 42,
			},
		},
	}, {
		name: "union leaf-list with IETF content (wrapper union)",
		in: &exampleBgpNeighbor{
			EnabledAddressFamilies: []exampleBgpNeighborEnabledAddressFamiliesUnion{
				&exampleBgpNeighborEnabledAddressFamiliesUnionString{"IPV6"},
				&exampleBgpNeighborEnabledAddressFamiliesUnionUint64{42},
			},
		},
		wantIETF: map[string]interface{}{
			"state": map[string]interface{}{
				"enabled-address-families": []interface{}{"IPV6", "42"},
			},
		},
		wantInternal: map[string]interface{}{
			"state": map[string]interface{}{
				"enabled-address-families": []interface{}{"IPV6", 42},
			},
		},
	}, {
		name: "binary example",
		in: &exampleBgpNeighbor{
			MessageDump: []byte{1, 2, 3, 4},
			Updates:     []Binary{[]byte{1, 2, 3}, {1, 2, 3, 4}},
		},
		wantIETF: map[string]interface{}{
			"state": map[string]interface{}{
				"message-dump": "AQIDBA==",
				"updates":      []string{"AQID", "AQIDBA=="},
			},
		},
		wantSame: true,
	}, {
		name: "binary example 2",
		in: &exampleBgpNeighbor{
			MessageDump: Binary(base64testString),
		},
		wantIETF: map[string]interface{}{
			"state": map[string]interface{}{
				"message-dump": base64testStringEncoded,
			},
		},
		wantSame: true,
	}, {
		name: "module append example",
		in: &ietfRenderExample{
			F1: String("foo"),
			F2: String("bar"),
			F3: &ietfRenderExampleChild{
				F4: String("baz"),
				F5: String("hat"),
			},
			F6: String("mat"),
			F7: String("bat"),
		},
		inAppendMod: true,
		wantIETF: map[string]interface{}{
			"f1mod:f1": "foo",
			"f1mod:config": map[string]interface{}{
				"f2mod:f6": "mat",
			},
			"f2mod:config": map[string]interface{}{
				"f2":       "bar",
				"f3mod:f7": "bat",
			},
			"f1mod:f3": map[string]interface{}{
				"f42mod:config": map[string]interface{}{
					"f4": "baz",
				},
				"f5": "hat",
			},
		},
		wantInternal: map[string]interface{}{
			"f1": "foo",
			"config": map[string]interface{}{
				"f2": "bar",
				"f6": "mat",
				"f7": "bat",
			},
			"f3": map[string]interface{}{
				"config": map[string]interface{}{
					"f4": "baz",
				},
				"f5": "hat",
			},
		},
	}, {
		name: "list at root",
		in: &listAtRoot{
			Foo: map[string]*listAtRootChild{
				"bar": {
					Bar: String("bar"),
				},
				"baz": {
					Bar: String("baz"),
				},
			},
		},
		wantIETF: map[string]interface{}{
			"foo": []interface{}{
				map[string]interface{}{"bar": "bar"},
				map[string]interface{}{"bar": "baz"},
			},
		},
		wantInternal: map[string]interface{}{
			"foo": map[string]interface{}{
				"bar": map[string]interface{}{
					"bar": "bar",
				},
				"baz": map[string]interface{}{
					"bar": "baz",
				},
			},
		},
	}, {
		name: "list at root enum keyed",
		in: &listAtRootEnumKeyed{
			Foo: map[EnumTest]*listAtRootChildEnumKeyed{
				EnumTest(1): {
					Bar: EnumTest(1),
				},
				EnumTest(2): {
					Bar: EnumTest(2),
				},
			},
		},
		wantIETF: map[string]interface{}{
			"foo": []interface{}{
				map[string]interface{}{"bar": "VAL_ONE"},
				map[string]interface{}{"bar": "VAL_TWO"},
			},
		},
		wantInternal: map[string]interface{}{
			"foo": map[string]interface{}{
				"VAL_ONE": map[string]interface{}{
					"bar": "VAL_ONE",
				},
				"VAL_TWO": map[string]interface{}{
					"bar": "VAL_TWO",
				},
			},
		},
	}, {
		name: "list at root enum keyed with zero enum",
		in: &listAtRootEnumKeyed{
			Foo: map[EnumTest]*listAtRootChildEnumKeyed{
				EnumTest(0): {
					Bar: EnumTest(0),
				},
				EnumTest(2): {
					Bar: EnumTest(2),
				},
			},
		},
		wantErr: true,
	}, {
		name: "list at root enum keyed but invalid enum value",
		in: &listAtRootEnumKeyed{
			Foo: map[EnumTest]*listAtRootChildEnumKeyed{
				EnumTest(42): {
					Bar: EnumTest(42),
				},
				EnumTest(2): {
					Bar: EnumTest(2),
				},
			},
		},
		wantErr: true,
	}, {
		name: "annotated struct",
		in: &annotatedJSONTestStruct{
			ΛFieldThree: []Annotation{
				&testAnnotation{AnnotationFieldOne: "alexander-valley"},
			},
		},
		wantIETF: map[string]interface{}{
			"@one": []interface{}{
				map[string]interface{}{"field": "alexander-valley"},
			},
			"config": map[string]interface{}{
				"@two": []interface{}{
					map[string]interface{}{"field": "alexander-valley"},
				},
			},
		},
		wantSame: true,
	}, {
		name: "annotation with two paths",
		in: &annotatedJSONTestStruct{
			Field: String("russian-river"),
			ΛField: []Annotation{
				&testAnnotation{AnnotationFieldOne: "alexander-valley"},
			},
		},
		wantIETF: map[string]interface{}{
			"field": "russian-river",
			"@field": []interface{}{
				map[string]interface{}{"field": "alexander-valley"},
			},
		},
		wantSame: true,
	}, {
		name: "error in annotation - cannot marshal",
		in: &annotatedJSONTestStruct{
			Field: String("dry-creek"),
			ΛField: []Annotation{
				&errorAnnotation{AnnotationField: "chalk-hill"},
			},
		},
		wantErr:     true,
		wantJSONErr: true,
	}, {
		name: "error in annotation - unmarshalable",
		in: &annotatedJSONTestStruct{
			Field: String("los-carneros"),
			ΛField: []Annotation{
				&unmarshalableJSON{AnnotationField: "knights-valley"},
			},
		},
		wantErr:     true,
		wantJSONErr: true,
	}, {
		name:     "unset enum",
		in:       &renderExample{EnumField: EnumTestUNSET},
		wantIETF: map[string]interface{}{},
		wantSame: true,
	}, {
		name: "set enum in union",
		in:   &renderExample{UnionValSimple: EnumTestVALONE},
		wantIETF: map[string]interface{}{
			"union-val-simple": "VAL_ONE",
		},
		wantSame: true,
	}, {
		name:     "unset enum in union",
		in:       &renderExample{UnionValSimple: EnumTestUNSET},
		wantIETF: map[string]interface{}{},
		wantSame: true,
	}, {
		name: "set enum in union (wrapper union)",
		in:   &renderExample{UnionVal: &renderExampleUnionEnum{EnumTestVALONE}},
		wantIETF: map[string]interface{}{
			"union-val": "VAL_ONE",
		},
		wantSame: true,
	}, {
		name:     "unset enum in union (wrapper union)",
		in:       &renderExample{UnionVal: &renderExampleUnionEnum{EnumTestUNSET}},
		wantIETF: map[string]interface{}{},
		wantSame: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name+" ConstructIETFJSON", func(t *testing.T) {
			gotietf, err := ConstructIETFJSON(tt.in, &RFC7951JSONConfig{
				AppendModuleName:             tt.inAppendMod,
				PrependModuleNameIdentityref: tt.inPrependModIref,
				RewriteModuleNames:           tt.inRewriteModuleNameRules,
				PreferShadowPath:             tt.inPreferShadowPath,
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("ConstructIETFJSON(%v): got unexpected error: %v, want error %v", tt.in, err, tt.wantErr)
			}
			if err != nil {
				return
			}

			_, err = json.Marshal(gotietf)
			if (err != nil) != tt.wantJSONErr {
				t.Fatalf("json.Marshal(%v): got unexpected error: %v, want error: %v", gotietf, err, tt.wantJSONErr)
			}
			if err != nil {
				return
			}

			if diff := pretty.Compare(gotietf, tt.wantIETF); diff != "" {
				t.Errorf("ConstructIETFJSON(%v): did not get expected output, diff(-got,+want):\n%v", tt.in, diff)
			}
		})

		if tt.wantSame || tt.wantInternal != nil {
			t.Run(tt.name+" ConstructInternalJSON", func(t *testing.T) {
				gotjson, err := ConstructInternalJSON(tt.in)
				if (err != nil) != tt.wantErr {
					t.Fatalf("ConstructJSON(%v): got unexpected error: %v", tt.in, err)
				}
				if err != nil {
					return
				}

				_, err = json.Marshal(gotjson)
				if (err != nil) != tt.wantJSONErr {
					t.Fatalf("json.Marshal(%v): got unexpected error: %v, want error: %v", gotjson, err, tt.wantJSONErr)
				}
				if err != nil {
					return
				}

				wantInternal := tt.wantInternal
				if tt.wantSame == true {
					wantInternal = tt.wantIETF
				}
				if diff := pretty.Compare(gotjson, wantInternal); diff != "" {
					t.Errorf("ConstructJSON(%v): did not get expected output, diff(-got,+want):\n%v", tt.in, diff)
				}
			})
		}
	}
}

// Synthesised types for TestUnionInterfaceValue
type unionTestOne struct {
	UField uFieldInterface
}

type uFieldInterface interface {
	IsU()
}

type uFieldString struct {
	U string
}

func (*uFieldString) IsU() {}

type uFieldInt32 struct {
	I int32
}

func (uFieldInt32) IsU() {}

type uFieldE struct {
	E EnumTest
}

func (*uFieldE) IsU() {}

type uFieldInt64 int64

func (*uFieldInt64) IsU() {}

type uFieldMulti struct {
	One string
	Two string
}

func (*uFieldMulti) IsU() {}

func TestUnwrapUnionInterfaceValue(t *testing.T) {

	// This is the only unwrap test that is used by the simple union API
	// (i.e. unsupported types).
	testZero := &unionTestOne{
		UField: &testutil.UnionUnsupported{"Foo"},
	}

	testOne := &unionTestOne{
		UField: &uFieldString{"Foo"},
	}

	testTwo := struct {
		U unionTestOne
	}{
		U: unionTestOne{},
	}

	testThree := &unionTestOne{
		UField: uFieldInt32{42},
	}

	valFour := uFieldInt64(32)
	testFour := &unionTestOne{
		UField: &valFour,
	}

	testFive := &unionTestOne{
		UField: &uFieldMulti{
			One: "one",
			Two: "two",
		},
	}

	testSix := &unionTestOne{
		UField: &uFieldE{EnumTestVALONE},
	}

	testSeven := &unionTestOne{
		UField: &uFieldE{EnumTestVALTHREE},
	}

	tests := []struct {
		name        string
		in          reflect.Value
		inAppendMod bool
		want        interface{}
		wantErr     bool
	}{{
		name: "simple valid unsupported type",
		in:   reflect.ValueOf(testZero).Elem().Field(0),
		want: "Foo",
	}, {
		name: "simple valid union (wrapped union)",
		in:   reflect.ValueOf(testOne).Elem().Field(0),
		want: "Foo",
	}, {
		name:    "invalid input, non interface",
		in:      reflect.ValueOf(42),
		wantErr: true,
	}, {
		name:    "invalid input, non pointer",
		in:      reflect.ValueOf(testTwo),
		wantErr: true,
	}, {
		name:    "invalid input, non struct pointer",
		in:      reflect.ValueOf(testThree).Elem().Field(0),
		wantErr: true,
	}, {
		name:    "invalid input, non struct pointer",
		in:      reflect.ValueOf(testFour).Elem().Field(0),
		wantErr: true,
	}, {
		name:    "invalid input, two fields in struct value",
		in:      reflect.ValueOf(testFive).Elem().Field(0),
		wantErr: true,
	}, {
		name: "valid enum union",
		in:   reflect.ValueOf(testSix).Elem().Field(0),
		want: "VAL_ONE",
	}, {
		name:        "valid enum with append mod",
		in:          reflect.ValueOf(testSix).Elem().Field(0),
		inAppendMod: true,
		want:        "foo:VAL_ONE",
	}, {
		name:    "enum without a string mapping",
		in:      reflect.ValueOf(testSeven).Elem().Field(0),
		wantErr: true,
	}}

	for _, tt := range tests {
		got, err := unwrapUnionInterfaceValue(tt.in, tt.inAppendMod)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: unwrapUnionInterfaceValue(%v): got unexpected error: %v", tt.name, tt.in, err)
			}
			continue
		}

		if got != tt.want {
			t.Errorf("%s: unwrapUnionInterfaceValue(%v): did not get expected value, got: %v, want: %v", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestUnionPtrValue(t *testing.T) {
	s := "hello world"

	type twoFieldStruct struct {
		S string
		T string
	}

	type oneFieldEnum struct {
		E EnumTest
	}

	tests := []struct {
		name            string
		inValue         reflect.Value
		inAppendModName bool
		want            interface{}
		wantErr         bool
	}{{
		// This is the only test that is used by the simple union API.
		name:    "simple value ptr for unsupported type",
		inValue: reflect.ValueOf(&testutil.UnionUnsupported{"one"}),
		want:    "one",
	}, {
		name:    "simple value ptr",
		inValue: reflect.ValueOf(&renderExampleUnionString{"one"}),
		want:    "one",
	}, {
		name:    "non-ptr input",
		inValue: reflect.ValueOf(renderExampleUnionString{"world"}),
		wantErr: true,
	}, {
		name:    "ptr to a non-struct",
		inValue: reflect.ValueOf(&s),
		wantErr: true,
	}, {
		name:    "two field struct",
		inValue: reflect.ValueOf(&twoFieldStruct{S: "hello", T: "world"}),
		wantErr: true,
	}, {
		name:    "bad enum value",
		inValue: reflect.ValueOf(&oneFieldEnum{E: EnumTestVALTHREE}),
		wantErr: true,
	}}

	for _, tt := range tests {
		got, err := unionPtrValue(tt.inValue, tt.inAppendModName)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: unionPtrValue(%v, %v): did not get expected error, got: %v, want error: %v", tt.name, tt.inValue, tt.inAppendModName, err, tt.wantErr)
		}

		if !cmp.Equal(got, tt.want) {
			t.Errorf("%s: unionPtrValue(%v, %v): did not get expected value, got: %v, want: %v", tt.name, tt.inValue, tt.inAppendModName, got, tt.want)
		}
	}
}

func TestLeaflistToSlice(t *testing.T) {
	unsupported := testutil.UnionUnsupported{"Foo"}

	tests := []struct {
		name               string
		inVal              reflect.Value
		inAppendModuleName bool
		wantSlice          []interface{}
		wantErr            bool
	}{{
		name:      "string",
		inVal:     reflect.ValueOf([]string{"one", "two"}),
		wantSlice: []interface{}{"one", "two"},
	}, {
		name:      "uint8",
		inVal:     reflect.ValueOf([]uint8{1, 2}),
		wantSlice: []interface{}{uint8(1), uint8(2)},
	}, {
		name:      "uint16",
		inVal:     reflect.ValueOf([]uint16{3, 4}),
		wantSlice: []interface{}{uint16(3), uint16(4)},
	}, {
		name:      "uint32",
		inVal:     reflect.ValueOf([]uint32{5, 6}),
		wantSlice: []interface{}{uint32(5), uint32(6)},
	}, {
		name:      "uint64",
		inVal:     reflect.ValueOf([]uint64{7, 8}),
		wantSlice: []interface{}{uint64(7), uint64(8)},
	}, {
		name:      "int8",
		inVal:     reflect.ValueOf([]int8{1, 2}),
		wantSlice: []interface{}{int8(1), int8(2)},
	}, {
		name:      "int16",
		inVal:     reflect.ValueOf([]int16{3, 4}),
		wantSlice: []interface{}{int16(3), int16(4)},
	}, {
		name:      "int32",
		inVal:     reflect.ValueOf([]int32{5, 6}),
		wantSlice: []interface{}{int32(5), int32(6)},
	}, {
		name:      "int64",
		inVal:     reflect.ValueOf([]int64{7, 8}),
		wantSlice: []interface{}{int64(7), int64(8)},
	}, {
		name:      "enumerated int64",
		inVal:     reflect.ValueOf([]EnumTest{EnumTestVALONE, EnumTestVALTWO}),
		wantSlice: []interface{}{"VAL_ONE", "VAL_TWO"},
	}, {
		name:               "enumerated int64 with append",
		inVal:              reflect.ValueOf([]EnumTest{EnumTestVALTWO, EnumTestVALONE}),
		inAppendModuleName: true,
		wantSlice:          []interface{}{"bar:VAL_TWO", "foo:VAL_ONE"},
	}, {
		name:      "float32",
		inVal:     reflect.ValueOf([]float32{float32(42)}),
		wantSlice: []interface{}{float64(42)},
	}, {
		name:      "float64",
		inVal:     reflect.ValueOf([]float64{64.84}),
		wantSlice: []interface{}{float64(64.84)},
	}, {
		name:      "boolean",
		inVal:     reflect.ValueOf([]bool{true, false}),
		wantSlice: []interface{}{true, false},
	}, {
		name:      "union",
		inVal:     reflect.ValueOf([]exampleUnion{testutil.UnionString("hello"), testutil.UnionFloat64(3.14), testutil.UnionInt64(42), EnumTestVALTWO, testBinary, &unsupported}),
		wantSlice: []interface{}{"hello", float64(3.14), int64(42), "VAL_TWO", []byte(base64testString), "Foo"},
	}, {
		name:      "union (wrapped union)",
		inVal:     reflect.ValueOf([]uFieldInterface{&uFieldString{"hello"}}),
		wantSlice: []interface{}{"hello"},
	}, {
		name:      "int",
		inVal:     reflect.ValueOf([]int{1}),
		wantSlice: []interface{}{int64(1)},
	}, {
		name:      "binary",
		inVal:     reflect.ValueOf([]Binary{Binary([]byte{1, 2, 3})}),
		wantSlice: []interface{}{[]byte{1, 2, 3}},
	}, {
		name:    "invalid type",
		inVal:   reflect.ValueOf([]complex128{complex(42.42, 84.84)}),
		wantErr: true,
	}}

	for _, tt := range tests {
		got, err := leaflistToSlice(tt.inVal, tt.inAppendModuleName)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: leaflistToSlice(%v): got unexpected error: %v", tt.name, tt.inVal.Interface(), err)
		}

		if !cmp.Equal(got, tt.wantSlice) {
			t.Errorf("%s: leaflistToSlice(%v): did not get expected slice, got: %v, want: %v", tt.name, tt.inVal.Interface(), got, tt.wantSlice)
		}
	}
}

// binary2 is a different defined type but with the same underlying []byte type.
type binary2 []byte

func TestKeyValueAsString(t *testing.T) {
	unsupported := testutil.UnionUnsupported{"Foo"}

	tests := []struct {
		i                interface{}
		want             string
		wantErrSubstring string
	}{
		{
			i:    int16(42),
			want: "42",
		},
		{
			i:    uint16(42),
			want: "42",
		},
		{
			i:    int16(-42),
			want: "-42",
		},
		{
			i:    string("42"),
			want: "42",
		},
		{
			i:    true,
			want: "true",
		},
		{
			i:    false,
			want: "false",
		},
		{
			i:    Binary{'b', 'i', 'n', 'a', 'r', 'y'},
			want: "YmluYXJ5",
		},
		{
			i:    Binary{'s'},
			want: "cw==",
		},
		{
			i:    binary2{'s'},
			want: "cw==",
		},
		{
			i:                []uint16{100, 101, 102},
			wantErrSubstring: "cannot convert slice of type uint16 to a string for use in a key",
		},
		{
			i:    EnumTest(2),
			want: "VAL_TWO",
		},
		{
			i:                EnumTest(42),
			wantErrSubstring: "cannot map enumerated value as type EnumTest has unknown value 42",
		},
		{
			i:                interface{}(nil),
			wantErrSubstring: "cannot convert type invalid to a string for use in a key",
		},
		{
			i:    &renderExampleUnionString{"hello"},
			want: "hello",
		},
		{
			i:    testutil.UnionString("hello"),
			want: "hello",
		},
		{
			i:    testutil.UnionInt8(-5),
			want: "-5",
		},
		{
			i:    testutil.UnionUint64(42),
			want: "42",
		},
		{
			i:    testutil.UnionFloat64(3.14),
			want: "3.14",
		},
		{
			i:    testutil.UnionBool(true),
			want: "true",
		},
		{
			i:    testBinary,
			want: base64testStringEncoded,
		},
		{
			i:    &unsupported,
			want: "Foo",
		},
		{
			i:    testutil.YANGEmpty(false),
			want: "false",
		},
	}

	for _, tt := range tests {
		s, e := KeyValueAsString(tt.i)
		if diff := errdiff.Substring(e, tt.wantErrSubstring); diff != "" {
			t.Errorf("got %v, want %v", e, tt.wantErrSubstring)
			if e != nil {
				continue
			}
		}
		if !cmp.Equal(s, tt.want) {
			t.Errorf("got %v, want %v", s, tt.want)
		}
	}
}

func TestEncodeTypedValue(t *testing.T) {
	tests := []struct {
		name             string
		inVal            interface{}
		inEnc            gnmipb.Encoding
		want             *gnmipb.TypedValue
		wantErrSubstring string
	}{{
		name:  "simple string encoding",
		inVal: "hello",
		want:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"hello"}},
	}, {
		name:  "enumeration",
		inVal: EnumTestVALONE,
		want:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_ONE"}},
	}, {
		name:  "leaf-list of enumeration",
		inVal: []EnumTest{EnumTestVALONE},
		want: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_LeaflistVal{
			&gnmipb.ScalarArray{
				Element: []*gnmipb.TypedValue{{
					Value: &gnmipb.TypedValue_StringVal{"VAL_ONE"},
				}},
			},
		}},
	}, {
		name:  "leaf-list of string",
		inVal: []string{"one", "two"},
		want: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_LeaflistVal{
			&gnmipb.ScalarArray{
				Element: []*gnmipb.TypedValue{{
					Value: &gnmipb.TypedValue_StringVal{"one"},
				}, {
					Value: &gnmipb.TypedValue_StringVal{"two"},
				}},
			},
		}},
	}, {
		name:             "invalid enum",
		inVal:            int64(42),
		wantErrSubstring: "cannot represent field value 42 as TypedValue",
	}, {
		name:  "binary",
		inVal: Binary([]byte{0x00, 0x01}),
		want:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_BytesVal{[]byte{0x00, 0x01}}},
	}, {
		name:  "empty",
		inVal: YANGEmpty(true),
		want:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_BoolVal{true}},
	}, {
		name:  "nil scalar",
		inVal: nil,
		want:  nil,
	}, {
		name:  "leaf-list",
		inVal: []string{"one", "two"},
		want: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_LeaflistVal{
			&gnmipb.ScalarArray{
				Element: []*gnmipb.TypedValue{{
					Value: &gnmipb.TypedValue_StringVal{"one"},
				}, {
					Value: &gnmipb.TypedValue_StringVal{"two"},
				}},
			},
		}},
	}, {
		name:  "pointer val",
		inVal: string("val"),
		want:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"val"}},
	}, {
		name:  "string union encoding",
		inVal: testutil.UnionString("hello"),
		want:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"hello"}},
	}, {
		name:  "Int64 union encoding",
		inVal: testutil.UnionInt64(42),
		want:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_IntVal{42}},
	}, {
		name:  "decimal64 union encoding",
		inVal: testutil.UnionFloat64(3.14),
		want:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_FloatVal{3.14}},
	}, {
		name:  "binary union encoding",
		inVal: testBinary,
		want:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_BytesVal{[]byte(base64testString)}},
	}, {
		name:  "bool type union encoding",
		inVal: testutil.UnionBool(true),
		want:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_BoolVal{true}},
	}, {
		name:  "empty type union encoding",
		inVal: testutil.YANGEmpty(true),
		want:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_BoolVal{true}},
	}, {
		name:  "slice union encoding",
		inVal: []exampleUnion{testutil.UnionString("hello"), testutil.UnionInt64(42), testutil.UnionFloat64(3.14), testBinary, testutil.UnionBool(true), testutil.YANGEmpty(false)},
		want: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_LeaflistVal{
			&gnmipb.ScalarArray{
				Element: []*gnmipb.TypedValue{
					{Value: &gnmipb.TypedValue_StringVal{"hello"}},
					{Value: &gnmipb.TypedValue_IntVal{42}},
					{Value: &gnmipb.TypedValue_FloatVal{3.14}},
					{Value: &gnmipb.TypedValue_BytesVal{[]byte(base64testString)}},
					{Value: &gnmipb.TypedValue_BoolVal{true}},
					{Value: &gnmipb.TypedValue_BoolVal{false}}},
			}},
		},
	}, {
		name: "struct val - ietf json",
		inVal: &ietfRenderExample{
			F1: String("hello"),
		},
		inEnc: gnmipb.Encoding_JSON_IETF,
		want: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_JsonIetfVal{[]byte(`{
  "f1mod:f1": "hello"
}`)}},
	}, {
		name: "struct val - ietf json different module",
		inVal: &ietfRenderExample{
			F2: String("hello"),
		},
		inEnc: gnmipb.Encoding_JSON_IETF,
		want: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_JsonIetfVal{[]byte(`{
  "f2mod:config": {
    "f2": "hello"
  }
}`)}},
	}, {
		name: "struct val - internal json",
		inVal: &ietfRenderExample{
			F1: String("hi"),
		},
		inEnc: gnmipb.Encoding_JSON,
		want: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_JsonVal{[]byte(`{
  "f1": "hi"
}`)}},
	}, {
		name:             "unsupported encoding",
		inVal:            &ietfRenderExample{},
		inEnc:            gnmipb.Encoding_PROTO,
		wantErrSubstring: "invalid encoding",
	}, {
		name:  "nil struct",
		inVal: (*ietfRenderExample)(nil),
		inEnc: gnmipb.Encoding_JSON_IETF,
		want:  nil,
	}, {
		name:  "nil pointer",
		inVal: (*string)(nil),
		inEnc: gnmipb.Encoding_JSON_IETF,
		want:  nil,
	}, {
		name:  "int64 pointer",
		inVal: Int64(42),
		inEnc: gnmipb.Encoding_JSON_IETF,
		want:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_IntVal{42}},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EncodeTypedValue(tt.inVal, tt.inEnc)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}

			if !proto.Equal(got, tt.want) {
				t.Fatalf("did not get expected value, got: %v, want: %v", got, tt.want)
			}
		})
	}
}

func mustPathElem(s string) []*gnmipb.PathElem {
	p, err := StringToStructuredPath(s)
	if err != nil {
		panic(err)
	}
	return p.Elem
}

func TestFindUpdatedLeaves(t *testing.T) {
	tests := []struct {
		name             string
		in               GoStruct
		inParent         *gnmiPath
		wantLeaves       map[*path]interface{}
		wantErrSubstring string
	}{{
		name: "simple struct, single field",
		in: &renderExample{
			Str: String("test"),
		},
		inParent: &gnmiPath{pathElemPath: []*gnmipb.PathElem{}},
		wantLeaves: map[*path]interface{}{
			{p: &gnmiPath{
				pathElemPath: mustPathElem("str"),
			}}: String("test"),
		},
	}, {
		name: "multiple fields",
		in: &renderExample{
			Str:       String("test"),
			IntVal:    Int32(42),
			Int64Val:  Int64(84),
			EnumField: EnumTestVALONE,
			LeafList:  []string{"one"},
		},
		inParent: &gnmiPath{pathElemPath: []*gnmipb.PathElem{}},
		wantLeaves: map[*path]interface{}{
			{p: &gnmiPath{
				pathElemPath: mustPathElem("str"),
			}}: String("test"),
			{p: &gnmiPath{
				pathElemPath: mustPathElem("int-val"),
			}}: Int32(42),
			{p: &gnmiPath{
				pathElemPath: mustPathElem("int64-val"),
			}}: Int64(84),
			{p: &gnmiPath{
				pathElemPath: mustPathElem("enum"),
			}}: "VAL_ONE",
			{p: &gnmiPath{
				pathElemPath: mustPathElem("leaf-list"),
			}}: []string{"one"},
		},
	}, {
		name: "map",
		in: &renderExample{
			List: map[uint32]*renderExampleList{
				42: {Val: String("field")},
			},
		},
		inParent: &gnmiPath{pathElemPath: []*gnmipb.PathElem{}},
		wantLeaves: map[*path]interface{}{
			{p: &gnmiPath{
				pathElemPath: mustPathElem("list[val=field]/state/val"),
			}}: String("field"),
			{p: &gnmiPath{
				pathElemPath: mustPathElem("list[val=field]/val"),
			}}: String("field"),
		},
	}, {
		name: "unsupported struct slice",
		in: &renderExample{
			KeylessList: []*renderExampleList{
				{Val: String("one")},
			},
		},
		inParent:         &gnmiPath{pathElemPath: []*gnmipb.PathElem{}},
		wantErrSubstring: "keyless list cannot be output",
	}, {
		name: "union",
		in: &renderExample{
			UnionValSimple: testutil.UnionInt64(42),
		},
		inParent: &gnmiPath{pathElemPath: []*gnmipb.PathElem{}},
		wantLeaves: map[*path]interface{}{
			{p: &gnmiPath{
				pathElemPath: mustPathElem("union-val-simple"),
			}}: testutil.UnionInt64(42),
		},
	}, {
		name: "union (wrapped union)",
		in: &renderExample{
			UnionVal: &renderExampleUnionInt64{42},
		},
		inParent: &gnmiPath{pathElemPath: []*gnmipb.PathElem{}},
		wantLeaves: map[*path]interface{}{
			{p: &gnmiPath{
				pathElemPath: mustPathElem("union-val"),
			}}: &renderExampleUnionInt64{42},
		},
	}}

	// cmpopts helper for us to be able to handle comparisons of map[*path]interface{}
	// by sorting their keys.
	pathLess := func(a, b *path) bool {
		ap := a.p.isPathElemPath()
		bp := b.p.isPathElemPath()

		if ap != bp {
			return false
		}

		if ap {
			return testutil.PathLess(&gnmipb.Path{Elem: a.p.pathElemPath}, &gnmipb.Path{Elem: b.p.pathElemPath})
		}

		return testutil.PathLess(&gnmipb.Path{Element: a.p.stringSlicePath}, &gnmipb.Path{Element: b.p.stringSlicePath})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLeaves := map[*path]interface{}{}
			if err := findUpdatedLeaves(gotLeaves, tt.in, tt.inParent); err != nil {
				if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
					t.Fatalf("did not get expected error, %v", err)
				}
				return
			}
			if diff := cmp.Diff(tt.wantLeaves, gotLeaves, cmp.AllowUnexported(path{}), cmp.AllowUnexported(gnmiPath{}), cmp.Comparer(proto.Equal), cmpopts.SortMaps(pathLess)); diff != "" {
				t.Fatalf("did not get expected leaves, diff(-want,+got):\n%s", diff)
			}
		})
	}
}

func TestMarshal7951(t *testing.T) {
	tests := []struct {
		desc             string
		in               interface{}
		inArgs           []Marshal7951Arg
		want             interface{}
		wantErrSubstring string
	}{{
		desc: "simple string ptr field",
		in:   String("test-string"),
		want: `"test-string"`,
	}, {
		desc:             "scalar string - unsupported type",
		in:               "invalid-scalar-string",
		wantErrSubstring: "unexpected field type",
	}, {
		desc: "simple GoStruct",
		in: &renderExample{
			Str: String("test-string"),
		},
		want: `{"str":"test-string"}`,
	}, {
		desc: "simple GoStruct with PreferShadowPath",
		in: &renderExample{
			Str: String("test-string"),
		},
		inArgs: []Marshal7951Arg{
			&RFC7951JSONConfig{PreferShadowPath: true},
		},
		want: `{"srt":"test-string"}`,
	}, {
		desc: "map of GoStructs",
		in: map[string]*renderExample{
			"one": {Str: String("one")},
			"two": {Str: String("two")},
		},
		want: `[{"str":"one"},{"str":"two"}]`,
	}, {
		desc: "map of GoStructs with PreferShadowPath",
		in: map[string]*renderExample{
			"one": {Str: String("one")},
			"two": {Str: String("two")},
		},
		inArgs: []Marshal7951Arg{
			&RFC7951JSONConfig{PreferShadowPath: true},
		},
		want: `[{"srt":"one"},{"srt":"two"}]`,
	}, {
		desc: "map of invalid type",
		in: map[string]string{
			"one": "two",
		},
		wantErrSubstring: "invalid GoStruct",
	}, {
		desc: "map of invalid GoStruct",
		in: map[string]*invalidGoStructField{
			"one": {Value: "one"},
		},
		wantErrSubstring: "got unexpected field type",
	}, {
		desc: "slice of structs",
		in: []*renderExample{
			{Str: String("one")},
		},
		want: `[{"str":"one"}]`,
	}, {
		desc: "slice of scalars",
		in:   []string{"one", "two"},
		want: `["one","two"]`,
	}, {
		desc: "slice of annotations",
		in: []*testAnnotation{
			{
				AnnotationFieldOne: "test",
			},
		},
		want: `[{"field":"test"}]`,
	}, {
		desc: "empty annotation slice",
		in:   []*testAnnotation{},
		want: `null`,
	}, {
		desc: "empty map",
		in:   map[string]*renderExample{},
		// null as empty array is not valid, RFC7951 section 5.4 specify that the array must be an array, and JSON empty arrays are not null value
		want: `[]`,
	}, {
		desc: "nil string pointer",
		in:   (*string)(nil),
		want: `null`,
	}, {
		desc: "empty type",
		in:   &renderExample{Empty: true},
		want: `{"empty":[null]}`,
	}, {
		desc: "indentation requested",
		in: &renderExample{
			Str: String("test-string"),
		},
		inArgs: []Marshal7951Arg{
			JSONIndent("  "),
		},
		want: `{
  "str": "test-string"
}`,
	}, {
		desc: "append module names requested",
		in: &ietfRenderExample{
			F1: String("hello"),
		},
		inArgs: []Marshal7951Arg{
			&RFC7951JSONConfig{AppendModuleName: true},
		},
		want: `{"f1mod:f1":"hello"}`,
	}, {
		desc: "complex children with module name prepend request",
		in: &ietfRenderExample{
			F2:        String("bar"),
			MixedList: []interface{}{EnumTestVALONE, "test", 42},
			EnumList: map[EnumTest]*ietfRenderExampleEnumList{
				EnumTestVALONE: {Key: EnumTestVALONE},
			},
		},
		inArgs: []Marshal7951Arg{
			&RFC7951JSONConfig{AppendModuleName: true},
			JSONIndent("  "),
		},
		want: `{
  "f1mod:enum-list": [
    {
      "config": {
        "key": "foo:VAL_ONE"
      },
      "key": "foo:VAL_ONE"
    }
  ],
  "f1mod:mixed-list": [
    "foo:VAL_ONE",
    "test",
    42
  ],
  "f2mod:config": {
    "f2": "bar"
  }
}`,
	}, {
		desc: "complex children with PrependModuleNameIdentityref=true",
		in: &ietfRenderExample{
			F2:        String("bar"),
			MixedList: []interface{}{EnumTestVALONE, "test", 42},
			EnumList: map[EnumTest]*ietfRenderExampleEnumList{
				EnumTestVALONE: {Key: EnumTestVALONE},
			},
		},
		inArgs: []Marshal7951Arg{
			&RFC7951JSONConfig{PrependModuleNameIdentityref: true},
			JSONIndent("  "),
		},
		want: `{
  "config": {
    "f2": "bar"
  },
  "enum-list": [
    {
      "config": {
        "key": "foo:VAL_ONE"
      },
      "key": "foo:VAL_ONE"
    }
  ],
  "mixed-list": [
    "foo:VAL_ONE",
    "test",
    42
  ]
}`,
	}, {
		desc: "complex children with AppendModuleName=true and PrependModuleNameIdentityref=true",
		in: &ietfRenderExample{
			F2:        String("bar"),
			MixedList: []interface{}{EnumTestVALONE, "test", 42},
			EnumList: map[EnumTest]*ietfRenderExampleEnumList{
				EnumTestVALONE: {Key: EnumTestVALONE},
			},
		},
		inArgs: []Marshal7951Arg{
			&RFC7951JSONConfig{AppendModuleName: true, PrependModuleNameIdentityref: true},
			JSONIndent("  "),
		},
		want: `{
  "f1mod:enum-list": [
    {
      "config": {
        "key": "foo:VAL_ONE"
      },
      "key": "foo:VAL_ONE"
    }
  ],
  "f1mod:mixed-list": [
    "foo:VAL_ONE",
    "test",
    42
  ],
  "f2mod:config": {
    "f2": "bar"
  }
}`,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := Marshal7951(tt.in, tt.inArgs...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(string(got), tt.want); diff != "" {
				t.Fatalf("did not get expected return value, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
