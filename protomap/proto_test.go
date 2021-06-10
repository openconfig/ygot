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
		inPrefix         *gpb.Path
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
		desc:    "not a wrapper message",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/message"): &gpb.Path{},
		},
		wantErrSubstring: "unimplemented",
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
		desc:    "field that is not directly a child",
		inProto: &epb.ExampleMessage{},
		inVals: map[*gpb.Path]interface{}{
			mustPath("/one/two/three"): "ignored",
		},
		wantProto: &epb.ExampleMessage{},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ProtoFromPaths(tt.inProto, tt.inVals, tt.inPrefix)
			if err != nil {
				if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
					t.Fatalf("did not get expected error, %s", diff)
				}
				return
			}

			if diff := cmp.Diff(tt.inProto, tt.wantProto, protocmp.Transform()); diff != "" {
				t.Fatalf("did not get expected results, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
