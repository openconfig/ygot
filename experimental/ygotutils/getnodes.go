// Package ygotutils implements utility functions for users of
// github.com/openconfig/ygot.
package ygotutils

import (
	"fmt"
	"reflect"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	scpb "google.golang.org/genproto/googleapis/rpc/code"
)

// GetNodes returns the nodes in the data tree at the indicated path via the callback function, relative to
// the supplied root struct. If the root struct is the tree root, the path may
// be absolute.
// It returns an error if the path is not found in the tree, or an element along
// the path is nil.
func GetNodes(schema *yang.Entry, rootStruct ygot.GoStruct, path *gpb.Path, callback func(*gpb.Path, interface{})) {
	// node, _, status :=
	calculated := &gpb.Path{Target: path.Target}
	getNodesInternal(schema, rootStruct, path, calculated, callback)
	//return node, status
}

// getNodesInternal is the internal implementation of GetNode. In
// addition to GetNode functionality, it can accept non GoStruct types e.g.
// map for a keyed list, or a leaf.
func getNodesInternal(schema *yang.Entry, rootStruct interface{}, path *gpb.Path, calculated *gpb.Path, callback func(*gpb.Path, interface{})) {
	if len(path.GetElem()) == 0 {
		zeroIndent()
		callback(calculated, rootStruct)
		return
		//return rootStruct, schema, statusOK
	}
	if isNil(rootStruct) {
		callback(nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("nil data element type %T, remaining path %v", rootStruct, path)))
		return
		// return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("nil data element type %T, remaining path %v", rootStruct, path))
	}
	if schema == nil {
		callback(nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("nil schema for data element type %T, remaining path %v", rootStruct, path)))
		return
		// return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("nil schema for data element type %T, remaining path %v", rootStruct, path))
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
		getNodesContainer(schema, rootStruct, path, calculated, callback)
		return
	case schema.IsList():
		// A list schema with the list data node. Must find the element selected
		// by the path.
		getNodesList(schema, rootStruct, path, calculated, callback)
		return
	}
	callback(nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("bad schema type for %s, struct type %T", schema.Name, rootStruct)))
	//return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("bad schema type for %s, struct type %T", schema.Name, rootStruct))
}

// getNodeContainer traverses the container rootStruct, which must be a
// struct ptr type and matches each field against the first path element in
// path. If a field matches, it recurses into that field with the remaining
// path.
func getNodesContainer(schema *yang.Entry, rootStruct interface{}, path *gpb.Path, calculated *gpb.Path, callback func(*gpb.Path, interface{})) {
	dbgPrintln("getNodeContainer: schema %s, next path %v, value %v", schema.Name, path.GetElem()[0], valueStr(rootStruct))

	rv := reflect.ValueOf(rootStruct)
	if !IsValueStructPtr(rv) {
		callback(nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("getNodeContainer: rootStruct has type %T, expect struct ptr", rootStruct)))
		return
		//	return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("getNodeContainer: rootStruct has type %T, expect struct ptr", rootStruct))
	}

	v := rv.Elem()

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		ft := v.Type().Field(i)

		// Skip ygot Annotation fields since they do not have a schema.
		if util.IsYgotAnnotation(ft) {
			continue
		}

		cschema, err := childSchema(schema, ft)
		if err != nil {
			callback(nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("error for schema for type %T, field name %s: %s", rootStruct, ft.Name, err)))
			return
			// return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("error for schema for type %T, field name %s: %s", rootStruct, ft.Name, err))
		}
		if cschema == nil {
			callback(nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("could not find schema for type %T, field name %s", rootStruct, ft.Name)))
			return
			// return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("could not find schema for type %T, field name %s", rootStruct, ft.Name))
		}
		cschema, err = resolveLeafRef(cschema)
		if err != nil {
			callback(nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("error for schema for type %T, field name %s: %s", rootStruct, ft.Name, err)))
			return
			//return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("error for schema for type %T, field name %s: %s", rootStruct, ft.Name, err))
		}

		dbgPrintln("check field name %s", cschema.Name)
		ps, err := schemaPaths(ft)
		if err != nil {
			callback(nil, errToStatus(err))
			return
			// return nil, nil, errToStatus(err)
		}
		for _, p := range ps {
			if pathMatchesPrefix(path, p) {
				// don't trim whole prefix  for keyed list since name and key
				// are a in the same element.
				to := len(p)
				if IsTypeMap(ft.Type) {
					to--
				}
				pathElem := &gpb.PathElem{Name: path.Elem[0].Name}
				newcalculated := &gpb.Path{Target: calculated.Target}
				newcalculated.Elem = append(newcalculated.Elem, calculated.Elem[:len(calculated.Elem)]...)
				newcalculated.Elem = append(newcalculated.Elem, pathElem)

				getNodesInternal(cschema, f.Interface(), trimGNMIPathPrefix(path, p[0:to]), newcalculated, callback)
				return
			}
		}
	}
	callback(nil, toStatus(scpb.Code_NOT_FOUND, fmt.Sprintf("could not find path in tree beyond schema node %s, (type %T), remaining path %v", schema.Name, rootStruct, path)))
	// return nil, nil, toStatus(scpb.Code_NOT_FOUND, fmt.Sprintf("could not find path in tree beyond schema node %s, (type %T), remaining path %v", schema.Name, rootStruct, path))
}

// getNodeList traverses the list rootStruct, which must be a map of struct
// type and matches each map key against the first path element in path. If the
// key matches completely, it recurses into that field with the remaining path.
func getNodesList(schema *yang.Entry, rootStruct interface{}, path *gpb.Path, calculated *gpb.Path, callback func(*gpb.Path, interface{})) {
	dbgPrintln("getNodeList: schema %s, next path %v, value %v", schema.Name, path.GetElem()[0], valueStr(rootStruct))

	rv := reflect.ValueOf(rootStruct)
	if schema.Key == "" {
		callback(nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("getNodeList: path %v cannot traverse unkeyed list type %T", path, rootStruct)))
		return
		// return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("getNodeList: path %v cannot traverse unkeyed list type %T", path, rootStruct))
	}
	if path.GetElem()[0].GetKey() == nil {
		callback(nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("getNodeList: path %v at %T points to list but does not specify a key element", path, rootStruct)))
		return
		// return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("getNodeList: path %v at %T points to list but does not specify a key element", path, rootStruct))
	}
	if !IsValueMap(rv) {
		// Only keyed lists can be traversed with a path.
		callback(nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("getNodeList: rootStruct has type %T, expect map", rootStruct)))
		return
		// return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("getNodeList: rootStruct has type %T, expect map", rootStruct))
	}

	listElementType := rv.Type().Elem().Elem()
	listKeyType := rv.Type().Key()
	// Iterate through all the map keys to see if any match the path.
	for _, k := range rv.MapKeys() {
		ev := rv.MapIndex(k)
		dbgPrintln("checking key %v, value %v", k.Interface(), valueStr(ev.Interface()))
		match := true
		keys := map[string]string{}
		if !IsValueStruct(k) {
			// Compare just the single value of the key represented as a string.
			pathKey, ok := path.GetElem()[0].GetKey()[schema.Key]
			if !ok {
				callback(nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("gnmi path %v does not contain a map entry for the schema key field name %s, parent type %T",
					path, schema.Key, rootStruct)))
				return
				// return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("gnmi path %v does not contain a map entry for the schema key field name %s, parent type %T",
				// 	path, schema.Key, rootStruct))
			}
			kv, err := getKeyValue(ev.Elem(), schema.Key)
			if err != nil {
				callback(nil, errToStatus(err))
				return
				// return nil, nil, errToStatus(err)
			}
			dbgPrintln("check simple key value %s", pathKey)
			match = (pathKey == "*") || (fmt.Sprint(kv) == pathKey)
			keys[schema.Key] = fmt.Sprint(kv)
		} else {
			// Must compare all the key fields.
			for i := 0; i < k.NumField(); i++ {
				kfn := listKeyType.Field(i).Name
				fv := ev.Elem().FieldByName(kfn)
				if !fv.IsValid() {
					callback(nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("element struct type %s does not contain key field %s", k.Type(), kfn)))
					return
					// return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("element struct type %s does not contain key field %s", k.Type(), kfn))
				}
				nv := fv
				if fv.Type().Kind() == reflect.Ptr {
					// Ptr values are deferenced in key struct.
					nv = nv.Elem()
				}
				kf, ok := listElementType.FieldByName(kfn)
				if !ok {
					callback(nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("element struct type %s does not contain key field %s", k.Type(), kfn)))
					return
					// return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("element struct type %s does not contain key field %s", k.Type(), kfn))
				}
				pathKey, ok := path.GetElem()[0].GetKey()[pathStructTagKey(kf)]
				if !ok {
					callback(nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("gnmi path %v does not contain a map entry for the schema key field name %s, parent type %T",
						path, schema.Key, rootStruct)))
					return
					// return nil, nil, toStatus(scpb.Code_INVALID_ARGUMENT, fmt.Sprintf("gnmi path %v does not contain a map entry for the schema key field name %s, parent type %T",
					// 	path, schema.Key, rootStruct))
				}
				keys[kfn] = fmt.Sprint(k.Field(i).Interface())
				if pathKey != "*" && pathKey != fmt.Sprint(k.Field(i).Interface()) {
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
			pathElem := &gpb.PathElem{Name: path.Elem[0].Name}
			pathElem.Key = keys
			newcalculated := &gpb.Path{Target: calculated.Target}
			newcalculated.Elem = append(newcalculated.Elem, calculated.Elem[:len(calculated.Elem)-1]...)
			newcalculated.Elem = append(newcalculated.Elem, pathElem)

			getNodesInternal(schema, ev.Interface(), popGNMIPath(path), newcalculated, callback)
			//return
		}
	}

	//return nil, nil, toStatus(scpb.Code_NOT_FOUND, fmt.Sprintf("could not find path in tree beyond schema node %s, (type %T), remaining path %v", schema.Name, rootStruct, path))
}
