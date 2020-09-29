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
	"regexp"
	"unicode/utf8"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-9.4.

// validateString validates value, which must be a Go string type, against the
// given schema.
func validateString(schema *yang.Entry, value interface{}) error {
	// Check that the schema itself is valid.
	if err := validateStringSchema(schema); err != nil {
		return err
	}

	vv := reflect.ValueOf(value)

	// Check that type of value is the type expected from the schema.
	if vv.Kind() != reflect.String {
		return fmt.Errorf("non string type %T with value %v for schema %s", value, value, schema.Name)
	}

	// This value could be a union typedef string, so convert it to make
	// sure it's the primitive string type.
	stringVal := vv.Convert(reflect.TypeOf("")).Interface().(string)

	// Check that the length is within the allowed range.
	allowedRanges := schema.Type.Length
	strLen := uint64(utf8.RuneCountInString(stringVal))
	if !lengthOk(allowedRanges, strLen) {
		return fmt.Errorf("length %d is outside range %v for schema %s", strLen, allowedRanges, schema.Name)
	}

	// Check that the value satisfies any regex patterns.
	patterns, isPOSIX := util.SanitizedPattern(schema.Type)
	for _, p := range patterns {
		var r *regexp.Regexp
		var err error
		if isPOSIX {
			r, err = regexp.CompilePOSIX(p)
		} else {
			r, err = regexp.Compile(p)
		}
		if err != nil {
			return err
		}
		if !r.MatchString(stringVal) {
			return fmt.Errorf("%q does not match regular expression pattern %q for schema %s", stringVal, r, schema.Name)
		}
	}

	return nil
}

// validateStringSlice validates value, which must be a Go string slice type,
// against the given schema.
func validateStringSlice(schema *yang.Entry, value interface{}) error {
	// Check that the schema itself is valid.
	if err := validateStringSchema(schema); err != nil {
		return err
	}

	// Check that type of value is the type expected from the schema.
	slice, ok := value.([]string)
	if !ok {
		return fmt.Errorf("non []string type %T with value %v for schema %s", value, value, schema.Name)
	}

	// Each slice element must be valid and unique.
	tbl := make(map[string]bool, len(slice))
	for i, val := range slice {
		if err := validateString(schema, val); err != nil {
			return fmt.Errorf("invalid element at index %d: %v for schema %s", i, err, schema.Name)
		}
		if tbl[val] {
			return fmt.Errorf("duplicate string: %q for schema %s", val, schema.Name)
		}
		tbl[val] = true
	}
	return nil
}

// validateStringSchema validates the given string type schema. This is a sanity
// check validation rather than a comprehensive validation against the RFC.
// It is assumed that such a validation is done when the schema is parsed from
// source YANG.
func validateStringSchema(schema *yang.Entry) error {
	if schema == nil {
		return fmt.Errorf("string schema is nil")
	}
	if schema.Type == nil {
		return fmt.Errorf("string schema %s Type is nil", schema.Name)
	}
	if schema.Type.Kind != yang.Ystring {
		return fmt.Errorf("string schema %s has wrong type %v", schema.Name, schema.Type.Kind)
	}

	patterns, isPOSIX := util.SanitizedPattern(schema.Type)
	for _, p := range patterns {
		var err error
		if isPOSIX {
			_, err = regexp.CompilePOSIX(p)
		} else {
			_, err = regexp.Compile(p)
		}
		if err != nil {
			return fmt.Errorf("error generating regexp %s %v for schema %s", p, err, schema.Name)
		}
	}

	return validateLengthSchema(schema)
}
