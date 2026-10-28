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
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// SchemaPaths returns all the paths in the path tag.
func SchemaPaths(f reflect.StructField) ([][]string, error) {
	pathTag, ok := f.Tag.Lookup("path")
	if !ok || pathTag == "" {
		return nil, fmt.Errorf("field %s did not specify a path", f.Name)
	}
	return parseTagPaths(pathTag), nil
}

// ShadowSchemaPaths returns all the paths in the shadow-path tag. If the tag
// doesn't exist, a nil slice is returned.
func ShadowSchemaPaths(f reflect.StructField) [][]string {
	pathTag, ok := f.Tag.Lookup("shadow-path")
	if !ok || pathTag == "" {
		return nil
	}
	return parseTagPaths(pathTag)
}

// parseTagPaths splits a non-empty pipe-delimited struct tag into caller-owned
// slash-split path slices with module prefixes stripped from every element.
func parseTagPaths(pathTag string) [][]string {
	numPaths := strings.Count(pathTag, "|") + 1
	numElems := numPaths + strings.Count(pathTag, "/")
	allElems := make([]string, numElems)
	out := make([][]string, numPaths)

	elemIdx := 0
	remPaths := pathTag
	for pIdx := 0; pIdx < numPaths; pIdx++ {
		bar := strings.IndexByte(remPaths, '|')
		var p string
		if bar == -1 {
			p = remPaths
		} else {
			p = remPaths[:bar]
			remPaths = remPaths[bar+1:]
		}

		start := elemIdx
		remSegs := p
		for {
			slash := strings.IndexByte(remSegs, '/')
			if slash == -1 {
				allElems[elemIdx] = StripModulePrefix(remSegs)
				elemIdx++
				break
			}
			allElems[elemIdx] = StripModulePrefix(remSegs[:slash])
			elemIdx++
			remSegs = remSegs[slash+1:]
		}
		out[pIdx] = allElems[start:elemIdx:elemIdx]
	}
	return out
}

// RelativeSchemaPath returns a path to the schema for the struct field f.
// Paths are embedded in the "path" struct tag and can be either simple:
//
//	e.g. "path:a"
//
// or composite (if path compression is used) e.g.
//
//	e.g. "path:config/a|a"
//
// In the latter case, this function returns {"config", "a"}, because only the
// longer path exists in the data tree and we want the schema for that node.
// This case is found in OpenConfig leaf-ref cases where the key of a list is a
// leafref; the schema *yang.Entry for the field is given by
// schema.Dir["config"].Dir["a"].
func RelativeSchemaPath(f reflect.StructField) ([]string, error) {
	return relativeSchemaPath(f, false)
}

// RelativeSchemaPathPreferShadow returns a shadow path (if exists) or path to
// the schema for the struct field f.
//
// Paths are embedded in the "shadow-path" and "path" struct tags and can be
// either simple:
//
//	e.g. "path:a"
//
// or composite (if path compression is used) e.g.
//
//	e.g. "path:config/a|a"
//
// In the latter case, this function returns {"config", "a"}, because only the
// longer path exists in the data tree and we want the schema for that node.
// This case is found in OpenConfig leaf-ref cases where the key of a list is a
// leafref; the schema *yang.Entry for the field is given by
// schema.Dir["config"].Dir["a"].
func RelativeSchemaPathPreferShadow(f reflect.StructField) ([]string, error) {
	return relativeSchemaPath(f, true)
}

// relativeSchemaPath returns a path to the schema for the struct field f.
// Paths are embedded in the "path" struct tag and can be either simple:
//
//	e.g. "path:a"
//
// or composite (if path compression is used) e.g.
//
//	e.g. "path:config/a|a"
//
// In the latter case, this function returns {"config", "a"}, because only the
// longer path exists in the data tree and we want the schema for that node.
// This case is found in OpenConfig leaf-ref cases where the key of a list is a
// leafref; the schema *yang.Entry for the field is given by
// schema.Dir["config"].Dir["a"].
//
// If preferShadowPath is false, the path values from the "path" tag are used.
// If preferShadowPath is true and the field has a "shadow-path" tag, then the
// path values from the "shadow-path" tag are used; if the field doesn't have
// the "shadow-path" tag, then the path values from the "path" tag are used.
func relativeSchemaPath(f reflect.StructField, preferShadowPath bool) ([]string, error) {
	selected, err := selectRelativeSchemaPathStr(f, preferShadowPath)
	if err != nil {
		return nil, err
	}
	return splitSlashPath(selected), nil
}

// splitSlashPath splits s across '/' into a caller-owned slice sized to the
// exact element count.
func splitSlashPath(s string) []string {
	n := strings.Count(s, "/") + 1
	out := make([]string, n)
	rem := s
	for i := 0; i < n-1; i++ {
		slash := strings.IndexByte(rem, '/')
		out[i] = rem[:slash]
		rem = rem[slash+1:]
	}
	out[n-1] = rem
	return out
}

// SchemaTreePath returns the schema tree path of the supplied yang.Entry
// skipping any nodes that are themselves not in the path (e.g., choice
// and case). The path is returned as a string prefixed with the module
// name (similarly to the behaviour of (*yang.Entry).Path()).
func SchemaTreePath(e *yang.Entry) string {
	return SlicePathToString(append([]string{""}, SchemaPathNoChoiceCase(e)...))
}

// SchemaTreePathNoModule takes an input yang.Entry, and returns its YANG schema
// path. The returned schema path does not include the root module name.
func SchemaTreePathNoModule(e *yang.Entry) string {
	return SlicePathToString(append([]string{""}, SchemaPathNoChoiceCase(e)[1:]...))
}

// SchemaPathNoChoiceCase takes an input yang.Entry and walks up the tree to find
// its path, expressed as a slice of strings, which is returned.
func SchemaPathNoChoiceCase(elem *yang.Entry) []string {
	var pp []string
	for _, e := range SchemaEntryPathNoChoiceCase(elem) {
		pp = append(pp, e.Name)
	}
	return pp
}

// SchemaEntryPathNoChoiceCase takes an input yang.Entry and walks up the tree to find
// its path, expressed as a slice of Entrys, which is returned.
func SchemaEntryPathNoChoiceCase(elem *yang.Entry) []*yang.Entry {
	var pp []*yang.Entry
	if elem == nil {
		return pp
	}
	e := elem
	for ; e.Parent != nil; e = e.Parent {
		if !IsChoiceOrCase(e) {
			pp = append(pp, e)
		}
	}
	pp = append(pp, e)

	// Reverse the slice that was specified to us as it was appended to
	// from the leaf to the root.
	for i := len(pp)/2 - 1; i >= 0; i-- {
		o := len(pp) - 1 - i
		pp[i], pp[o] = pp[o], pp[i]
	}
	return pp
}

// FirstChild returns the first child entry that matches path from the given
// root. When comparing the path, only nodes that appear in the data tree
// are considered. It returns nil if no node matches the path.
func FirstChild(schema *yang.Entry, path []string) *yang.Entry {
	if schema == nil || len(path) == 0 {
		return nil
	}
	first := StripModulePrefix(path[0])
	if ch, ok := schema.Dir[first]; ok && !IsChoiceOrCase(ch) && ch.Name == first {
		if m := firstMatchingStripped(ch, path[1:]); m != nil {
			return m
		}
	}
	return firstChildSlow(schema, first, path[1:])
}

// firstChildSlow searches schema.Dir recursively through any choice or case
// subtrees for the first non-choice, non-case entry matching first followed by
// rem.
func firstChildSlow(schema *yang.Entry, first string, rem []string) *yang.Entry {
	for _, ch := range schema.Dir {
		switch {
		case !IsChoiceOrCase(ch):
			if ch.Name == first {
				if m := firstMatchingStripped(ch, rem); m != nil {
					return m
				}
			}
		case ch.IsDir():
			if m := firstChildSlow(ch, first, rem); m != nil {
				return m
			}
		}
	}
	return nil
}

// firstMatchingStripped traverses s.Dir along rem, stripping any module prefix
// from each element, and returns the terminal *yang.Entry or nil if absent.
func firstMatchingStripped(s *yang.Entry, rem []string) *yang.Entry {
	for i := 0; i < len(rem); i++ {
		s = s.Dir[StripModulePrefix(rem[i])]
		if s == nil {
			return nil
		}
	}
	return s
}

// removeXPATHPredicates removes predicates from an XPath string. e.g.,
// removeXPATHPredicates(/foo/bar[name="foo"]/config/baz -> /foo/bar/config/baz.
func removeXPATHPredicates(s string) (string, error) {
	var b bytes.Buffer
	for i := 0; i < len(s); {
		ss := s[i:]
		si, ei := strings.Index(ss, "["), strings.Index(ss, "]")
		switch {
		case si == -1 && ei == -1:
			// This substring didn't contain a [] pair, therefore write it
			// to the buffer.
			b.WriteString(ss)
			// Move to the last character of the substring.
			i += len(ss)
		case si == -1 || ei == -1:
			// This substring contained a mismatched pair of []s.
			return "", fmt.Errorf("mismatched brackets within substring %s of %s, [ pos: %d, ] pos: %d", ss, s, si, ei)
		case si > ei:
			// This substring contained a ] before a [.
			return "", fmt.Errorf("incorrect ordering of [] within substring %s of %s, [ pos: %d, ] pos: %d", ss, s, si, ei)
		default:
			// This substring contained a matched set of []s.
			b.WriteString(ss[0:si])
			i += ei + 1
		}
	}

	return b.String(), nil
}

// FindLeafRefSchema returns a schema Entry at the path pathStr relative to
// schema if it exists, or an error otherwise.
// pathStr has either:
//   - the relative form "../a/b/../b/c", where ".." indicates the parent of the
//     node, or
//   - the absolute form "/a/b/c", which indicates the absolute path from the
//     root of the schema tree.
func FindLeafRefSchema(schema *yang.Entry, pathStr string) (*yang.Entry, error) {
	if pathStr == "" {
		return nil, fmt.Errorf("leafref schema %s has empty path", schema.Name)
	}

	refSchema := schema
	pathStr, err := removeXPATHPredicates(pathStr)
	if err != nil {
		return nil, err
	}
	path := strings.Split(pathStr, "/")

	// For absolute path, reset to root of the schema tree.
	if pathStr[0] == '/' {
		refSchema = TopLevelModule(schema)
		path = path[1:]
	}

	for i := 0; i < len(path); i++ {
		pe, err := stripModulePrefixWithCheck(path[i])
		if err != nil {
			return nil, fmt.Errorf("leafref schema %s path %s: %v", schema.Name, pathStr, err)
		}

		if pe == ".." {
			if refSchema.Parent == nil {
				return nil, fmt.Errorf("parent of %s is nil for leafref schema %s with path %s", refSchema.Name, schema.Name, pathStr)
			}
			for refSchema = refSchema.Parent; IsChoiceOrCase(refSchema); refSchema = refSchema.Parent {
			}
			continue
		}
		entries, err := findFirstNonChoiceOrCaseEntry(refSchema)
		if err != nil {
			return nil, err
		}
		if entries[pe] == nil {
			return nil, fmt.Errorf("schema node %s is nil for leafref schema %s with path %s", pe, schema.Name, pathStr)
		}
		refSchema = entries[pe]
	}

	return refSchema, nil
}

// StripModulePrefixesStr returns "in" with each element with the format "A:B"
// changed to "B".
func StripModulePrefixesStr(in string) string {
	return strings.Join(stripModulePrefixes(strings.Split(in, "/")), "/")
}

// stripModulePrefixes returns "in" with each element with the format "A:B"
// changed to "B".
func stripModulePrefixes(in []string) []string {
	var out []string
	for _, v := range in {
		out = append(out, StripModulePrefix(v))
	}
	return out
}

// stripModulePrefixWithCheck removes the prefix from a YANG path element, and
// returns an error for unexpected formats. For example, removing foo from
// "foo:bar".  Such qualified paths are used in YANG modules where remote paths
// are referenced.
func stripModulePrefixWithCheck(name string) (string, error) {
	idx := strings.IndexByte(name, ':')
	switch {
	case idx == -1:
		return name, nil
	case strings.IndexByte(name[idx+1:], ':') == -1:
		return name[idx+1:], nil
	default:
		return "", fmt.Errorf("path element did not form a valid name (name, prefix:name): %v", name)
	}
}

// StripModulePrefix removes the prefix from a YANG path element, and
// the string format is invalid, simply returns the argument. For example,
// removing foo from "foo:bar". Such qualified paths are used in YANG modules
// where remote paths are referenced.
func StripModulePrefix(name string) string {
	idx := strings.IndexByte(name, ':')
	if idx == -1 || strings.IndexByte(name[idx+1:], ':') != -1 {
		return name
	}
	return name[idx+1:]
}

// ReplacePathSuffix replaces the non-prefix part of a prefixed path name, or
// the whole path name otherwise.
// e.g. If replacing foo -> bar
// - "foo" becomes "bar"
// - "a:foo" becomes "a:bar"
func ReplacePathSuffix(name, newSuffix string) (string, error) {
	ps := strings.Split(name, ":")
	switch len(ps) {
	case 1:
		return newSuffix, nil
	case 2:
		return ps[0] + ":" + newSuffix, nil
	}
	return "", fmt.Errorf("ygot.util: path element did not form a valid name (name, prefix:name): %q", name)
}

// PathStringToElements splits the string s, which represents a gNMI string
// path into its constituent elements. It does not parse keys, which are left
// unchanged within the path - but removes escape characters from element
// names. The path returned omits any leading or trailing empty elements when
// splitting on the / character.
func PathStringToElements(path string) []string {
	parts := SplitPath(path)
	// Remove leading empty element
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	// Remove trailing empty element
	if len(parts) > 0 && path[len(path)-1] == '/' {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// SplitPath splits path across unescaped /.
// Any / inside square brackets are ignored.
func SplitPath(path string) []string {
	var parts []string
	var buf bytes.Buffer

	var inKey, inEscape bool

	var ch rune
	for _, ch = range path {
		switch {
		case ch == '[' && !inEscape:
			inKey = true
		case ch == ']' && !inEscape:
			inKey = false
		case ch == '\\' && !inEscape && !inKey:
			inEscape = true
			continue
		case ch == '/' && !inEscape && !inKey:
			parts = append(parts, buf.String())
			buf.Reset()
			continue
		}

		buf.WriteRune(ch)
		inEscape = false
	}

	if buf.Len() != 0 || (len(path) != 1 && ch == '/') {
		parts = append(parts, buf.String())
	}

	return parts
}

// SlicePathToString concatenates a slice of strings into a / separated path. i.e.,
// []string{"", "foo", "bar"} becomes "/foo/bar". Paths in YANG are generally
// represented in this return format, but the []string format is more flexible
// for internal use.
func SlicePathToString(parts []string) string {
	var buf bytes.Buffer
	for i, p := range parts {
		buf.WriteString(p)
		if i != len(parts)-1 {
			buf.WriteRune('/')
		}
	}
	return buf.String()
}
