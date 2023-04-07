// Copyright 2023 Google Inc.
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

package gnmidiff

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

type updateValue struct {
	val interface{}
	// fromScalar indicates that the value was retrieved from a TypedValue
	// scalar type and not JSON, and thus the diff routine must be
	// cognizant that a uint64 or int64 value may actually need to be a
	// string when comparing against JSON.
	fromScalar bool
}

// setRequestIntent represents the minimal intent of a SetRequest.
//
// replaces in a SetRequest must be reduced into delete and updates.
type setRequestIntent struct {
	// Deletes are deletions to any node.
	Deletes map[string]struct{}
	// Updates are leaf updates only.
	Updates map[string]updateValue
}

// writeUpdate populates the intent with the leaf values provided.
//
// - errorOnOverwrite indicates that if the update overwrote any current values
// in the intent, then error out.
func (intent *setRequestIntent) writeUpdate(pathToLeaf string, val interface{}, fromScalar, errorOnOverwrite bool) error {
	if prevVal, ok := intent.Updates[pathToLeaf]; errorOnOverwrite && ok && val != prevVal.val {
		return fmt.Errorf("gnmidiff: leaf value set twice with different values in SetRequest: %v", pathToLeaf)
	}
	intent.Updates[pathToLeaf] = updateValue{val: val, fromScalar: fromScalar}
	return nil
}

// populateUpdate populates all leaf updates at the given path into the intent.
//
// For any leaf updates, the corresponding path in the intent's delete is
// removed (if exists).
//
// - schema is intended to be provided via the function defined in generated
// ygot code (e.g. exampleoc.Schema).
// If schema is nil, then it is assumed any JSON value in the input
// SetRequest MUST conform to the OpenConfig YANG style guidelines. Otherwise
// it is impossible to infer the list keys of a leaf update.
// - errorOnOverwrite indicates that if the update overwrote any current values
// in the intent, then error out.
//
// Note: The input path must NOT end with "/".
//
// e.g. /a for b/c="foo" would introduce an update of /a/b/c="foo" into the intent.
func (intent *setRequestIntent) populateUpdate(path string, tv *gpb.TypedValue, schema *ytypes.Schema, errorOnOverwrite bool) error {
	if len(path) > 0 && path[len(path)-1] == '/' {
		return fmt.Errorf("gnmidiff: invalid input path %q, must not end with \"/\"", path)
	}

	// Populate updates when the schema is unknown.
	if schema == nil {
		return populateUpdateNoSchema(intent, path, tv, errorOnOverwrite)
	}

	gpath, err := ygot.StringToStructuredPath(path)
	if err != nil {
		return fmt.Errorf("gnmidiff: %v", err)
	}

	// The code below uses the schema to populate leaves specified in the
	// input TypedValue into the generated ygot-GoStruct, and then
	// marshals the flattened leaf updates using ygot.TogNMINotifications.
	//
	// The procedure for populating the leaves differs by whether the input
	// path is a leaf or a non-leaf.
	//
	// - For leaves, the procedure is straightforward: simply unmarshal and
	// then marshal.
	// - For non-leaves, the procedure is more complicated due to the
	// implicit creation of list keys when a leaf is unmarshalled into a
	// GoStruct: when the input path points to a list entry, but the input
	// TypedValue does not contain list keys, then ygot.TogNMINotifications
	// will marshal these list keys. Although this is technically not
	// different in intent, for consistency with the non-schema-aware
	// result, these list keys should not be part of the returned intent.
	//
	// To solve this problem, the unmarshal target is set to be the
	// non-leaf node itself, which avoids the implicit creation of these
	// list keys.
	if !schema.IsValid() {
		return fmt.Errorf("gnmidiff: input schema is not valid: %+v", schema)
	}
	rootSchema := schema.RootSchema()
	targetSchema, err := util.FindLeafRefSchema(rootSchema, path)
	if err != nil {
		return fmt.Errorf("gnmidiff: error finding target schema: %v", err)
	}
	setNodeTargetSchema := rootSchema
	// Create a new empty root since we don't want previous updates to
	// count as part of the current update.
	setNodeTarget, ok := reflect.New(reflect.TypeOf(schema.Root).Elem()).Interface().(ygot.GoStruct)
	if !ok {
		return fmt.Errorf("schema root is a non-GoStruct, this is not allowed: %T, %v", schema.Root, schema.Root)
	}
	setNodePath := &gpb.Path{}
	if targetSchema.IsLeaf() || targetSchema.IsLeafList() {
		// leaf replace is the same as a leaf update, so remove
		// the deletion action to keep the intent minimal.
		delete(intent.Deletes, path)
	} else {
		// For a non-leaf update, we want to call SetNode on
		// the non-leaf node itself to avoid creating key
		// leaves as a side-effect of unmarshalling a path that
		// ends at a list entry.
		setNodePath = gpath
		gpath = &gpb.Path{}

		nodeI, _, err := ytypes.GetOrCreateNode(rootSchema, schema.Root, setNodePath)
		if err != nil {
			return fmt.Errorf("failed to GetOrCreate the prefix node: %v", err)
		}
		targetType := reflect.TypeOf(nodeI).Elem()
		var ok bool
		if setNodeTarget, ok = reflect.New(targetType).Interface().(ygot.GoStruct); !ok {
			return fmt.Errorf("prefix path points to a non-GoStruct, this is not allowed: %T, %v", nodeI, nodeI)
		}
		setNodeTargetSchema = schema.SchemaTree[targetType.Name()]
	}

	if err := ytypes.SetNode(setNodeTargetSchema, setNodeTarget, gpath, tv, &ytypes.InitMissingElements{}); err != nil {
		return fmt.Errorf("gnmidiff: error unmarshalling update: %v", err)
	}

	jsonBytes, err := ygot.Marshal7951(setNodeTarget)
	if err != nil {
		return fmt.Errorf("gnmidiff: error marshalling GoStruct: %v", err)
	}

	updates, err := flattenOCJSON(jsonBytes, false)
	if err != nil {
		return err
	}
	prefixPathStr := ""
	if len(setNodePath.Elem) > 0 {
		if prefixPathStr, err = ygot.PathToString(setNodePath); err != nil {
			return fmt.Errorf("gnmidiff: error creating prefix path string: %v", err)
		}
	}
	for subpath, val := range updates {
		pathToLeaf := prefixPathStr + subpath
		if !strings.HasPrefix(pathToLeaf, path) {
			// This was a list key that was created by SetNode, so drop it.
			continue
		}
		if err := intent.writeUpdate(pathToLeaf, val, false, errorOnOverwrite); err != nil {
			return err
		}
	}
	return nil
}

// populateUpdateNoSchema populates all leaf updates at the given path into the intent.
//
// e.g. /a for b/c="foo" would introduce an update of /a/b/c="foo" into the intent.
//
// - errorOnOverwrite indicates that if the update overwrote any current values
// in the intent, then error out.
func populateUpdateNoSchema(intent *setRequestIntent, path string, tv *gpb.TypedValue, errorOnOverwrite bool) error {
	var leafVal interface{}
	isLeaf := true
	fromScalar := false
	switch tv.GetValue().(type) {
	case *gpb.TypedValue_JsonIetfVal:
		updates, err := flattenOCJSON(tv.GetJsonIetfVal(), false)
		if err != nil {
			return err
		}
		if val, ok := updates[""]; len(updates) == 1 && ok {
			leafVal = val
		} else {
			isLeaf = false
			for subpath, val := range updates {
				if err := intent.writeUpdate(path+subpath, val, false, errorOnOverwrite); err != nil {
					return err
				}
			}
		}
	default:
		fromScalar = true
		var err error
		leafVal, err = protoLeafToJSON(tv)
		if err != nil {
			return err
		}
	}

	if isLeaf {
		// leaf replace is the same as a leaf update, so remove
		// the deletion action to keep the intent minimal.
		delete(intent.Deletes, path)
		if err := intent.writeUpdate(path, leafVal, fromScalar, errorOnOverwrite); err != nil {
			return err
		}
	}
	return nil
}

// binaryBase64 takes an input byte slice and returns it as a base64
// encoded string.
func binaryBase64(i []byte) string {
	return base64.StdEncoding.EncodeToString(i)
}

// protoLeafToJSON converts a gNMI proto scalar to the equivalent RFC7951 JSON
// representation.
//
// NOTE: The lossiness in TypedValue proto scalars prevents a one-to-one
// mapping to JSON, namely for int64/uint64.
func protoLeafToJSON(tv *gpb.TypedValue) (interface{}, error) {
	switch tv.GetValue().(type) {
	case *gpb.TypedValue_StringVal:
		return tv.GetStringVal(), nil
	case *gpb.TypedValue_BoolVal:
		return tv.GetBoolVal(), nil
	case *gpb.TypedValue_IntVal:
		// NOTE: If that actual IntVal is an int64 in YANG, then this
		// should've been a string. However, there is no way to tell
		// that because TypedValue is lossy.
		return float64(tv.GetIntVal()), nil
	case *gpb.TypedValue_UintVal:
		// NOTE: If that actual UintVal is an uint64 in YANG, then this
		// should've been a string. However, there is no way to tell
		// that because TypedValue is lossy.
		return float64(tv.GetUintVal()), nil
	case *gpb.TypedValue_DoubleVal:
		return strconv.FormatFloat(tv.GetDoubleVal(), 'f', -1, 64), nil
	case *gpb.TypedValue_LeaflistVal:
		elems := tv.GetLeaflistVal().GetElement()
		ss := make([]interface{}, len(elems))
		for x, e := range elems {
			var err error
			if ss[x], err = protoLeafToJSON(e); err != nil {
				return nil, err
			}
		}
		return ss, nil
	case *gpb.TypedValue_BytesVal:
		return binaryBase64(tv.GetBytesVal()), nil
	default:
		return nil, fmt.Errorf("gnmidiff: TypedValue type %T is not a scalar type", tv.GetValue())
	}
}
