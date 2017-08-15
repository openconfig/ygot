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
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-9.7.

// validateBitset validates value, which must be a Go string type, against the
// given schema.
func validateBitset(schema *yang.Entry, value interface{}) error {
	// Check that the schema itself is valid.
	if err := validateBitsetSchema(schema); err != nil {
		return err
	}

	// Check that type of value is the type expected from the schema.
	val, ok := value.(string)
	if !ok {
		return fmt.Errorf("non bitset type %T with value %v for schema %s", value, value, schema.Name)
	}

	// Check that the bitset names are defined.
	bitsetNames := strings.Split(val, " ")
	for _, name := range bitsetNames {
		if !schema.Type.Bit.IsDefined(name) {
			return fmt.Errorf("nonexistent bit name: %q for schema %s", name, schema.Name)
		}
	}
	return nil
}

// validateBitsetSlice validates value, which must be a Go string slice type,
// against the given schema.
func validateBitsetSlice(schema *yang.Entry, value interface{}) error {
	// Check that the schema itself is valid.
	if err := validateBitsetSchema(schema); err != nil {
		return err
	}

	// Check that type of value is the type expected from the schema.
	slice, ok := value.([]string)
	if !ok {
		return fmt.Errorf("non []string type %T with value: %v for schema %s", value, value, schema.Name)
	}

	// Each slice element must be valid and unique.
	tbl := make(map[string]bool, len(slice))
	for i, val := range slice {
		if err := validateBitset(schema, val); err != nil {
			return fmt.Errorf("invalid element at index %d: %v for schema %s", i, err, schema.Name)
		}
		bitsetSlice := strings.Split(val, " ")
		sort.Strings(bitsetSlice)
		str := strings.Join(bitsetSlice, " ")
		if tbl[str] {
			return fmt.Errorf("duplicate bit set: %v for schema %s", val, schema.Name)
		}
		tbl[str] = true
	}
	return nil
}

// validateBitsetSchema validates the given Bitset type schema. This is a sanity
// check validation rather than a comprehensive validation against the RFC.
// It is assumed that such a validation is done when the schema is parsed from
// source YANG.
func validateBitsetSchema(schema *yang.Entry) error {
	if schema == nil {
		return fmt.Errorf("bitset schema is nil")
	}
	if schema.Type == nil {
		return fmt.Errorf("bitset schema %s Type is nil", schema.Name)
	}
	if schema.Type.Kind != yang.Ybits {
		return fmt.Errorf("bitset schema %s has wrong type %v", schema.Name, schema.Type.Kind)
	}

	return nil
}
