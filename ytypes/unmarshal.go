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

package ytypes

import (
	"errors"
	"fmt"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
)

// UnmarshalOpt is an interface used for any option to be supplied
// to the Unmarshal function. Types implementing it can be used to
// control the behaviour of JSON unmarshalling.
type UnmarshalOpt interface {
	IsUnmarshalOpt()
}

// ComplianceErrors contains the compliance errors encountered from an Unmarshal operation.
type ComplianceErrors struct {
	// Errors represent generic errors for now, until we make a decision on what specific types
	// of errors should be returned.
	Errors []error
}

func (c *ComplianceErrors) Error() string {
	if c == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString("Noncompliance errors:")
	if len(c.Errors) != 0 {
		for _, e := range c.Errors {
			b.WriteString("\n\t")
			b.WriteString(e.Error())
		}
	} else {
		b.WriteString(" None")
	}
	b.WriteString("\n")
	return b.String()
}

func (c *ComplianceErrors) append(errs ...error) *ComplianceErrors {
	if c == nil {
		return &ComplianceErrors{Errors: errs}
	}

	c.Errors = append(c.Errors, errs...)
	return c
}

// BestEffortUnmarshal is an unmarshal option that accumulates errors while unmarshalling,
// and continues the unmarshaling process. An unmarshal now return a ComplianceErrors struct,
// instead of a single error.
type BestEffortUnmarshal struct{}

// IsUnmarshalOpt marks BestEffortUnmarshal as a valid UnmarshalOpt.
func (*BestEffortUnmarshal) IsUnmarshalOpt() {}

// IgnoreExtraFields is an unmarshal option that controls the
// behaviour of the Unmarshal function when additional fields are
// found in the input JSON. By default, an error will be returned,
// by specifying the IgnoreExtraFields option to Unmarshal, extra
// fields will be discarded.
type IgnoreExtraFields struct{}

// IsUnmarshalOpt marks IgnoreExtraFields as a valid UnmarshalOpt.
func (*IgnoreExtraFields) IsUnmarshalOpt() {}

// IsUnmarshalOpt marks PreferShadowPath as a valid UnmarshalOpt.
// See PreferShadowPath's definition in node.go.
func (*PreferShadowPath) IsUnmarshalOpt() {}

// Unmarshal recursively unmarshals JSON data tree in value into the given
// parent, using the given schema. Any values already in the parent that are
// not present in value are preserved. If provided schema is a leaf or leaf
// list, parent must be referencing the parent GoStruct.
func Unmarshal(schema *yang.Entry, parent interface{}, value interface{}, opts ...UnmarshalOpt) error {
	return unmarshalGeneric(schema, parent, value, JSONEncoding, opts...)
}

// Encoding specifies how the value provided to UnmarshalGeneric function is encoded.
type Encoding int

const (
	// JSONEncoding indicates that provided value is JSON encoded.
	JSONEncoding Encoding = iota

	// GNMIEncoding indicates that provided value is gNMI TypedValue.
	GNMIEncoding

	// gNMIEncodingWithJSONTolerance indicates that provided value is gNMI
	// TypedValue, but it tolerates the case that the values were produced
	// from JSON and that a tolerance may be needed (e.g. positive int is
	// accepted as an uint).
	// This is made unexported because the feature is unstable and could
	// change at any point.
	gNMIEncodingWithJSONTolerance
)

// unmarshalGeneric unmarshals the provided value encoded with the given
// encoding type into the parent with the provided schema. When encoding mode
// is GNMIEncoding, the schema needs to be pointing to a leaf or leaf list
// schema.
func unmarshalGeneric(schema *yang.Entry, parent interface{}, value interface{}, enc Encoding, opts ...UnmarshalOpt) error {
	util.Indent()
	defer util.Dedent()

	if schema == nil {
		return fmt.Errorf("nil schema for parent type %T, value %v (%T)", parent, value, value)
	}
	util.DbgPrint("Unmarshal value %v, type %T, into parent type %T, schema name %s", util.ValueStrDebug(value), value, parent, schema.Name)

	if enc == GNMIEncoding && !(schema.IsLeaf() || schema.IsLeafList()) {
		return errors.New("unmarshalling a non leaf node isn't supported in GNMIEncoding mode")
	}

	if hasBestEffortUnmarshal(opts) {
		return errors.New("unmarshalGeneric passed unsupported option BestEffortUnmarshal")
	}

	switch {
	case schema.IsLeaf():
		return unmarshalLeaf(schema, parent, value, enc, opts...)
	case schema.IsLeafList():
		return unmarshalLeafList(schema, parent, value, enc, opts...)
	case schema.IsList():
		return unmarshalList(schema, parent, value, enc, opts...)
	case schema.IsChoice():
		return fmt.Errorf("cannot pass choice schema %s to Unmarshal", schema.Name)
	case schema.IsContainer():
		return unmarshalContainer(schema, parent, value, enc, opts...)
	}
	return fmt.Errorf("unknown schema type for type %T, value %v", value, value)
}

// hasIgnoreExtraFields determines whether the supplied slice of UnmarshalOpts contains
// the IgnoreExtraFields option.
func hasIgnoreExtraFields(opts []UnmarshalOpt) bool {
	for _, o := range opts {
		if _, ok := o.(*IgnoreExtraFields); ok {
			return true
		}
	}
	return false
}

// hasPreferShadowPath determines whether the supplied slice of UnmarshalOpts
// contains the PreferShadowPath option.
func hasPreferShadowPath(opts []UnmarshalOpt) bool {
	for _, o := range opts {
		if _, ok := o.(*PreferShadowPath); ok {
			return true
		}
	}
	return false
}

// hasBestEffortUnmarshal determines whether the supplied slice of UnmarshalOpts
// contains the BestEffortUnmarshal option.
func hasBestEffortUnmarshal(opts []UnmarshalOpt) bool {
	for _, o := range opts {
		if _, ok := o.(*BestEffortUnmarshal); ok {
			return true
		}
	}

	return false
}
