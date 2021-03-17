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

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-9.8.

// ValidateBinaryRestrictions checks that the given binary string matches the
// schema's length restrictions (if any). It returns an error if the validation
// fails.
func ValidateBinaryRestrictions(schemaType *yang.YangType, binaryVal []byte) error {
	allowedRanges := schemaType.Length
	if !lengthOk(allowedRanges, uint64(len(binaryVal))) {
		return fmt.Errorf("length %d is outside range %v", len(binaryVal), allowedRanges)
	}
	return nil
}

// validateBinary validates value, which must be a Go string type, against the
// given schema.
func validateBinary(schema *yang.Entry, value interface{}) error {
	if util.IsValueNil(value) {
		return nil
	}
	// Check that the schema itself is valid.
	if err := validateBinarySchema(schema); err != nil {
		return err
	}

	// Check that type of value is the type expected from the schema.
	if !isBinaryType(reflect.TypeOf(value)) {
		return fmt.Errorf("non binary type %T with value %v for schema %s", value, value, schema.Name)
	}

	// Check that the length is within the allowed range.
	binaryVal := reflect.ValueOf(value).Bytes()

	if err := ValidateBinaryRestrictions(schema.Type, binaryVal); err != nil {
		return fmt.Errorf("schema %q: %v", schema.Name, err)
	}
	return nil
}

// validateBinarySlice validates value, which must be a Go string slice type,
// against the given schema.
func validateBinarySlice(schema *yang.Entry, value interface{}) error {
	// Check that the schema itself is valid.
	if err := validateBinarySchema(schema); err != nil {
		return err
	}

	// Check that type of value is the type expected from the schema.
	if !isBinarySliceType(reflect.TypeOf(value)) {
		return fmt.Errorf("non []Binary type %T with value: %v for schema %s", value, value, schema.Name)
	}

	// Each slice element must be valid and unique.
	v := reflect.ValueOf(value)
	tbl := make(map[string]bool, v.Len())
	for i := 0; i < v.Len(); i++ {
		val := v.Index(i)
		if err := validateBinary(schema, val.Interface()); err != nil {
			return fmt.Errorf("invalid element at index %d: %v", i, err)
		}
		binaryVal := val.Bytes()
		if tbl[string(binaryVal)] {
			return fmt.Errorf("duplicate binary type: %v for schema %s", binaryVal, schema.Name)
		}
		tbl[string(binaryVal)] = true
	}
	return nil
}

// validateBinarySchema validates the given binary type schema. This is a quick
// check rather than a comprehensive validation against the RFC.
// It is assumed that such a validation is done when the schema is parsed from
// source YANG.
func validateBinarySchema(schema *yang.Entry) error {
	if schema == nil {
		return fmt.Errorf("binary schema is nil")
	}
	if schema.Type == nil {
		return fmt.Errorf("binary schema %s Type is nil", schema.Name)
	}
	if schema.Type.Kind != yang.Ybinary {
		return fmt.Errorf("binary schema %s has wrong type %v", schema.Name, schema.Type.Kind)
	}
	return validateLengthSchema(schema)
}

// isBinaryType reports whether input t is a Binary type derived from []byte.
func isBinaryType(t reflect.Type) bool {
	return t != nil && t.Name() == ygot.BinaryTypeName && t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8
}

// isBinarySliceType reports whether input t is a []Binary type
func isBinarySliceType(t reflect.Type) bool {
	return t != nil && t.Kind() == reflect.Slice && isBinaryType(t.Elem())
}
