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
	"errors"
	"fmt"
	"reflect"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"

	log "github.com/golang/glog"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// IsTypeStruct reports whether t is a struct type.
func IsTypeStruct(t reflect.Type) bool {
	return t.Kind() == reflect.Struct
}

// IsTypeStructPtr reports whether v is a struct ptr type.
func IsTypeStructPtr(t reflect.Type) bool {
	if t == reflect.TypeOf(nil) {
		return false
	}
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct
}

// IsTypeSlice reports whether v is a slice type.
func IsTypeSlice(t reflect.Type) bool {
	return t.Kind() == reflect.Slice
}

// IsTypeSlicePtr reports whether v is a slice ptr type.
func IsTypeSlicePtr(t reflect.Type) bool {
	if t == reflect.TypeOf(nil) {
		return false
	}
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Slice
}

// IsTypeMap reports whether v is a map type.
func IsTypeMap(t reflect.Type) bool {
	if t == reflect.TypeOf(nil) {
		return false
	}
	return t.Kind() == reflect.Map
}

// IsTypeInterface reports whether v is an interface.
func IsTypeInterface(t reflect.Type) bool {
	if t == reflect.TypeOf(nil) {
		return false
	}
	return t.Kind() == reflect.Interface
}

// IsTypeSliceOfInterface reports whether v is a slice of interface.
func IsTypeSliceOfInterface(t reflect.Type) bool {
	if t == reflect.TypeOf(nil) {
		return false
	}
	return t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Interface
}

// IsNilOrInvalidValue reports whether v is nil or reflect.Zero.
func IsNilOrInvalidValue(v reflect.Value) bool {
	return !v.IsValid() || (v.Kind() == reflect.Ptr && v.IsNil()) || IsValueNil(v.Interface())
}

// IsValueNil returns true if either value is nil, or has dynamic type {ptr,
// map, slice} with value nil.
func IsValueNil(value interface{}) bool {
	if value == nil {
		return true
	}
	switch reflect.TypeOf(value).Kind() {
	case reflect.Slice, reflect.Ptr, reflect.Map:
		return reflect.ValueOf(value).IsNil()
	}
	return false
}

// IsValueNilOrDefault returns true if either IsValueNil(value) or the default
// value for the type.
func IsValueNilOrDefault(value interface{}) bool {
	if IsValueNil(value) {
		return true
	}
	if !IsValueScalar(reflect.ValueOf(value)) {
		// Default value is nil for non-scalar types.
		return false
	}
	return value == reflect.New(reflect.TypeOf(value)).Elem().Interface()
}

// IsValuePtr reports whether v is a ptr type.
func IsValuePtr(v reflect.Value) bool {
	return v.Kind() == reflect.Ptr
}

// IsValueInterface reports whether v is an interface type.
func IsValueInterface(v reflect.Value) bool {
	return v.Kind() == reflect.Interface
}

// IsValueStruct reports whether v is a struct type.
func IsValueStruct(v reflect.Value) bool {
	return v.Kind() == reflect.Struct
}

// IsValueStructPtr reports whether v is a struct ptr type.
func IsValueStructPtr(v reflect.Value) bool {
	return v.Kind() == reflect.Ptr && IsValueStruct(v.Elem())
}

// IsValueMap reports whether v is a map type.
func IsValueMap(v reflect.Value) bool {
	return v.Kind() == reflect.Map
}

// IsValueSlice reports whether v is a slice type.
func IsValueSlice(v reflect.Value) bool {
	return v.Kind() == reflect.Slice
}

// IsValueScalar reports whether v is a scalar type.
func IsValueScalar(v reflect.Value) bool {
	if IsNilOrInvalidValue(v) {
		return false
	}
	if IsValuePtr(v) {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	return !IsValueStruct(v) && !IsValueMap(v) && !IsValueSlice(v)
}

// ValuesAreSameType returns true if v1 and v2 has the same reflect.Type,
// otherwise it returns false.
func ValuesAreSameType(v1 reflect.Value, v2 reflect.Value) bool {
	return v1.Type() == v2.Type()
}

// IsValueInterfaceToStructPtr reports whether v is an interface that contains a
// pointer to a struct.
func IsValueInterfaceToStructPtr(v reflect.Value) bool {
	return IsValueInterface(v) && IsValueStructPtr(v.Elem())
}

// IsStructValueWithNFields returns true if the reflect.Value representing a
// struct v has n fields.
func IsStructValueWithNFields(v reflect.Value, n int) bool {
	return IsValueStruct(v) && v.NumField() == n
}

// InsertIntoSlice inserts value into parent which must be a slice ptr.
func InsertIntoSlice(parentSlice interface{}, value interface{}) error {
	DbgPrint("InsertIntoSlice into parent type %T with value %v, type %T", parentSlice, ValueStrDebug(value), value)

	pv := reflect.ValueOf(parentSlice)
	t := reflect.TypeOf(parentSlice)
	v := reflect.ValueOf(value)

	if !IsTypeSlicePtr(t) {
		return fmt.Errorf("InsertIntoSlice parent type is %s, must be slice ptr", t)
	}

	pv.Elem().Set(reflect.Append(pv.Elem(), v))
	DbgPrint("new list: %v\n", pv.Elem().Interface())

	return nil
}

// InsertIntoMap inserts value with key into parent which must be a map.
func InsertIntoMap(parentMap interface{}, key interface{}, value interface{}) error {
	DbgPrint("InsertIntoMap into parent type %T with key %v(%T) value \n%s\n (%T)",
		parentMap, ValueStrDebug(key), key, pretty.Sprint(value), value)

	v := reflect.ValueOf(parentMap)
	t := reflect.TypeOf(parentMap)
	kv := reflect.ValueOf(key)
	vv := reflect.ValueOf(value)

	if t.Kind() != reflect.Map {
		return fmt.Errorf("InsertIntoMap parent type is %s, must be map", t)
	}

	v.SetMapIndex(kv, vv)

	return nil
}

// UpdateField updates a field called fieldName (which must exist, but may be
// nil) in parentStruct, with value fieldValue. If the field is a slice,
// fieldValue is appended.
func UpdateField(parentStruct interface{}, fieldName string, fieldValue interface{}) error {
	DbgPrint("UpdateField field %s of parent type %T with value %v", fieldName, parentStruct, ValueStrDebug(fieldValue))

	if IsValueNil(parentStruct) {
		return fmt.Errorf("parent is nil in UpdateField for field %s", fieldName)
	}

	pt := reflect.TypeOf(parentStruct)

	if !IsTypeStructPtr(pt) {
		return fmt.Errorf("parent type %T must be a struct ptr", parentStruct)
	}
	ft, ok := pt.Elem().FieldByName(fieldName)
	if !ok {
		return fmt.Errorf("parent type %T does not have a field name %s", parentStruct, fieldName)
	}

	if ft.Type.Kind() == reflect.Slice {
		return InsertIntoSliceStructField(parentStruct, fieldName, fieldValue)
	}

	return InsertIntoStruct(parentStruct, fieldName, fieldValue)
}

// InsertIntoStruct updates a field called fieldName (which must exist, but may
// be nil) in parentStruct, with value fieldValue.
// If the struct field type is a ptr and the value is non-ptr, the field is
// populated with the corresponding ptr type.
func InsertIntoStruct(parentStruct interface{}, fieldName string, fieldValue interface{}) error {
	DbgPrint("InsertIntoStruct field %s of parent type %T with value %v", fieldName, parentStruct, ValueStrDebug(fieldValue))

	v, t := reflect.ValueOf(fieldValue), reflect.TypeOf(fieldValue)
	pv, pt := reflect.ValueOf(parentStruct), reflect.TypeOf(parentStruct)

	if !IsTypeStructPtr(pt) {
		return fmt.Errorf("parent type %T must be a struct ptr", parentStruct)
	}
	ft, ok := pt.Elem().FieldByName(fieldName)
	if !ok {
		return fmt.Errorf("parent type %T does not have a field name %s", parentStruct, fieldName)
	}

	// YANG empty fields are represented as a derived bool value defined in the
	// generated code. Here we cast the value to the type in the generated code.
	if ft.Type.Kind() == reflect.Bool && t.Kind() == reflect.Bool {
		nv := reflect.New(ft.Type).Elem()
		nv.SetBool(v.Bool())
		v = nv
	}

	// YANG binary fields are represented as a derived []byte value defined in the
	// generated code. Here we cast the value to the type in the generated code.
	// This will also cast a []uint8 value since byte is an alias for uint8.
	if ft.Type.Kind() == reflect.Slice && t.Kind() == reflect.Slice && ft.Type.Elem().Kind() == reflect.Uint8 && t.Elem().Kind() == reflect.Uint8 {
		nv := reflect.New(ft.Type).Elem()
		nv.SetBytes(v.Bytes())
		v = nv
	}

	n := v
	if n.IsValid() && (ft.Type.Kind() == reflect.Ptr && t.Kind() != reflect.Ptr) {
		n = reflect.New(t)
		n.Elem().Set(v)
	}

	if !n.IsValid() {
		if ft.Type.Kind() != reflect.Ptr {
			return fmt.Errorf("cannot assign value %v (type %T) to struct field %s (type %v) in struct %T", fieldValue, fieldValue, fieldName, ft.Type, parentStruct)
		}
		n = reflect.Zero(ft.Type)
	}

	if !isFieldTypeCompatible(ft, n) {
		return fmt.Errorf("cannot assign value %v (type %T) to struct field %s (type %v) in struct %T", fieldValue, fieldValue, fieldName, ft.Type, parentStruct)
	}

	pv.Elem().FieldByName(fieldName).Set(n)

	return nil
}

// InsertIntoSliceStructField inserts fieldValue into a field of type slice in
// parentStruct called fieldName (which must exist, but may be nil).
func InsertIntoSliceStructField(parentStruct interface{}, fieldName string, fieldValue interface{}) error {
	DbgPrint("InsertIntoSliceStructField field %s of parent type %T with value %v", fieldName, parentStruct, ValueStrDebug(fieldValue))

	v, t := reflect.ValueOf(fieldValue), reflect.TypeOf(fieldValue)
	pv, pt := reflect.ValueOf(parentStruct), reflect.TypeOf(parentStruct)

	if !IsTypeStructPtr(pt) {
		return fmt.Errorf("parent type %T must be a struct ptr", parentStruct)
	}
	ft, ok := pt.Elem().FieldByName(fieldName)
	if !ok {
		return fmt.Errorf("parent type %T does not have a field name %s", parentStruct, fieldName)
	}
	if ft.Type.Kind() != reflect.Slice {
		return fmt.Errorf("parent type %T, field name %s is type %s, must be a slice", parentStruct, fieldName, ft.Type)
	}
	et := ft.Type.Elem()

	n := v
	if n.IsValid() && (et.Kind() == reflect.Ptr && t.Kind() != reflect.Ptr) {
		n = reflect.New(t)
		n.Elem().Set(v)
	}
	if !n.IsValid() {
		n = reflect.Zero(et)
	}
	if !IsValueTypeCompatible(et, n) {
		return fmt.Errorf("cannot assign value %v (type %T) to struct field %s (type %v) in struct %T", fieldValue, fieldValue, fieldName, et, parentStruct)
	}

	nl := reflect.Append(pv.Elem().FieldByName(fieldName), n)
	pv.Elem().FieldByName(fieldName).Set(nl)

	return nil
}

// InsertIntoMapStructField inserts fieldValue into a field of type map in
// parentStruct called fieldName (which must exist, but may be nil), using the
// given key. If the key already exists in the map, the corresponding value is
// updated.
func InsertIntoMapStructField(parentStruct interface{}, fieldName string, key, fieldValue interface{}) error {
	DbgPrint("InsertIntoMapStructField field %s of parent type %T with key %v, value %v", fieldName, parentStruct, key, ValueStrDebug(fieldValue))

	v := reflect.ValueOf(parentStruct)
	t := reflect.TypeOf(parentStruct)
	if v.Kind() == reflect.Ptr {
		t = reflect.TypeOf(v.Elem().Interface())
	}
	ft, ok := t.FieldByName(fieldName)
	if !ok {
		return fmt.Errorf("field %s not found in parent type %T", fieldName, parentStruct)
	}

	if ft.Type.Kind() != reflect.Map {
		return fmt.Errorf("field %s to insert into must be a map, type is %v", fieldName, ft.Type.Kind())
	}
	vv := v
	if v.Kind() == reflect.Ptr {
		vv = v.Elem()
	}
	fvn := reflect.TypeOf(vv.FieldByName(fieldName).Interface()).Elem()
	if fvn.Kind() != reflect.ValueOf(fieldValue).Kind() && !(fieldValue == nil && fvn.Kind() == reflect.Ptr) {
		return fmt.Errorf("cannot assign value %v (type %T) to field %s (type %v) in struct %s",
			fieldValue, fieldValue, fieldName, fvn.Kind(), t.Name())
	}

	n := reflect.New(fvn)
	if fieldValue != nil {
		n.Elem().Set(reflect.ValueOf(fieldValue))
	}
	fv := v.Elem().FieldByName(fieldName)
	if fv.IsNil() {
		fv.Set(reflect.MakeMap(fv.Type()))
	}
	fv.SetMapIndex(reflect.ValueOf(key), n.Elem())

	return nil
}

// InitializeStructField initializes the given field in the given struct. Only
// pointer fields and some of the composite types are initialized(Map).
// It initializes to zero value of the underlying type if the field is a pointer.
// If the field is a slice, no need to initialize as appending a new element
// will do the same thing. Note that if the field is initialized already, this
// function doesn't re-initialize it.
func InitializeStructField(parent interface{}, fieldName string, initializeLeafs bool) error {
	if parent == nil {
		return errors.New("parent is nil")
	}
	pV := reflect.ValueOf(parent)
	if IsValuePtr(pV) {
		pV = pV.Elem()
	}

	if !IsValueStruct(pV) {
		return fmt.Errorf("%T is not a struct kind", parent)
	}

	fV := pV.FieldByName(fieldName)
	if !fV.IsValid() {
		return fmt.Errorf("invalid %T %v field value", parent, fieldName)
	}
	switch {
	case IsValuePtr(fV) && fV.IsNil():
		if v := reflect.New(fV.Type().Elem()); initializeLeafs || !IsValueScalar(v) {
			fV.Set(v)
		}
	case IsValueMap(fV) && fV.IsNil():
		fV.Set(reflect.MakeMap(fV.Type()))
	}

	return nil
}

// isFieldTypeCompatible reports whether f.Set(v) can be called successfully on
// a struct field f corresponding to ft. It is assumed that f is exported and
// addressable.
func isFieldTypeCompatible(ft reflect.StructField, v reflect.Value) bool {
	if ft.Type.Kind() == reflect.Ptr {
		if !v.IsValid() {
			return true
		}
		return v.Type().AssignableTo(ft.Type)
	}

	if !v.IsValid() {
		return false
	}

	return v.Type().AssignableTo(ft.Type)
}

// IsValueTypeCompatible reports whether f.Set(v) can be called successfully on
// a struct field f with type t. It is assumed that f is exported and
// addressable.
func IsValueTypeCompatible(t reflect.Type, v reflect.Value) bool {
	switch {
	case !v.IsValid():
		return t.Kind() == reflect.Ptr
	case t.Kind() != reflect.Interface:
		return v.Type().Kind() == t.Kind()
	default:
		return v.Type().Implements(t)
	}
}

// DeepEqualDerefPtrs compares the values of a and b. If either value is a ptr,
// it is dereferenced prior to the comparison.
func DeepEqualDerefPtrs(a, b interface{}) bool {
	aa := a
	bb := b
	if !IsValueNil(a) && reflect.TypeOf(a).Kind() == reflect.Ptr {
		aa = reflect.ValueOf(a).Elem().Interface()
	}
	if !IsValueNil(b) && reflect.TypeOf(b).Kind() == reflect.Ptr {
		bb = reflect.ValueOf(b).Elem().Interface()
	}
	return reflect.DeepEqual(aa, bb)
}

// ChildSchema returns the schema for the struct field f, if f contains a valid
// path tag and the schema path is found in the schema tree. It returns an error
// if the struct tag is invalid, or nil if tag is valid but the schema is not
// found in the tree at the specified path.
func ChildSchema(schema *yang.Entry, f reflect.StructField) (*yang.Entry, error) {
	return childSchema(schema, f, false)
}

// ChildSchemaPreferShadow returns the shadow schema (if exists) or schema for
// the struct field f, if f contains a valid "shadow-path" or "path" tag and
// the schema path is found in the schema tree. It returns an error if the
// struct tag is invalid, or nil if tag is valid but the schema is not found in
// the tree at the specified path.
func ChildSchemaPreferShadow(schema *yang.Entry, f reflect.StructField) (*yang.Entry, error) {
	return childSchema(schema, f, true)
}

// childSchema returns the schema for the struct field f, if f contains a valid
// path tag and the schema path is found in the schema tree. It returns an error
// if the struct tag is invalid, or nil if tag is valid but the schema is not
// found in the tree at the specified path.
//
// If preferShadowPath is false, the path values from the "path" tag are used.
// If preferShadowPath is true and the field has a "shadow-path" tag, then the
// path values from the "shadow-path" tag are used; if the field doesn't have
// the "shadow-path" tag, then the path values from the "path" tag are used.
func childSchema(schema *yang.Entry, f reflect.StructField, preferShadowPath bool) (*yang.Entry, error) {
	pathTag, _ := f.Tag.Lookup("path")
	shadowPathTag, _ := f.Tag.Lookup("shadow-path")
	DbgSchema("childSchema for schema %s, field %s, path tag %s, shadow-path tag\n", schema.Name, f.Name, pathTag, shadowPathTag)
	p, err := relativeSchemaPath(f, preferShadowPath)
	if err != nil {
		return nil, err
	}

	// Containers may have the container schema name as the first element in the
	// path tag for each field e.g. System { Dns ... path: "system/dns"
	// Strip this off since the supplied schema already refers to the struct
	// schema element.
	if schema.IsContainer() && len(p) > 1 && p[0] == schema.Name {
		p = p[1:]
	}
	DbgSchema("RelativeSchemaPath yields %v\n", p)
	// For empty path, return the parent schema.
	childSchema := schema
	foundSchema := true
	// Traverse the returned schema path to get the child schema.
	DbgSchema("traversing schema Dirs...")
	for ; len(p) > 0; p = p[1:] {
		DbgSchema("/%s", p[0])
		p := StripModulePrefix(p[0])
		ns, ok := childSchema.Dir[p]
		if !ok {
			foundSchema = false
			break
		}
		childSchema = ns
	}
	if foundSchema {
		DbgSchema(" - found\n")
		return childSchema, nil
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
	for path, entry := range entries {
		splitPath := SplitPath(path)
		name := splitPath[len(splitPath)-1]
		DbgSchema("%s ? ", name)

		if StripModulePrefix(name) == p[0] {
			DbgSchema(" - match\n")
			return entry, nil
		}
	}

	DbgSchema(" - no matches\n")
	return nil, nil
}

// stringMapKeys returns the keys for map m.
func stringMapKeys(m map[string]*yang.Entry) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// NodeInfo describes a node in a tree being traversed. It is passed to the
// iterator function supplied to a traversal driver function like ForEachField.
type NodeInfo struct {
	// Schema is the schema for the node.
	Schema *yang.Entry
	// Path is the relative path from the parent to the current schema node.
	PathFromParent []string
	// Parent is a ptr to the containing node.
	Parent *NodeInfo
	// StructField is the StructField for the field being traversed.
	StructField reflect.StructField
	// FieldValue is the Value for the field being traversed.
	FieldValue reflect.Value
	// FieldKeys is the slice of keys in the map being traversed. nil if type
	// being traversed is not a map.
	FieldKeys []reflect.Value
	// FieldKey is the key of the map element being traversed. ValueOf(nil) if
	// type being traversed is not a map.
	FieldKey reflect.Value
	// Annotation is a field that can be populated by an iterFunction such that
	// context can be carried with a node throughout the iteration.
	Annotation []interface{}
}

// PathQueryMemo caches nodes retrieved from (string) path queries. This memo
// may be useful if an algorithm may do multiple queries against the same path
// from the same node, any of which could be very expensive since the tree could
// be deep and wide.
type PathQueryMemo map[string]PathQueryResult

// PathQueryResult stores a datanode query result.
type PathQueryResult struct {
	Nodes []interface{}
	Err   error
}

// PathQueryNodeMemo caches previous path queries done against a particular node.
// Parent pointer allows looking up the memos of its ancestor nodes.
type PathQueryNodeMemo struct {
	Parent *PathQueryNodeMemo
	Memo   PathQueryMemo
}

// GetRoot returns the PathQueryNodeMemo of the current node's tree's root.
func (node *PathQueryNodeMemo) GetRoot() *PathQueryNodeMemo {
	for node.Parent != nil {
		node = node.Parent
	}
	return node
}

// FieldIteratorFunc is an iteration function for arbitrary field traversals.
// in, out are passed through from the caller to the iteration visitor function
// and can be used to pass state in and out. They are not otherwise touched.
// It returns a slice of errors encountered while processing the field.
type FieldIteratorFunc func(ni *NodeInfo, in, out any) Errors

// IterationAction is an enumeration representing different iteration actions.
//
//go:generate stringer -type=IterationAction
type IterationAction uint

const (
	// ContinueIteration means to continue the preorder traversal.
	ContinueIteration = IterationAction(iota)
	// DoNotIterateDescendants means to continue traversal but skip the
	// descendant elements of this subtree node.
	DoNotIterateDescendants
)

// FieldIteratorFunc2 is an iteration function for arbitrary field traversals.
// in, out are passed through from the caller to the iteration visitor function
// and can be used to pass state in and out. They are not otherwise touched.
// It returns what next iteration action to take as well as an error.
// TODO Deprecate the ForEachField and ForEachDataField functions in favor of Walk.
type FieldIteratorFunc2 func(ni *NodeInfo, in, out any) (IterationAction, Errors)

// ForEachField recursively iterates through the fields of value (which may be
// any Go type) and executes iterFunction on each field. Any nil fields
// (including value) are traversed in the schema tree only. This is done to
// support iterations that need to detect the absence of some data item e.g.
// leafref. Fields that are present in value that are explicitly noted not to
// have a corresponding schema (e.g., annotation/metadata fields added by ygen)
// are skipped during traversal.
//
// - schema is the schema corresponding to value.
// - in, out are passed to the iterator function and can be used to carry state
// and return results from the iterator.
// - iterFunction is executed on each scalar field.
//
// It returns a slice of errors encountered while processing the struct.
//
// See util.Walk if more dynamic control of the traversal is needed.
func ForEachField(schema *yang.Entry, value any, in, out any, iterFunction FieldIteratorFunc) Errors {
	if IsValueNil(value) {
		return nil
	}
	var v Visitor
	errCollector := new(DefaultWalkErrors)
	if inMemo, ok := in.(*PathQueryNodeMemo); ok {
		v = forEachFieldMemoVisitor{
			// Any existing Memo map passed in will be ignored.
			parent:         inMemo.Parent,
			out:            out,
			iterFunction:   iterFunction,
			errorCollector: errCollector,
		}
	} else {
		v = &forEachFieldVisitor{
			in:             in,
			out:            out,
			iterFunction:   iterFunction,
			errorCollector: errCollector,
		}
	}

	// For back-compatibility with ForFieldField, we explicitly interleave the errors from
	// both the traversal and the iterFunction into the shared Errors slice.
	_ = Walk(v, WalkNodeFromGoStruct(value), DefaultWalkOptions().WithWalkErrors(errCollector).WithSchema(schema))
	// We ignore the returned WalkErrors because we know it's a reference to v.errorCollector.
	return errCollector.Errors
}

// iterFuncToIterFunc2 converts a FieldIteratorFunc to FieldIteratorFunc2.
func iterFuncToIterFunc2(iterFunction FieldIteratorFunc) FieldIteratorFunc2 {
	return func(ni *NodeInfo, in, out any) (IterationAction, Errors) {
		return ContinueIteration, iterFunction(ni, in, out)
	}
}

type forEachFieldVisitor struct {
	in, out        any
	errorCollector *DefaultWalkErrors
	iterFunction   FieldIteratorFunc
}

func (vf *forEachFieldVisitor) Visit(node WalkNode) Visitor {
	if node == nil {
		return nil
	}
	ni := node.NodeInfo()
	if err := vf.iterFunction(ni, vf.in, vf.out); err != nil {
		vf.errorCollector.Collect(err)
	}
	return vf
}

type forEachFieldMemoVisitor struct {
	parent         *PathQueryNodeMemo
	out            any
	errorCollector *DefaultWalkErrors
	iterFunction   FieldIteratorFunc
}

func (vf forEachFieldMemoVisitor) Visit(node WalkNode) Visitor {
	if node == nil {
		return nil
	}
	// Each children sibling needs a dedicated (not shared) Memo data structure (map).
	in := &PathQueryNodeMemo{
		Parent: vf.parent,
		Memo:   PathQueryMemo{},
	}
	ni := node.NodeInfo()
	if err := vf.iterFunction(ni, in, vf.out); err != nil {
		vf.errorCollector.Collect(err)
	}
	// Since we use a value receiver vf instead of a pointer receiver,
	// vf is passed as a copy and it's safe to manipulate.
	vf.parent = in
	return vf
}

// ForEachDataField iterates the value supplied and calls the iterFunction for
// each data tree node found in the supplied value. No schema information is required
// to perform the iteration. The in and out arguments are passed to the iterFunction
// without inspection by this function, and can be used by the caller to store
// input and output during the iteration through the data tree.
//
// Deprecated: Use ForEachDataField2 or util.Walk instead.
func ForEachDataField(value, in, out any, iterFunction FieldIteratorFunc) Errors {
	if IsValueNil(value) {
		return nil
	}

	return ForEachDataField2(value, in, out, iterFuncToIterFunc2(iterFunction))
}

// ForEachDataField2 is an improved ForEachDataField that allows iteration over
// the data tree in the supplied value with custom iteration behaviour at each
// iteration step.
//
// ForEachDataField2 calls iterFunction for each data tree node found in the
// supplied value. No schema information is required to perform the iteration.
// The in and out arguments are passed to the iterFunction without inspection
// by this function, and can be used by the caller to store input and output
// during the iteration through the data tree.
//
// See util.Walk if more dynamic control of the traversal is needed.
func ForEachDataField2(value, in, out any, iterFunction FieldIteratorFunc2) Errors {
	if IsValueNil(value) {
		return nil
	}
	errCollector := new(DefaultWalkErrors)
	v := &forEachDataField2Visitor{
		in:             in,
		out:            out,
		iterFunction2:  iterFunction,
		errorCollector: errCollector,
	}
	// For back-compatibility with ForFieldField, we explicitly interleave the errors from
	// both the traversal and the iterFunction into the shared Errors slice.
	_ = Walk(v, WalkNodeFromGoStruct(value), DefaultWalkOptions().WithWalkErrors(errCollector))
	// We ignore the returned WalkErrors because we know it's a reference to v.errorCollector.
	return errCollector.Errors
}

type forEachDataField2Visitor struct {
	in, out        any
	errorCollector *DefaultWalkErrors
	iterFunction2  FieldIteratorFunc2
}

func (vf *forEachDataField2Visitor) Visit(node WalkNode) Visitor {
	if node == nil {
		return nil
	}
	ni := node.NodeInfo()
	action, err := vf.iterFunction2(ni, vf.in, vf.out)
	if err != nil {
		vf.errorCollector.Collect(err)
	}
	vf.errorCollector.Errors = AppendErrs(vf.errorCollector.Errors, err)
	if action == DoNotIterateDescendants {
		return nil
	}
	return vf
}

// GetNodes returns the nodes in the data tree at the indicated path, relative
// to the supplied root and their corresponding schemas at the same slice index.
// schema is the schema for root.
// If the key for a list node is missing, all values in the list are returned.
// If the key is partial, all nodes matching the values present in the key are
// returned.
// If the root is the tree root, the path may be absolute.
// GetNodes returns an error if the path is not found in the tree, or an element
// along the path is nil.
//
// Deprecated: Use ytypes.GetNode with the option &ytypes.GetPartialKeyMatch{}
// and &ytypes.GetTolerateNil{} instead.
func GetNodes(schema *yang.Entry, root interface{}, path *gpb.Path) ([]interface{}, []*yang.Entry, error) {
	return getNodesInternal(schema, root, path)
}

// getNodesInternal is the internal implementation of GetNode. In addition to
// GetNode functionality, it can accept non GoStruct types e.g. map for a keyed
// list, or a leaf.
// See GetNodes for parameter and return value descriptions.
func getNodesInternal(schema *yang.Entry, root interface{}, path *gpb.Path) ([]interface{}, []*yang.Entry, error) {
	if IsValueNil(root) {
		ResetIndent()
		return nil, nil, nil
	}
	if len(path.GetElem()) == 0 {
		ResetIndent()
		return []interface{}{root}, []*yang.Entry{schema}, nil
	}
	if schema == nil {
		return nil, nil, fmt.Errorf("nil schema for data element type %T, remaining path %v", root, path)
	}
	// Strip off the absolute path prefix since the relative and absolute paths
	// are assumed to be equal.
	if path.GetElem()[0].GetName() == "" {
		path.Elem = path.GetElem()[1:]
	}

	Indent()
	DbgPrint("GetNode next path %v, value %v", path.GetElem()[0], ValueStrDebug(root))

	switch {
	case schema.IsContainer() || (schema.IsList() && IsTypeStructPtr(reflect.TypeOf(root))):
		// Either a container or list schema with struct data node (which could
		// be an element of a list).
		return getNodesContainer(schema, root, path)
	case schema.IsList():
		// A list schema with the list parent container node as the root.
		return getNodesList(schema, root, path)
	}

	return nil, nil, fmt.Errorf("bad schema type for %s, struct type %T", schema.Name, root)
}

// getNodesContainer traverses the container root, which must be a struct ptr
// type and matches each field against the first path element in path. If a
// field matches, it recurses into that field with the remaining path.
func getNodesContainer(schema *yang.Entry, root interface{}, path *gpb.Path) ([]interface{}, []*yang.Entry, error) {
	DbgPrint("getNodesContainer: schema %s, next path %v, value %v", schema.Name, path.GetElem()[0], ValueStrDebug(root))

	rv := reflect.ValueOf(root)
	if !IsValueStructPtr(rv) {
		return nil, nil, fmt.Errorf("getNodesContainer: root has type %T, expect struct ptr", root)
	}

	v := rv.Elem()

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		ft := v.Type().Field(i)

		// Skip annotation fields, since they do not have a schema.
		if IsYgotAnnotation(ft) {
			continue
		}

		cschema, err := ChildSchema(schema, ft)
		if err != nil {
			return nil, nil, fmt.Errorf("error for schema for type %T, field name %s: %s", root, ft.Name, err)
		}
		if cschema == nil {
			return nil, nil, fmt.Errorf("could not find schema for type %T, field name %s", root, ft.Name)
		}

		ps, err := SchemaPaths(ft)
		DbgPrint("check field name %s, paths %v", cschema.Name, ps)
		if err != nil {
			return nil, nil, err
		}
		for _, p := range ps {
			if PathMatchesPrefix(path, p) {
				// don't trim whole prefix  for keyed list since name and key
				// are a in the same element.
				to := len(p)
				if IsTypeMap(ft.Type) {
					to--
				}
				return getNodesInternal(cschema, f.Interface(), TrimGNMIPathPrefix(path, p[0:to]))
			}
		}
	}

	return nil, nil, DbgErr(fmt.Errorf("could not find path in tree beyond schema node %s, (type %T), remaining path %v", schema.Name, root, path))
}

// getNodesList traverses the list root, which must be a map of struct
// type and matches each map key against the keys specified in the first
// PathElem of the Path. If the key matches, it recurses into that field with
// the remaining path. If empty key is specified, all list elements match.
func getNodesList(schema *yang.Entry, root interface{}, path *gpb.Path) ([]interface{}, []*yang.Entry, error) {
	DbgPrint("getNodesList: schema %s, next path %v, value %v", schema.Name, path.GetElem()[0], ValueStrDebug(root))

	rv := reflect.ValueOf(root)
	if schema.Key == "" {
		return nil, nil, fmt.Errorf("getNodesList: path %v cannot traverse unkeyed list type %T", path, root)
	}
	if !IsValueMap(rv) {
		// Only keyed lists can be traversed with a path.
		return nil, nil, fmt.Errorf("getNodesList: root has type %T, expect map", root)
	}
	emptyKey := false
	if len(path.GetElem()[0].GetKey()) == 0 {
		DbgPrint("path %v at %T points to list with empty wildcard key", path, root)
		emptyKey = true
	}

	listElementType := rv.Type().Elem().Elem()
	listKeyType := rv.Type().Key()

	var matchNodes []interface{}
	var matchSchemas []*yang.Entry

	// Iterate through all the map keys to see if any match the path.
	for _, k := range rv.MapKeys() {
		ev := rv.MapIndex(k)
		DbgPrint("checking key %v, value %v", k.Interface(), ValueStrDebug(ev.Interface()))
		match := true
		if !emptyKey { // empty key matches everything.
			if !IsValueStruct(k) {
				// Compare just the single value of the key represented as a string.
				pathKey, ok := path.GetElem()[0].GetKey()[schema.Key]
				if !ok {
					return nil, nil, fmt.Errorf("gnmi path %v does not contain a map entry for the schema key field name %s, parent type %T",
						path, schema.Key, root)
				}
				kv, err := getKeyValue(ev.Elem(), schema.Key)
				if err != nil {
					return nil, nil, err
				}
				// NOTE: Normally we'd like to use ygot.KeyValueAsString for conversion
				// to a key's PathElem string representation, but since this is just a
				// temporary path used during validation, we don't care if it is slightly
				// off from the specification -- only that it works to uniquely identify
				// the key value.
				match = (fmt.Sprint(kv) == pathKey)
				DbgPrint("check simple key value %s==%s ? %t", kv, pathKey, match)
			} else {
				// Must compare all the key fields.
				for i := 0; i < k.NumField(); i++ {
					kfn := listKeyType.Field(i).Name
					fv := ev.Elem().FieldByName(kfn)
					if !fv.IsValid() {
						return nil, nil, fmt.Errorf("element struct type %s does not contain key field %s", k.Type(), kfn)
					}
					// FIXME(wenbli): This block was here but is not doing anything. We need to ensure that no functionality would be missing by removing it.
					//nv := fv
					//if fv.Type().Kind() == reflect.Ptr {
					//	// Ptr values are deferenced in key struct.
					//	nv = nv.Elem()
					//}
					kf, ok := listElementType.FieldByName(kfn)
					if !ok {
						return nil, nil, fmt.Errorf("element struct type %s does not contain key field %s", k.Type(), kfn)
					}
					pathKey, ok := path.GetElem()[0].GetKey()[pathStructTagKey(kf)]
					if !ok {
						// If the key is not filled, it is assumed to match.
						continue
					}
					// As above, we don't require this to be the exact YANG enum string
					// representation.
					if pathKey != fmt.Sprint(k.Field(i).Interface()) {
						match = false
						break
					}
					DbgPrint("key field value %s matches", pathKey)
				}
			}
		}

		if match {
			// Pass in the list schema, but the actual selected element
			// rather than the whole list.
			DbgPrint("key matches")
			n, s, err := getNodesInternal(schema, ev.Interface(), PopGNMIPath(path))
			if err != nil {
				return nil, nil, err
			}
			if n != nil {
				matchNodes = append(matchNodes, n...)
				matchSchemas = append(matchSchemas, s...)
			}
		}
	}

	if len(matchNodes) == 0 {
		return nil, nil, nil
	}
	return matchNodes, matchSchemas, nil
}

// pathStructTagKey returns the string label of the struct field sf when it is
// used in a YANG list. This is the last path element of the struct path tag.
func pathStructTagKey(f reflect.StructField) string {
	p, err := RelativeSchemaPath(f)
	if err != nil {
		log.Errorf("struct field %s does not have a path tag, bad schema?", f.Name)
		return ""
	}
	return p[len(p)-1]
}

// getKeyValue returns the value from the structVal field whose last path
// element is key. The value is dereferenced if it is a ptr type. This function
// is used to create a key value for a keyed list.
// getKeyValue returns an error if no path in any of the fields of structVal has
// key as the last path element.
func getKeyValue(structVal reflect.Value, key string) (interface{}, error) {
	for i := 0; i < structVal.NumField(); i++ {
		f := structVal.Type().Field(i)
		p, err := RelativeSchemaPath(f)
		if err != nil {
			return nil, err
		}
		if p[len(p)-1] == key {
			fv := structVal.Field(i)
			if fv.Type().Kind() == reflect.Ptr {
				// The type for the key is the dereferenced type, if the type
				// is a ptr.
				if !fv.Elem().IsValid() {
					return nil, fmt.Errorf("key field %s (%s) has nil value %v", key, fv.Type(), fv)
				}
				return fv.Elem().Interface(), nil
			}
			return fv.Interface(), nil
		}
	}

	return nil, fmt.Errorf("could not find key field %s in struct type %s", key, structVal.Type())
}
