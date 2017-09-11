// Package ygotutils implements utility functions for users of
// github.com/openconfig/ygot.
package ygotutils

import (
	"fmt"
	"reflect"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	scpb "google.golang.org/genproto/googleapis/rpc/code"
	spb "google.golang.org/genproto/googleapis/rpc/status"
)

// GetNode returns the node in the data tree at the indicated path, relative to
// the supplied root struct. If the root struct is the tree root, the path may
// be absolute.
// It returns an error if the path is not found in the tree, or an element along
// the path is nil.
func GetNode(schema *yang.Entry, rootStruct ygot.GoStruct, path *gpb.Path) (interface{}, spb.Status) {
	node, _, status := getNodeInternal(schema, rootStruct, path)
	return node, status
}

// NewNode returns a new, empty struct element of the type indicated by the
// given path in the data tree.
// Note that the actual data tree is not required
// since this is a new, empty node. The path is simply used to traverse the
// schema tree - any key values in the path are ignored.
func NewNode(rootType reflect.Type, path *gpb.Path) (interface{}, spb.Status) {
	if len(path.GetElem()) == 0 {
		zeroIndent()
		dbgPrintln("creating new object of type %s", rootType)
		if rootType.Kind() == reflect.Ptr {
			return reflect.New(rootType.Elem()).Interface(), statusOK
		}
		return reflect.New(rootType).Elem().Interface(), statusOK
	}
	// Strip off the absolute path prefix since the relative and absolute paths
	// are assumed to be equal.
	if path.GetElem()[0].GetName() == "" {
		path.Elem = path.GetElem()[1:]
	}

	indent()
	dbgPrintln("NewNode type %s, next path %v", rootType, path.GetElem()[0])

	switch {
	case IsTypeStructPtr(rootType):
		return newNodeContainerType(rootType, path)
	case IsTypeMap(rootType) || IsTypeSlicePtr(rootType):
		return newNodeListType(rootType, path)
	}

	return nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("bad data type for %s, must be ptr to struct, slice, or map", rootType))
}

// getNodeInternal is the internal implementation of GetNode. In
// addition to GetNode functionality, it can accept non GoStruct types e.g.
// map for a keyed list, or a leaf.
func getNodeInternal(schema *yang.Entry, rootStruct interface{}, path *gpb.Path) (interface{}, *yang.Entry, spb.Status) {
	if len(path.GetElem()) == 0 {
		zeroIndent()
		return rootStruct, schema, statusOK
	}
	if isNil(rootStruct) {
		return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("nil data element type %T, remaining path %v", rootStruct, path))
	}
	if schema == nil {
		return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("nil schema for data element type %T, remaining path %v", rootStruct, path))
	}
	// Strip off the absolute path prefix since the relative and absolute paths
	// are assumed to be equal.
	if path.GetElem()[0].GetName() == "" {
		path.Elem = path.GetElem()[1:]
	}

	indent()
	dbgPrintln("GetNode next path %v, value %v", path.GetElem()[0], valueStr(rootStruct))

	switch {
	case schema.IsContainer() || (schema.IsList() && IsTypeStructPtr(reflect.TypeOf(rootStruct))):
		// Either a container or list schema with struct data node (which could
		// be an element of a list).
		return getNodeContainer(schema, rootStruct, path)
	case schema.IsList():
		// A list schema with the list data node. Must find the element selected
		// by the path.
		return getNodeList(schema, rootStruct, path)
	}

	return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("bad schema type for %s, struct type %T", schema.Name, rootStruct))
}

// getNodeContainer traverses the container rootStruct, which must be a
// struct ptr type and matches each field against the first path element in
// path. If a field matches, it recurses into that field with the remaining
// path.
func getNodeContainer(schema *yang.Entry, rootStruct interface{}, path *gpb.Path) (interface{}, *yang.Entry, spb.Status) {
	dbgPrintln("getNodeContainer: schema %s, next path %v, value %v", schema.Name, path.GetElem()[0], valueStr(rootStruct))

	rv := reflect.ValueOf(rootStruct)
	if !IsValueStructPtr(rv) {
		return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("getNodeContainer: rootStruct has type %T, expect struct ptr", rootStruct))
	}

	v := rv.Elem()

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		ft := v.Type().Field(i)
		cschema, err := childSchema(schema, ft)
		if err != nil {
			return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("error for schema for type %T, field name %s: %s", rootStruct, ft.Name, err))
		}
		if cschema == nil {
			return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("could not find schema for type %T, field name %s", rootStruct, ft.Name))
		}
		cschema, err = resolveLeafRef(cschema)
		if err != nil {
			return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("error for schema for type %T, field name %s: %s", rootStruct, ft.Name, err))
		}

		dbgPrintln("check field name %s", cschema.Name)
		ps, err := schemaPaths(ft)
		if err != nil {
			return nil, nil, errToStatus(err)
		}
		for _, p := range ps {
			if pathMatchesPrefix(path, p) {
				// don't trim whole prefix  for keyed list since name and key
				// are a in the same element.
				to := len(p)
				if IsTypeMap(ft.Type) {
					to--
				}
				return getNodeInternal(cschema, f.Interface(), trimGNMIPathPrefix(path, p[0:to]))
			}
		}
	}

	return nil, nil, toStatus(scpb.Code_NOT_FOUND, fmt.Sprintf("could not find path in tree beyond schema node %s, (type %T), remaining path %v", schema.Name, rootStruct, path))
}

// getNodeList traverses the list rootStruct, which must be a map of struct
// type and matches each map key against the first path element in path. If the
// key matches completely, it recurses into that field with the remaining path.
func getNodeList(schema *yang.Entry, rootStruct interface{}, path *gpb.Path) (interface{}, *yang.Entry, spb.Status) {
	dbgPrintln("getNodeList: schema %s, next path %v, value %v", schema.Name, path.GetElem()[0], valueStr(rootStruct))

	rv := reflect.ValueOf(rootStruct)
	if schema.Key == "" {
		return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("getNodeList: path %v cannot traverse unkeyed list type %T", path, rootStruct))
	}
	if path.GetElem()[0].GetKey() == nil {
		return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("getNodeList: path %v at %T points to list but does not specify a key element", path, rootStruct))
	}
	if !IsValueMap(rv) {
		// Only keyed lists can be traversed with a path.
		return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("getNodeList: rootStruct has type %T, expect map", rootStruct))
	}

	listElementType := rv.Type().Elem().Elem()
	listKeyType := rv.Type().Key()

	// Iterate through all the map keys to see if any match the path.
	for _, k := range rv.MapKeys() {
		ev := rv.MapIndex(k)
		dbgPrintln("checking key %v, value %v", k.Interface(), valueStr(ev.Interface()))
		match := true
		if !IsValueStruct(k) {
			// Compare just the single value of the key represented as a string.
			pathKey, ok := path.GetElem()[0].GetKey()[schema.Key]
			if !ok {
				return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("gnmi path %v does not contain a map entry for the schema key field name %s, parent type %T",
					path, schema.Key, rootStruct))
			}
			kv, err := getKeyValue(ev.Elem(), schema.Key)
			if err != nil {
				return nil, nil, errToStatus(err)
			}
			dbgPrintln("check simple key value %s", pathKey)
			match = (fmt.Sprint(kv) == pathKey)
		} else {
			// Must compare all the key fields.
			for i := 0; i < k.NumField(); i++ {
				kfn := listKeyType.Field(i).Name
				fv := ev.Elem().FieldByName(kfn)
				if !fv.IsValid() {
					return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("element struct type %s does not contain key field %s", k.Type(), kfn))
				}
				nv := fv
				if fv.Type().Kind() == reflect.Ptr {
					// Ptr values are deferenced in key struct.
					nv = nv.Elem()
				}
				kf, ok := listElementType.FieldByName(kfn)
				if !ok {
					return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("element struct type %s does not contain key field %s", k.Type(), kfn))
				}
				pathKey, ok := path.GetElem()[0].GetKey()[pathStructTagKey(kf)]
				if !ok {
					return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("gnmi path %v does not contain a map entry for the schema key field name %s, parent type %T",
						path, schema.Key, rootStruct))
				}
				if pathKey != fmt.Sprint(k.Field(i).Interface()) {
					match = false
					break
				}
				dbgPrintln("key field value %s matches", pathKey)
			}
		}

		if match {
			// Pass in the list schema, but the actual selected element
			// rather than the whole list.
			dbgPrintln("whole key matches")
			return getNodeInternal(schema, ev.Interface(), popGNMIPath(path))
		}
	}

	return nil, nil, toStatus(scpb.Code_NOT_FOUND, fmt.Sprintf("could not find path in tree beyond schema node %s, (type %T), remaining path %v", schema.Name, rootStruct, path))
}

// newNodeContainerType traverses the container, which must be a struct ptr
// and tries to match each field with the path prefix.
// If a match is found, it removes the matching prefix and recurses the
// corresponding field type with the remaining path.
func newNodeContainerType(rootType reflect.Type, path *gpb.Path) (interface{}, spb.Status) {
	dbgPrintln("newNodeContainerType: type %s, next path %v", rootType, path.GetElem()[0])

	if !IsTypeStructPtr(rootType) {
		return nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("newNodeContainerType: rootType has type %s, expect struct ptr", rootType))
	}

	t := rootType.Elem()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		ps, err := schemaPaths(f)
		if err != nil {
			return nil, errToStatus(err)
		}
		for _, p := range ps {
			if pathMatchesPrefix(path, p) {
				// don't trim whole prefix  for keyed list since name and key
				// are a in the same element.
				to := len(p)
				if IsTypeMap(f.Type) {
					to--
				}
				return NewNode(f.Type, trimGNMIPathPrefix(path, p[0:to]))
			}
		}
	}

	return nil, toStatus(scpb.Code_NOT_FOUND, fmt.Sprintf("could not find path in tree beyond type %s, remaining path %v", rootType, path))
}

// newNodeListType traverses the list, which must be a map of struct type,
// to its element struct type. It removes the front element from path and
// recurses the struct with the remaining path.
func newNodeListType(rootType reflect.Type, path *gpb.Path) (interface{}, spb.Status) {
	dbgPrintln("newNodeListType: type %s, next path %v", rootType, path.GetElem()[0])

	var listElementType reflect.Type
	switch {
	case IsTypeMap(rootType):
		listElementType = rootType.Elem()
	case IsTypeSlicePtr(rootType):
		listElementType = rootType.Elem().Elem()
	default:
		return nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("newNodeListType: rootType has type %s, expect map or slice ptr", rootType))
	}

	// Nothing to do execept to pop off the key and pass the container type
	// with the list schema.
	return NewNode(listElementType, popGNMIPath(path))
}
