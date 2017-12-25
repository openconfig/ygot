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

package ygot

import "reflect"

// GoStruct is an interface which can be implemented by Go structs that are
// generated to represent a YANG container or list member. It simply allows
// handling code to ensure that it is interacting with a struct that will meet
// the expectations of the interface - such as the fields being tagged with
// appropriate metadata (tags) that allow mapping of the struct into a YANG
// schematree.
type GoStruct interface {
	// IsYANGGoStruct is a marker method that indicates that the struct
	// implements the GoStruct interface.
	IsYANGGoStruct()
}

// ValidatedGoStruct is an interface which can be implemented by Go structs
// that are generated to represent a YANG container or list member that have
// the corresponding function to be validated against the a YANG schema.
type ValidatedGoStruct interface {
	// GoStruct ensures that the interface for a standard GoStruct
	// is embedded.
	GoStruct
	// Validate compares the contents of the implementing struct against
	// the YANG schema, and returns an error if the struct's contents
	// are not valid, or nil if the struct complies with the schema.
	Validate(...ValidationOption) error
	// ΛEnumTypeMap returns the set of enumerated types that are contained
	// in the generated code.
	ΛEnumTypeMap() map[string][]reflect.Type
}

// ValidationOption is an interface that is implemented for each struct
// which presents configuration parameters for validation options through the
// Validate public API.
type ValidationOption interface {
	IsValidationOption()
}

// KeyHelperGoStruct is an interface which can be implemented by Go structs
// that are generated to represent a YANG container or list member that has
// the corresponding function to retrieve the list keys as a map.
type KeyHelperGoStruct interface {
	// GoStruct ensures that the interface for a standard GoStruct
	// is embedded.
	GoStruct
	// ΛListKeyMap defines a helper method that returns a map of the
	// keys of a list element.
	ΛListKeyMap() (map[string]interface{}, error)
}

// GoEnum is an interface which can be implemented by derived types which
// represent an enumerated value within a YANG schema. This allows handling
// code that finds struct fields that implement this interface to do specific
// mapping to other types when translating to a particular schematree.
type GoEnum interface {
	// IsYANGGoEnum is a marker method that indicates that the
	// struct implements the GoEnum interface.
	IsYANGGoEnum()
	// ΛMap is a method associated with each enumeration that retrieves a
	// map of the enumeration types to values that are associated with a
	// generated code file. The ygen library generates a static map of
	// enumeration values that this method returns.
	ΛMap() map[string]map[int64]EnumDefinition
}

// EnumDefinition is used to store the details of an enumerated value. All YANG
// enumerated values (enumeration, identityref) has a Name which represents the
// string name used for the enumerated value in the YANG module (which may not
// be Go safe). Enumerated types that are derived from identity values also
// have an associated DefiningModule, such that they can be serialised to the
// correct RFC7951 JSON format (see Section 6.8 of RFC7951),
// https://tools.ietf.org/html/rfc7951#section-6.8
type EnumDefinition struct {
	// Name is the string name of the enumerated value.
	Name string
	// DefiningModule specifies the module within which the enumeration was
	// defined. Only populated for identity values.
	DefiningModule string
}
