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

import (
	"fmt"
	"reflect"
)

// PresenceContainer is an interface which can be implemented by Go structs that are
// generated to represent a YANG presence container.
type PresenceContainer interface {
	// IsPresence is a marker method that indicates that the struct
	// implements the PresenceContainer interface.
	IsPresence()
}

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

// ValidatedGoStruct is an interface implemented by all Go structs (YANG
// container or lists), *except* when the default validate_fn_name generation
// flag is overridden.
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
	// ΛBelongingModule returns the module in which the GoStruct was
	// defined per https://datatracker.ietf.org/doc/html/rfc7951#section-4.
	// If the GoStruct is the fakeroot, then the empty string will be
	// returned.
	//
	// Strictly, this value is the name of the module having the same XML
	// namespace as this node.
	// For more information on YANG's XML namespaces see
	// https://datatracker.ietf.org/doc/html/rfc7950#section-5.3
	ΛBelongingModule() string
}

// ValidateGoStruct validates a GoStruct.
func ValidateGoStruct(goStruct GoStruct, vopts ...ValidationOption) error {
	vroot, ok := goStruct.(validatedGoStruct)
	if !ok {
		return fmt.Errorf("GoStruct cannot be validated: (%T, %v)", goStruct, goStruct)
	}
	return vroot.ΛValidate(vopts...)
}

// validatedGoStruct is an interface used for validating GoStructs.
// This interface is implemented by all Go structs (YANG container or lists),
// regardless of generation flag.
type validatedGoStruct interface {
	// GoStruct ensures that the interface for a standard GoStruct
	// is embedded.
	GoStruct
	// ΛValidate compares the contents of the implementing struct against
	// the YANG schema, and returns an error if the struct's contents
	// are not valid, or nil if the struct complies with the schema.
	ΛValidate(...ValidationOption) error
}

// ValidationOption is an interface that is implemented for each struct
// which presents configuration parameters for validation options through the
// Validate public API.
type ValidationOption interface {
	IsValidationOption()
}

// GoOrderedMap is an interface which can be implemented by Go structs that are
// generated to represent a YANG "ordered-by user" list. It simply allows
// handling code to ensure that it is interacting with a struct that will meet
// the expectations of the interface - such as the existence of a Values()
// method that allows the retrieval of the list elements within the ordered
// list.
type GoOrderedMap interface {
	// IsYANGOrderedList is a marker method that indicates that the struct
	// implements the GoOrderedMap interface.
	IsYANGOrderedList()
	// Len returns the size of the ordered list.
	Len() int
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

// GoKeyStruct is an interface which can be implemented by Go key
// structs that are generated to represent a YANG multi-keyed list's key that
// has the corresponding function to retrieve the list keys as a map.
type GoKeyStruct interface {
	// IsYANGGoKeyStruct ensures that the interface for a standard
	// GoKeyStruct is embedded.
	IsYANGGoKeyStruct()
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
	// String provides the string representation of the enum, which will be
	// the YANG name if it's in its defined range.
	String() string
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
	// Value is an optionally-populated field that specifies the value of
	// an enumerated type.
	//
	// TODO: Consider removing this field and using a custom type in the
	// ygen package since only the IR generation populates this field.
	//
	// When populated, the following values are recommended:
	// For enumerations, this value is determined by goyang.
	// For identityrefs, this value is determined by the lexicographical
	// ordering of the identityref name, starting with 0 to be consistent
	// with goyang's enumeration numbering.
	Value int
}

// Annotation defines an interface that is implemented by optional metadata
// fields within a GoStruct. Annotations are stored within each struct, and
// for a struct field, for example:
//
//	type GoStructExample struct {
//	   ΛMetadata []*ygot.Annotation `path:"@"`
//	   StringField *string `path:"string-field"`
//	   ΛStringField []*ygot.Annotation `path:"@string-field"`
//	}
//
// The ΛMetadata and ΛStringField fields can be populated with a slice of
// arbitrary types implementing the Annotation interface.
//
// Each Annotation must implement the MarshalJSON and UnmarshalJSON methods,
// such that its content can be serialised and deserialised from JSON. Using
// the approach described in RFC7952 can be used to store metadata within
// RFC7951-serialised JSON.
type Annotation interface {
	// MarshalJSON is used to marshal the annotation to JSON. It ensures that
	// the json.Marshaler interface is implemented.
	MarshalJSON() ([]byte, error)
	// UnmarshalJSON is used to unmarshal JSON into the Annotation. It ensures that
	// the json.Unmarshaler interface is implemented.
	UnmarshalJSON([]byte) error
}
