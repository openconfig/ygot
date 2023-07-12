package util

import (
	"fmt"
	"reflect"

	log "github.com/golang/glog"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/internal/yreflect"
)

// WalkNode is an abstraction of NodeInfo during the util.Walk.
type WalkNode interface {
	NodeInfo() *NodeInfo
}

// WalkErrors is an abstraction of collecting Errors during the util.Walk.
type WalkErrors interface {
	Collect(error)
}

// Visitor has a visit method that is invoked for each node encountered by Walk.
type Visitor interface {
	Visit(node WalkNode) (w Visitor)
}

// Walk traverses the nodes with a customized visitor.
//
// It traverses a GoStruct in depth-first order: It starts by calling
// v.Visit(node); node must not be nil. If the visitor w returned by
// v.Visit(node) is not nil, Walk is invoked recursively with visitor
// w for each of the non-nil children of node, followed by a call of
// w.Visit(nil).
//
// By default, the traversal only visit existing GoStruct fields.
// If a customized schema is provided via WithSchema WalkOptions,
// then the traversal will visit the schema entry even if the GoStruct does not populate it.
//
// The Visitor should handle its own error reporting and early termination.
// Any error during the traversal that is not part of Visitor will be aggregated into the
// returned WalkErrors.
// If not overwritten, the returned WalkErrors is of DefaultWalkErrors type, which is an alias of Errors.
func Walk(v Visitor, node WalkNode, o *WalkOptions) WalkErrors {
	if o == nil {
		o = &WalkOptions{}
	}
	if o.WalkErrors == nil {
		o.WalkErrors = &DefaultWalkErrors{}
	}
	if o.schema == nil {
		walkDataFieldInternal(v, node, o)
		return o.WalkErrors
	}
	node = WalkNodeFromNodeInfo(&NodeInfo{
		Schema:     o.schema,
		FieldValue: node.NodeInfo().FieldValue,
	})
	walkFieldInternal(v, node, o)
	return o.WalkErrors
}

var _ WalkNode = (*walkNodeInfo)(nil)

type walkNodeInfo struct {
	ni *NodeInfo
}

func (w *walkNodeInfo) NodeInfo() *NodeInfo {
	return w.ni
}

// WalkNodeFromGoStruct converts a GoStruct to WalkNode with empty schema.
func WalkNodeFromGoStruct(value any) WalkNode {
	return WalkNodeFromNodeInfo(&NodeInfo{
		FieldValue: reflect.ValueOf(value),
	})
}

// WalkNodeFromNodeInfo converts a NodeInfo to WalkNode.
func WalkNodeFromNodeInfo(ni *NodeInfo) WalkNode {
	return &walkNodeInfo{ni: ni}
}

// WalkOptions are configurations of the Walk function.
type WalkOptions struct {
	WalkErrors
	schema *yang.Entry
}

// DefaultWalkOptions initialize a WalkOptions.
func DefaultWalkOptions() *WalkOptions {
	return &WalkOptions{}
}

// WithSchema traverses schema entries even when a node is not populated with data.
func (o *WalkOptions) WithSchema(schema *yang.Entry) *WalkOptions {
	o.schema = schema
	return o
}

// WithWalkErrors customizes the WalkErrors returned by the Walk function.
// If unspecified, the DefaultWalkErrors is used.
func (o *WalkOptions) WithWalkErrors(we WalkErrors) *WalkOptions {
	o.WalkErrors = we
	return o
}

// DefaultWalkErrors is the default WalkErrors used unless overwritten via WithWalkErrors.
type DefaultWalkErrors struct {
	Errors
}

var _ WalkErrors = (*DefaultWalkErrors)(nil)

// Collect is used by Walk to aggregate errors during the traversal.
func (e *DefaultWalkErrors) Collect(err error) {
	e.Errors = AppendErr(e.Errors, err)
}

// walkFieldInternal traverses a GoStruct in depth-first order: It starts by calling
// v.Visit(node); node must not be nil. If the visitor w returned by
// v.Visit(node) is not nil, Walk is invoked recursively with visitor
// w for each of the non-nil children of node, followed by a call of
// w.Visit(nil).
// It behaves similar to ForEachField to determine the set of children of a node.
// Precondition: o.errorCollector must be initialized.
func walkFieldInternal(visitor Visitor, node WalkNode, o *WalkOptions) {
	ni := node.NodeInfo()
	// Ignore nil field.
	if IsValueNil(ni) {
		return
	}
	// If the field is an annotation, then we do not process it any further, including
	// skipping running the iterFunction.
	if IsYgotAnnotation(ni.StructField) {
		return
	}
	// walk the node itself
	childVisitor := visitor.Visit(node)
	if childVisitor == nil {
		return
	}
	defer childVisitor.Visit(nil)

	v := ni.FieldValue
	t := v.Type()

	// walk children
	orderedMap, isOrderedMap := v.Interface().(goOrderedMap)

	switch {
	case isOrderedMap, IsTypeSlice(t), IsTypeMap(t):
		schema := *(ni.Schema)
		schema.ListAttr = nil

		var relPath []string
		if !schema.IsLeafList() {
			// Leaf-list elements share the parent schema with listattr unset.
			relPath = []string{schema.Name}
		}

		var elemType reflect.Type
		switch {
		case isOrderedMap:
			var err error
			elemType, err = yreflect.OrderedMapElementType(orderedMap)
			if err != nil {
				o.WalkErrors.Collect(err)
				return
			}
		default:
			elemType = t.Elem()
		}

		nn := *ni
		// The schema for each element is the list schema minus the list
		// attrs.
		nn.Schema = &schema
		nn.Parent = ni
		nn.PathFromParent = relPath

		visitListElement := func(k, v reflect.Value) {
			nn := nn
			nn.FieldValue = v
			nn.FieldKey = k
			walkFieldInternal(childVisitor, WalkNodeFromNodeInfo(&nn), o)
		}

		switch {
		case IsNilOrInvalidValue(v):
			// Traverse the type tree only from this point.
			visitListElement(reflect.Value{}, reflect.Zero(elemType))
		case IsTypeSlice(t):
			for i := 0; i < ni.FieldValue.Len(); i++ {
				visitListElement(reflect.Value{}, ni.FieldValue.Index(i))
			}
		case isOrderedMap:
			var err error
			nn.FieldKeys, err = yreflect.OrderedMapKeys(orderedMap)
			if err != nil {
				o.WalkErrors.Collect(err)
				return
			}
			if err := yreflect.RangeOrderedMap(orderedMap, func(k, v reflect.Value) bool {
				visitListElement(k, v)
				return true
			}); err != nil {
				o.WalkErrors.Collect(err)
			}
		case IsTypeMap(t):
			nn.FieldKeys = ni.FieldValue.MapKeys()
			for _, key := range ni.FieldValue.MapKeys() {
				visitListElement(key, ni.FieldValue.MapIndex(key))
			}
		}

	case IsTypeStructPtr(t):
		t = t.Elem()
		if !IsNilOrInvalidValue(v) {
			v = v.Elem()
		}
		fallthrough
	case IsTypeStruct(t):
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)

			// Do not handle annotation fields, since they have no schema.
			if IsYgotAnnotation(sf) {
				continue
			}

			nn := &NodeInfo{
				Parent:      ni,
				StructField: sf,
			}
			if !IsNilOrInvalidValue(v) {
				nn.FieldValue = v.Field(i)
			} else {
				nn.FieldValue = reflect.Zero(sf.Type)
			}
			ps, err := SchemaPaths(nn.StructField)
			if err != nil {
				o.WalkErrors.Collect(err)
				return
			}

			for _, p := range ps {
				nn.Schema = FirstChild(ni.Schema, p)
				if nn.Schema == nil {
					e := fmt.Errorf("forEachFieldInternal could not find child schema with path %v from schema name %s", p, ni.Schema.Name)
					DbgPrint(e.Error())
					// TODO(wenovus) Consider making this into an error.
					log.Errorln(e)
					continue
				}
				nn.PathFromParent = p
				walkFieldInternal(childVisitor, WalkNodeFromNodeInfo(nn), o)
			}
		}
	}
}

// walkDataFieldInternal traverses a GoStruct in depth-first order: It starts by calling
// v.Visit(node); node must not be nil. If the visitor w returned by
// v.Visit(node) is not nil, Walk is invoked recursively with visitor
// w for each of the non-nil children of node, followed by a call of
// w.Visit(nil).
// It behaves similar to ForEachDataField2 to determine the set of children of a node.
// Precondition: o.errorCollector must be initialized.
func walkDataFieldInternal(visitor Visitor, node WalkNode, o *WalkOptions) {
	ni := node.NodeInfo()
	if IsValueNil(ni) {
		return
	}

	if IsNilOrInvalidValue(ni.FieldValue) {
		// Skip any fields that are nil within the data tree, since we
		// do not need to iterate on them.
		return
	}

	// walk the node itself
	childVisitor := visitor.Visit(node)
	if childVisitor == nil {
		return
	}
	defer childVisitor.Visit(nil)

	v := ni.FieldValue
	t := v.Type()

	orderedMap, isOrderedMap := v.Interface().(goOrderedMap)

	// Determine whether we need to recurse into the field, or whether it is
	// a leaf or leaf-list, which are not recursed into when traversing the
	// data tree.
	switch {
	case isOrderedMap:
		// Handle the case of a keyed map, which is a YANG list.
		if IsNilOrInvalidValue(v) {
			return
		}
		nn := *ni
		nn.Parent = ni
		var err error
		nn.FieldKeys, err = yreflect.OrderedMapKeys(orderedMap)
		if err != nil {
			o.WalkErrors.Collect(err)
			return
		}
		if err := yreflect.RangeOrderedMap(orderedMap, func(k, v reflect.Value) bool {
			nn := nn
			nn.FieldValue = v
			nn.FieldKey = k
			walkDataFieldInternal(childVisitor, WalkNodeFromNodeInfo(&nn), o)
			return true
		}); err != nil {
			o.WalkErrors.Collect(err)
		}
	case IsTypeStructPtr(t):
		// A struct pointer in a GoStruct is a pointer to another container within
		// the YANG, therefore we dereference the pointer and then recurse. If the
		// pointer is nil, then we do not need to do this since the data tree branch
		// is unset in the schema.
		t = t.Elem()
		v = v.Elem()
		fallthrough
	case IsTypeStruct(t):
		// Handle non-pointer structs by recursing into each field of the struct.
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			nn := &NodeInfo{
				Parent:      ni,
				StructField: sf,
				FieldValue:  reflect.Zero(sf.Type),
			}

			nn.FieldValue = v.Field(i)
			ps, err := SchemaPaths(nn.StructField)
			if err != nil {
				o.WalkErrors.Collect(err)
				return
			}
			// In the case that the field expands to >1 different data tree path,
			// i.e., SchemaPaths above returns more than one path, then we recurse
			// for each schema path. This ensures that the iterator
			// function runs for all expansions of the data tree as well as the GoStruct
			// fields.
			for _, p := range ps {
				nn.PathFromParent = p
				if IsTypeSlice(sf.Type) || IsTypeMap(sf.Type) {
					// Since lists can have path compression - where the path contains more
					// than one element, ensure that the schema path we received is only two
					// elements long. This protects against compression errors where there are
					// trailing spaces (e.g., a path tag of config/bar/).
					nn.PathFromParent = p[0:1]
				}
				walkDataFieldInternal(childVisitor, WalkNodeFromNodeInfo(nn), o)
			}
		}
	case IsTypeSlice(t):
		// Only iterate in the data tree if the slice is of structs, otherwise
		// for leaf-lists we only run once.
		if !IsTypeStructPtr(t.Elem()) && !IsTypeStruct(t.Elem()) {
			return
		}

		for i := 0; i < ni.FieldValue.Len(); i++ {
			nn := *ni
			nn.Parent = ni
			// The name of the list is the same in each of the entries within the
			// list therefore, we do not need to set the path to be different from
			// the parent.
			nn.PathFromParent = ni.PathFromParent
			nn.FieldValue = ni.FieldValue.Index(i)
			walkDataFieldInternal(childVisitor, WalkNodeFromNodeInfo(&nn), o)
		}
	case IsTypeMap(t):
		// Handle the case of a keyed map, which is a YANG list.
		if IsNilOrInvalidValue(v) {
			return
		}
		for _, key := range ni.FieldValue.MapKeys() {
			nn := *ni
			nn.Parent = ni
			nn.FieldValue = ni.FieldValue.MapIndex(key)
			nn.FieldKey = key
			nn.FieldKeys = ni.FieldValue.MapKeys()
			walkDataFieldInternal(childVisitor, WalkNodeFromNodeInfo(&nn), o)
		}
	}
}
