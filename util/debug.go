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

// globalIndent is used to control Indent level.
var globalIndent = ""

// Indent increases DbgPrint Indent level.
func Indent() {
	globalIndent += ". "
}

// Dedent decreases DbgPrint Indent level.
func Dedent() {
	globalIndent = strings.TrimPrefix(globalIndent, ". ")
}

// ValueStr returns a string representation of value which may be a value, ptr,
// or struct type.
func ValueStr(value interface{}) string {
	kind := reflect.ValueOf(value).Kind()
	switch kind {
	case reflect.Ptr:
		if reflect.ValueOf(value).IsNil() || !reflect.ValueOf(value).IsValid() {
			return "nil"
		}
		return strings.Replace(ValueStr(reflect.ValueOf(value).Elem().Interface()), ")", " ptr)", -1)
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
			out += ValueStr(structElems.Field(i).Interface())
		}
		return "{ " + out + " }"
	}
	out := fmt.Sprintf("%v (type %v)", value, kind)
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
