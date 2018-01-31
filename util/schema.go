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

// IsLeafRef reports whether schema is a leafref schema node type.
func IsLeafRef(schema *yang.Entry) bool {
	if schema == nil || schema.Type == nil {
		return false
	}
	return schema.Type.Kind == yang.Yleafref
}

// IsChoiceOrCase returns true if the entry is either a 'case' or a 'choice'
// node within the schema. These are schema nodes only, and the code generation
// operates on data tree paths.
func IsChoiceOrCase(e *yang.Entry) bool {
	if e == nil {
		return false
	}
	return e.IsChoice() || e.IsCase()
}

// IsFakeRoot reports whether the supplied yang.Entry represents the synthesised
// root entity in the generated code.
func IsFakeRoot(e *yang.Entry) bool {
	if e == nil {
		return false
	}
	if _, ok := e.Annotation["isFakeRoot"]; ok {
		return true
	}
	return false
}

// IsUnkeyedList reports whether e is an unkeyed list.
func IsUnkeyedList(e *yang.Entry) bool {
	if e == nil {
		return false
	}
	return e.IsList() && e.Key == ""
}

// IsYgotAnnotation reports whether struct field s is an annotation field.
func IsYgotAnnotation(s reflect.StructField) bool {
	_, ok := s.Tag.Lookup("ygotAnnotation")
	return ok
}

// SchemaPaths returns all the paths in the path tag.
func SchemaPaths(f reflect.StructField) ([][]string, error) {
	var out [][]string
	pathTag, ok := f.Tag.Lookup("path")
	if !ok || pathTag == "" {
		return nil, fmt.Errorf("field %s did not specify a path", f.Name)
	}

	ps := strings.Split(pathTag, "|")
	for _, p := range ps {
		out = append(out, StripModulePrefixes(strings.Split(p, "/")))
	}
	return out, nil
}

// ChildSchema returns the first child schema that matches path from the given
// schema root. When comparing the path, only nodes that appear in the data tree
// are considered. It returns nil if no node matches the path.
func ChildSchema(schema *yang.Entry, path []string) *yang.Entry {
	path = StripModulePrefixes(path)
	entries := FindFirstNonChoiceOrCase(schema)

	for _, e := range entries {
		m := MatchingNonChoiceCaseSchema(e, path)
		if m != nil {
			return m
		}
	}

	return nil
}

// FindFirstNonChoiceOrCase recursively traverses the schema tree and returns a
// map with the set of the first nodes in every path that are neither case nor
// choice nodes. The keys in the map are the paths to the matching elements
// from the parent data struct, which always have length 1.
func FindFirstNonChoiceOrCase(e *yang.Entry) map[string]*yang.Entry {
	m := make(map[string]*yang.Entry)
	for _, ch := range e.Dir {
		addToEntryMap(m, findFirstNonChoiceOrCaseInternal(ch))
	}
	return m
}

// findFirstNonChoiceOrCaseInternal is an internal part of
// FindFirstNonChoiceOrCase.
func findFirstNonChoiceOrCaseInternal(e *yang.Entry) map[string]*yang.Entry {
	m := make(map[string]*yang.Entry)
	switch {
	case !IsChoiceOrCase(e):
		m[e.Name] = e
	case e.IsDir():
		for _, ch := range e.Dir {
			addToEntryMap(m, findFirstNonChoiceOrCaseInternal(ch))
		}
	}
	return m
}

// addToEntryMap merges from into to.
func addToEntryMap(to, from map[string]*yang.Entry) map[string]*yang.Entry {
	for k, v := range from {
		to[k] = v
	}
	return to
}

// MatchingNonChoiceCaseSchema returns the child schema at the given path from
// schema if one is found, or nil otherwise.
func MatchingNonChoiceCaseSchema(schema *yang.Entry, path []string) *yang.Entry {
	s := schema
	if len(path) == 0 || schema.Name != path[0] {
		return nil
	}
	path = path[1:]
	for i := 0; i < len(path); i++ {
		if s = s.Dir[path[i]]; s == nil {
			return nil
		}
	}
	return s
}

// ResolveIfLeafRef returns a ptr to the schema pointed to by the leaf-ref path
// in schema if it's a leafref, or schema itself if it's not.
func ResolveIfLeafRef(schema *yang.Entry) (*yang.Entry, error) {
	if schema == nil {
		return nil, nil
	}
	// fakeroot or test cases may have this unset. They are definitely not
	// leafrefs.
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
		DbgPrint("follow schema leaf-ref from %s to %s, type %v", orig.Name, s.Name, s.Type.Kind)
	}
	return s, nil
}

// findLeafRefSchema returns a schema Entry at the path pathStr relative to
// schema if it exists, or an error otherwise.
// pathStr has either:
//  - the relative form "../a/b/../b/c", where ".." indicates the parent of the
//    node, or
//  - the absolute form "/a/b/c", which indicates the absolute path from the
//    root of the schema tree.
func findLeafRefSchema(schema *yang.Entry, pathStr string) (*yang.Entry, error) {
	if pathStr == "" {
		return nil, fmt.Errorf("leafref schema %s has empty path", schema.Name)
	}

	refSchema := schema
	path := strings.Split(pathStr, "/")

	// For absolute path, reset to root of the schema tree.
	if pathStr[0] == '/' {
		refSchema = schemaTreeRoot(schema)
		path = path[1:]
	}

	for i := 0; i < len(path); i++ {
		pe := StripModulePrefix(path[i])
		if pe == ".." {
			if refSchema.Parent == nil {
				return nil, fmt.Errorf("parent of %s is nil for leafref schema %s with path %s", refSchema.Name, schema.Name, pathStr)
			}
			refSchema = refSchema.Parent
			continue
		}
		if refSchema.Dir[pe] == nil {
			return nil, fmt.Errorf("schema node %s is nil for leafref schema %s with path %s", pe, schema.Name, pathStr)
		}
		refSchema = refSchema.Dir[pe]
	}

	return refSchema, nil
}

// FieldSchema returns the schema for the struct field f, if f contains a valid
// path tag and the schema path is found in the schema tree. It returns an error
// if the struct tag is invalid, or nil if tag is valid but the schema is not
// found in the tree at the specified path.
func FieldSchema(schema *yang.Entry, f reflect.StructField) (*yang.Entry, error) {
	DbgSchema("FieldSchema for parent schema %s, struct field %s\n", schema.Name, f.Name)
	p, err := pathToSchema(f)
	if err != nil {
		return nil, err
	}
	p = StripModulePrefixes(p)
	DbgSchema("pathToSchema yields %v\n", p)
	s := schema
	found := true
	DbgSchema("traversing schema Dirs...")
	for ; len(p) > 0; p = p[1:] {
		DbgSchema("/%s", p[0])
		var ok bool
		s, ok = s.Dir[p[0]]
		if !ok {
			found = false
			break
		}
	}
	if found {
		DbgSchema(" - found\n")
		return s, nil
	}
	DbgSchema(" - not found\n")

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
	entries := FindFirstNonChoiceOrCase(schema)

	DbgSchema("checking for %s against non choice/case entries: %v\n", p[0], stringMapKeys(entries))
	for pe, entry := range entries {
		DbgSchema("%s ? ", pe)
		if pe == p[0] {
			DbgSchema(" - match\n")
			return entry, nil
		}
	}

	DbgSchema(" - no matches\n")
	return nil, nil
}

// pathToSchema returns a path to the schema for the struct field f.
// Paths are embedded in the "path" struct tag and can be either simple:
//   e.g. "path:a"
// or composite (if path compression is used) e.g.
//   e.g. "path:config/a|a"
// In the latter case, this function returns {"config", "a"}, because only
// the longer path exists in the data tree and we want the schema for that
// node.
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

// StripModulePrefixesStr returns "in" with each element with the format "A:B"
// changed to "B".
func StripModulePrefixesStr(in string) string {
	return strings.Join(StripModulePrefixes(strings.Split(in, "/")), "/")
}

// StripModulePrefixes returns "in" with each element with the format "A:B"
// changed to "B".
func StripModulePrefixes(in []string) []string {
	var out []string
	for _, v := range in {
		out = append(out, StripModulePrefix(v))
	}
	return out
}

// StripModulePrefix returns s with any prefix up to and including the last ':'
// character removed.
func StripModulePrefix(s string) string {
	sv := strings.Split(s, ":")
	return sv[len(sv)-1]
}
