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

// Package util implements utlity functions not specific to any ygot package.
package util

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

var (
	// YangMaxNumber represents the maximum value for any integer type.
	YangMaxNumber = yang.Number{Kind: yang.MaxNumber}
	// YangMinNumber represents the minimum value for any integer type.
	YangMinNumber = yang.Number{Kind: yang.MinNumber}
)

// CompressedSchemaAnnotation stores the name of the annotation indicating
// whether a set of structs were built with -compress_path. It is appended
// to the yang.Entry struct of the root entity of the structs within the
// SchemaTree.
const CompressedSchemaAnnotation string = "isCompressedSchema"

// Children returns all child elements of a directory element e that are not
// RPC entries.
func Children(e *yang.Entry) []*yang.Entry {
	var entries []*yang.Entry

	for _, e := range e.Dir {
		if e.RPC == nil {
			entries = append(entries, e)
		}
	}
	return entries
}

// SchemaTreeRoot returns the root of the schema tree, given any node in that
// tree. It returns nil if schema is nil.
func SchemaTreeRoot(schema *yang.Entry) *yang.Entry {
	if schema == nil {
		return nil
	}

	root := schema
	for root.Parent != nil {
		root = root.Parent
	}

	return root
}

// HasOnlyChild returns true if the directory passed to it only has a single
// element below it.
func HasOnlyChild(e *yang.Entry) bool {
	return e.Dir != nil && len(Children(e)) == 1
}

// IsRoot returns true if the entry is an entity at the root of the tree.
func IsRoot(e *yang.Entry) bool {
	return e.Parent == nil
}

// IsConfigState returns true if the entry is an entity that represents a
// container called config or state.
func IsConfigState(e *yang.Entry) bool {
	return e.IsDir() && (e.Name == "config" || e.Name == "state")
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

// IsUnionType returns true if the entry is a union within the YANG schema,
// checked by determining the length of the Type slice within the YangType.
func IsUnionType(t *yang.YangType) bool {
	if t == nil {
		return false
	}
	return len(t.Type) > 0
}

// IsEnumeratedType returns true if the entry is an enumerated type within the
// YANG schema - i.e., an enumeration or identityref leaf.
func IsEnumeratedType(t *yang.YangType) bool {
	if t == nil {
		return false
	}
	return t.Kind == yang.Yenum || t.Kind == yang.Yidentityref
}

// IsAnydata returns true if the entry is an Anydata node.
func IsAnydata(e *yang.Entry) bool {
	if e == nil {
		return false
	}
	return e.Kind == yang.AnyDataEntry
}

// IsLeafRef reports whether schema is a leafref schema node type.
func IsLeafRef(schema *yang.Entry) bool {
	if schema == nil || schema.Type == nil {
		return false
	}
	return schema.Type.Kind == yang.Yleafref
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

// IsKeyedList returns true if the supplied yang.Entry represents a keyed list.
func IsKeyedList(e *yang.Entry) bool {
	if e == nil {
		return false
	}
	return e.IsList() && e.Key != ""
}

// IsUnkeyedList reports whether e is an unkeyed list.
func IsUnkeyedList(e *yang.Entry) bool {
	if e == nil {
		return false
	}
	return e.IsList() && e.Key == ""
}

// IsOCCompressedValidElement returns true if the element would be output in the
// compressed YANG code.
func IsOCCompressedValidElement(e *yang.Entry) bool {
	switch {
	case HasOnlyChild(e) && Children(e)[0].IsList():
		// This is a surrounding container for a list which is removed from the
		// structure.
		return false
	case IsRoot(e):
		// This is a top-level module within the goyang structure, so is not output
		return false
	case IsConfigState(e):
		// This is a container that is called config or state, which is removed from
		// a compressed OpenConfig schema.
		return false
	case IsChoiceOrCase(e):
		// This is a choice or case node that is removed from the overall schema
		// so code generation does not occur for it.
		return false
	}
	return true
}

// IsCompressedSchema determines whether the yang.Entry s provided is part of a
// generated set of structs that have schema compression enabled. It traverses
// to the schema root, and determines the presence of an annotation with the name
// CompressedSchemaAnnotation which is added by ygen.
func IsCompressedSchema(s *yang.Entry) bool {
	var e *yang.Entry
	for e = s; e.Parent != nil; e = e.Parent {
	}
	_, ok := e.Annotation[CompressedSchemaAnnotation]
	return ok
}

// IsYgotAnnotation reports whether struct field s is an annotation field.
func IsYgotAnnotation(s reflect.StructField) bool {
	_, ok := s.Tag.Lookup("ygotAnnotation")
	return ok
}

// IsSimpleEnumerationType returns true when the type supplied is a simple
// enumeration (i.e., a leaf that is defined as type enumeration { ... },
// and is not a typedef that contains an enumeration, or a union that
// contains an enumeration which may have enum values specified. The type
// name enumeration is used in these cases by goyang.
func IsSimpleEnumerationType(t *yang.YangType) bool {
	if t == nil {
		return false
	}
	return t.Kind == yang.Yenum && t.Name == yang.TypeKindToName[yang.Yenum]
}

// IsIdentityrefLeaf returns true if the supplied yang.Entry represents an
// identityref.
func IsIdentityrefLeaf(e *yang.Entry) bool {
	return e.Type.IdentityBase != nil
}

// IsYANGBaseType determines whether the supplied YangType is a built-in type
// in YANG, or a derived type (i.e., typedef).
func IsYANGBaseType(t *yang.YangType) bool {
	_, ok := yang.TypeKindFromName[t.Name]
	return ok
}

// SanitizedPattern returns the values of the posix-pattern extension
// statements for the YangType. If it's empty, then it returns the values from
// the pattern statements with anchors attached (if missing).
// It also returns whether the patterns are POSIX.
func SanitizedPattern(t *yang.YangType) ([]string, bool) {
	if len(t.POSIXPattern) != 0 {
		return t.POSIXPattern, true
	}
	var pat []string
	for _, p := range t.Pattern {
		// fixYangRegexp adds ^(...)$ around the pattern - the result is
		// equivalent to a full match of whole string.
		pat = append(pat, fixYangRegexp(p))
	}
	return pat, false
}

// fixYangRegexp takes a pattern regular expression from a YANG module and
// returns it into a format which can be used by the Go regular expression
// library. YANG uses a W3C standard that is defined to be implicitly anchored
// at the head or tail of the expression. See
// https://www.w3.org/TR/2004/REC-xmlschema-2-20041028/#regexs for details.
func fixYangRegexp(pattern string) string {
	var buf bytes.Buffer
	var inEscape bool
	var prevChar rune
	addParens := false

	for i, ch := range pattern {
		if i == 0 && ch != '^' {
			buf.WriteRune('^')
			// Add parens around entire expression to prevent logical
			// subexpressions associating with leading/trailing ^ / $.
			buf.WriteRune('(')
			addParens = true
		}

		switch ch {
		case '$':
			// Dollar signs need to be escaped unless they are at
			// the end of the pattern, or are already escaped.
			if !inEscape && i != len(pattern)-1 {
				buf.WriteRune('\\')
			}
		case '^':
			// Carets need to be escaped unless they are already
			// escaped, indicating set negation ([^.*]) or at the
			// start of the string.
			if !inEscape && prevChar != '[' && i != 0 {
				buf.WriteRune('\\')
			}
		}

		// If the previous character was an escape character, then we
		// leave the escape, otherwise check whether this is an escape
		// char and if so, then enter escape.
		inEscape = !inEscape && ch == '\\'

		if i == len(pattern)-1 && addParens && ch == '$' {
			buf.WriteRune(')')
		}

		buf.WriteRune(ch)

		if i == len(pattern)-1 && ch != '$' {
			if addParens {
				buf.WriteRune(')')
			}
			buf.WriteRune('$')
		}

		prevChar = ch
	}

	return buf.String()
}

// IsConfig takes a yang.Entry and traverses up the tree to find the config
// state of that element. In YANG, if the config parameter is unset, then it is
// is inherited from the parent of the element - hence we must walk up the tree to find
// the state. If the element at the top of the tree does not have config set, then config
// is true. See https://tools.ietf.org/html/rfc6020#section-7.19.1.
func IsConfig(e *yang.Entry) bool {
	for ; e.Parent != nil; e = e.Parent {
		switch e.Config {
		case yang.TSTrue:
			return true
		case yang.TSFalse:
			return false
		}
	}

	// Reached the last element in the tree without explicit configuration
	// being set.
	return e.Config != yang.TSFalse
}

// isPathChild takes an input slice of strings representing a path and determines
// whether b is a child of a within the YANG schema.
func isPathChild(a, b []string) bool {
	// If b does not have a greater path length than a, it cannot be a child. If
	// b has more than one element than a, it must be at least a grandchild.
	if len(b) <= len(a) || len(b) > len(a)+1 {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// IsDirectEntryChild determines whether the entry c is a direct child of the
// entry p within the output code. If compressPaths is set, a check to determine
// whether c would be a direct child after schema compression is performed.
func IsDirectEntryChild(p, c *yang.Entry, compressPaths bool) bool {
	ppp := strings.Split(p.Path(), "/")
	cpp := strings.Split(c.Path(), "/")
	dc := isPathChild(ppp, cpp)

	// If we are not compressing paths, then directly return whether the child
	// is a path of the parent.
	if !compressPaths {
		return dc
	}

	// If the length of the child path is greater than two larger than the
	// parent path, then this means that it cannot be a direct child, since all
	// path compression will remove only one level of hierarchy (config/state or
	// a surrounding container at maximum). We also check that the length of
	// the child path is more specific than or equal to the length of the parent
	// path in which case this cannot be a child.
	if len(cpp) > len(ppp)+2 || len(cpp) <= len(ppp) {
		return false
	}

	if IsConfigState(c.Parent) {
		// If the parent of this entity was the config/state container, then this
		// level of the hierarchy will have been removed so we check whether the
		// parent of both are equal and return this.
		return p.Path() == c.Parent.Parent.Path()
	}

	// If the child is a list, then we check whether the parent has only one
	// child (i.e., is a surrounding container) and then check whether the
	// single child is the child we were provided.
	if c.IsList() {
		ppe, ok := p.Dir[c.Parent.Name]
		if !ok {
			// Can't be a valid child because the parent of the entity doesn't exist
			// within this container.
			return false
		}
		if !HasOnlyChild(ppe) {
			return false
		}

		// We are guaranteed to have 1 child (and not zero) since HasOnlyChild will
		// return false for directories with 0 children.
		return Children(ppe)[0].Path() == c.Path()
	}

	return dc
}

// FindFirstNonChoiceOrCase recursively traverses the schema tree and returns a
// map with the set of the first nodes in every path that are neither case nor
// choice nodes. The keys in the map are the paths from root to the matching
// elements. If the path to the parent data struct is needed, since it always
// has length 1, this is simply the last path element of the key.
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
		m[e.Path()] = e
	case e.IsDir():
		for _, ch := range e.Dir {
			addToEntryMap(m, findFirstNonChoiceOrCaseInternal(ch))
		}
	}
	return m
}

// addToEntryMap merges from into to, overwriting overlapping key-value pairs.
func addToEntryMap(to, from map[string]*yang.Entry) map[string]*yang.Entry {
	for k, v := range from {
		to[k] = v
	}
	return to
}

// FlattenedTypes returns in tree order (in-order) the subtypes of a union type.
func FlattenedTypes(types []*yang.YangType) []*yang.YangType {
	var ret []*yang.YangType
	for _, t := range types {
		if IsUnionType(t) {
			ret = append(ret, FlattenedTypes(t.Type)...)
		} else {
			ret = append(ret, t)
		}
	}
	return ret
}

// EnumeratedUnionTypes recursively searches the set of yang.YangTypes supplied
// to extract the enumerated types that are within a union. The set of input
// yang.YangTypes is expected to be the slice of types of the union type.
// It returns the enumerated types in tree order of appearance.
func EnumeratedUnionTypes(types []*yang.YangType) []*yang.YangType {
	var eTypes []*yang.YangType
	for _, t := range types {
		switch {
		case IsEnumeratedType(t):
			eTypes = append(eTypes, t)
		case IsUnionType(t):
			eTypes = append(eTypes, EnumeratedUnionTypes(t.Type)...)
		}
	}
	return eTypes
}

// DefiningType returns the type of definition of a subtype within a leaf type.
// In the trivial case that the subtype is the leaf type itself, the leaf type
// is returned; otherwise, subtype refers to a terminal union subtype within
// the leaf's union type. An error is returned if the type does not belong to the
// leaf type.
//
// The "defining type" of a union subtype is the closest, or innermost defining
// type to which the subtype belongs. The "defining type" can either mean a
// typedef-defined type or a leaf-defined type.
//
// Examples of the defining type of union subtypes within a top-level union
// used under a leaf:
// - a typedef within any kind or level of unions.
//   - defining type is the typedef itself -- the closest place of definition.
// - a non-typedef within a non-typedef union.
//   - defining type is the union (i.e. type of the leaf, which defines it)
// - a non-typedef within a non-typedef union within a non-typedef union.
//   - defining type is the outer union (i.e. type of the leaf, which defines it).
// - a non-typedef within a typedef union within a non-typedef union.
//   - defining type is the (inner) typedef union.
func DefiningType(subtype *yang.YangType, leafType *yang.YangType) (*yang.YangType, error) {
	if subtype == leafType {
		// Trivial case where the subtype is the leaf type itself.
		// The leaf type is a place of definition, and it's also the closest.
		return leafType, nil
	}
	return unionDefiningType(subtype, leafType, leafType)
}

// unionDefiningType returns the type of definition of a union subtype.
// subtype is the union subtype, unionType is the current union type where
// we're looking for the subtype, and definingType is the defining type of
// unionType. An error is returned if the subtype was not found within the union.
func unionDefiningType(subtype *yang.YangType, unionType *yang.YangType, definingType *yang.YangType) (*yang.YangType, error) {
	for _, t := range unionType.Type {
		definingType := definingType
		if !IsYANGBaseType(t) {
			definingType = t
		}
		switch {
		case t == subtype:
			return definingType, nil
		case IsUnionType(t):
			if defType, err := unionDefiningType(subtype, t, definingType); err == nil {
				return defType, nil
			}
		}
	}
	return nil, fmt.Errorf("ygot/util: subtype %q not found within provided containing type %q", subtype.Name, unionType.Name)
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
		ns, err := FindLeafRefSchema(s, s.Type.Path)
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

// ListKeyFieldsMap returns a map[string]bool where the keys of the map
// are the fields that are the keys of the list described by the supplied
// yang.Entry. In the case the yang.Entry does not described a keyed list,
// an empty map is returned.
func ListKeyFieldsMap(e *yang.Entry) map[string]bool {
	r := map[string]bool{}
	for _, k := range strings.Fields(e.Key) {
		if k != "" {
			r[k] = true
		}
	}
	return r
}
