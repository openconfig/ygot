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
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/kylelemons/godebug/pretty"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestStripPrefix(t *testing.T) {
	tests := []struct {
		name     string
		inPath   []interface{}
		inPrefix []interface{}
		wantPath []interface{}
		wantErr  bool
	}{{
		name:     "simple prefix case",
		inPath:   []interface{}{"one", "two", "three"},
		inPrefix: []interface{}{"one"},
		wantPath: []interface{}{"two", "three"},
	}, {
		name:     "two element prefix",
		inPath:   []interface{}{"one", "two", "three"},
		inPrefix: []interface{}{"one", "two"},
		wantPath: []interface{}{"three"},
	}, {
		name:     "non-string case",
		inPath:   []interface{}{1, 2, 3},
		inPrefix: []interface{}{1, 2},
		wantPath: []interface{}{3},
	}, {
		name:     "invalid prefix",
		inPath:   []interface{}{"four", "five", "six"},
		inPrefix: []interface{}{"one"},
		wantErr:  true,
	}}

	for _, tt := range tests {
		got, err := stripPrefix(tt.inPath, tt.inPrefix)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: stripPrefix(%v, %v): got unexpected error: %v", tt.name, tt.inPath, tt.inPrefix, got)
			}
			continue
		}

		if !reflect.DeepEqual(got, tt.wantPath) {
			t.Errorf("%s: stripPrefix(%v, %v): did not get expected path, got: %v, want: %v", tt.name, tt.inPath, tt.inPrefix, got, tt.wantPath)
		}
	}
}

func TestInterfacePathAsgNMIPath(t *testing.T) {
	tests := []struct {
		name string
		in   []interface{}
		want *gnmipb.Path
	}{{
		name: "simple path",
		in:   []interface{}{"one", "two", "three"},
		want: &gnmipb.Path{
			Element: []string{"one", "two", "three"},
		},
	}, {
		name: "non-string path",
		in:   []interface{}{"one", 42, "fourteen thousand eight hundred and twenty three", 42.24},
		want: &gnmipb.Path{
			Element: []string{"one", "42", "fourteen thousand eight hundred and twenty three", "42.24"},
		},
	}}

	for _, tt := range tests {
		if got := interfacePathAsgNMIPath(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: interfacePathAsgNMIPath(%v): did not get correct output, got: %v, want: %v", tt.name, tt.in, got, tt.want)
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

// renderExample is used within TestTogNMINotifications as a GoStruct.
type renderExample struct {
	Str           *string                             `path:"str"`
	IntVal        *int32                              `path:"int-val"`
	EnumField     EnumTest                            `path:"enum"`
	Ch            *renderExampleChild                 `path:"ch"`
	LeafList      []string                            `path:"leaf-list"`
	MixedList     []interface{}                       `path:"mixed-list"`
	List          map[uint32]*renderExampleList       `path:"list"`
	EnumList      map[EnumTest]*renderExampleEnumList `path:"enum-list"`
	UnionVal      renderExampleUnion                  `path:"union-val"`
	UnionLeafList []renderExampleUnion                `path:"union-list"`
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
func (EnumTest) ΛMap() map[string]map[int64]string {
	return map[string]map[int64]string{
		"EnumTest": {
			1: "VAL_ONE",
			2: "VAL_TWO",
		},
	}
}

const (
	// C_TestVALONE is used to represent VAL_ONE of the /c/test
	// enumerated leaf in the schema-with-list test.
	EnumTestVALONE EnumTest = 1
	// C_TestVALTWO is used to represent VAL_TWO of the /c/test
	// enumerated leaf in the schema-with-list test.
	EnumTestVALTWO EnumTest = 2
)

func TestTogNMINotifications(t *testing.T) {
	tests := []struct {
		name        string
		inTimestamp int64
		inStruct    GoStruct
		inPrefix    []interface{}
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
			true, EnumTestVALTWO,
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
		inPrefix: []interface{}{"base"},
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
		inPrefix: []interface{}{"heart", "of", "gold"},
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
	}}

	for _, tt := range tests {
		got, err := TogNMINotifications(tt.inStruct, tt.inTimestamp, tt.inPrefix)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: TogNMINotifications(%v, %v, %v): got unexpected error: %v", tt.name, tt.inStruct, tt.inTimestamp, tt.inPrefix, err)
			}
			continue
		}

		// Avoid test flakiness by ignoring the update ordering. Required because
		// there is no order to the map of fields that are returned by the struct
		// output.
		if !notificationSetEqual(got, tt.want) {
			diff := pretty.Compare(got, tt.want)
			t.Errorf("%s: TogNMINotifications(%v, %v, %v): did not get expected Notification, diff(-got,+want):%s\n", tt.name, tt.inStruct, tt.inTimestamp, tt.inPrefix, diff)
		}
	}
}

// notificationSetEqual checks whether two slices of gNMI Notification messages are
// equal, ignoring the order of the Notifications.
func notificationSetEqual(a, b []*gnmipb.Notification) bool {
	if len(a) != len(b) {
		return false
	}

	res := map[bool]int{}
	for _, aelem := range a {
		var matched bool
		for _, belem := range b {
			if updateSetEqual(aelem.Update, belem.Update) {
				matched = true
				break
			}
		}
		res[matched]++
	}

	return res[false] != 0
}

// updateSetEqual checks whether two slices of gNMI Updates are equal, ignoring their
// order.
func updateSetEqual(a, b []*gnmipb.Update) bool {
	if len(a) != len(b) {
		return false
	}

	aSet := map[*gnmipb.Path]*gnmipb.Update{}
	for _, aelem := range a {
		aSet[aelem.Path] = aelem
	}

	for _, belem := range b {
		aelem, ok := aSet[belem.Path]
		if !ok {
			return false
		}

		if !reflect.DeepEqual(aelem, belem) {
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

func TestConstructJSON(t *testing.T) {
	tests := []struct {
		name        string
		in          GoStruct
		inAppendMod bool
		wantIETF    map[string]interface{}
		wantJSON    map[string]interface{}
		wantSame    bool
		wantErr     bool
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
		name: "simple render",
		in: &renderExample{
			Str: String("hello"),
		},
		wantIETF: map[string]interface{}{
			"str": "hello",
		},
		wantSame: true,
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
		wantJSON: map[string]interface{}{
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
		},
		wantIETF: map[string]interface{}{
			"str":        "hello",
			"leaf-list":  []string{"hello", "world"},
			"int-val":    42,
			"enum":       "VAL_TWO",
			"mixed-list": []interface{}{"42"},
		},
		wantJSON: map[string]interface{}{
			"str":        "hello",
			"leaf-list":  []string{"hello", "world"},
			"int-val":    42,
			"enum":       "VAL_TWO",
			"mixed-list": []interface{}{42},
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
		wantJSON: map[string]interface{}{
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
		wantIETF: map[string]interface{}{
			"ch": map[string]interface{}{"val": "42"},
			"enum-list": []interface{}{
				map[string]interface{}{
					"config": map[string]interface{}{
						"key": "VAL_ONE",
					},
					"key": "VAL_ONE",
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
			"mixed-list": []interface{}{"VAL_ONE", "test", uint32(42)},
		},
		wantJSON: map[string]interface{}{
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
		wantJSON: map[string]interface{}{
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
		name: "union with IETF content",
		in: &exampleBgpNeighbor{
			TransportAddress: &exampleTransportAddressUint64{42},
		},
		wantIETF: map[string]interface{}{
			"state": map[string]interface{}{
				"transport-address": "42",
			},
		},
		wantJSON: map[string]interface{}{
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
		wantJSON: map[string]interface{}{
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
		wantJSON: map[string]interface{}{
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

		wantJSON := tt.wantJSON
		if tt.wantSame == true {
			wantJSON = tt.wantIETF
		}
		if diff := pretty.Compare(gotjson, wantJSON); diff != "" {
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

	tests := []struct {
		name    string
		in      reflect.Value
		want    interface{}
		wantErr bool
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
	}}

	for _, tt := range tests {
		got, err := unionInterfaceValue(tt.in)
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
