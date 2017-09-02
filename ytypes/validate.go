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

	"github.com/openconfig/ygot/ygot"
)

// Validate recursively validates the value of the given data tree struct
// against the given schema.
func Validate(schema *yang.Entry, value interface{}) (errors Errors) {
	// Nil value means the field is unset.
	if isNil(value) {
		return nil
	}
	if schema == nil {
		return appendErr(errors, fmt.Errorf("nil schema for type %T, value %v", value, value))
	}
	dbgPrint("Validate with value %v, type %T, schema name %s", valueStr(value), value, schema.Name)

	switch {
	case schema.IsLeaf():
		return validateLeaf(schema, value)
	case schema.IsContainer():
		gsv, ok := value.(ygot.GoStruct)
		if !ok {
			return appendErr(errors, fmt.Errorf("type %T is not a GoStruct for schema %s", value, schema.Name))
		}
		errors = validateContainer(schema, gsv)
	case schema.IsLeafList():
		return validateLeafList(schema, value)
	case schema.IsList():
		return validateList(schema, value)
	case schema.IsChoice():
		return appendErr(errors, fmt.Errorf("cannot pass choice schema %s to Validate", schema.Name))
	default:
		errors = appendErr(errors, fmt.Errorf("unknown schema type for type %T, value %v", value, value))
	}
	return errors
}
