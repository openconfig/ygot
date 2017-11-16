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

// resolveProtoTypeArgs specifies input parameters required for resolving types
// from YANG to protobuf.
// TODO(robjs): Consider embedding resolveProtoTypeArgs in this struct per
// discussion in https://github.com/openconfig/ygot/pull/57.
type resolveProtoTypeArgs struct {
	// basePackageNAme is the name of the package within which all generated packages
	// are to be generated.
	basePackageName string
	// enumPackageName is the name of the package within which global enumerated values
	// are defined (i.e., typedefs that contain enumerations, or YANG identities).
	enumPackageName string
	// scalaraTypeInSingleTypeUnion specifies whether scalar types should be used
	// when a union contains only one base type, or whether the protobuf wrapper
	// types should be used.
	scalarTypeInSingleTypeUnion bool
}

// yangTypeToProtoType takes an input resolveTypeArgs (containing a yang.YangType
// and a context node) and returns the protobuf type that it is to be represented
// by. The types that are used in the protobuf are wrapper types as described
// in the YANG to Protobuf translation specification.
//
// The type returned is a wrapper protobuf such that in proto3 an unset field
// can be distinguished from one set to the nil value.
//
// See https://github.com/openconfig/ygot/blob/master/docs/yang-to-protobuf-transformations-spec.md
// for additional details as to the transformation from YANG to Protobuf.
func (s *genState) yangTypeToProtoType(args resolveTypeArgs, pargs resolveProtoTypeArgs) (*mappedType, error) {
	// Handle typedef cases.
	mtype, err := s.enumeratedTypedefTypeName(args, fmt.Sprintf("%s.%s.", pargs.basePackageName, pargs.enumPackageName), true)
	if err != nil {
		return nil, err
	}
	if mtype != nil {
		// mtype is set to non-nil when this was a valid enumeration
		// within a typedef.
		return mtype, nil
	}

	switch args.yangType.Kind {
	case yang.Yint8, yang.Yint16, yang.Yint32, yang.Yint64:
		return &mappedType{nativeType: "ywrapper.IntValue"}, nil
	case yang.Yuint8, yang.Yuint16, yang.Yuint32, yang.Yuint64:
		return &mappedType{nativeType: "ywrapper.UintValue"}, nil
	case yang.Ybinary:
		return &mappedType{nativeType: "ywrapper.BytesValue"}, nil
	case yang.Ybool, yang.Yempty:
		return &mappedType{nativeType: "ywrapper.BoolValue"}, nil
	case yang.Ystring:
		return &mappedType{nativeType: "ywrapper.StringValue"}, nil
	case yang.Ydecimal64:
		return &mappedType{nativeType: "ywrapper.Decimal64Value"}, nil
	case yang.Yleafref:
		// We look up the leafref in the schema tree to be able to
		// determine what type to map to.
		target, err := s.resolveLeafrefTarget(args.yangType.Path, args.contextEntry)
		if err != nil {
			return nil, err
		}
		return s.yangTypeToProtoType(resolveTypeArgs{yangType: target.Type, contextEntry: target}, pargs)
	case yang.Yenum:
		// Return any enumeration simply as the leaf's CamelCase name
		// since it will be mapped to the correct name at output file to ensure
		// that there are no collisions. Enumerations are mapped to an embedded
		// enum within the message.
		if args.contextEntry == nil {
			return nil, fmt.Errorf("cannot map enumeration without context entry: %v", args)
		}
		return &mappedType{
			nativeType:        yang.CamelCase(args.contextEntry.Name),
			isEnumeratedValue: true,
		}, nil
	case yang.Yidentityref:
		// TODO(https://github.com/openconfig/ygot/issues/33) - refactor to allow
		// this call outside of the switch.
		if args.contextEntry == nil {
			return nil, fmt.Errorf("cannot map identityref without context entry: %v", args)
		}
		return &mappedType{
			nativeType:        s.protoIdentityName(pargs, args.contextEntry.Type.IdentityBase),
			isEnumeratedValue: true,
		}, nil
	case yang.Yunion:
		return s.protoUnionType(args, pargs)
	default:
		// TODO(robjs): Implement types that are missing within this function.
		// Missing types are:
		//  - binary
		//  - bits
		// We cannot return an interface{} in protobuf, so therefore
		// we just throw an error with types that we cannot map.
		return nil, fmt.Errorf("unimplemented type: %v", args.yangType.Kind)
	}
}

// yangTypeToProtoScalarType takes an input resolveTypeArgs and returns the protobuf
// in-built type that is used to represent it. It is used within list keys where the
// value cannot be nil/unset.
func (s *genState) yangTypeToProtoScalarType(args resolveTypeArgs, pargs resolveProtoTypeArgs) (*mappedType, error) {
	// Handle typedef cases.
	mtype, err := s.enumeratedTypedefTypeName(args, fmt.Sprintf("%s.%s.", pargs.basePackageName, pargs.enumPackageName), true)
	if err != nil {
		return nil, err
	}
	if mtype != nil {
		// mtype is set to non-nil when this was a valid enumeration
		// within a typedef.
		return mtype, nil
	}
	switch args.yangType.Kind {
	case yang.Yint8, yang.Yint16, yang.Yint32, yang.Yint64:
		return &mappedType{nativeType: "sint64"}, nil
	case yang.Yuint8, yang.Yuint16, yang.Yuint32, yang.Yuint64:
		return &mappedType{nativeType: "uint64"}, nil
	case yang.Ybinary:
		return &mappedType{nativeType: "bytes"}, nil
	case yang.Ybool, yang.Yempty:
		return &mappedType{nativeType: "bool"}, nil
	case yang.Ystring:
		return &mappedType{nativeType: "string"}, nil
	case yang.Ydecimal64:
		// Decimal64 continues to be a message even when we are mapping scalars
		// as there is not an equivalent Protobuf type.
		return &mappedType{nativeType: "ywrapper.Decimal64Value"}, nil
	case yang.Yleafref:
		target, err := s.resolveLeafrefTarget(args.yangType.Path, args.contextEntry)
		if err != nil {
			return nil, err
		}
		return s.yangTypeToProtoScalarType(resolveTypeArgs{yangType: target.Type, contextEntry: target}, pargs)
	case yang.Yenum:
		// Return any enumeration simply as the leaf's CamelCase name
		// since it will be mapped to the correct name at output file to ensure
		// that there are no collisions. Enumerations are mapped to an embedded
		// enum within the message.
		if args.contextEntry == nil {
			return nil, fmt.Errorf("cannot map enumeration without context entry: %v", args)
		}
		return &mappedType{
			nativeType:        yang.CamelCase(args.contextEntry.Name),
			isEnumeratedValue: true,
		}, nil
	case yang.Yidentityref:
		if args.contextEntry == nil {
			return nil, fmt.Errorf("cannot map identityref without context entry: %v", args)
		}
		return &mappedType{
			nativeType:        s.protoIdentityName(pargs, args.contextEntry.Type.IdentityBase),
			isEnumeratedValue: true,
		}, nil
	case yang.Yunion:
		return s.protoUnionType(args, pargs)
	default:
		// TODO(robjs): implement missing types.
		//	- binary
		//	- bits
		return nil, fmt.Errorf("unimplemented type in scalar generation: %s", args.yangType.Kind)
	}
}

// protoUnionType resolves the types that are included within the YangType in resolveTypeArgs into the
// scalar type that can be included in a protobuf oneof. The basePackageName and enumPackageName are used
// to determine the paths that are used for enumerated types within the YANG schema. Each union is
// resolved into a oneof that contains the scalar types, for example:
//
// leaf a {
//	type union {
//		type string;
//		type int32;
//	}
// }
//
// Is represented in the output protobuf as:
//
// oneof a {
//	string a_string = NN;
//	int32 a_int32 = NN;
// }
//
// The mappedType's unionTypes can be output through a template into the oneof.
func (s *genState) protoUnionType(args resolveTypeArgs, pargs resolveProtoTypeArgs) (*mappedType, error) {
	unionTypes := make(map[string]*yang.YangType)
	if errs := s.protoUnionSubTypes(args.yangType, args.contextEntry, unionTypes, pargs); errs != nil {
		return nil, fmt.Errorf("errors mapping element: %v", errs)
	}

	// Handle the case that there is just one protobuf type within the union.
	if len(unionTypes) == 1 {
		for st, t := range unionTypes {
			// Handle the case whereby there is an identityref and we simply
			// want to return the type that has been resolved.
			if t.Kind == yang.Yidentityref || t.Kind == yang.Yenum {
				return &mappedType{
					nativeType:        st,
					isEnumeratedValue: true,
				}, nil
			}

			var n *mappedType
			var err error
			// Resolve the type of the single type within the union according to whether
			// we want scalar types or not. This is used in contexts where there may
			// be a union that is within a key message, which never uses wrapper types
			// since the keys of a list must all be set.
			if pargs.scalarTypeInSingleTypeUnion {
				n, err = s.yangTypeToProtoScalarType(resolveTypeArgs{
					yangType:     t,
					contextEntry: args.contextEntry,
				}, pargs)
			} else {
				n, err = s.yangTypeToProtoType(resolveTypeArgs{
					yangType:     t,
					contextEntry: args.contextEntry,
				}, pargs)
			}

			if err != nil {
				return nil, fmt.Errorf("error mapping single type within a union: %v", err)
			}
			return n, nil
		}
	}

	// Rewrite the map to be the expected format for the mappedType return value,
	// we sort the keys into alphabetical order to avoid test flakes.
	keys := []string{}
	for k := range unionTypes {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	rtypes := make(map[string]int)
	for _, k := range keys {
		rtypes[k] = len(rtypes)
	}

	return &mappedType{
		unionTypes: rtypes,
	}, nil
}

// protoUnionSubTypes extracts all possible subtypes of a YANG union. It returns a map keyed by the mapped type
// along with any errors that occur. The context entry is used to map types when the leaf that the type is associated
// with is required for mapping. The currentType map is updated as an in-out argument. The basePackageName and enumPackageName
// are used to map enumerated typedefs and identityrefs to the correct type. It returns a slice of errors if they occur
// mapping subtypes.
func (s *genState) protoUnionSubTypes(subtype *yang.YangType, ctx *yang.Entry, currentTypes map[string]*yang.YangType, pargs resolveProtoTypeArgs) []error {
	var errs []error
	if isUnionType(subtype) {
		for _, st := range subtype.Type {
			errs = append(errs, s.protoUnionSubTypes(st, ctx, currentTypes, pargs)...)
		}
		return errs
	}

	var mtype *mappedType
	switch subtype.Kind {
	case yang.Yidentityref:
		// Handle the case that the context entry is not the correct entry to deal with. This occurs when the subtype is
		// an identityref.
		mtype = &mappedType{
			nativeType:        s.protoIdentityName(pargs, subtype.IdentityBase),
			isEnumeratedValue: true,
		}
	default:
		var err error
		mtype, err = s.yangTypeToProtoScalarType(resolveTypeArgs{yangType: subtype, contextEntry: ctx}, pargs)
		if err != nil {
			return append(errs, err)
		}
	}

	// Only append the type if it not one that is currently in the list. The proto oneof only has the
	// base type that is included.
	if _, ok := currentTypes[mtype.nativeType]; !ok {
		currentTypes[mtype.nativeType] = subtype
	}

	return errs
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
	if e.Node != nil && e.Node.NName() == rootElementNodeName {
		return ""
	}

	parent := e.Parent
	// In the case of path compression, then the parent of a list is the parent
	// one level up, as is the case for if there are config and state containers.
	if compressPaths && e.IsList() || compressPaths && isConfigState(e) {
		parent = e.Parent.Parent
	}

	// If this entry has already had its parent's package calculated for it, then
	// simply return the already calculated name.
	if pkg, ok := s.uniqueProtoPackages[parent.Path()]; ok {
		return pkg
	}

	parts := []string{}
	for p := parent; p != nil; p = p.Parent {
		if compressPaths && !isOCCompressedValidElement(p) || !compressPaths && isChoiceOrCase(p) {
			// If compress paths is enabled, and this entity would not
			// have been included in the generated protobuf output, therefore
			// we also exclude it from the package name.
			continue
		}
		parts = append(parts, safeProtoIdentifierName(p.Name))
	}

	// Reverse the slice since we traversed from leaf back to root.
	for i := len(parts)/2 - 1; i >= 0; i-- {
		parts[i], parts[len(parts)-1-i] = parts[len(parts)-1-i], parts[i]
	}

	// Make the name unique since foo.bar.baz-bat and foo.bar.baz_bat will
	// become the same name in the safeProtoIdentifierName transformation above.
	n := makeNameUnique(strings.Join(parts, "."), s.definedGlobals)
	s.definedGlobals[n] = true

	// Record the mapping between this entry's parent and the defined
	// package name that was used.
	s.uniqueProtoPackages[parent.Path()] = n

	return n
}

// protoIdentityName returns the name that should be used for an identityref base.
func (s *genState) protoIdentityName(pargs resolveProtoTypeArgs, i *yang.Identity) string {
	return fmt.Sprintf("%s.%s.%s", pargs.basePackageName, pargs.enumPackageName, s.identityrefBaseTypeFromIdentity(i, true))
}
