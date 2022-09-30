// Copyright 2018 Google Inc.
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

package ytypes

import (
	"errors"
	"reflect"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

// Schema specifies the common types that are part of a generated ygot schema, such that
// it can be referenced and handled in calling application code.
type Schema struct {
	Root       ygot.GoStruct          // Root is the ygot.GoStruct that acts as the root for a schema, it is nil if there is no generated fakeroot.
	SchemaTree map[string]*yang.Entry // SchemaTree is the extracted schematree for the generated schema.
	Unmarshal  UnmarshalFunc          // Unmarshal is a function that can unmarshal RFC7951 JSON into the specified Root type.
}

// IsValid determines whether all required fields of the UnmarshalIETFJSON struct
// have been populated.
func (s *Schema) IsValid() bool {
	return s.SchemaTree != nil && s.Root != nil && s.Unmarshal != nil
}

// RootSchema returns the YANG entry schema corresponding to the type of the root within
// the schema.
func (s *Schema) RootSchema() *yang.Entry {
	return s.SchemaTree[reflect.TypeOf(s.Root).Elem().Name()]
}

// Validate performs schema validation on the schema root.
func (s *Schema) Validate(vopts ...ygot.ValidationOption) error {
	if !s.IsValid() {
		return errors.New("invalid schema: not fully populated")
	}
	return ygot.ValidateGoStruct(s.Root, vopts...)
}

// UnmarshalFunc defines a common signature for an RFC7951 to ygot.GoStruct unmarshalling function
type UnmarshalFunc func([]byte, ygot.GoStruct, ...UnmarshalOpt) error
