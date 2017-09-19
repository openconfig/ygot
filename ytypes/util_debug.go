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
	"strings"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
)

var (
	// debugLibrary controls the debugging output from the library data tree
	// traversal.
	debugLibrary = false
	// debugSchema controls the debugging output from the library from schema
	// matching code. Generates lots of output, so this should be used
	// selectively per test case.
	debugSchema = false
	// maxCharsPerLine is the maximum number of characters per line from
	// dbgPrint and dbgSchema. Additional characters are truncated.
	maxCharsPerLine = 1000
	// maxValueStrLen is the maximum number of characters output from valueStr.
	maxValueStrLen = 150
)

// dbgPrint prints v if the package global variable debugLibrary is set.
// v has the same format as Printf. A trailing newline is added to the output.
func dbgPrint(v ...interface{}) {
	if !debugLibrary {
		return
	}
	out := fmt.Sprintf(v[0].(string), v[1:]...)
	if len(out) > maxCharsPerLine {
		out = out[:maxCharsPerLine]
	}
	fmt.Println(globalIndent + out)
}

// dbgSchema prints v if the package global variable debugSchema is set.
// v has the same format as Printf.
func dbgSchema(v ...interface{}) {
	if debugSchema {
		fmt.Printf(v[0].(string), v[1:]...)
	}
}

// globalIndent is used to control indent level.
var globalIndent = ""

// indent increases dbgPrint indent level.
func indent() {
	globalIndent += ". "
}

// dedent decreases dbgPrint indent level.
func dedent() {
	globalIndent = strings.TrimPrefix(globalIndent, ". ")
}

// valueStr returns a string representation of value which may be a value, ptr,
// or struct type.
func valueStr(value interface{}) string {
	kind := reflect.ValueOf(value).Kind()
	switch kind {
	case reflect.Ptr:
		if reflect.ValueOf(value).IsNil() || !reflect.ValueOf(value).IsValid() {
			return "nil"
		}
		return strings.Replace(valueStr(reflect.ValueOf(value).Elem().Interface()), ")", " ptr)", -1)
	case reflect.Struct:
		var out string
		structElems := reflect.ValueOf(value)
		for i := 0; i < structElems.NumField(); i++ {
			if i != 0 {
				out += ", "
			}
			if !structElems.Field(i).CanInterface() {
				continue
			}
			out += valueStr(structElems.Field(i).Interface())
		}
		return "{ " + out + " }"
	}
	out := fmt.Sprintf("%v (type %v)", value, kind)
	if len(out) > maxValueStrLen {
		out = out[:maxValueStrLen] + "..."
	}
	return out
}

// DataSchemaTreesString outputs a combined data/schema tree string where schema
// is displayed alongside the data tree e.g.
//  [device (container)]
//   RoutingPolicy [routing-policy (container)]
//     DefinedSets [defined-sets (container)]
//       PrefixSet [prefix-set (list)]
//       prefix1
//         prefix1
//         {255.255.255.0/20 20..24}
//           IpPrefix : "255.255.255.0/20" [ip-prefix (leaf)]
//           MasklengthRange : "20..24" [masklength-range (leaf)]
//         PrefixSetName : "prefix1" [prefix-set-name (leaf)]
func DataSchemaTreesString(schema *yang.Entry, dataTree interface{}) string {
	printFieldsIterFunc := func(ni *SchemaNodeInfo, in, out interface{}) (errs []error) {
		outs := out.(*string)
		prefix := ""
		for i := 0; i < len(ni.Path); i++ {
			prefix += "  "
		}

		fStr := fmt.Sprintf("%s%s", prefix, ni.FieldType.Name)
		schemaStr := fmt.Sprintf("[%s (%s)]", ni.Schema.Name, schemaTypeStr(ni.Schema))
		switch {
		case isValueScalar(ni.FieldValue):
			*outs += fmt.Sprintf("%s : %s %s\n", fStr, pretty.Sprint(ni.FieldValue.Interface()), schemaStr)
		case !IsNilOrInvalidValue(ni.NodeInfo.FieldKey):
			*outs += fmt.Sprintf("%s%v\n", prefix, ni.NodeInfo.FieldKey)

		case !IsNilOrInvalidValue(ni.FieldValue):
			*outs += fmt.Sprintf("%s %s\n", fStr, schemaStr)
		}
		return
	}
	var outStr string
	ForEachSchemaNode(schema, dataTree, nil, &outStr, printFieldsIterFunc)
	return outStr
}

// schemaTypeStr returns a string representation of the type of element schema
// represents e.g. "container", "choice" etc.
func schemaTypeStr(schema *yang.Entry) string {
	switch {
	case schema.IsChoice():
		return "choice"
	case schema.IsContainer():
		return "container"
	case schema.IsCase():
		return "case"
	case schema.IsList():
		return "list"
	case schema.IsLeaf():
		return "leaf"
	case schema.IsLeafList():
		return "leaf-list"
	}
	return "other"
}

// SchemaNodeInfo describes a node in a YANG schema tree being traversed. It is
// passed to an function
type SchemaNodeInfo struct {
	// NodeInfo is inherited.
	NodeInfo
	// Path is the path to the current schema node.
	Path []string
	// Schema is the schema for the current node being traversed.
	Schema *yang.Entry
}

// SchemaNodeIteratorFunc is an iteration function for traversing YANG schema
// trees.
// in, out are passed through from the caller to the iteration and can be used
// to pass state in and out.
// It returns a slice of errors encountered while processing the field.
type SchemaNodeIteratorFunc func(ni *SchemaNodeInfo, in, out interface{}) []error

// ForEachSchemaNode recursively iterates through the nodes in schema and
// executes iterFunction on each field.
// in, out are passed through from the caller to the iteration and can be used
// arbitrarily in the iteration function to carry state and results.
// It returns a slice of errors encountered while processing the struct.
func ForEachSchemaNode(schema *yang.Entry, value interface{}, in, out interface{}, iterFunction SchemaNodeIteratorFunc) (errs []error) {
	if isNil(value) {
		return nil
	}
	return forEachSchemaNodeInternal(&SchemaNodeInfo{Schema: schema, NodeInfo: NodeInfo{FieldValue: reflect.ValueOf(value)}}, in, out, iterFunction)
}

// forEachSchemaNodeInternal recursively iterates through the nodes in ni.schema
// and executes iterFunction on each field.
// in, out are passed through from the caller to the iteration and can be used
// arbitrarily in the iteration function to carry state and results.
func forEachSchemaNodeInternal(ni *SchemaNodeInfo, in, out interface{}, iterFunction SchemaNodeIteratorFunc) (errs []error) {
	if IsNilOrInvalidValue(ni.FieldValue) {
		return nil
	}

	errs = appendErrs(errs, iterFunction(ni, in, out))

	switch {
	case IsValueStruct(ni.FieldValue) || IsValueStructPtr(ni.FieldValue):
		structElems := PtrToValue(ni.FieldValue)
		for i := 0; i < structElems.NumField(); i++ {
			cschema, err := childSchema(ni.Schema, structElems.Type().Field(i))
			if err != nil {
				errs = appendErr(errs, fmt.Errorf("%s: %v", structElems.Type().Field(i).Name, err))
				continue
			}
			if cschema == nil {
				continue
			}
			nn := *ni
			nn.Schema = cschema
			nn.Path = append(ni.Path, cschema.Name)
			nn.ParentStruct = ni.FieldValue.Interface()
			nn.FieldType = structElems.Type().Field(i)
			nn.FieldValue = structElems.Field(i)

			errs = appendErrs(errs, forEachSchemaNodeInternal(&nn, in, out, iterFunction))
		}

	case IsValueSlice(ni.FieldValue):
		for i := 0; i < ni.FieldValue.Len(); i++ {
			nn := *ni
			nn.FieldValue = ni.FieldValue.Index(i)

			errs = appendErrs(errs, forEachSchemaNodeInternal(&nn, in, out, iterFunction))
		}

	case IsValueMap(ni.FieldValue):
		for _, key := range ni.FieldValue.MapKeys() {
			nn := *ni
			nn.FieldValue = ni.FieldValue.MapIndex(key)
			nn.FieldKey = key
			nn.FieldKeys = ni.FieldValue.MapKeys()

			errs = appendErrs(errs, forEachSchemaNodeInternal(&nn, in, out, iterFunction))
		}
	}

	return nil
}
