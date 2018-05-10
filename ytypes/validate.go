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
	"fmt"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
)

// LeafrefOptions controls the behaviour of validation functions for leaf-ref
// data types.
type LeafrefOptions struct {
	// IgnoreMissingData determines whether leafrefs that target a node
	// that does not exist should return an error to the calling application. When
	// set to true, no error is returned.
	//
	// This functionality is typically used where a partial set of schema information
	// is populated, but validation is required - for example, configuration for
	// a protocol within OpenConfig references an interface, but the schema being
	// validated does not contain the interface definitions.
	IgnoreMissingData bool
	// Log specifies whether log entries should be created where a leafref
	// cannot be successfully resolved.
	Log bool
}

// IsValidationOption ensures that LeafrefOptions implements the ValidationOption
// interface.
func (*LeafrefOptions) IsValidationOption() {}

// Validate recursively validates the value of the given data tree struct
// against the given schema.
func Validate(schema *yang.Entry, value interface{}, opts ...ygot.ValidationOption) util.Errors {
	// Nil value means the field is unset.
	if util.IsValueNil(value) {
		return nil
	}
	if schema == nil {
		return util.NewErrs(fmt.Errorf("nil schema for type %T, value %v", value, value))
	}

	// TODO(robjs): Consider making this function a utility function when
	// additional validation options are added here. Note that this code
	// currently will accept multiple of the same option being specified,
	// and overwrite with the last within the options slice, rather than
	// explicitly returning an error.
	var leafrefOpt *LeafrefOptions
	for _, o := range opts {
		switch o.(type) {
		case *LeafrefOptions:
			leafrefOpt = o.(*LeafrefOptions)
		}
	}

	var errs util.Errors
	if util.IsFakeRoot(schema) {
		// Leafref validation traverses entire tree from the root. Do this only
		// once from the fakeroot.
		errs = ValidateLeafRefData(schema, value, leafrefOpt)
	}

	util.DbgPrint("Validate with value %v, type %T, schema name %s", util.ValueStr(value), value, schema.Name)

	switch {
	case schema.IsLeaf():
		return util.AppendErrs(errs, validateLeaf(schema, value))
	case schema.IsContainer():
		gsv, ok := value.(ygot.GoStruct)
		if !ok {
			return util.AppendErr(errs, fmt.Errorf("type %T is not a GoStruct for schema %s", value, schema.Name))
		}
		return util.AppendErrs(errs, validateContainer(schema, gsv))
	case schema.IsLeafList():
		return util.AppendErrs(errs, validateLeafList(schema, value))
	case schema.IsList():
		return util.AppendErrs(errs, validateList(schema, value))
	case schema.IsChoice():
		return util.AppendErrs(errs, util.NewErrs(fmt.Errorf("cannot pass choice schema %s to Validate", schema.Name)))
	}

	return util.AppendErrs(errs, util.NewErrs(fmt.Errorf("unknown schema type for type %T, value %v", value, value)))
}
