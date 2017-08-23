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
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// yangTypeToProtoType takes an input resolveTypeArgs (containing a yang.YangType
// and a context node) and returns the protobuf type that it is to be represented
// by. The types that are used in the protobuf are wrapper types as described
// in the YANG to Protobuf translation specification.
//
// The type returned is a wrapper protobuf such that in proto3 an unset field
// can be distinguished from one set to the nil value.
//
// TODO(robjs): Add a link to the translation specification when published.
func (*genState) yangTypeToProtoType(args resolveTypeArgs) (mappedType, error) {
	switch args.yangType.Kind {
	case yang.Yint8, yang.Yint16, yang.Yint32, yang.Yint64:
		return mappedType{nativeType: "ywrapper.IntValue"}, nil
	case yang.Yuint8, yang.Yuint16, yang.Yuint32, yang.Yuint64:
		return mappedType{nativeType: "ywrapper.UintValue"}, nil
	case yang.Ybool, yang.Yempty:
		return mappedType{nativeType: "ywrapper.BoolValue"}, nil
	case yang.Ystring:
		return mappedType{nativeType: "ywrapper.StringValue"}, nil
	case yang.Ydecimal64:
		return mappedType{nativeType: "ywrapper.Decimal64Value"}, nil
	default:
		// TODO(robjs): Implement types that are missing within this function.
		// Missing types are:
		//  - enumeration
		//  - identityref
		//  - binary
		//  - bits
		//  - union
		// We cannot return an interface{} in protobuf, so therefore
		// we just throw an error with types that we cannot map.
		return mappedType{}, fmt.Errorf("unimplemented type: %v", args.yangType.Kind)
	}
}

// protoMsgName takes a yang.Entry and converts it to its protobuf message name,
// ensuring that the name that is returned is unique within the package that it is
// being contained within.
func (s *genState) protoMsgName(e *yang.Entry, compressPaths bool) string {
	pkg := protobufPackage(e, compressPaths)
	if _, ok := s.uniqueProtoMsgNames[pkg]; !ok {
		s.uniqueProtoMsgNames[pkg] = map[string]bool{}
	}

	n := makeNameUnique(yang.CamelCase(e.Name), s.uniqueProtoMsgNames[pkg])
	s.uniqueProtoMsgNames[pkg][n] = true

	return n
}

// protobufPackage generates a protobuf package name for a yang.Entry by taking its
// parent's path and converting it to a protobuf-safe name. i.e., an entry with
// the path /openconfig-interfaces/interfaces/interface/config/name returns
// openconfig_interfaces.interfaces.interface.config. If path compression is
// enabled then entities that would not have messages generated from them
// are omitted from the path, i.e., /openconfig-interfaces/interfaces/interface/config/name
// becomes interface (since modules, surrounding containers, and config/state containers
// are not considered with path compression enabled.
func protobufPackage(e *yang.Entry, compressPaths bool) string {
	parts := []string{}
	for p := e.Parent; p.Parent != nil; p = p.Parent {
		if compressPaths && !isOCCompressedValidElement(e) {
			// If compress paths is enabled, and this entity would not
			// have been included in the generated protobuf output, therefore
			// we also exclude it from the package name.
			continue
		}
		parts = append(parts, safeProtoFieldName(p.Name))
	}
	sort.Reverse(sort.StringSlice(parts))
	return strings.Join(parts, ".")
}
