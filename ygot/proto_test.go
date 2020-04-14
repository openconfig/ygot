package ygot

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"google.golang.org/protobuf/proto"

	wpb "github.com/openconfig/ygot/proto/ywrapper"
	epb "github.com/openconfig/ygot/ygot/testdata/exschemapath"
)

func TestPathsFromProto(t *testing.T) {
	tests := []struct {
		desc             string
		inMsg            proto.Message
		wantPaths        map[string]interface{}
		wantErrSubstring string
	}{{
		desc: "simple proto with a single populated path",
		inMsg: &epb.Interface{
			Description: &wpb.StringValue{Value: "hello"},
		},
		wantPaths: map[string]interface{}{
			"/interfaces/interface/config/description": "hello",
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
		wantPaths: map[string]interface{}{
			"/bool":   true,
			"/bytes":  []byte{1, 2, 3, 4},
			"/int":    int64(42),
			"/string": "hello",
			"/uint":   uint64(42),
		},
	}, {
		desc: "child messages currently unsupported",
		inMsg: &epb.ExampleMessage{
			Ex: &epb.ExampleMessageChild{},
		},
		wantErrSubstring: "unknown type as field value",
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
		wantPaths: map[string]interface{}{
			"/interfaces/interface/config/name": "value",
			"/interfaces/interface/name":        "value",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := pathsFromProto(tt.inMsg)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}

			if diff := cmp.Diff(got, tt.wantPaths, cmp.Comparer(proto.Equal)); diff != "" {
				t.Fatalf("did not get expected results, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
