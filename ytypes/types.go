package ytypes

import (
	"google3/third_party/golang/goyang/pkg/yang/yang"
	"google3/third_party/golang/ygot/ygot/ygot"
)

// Schema specifies the common types that are part of a generated ygot schema, such that
// it can be referenced and handled in calling application code.
type Schema struct {
  Root       ygot.ValidatedGoStruct // Root is the ValidatedGoStruct that acts as the root for a schema, it is nil if there is no generated fakeroot.
  SchemaTree map[string]*yang.Entry // SchemaTree is the extracted schematree for the generated schema.
  Unmarshal  UnmarshalFunc          // Unmarshal is a function that can unmarshal RFC7951 JSON into the specified Root type.
}

// UnmarshalFunc defines a common signature for an RFC7951 to GoStruct unmarshalling function
type UnmarshalFunc func([]byte, ygot.GoStruct, ...UnmarshalOpt) error
