package ygot

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	yextpb "github.com/openconfig/ygot/proto/yext"
	wpb "github.com/openconfig/ygot/proto/ywrapper"
)

// pathsFromProto returns, from a populated proto, a map between the YANG schema
// path (as specified in the yext.schemapath extension) and the value populated in
// the message.
func pathsFromProto(p proto.Message) (map[string]interface{}, error) {
	m := p.ProtoReflect()
	pp := map[string]interface{}{}
	var rangeErr error
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		po := fd.Options().(*descriptorpb.FieldOptions)
		if ex := proto.GetExtension(po, yextpb.E_Schemapath).(string); ex != "" {
			switch t := v.Message().Interface().(type) {
			case *wpb.BoolValue:
				pp[ex] = t.GetValue()
			case *wpb.BytesValue:
				pp[ex] = t.GetValue()
			case *wpb.Decimal64Value:
				rangeErr = fmt.Errorf("unhandled type, decimal64")
				return false
			case *wpb.IntValue:
				pp[ex] = t.GetValue()
			case *wpb.StringValue:
				pp[ex] = t.GetValue()
			case *wpb.UintValue:
				pp[ex] = t.GetValue()
			case proto.Message:
				rangeErr = fmt.Errorf("unknown type as field value, type: %T, value: %v", ex, ex)
				return false
			}
		}
		return true
	})

	if rangeErr != nil {
		return nil, rangeErr
	}

	return pp, nil
}
