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
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-9.5.

// validateBool validates value, which must be a Go bool type, against the
// given schema.
func validateBool(schema *yang.Entry, value interface{}) error {
	// Check that the schema itself is valid.
	if err := validateBoolSchema(schema); err != nil {
		return err
	}

	// Check that type of value is the type expected from the schema.
	if _, ok := value.(bool); !ok {
		return fmt.Errorf("non bool type %T with value %v for schema %s", value, value, schema.Name)
	}

	return nil
}

// validateBoolSlice validates value, which must be a Go bool slice type,
// against the given schema.
func validateBoolSlice(schema *yang.Entry, value interface{}) error {
	// Check that the schema itself is valid.
	if err := validateBoolSchema(schema); err != nil {
		return err
	}

	// Check that type of value is the type expected from the schema.
	slice, ok := value.([]bool)
	if !ok {
		return fmt.Errorf("non []bool type %T with value: %v for schema %s", value, value, schema.Name)
	}

	// Each slice element must be valid and unique.
	tbl := make(map[bool]bool)
	for _, val := range slice {
		// A bool value is always valid. There is no need to validate each bool element.
		if tbl[val] {
			return fmt.Errorf("duplicate bool: %v for schema %s", val, schema.Name)
		}
		tbl[val] = true
	}

	return nil
}

// validateBoolSchema validates the given bool type schema. This is a sanity
// check validation rather than a comprehensive validation against the RFC.
// It is assumed that such a validation is done when the schema is parsed from
// source YANG.
func validateBoolSchema(schema *yang.Entry) error {
	if schema == nil {
		return fmt.Errorf("bool schema is nil")
	}
	if schema.Type == nil {
		return fmt.Errorf("bool schema %s Type is nil", schema.Name)
	}
	if schema.Type.Kind != yang.Ybool {
		return fmt.Errorf("bool schema %s has wrong type %v", schema.Name, schema.Type.Kind)
	}

	return nil
}
