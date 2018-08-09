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

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// Type retrieveNodeArgs contains the set of parameters that changes
// behavior of how retrieveNode works.
type retrieveNodeArgs struct {
	// If delete is set to true, retrieve node deletes the node at the
	// to supplied path.
	delete bool
	// If partialKeyMatch is set to true, retrieveNode tolerates missing
	// key(s) in the given path. If no key is provided, all the nodes
	// in the keyed list are treated as match. If some of the keys are
	// provided, it returns the nodes corresponding to provided keys.
	partialKeyMatch bool
	// If modifyRoot is set to true, retrieveNode traverses the GoStruct
	// and initialies nodes or inserting keys into maps if they do not exist.
	modifyRoot bool
	// If val is set to a non-nil value, leaf/leaflist node corresponding
	// to the given path is updated with this value.
	val interface{}
}

// retrieveNode is an internal function that retrieves the node specified by
// the supplied path from the root which must have the schema supplied.
// retrieveNodeArgs change the way retrieveNode works.
// retrieveNode returns the list of matching nodes and their schemas, and error.
// Note that retrieveNode may mutate the tree even if it fails.
func retrieveNode(schema *yang.Entry, root interface{}, path *gpb.Path, args retrieveNodeArgs) ([]interface{}, []*yang.Entry, error) {
	switch {
	case path == nil || len(path.Elem) == 0:
		return []interface{}{root}, []*yang.Entry{schema}, nil
	case util.IsValueNil(root):
		return nil, nil, status.Errorf(codes.InvalidArgument, "root is nil for type %T, path %v", root, path)
	case schema == nil:
		return nil, nil, status.Errorf(codes.InvalidArgument, "schema is nil for type %T, path %v", root, path)
	}

	switch {
	// Check if the schema is a container, or the schema is a list and the parent provided is a member of that list.
	case schema.IsContainer() || (schema.IsList() && util.IsTypeStructPtr(reflect.TypeOf(root))):
		return retrieveNodeContainer(schema, root, path, args)
	case schema.IsList():
		return retrieveNodeList(schema, root, path, args)
	}
	return nil, nil, status.Errorf(codes.InvalidArgument, "can not use a parent that is not a container or list; schema %v root %T, path %v", schema, root, path)
}

// retrieveNodeContainer is an internal function and operates on GoStruct. It retrieves
// the node by the supplied path from the root which must have the schema supplied.
// It recurses by calling retrieveNode. If modifyRoot is set to true, nodes along the path are initialized
// if they are nil. If val isn't nil, then it is set on the leaf or leaflist node.
// Note that root is modified even if function returns error status.
func retrieveNodeContainer(schema *yang.Entry, root interface{}, path *gpb.Path, args retrieveNodeArgs) ([]interface{}, []*yang.Entry, error) {
	rv := reflect.ValueOf(root)
	if !util.IsTypeStructPtr(rv.Type()) {
		return nil, nil, status.Errorf(codes.InvalidArgument, "got %T, want struct ptr root in retrieveNodeContainer", root)
	}

	// dereference reflect value as it points to a pointer.
	v := rv.Elem()

	for i := 0; i < v.NumField(); i++ {
		fv, ft := v.Field(i), v.Type().Field(i)

		if util.IsYgotAnnotation(ft) {
			continue
		}

		cschema, err := childSchema(schema, ft)
		switch {
		case err != nil:
			return nil, nil, status.Errorf(codes.Unknown, "failed to get child schema for %T, field %s: %s", root, ft.Name, err)
		case cschema == nil:
			return nil, nil, status.Errorf(codes.NotFound, "could not find schema for type %T, field %s", root, ft.Name)
		default:
			if cschema, err = resolveLeafRef(cschema); err != nil {
				return nil, nil, status.Errorf(codes.Unknown, "failed to resolve schema for %T, field %s: %s", root, ft.Name, err)
			}
		}

		schPaths, err := util.SchemaPaths(ft)
		if err != nil {
			return nil, nil, status.Errorf(codes.Unknown, "failed to get schema paths for %T, field %s: %s", root, ft.Name, err)
		}

		for _, p := range schPaths {
			if !util.PathMatchesPrefix(path, p) {
				continue
			}
			to := len(p)
			if util.IsTypeMap(ft.Type) {
				to--
			}
			// If the node is a leaf node and path is exhausted, check whether val is set to a non-nil value. If
			// these are satisfied, the leaf value is updated.
			if (cschema.IsLeaf() || cschema.IsLeafList()) && len(path.Elem) == to && !util.IsValueNil(args.val) {
				return nil, nil, status.Errorf(codes.Unimplemented, "setting leaf/leaflist node is unimplemented")
			}

			if args.modifyRoot {
				if err := util.InitializeStructField(root, ft.Name); err != nil {
					return nil, nil, status.Errorf(codes.Unknown, "failed to initialize struct field %s in %T, child schema %v, path %v", ft.Name, root, cschema, path)
				}
			}
			return retrieveNode(cschema, fv.Interface(), util.TrimGNMIPathPrefix(path, p[0:to]), args)
		}
	}

	return nil, nil, status.Errorf(codes.NotFound, "no match found in %T, for path %v", root, path)
}

// retrieveNodeList is an internal function and operates on a map. It returns the nodes matching
// with keys corresponding to the key supplied in path.
// Function returns list of nodes, list of schemas and error.
func retrieveNodeList(schema *yang.Entry, root interface{}, path *gpb.Path, args retrieveNodeArgs) ([]interface{}, []*yang.Entry, error) {
	rv := reflect.ValueOf(root)
	switch {
	case schema.Key == "":
		return nil, nil, status.Errorf(codes.InvalidArgument, "unkeyed list can't be traversed, type %T, path %v", root, path)
	case len(path.GetElem()) == 0:
		return nil, nil, status.Errorf(codes.InvalidArgument, "path length is 0, schema %v, root %v", schema, root)
	case path.GetElem()[0].GetKey() == nil:
		return nil, nil, status.Errorf(codes.InvalidArgument, "path %v at %T points to a list without a key element", path, root)
	case !util.IsValueMap(rv):
		return nil, nil, status.Errorf(codes.InvalidArgument, "root has type %T, expect map", root)
	}

	var matchNodes []interface{}
	var matchSchemas []*yang.Entry

	listKeyT := rv.Type().Key()
	listElemT := rv.Type().Elem()
	for _, k := range rv.MapKeys() {
		listElemV := rv.MapIndex(k)

		// Handle lists with a single key.
		if !util.IsValueStruct(k) {
			// Handle the special case that we have zero keys specified only when we are handling lists
			// with partial keys specified.
			if len(path.GetElem()[0].GetKey()) == 0 && args.partialKeyMatch {
				nodes, schemas, err := retrieveNode(schema, listElemV.Interface(), util.PopGNMIPath(path), args)
				if err != nil {
					return nil, nil, err
				}

				matchNodes = append(matchNodes, nodes...)
				matchSchemas = append(matchSchemas, schemas...)

				continue
			}

			// Otherwise, check for equality of the key.
			pathKey, ok := path.GetElem()[0].GetKey()[schema.Key]
			if !ok {
				return nil, nil, status.Errorf(codes.NotFound, "schema key %s is not found in gNMI path %v, root %T", schema.Key, path, root)
			}

			kv, err := getKeyValue(listElemV.Elem(), schema.Key)
			if err != nil {
				return nil, nil, status.Errorf(codes.Unknown, "failed to get key value for %v, path %v: %v", listElemV.Interface(), path, err)
			}
			if fmt.Sprint(kv) == pathKey {
				return retrieveNode(schema, listElemV.Interface(), util.PopGNMIPath(path), args)
			}
			continue
		}

		match := true
		for i := 0; i < k.NumField(); i++ {
			fieldName := listKeyT.Field(i).Name
			fieldValue := k.Field(i)
			if !fieldValue.IsValid() {
				return nil, nil, status.Errorf(codes.InvalidArgument, "invalid field %s in %T", fieldName, k)
			}

			elemFieldT, ok := listElemT.Elem().FieldByName(fieldName)
			if !ok {
				return nil, nil, status.Errorf(codes.NotFound, "element struct type %v does not contain key field %s", listElemT, fieldName)
			}

			schemaKey, err := directDescendantSchema(elemFieldT)
			if err != nil {
				return nil, nil, status.Errorf(codes.Unknown, "unable to get direct descendant schema name for %v: %v", schemaKey, err)
			}

			pathKey, ok := path.GetElem()[0].GetKey()[schemaKey]
			// If key isn't found in the path key, treat it as error if partialKeyMatch is set to false.
			// Otherwise, continue searching other keys of key struct and count the value as match
			// if other keys are also match.
			switch {
			case !ok && !args.partialKeyMatch:
				return nil, nil, status.Errorf(codes.NotFound, "gNMI path %v does not contain a map entry for schema %v, root %T", path, schemaKey, root)
			case !ok && args.partialKeyMatch:
				// If the key wasn't specified, then skip the comparison of value.
				continue
			}
			keyAsString, err := ygot.KeyValueAsString(fieldValue.Interface())
			if err != nil {
				return nil, nil, status.Errorf(codes.Unknown, "failed to convert the field value to string, field %v: %v", fieldName, err)
			}
			if pathKey != keyAsString {
				match = false
				break
			}
		}

		if match {
			nodes, schemas, err := retrieveNode(schema, listElemV.Interface(), util.PopGNMIPath(path), args)
			if err != nil {
				return nil, nil, err
			}

			if nodes != nil {
				matchNodes = append(matchNodes, nodes...)
				matchSchemas = append(matchSchemas, schemas...)
			}
		}
	}

	if len(matchNodes) == 0 && args.modifyRoot {
		key, err := insertAndGetKey(schema, root, path.GetElem()[0].GetKey())
		if err != nil {
			return nil, nil, err
		}
		nodes, schemas, err := retrieveNode(schema, rv.MapIndex(reflect.ValueOf(key)).Interface(), util.PopGNMIPath(path), args)
		if err != nil {
			return nil, nil, err
		}
		matchNodes = append(matchNodes, nodes...)
		matchSchemas = append(matchSchemas, schemas...)
	}

	return matchNodes, matchSchemas, nil
}

// GetOrCreateNode function retrieves the node specified by the supplied path from the root which must have the
// schema supplied. It strictly matches keys in the path, in other words doesn't treat partial match as match.
// However, if there is no match, a new entry in the map is created. GetOrCreateNode also initializes the nodes
// along the path if they are nil.
// Function returns the value and schema of the node as well as error.
// Note that this function may modify the supplied root even if the function fails.
func GetOrCreateNode(schema *yang.Entry, root interface{}, path *gpb.Path) (interface{}, *yang.Entry, error) {
	nodes, schemas, err := retrieveNode(schema, root, path, retrieveNodeArgs{modifyRoot: true})
	if err != nil {
		return nil, nil, err
	}

	// There must be a result as this function initializes nodes along the supplied path.
	return nodes[0], schemas[0], nil
}

// TreeNode wraps an individual entry within a YANG data tree to return to a caller.
type TreeNode struct {
	// Schema is the schema entry for the data tree node, specified as a goyang Entry struct.
	Schema *yang.Entry
	// Data is the data node found at the path.
	Data interface{}
}

// sliceTreeNode returns a slice of TreeNodes corresponding to the input data and schema slices. The
// data and schema slices must have the same order, such that an entry at index 0 corresponds to
// the same node within the YANG data tree.
func sliceTreeNode(data []interface{}, schema []*yang.Entry) ([]*TreeNode, error) {
	if len(data) != len(schema) {
		return nil, fmt.Errorf("invalid input, slices are different lengths data %d, schema %d", len(data), len(schema))
	}

	tn := []*TreeNode{}
	for i := range data {
		tn = append(tn, &TreeNode{
			Schema: schema[i],
			Data:   data[i],
		})
	}
	return tn, nil
}

// GetNode retrieves the node specified by the supplied path from the specified root, whose schema must
// also be supplied. It takes a set of options which can be used to specify get behaviours, such as
// allowing partial match. If there are no matches for the path, an error is returned.
func GetNode(schema *yang.Entry, root interface{}, path *gpb.Path, opts ...GetNodeOpt) ([]*TreeNode, error) {
	nodes, schemas, err := retrieveNode(schema, root, path, retrieveNodeArgs{
		// We never want to modify the input root, so we specify modifyRoot.
		modifyRoot:      false,
		partialKeyMatch: hasPartialKeyMatch(opts),
	})

	if err != nil {
		return nil, err
	}

	tn, err := sliceTreeNode(nodes, schemas)
	if err != nil {
		return nil, status.Newf(codes.Internal, "invalid GetNode result: %v", err).Err()
	}

	return tn, nil
}

// GetNodeOpt defines an interface that can be used to supply arguments to functions using GetNode.
type GetNodeOpt interface {
	// IsGetNodeOpt is a marker method that is used to identify an instance of GetNodeOpt.
	IsGetNodeOpt()
}

// GetPartialKeyMatch specifies that a match within GetNode should be allowed to partially match
// keys for list entries.
type GetPartialKeyMatch struct{}

// IsGetNodeOpt implements the GetNodeOpt interface.
func (*GetPartialKeyMatch) IsGetNodeOpt() {}

// hasPartialKeyMatch determines whether there is an instance of GetPartialKeyMatch within the supplied
// GetNodeOpt slice. It is used to determine whether partial key matches should be allowed in an operation
// involving a GetNode.
func hasPartialKeyMatch(opts []GetNodeOpt) bool {
	for _, o := range opts {
		if _, ok := o.(*GetPartialKeyMatch); ok {
			return true
		}
	}
	return false
}
