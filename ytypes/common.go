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

// Package ytypes implements YANG type validation logic.
package ytypes

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

var (
	// debugLibrary controls the debugging output from the library data tree
	// traversal.
	debugLibrary = false
	// debugSchema controls the debugging output from the library from schema
	// matching code. Generates lots of output, so this should be used
	// selectively per test case.
	debugSchema = false

	// YangMaxNumber represents the maximum value for any integer type.
	YangMaxNumber = yang.Number{Kind: yang.MaxNumber}
	// YangMinNumber represents the minimum value for any integer type.
	YangMinNumber = yang.Number{Kind: yang.MinNumber}
)

// schemaTreeRoot returns the root of the schema tree, given any node in that
// tree. It returns nil if schema is nil.
func schemaTreeRoot(schema *yang.Entry) *yang.Entry {
	if schema == nil {
		return nil
	}

	root := schema
	for root.Parent != nil {
		root = root.Parent
	}

	return root
}

// validateLengthSchema validates whether the given schema has a valid length
// specification.
func validateLengthSchema(schema *yang.Entry) error {
	if len(schema.Type.Length) == 0 {
		return nil
	}
	for _, r := range schema.Type.Length {
		// This is a limited sanity check. It's assumed that a full check is
		// done in the goyang parser.
		minLen, maxLen := r.Min, r.Max
		if minLen.Kind != yang.MinNumber && minLen.Kind != yang.Positive {
			return fmt.Errorf("length Min must be Positive or MinNumber: %v for schema %s", minLen, schema.Name)
		}
		if maxLen.Kind != yang.MaxNumber && maxLen.Kind != yang.Positive {
			return fmt.Errorf("length Max must be Positive or MaxNumber: %v for schema %s", minLen, schema.Name)
		}
		if maxLen.Less(minLen) {
			return fmt.Errorf("schema has bad length min[%v] > max[%v] for schema %s", minLen, maxLen, schema.Name)
		}
	}

	return nil
}

// lengthOk reports whether the given value of length falls within the ranges
// allowed by yrs. Always returns true is yrs is empty.
func lengthOk(yrs yang.YangRange, val uint64) bool {
	return isInRanges(yrs, yang.FromUint(val))
}

// isInRanges reports whether the given value falls within the ranges allowed by
// yrs. Always returns true is yrs is empty.
func isInRanges(yrs yang.YangRange, val yang.Number) bool {
	if len(yrs) == 0 {
		return true
	}
	for _, yr := range yrs {
		if isInRange(yr, val) {
			return true
		}
	}
	return false
}

// isInRange reports whether the given value falls within the range allowed by
// yr.
func isInRange(yr yang.YRange, val yang.Number) bool {
	return (val.Less(yr.Max) || val.Equal(yr.Max)) &&
		(yr.Min.Less(val) || yr.Min.Equal(val))
}

// validateListAttr validates any attributes of value present in the schema,
// such as min/max elements. The schema and value can be a container,
// list, or leaf-list type.
func validateListAttr(schema *yang.Entry, value interface{}) (errors []error) {
	if schema == nil {
		return appendErr(errors, fmt.Errorf("schema is nil"))
	}
	if schema.ListAttr == nil {
		return appendErr(errors, fmt.Errorf("schema %s ListAttr is nil", schema.Name))
	}

	var size int
	if value == nil {
		size = 0
	} else {
		switch reflect.TypeOf(value).Kind() {
		case reflect.Slice, reflect.Map:
			size = reflect.ValueOf(value).Len()
		default:
			return appendErr(errors, fmt.Errorf("value %v type %T must be map or slice type for schema %s", value, value, schema.Name))
		}
	}

	// If min/max element attr is present in the schema, this must be a list or
	// leaf-list. Check that the data tree falls within the required size
	// bounds.
	if v := schema.ListAttr.MinElements; v != nil {
		if minN, err := yang.ParseNumber(v.Name); err != nil {
			errors = appendErr(errors, err)
		} else if min, err := minN.Int(); err != nil {
			errors = appendErr(errors, err)
		} else if min < 0 {
			errors = appendErr(errors, fmt.Errorf("list %s has negative min required elements", schema.Name))
		} else if int64(size) < min {
			errors = appendErr(errors, fmt.Errorf("list %s contains fewer than min required elements: %d < %d", schema.Name, size, min))
		}
	}
	if v := schema.ListAttr.MaxElements; v != nil {
		if maxN, err := yang.ParseNumber(v.Name); err != nil {
			errors = appendErr(errors, err)
		} else if max, err := maxN.Int(); err != nil {
			errors = appendErr(errors, err)
		} else if max < 0 {
			errors = appendErr(errors, fmt.Errorf("list %s has negative max required elements", schema.Name))
		} else if int64(size) > max {
			errors = appendErr(errors, fmt.Errorf("list %s contains more than max allowed elements: %d > %d", schema.Name, size, max))
		}
	}

	return
}

// isChoiceOrCase returns true if the entry is either a 'case' or a 'choice'
// node within the schema. These are schema nodes only, and the code generation
// operates on data tree paths.
func isChoiceOrCase(e *yang.Entry) bool {
	return e.IsChoice() || e.IsCase()
}

// findFirstNonChoiceOrCase traverses the data tree and determines the first
// directory nodes from a root e that are neither case nor choice nodes. The
// map, m, is updated in place to append new entries that are found when
// recursively traversing the set of choice/case nodes. The keys in the map
// are the schema element names of the matching elements.
func findFirstNonChoiceOrCase(e *yang.Entry, m map[string]*yang.Entry) {
	switch {
	case !isChoiceOrCase(e):
		m[e.Name] = e
	case e.IsDir():
		for _, ch := range e.Dir {
			findFirstNonChoiceOrCase(ch, m)
		}
	}
}

// dbgPrint prints v if the package global variable debugLibrary is set.
// v has the same format as Printf. A trailing newline is added to the output.
func dbgPrint(v ...interface{}) {
	if !debugLibrary {
		return
	}
	fmt.Printf(v[0].(string), v[1:]...)
	fmt.Println()
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
			out += valueStr(structElems.Field(i).Interface())
		}
		return "{ " + out + " }"
	default:
	}
	return fmt.Sprintf("%v (type %v)", value, kind)
}

// Errors is a slice of error.
type Errors []error

// Error implements the error#Error method.
func (e Errors) Error() string {
	return errStr([]error(e))
}

// String implements the stringer#String method.
func (e Errors) String() string {
	return e.Error()
}

// appendErr appends err to errors if it is not nil and returns the result.
func appendErr(errors []error, err error) []error {
	if len(errors) == 0 && err == nil {
		return nil
	}
	return append(errors, err)
}

// appendErrs appends newErrs to errors and returns the result.
func appendErrs(errors []error, newErrs []error) []error {
	if len(errors) == 0 && len(newErrs) == 0 {
		return nil
	}
	return append(errors, newErrs...)
}

// errStr returns a string representation of errors.
func errStr(errors []error) string {
	var out string
	for i, e := range errors {
		if e == nil {
			continue
		}
		if i != 0 {
			out += ", "
		}
		out += e.Error()
	}
	return out
}

// mapToStrSlice converts a string set expressed as a map m, into a slice of
// strings ss and returns it.
func mapToStrSlice(m map[string]bool) (ss []string) {
	for k := range m {
		ss = append(ss, k)
	}
	return
}

func dbgSchema(v ...interface{}) {
	if debugSchema {
		fmt.Printf(v[0].(string), v[1:]...)
	}
}

// childSchema returns the schema for the struct field f, if f contains a valid
// path tag and the schema path is found in the schema tree. Returns an error
// if the struct tag is invalid. Returns nil if tag is valid but the schema is
// not found in the tree at the specified path.
func childSchema(schema *yang.Entry, f reflect.StructField) (*yang.Entry, error) {
	pathTag, _ := f.Tag.Lookup("path")
	dbgSchema("childSchema for schema %s, field %s, tag %s\n", schema.Name, f.Name, pathTag)
	if rootName, ok := f.Tag.Lookup("rootname"); ok {
		return schema.Dir[rootName], nil
	}
	p, err := pathToSchema(f)
	if err != nil {
		return nil, err
	}

	// Containers have the container schema name as the first element in the
	// path tag for each field e.g. System { Dns ... path: "system/dns"
	// Strip this off since the supplied schema already refers to the struct
	// schema element.
	if schema.IsContainer() && len(p) > 1 && p[0] == schema.Name {
		p = p[1:]
	}
	dbgSchema("pathToSchema yields %v\n", p)
	// For empty path, return the parent schema.
	childSchema := schema
	foundSchema := true
	ok := false
	// Traverse the returned schema path to get the child schema.
	dbgSchema("traversing schema Dirs...")
	for _, pe := range p {
		dbgSchema("/%s", pe)
		childSchema, ok = childSchema.Dir[pe]
		if !ok {
			foundSchema = false
			break
		}
	}
	if foundSchema {
		dbgSchema(" - found\n")
		return childSchema, nil
	}
	dbgSchema(" - not found\n")

	// Path is not null and was not found in the schema. It could be inside a
	// choice/case schema element which is not represented in the path tags.
	// e.g. choice1/case1/leaf1 could have abbreviated tag `path: "leaf1"`.
	// In this case, try to match against any named elements within any choice/
	// case subtrees. These are guaranteed to be unique within the current
	// level namespace so a path tag name match will be unique if one is found.
	if len(p) != 1 {
		// Nodes within choice/case have a path tag with only the last schema
		// path element i.e. choice1/case1/leaf1 path in the schema will have
		// struct tag `path:"leaf1"`. This implies that only paths with length
		// 1 are eligible for this matching.
		return nil, nil
	}
	entries := make(map[string]*yang.Entry)
	for _, ch := range schema.Dir {
		if isChoiceOrCase(ch) {
			findFirstNonChoiceOrCase(ch, entries)
		}
	}

	dbgSchema("checking for %s against non choice/case entries: %v\n", p[0], mapKeys(entries))
	for name, entry := range entries {
		dbgSchema("%s ? ", name)

		if name == p[0] {
			dbgSchema(" - match\n")
			return entry, nil
		}
	}

	dbgSchema(" - no matches\n")
	return nil, nil
}

// mapKeys returns the keys for map m.
func mapKeys(m map[string]*yang.Entry) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// pathToSchema returns a path to the schema for the struct field f.
// Paths are embedded in the "path" struct tag and can be either simple:
//   e.g. "path:a"
// or composite e.g.
//   e.g. "path:config/a|a"
// which is found in OpenConfig leaf-ref cases where the key of a list is a
// leafref. In the latter case, this function returns {"config", "a"}, and the
// schema *yang.Entry for the field is given by schema.Dir["config"].Dir["a"].
func pathToSchema(f reflect.StructField) ([]string, error) {
	pathAnnotation, ok := f.Tag.Lookup("path")
	if !ok {
		return nil, fmt.Errorf("field %s did not specify a path", f.Name)
	}

	paths := strings.Split(pathAnnotation, "|")
	if len(paths) == 1 {
		return strings.Split(pathAnnotation, "/"), nil
	}
	for _, pv := range paths {
		pe := strings.Split(pv, "/")
		if len(pe) > 1 {
			return pe, nil
		}
	}

	return nil, fmt.Errorf("field %s had path tag %s with |, but no elements of form a/b", f.Name, pathAnnotation)
}

// isNil is a general purpose nil check for the kinds of value types expected in
// this package.
func isNil(value interface{}) bool {
	if value == nil {
		return true
	}
	switch reflect.TypeOf(value).Kind() {
	case reflect.Slice, reflect.Ptr, reflect.Map:
		return reflect.ValueOf(value).IsNil()
	default:
	}
	return false
}

// yangBuiltinTypeToGoType returns a pointer to the Go built-in value with
// the type corresponding to the provided YANG type. Returns nil for any type
// which is not an integer, float, string, boolean, or binary kind.
func yangBuiltinTypeToGoType(t yang.TypeKind) interface{} {
	switch t {
	case yang.Yint8:
		return int8(0)
	case yang.Yint16:
		return int16(0)
	case yang.Yint32:
		return int32(0)
	case yang.Yint64:
		return int64(0)
	case yang.Yuint8:
		return uint8(0)
	case yang.Yuint16:
		return uint16(0)
	case yang.Yuint32:
		return uint32(0)
	case yang.Yuint64:
		return uint64(0)
	case yang.Ybool, yang.Yempty:
		return bool(false)
	case yang.Ystring:
		return string("")
	case yang.Ydecimal64:
		return float64(0)
	case yang.Ybinary:
		return []byte(nil)
	default:
		// TODO(mostrowski): handle bitset.
	}
	return nil
}

// yangBuiltinTypeToGoPtrType returns a pointer to the Go built-in value with
// the ptr type corresponding to the provided YANG type. Returns nil for any
// type which is not an integer, float, string, boolean or binary kind.
func yangBuiltinTypeToGoPtrType(t yang.TypeKind) interface{} {
	switch t {
	case yang.Yint8:
		return ygot.Int8(0)
	case yang.Yint16:
		return ygot.Int16(0)
	case yang.Yint32:
		return ygot.Int32(0)
	case yang.Yint64:
		return ygot.Int64(0)
	case yang.Yuint8:
		return ygot.Uint8(0)
	case yang.Yuint16:
		return ygot.Uint16(0)
	case yang.Yuint32:
		return ygot.Uint32(0)
	case yang.Yuint64:
		return ygot.Uint64(0)
	case yang.Ybool, yang.Yempty:
		return ygot.Bool(false)
	case yang.Ystring:
		return ygot.String("")
	case yang.Ydecimal64:
		return ygot.Float64(0)
	case yang.Ybinary:
		return []byte(nil)
	default:
		// TODO(mostrowski): handle bitset.
	}
	return nil
}

// yangTypeToLeafEntry returns a leaf Entry with Type set to t.
func yangTypeToLeafEntry(t *yang.YangType) *yang.Entry {
	return &yang.Entry{
		Kind: yang.LeafEntry,
		Type: t,
	}
}

// isFakeRoot determines whether the supplied yang.Entry represents
// the synthesised root entity in the generated code.
func isFakeRoot(e *yang.Entry) bool {
	if _, ok := e.Annotation["isFakeRoot"]; ok {
		return true
	}
	return false
}
