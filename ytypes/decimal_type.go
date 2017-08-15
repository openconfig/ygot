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

// Refer to: https://tools.ietf.org/html/rfc6020#section-9.3.

// validateDecimal validates value, which must be a Go float64 type, against the
// given schema.
func validateDecimal(schema *yang.Entry, value interface{}) error {
	// Check that the schema itself is valid.
	if err := validateDecimalSchema(schema); err != nil {
		return err
	}

	// Check that type of value is the type expected from the schema.
	f, ok := value.(float64)
	if !ok {
		return fmt.Errorf("non float64 type %T with value %v for schema %s", value, value, schema.Name)
	}

	if !isInRanges(schema.Type.Range, yang.FromFloat(f)) {
		return fmt.Errorf("decimal value %v is outside specified ranges for schema %s", value, schema.Name)
	}

	return nil
}

// validateDecimalSlice validates value, which must be a Go float64 slice type,
// against the given schema.
func validateDecimalSlice(schema *yang.Entry, value interface{}) error {
	// Check that the schema itself is valid.
	if err := validateDecimalSchema(schema); err != nil {
		return err
	}

	// Check that type of value is the type expected from the schema.
	slice, ok := value.([]float64)
	if !ok {
		return fmt.Errorf("non []float64 type %T with value: %v for schema %s", value, value, schema.Name)
	}

	// Each slice element must be valid and unique.
	tbl := make(map[float64]bool, len(slice))
	for i, val := range slice {
		if err := validateDecimal(schema, val); err != nil {
			return fmt.Errorf("invalid element at index %d: %v for schema %s", i, err, schema.Name)
		}
		if tbl[val] {
			return fmt.Errorf("duplicate decimal: %v for schema %s", val, schema.Name)
		}
		tbl[val] = true
	}
	return nil
}

// validateDecimalSchema validates the given decimal type schema. This is a
// sanity check validation rather than a comprehensive validation against the
// RFC. It is assumed that such a validation is done when the schema is parsed
// from source YANG.
func validateDecimalSchema(schema *yang.Entry) error {
	if schema == nil {
		return fmt.Errorf("decimal schema is nil")
	}
	if schema.Type == nil {
		return fmt.Errorf("decimal schema %s Type is nil", schema.Name)
	}
	if schema.Type.Kind != yang.Ydecimal64 {
		return fmt.Errorf("decimal schema %s has wrong type %v", schema.Name, schema.Type.Kind)
	}

	return nil
}
