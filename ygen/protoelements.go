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
