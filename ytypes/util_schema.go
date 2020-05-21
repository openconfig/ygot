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
func validateListAttr(schema *yang.Entry, value interface{}) util.Errors {
	var errors []error
	if schema == nil {
		return util.NewErrs(fmt.Errorf("schema is nil"))
	}
	if schema.ListAttr == nil {
		return util.NewErrs(fmt.Errorf("schema %s ListAttr is nil", schema.Name))
	}

	var size int
	if value == nil {
		size = 0
	} else {
		switch reflect.TypeOf(value).Kind() {
		case reflect.Slice, reflect.Map:
			size = reflect.ValueOf(value).Len()
		default:
			return util.NewErrs(fmt.Errorf("value %v type %T must be map or slice type for schema %s", value, value, schema.Name))
		}
	}

	// If min/max element attr is present in the schema, this must be a list or
	// leaf-list. Check that the data tree falls within the required size
	// bounds.
	if v := schema.ListAttr.MinElements; v != nil {
		if minN, err := yang.ParseInt(v.Name); err != nil {
			errors = util.AppendErr(errors, err)
		} else if min, err := minN.Int(); err != nil {
			errors = util.AppendErr(errors, err)
		} else if min < 0 {
			errors = util.AppendErr(errors, fmt.Errorf("list %s has negative min required elements", schema.Name))
		} else if int64(size) < min {
			errors = util.AppendErr(errors, fmt.Errorf("list %s contains fewer than min required elements: %d < %d", schema.Name, size, min))
		}
	}
	if v := schema.ListAttr.MaxElements; v != nil {
		if maxN, err := yang.ParseInt(v.Name); err != nil {
			errors = util.AppendErr(errors, err)
		} else if max, err := maxN.Int(); err != nil {
			errors = util.AppendErr(errors, err)
		} else if max < 0 {
			errors = util.AppendErr(errors, fmt.Errorf("list %s has negative max required elements", schema.Name))
		} else if int64(size) > max {
			errors = util.AppendErr(errors, fmt.Errorf("list %s contains more than max allowed elements: %d > %d", schema.Name, size, max))
		}
	}

	return errors
}

// isValueScalar reports whether v is a scalar (non-composite) type.
func isValueScalar(v reflect.Value) bool {
	return !util.IsValueStruct(v) && !util.IsValueStructPtr(v) && !util.IsValueMap(v) && !util.IsValueSlice(v)
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
		pm[util.StripModulePrefix(sp[0])] = true
	}
	util.DbgSchema("check dataPaths %v against dataTree %v\n", pm, jsonTree)
	for jf := range jsonTree {
		if !pm[util.StripModulePrefix(jf)] {
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

// derefIfStructPtr returns the dereferenced reflect.Value of value if it is a
// ptr, or value if it is not.
func derefIfStructPtr(value reflect.Value) reflect.Value {
	if util.IsValueStructPtr(value) {
		return value.Elem()
	}
	return value
}
