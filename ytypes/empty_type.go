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
	"github.com/openconfig/ygot/ygot"
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-9.11.

// validateEmpty validates value, which must be a derived type corresponding
// to the ygot.EmptyTypeName, against the given schema.
func validateEmpty(schema *yang.Entry, value interface{}) error {
	// Check that the schema itself is valid.
	if err := validateEmptySchema(schema); err != nil {
		return err
	}

	if schema.Type.Kind == yang.Yempty {
		if reflect.TypeOf(value).Name() != ygot.EmptyTypeName {
			return fmt.Errorf("non derived type %T with value %v for schema %s", value, value, schema.Name)
		}
	}

	return nil
}

// validateEmptySchema validates the given empty type schema. This is a sanity
// check validation rather than a comprehensive validation against the RFC.
// It is assumed that such a validation is done when the schema is parsed from
// source YANG.
func validateEmptySchema(schema *yang.Entry) error {
	if schema == nil {
		return fmt.Errorf("empty schema is nil")
	}
	if schema.Type == nil {
		return fmt.Errorf("empty schema %s Type is nil", schema.Name)
	}
	if schema.Type.Kind != yang.Yempty {
		return fmt.Errorf("empty schema %s has wrong type %v", schema.Name, schema.Type.Kind)
	}

	return nil
}
