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

// Unmarshal recursively unmarshals JSON data tree in value into the given
// parent, using the given schema. Any values already in the parent that are
// not present in value are preserved.
func Unmarshal(schema *yang.Entry, parent interface{}, value interface{}) error {
	indent()
	defer dedent()

	// Nil value means the field is unset.
	if isNil(value) {
		return nil
	}
	if schema == nil {
		return fmt.Errorf("nil schema for parent type %T, value %v (%T)", parent, value, value)
	}
	dbgPrint("Unmarshal value %v, type %T, into parent type %T, schema name %s", valueStr(value), value, parent, schema.Name)

	switch {
	case schema.IsLeaf():
		return unmarshalLeaf(schema, parent, value)
	case schema.IsLeafList():
		return unmarshalLeafList(schema, parent, value)
	case schema.IsList():
		return unmarshalList(schema, parent, value)
	case schema.IsChoice():
		return fmt.Errorf("cannot pass choice schema %s to Unmarshal", schema.Name)
	case schema.IsContainer():
		return unmarshalContainer(schema, parent, value)
	}
	return fmt.Errorf("unknown schema type for type %T, value %v", value, value)
}
