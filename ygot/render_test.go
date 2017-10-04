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
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/kylelemons/godebug/pretty"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
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

		if diff := pretty.Compare(tt.inPath, tt.want); diff != "" {
			t.Errorf("%s: (gnmiPath)(%#v).AppendName(%s): did not get expected path, diff(-got,+want):\n%s", tt.name, tt.inPath, tt.inName, diff)
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
		if got := tt.inPath.Copy(); !reflect.DeepEqual(got, tt.inPath) {
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

		if err == nil && !reflect.DeepEqual(gotLast, tt.wantLastPathElem) {
			t.Errorf("%s: %v.LastPathElem(), did not get expected last element, got: %v, want: %v", tt.name, tt.inPath, gotLast, tt.wantLastPathElem)
		}

		np := tt.inPath.Copy()
		err = np.SetIndex(tt.inIndex, tt.inValue)
		if (err != nil) != tt.wantSetIndexErr {
			t.Errorf("%s: %v.SetIndex(%d, %v): did not get expected error, got: %v, wantErr: %v", tt.name, tt.inPath, tt.inIndex, tt.inValue, err, tt.wantSetIndexErr)
		}

		if err == nil && !reflect.DeepEqual(np, tt.wantPath) {
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
			t.Errorf("%s: %v.ToProto, did not get expected return value, got: %s, want: %s", tt.name, tt.inPath, proto.MarshalTextString(got), proto.MarshalTextString(tt.wantProto))
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

		if !reflect.DeepEqual(got, tt.want) {
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
}

func (*pathElemMultiKey) IsYANGGoStruct() {}

func (e *pathElemMultiKey) ΛListKeyMap() (map[string]interface{}, error) {
	if e.I == nil || e.J == nil || e.S == nil || e.E == (EnumTest)(0) || e.X == nil {
		return nil, fmt.Errorf("unset keys")
	}
	return map[string]interface{}{
		"i": *e.I,
		"j": *e.J,
		"s": *e.S,
		"e": e.E,
		"x": e.X,
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

		if diff := pretty.Compare(got, tt.wantPath); diff != "" {
			//	if !reflect.DeepEqual(got, tt.wantPath) {
			t.Errorf("%s: appendgNMIPathElemKey(%v, %v): did not get expected return path, diff(-got,+want):\n%s", tt.name, tt.inValue, tt.inPath, diff)
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
	Str           *string                             `path:"str"`
	IntVal        *int32                              `path:"int-val"`
	FloatVal      *float32                            `path:"floatval"`
	EnumField     EnumTest                            `path:"enum"`
	Ch            *renderExampleChild                 `path:"ch"`
	LeafList      []string                            `path:"leaf-list"`
	MixedList     []interface{}                       `path:"mixed-list"`
	List          map[uint32]*renderExampleList       `path:"list"`
	EnumList      map[EnumTest]*renderExampleEnumList `path:"enum-list"`
	UnionVal      renderExampleUnion                  `path:"union-val"`
	UnionLeafList []renderExampleUnion                `path:"union-list"`
	Binary        Binary                              `path:"binary"`
	KeylessList   []*renderExampleList                `path:"keyless-list"`
	InvalidMap    map[string]*invalidGoStruct         `path:"invalid-gostruct-map"`
	InvalidPtr    *invalidGoStruct                    `path:"invalid-gostruct"`
	Empty         YANGEmpty                           `path:"empty"`
}

// IsYANGGoStruct ensures that the renderExample type implements the GoStruct
// interface.
func (*renderExample) IsYANGGoStruct() {}

// renderExampleUnion is an interface that is used to represent a mixed type
// union.
type renderExampleUnion interface {
	IsRenderUnionExample()
}

type renderExampleUnionString struct {
	String string
}

func (*renderExampleUnionString) IsRenderUnionExample() {}

type renderExampleUnionInt8 struct {
	Int8 int8
}

func (*renderExampleUnionInt8) IsRenderUnionExample() {}

// renderExampleUnionInvalid is an invalid union struct.
type renderExampleUnionInvalid struct {
	String string
	Int8   int8
}

func (*renderExampleUnionInvalid) IsRenderUnionExample() {}

// renderExampleChild is a child of the renderExample struct.
type renderExampleChild struct {
	Val *uint64 `path:"val"`
}

// IsYANGGoStruct implements the GoStruct interface.
func (*renderExampleChild) IsYANGGoStruct() {}

// renderExampleList is a list entry in the renderExample struct.
type renderExampleList struct {
	Val *string `path:"val|state/val"`
}

// IsYANGGoStruct implements the GoStruct interface.
func (*renderExampleList) IsYANGGoStruct() {}

// renderExampleEnumList is a list entry that is keyed on an enum
// in renderExample.
type renderExampleEnumList struct {
	Key EnumTest `path:"config/key|key"`
}

// IsYANGGoStruct implements the GoStruct interface.
func (*renderExampleEnumList) IsYANGGoStruct() {}

// EnumTest is a synthesised derived type which is used to represent
// an enumeration in the YANG schema.
type EnumTest int64

// IsYANGEnumeration ensures that the EnumTest derived enum type implemnts
// the GoEnum interface.
func (EnumTest) IsYANGGoEnum() {}

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

// pathElemExample is an example struct used for rendering using gNMI PathElems.
type pathElemExample struct {
	List        map[string]*pathElemExampleChild                                  `path:"list"`
	StringField *string                                                           `path:"string-field"`
	MKey        map[pathElemExampleMultiKeyChildKey]*pathElemExampleMultiKeyChild `path:"m-key"`
}

// IsYANGGoStruct ensures that pathElemExample implements GoStruct.
func (*pathElemExample) IsYANGGoStruct() {}

// pathElemExampleChild is an example struct that is used as a list child struct.
type pathElemExampleChild struct {
	Val        *string `path:"val|config/val"`
	OtherField *uint8  `path:"other-field"`
}

// IsYANGGoStruct ensures that pathElemExampleChild implements GoStruct.
func (*pathElemExampleChild) IsYANGGoStruct() {}

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
func (*pathElemUnserialisable) IsYANGGoStruct() {}

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

// IsYANGGoStruct ensures that pathElemExampleMultiKeyChild implements the GoStruct
// interface.
func (*pathElemExampleMultiKeyChild) IsYANGGoStruct() {}

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

const (
	// EnumTestVALONE is used to represent VAL_ONE of the /c/test
	// enumerated leaf in the schema-with-list test.
	EnumTestVALONE EnumTest = 1
	// EnumTestVALTWO is used to represent VAL_TWO of the /c/test
	// enumerated leaf in the schema-with-list test.
	EnumTestVALTWO EnumTest = 2
	// EnumTestVALTHREE is an an enum value that does not have
	// a corresponding string mapping.
	EnumTestVALTHREE = 3
)

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
		name:        "struct with union",
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
		name:        "invalid union",
		inTimestamp: 42,
		inStruct:    &renderExample{UnionVal: &renderExampleUnionInvalid{String: "hello", Int8: 42}},
		wantErr:     true,
	}, {
		name:        "string with leaf-list of union",
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
			Ch:     &renderExampleChild{Uint64(42)},
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
		got, err := TogNMINotifications(tt.inStruct, tt.inTimestamp, tt.inConfig)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: TogNMINotifications(%v, %v, %v): got unexpected error: %v", tt.name, tt.inStruct, tt.inTimestamp, tt.inConfig, err)
			}
			continue
		}

		// Avoid test flakiness by ignoring the update ordering. Required because
		// there is no order to the map of fields that are returned by the struct
		// output.

		if !notificationSetEqual(got, tt.want) {
			diff := pretty.Compare(got, tt.want)
			t.Errorf("%s: TogNMINotifications(%v, %v): did not get expected Notification, diff(-got,+want):%s\n", tt.name, tt.inStruct, tt.inTimestamp, diff)
		}
	}
}

// notificationSetEqual checks whether two slices of gNMI Notification messages are
// equal, ignoring the order of the Notifications.
func notificationSetEqual(a, b []*gnmipb.Notification) bool {
	if len(a) != len(b) {
		return false
	}

	matchall := true
	for _, aelem := range a {
		var matched bool
		for _, belem := range b {
			if updateSetEqual(aelem.Update, belem.Update) {
				matched = true
				break
			}
		}
		if !matched {
			matchall = false
		}
	}

	return matchall
}

// updateSetEqual checks whether two slices of gNMI Updates are equal, ignoring their
// order.
func updateSetEqual(a, b []*gnmipb.Update) bool {
	if len(a) != len(b) {
		return false
	}

	for _, aelem := range a {
		var matched bool
		for _, belem := range b {
			if proto.Equal(aelem, belem) {
				matched = true
				break
			}
		}

		if !matched {
			return false
		}
	}

	return true
}

// exampleDevice and the following structs are a set of structs used for more
// complex testing in TestConstructIETFJSON
type exampleDevice struct {
	Bgp *exampleBgp `path:""`
}

func (*exampleDevice) IsYANGGoStruct() {}

type exampleBgp struct {
	Global   *exampleBgpGlobal              `path:"bgp/global"`
	Neighbor map[string]*exampleBgpNeighbor `path:"bgp/neighbors/neighbor"`
}

func (*exampleBgp) IsYANGGoStruct() {}

type exampleBgpGlobal struct {
	As       *uint32 `path:"config/as"`
	RouterID *string `path:"config/router-id"`
}

func (*exampleBgpGlobal) IsYANGGoStruct() {}

type exampleBgpNeighbor struct {
	Description            *string                                         `path:"config/description"`
	Enabled                *bool                                           `path:"config/enabled"`
	NeighborAddress        *string                                         `path:"config/neighbor-address|neighbor-address"`
	PeerAs                 *uint32                                         `path:"config/peer-as"`
	TransportAddress       exampleTransportAddress                         `path:"state/transport-address"`
	EnabledAddressFamilies []exampleBgpNeighborEnabledAddressFamiliesUnion `path:"state/enabled-address-families"`
	MessageDump            Binary                                          `path:"state/message-dump"`
	Updates                []Binary                                        `path:"state/updates"`
}

func (*exampleBgpNeighbor) IsYANGGoStruct() {}

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

type exampleBgpNeighborEnabledAddressFamiliesUnionEnum struct {
	E EnumTest
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

// invalidGoStruct explicitly does not implement the GoStruct interface.
type invalidGoStruct struct {
	Value *string
}

type invalidGoStructChild struct {
	Child *invalidGoStruct `path:"child"`
}

func (*invalidGoStructChild) IsYANGGoStruct() {}

type invalidGoStructField struct {
	// A string is not directly allowed inside a GoStruct
	Value string `path:"value"`
}

func (*invalidGoStructField) IsYANGGoStruct() {}

// invalidGoStructEntity is a GoStruct that contains invalid path data.
type invalidGoStructEntity struct {
	EmptyPath   *string `path:""`
	NoPath      *string
	InvalidEnum int64 `path:"an-enum"`
}

func (*invalidGoStructEntity) IsYANGGoStruct() {}

type invalidGoStructMapChild struct {
	InvalidField string
}

func (*invalidGoStructMapChild) IsYANGGoStruct() {}

type invalidGoStructMap struct {
	Map    map[string]*invalidGoStructMapChild `path:"foobar"`
	FooBar map[string]*invalidGoStruct         `path:"baz"`
}

func (*invalidGoStructMap) IsYANGGoStruct() {}

type structWithMultiKey struct {
	Map map[mapKey]*structMultiKeyChild `path:"foo"`
}

func (*structWithMultiKey) IsYANGGoStruct() {}

type mapKey struct {
	F1 string `path:"fOne"`
	F2 string `path:"fTwo"`
}

type structMultiKeyChild struct {
	F1 *string `path:"config/fOne|fOne"`
	F2 *string `path:"config/fTwo|fTwo"`
}

func (*structMultiKeyChild) IsYANGGoStruct() {}

type ietfRenderExample struct {
	F1 *string                 `path:"f1" module:"f1mod"`
	F2 *string                 `path:"config/f2" module:"f2mod"`
	F3 *ietfRenderExampleChild `path:"f3" module:"f1mod"`
}

func (*ietfRenderExample) IsYANGGoStruct() {}

type ietfRenderExampleChild struct {
	F4 *string `path:"config/f4" module:"f42mod"`
	F5 *string `path:"f5" module:"f1mod"`
}

func (*ietfRenderExampleChild) IsYANGGoStruct() {}

type listAtRoot struct {
	Foo map[string]*listAtRootChild `path:"foo" rootname:"foo" module:"m1"`
}

func (*listAtRoot) IsYANGGoStruct() {}

type listAtRootChild struct {
	Bar *string `path:"bar" module:"m1"`
}

func (*listAtRootChild) IsYANGGoStruct() {}

// Types to ensure correct serialisation of elements with different
// modules at the root.
type diffModAtRoot struct {
	Child *diffModAtRootChild `path:"" module:"m1"`
	Elem  *diffModAtRootElem  `path:"" module:"m1"`
}

func (*diffModAtRoot) IsYANGGoStruct() {}

type diffModAtRootChild struct {
	ValueOne   *string `path:"/foo/value-one" module:"m2"`
	ValueTwo   *string `path:"/foo/value-two" module:"m3"`
	ValueThree *string `path:"/foo/value-three" module:"m1"`
}

func (*diffModAtRootChild) IsYANGGoStruct() {}

type diffModAtRootElem struct {
	C *diffModAtRootElemTwo `path:"/baz/c" module:"m1"`
}

func (*diffModAtRootElem) IsYANGGoStruct() {}

type diffModAtRootElemTwo struct {
	Name *string `path:"name" module:"m1"`
}

func (*diffModAtRootElemTwo) IsYANGGoStruct() {}

func TestConstructJSON(t *testing.T) {
	tests := []struct {
		name         string
		in           GoStruct
		inAppendMod  bool
		wantIETF     map[string]interface{}
		wantInternal map[string]interface{}
		wantSame     bool
		wantErr      bool
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
		name: "union example",
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
		name: "union example",
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
		name: "union with IETF content",
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
		name: "union leaf-list with IETF content",
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
		name: "module append example",
		in: &ietfRenderExample{
			F1: String("foo"),
			F2: String("bar"),
			F3: &ietfRenderExampleChild{
				F4: String("baz"),
				F5: String("hat"),
			},
		},
		inAppendMod: true,
		wantIETF: map[string]interface{}{
			"f1mod:f1": "foo",
			"f2mod:config": map[string]interface{}{
				"f2": "bar",
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
	}}

	for _, tt := range tests {
		gotietf, err := ConstructIETFJSON(tt.in, &RFC7951JSONConfig{
			AppendModuleName: tt.inAppendMod,
		})
		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: ConstructIETFJSON(%v): got unexpected error: %v", tt.name, tt.in, err)
			}
			continue
		}

		if diff := pretty.Compare(gotietf, tt.wantIETF); diff != "" {
			t.Errorf("%s: ConstructIETFJSON(%v): did not get expected output, diff(-got,+want):\n%v", tt.name, tt.in, diff)
		}

		gotjson, err := ConstructInternalJSON(tt.in)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: ConstructJSON(%v): got unexpected error: %v", tt.name, tt.in, err)
			}
			continue
		}

		wantInternal := tt.wantInternal
		if tt.wantSame == true {
			wantInternal = tt.wantIETF
		}
		if diff := pretty.Compare(gotjson, wantInternal); diff != "" {
			t.Errorf("%s: ConstructJSON(%v): did not get expected output, diff(-got,+want):\n%v", tt.name, tt.in, diff)
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

func TestUnionInterfaceValue(t *testing.T) {

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
		name: "simple valid union",
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
		got, err := unionInterfaceValue(tt.in, tt.inAppendMod)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: unionInterfaceValue(%v): got unexpected error: %v", tt.name, tt.in, err)
			}
			continue
		}

		if got != tt.want {
			t.Errorf("%s: unionInterfaceValue(%v): did not get expected value, got: %v, want: %v", tt.name, tt.in, got, tt.want)
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

		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: unionPtrValue(%v, %v): did not get expected value, got: %v, want: %v", tt.name, tt.inValue, tt.inAppendModName, got, tt.want)
		}
	}
}

func TestLeaflistToSlice(t *testing.T) {
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

		if !reflect.DeepEqual(got, tt.wantSlice) {
			t.Errorf("%s: leaflistToSlice(%v): did not get expected slice, got: %v, want: %v", tt.name, tt.inVal.Interface(), got, tt.wantSlice)
		}
	}
}
