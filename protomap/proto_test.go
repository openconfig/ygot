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
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := map[*gpb.Path]interface{}{}
			err := pathsFromProtoInternal(tt.inMsg, got, tt.inBasePath)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}

			// Ensure that we don't need to write out an empty map in each of the
			// cases above. This is just to make the test definition table cleaner
			// whilst allowing us to manipulate basePath in the tests.
			if len(got) == 0 {
				got = nil
			}

			if diff := cmp.Diff(got, tt.wantPaths, protocmp.Transform(), cmpopts.SortMaps(testutil.PathLess)); diff != "" {
				t.Fatalf("did not get expected results, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
