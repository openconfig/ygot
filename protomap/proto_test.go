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

package protomap

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/gnmi/errdiff"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	wpb "github.com/openconfig/ygot/proto/ywrapper"
	epb "github.com/openconfig/ygot/protomap/testdata/exschemapath"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygot"
)

func mustPath(p string) *gpb.Path {
	sp, err := ygot.StringToStructuredPath(p)
	if err != nil {
		panic(fmt.Sprintf("cannot parse path %s to proto, %v", p, err))
	}
	return sp
}

func TestPathsFromProtoInternal(t *testing.T) {
	tests := []struct {
		desc             string
		inMsg            proto.Message
		inBasePath       *gpb.Path
		wantPaths        map[*gpb.Path]interface{}
		wantErrSubstring string
	}{{
		desc: "simple proto with a single populated path",
		inMsg: &epb.Interface{
			Description: &wpb.StringValue{Value: "hello"},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/interfaces/interface/config/description"): "hello",
		},
	}, {
		desc: "example message - supported fields",
		inMsg: &epb.ExampleMessage{
			Bo: &wpb.BoolValue{Value: true},
			By: &wpb.BytesValue{Value: []byte{1, 2, 3, 4}},
			// De is currently unsupported, needs parsing of decimal64 values.
			In:  &wpb.IntValue{Value: 42},
			Str: &wpb.StringValue{Value: "hello"},
			Ui:  &wpb.UintValue{Value: 42},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/bool"):   true,
			mustPath("/bytes"):  []byte{1, 2, 3, 4},
			mustPath("/int"):    int64(42),
			mustPath("/string"): "hello",
			mustPath("/uint"):   uint64(42),
		},
	}, {
		desc: "child message with single field",
		inMsg: &epb.ExampleMessage{
			Ex: &epb.ExampleMessageChild{
				Str: &wpb.StringValue{Value: "hello"},
			},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/message/str"): "hello",
		},
	}, {
		desc: "oneof field - string",
		inMsg: &epb.ExampleMessage{
			OneofField: &epb.ExampleMessage_OneofOne{
				OneofOne: "hello world",
			},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/oneof"): "hello world",
		},
	}, {
		desc: "oneof field - uint64",
		inMsg: &epb.ExampleMessage{
			OneofField: &epb.ExampleMessage_OneofTwo{
				OneofTwo: uint64(42),
			},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/oneof"): uint64(42),
		},
	}, {
		desc: "decimal64 messages currently unsupported",
		inMsg: &epb.ExampleMessage{
			De: &wpb.Decimal64Value{Digits: 1234, Precision: 1},
		},
		wantErrSubstring: "unhandled type, decimal64",
	}, {
		desc: "multiple paths specified",
		inMsg: &epb.Root_InterfaceKey{
			Name: "value",
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/interfaces/interface/config/name"): "value",
			mustPath("/interfaces/interface/name"):        "value",
		},
	}, {
		desc: "invalid message with a map",
		inMsg: &epb.InvalidMessage{
			MapField: map[string]string{"hello": "world"},
		},
		wantErrSubstring: "map fields are not supported",
	}, {
		desc: "invalid message with missing annotation",
		inMsg: &epb.InvalidMessage{
			NoAnnotation: "invalid-field",
		},
		wantErrSubstring: "received field with invalid annotation",
	}, {
		desc: "list with single key",
		inMsg: &epb.ExampleMessage{
			Em: []*epb.ExampleMessageKey{{
				SingleKey: "key-one",
				Member: &epb.ExampleMessageListMember{
					Str: &wpb.StringValue{Value: "hello-world"},
				},
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/list-name[single-key=key-one]/single-key"):        "key-one",
			mustPath("/list-name[single-key=key-one]/config/single-key"): "key-one",
			mustPath("/list-name[single-key=key-one]/another-field"):     "hello-world",
		},
	}, {
		desc: "nested list",
		inMsg: &epb.ExampleMessage{
			Em: []*epb.ExampleMessageKey{{
				SingleKey: "key-one",
				Member: &epb.ExampleMessageListMember{
					Str: &wpb.StringValue{Value: "hello-world"},
					ChildList: []*epb.NestedListKey{{
						KeyOne: "key",
						Field: &epb.NestedListMember{
							Str: &wpb.StringValue{Value: "two"},
						},
					}},
				},
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/list-name[single-key=key-one]/single-key"):                      "key-one",
			mustPath("/list-name[single-key=key-one]/config/single-key"):               "key-one",
			mustPath("/list-name[single-key=key-one]/another-field"):                   "hello-world",
			mustPath("/list-name[single-key=key-one]/child-list[key-one=key]/key-one"): "key",
			mustPath("/list-name[single-key=key-one]/child-list[key-one=key]/str"):     "two",
		},
	}, {
		desc: "list with single key, multiple elements",
		inMsg: &epb.ExampleMessage{
			Em: []*epb.ExampleMessageKey{{
				SingleKey: "k1",
				Member: &epb.ExampleMessageListMember{
					Str: &wpb.StringValue{Value: "val-one"},
				},
			}, {
				SingleKey: "k2",
				Member: &epb.ExampleMessageListMember{
					Str: &wpb.StringValue{Value: "val-two"},
				},
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/list-name[single-key=k1]/single-key"):        "k1",
			mustPath("/list-name[single-key=k1]/config/single-key"): "k1",
			mustPath("/list-name[single-key=k1]/another-field"):     "val-one",
			mustPath("/list-name[single-key=k2]/single-key"):        "k2",
			mustPath("/list-name[single-key=k2]/config/single-key"): "k2",
			mustPath("/list-name[single-key=k2]/another-field"):     "val-two",
		},
	}, {
		desc: "list with single key, no value specified",
		inMsg: &epb.ExampleMessage{
			Em: []*epb.ExampleMessageKey{{}},
		},
		wantErrSubstring: "nil list member",
	}, {
		desc: "list with multiple keys",
		inMsg: &epb.ExampleMessage{
			Multi: []*epb.ExampleMessageMultiKey{{
				Index: 0,
				Name:  "zero",
				Member: &epb.MultiKeyListMember{
					Child: &wpb.StringValue{Value: "zero-child"},
				},
			}, {
				Index: 1,
				Name:  "one",
				Member: &epb.MultiKeyListMember{
					Child: &wpb.StringValue{Value: "one-child"},
				},
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/multi-list[index=0][name=zero]/index"):        uint32(0),
			mustPath("/multi-list[index=0][name=zero]/config/index"): uint32(0),
			mustPath("/multi-list[index=0][name=zero]/name"):         "zero",
			mustPath("/multi-list[index=0][name=zero]/config/name"):  "zero",
			mustPath("/multi-list[index=0][name=zero]/config/child"): "zero-child",
			mustPath("/multi-list[index=1][name=one]/index"):         uint32(1),
			mustPath("/multi-list[index=1][name=one]/config/index"):  uint32(1),
			mustPath("/multi-list[index=1][name=one]/name"):          "one",
			mustPath("/multi-list[index=1][name=one]/config/name"):   "one",
			mustPath("/multi-list[index=1][name=one]/config/child"):  "one-child",
		},
	}, {
		desc: "list with multiple paths",
		inMsg: &epb.InvalidMessage{
			Km: []*epb.ExampleMessageKey{{
				SingleKey: "test",
				Member: &epb.ExampleMessageListMember{
					Str: &wpb.StringValue{Value: "failed"},
				},
			}},
		},
		wantErrSubstring: "invalid list, does not map to 1 schema path",
	}, {
		desc: "repeated field that is not a list - unsupported",
		inMsg: &epb.InvalidMessage{
			Ke: []string{"one"},
		},
		wantErrSubstring: "invalid list, value is not a proto message",
	}, {
		desc: "list with bad key type",
		inMsg: &epb.InvalidMessage{
			Bk: []*epb.BadMessageKey{{
				BadKeyType: 1.0,
			}},
		},
		wantErrSubstring: "cannot map list key",
	}, {
		desc: "list with bad field type",
		inMsg: &epb.InvalidMessage{
			Bm: []*epb.BadMessageMember{{
				Key:     "one",
				BadType: []string{"one", "two"},
			}},
		},
		wantErrSubstring: "list field is of unexpected map or list type",
	}, {
		desc: "invalid annotated path",
		inMsg: &epb.InvalidMessage{
			InvalidAnnotatedPath: &wpb.StringValue{Value: "one"},
		},
		wantErrSubstring: "received invalid annotated path",
	}, {
		desc: "invalid key names",
		inMsg: &epb.InvalidMessage{
			BkTwo: []*epb.BadMessageKeyTwo{{
				Key: "one",
			}},
		},
		wantErrSubstring: "received list key with leaf names that do not match",
	}, {
		desc: "multiple paths for a container",
		inMsg: &epb.InvalidMessage{
			MultipleAnnotationsForContainer: &epb.InvalidMessage{},
		},
		wantErrSubstring: "invalid container, maps to >1 schema path",
	}, {
		desc: "invalid path in list key",
		inMsg: &epb.InvalidMessage{
			Bkpm: []*epb.BadKeyPathMessage{{
				Key: "hello world",
			}},
		},
		wantErrSubstring: "invalid path",
	}, {
		desc: "bad annotation in list key",
		inMsg: &epb.InvalidMessage{
			Ikpk: []*epb.InvalidKeyPathKey{{
				Key: "hello world",
			}},
		},
		wantErrSubstring: "error parsing path /one[two]",
	}, {
		desc: "leaf-list of string",
		inMsg: &epb.ExampleMessage{
			LeaflistString: []*wpb.StringValue{{
				Value: "one",
			}, {
				Value: "two",
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/leaflist-string"): []interface{}{"one", "two"},
		},
	}, {
		desc: "leaf-list of bool",
		inMsg: &epb.ExampleMessage{
			LeaflistBool: []*wpb.BoolValue{{
				Value: true,
			}, {
				Value: false,
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/leaflist-bool"): []interface{}{true, false},
		},
	}, {
		desc: "leaf-list of integer",
		inMsg: &epb.ExampleMessage{
			LeaflistInt: []*wpb.IntValue{{
				Value: 42,
			}, {
				Value: 84,
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/leaflist-int"): []interface{}{int64(42), int64(84)},
		},
	}, {
		desc: "leaf-list of unsigned integer",
		inMsg: &epb.ExampleMessage{
			LeaflistUint: []*wpb.UintValue{{
				Value: 42,
			}, {
				Value: 84,
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/leaflist-uint"): []interface{}{uint64(42), uint64(84)},
		},
	}, {
		desc: "leaf-list of bytes",
		inMsg: &epb.ExampleMessage{
			LeaflistBytes: []*wpb.BytesValue{{
				Value: []byte{42},
			}, {
				Value: []byte{84},
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/leaflist-bytes"): []interface{}{[]byte{42}, []byte{84}},
		},
	}, {
		desc: "leaf-list of decimal64",
		inMsg: &epb.ExampleMessage{
			LeaflistDecimal64: []*wpb.Decimal64Value{{
				Digits:    4242,
				Precision: 2,
			}, {
				Digits:    8484,
				Precision: 2,
			}},
		},
		wantErrSubstring: "unhandled type, decimal64",
	}, {
		desc: "leaf-list of union",
		inMsg: &epb.ExampleMessage{
			LeaflistUnion: []*epb.ExampleUnion{{
				Str: "hello",
			}, {
				Uint: 42,
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/leaflist-union"): []interface{}{"hello", uint64(42)},
		},
	}, {
		desc: "leaf-list of union where two fields are populated",
		inMsg: &epb.ExampleMessage{
			LeaflistUnion: []*epb.ExampleUnion{{
				Str:  "hello",
				Uint: 84,
			}, {
				Uint: 42,
			}},
		},
		wantErrSubstring: "multiple populated fields within union message",
	}, {
		desc: "leaf-list of union with enumeration",
		inMsg: &epb.ExampleMessage{
			LeaflistUnion: []*epb.ExampleUnion{{
				Str: "hello",
			}, {
				Enum: epb.ExampleEnum_ENUM_VALFORTYTWO,
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("/leaflist-union"): []interface{}{"hello", "VAL_FORTYTWO"},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := PathsFromProto(tt.inMsg)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}

			if diff := cmp.Diff(got, tt.wantPaths, protocmp.Transform(), cmpopts.EquateEmpty(), cmpopts.SortMaps(testutil.PathLess)); diff != "" {
				t.Fatalf("did not get expected results, diff(-got,+want):\n%s", diff)
			}
		})
	}
}

func TestProtoFromPaths(t *testing.T) {
	tests := []struct {
		desc             string
		inProto          proto.Message
		inVals           map[*gpb.Path]interface{}
		inOpt            []UnmapOpt
		wantProto        proto.Message
		wantErrSubstring string
	}{{
		desc:    "string field",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/string"): "hello",
		},
		wantProto: &epb.ExampleMessage{
			Str: &wpb.StringValue{Value: "hello"},
		},
	}, {
		desc:    "uint field",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/uint"): uint(18446744073709551615),
		},
		wantProto: &epb.ExampleMessage{
			Ui: &wpb.UintValue{Value: 18446744073709551615},
		},
	}, {
		desc:    "uint field as TypedValue",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/uint"): &gpb.TypedValue{
				Value: &gpb.TypedValue_UintVal{UintVal: 64},
			},
		},
		wantProto: &epb.ExampleMessage{
			Ui: &wpb.UintValue{Value: 64},
		},
	}, {
		desc:    "non uint value for uint",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/uint"): "invalid",
		},
		wantErrSubstring: "got non-uint value for uint field",
	}, {
		desc:    "string field as typed value",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/string"): &gpb.TypedValue{
				Value: &gpb.TypedValue_StringVal{StringVal: "hello-world"},
			},
		},
		wantProto: &epb.ExampleMessage{
			Str: &wpb.StringValue{Value: "hello-world"},
		},
	}, {
		desc:    "wrong field type",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/string"): 42,
		},
		wantErrSubstring: "got non-string value for string field",
	}, {
		desc:    "unknown field",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/unknown"): "hi!",
		},
		wantErrSubstring: "did not map path",
	}, {
		desc:    "enumeration with valid value",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/enum"): "VAL_ONE",
		},
		wantProto: &epb.ExampleMessage{
			En: epb.ExampleEnum_ENUM_VALONE,
		},
	}, {
		desc:    "enumeration with unknown value",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/enum"): "NO-EXIST",
		},
		wantErrSubstring: "got unknown value in enumeration",
	}, {
		desc:    "enumeration with unknown type",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/enum"): false,
		},
		wantErrSubstring: "got unknown type for enumeration",
	}, {
		desc:    "enumeration with typedvalue",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/enum"): &gpb.TypedValue{
				Value: &gpb.TypedValue_StringVal{
					StringVal: "VAL_FORTYTWO",
				},
			},
		},
		wantProto: &epb.ExampleMessage{
			En: epb.ExampleEnum_ENUM_VALFORTYTWO,
		},
	}, {
		desc:    "enumeration with bad typedvalue",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/enum"): &gpb.TypedValue{
				Value: &gpb.TypedValue_BoolVal{BoolVal: false},
			},
		},
		wantErrSubstring: "supplied TypedValue for enumeration must be a string",
	}, {
		desc:             "nil input",
		wantErrSubstring: "nil protobuf supplied",
	}, {
		desc:    "bytes value from typed value",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/bytes"): &gpb.TypedValue{
				Value: &gpb.TypedValue_BytesVal{BytesVal: []byte{1, 2, 3}},
			},
		},
		wantProto: &epb.ExampleMessage{
			By: &wpb.BytesValue{Value: []byte{1, 2, 3}},
		},
	}, {
		desc:    "bytes value from  value",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/bytes"): []byte{4, 5, 6},
		},
		wantProto: &epb.ExampleMessage{
			By: &wpb.BytesValue{Value: []byte{4, 5, 6}},
		},
	}, {
		desc:    "non-bytes for bytes field",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/bytes"): 42,
		},
		wantErrSubstring: "got non-byte slice value for bytes field",
	}, {
		desc:    "compressed schema",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/state/compress"): "hello-world",
		},
		wantProto: &epb.ExampleMessage{
			Compress: &wpb.StringValue{Value: "hello-world"},
		},
	}, {
		desc:    "trim prefix",
		inProto: &epb.Interface{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/interfaces/interface/config/description"): "interface-42",
		},
		inOpt: []UnmapOpt{
			ProtobufMessagePrefix(mustPath("/interfaces/interface")),
		},
		wantProto: &epb.Interface{
			Description: &wpb.StringValue{Value: "interface-42"},
		},
	}, {
		desc:    "trim prefix with valPrefix",
		inProto: &epb.Interface{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("description"): "interface-42",
		},
		inOpt: []UnmapOpt{
			ProtobufMessagePrefix(mustPath("/interfaces/interface")),
			ValuePathPrefix(mustPath("/interfaces/interface/config")),
		},
		wantProto: &epb.Interface{
			Description: &wpb.StringValue{Value: "interface-42"},
		},
	}, {
		desc:    "invalid message with unsupported field",
		inProto: &epb.InvalidMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("three"): "str",
		},
		wantErrSubstring: "map fields are not supported",
	}, {
		desc:    "missing annotation",
		inProto: &epb.InvalidAnnotationMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("three"): "str",
		},
		wantErrSubstring: "invalid annotation",
	}, {
		desc:    "invalid message with bad field type",
		inProto: &epb.BadMessageKeyTwo{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("one"): "42",
		},
		wantErrSubstring: "unknown field kind",
	}, {
		desc:    "extra paths, not ignored",
		inProto: &epb.Interface{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("config/name"):        "interface-42",
			mustPath("config/description"): "portal-to-wonderland",
		},
		inOpt: []UnmapOpt{
			ProtobufMessagePrefix(mustPath("/interfaces/interface")),
			ValuePathPrefix(mustPath("/interfaces/interface")),
		},
		wantProto: &epb.Interface{
			Description: &wpb.StringValue{Value: "interface-42"},
		},
		wantErrSubstring: `did not map path elem`,
	}, {
		desc:    "extra paths, ignored",
		inProto: &epb.Interface{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("config/name"):        "interface-42",
			mustPath("config/description"): "portal-to-wonderland",
		},
		inOpt: []UnmapOpt{
			ProtobufMessagePrefix(mustPath("/interfaces/interface")),
			IgnoreExtraPaths(),
			ValuePathPrefix(mustPath("/interfaces/interface")),
		},
		wantProto: &epb.Interface{
			Description: &wpb.StringValue{Value: "portal-to-wonderland"},
		},
	}, {
		desc:    "field that is not directly a child",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/one/two/three"): "ignored",
		},
		wantProto: &epb.ExampleMessage{},
	}, {
		desc:    "value prefix specified - schema path",
		inProto: &epb.Interface{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("description"): "interface-42",
		},
		inOpt: []UnmapOpt{
			ProtobufMessagePrefix(mustPath("/interfaces/interface")),
			ValuePathPrefix(mustPath("/interfaces/interface/config")),
		},
		wantProto: &epb.Interface{
			Description: &wpb.StringValue{Value: "interface-42"},
		},
	}, {
		desc:    "value prefix specified - data tree path",
		inProto: &epb.Interface{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("description"): "interface-42",
		},
		inOpt: []UnmapOpt{
			ProtobufMessagePrefix(mustPath("/interfaces/interface")),
			ValuePathPrefix(mustPath("/interfaces/interface[name=ethernet42]/config")),
		},
		wantProto: &epb.Interface{
			Description: &wpb.StringValue{Value: "interface-42"},
		},
	}, {
		desc:    "bad trimmed value",
		inProto: &epb.Interface{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("config/description"): "interface-84",
		},
		inOpt: []UnmapOpt{
			ProtobufMessagePrefix(mustPath("/interfaces/fish")),
		},
		wantErrSubstring: "invalid path provided, absolute paths must be used",
	}, {
		desc:    "relative paths to protobuf prefix",
		inProto: &epb.Interface{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("config/description"): "value",
		},
		inOpt: []UnmapOpt{
			ProtobufMessagePrefix(mustPath("/interfaces/interface")),
			ValuePathPrefix(mustPath("/interfaces/interface")),
		},
		wantProto: &epb.Interface{
			Description: &wpb.StringValue{Value: "value"},
		},
	}, {
		desc:    "child message",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/message/str"): "hello",
		},
		wantProto: &epb.ExampleMessage{
			Ex: &epb.ExampleMessageChild{
				Str: &wpb.StringValue{Value: "hello"},
			},
		},
	}, {
		desc:    "nested children",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/nested/one"):       "one",
			mustPath("/nested/child/one"): "one",
			mustPath("/nested/child/two"): "two",
		},
		wantProto: &epb.ExampleMessage{
			Nested: &epb.ExampleNestedMessage{
				One: &wpb.StringValue{Value: "one"},
				Child: &epb.ExampleNestedGrandchild{
					One: &wpb.StringValue{Value: "one"},
					Two: &wpb.StringValue{Value: "two"},
				},
			},
		},
	}, {
		desc:    "list item - one entry",
		inProto: &epb.Root{},
		inVals: map[*gpb.Path]any{
			mustPath("/interfaces/interface[name=eth0]/config/description"): "hello-world",
		},
		wantProto: &epb.Root{
			Interface: []*epb.Root_InterfaceKey{{
				Name: "eth0",
				Interface: &epb.Interface{
					Description: &wpb.StringValue{Value: "hello-world"},
				},
			}},
		},
	}, {
		desc:    "list item - two entries",
		inProto: &epb.Root{},
		inVals: map[*gpb.Path]any{
			mustPath("/interfaces/interface[name=eth0]/config/description"): "hello-world",
			mustPath("/interfaces/interface[name=eth1]/config/description"): "hello-mars",
		},
		wantProto: &epb.Root{
			Interface: []*epb.Root_InterfaceKey{{
				Name: "eth0",
				Interface: &epb.Interface{
					Description: &wpb.StringValue{Value: "hello-world"},
				},
			}, {
				Name: "eth1",
				Interface: &epb.Interface{
					Description: &wpb.StringValue{Value: "hello-mars"},
				},
			}},
		},
	}, {
		desc:    "nested list - one entry",
		inProto: &epb.Root{},
		inVals: map[*gpb.Path]any{
			mustPath("/interfaces/interface[name=eth0]/config/description"):                                      "int",
			mustPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=42]/config/description"): "subint",
		},
		wantProto: &epb.Root{
			Interface: []*epb.Root_InterfaceKey{{
				Name: "eth0",
				Interface: &epb.Interface{
					Description: &wpb.StringValue{Value: "int"},
					Subinterface: []*epb.Interface_SubinterfaceKey{{
						Index: 42,
						Subinterface: &epb.Subinterface{
							Description: &wpb.StringValue{Value: "subint"},
						},
					}},
				},
			}},
		},
	}, {
		desc:    "nested list - multiple entries",
		inProto: &epb.Root{},
		inVals: map[*gpb.Path]any{
			mustPath("/interfaces/interface[name=eth0]/config/description"):                                      "int",
			mustPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=42]/config/description"): "subint42",
			mustPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=84]/config/description"): "subint84",
		},
		wantProto: &epb.Root{
			Interface: []*epb.Root_InterfaceKey{{
				Name: "eth0",
				Interface: &epb.Interface{
					Description: &wpb.StringValue{Value: "int"},
					Subinterface: []*epb.Interface_SubinterfaceKey{{
						Index: 42,
						Subinterface: &epb.Subinterface{
							Description: &wpb.StringValue{Value: "subint42"},
						},
					}, {
						Index: 84,
						Subinterface: &epb.Subinterface{
							Description: &wpb.StringValue{Value: "subint84"},
						},
					}},
				},
			}},
		},
	}, {
		desc:    "single list - incorrect key specified",
		inProto: &epb.Root{},
		inVals: map[*gpb.Path]any{
			mustPath("/interfaces/interface[notkey=eth0]/config/description"): "hello-world",
		},
		wantErrSubstring: "missing key",
	}, {
		desc:    "single list - additional key specified",
		inProto: &epb.Root{},
		inVals: map[*gpb.Path]any{
			mustPath("/interfaces/interface[name=eth0][type=ETHERNET]/config/description"): "invalid",
		},
		wantErrSubstring: "received additional keys",
	}, {
		desc:    "leaf-list of non-union",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-string"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{{
							Value: &gpb.TypedValue_StringVal{StringVal: "one"},
						}, {
							Value: &gpb.TypedValue_StringVal{StringVal: "two"},
						}, {
							Value: &gpb.TypedValue_StringVal{StringVal: "three"},
						}},
					},
				},
			},
		},
		wantProto: &epb.ExampleMessage{
			LeaflistString: []*wpb.StringValue{{
				Value: "one",
			}, {
				Value: "two",
			}, {
				Value: "three",
			}},
		},
	}, {
		desc:    "leaf-list of non-union, simple values",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-string"): []string{"hello", "world"},
		},
		wantProto: &epb.ExampleMessage{
			LeaflistString: []*wpb.StringValue{{
				Value: "hello",
			}, {
				Value: "world",
			}},
		},
	}, {
		desc:    "leaf-list - wrong type for repeated string",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-string"): "fish",
		},
		wantErrSubstring: "invalid type",
	}, {
		desc:    "leaf-list - zero length typed value",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-string"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{},
					},
				},
			},
		},
		wantErrSubstring: "invalid leaf-list value",
	}, {
		desc:    "leaf-list - wrong field in typed value",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-string"): &gpb.TypedValue{
				Value: &gpb.TypedValue_StringVal{
					StringVal: "fish",
				},
			},
		},
		wantErrSubstring: "invalid leaf-list value",
	}, {
		desc:    "leaf-list - wrong type for uint",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-uint"): []string{"one", "two"},
		},
		wantErrSubstring: "invalid type",
	}, {
		desc:    "leaf-list - wrong type for uint64",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-uint"): []string{"one", "two"},
		},
		wantErrSubstring: "invalid type",
	}, {
		desc:    "leaf-list - wrong type for bool",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-bool"): []string{"one", "two"},
		},
		wantErrSubstring: "invalid type",
	}, {
		desc:    "leaf-list - wrong type for bytes",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-bytes"): []string{"one", "two"},
		},
		wantErrSubstring: "invalid type",
	}, {
		desc:    "leaf-list of uint64",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-uint"): []uint64{1, 2, 3, 4},
		},
		wantProto: &epb.ExampleMessage{
			LeaflistUint: []*wpb.UintValue{{
				Value: 1,
			}, {
				Value: 2,
			}, {
				Value: 3,
			}, {
				Value: 4,
			}},
		},
	}, {
		desc:    "leaf-list of typed value uint64",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-uint"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{{
							Value: &gpb.TypedValue_UintVal{UintVal: 1},
						}, {
							Value: &gpb.TypedValue_UintVal{UintVal: 2},
						}, {
							Value: &gpb.TypedValue_UintVal{UintVal: 3},
						}},
					},
				},
			},
		},
		wantProto: &epb.ExampleMessage{
			LeaflistUint: []*wpb.UintValue{{
				Value: 1,
			}, {
				Value: 2,
			}, {
				Value: 3,
			}},
		},
	}, {
		desc:    "leaf-list of int",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-int"): []int64{1, 2, 3, 4},
		},
		wantProto: &epb.ExampleMessage{
			LeaflistInt: []*wpb.IntValue{{
				Value: 1,
			}, {
				Value: 2,
			}, {
				Value: 3,
			}, {
				Value: 4,
			}},
		},
	}, {
		desc:    "leaf-list of typed value int",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-int"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{{
							Value: &gpb.TypedValue_IntVal{IntVal: 1},
						}, {
							Value: &gpb.TypedValue_IntVal{IntVal: 2},
						}, {
							Value: &gpb.TypedValue_IntVal{IntVal: 3},
						}},
					},
				},
			},
		},
		wantProto: &epb.ExampleMessage{
			LeaflistInt: []*wpb.IntValue{{
				Value: 1,
			}, {
				Value: 2,
			}, {
				Value: 3,
			}},
		},
	}, {
		desc:    "leaf-list - wrong type for int",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-int"): "fish",
		},
		wantErrSubstring: "invalid type",
	}, {
		desc:    "leaf-list of bool",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-bool"): []bool{true, false},
		},
		wantProto: &epb.ExampleMessage{
			LeaflistBool: []*wpb.BoolValue{{
				Value: true,
			}, {
				Value: false,
			}},
		},
	}, {
		desc:    "leaf-list of typed value bool",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-bool"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{{
							Value: &gpb.TypedValue_BoolVal{BoolVal: true},
						}, {
							Value: &gpb.TypedValue_BoolVal{BoolVal: false},
						}},
					},
				},
			},
		},
		wantProto: &epb.ExampleMessage{
			LeaflistBool: []*wpb.BoolValue{{
				Value: true,
			}, {
				Value: false,
			}},
		},
	}, {
		desc:    "leaf-list - int - wrong type typed value input",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			// int value containing bools.
			mustPath("/leaflist-int"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{{
							Value: &gpb.TypedValue_BoolVal{BoolVal: true},
						}, {
							Value: &gpb.TypedValue_BoolVal{BoolVal: false},
						}},
					},
				},
			},
		},
		wantErrSubstring: "wrong typed value",
	}, {
		desc:    "leaf-list - string - wrong type typed value input",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			// string value containing bools.
			mustPath("/leaflist-string"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{{
							Value: &gpb.TypedValue_BoolVal{BoolVal: true},
						}, {
							Value: &gpb.TypedValue_BoolVal{BoolVal: false},
						}},
					},
				},
			},
		},
		wantErrSubstring: "wrong typed value",
	}, {
		desc:    "leaf-list - uint - wrong type typed value input",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			// uint value containing bools.
			mustPath("/leaflist-uint"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{{
							Value: &gpb.TypedValue_BoolVal{BoolVal: true},
						}, {
							Value: &gpb.TypedValue_BoolVal{BoolVal: false},
						}},
					},
				},
			},
		},
		wantErrSubstring: "wrong typed value",
	}, {
		desc:    "leaf-list  - bool - wrong type typed value input",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-bool"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{{
							Value: &gpb.TypedValue_IntVal{IntVal: 1},
						}, {
							Value: &gpb.TypedValue_IntVal{IntVal: 2},
						}, {
							Value: &gpb.TypedValue_IntVal{IntVal: 3},
						}},
					},
				},
			},
		},
		wantErrSubstring: "wrong typed value",
	}, {
		desc:    "leaf-list of bytes",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-bytes"): [][]byte{{1}, {2}},
		},
		wantProto: &epb.ExampleMessage{
			LeaflistBytes: []*wpb.BytesValue{{
				Value: []byte{1},
			}, {
				Value: []byte{2},
			}},
		},
	}, {
		desc:    "leaf-list of typed value bytes",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-bytes"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{{
							Value: &gpb.TypedValue_BytesVal{BytesVal: []byte{1}},
						}, {
							Value: &gpb.TypedValue_BytesVal{BytesVal: []byte{2}},
						}},
					},
				},
			},
		},
		wantProto: &epb.ExampleMessage{
			LeaflistBytes: []*wpb.BytesValue{{
				Value: []byte{1},
			}, {
				Value: []byte{2},
			}},
		},
	}, {
		desc:    "leaf-list - wrong type for bytes",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-bytes"): "fish",
		},
		wantErrSubstring: "invalid type",
	}, {
		desc:    "leaf-list - wrong type of typed value for bytes",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-bytes"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{{
							Value: &gpb.TypedValue_IntVal{IntVal: 1},
						}, {
							Value: &gpb.TypedValue_IntVal{IntVal: 2},
						}, {
							Value: &gpb.TypedValue_IntVal{IntVal: 3},
						}},
					},
				},
			},
		},
		wantErrSubstring: "wrong typed value",
	}, {
		desc:    "leaf-list - unhandled deprecated type",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-decimal64"): []float64{0.1, 0.2},
		},
		wantErrSubstring: "unhandled leaf-list value",
	}, {
		desc:    "leaf-list - unions - enum and uint",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-union-b"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{{
							Value: &gpb.TypedValue_StringVal{StringVal: "VAL_ONE"},
						}, {
							Value: &gpb.TypedValue_UintVal{UintVal: 1},
						}},
					},
				},
			},
		},
		wantProto: &epb.ExampleMessage{
			LeaflistUnionB: []*epb.ExampleUnionUnambiguous{{
				Enum: epb.ExampleEnum_ENUM_VALONE,
			}, {
				Uint: 1,
			}},
		},
	}, {
		desc:    "leaf-list - unions - uint and string",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-union"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{{
							Value: &gpb.TypedValue_StringVal{StringVal: "hi mars!"},
						}, {
							Value: &gpb.TypedValue_UintVal{UintVal: 1},
						}},
					},
				},
			},
		},
		wantProto: &epb.ExampleMessage{
			LeaflistUnion: []*epb.ExampleUnion{{
				Str: "hi mars!",
			}, {
				Uint: 1,
			}},
		},
	}, {
		desc:    "leaf-list - unions - wrong type of input",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-union"): "fish",
		},
		wantErrSubstring: "invalid value",
	}, {
		desc:    "leaf-list - unions - slice of non-typed values",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-union"): &gpb.Notification{},
		},
		wantErrSubstring: "invalid struct type",
	}, {
		desc:    "leaf-list - unions - currently unhandled type",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-union"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{{
							Value: &gpb.TypedValue_IntVal{IntVal: 42},
						}},
					},
				},
			},
		},
		wantErrSubstring: "unhandled type",
	}, {
		desc:    "leaf-list - unions - slice input",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-union"): []any{"hello", "world", uint64(1)},
		},
		wantProto: &epb.ExampleMessage{
			LeaflistUnion: []*epb.ExampleUnion{{
				Str: "hello",
			}, {
				Str: "world",
			}, {
				Uint: 1,
			}},
		},
	}, {
		desc:    "leaf-list - unions - slice input - enum",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/leaflist-union-b"): []any{uint64(1), "VAL_ONE"},
		},
		wantProto: &epb.ExampleMessage{
			LeaflistUnionB: []*epb.ExampleUnionUnambiguous{{
				Uint: 1,
			}, {
				Enum: epb.ExampleEnum_ENUM_VALONE,
			}},
		},
	}, {
		// TODO(robjs): support unions within fields directly.
		desc:    "union",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]any{
			mustPath("/union"): "fish",
		},
		wantErrSubstring: `did not map path elem:{name:"union"}`,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ProtoFromPaths(tt.inProto, tt.inVals, tt.inOpt...)
			if err != nil {
				if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
					t.Fatalf("did not get expected error, %s", diff)
				}
				return
			}

			if diff := cmp.Diff(tt.inProto, tt.wantProto,
				protocmp.Transform(),
				protocmp.SortRepeatedFields(&epb.Root{}, "interface"),
				protocmp.SortRepeatedFields(&epb.Interface{}, "subinterface"),
			); diff != "" {
				t.Fatalf("did not get expected results, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
