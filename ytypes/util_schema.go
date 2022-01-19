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
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
)

//lint:file-ignore U1000 Ignore all unused code, it represents generated code.

// validateLengthSchema validates whether the given schema has a valid length
// specification.
func validateLengthSchema(schema *yang.Entry) error {
	if len(schema.Type.Length) == 0 {
		return nil
	}
	for _, r := range schema.Type.Length {
		// This is a limited check. It's assumed that a full check is
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
func validateListAttr(schema *yang.Entry, value interface{}) util.Errors {
	var errors []error
	if schema == nil {
		return util.NewErrs(fmt.Errorf("schema is nil"))
	}
	if schema.ListAttr == nil {
		return util.NewErrs(fmt.Errorf("schema %s ListAttr is nil", schema.Name))
	}

	var size uint64
	if value != nil {
		switch reflect.TypeOf(value).Kind() {
		case reflect.Slice, reflect.Map:
			size = uint64(reflect.ValueOf(value).Len())
		default:
			return util.NewErrs(fmt.Errorf("value %v type %T must be map or slice type for schema %s", value, value, schema.Name))
		}
	}

	// If min/max element attr is present in the schema, this must be a list or
	// leaf-list. Check that the data tree falls within the required size
	// bounds.
	if size < schema.ListAttr.MinElements {
		errors = util.AppendErr(errors, fmt.Errorf("list %s contains fewer than min required elements: %d < %d", schema.Name, size, schema.ListAttr.MinElements))
	}
	// 0 is an invalid value for MaxElements
	// (https://tools.ietf.org/html/rfc7950#section-7.7.6).
	// For useability it best represents the value "unbounded".
	if schema.ListAttr.MaxElements != 0 && size > schema.ListAttr.MaxElements {
		errors = util.AppendErr(errors, fmt.Errorf("list %s contains more than max allowed elements: %d > %d", schema.Name, size, schema.ListAttr.MaxElements))
	}
	return errors
}

// absoluteSchemaDataPath returns the absolute path of the schema, excluding
// any choice or case entries. Choice and case are excluded since they exist
// neither within the data or schema tree.
func absoluteSchemaDataPath(schema *yang.Entry) string {
	out := []string{schema.Name}
	for s := schema.Parent; s != nil; s = s.Parent {
		if !util.IsChoiceOrCase(s) && !util.IsFakeRoot(s) {
			out = append([]string{s.Name}, out...)
		}
	}

	return "/" + strings.Join(out, "/")
}

// pathTagFromField extracts the "path" tag from the struct field f,
// if no tag is found an error is returned. If the tag consts of more
// than one path, they are returned exactly as they are specified in
// the input struct - i.e., separated by "|".
func pathTagFromField(f reflect.StructField) (string, error) {
	pathAnnotation, ok := f.Tag.Lookup("path")
	if !ok {
		return "", fmt.Errorf("field %s did not specify a path", f.Name)
	}
	return pathAnnotation, nil
}

// directDescendantSchema returns the direct descendant schema for the struct
// field f. Paths are embedded in the "path" struct tag and can be either simple:
//   e.g. "path:a"
// or composite e.g.
//   e.g. "path:config/a|a"
// Function checks for presence of first schema without '/' and returns it.
func directDescendantSchema(f reflect.StructField) (string, error) {
	pathAnnotation, err := pathTagFromField(f)
	if err != nil {
		return "", err
	}
	paths := strings.Split(pathAnnotation, "|")

	for _, pth := range paths {
		if len(strings.Split(pth, "/")) == 1 {
			return pth, nil
		}
	}
	return "", fmt.Errorf("failed to find a schema path for %v", f)
}

// dataTreePaths returns all the data tree paths corresponding to schemaPaths.
// Any intermediate nodes not found in the data tree (i.e. choice/case) are
// removed from the paths.
func dataTreePaths(parentSchema, schema *yang.Entry, f reflect.StructField) ([][]string, error) {
	out, err := util.SchemaPaths(f)
	if err != nil {
		return nil, err
	}
	n, err := removeNonDataPathElements(parentSchema, schema, out)
	util.DbgPrint("have paths %v, removing non-data from %s -> %v", out, schema.Name, n)
	return n, err
}

// shadowDataTreePaths returns all the shadow data tree paths corresponding to schemaPaths.
// Any intermediate nodes not found in the data tree (i.e. choice/case) are
// removed from the paths.
func shadowDataTreePaths(parentSchema, schema *yang.Entry, f reflect.StructField) ([][]string, error) {
	out := util.ShadowSchemaPaths(f)
	n, err := removeNonDataPathElements(parentSchema, schema, out)
	util.DbgPrint("have shadow paths %v, removing non-data from %s -> %v", out, schema.Name, n)
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
			if !util.IsChoiceOrCase(s) {
				po = append(po, pe)
			}
		}
		out = append(out, po)
	}

	return out, nil
}

// checkDataTreeAgainstPaths checks that all paths that are defined in jsonTree match at least one of
// the paths that are supplied in dataPaths. A match exists when a path prefix of a jsonTree element is
// equal to one or more of the given dataPaths. Since each dataPath is checked as a valid prefix, the
// maximum depth of the check is limited to the length of the dataPaths specified. For example, if the jsonTree
// contains an element which has the path /foo/bar/baz, and dataPaths specifies /foo, then only /foo is checked
// to be a valid path, no assertions are made about the validity of 'bar' as a child of 'foo'.
//
// checkDataTreePaths returns an error if there are fields that are in the JSON that are not specified in the dataPaths.
func checkDataTreeAgainstPaths(jsonTree map[string]interface{}, dataPaths [][]string) error {
	// Primarily, we build a trie that consists of all the valid paths that we were provided
	// in the dataPaths tree.
	tree := map[string]interface{}{}
	for _, ch := range dataPaths {
		parent := tree
		for i := 0; i < len(ch)-1; i++ {
			chn := util.StripModulePrefix(ch[i])
			if parent[chn] == nil {
				parent[chn] = map[string]interface{}{}
			}
			parent = parent[chn].(map[string]interface{})
		}
		parent[util.StripModulePrefix(ch[len(ch)-1])] = true
	}

	var missingKeys []string
	var unexpectedLeafNodes []string
	// We have to define the function up-front so that we can recursively call the
	// anonymous function.
	var checkTree func(map[string]interface{}, map[string]interface{})
	checkTree = func(jsonTree map[string]interface{}, keyTree map[string]interface{}) {
		for key := range jsonTree {
			shortKey := util.StripModulePrefix(key)
			if _, ok := keyTree[shortKey]; !ok {
				missingKeys = append(missingKeys, shortKey)
			}
			if ct, ok := keyTree[shortKey].(map[string]interface{}); ok {
				// If this is a non-leaf node for keyTree, then
				// it should also be a non-leaf node for jsonTree.
				// The converse is not true, since keyTree is
				// just a partial path.
				if jt, ok := jsonTree[key].(map[string]interface{}); ok {
					checkTree(jt, ct)
				} else {
					unexpectedLeafNodes = append(unexpectedLeafNodes, shortKey)
				}
			}
		}
	}
	checkTree(jsonTree, tree)
	switch len(missingKeys) {
	case 0:
	case 1:
		// Retain backwards compatibility with previous implementation that reported
		// only the first error key.
		return fmt.Errorf("JSON contains unexpected field %s", missingKeys[0])
	default:
		sort.Strings(missingKeys)
		return fmt.Errorf("JSON contains unexpected field %v", missingKeys)
	}

	if len(unexpectedLeafNodes) != 0 {
		return fmt.Errorf("JSON contains unexpected leaf field(s) %v at non-leaf node", unexpectedLeafNodes)
	}
	return nil
}

// schemaToStructFieldName returns the string name of the field, which must be
// contained in parent (a struct ptr), given the schema for the field.
// It returns empty string and nil error if the field does not exist in the
// parent struct.
func schemaToStructFieldName(schema *yang.Entry, parent interface{}) (string, *yang.Entry, error) {

	v := reflect.ValueOf(parent)
	if util.IsNilOrInvalidValue(v) {
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
		p, err := util.RelativeSchemaPath(f)
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
		pe := util.StripModulePrefix(path[i])
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
		n := util.StripModulePrefix(p[len(p)-1])
		if s.Name != n {
			return false
		}
		s = s.Parent
		p = p[:len(p)-1]
	}

	return len(p) == 0
}
