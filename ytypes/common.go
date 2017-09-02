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
// TODO(b/65052436): split this file up into logical units.
package ytypes

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	log "github.com/golang/glog"
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
	maxValueStrLen = 50

	// YangMaxNumber represents the maximum value for any integer type.
	YangMaxNumber = yang.Number{Kind: yang.MaxNumber}
	// YangMinNumber represents the minimum value for any integer type.
	YangMinNumber = yang.Number{Kind: yang.MinNumber}
)

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

// isNil is a general purpose nil check for the kinds of value types expected in
// this package.
func isNil(value interface{}) bool {
	if value == nil {
		return true
	}
	switch reflect.TypeOf(value).Kind() {
	case reflect.Slice, reflect.Ptr, reflect.Map:
		return reflect.ValueOf(value).IsNil()
	}
	return false
}

// isValueScalar reports whether v is a scalar (non-composite) type.
func isValueScalar(v reflect.Value) bool {
	return !IsValueStruct(v) && !IsValueStructPtr(v) && !IsValueMap(v) && !IsValueSlice(v)
}

// isFakeRoot reports whether the supplied yang.Entry represents the synthesised
// root entity in the generated code.
func isFakeRoot(e *yang.Entry) bool {
	if _, ok := e.Annotation["isFakeRoot"]; ok {
		return true
	}
	return false
}

// isUnkeyedList reports whether e is an unkeyed list.
func isUnkeyedList(e *yang.Entry) bool {
	return e.IsList() && e.Key == ""
}

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

// getJSONTreeValForField returns the JSON subtree of the provided tree that
// corresponds to any of the paths in struct field f.
// If no such JSON subtree exists, it returns nil, nil.
// If more than one path has a JSON subtree, the function returns an error if
// the two subtrees are unequal.
func getJSONTreeValForField(parentSchema, schema *yang.Entry, f reflect.StructField, tree interface{}) (interface{}, error) {
	ps, err := dataTreePaths(parentSchema, schema, f)
	if err != nil {
		return nil, err
	}
	var out interface{}
	var outPath []string
	for _, p := range ps {
		if jr, ok := getJSONTreeValForPath(tree, p); ok {
			if out != nil && !reflect.DeepEqual(out, jr) {
				return nil, fmt.Errorf("values at paths %v and %v are different: %v != %v", outPath, p, out, jr)
			}
			out = jr
			outPath = p
		}
	}

	return out, nil
}

// getJSONTreeValForPath returns a JSON subtree from tree at the given path from
// the root. If returns (nil, false) if no subtree is found at the given path.
func getJSONTreeValForPath(tree interface{}, path []string) (interface{}, bool) {
	t := tree
	for i := 0; i < len(path); i++ {
		tt, ok := t.(map[string]interface{})
		if !ok {
			return nil, false
		}
		nv, ok := mapValueNoPrefix(tt, path[i])
		if !ok {
			return nil, false
		}
		t = nv
	}
	dbgPrint("path:%v matches tree %s, return subtree %s", path, valueStr(tree), valueStr(t))

	return t, true
}

// childSchema returns the schema for the struct field f, if f contains a valid
// path tag and the schema path is found in the schema tree. It returns an error
// if the struct tag is invalid, or nil if tag is valid but the schema is not
// found in the tree at the specified path.
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
	// Traverse the returned schema path to get the child schema.
	dbgSchema("traversing schema Dirs...")
	for ; len(p) > 0; p = p[1:] {
		dbgSchema("/%s", p[0])
		ns, ok := childSchema.Dir[stripModulePrefix(p[0])]
		if !ok {
			foundSchema = false
			break
		}
		childSchema = ns
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
	for _, ch := range childSchema.Dir {
		if isChoiceOrCase(ch) {
			findFirstNonChoiceOrCase(ch, entries)
		}
	}

	dbgSchema("checking for %s against non choice/case entries: %v\n", p[0], stringMapKeys(entries))
	for name, entry := range entries {
		dbgSchema("%s ? ", name)

		if stripModulePrefix(name) == p[0] {
			dbgSchema(" - match\n")
			return entry, nil
		}
	}

	dbgSchema(" - no matches\n")
	return nil, nil
}

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

// findFirstNonChoiceOrCase recursively traverses the schema tree and populates
// m with the set of the first nodes in every path that neither case nor choice
// nodes. The keys in the map are the schema element names of the matching
// elements.
// TODO(b/64810349): move to common library.
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
		pathAnnotation = strings.TrimPrefix(pathAnnotation, "/")
		return strings.Split(pathAnnotation, "/"), nil
	}
	for _, pv := range paths {
		pv = strings.TrimPrefix(pv, "/")
		pe := strings.Split(pv, "/")
		if len(pe) > 1 {
			return pe, nil
		}
	}

	return nil, fmt.Errorf("field %s had path tag %s with |, but no elements of form a/b", f.Name, pathAnnotation)
}

// schemaPaths returns all the paths in the path tag, plus the path value of
// the rootname tag, if one is present.
func schemaPaths(schema *yang.Entry, f reflect.StructField) ([][]string, error) {
	var out [][]string
	rootTag, ok := f.Tag.Lookup("rootname")
	if ok {
		out = append(out, strings.Split(rootTag, "/"))
	}
	pathTag, ok := f.Tag.Lookup("path")
	if (!ok || pathTag == "") && rootTag == "" {
		return nil, fmt.Errorf("field %s did not specify a path", f.Name)
	}
	if pathTag == "" {
		return out, nil
	}

	ps := strings.Split(pathTag, "|")
	for _, p := range ps {
		sp := removeRootPrefix(strings.Split(p, "/"))
		out = append(out, stripModulePrefixes(sp))
	}
	return out, nil
}

// dataTreePaths returns all the data tree paths corresponding to schemaPaths.
// Any intermediate nodes not found in the data tree (i.e. choice/case) are
// removed from the paths.
func dataTreePaths(parentSchema, schema *yang.Entry, f reflect.StructField) ([][]string, error) {
	out, err := schemaPaths(schema, f)
	if err != nil {
		return nil, err
	}
	n, err := removeNonDataPathElements(parentSchema, schema, out)
	dbgPrint("have paths %v, removing non-data from %s -> %v", out, schema.Name, n)
	return n, err
}

// removeNonDataPathElements removes any path elements in paths not found in
// the data tree given the terminal node schema and the schema of its parent.
func removeNonDataPathElements(parentSchema, schema *yang.Entry, paths [][]string) ([][]string, error) {
	var out [][]string
	for _, path := range paths {
		var po []string
		s := parentSchema
		if path[0] == s.Name {
			po = append(po, path[0])
			path = path[1:]
		}
		for _, pe := range path {
			s = s.Dir[pe]
			if s == nil {
				// Some paths exist only in the data tree but not the schema
				// tree. In this case just retain the path purely on trust.
				// TODO(mostrowski): make this more robust. It should be in
				// the root only.
				po = path
				break
			}
			if !isChoiceOrCase(s) {
				po = append(po, pe)
			}
		}
		out = append(out, po)
	}

	return out, nil
}

// checkDataTreeAgainstPaths checks each of dataPaths against the first level
// of the data tree. It returns an error with the first element in the data tree first
// level that is not found in dataPaths.
// This function is used to verify that the jsonTree does not contain any elements
// in the first level that do not have data paths found in the schema.
func checkDataTreeAgainstPaths(jsonTree map[string]interface{}, dataPaths [][]string) error {
	// Go over all first level JSON tree map keys to make sure they all point
	// to valid schema paths.
	pm := map[string]bool{}
	for _, sp := range dataPaths {
		pm[stripModulePrefix(sp[0])] = true
	}
	dbgSchema("check dataPaths %v against dataTree %v\n", pm, jsonTree)
	for jf := range jsonTree {
		if !pm[stripModulePrefix(jf)] {
			return fmt.Errorf("JSON contains unexpected field %s", jf)
		}
	}

	return nil
}

// removeRootPrefix removes the root prefix from root schema entities e.g.
// Bgp_Global has path "/bgp/global" == {"", "bgp", "global"}
//   -> {"global"}
func removeRootPrefix(path []string) []string {
	if len(path) < 2 || path[0] != "" {
		// not a root path
		return path
	}
	return path[2:]
}

// mapValueNoPrefix returns the map value matching the key in the absence of
// module prefixes in either the provided key or the map key.
func mapValueNoPrefix(t map[string]interface{}, key string) (interface{}, bool) {
	for k, v := range t {
		if stripModulePrefix(k) == stripModulePrefix(key) {
			return v, true
		}
	}

	return nil, false
}

// resolveLeafRef returns a ptr to the schema pointed to by the provided leaf-ref
// schema. It returns schema itself if schema is not a leaf-ref.
func resolveLeafRef(schema *yang.Entry) (*yang.Entry, error) {
	if schema == nil {
		return nil, nil
	}
	// TODO(mostrowski): this should only be possible in fakeroot. Add an
	// explicit check for that once data is available in the schema.
	if schema.Type == nil {
		return schema, nil
	}

	orig := schema
	s := schema
	for ykind := s.Type.Kind; ykind == yang.Yleafref; {
		ns, err := findLeafRefSchema(s, s.Type.Path)
		if err != nil {
			return schema, err
		}
		s = ns
		ykind = s.Type.Kind
	}

	if s != orig {
		dbgPrint("follow schema leaf-ref from %s to %s, type %v", orig.Name, s.Name, s.Type.Kind)
	}
	return s, nil
}

// schemaToStructFieldName returns the string name of the field, which must be
// contained in parent (a struct ptr), given the schema for the field.
// It returns empty string and nil error if the field does not exist in the
// parent struct.
func schemaToStructFieldName(schema *yang.Entry, parent interface{}) (string, *yang.Entry, error) {
	v := reflect.ValueOf(parent)
	if IsNilOrInvalidValue(v) {
		return "", nil, fmt.Errorf("parent field is nil in schemaToStructFieldName for node %s", schema.Name)
	}

	t := reflect.TypeOf(parent)
	switch t.Kind() {
	case reflect.Map, reflect.Slice:
		t = t.Elem()
	}
	// If parent is a map of struct ptrs, still need to deref the element type.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fieldName := f.Name
		p, err := pathToSchema(f)
		if err != nil {
			return "", nil, err
		}
		if hasRelativePath(schema, p) {
			return fieldName, schema, nil
		}
		if ns := findSchemaAtPath(schema, p); ns != nil {
			return fieldName, ns, nil
		}
	}

	return "", nil, fmt.Errorf("struct field %s not found in parent %v, type %T", schema.Name, parent, parent)
}

// findSchemaAtPath returns the schema at the given path, ignoring module
// prefixes in the path. It returns nil if no schema is found.
func findSchemaAtPath(schema *yang.Entry, path []string) *yang.Entry {
	s := schema
	for i := 0; i < len(path); i++ {
		pe := stripModulePrefix(path[i])
		if s.Dir[pe] == nil {
			return nil
		}
		s = s.Dir[pe]
	}
	return s
}

// hasRelativePath reports whether the given schema node matches the given
// relative path in the schema tree. It walks the schema tree towards the root,
// comparing each path element against nodes in the tree. It returns success
// only if all path elements are present as parent nodes in the schema tree.
func hasRelativePath(schema *yang.Entry, path []string) bool {
	s, p := schema, path
	for {
		if s == nil || len(p) == 0 {
			break
		}
		n := stripModulePrefix(p[len(p)-1])
		if s.Name != n {
			return false
		}
		s = s.Parent
		p = p[:len(p)-1]
	}

	return len(p) == 0
}

// enumStringToIntValue returns the integer value that enumerated string value
// of type fieldName maps to in the parent, which must be a struct ptr.
func enumStringToIntValue(parent interface{}, fieldName, value string) (int64, error) {
	v := reflect.ValueOf(parent)
	if !IsValueStructPtr(v) {
		return 0, fmt.Errorf("enumStringToIntValue: %T is not a struct ptr", parent)
	}
	field := v.Elem().FieldByName(fieldName)
	if !field.IsValid() {
		return 0, fmt.Errorf("%s is not a valid enum field name in %T", fieldName, parent)
	}

	mapMethod := reflect.New(field.Type()).Elem().MethodByName("ΛMap")
	if !mapMethod.IsValid() {
		return 0, fmt.Errorf("%s in %T does not have a ΛMap function", fieldName, parent)
	}

	ec := mapMethod.Call(nil)
	if len(ec) == 0 {
		return 0, fmt.Errorf("%s in %T ΛMap function returns empty value", fieldName, parent)
	}
	ei := ec[0].Interface()
	enumMap, ok := ei.(map[string]map[int64]ygot.EnumDefinition)
	if !ok {
		return 0, fmt.Errorf("%s in %T ΛMap function returned wrong type %T, want map[string]map[int64]ygot.EnumDefinition",
			fieldName, parent, ei)
	}

	m, ok := enumMap[field.Type().Name()]
	if !ok {
		return 0, fmt.Errorf("%s is not a valid enum field name, has type %s", fieldName, field.Type().Name())
	}

	for k, v := range m {
		if stripModulePrefix(v.Name) == stripModulePrefix(value) {
			return k, nil
		}
	}

	return 0, fmt.Errorf("%s is not a valid value for enum field %s", value, fieldName)
}

// yangBuiltinTypeToGoType returns a pointer to the Go built-in value with
// the type corresponding to the provided YANG type. It returns nil for any type
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
		log.Errorf("unexpected type %v in yangBuiltinTypeToGoPtrType", t)
	}
	return nil
}

// yangToJSONType returns the Go type that json.Unmarshal would render a value
// into for the given YANG leaf node schema type.
func yangToJSONType(t yang.TypeKind) reflect.Type {
	switch t {
	case yang.Yint8, yang.Yint16, yang.Yint32,
		yang.Yuint8, yang.Yuint16, yang.Yuint32:
		return reflect.TypeOf(float64(0))
	case yang.Ybinary, yang.Ydecimal64, yang.Yenum, yang.Yidentityref, yang.Yint64, yang.Yuint64, yang.Ystring:
		return reflect.TypeOf(string(""))
	case yang.Ybool, yang.Yempty:
		return reflect.TypeOf(bool(false))
	case yang.Yunion:
		return reflect.TypeOf(nil)
	default:
		// TODO(mostrowski): handle bitset.
		log.Errorf("unexpected type %v in yangToJSONType", t)
	}
	return reflect.TypeOf(nil)
}

// yangFloatIntToGoType the appropriate int type for YANG type t, set with value
// v, which must be within the allowed range for the type.
func yangFloatIntToGoType(t yang.TypeKind, v float64) (interface{}, error) {
	if err := checkJSONFloat64Range(t, v); err != nil {
		return nil, err
	}

	switch t {
	case yang.Yint8:
		return int8(v), nil
	case yang.Yint16:
		return int16(v), nil
	case yang.Yint32:
		return int32(v), nil
	case yang.Yuint8:
		return uint8(v), nil
	case yang.Yuint16:
		return uint16(v), nil
	case yang.Yuint32:
		return uint32(v), nil
	}
	return nil, fmt.Errorf("unexpected YANG type %v", t)
}

// checkJSONFloat64Range checks whether f is in range for the given YANG type.
func checkJSONFloat64Range(t yang.TypeKind, f float64) error {
	minMax := map[yang.TypeKind]struct {
		min int64
		max int64
	}{
		yang.Yint8:   {-128, 127},
		yang.Yint16:  {-32768, 32767},
		yang.Yint32:  {-2147483648, 2147483647},
		yang.Yuint8:  {0, 255},
		yang.Yuint16: {0, 65535},
		yang.Yuint32: {0, 4294967295},
	}

	if _, ok := minMax[t]; !ok {
		return fmt.Errorf("checkJSONFloat64Range bad YANG type %v", t)
	}
	if int64(f) < minMax[t].min || int64(f) > minMax[t].max {
		return fmt.Errorf("value %d falls outside the int range [%d, %d]", int64(f), minMax[t].min, minMax[t].max)
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

// yangKindToLeafEntry returns a leaf Entry with Type Kind set to k.
func yangKindToLeafEntry(k yang.TypeKind) *yang.Entry {
	return &yang.Entry{
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: k,
		},
	}
}

// stringMapKeys returns the keys for map m.
func stringMapKeys(m map[string]*yang.Entry) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// stringMapSetToSlice converts a string set expressed as a map m, into a slice
// of strings.
func stringMapSetToSlice(m map[string]interface{}) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// stripModulePrefixes returns "in" with each element with the format "A:B" changed
// to "B".
func stripModulePrefixes(in []string) []string {
	var out []string
	for _, v := range in {
		out = append(out, stripModulePrefix(v))
	}
	return out
}

// stripModulePrefix returns s with any prefix up to and including the last ':'
// character removed.
func stripModulePrefix(s string) string {
	sv := strings.Split(s, ":")
	return sv[len(sv)-1]
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
