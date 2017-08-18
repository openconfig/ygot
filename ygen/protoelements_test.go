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
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
)

func TestYangTypeToProtoType(t *testing.T) {
	tests := []struct {
		name    string
		in      resolveTypeArgs
		want    mappedType
		wantErr bool
	}{{
		name: "int8",
		in:   resolveTypeArgs{yangType: &yang.YangType{Kind: yang.Yint8}},
		want: mappedType{nativeType: "ywrapper.IntValue"},
	}, {
		name: "int16",
		in:   resolveTypeArgs{yangType: &yang.YangType{Kind: yang.Yint16}},
		want: mappedType{nativeType: "ywrapper.IntValue"},
	}, {
		name: "int32",
		in:   resolveTypeArgs{yangType: &yang.YangType{Kind: yang.Yint32}},
		want: mappedType{nativeType: "ywrapper.IntValue"},
	}, {
		name: "int64",
		in:   resolveTypeArgs{yangType: &yang.YangType{Kind: yang.Yint64}},
		want: mappedType{nativeType: "ywrapper.IntValue"},
	}, {
		name: "uint8",
		in:   resolveTypeArgs{yangType: &yang.YangType{Kind: yang.Yuint8}},
		want: mappedType{nativeType: "ywrapper.UintValue"},
	}, {
		name: "uint16",
		in:   resolveTypeArgs{yangType: &yang.YangType{Kind: yang.Yuint16}},
		want: mappedType{nativeType: "ywrapper.UintValue"},
	}, {
		name: "uint32",
		in:   resolveTypeArgs{yangType: &yang.YangType{Kind: yang.Yuint32}},
		want: mappedType{nativeType: "ywrapper.UintValue"},
	}, {
		name: "uint64",
		in:   resolveTypeArgs{yangType: &yang.YangType{Kind: yang.Yuint64}},
		want: mappedType{nativeType: "ywrapper.UintValue"},
	}, {
		name: "bool",
		in:   resolveTypeArgs{yangType: &yang.YangType{Kind: yang.Ybool}},
		want: mappedType{nativeType: "ywrapper.BoolValue"},
	}, {
		name: "empty",
		in:   resolveTypeArgs{yangType: &yang.YangType{Kind: yang.Yempty}},
		want: mappedType{nativeType: "ywrapper.BoolValue"},
	}, {
		name: "string",
		in:   resolveTypeArgs{yangType: &yang.YangType{Kind: yang.Ystring}},
		want: mappedType{nativeType: "ywrapper.StringValue"},
	}, {
		name: "decimal64",
		in:   resolveTypeArgs{yangType: &yang.YangType{Kind: yang.Ydecimal64}},
		want: mappedType{nativeType: "ywrapper.Decimal64Value"},
	}}

	for _, tt := range tests {
		s := newGenState()
		got, err := s.yangTypeToProtoType(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: yangTypeToProtoType(%v): got unexpected error: %v", tt.name, tt.in, err)
			continue
		}

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("%s: yangTypeToProtoType(%v): did not get correct type, diff(-got,+want):\n%s", tt.name, tt.in, diff)
		}
	}
}
