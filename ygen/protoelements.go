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

// yangTypeToProtoType takes an input resolveTypeArgs (containing a yang.YangType
// and a context node) and returns the protobuf type that it is to be represented
// by. The types that are used in the protobuf are wrapper types as described
// in the YANG to Protobuf translation specification.
//
// The type returned is a wrapper protobuf such that in proto3 an unset field
// can be distinguished from one set to the nil value.
//
// TODO(robjs): Add a link to the translation specification when published.
func (s *genState) yangTypeToProtoType(args resolveTypeArgs, basePackageName, enumPackageName string) (mappedType, error) {
	// Handle typedef cases.
	mtype, err := s.enumeratedTypedefTypeName(args, fmt.Sprintf("%s.%s.", basePackageName, enumPackageName))
	switch {
	case mtype != nil:
		// mtype is set to non-nil when this was a valid enumeration
		// within a typedef.
		return *mtype, nil
	case err != nil:
		// err is non-nil when this was a typedef which included an
		// invalid type.
		return mappedType{}, err
	}

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
	case yang.Yleafref:
		// We look up the leafref in the schema tree to be able to
		// determine what type to map to.
		target, err := s.resolveLeafrefTarget(args.yangType.Path, args.contextEntry)
		if err != nil {
			return mappedType{}, err
		}
		return s.yangTypeToProtoType(resolveTypeArgs{yangType: target.Type, contextEntry: target}, basePackageName, enumPackageName)
	case yang.Yenum:
		// Return any enumeration simply as the leaf's CamelCase name
		// since it will be mapped to the correct name at output file to ensure
		// that there are no collisions. Enumerations are mapped to an embedded
		// enum within the message.
		if args.contextEntry == nil {
			return mappedType{}, fmt.Errorf("cannot map enumeration without context entry: %v", args)
		}
		return mappedType{nativeType: yang.CamelCase(args.contextEntry.Name)}, nil
	case yang.Yidentityref:
		if args.contextEntry == nil {
			return mappedType{}, fmt.Errorf("cannot map identityref without context entry: %v", args)
		}
		return mappedType{
			nativeType: fmt.Sprintf("%s.%s.%s", basePackageName, enumPackageName, s.resolveIdentityRefBaseType(args.contextEntry)),
		}, nil
	default:
		// TODO(robjs): Implement types that are missing within this function.
		// Missing types are:
		//  - binary
		//  - bits
		//  - union
		// We cannot return an interface{} in protobuf, so therefore
		// we just throw an error with types that we cannot map.
		return mappedType{}, fmt.Errorf("unimplemented type: %v", args.yangType.Kind)
	}
}

// yangTypeToProtoScalarType takes an input resolveTypeArgs and returns the protobuf
// in-built type that is used to represent it. It is used within list keys where the
// value cannot be nil/unset.
func (s *genState) yangTypeToProtoScalarType(args resolveTypeArgs) (mappedType, error) {
	switch args.yangType.Kind {
	case yang.Yint8, yang.Yint16, yang.Yint32, yang.Yint64:
		return mappedType{nativeType: "sint64"}, nil
	case yang.Yuint8, yang.Yuint16, yang.Yuint32, yang.Yuint64:
		return mappedType{nativeType: "uint64"}, nil
	case yang.Ybool, yang.Yempty:
		return mappedType{nativeType: "bool"}, nil
	case yang.Ystring:
		return mappedType{nativeType: "string"}, nil
	case yang.Yleafref:
		target, err := s.resolveLeafrefTarget(args.yangType.Path, args.contextEntry)
		if err != nil {
			return mappedType{}, nil
		}
		return s.yangTypeToProtoScalarType(resolveTypeArgs{yangType: target.Type, contextEntry: target})
	case yang.Yenum:
		// Return any enumeration simply as the leaf's CamelCase name
		// since it will be mapped to the correct name at output file to ensure
		// that there are no collisions. Enumerations are mapped to an embedded
		// enum within the message.
		if args.contextEntry == nil {
			return mappedType{}, fmt.Errorf("cannot map enumeration without context entry: %v", args)
		}
		return mappedType{nativeType: yang.CamelCase(args.contextEntry.Name)}, nil
	default:
		// TODO(robjs): implement missing types.
		//	- enumeration
		//	- identityref
		//	- binary
		//	- bits
		//	- union
		return mappedType{}, fmt.Errorf("unimplemented type: %s", args.yangType.Kind)
	}
}

// protoMsgName takes a yang.Entry and converts it to its protobuf message name,
// ensuring that the name that is returned is unique within the package that it is
// being contained within.
func (s *genState) protoMsgName(e *yang.Entry, compressPaths bool) string {
	// Return a cached name if one has already been computed.
	if n, ok := s.uniqueDirectoryNames[e.Path()]; ok {
		return n
	}

	pkg := s.protobufPackage(e, compressPaths)
	if _, ok := s.uniqueProtoMsgNames[pkg]; !ok {
		s.uniqueProtoMsgNames[pkg] = make(map[string]bool)
	}

	n := makeNameUnique(yang.CamelCase(e.Name), s.uniqueProtoMsgNames[pkg])
	s.uniqueProtoMsgNames[pkg][n] = true

	// Record that this was the proto message name that was used.
	s.uniqueDirectoryNames[e.Path()] = n

	return n
}

// protobufPackage generates a protobuf package name for a yang.Entry by taking its
// parent's path and converting it to a protobuf-style name. i.e., an entry with
// the path /openconfig-interfaces/interfaces/interface/config/name returns
// openconfig_interfaces.interfaces.interface.config. If path compression is
// enabled then entities that would not have messages generated from them
// are omitted from the path, i.e., /openconfig-interfaces/interfaces/interface/config/name
// becomes interface (since modules, surrounding containers, and config/state containers
// are not considered with path compression enabled.
func (s *genState) protobufPackage(e *yang.Entry, compressPaths bool) string {
	// If this entry has already had its parent's package calculated for it, then
	// simply return the already calculated name.
	if pkg, ok := s.uniqueProtoPackages[e.Parent.Path()]; ok {
		return pkg
	}

	parts := []string{}
	for p := e.Parent; p != nil; p = p.Parent {
		if compressPaths && !isOCCompressedValidElement(p) || !compressPaths && isChoiceOrCase(p) {
			// If compress paths is enabled, and this entity would not
			// have been included in the generated protobuf output, therefore
			// we also exclude it from the package name.
			continue
		}
		parts = append(parts, safeProtoFieldName(p.Name))
	}

	// Reverse the slice since we traversed from leaf back to root.
	for i := len(parts)/2 - 1; i >= 0; i-- {
		parts[i], parts[len(parts)-1-i] = parts[len(parts)-1-i], parts[i]
	}

	// Make the name unique since foo.bar.baz-bat and foo.bar.baz_bat will
	// become the same name in the safeProtoName transformation above.
	n := makeNameUnique(strings.Join(parts, "."), s.definedGlobals)
	s.definedGlobals[n] = true

	// Record the mapping between this entry's parent and the defined
	// package name that was used.
	s.uniqueProtoPackages[e.Parent.Path()] = n

	return n
}
