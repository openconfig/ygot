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
	"reflect"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-7.7.

// validateLeafList validates each of the values in value against the given
// schema. value is expected to be a slice of the Go type corresponding to the
// YANG type in the schema.
func validateLeafList(schema *yang.Entry, value interface{}) util.Errors {
	var errors []error
	if util.IsValueNil(value) {
		return nil
	}
	// Check that the schema itself is valid.
	if err := validateLeafListSchema(schema); err != nil {
		return util.NewErrs(err)
	}

	util.DbgPrint("validateLeafList with value %v, type %T, schema name %s", util.ValueStrDebug(value), value, schema.Name)

	switch reflect.TypeOf(value).Kind() {
	case reflect.Slice:
		v := reflect.ValueOf(value)
		for i := 0; i < v.Len(); i++ {
			cv := v.Index(i).Interface()

			// Handle the case that this is a leaf-list of enumerated values, where we expect that the
			// input to validateLeaf is a scalar value, rather than a pointer.
			if _, ok := cv.(ygot.GoEnum); ok {
				errors = util.AppendErrs(errors, validateLeaf(schema, cv))
			} else {
				errors = util.AppendErrs(errors, validateLeaf(schema, &cv))
			}

		}
	default:
		errors = util.AppendErr(errors, fmt.Errorf("expected slice type for %s, got %T", schema.Name, value))
	}

	return errors
}

// validateLeafListSchema validates the given list type schema. This is a sanity
// check validation rather than a comprehensive validation against the RFC.
// It is assumed that such a validation is done when the schema is parsed from
// source YANG.
func validateLeafListSchema(schema *yang.Entry) error {
	if schema == nil {
		return fmt.Errorf("list schema is nil")
	}
	if !schema.IsLeafList() {
		return fmt.Errorf("schema for %s with type %v is not leaf list type", schema.Name, schema.Kind)
	}

	if schema.Type.Kind == yang.Yempty {
		return fmt.Errorf("schema for %s contains leaf-list of empty type, invalid YANG", schema.Name)
	}

	return nil
}

// unmarshalLeafList unmarshals given value into a Go slice parent.
//   schema is the schema of the schema node corresponding to the field being
//     unmamshaled into
//   enc is the encoding type used to encode the value
//   value is a JSON array if enc is JSONEncoding, represented as Go slice
//   value is a gNMI TypedValue if enc is GNMIEncoding, represented as TypedValue_LeafListVal
func unmarshalLeafList(schema *yang.Entry, parent interface{}, value interface{}, enc Encoding) error {
	if util.IsValueNil(value) {
		return nil
	}
	// Check that the schema itself is valid.
	if err := validateLeafListSchema(schema); err != nil {
		return err
	}

	util.DbgPrint("unmarshalLeafList value %v, type %T, into parent type %T, schema name %s", util.ValueStrDebug(value), value, parent, schema.Name)

	// The leaf schema is just the leaf-list schema without the list attrs.
	leafSchema := *schema
	leafSchema.ListAttr = nil

	switch enc {
	case GNMIEncoding:
		if _, ok := value.(*gpb.TypedValue); !ok {
			return fmt.Errorf("unmarshalLeafList for schema %s: value %v: got type %T, expect *gpb.TypedValue", schema.Name, util.ValueStr(value), value)
		}
		tv := value.(*gpb.TypedValue)
		sa, ok := tv.GetValue().(*gpb.TypedValue_LeaflistVal)
		if !ok {
			return fmt.Errorf("unmarshalLeafList for schema %s: value %v: got type %T, expect *gpb.TypedValue_LeaflistVal set in *gpb.TypedValue", schema.Name, util.ValueStr(value), tv.GetValue())
		}
		for _, v := range sa.LeaflistVal.GetElement() {
			if err := unmarshalGeneric(&leafSchema, parent, v, enc); err != nil {
				return err
			}
		}
	case JSONEncoding:
		leafList, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("unmarshalLeafList for schema %s: value %v: got type %T, expect []interface{}", schema.Name, util.ValueStr(value), value)
		}

		for _, leaf := range leafList {
			if err := unmarshalGeneric(&leafSchema, parent, leaf, enc); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unknown encoding %v", enc)
	}

	return nil
}
