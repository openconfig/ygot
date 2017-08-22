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
package ygen

import (
	"fmt"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// protoMsgField describes a field of a protobuf message.
type protoMsgField struct {
	Tag        uint32            // Tag is the field number that should be used in the protobuf message.
	Name       string            // Name is the field's name.
	Type       string            // Type is the protobuf type for the field.
	IsRepeated bool              // IsRepeated indicates whether the field is repeated.
	Extensions map[string]string // Extensions is the set of field tags that are applied to the field.
}

// protoMsg describes a protobuf message.
type protoMsg struct {
	Name     string           // Name is the name of the protobuf message to be output.
	YANGPath string           // YANGPath stores the path that the message corresponds to within the YANG schema.
	Fields   []*protoMsgField // Fields is a slice of the fields that are within the message.
}

// genProtoMsg takes an input yangDirectory which describes a container or list entry
// within the YANG schema and returns a protoMsg which can be mapped to the protobuf
// code representing it. It uses the set of messages that have been extracted and the
// current generator state to map to other messages and ensure uniqueness of names.
func genProtoMsg(msg *yangDirectory, msgs map[string]*yangDirectory, state *genState) (protoMsg, []error) {
	var errs []error

	msgDef := protoMsg{
		// msg.name is already specified to be CamelCase in the form we expect it
		// to be for the protobuf message name.
		Name:     msg.name,
		YANGPath: slicePathToString(msg.path),
	}

	definedFieldNames := map[string]bool{}

	for name, field := range msg.fields {
		fieldDef := &protoMsgField{
			Name: makeNameUnique(safeProtoFieldName(name), definedFieldNames),
		}

		t, err := protoTagForEntry(field)
		if err != nil {
			errs = append(errs, fmt.Errorf("proto: could not generate tag for field %s: %v", field.Name, err))
		}
		fieldDef.Tag = t

		switch {
		case field.IsList():
			errs = append(errs, fmt.Errorf("proto: list generation unimplemented for %s", field.Path()))
			continue
		case field.IsDir():
			msgName, ok := state.uniqueStructNames[field.Path()]
			if !ok {
				errs = append(errs, fmt.Errorf("proto: could not resolve %s into a defined struct", field.Path()))
				continue
			}
			fieldDef.Type = msgName
		default:
			// This is a YANG leaf, or leaf-list.
			protoType, err := state.yangTypeToProtoType(resolveTypeArgs{yangType: field.Type, contextEntry: field})
			if err != nil {
				errs = append(errs, err)
				continue
			}

			fieldDef.Type = protoType.nativeType

			if field.ListAttr != nil {
				fieldDef.IsRepeated = true
			}
		}

		msgDef.Fields = append(msgDef.Fields, fieldDef)
	}
	return msgDef, errs
}

// safeProtoFieldName takes an input string which represents the name of a YANG schema
// element and sanitises for use as a protobuf field name.
func safeProtoFieldName(name string) string {
	replacer := strings.NewReplacer(
		".", "_",
		"-", "_",
		"/", "_",
	)
	return replacer.Replace(name)
}

// fieldTag returns a protobuf tag value for the entry e. The tag value supplied is
// between 1 and 2^29-1. The values 19,000-19,999 are excluded as these are explicitly
// reserved for protobuf-internal use by https://developers.google.com/protocol-buffers/docs/proto3.
func protoTagForEntry(e *yang.Entry) (uint32, error) {
	// TODO(robjs): Replace this function with the final implementation
	// once concluded.
	return 1, nil
}
