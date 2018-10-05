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

package util

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
)

var (
	// debugLibrary controls the debugging output from the library data tree
	// traversal. Since this setting causes global variables to be manipulated
	// controlling the output of the library, it MUST NOT be used in a setting
	// whereby thread-safety is required.
	debugLibrary = false
	// debugSchema controls the debugging output from the library from schema
	// matching code. Generates lots of output, so this should be used
	// selectively per test case.
	debugSchema = false
	// maxCharsPerLine is the maximum number of characters per line from
	// DbgPrint and DbgSchema. Additional characters are truncated.
	maxCharsPerLine = 1000
	// maxValueStrLen is the maximum number of characters output from ValueStr.
	maxValueStrLen = 150
)

// DbgPrint prints v if the package global variable debugLibrary is set.
// v has the same format as Printf. A trailing newline is added to the output.
func DbgPrint(v ...interface{}) {
	if !debugLibrary {
		return
	}
	out := fmt.Sprintf(v[0].(string), v[1:]...)
	if len(out) > maxCharsPerLine {
		out = out[:maxCharsPerLine]
	}
	fmt.Println(globalIndent + out)
}

// DbgSchema prints v if the package global variable debugSchema is set.
// v has the same format as Printf.
func DbgSchema(v ...interface{}) {
	if debugSchema {
		fmt.Printf(v[0].(string), v[1:]...)
	}
}

// DbgErr DbgPrints err and returns it.
func DbgErr(err error) error {
	DbgPrint("ERR: " + err.Error())
	return err
}

// globalIndent is used to control Indent level.
var globalIndent = ""

// Indent increases DbgPrint Indent level.
func Indent() {
	if !debugLibrary {
		return
	}
	globalIndent += ". "
}

// Dedent decreases DbgPrint Indent level.
func Dedent() {
	if !debugLibrary {
		return
	}
	globalIndent = strings.TrimPrefix(globalIndent, ". ")
}

// ResetIndent sets the indent level to zero.
func ResetIndent() {
	if !debugLibrary {
		return
	}
	globalIndent = ""
}

// ValueStrDebug returns "<not calculated>" if the package global variable
// debugLibrary is not set. Otherwise, it is the same as ValueStr.
// Use this function instead of ValueStr for debugging purpose, e.g. when the
// output is passed to DbgPrint, because ValueStr calls can be the bottleneck
// for large input.
func ValueStrDebug(value interface{}) string {
	if !debugLibrary {
		return "<not calculated>"
	}
	return ValueStr(value)
}

// ValueStr returns a string representation of value which may be a value, ptr,
// or struct type.
func ValueStr(value interface{}) string {
	out := valueStrInternal(value)
	if len(out) > maxValueStrLen {
		out = out[:maxValueStrLen] + "..."
	}
	return out
}

// ValueStrInternal is the internal implementation of ValueStr.
func valueStrInternal(value interface{}) string {
	v := reflect.ValueOf(value)
	kind := v.Kind()
	switch kind {
	case reflect.Ptr:
		if v.IsNil() || !v.IsValid() {
			return "nil"
		}
		return strings.Replace(ValueStr(v.Elem().Interface()), ")", " ptr)", -1)
	case reflect.Slice:
		var out string
		for i := 0; i < v.Len(); i++ {
			if i != 0 {
				out += ", "
			}
			out += ValueStr(v.Index(i).Interface())
		}
		return "[ " + out + " ]"
	case reflect.Struct:
		var out string
		for i := 0; i < v.NumField(); i++ {
			if i != 0 {
				out += ", "
			}
			if !v.Field(i).CanInterface() {
				continue
			}
			out += ValueStr(v.Field(i).Interface())
		}
		return "{ " + out + " }"
	}
	out := fmt.Sprintf("%v (%v)", value, kind)
	if len(out) > maxValueStrLen {
		out = out[:maxValueStrLen] + "..."
	}
	return out
}

// SchemaTypeStr returns a string representation of the type of element schema
// represents e.g. "container", "choice" etc.
func SchemaTypeStr(schema *yang.Entry) string {
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

// YangTypeToDebugString returns a debug string representation of a YangType.
func YangTypeToDebugString(yt *yang.YangType) string {
	out := fmt.Sprintf("(TypeKind: %s", yang.TypeKindToName[yt.Kind])
	if len(yt.Pattern) != 0 {
		out += fmt.Sprintf(", Pattern: %s", strings.Join(yt.Pattern, " or "))
	}
	if len(yt.Range) != 0 {
		out += fmt.Sprintf(", Range: %s", yt.Range.String())
	}
	return out + ")"
}

// SchemaTreeString returns the schema hierarchy tree as a string with node
// names and types only e.g.
// clock (container)
//   timezone (choice)
//     timezone-name (case)
//       timezone-name (leaf)
//     timezone-utc-offset (case)
//       timezone-utc-offset (leaf)
func SchemaTreeString(schema *yang.Entry, prefix string) string {
	out := prefix + schema.Name + " (" + SchemaTypeStr(schema) + ")" + "\n"
	for _, ch := range schema.Dir {
		out += SchemaTreeString(ch, prefix+"  ")
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
	printFieldsIterFunc := func(ni *NodeInfo, in, out interface{}) (errs Errors) {
		outs := out.(*string)
		prefix := ""
		for i := 0; i < len(strings.Split(ni.Schema.Path(), "/")); i++ {
			prefix += "  "
		}

		fStr := fmt.Sprintf("%s%s", prefix, ni.StructField.Name)
		schemaStr := fmt.Sprintf("[%s (%s)]", ni.Schema.Name, SchemaTypeStr(ni.Schema))
		switch {
		case IsValueScalar(ni.FieldValue):
			*outs += fmt.Sprintf("  %s : %s %s\n", fStr, pretty.Sprint(ni.FieldValue.Interface()), schemaStr)
		case !IsNilOrInvalidValue(ni.FieldKey):
			*outs += fmt.Sprintf("%s%v\n", prefix, ni.FieldKey)
		case !IsNilOrInvalidValue(ni.FieldValue):
			*outs += fmt.Sprintf("%s %s\n", fStr, schemaStr)
		}
		return
	}
	var outStr string
	errs := ForEachField(schema, dataTree, nil, &outStr, printFieldsIterFunc)
	if errs != nil {
		outStr = errs.String()
	}

	return outStr
}
