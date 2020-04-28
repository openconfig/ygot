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

package ygot

import (
	"fmt"
	"strings"

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
			// Set to scalar value by default -- we extract the value from the
			// wrapper message, or child messages if required.
			val := v.Interface()
			if fd.Kind() == protoreflect.MessageKind {
				switch t := v.Message().Interface().(type) {
				case *wpb.BoolValue:
					val = t.GetValue()
				case *wpb.BytesValue:
					val = t.GetValue()
				case *wpb.Decimal64Value:
					rangeErr = fmt.Errorf("unhandled type, decimal64")
					return false
				case *wpb.IntValue:
					val = t.GetValue()
				case *wpb.StringValue:
					val = t.GetValue()
				case *wpb.UintValue:
					val = t.GetValue()
				case proto.Message:
					rangeErr = fmt.Errorf("unknown type as field value, type: %T, value: %v", ex, ex)
					return false
				}
			}

			// Handle cases where there is >1 path specified for a field based on
			// path compression.
			for _, path := range strings.Split(ex, "|") {
				pp[path] = val
			}
		}
		return true
	})

	if rangeErr != nil {
		return nil, rangeErr
	}

	return pp, nil
}
