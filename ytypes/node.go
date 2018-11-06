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
	"reflect"

	"github.com/golang/protobuf/proto"
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
func retrieveNode(schema *yang.Entry, root interface{}, path, traversedPath *gpb.Path, args retrieveNodeArgs) ([]*TreeNode, error) {
	switch {
	case path == nil || len(path.Elem) == 0:
		// When args.val is non-nil and the schema isn't nil, further check whether
		// the node has a non-leaf schema. Setting a non-leaf schema isn't allowed.
		if !util.IsValueNil(args.val) && schema != nil {
			if !(schema.IsLeaf() || schema.IsLeafList()) {
				return nil, status.Errorf(codes.Unknown, "path %v points to a node with non-leaf schema %v", traversedPath, schema)
			}
		}
		return []*TreeNode{{
			Path:   traversedPath,
			Schema: schema,
			Data:   root,
		}}, nil
	case util.IsValueNil(root):
		return nil, status.Errorf(codes.NotFound, "could not find children %v at path %v", path, traversedPath)
	case schema == nil:
		return nil, status.Errorf(codes.InvalidArgument, "schema is nil for type %T, path %v", root, path)
	}

	switch {
	// Check if the schema is a container, or the schema is a list and the parent provided is a member of that list.
	case schema.IsContainer() || (schema.IsList() && util.IsTypeStructPtr(reflect.TypeOf(root))):
		return retrieveNodeContainer(schema, root, path, traversedPath, args)
	case schema.IsList():
		return retrieveNodeList(schema, root, path, traversedPath, args)
	}
	return nil, status.Errorf(codes.InvalidArgument, "can not use a parent that is not a container or list; schema %v root %T, path %v", schema, root, path)
}

// retrieveNodeContainer is an internal function and operates on GoStruct. It retrieves
// the node by the supplied path from the root which must have the schema supplied.
// It recurses by calling retrieveNode. If modifyRoot is set to true, nodes along the path are initialized
// if they are nil. If val isn't nil, then it is set on the leaf or leaflist node.
// Note that root is modified even if function returns error status.
func retrieveNodeContainer(schema *yang.Entry, root interface{}, path *gpb.Path, traversedPath *gpb.Path, args retrieveNodeArgs) ([]*TreeNode, error) {
	rv := reflect.ValueOf(root)
	if !util.IsTypeStructPtr(rv.Type()) {
		return nil, status.Errorf(codes.InvalidArgument, "got %T, want struct ptr root in retrieveNodeContainer", root)
	}

	// dereference reflect value as it points to a pointer.
	v := rv.Elem()

	for i := 0; i < v.NumField(); i++ {
		fv, ft := v.Field(i), v.Type().Field(i)

		cschema, err := childSchema(schema, ft)
		if !util.IsYgotAnnotation(ft) {
			switch {
			case err != nil:
				return nil, status.Errorf(codes.Unknown, "failed to get child schema for %T, field %s: %s", root, ft.Name, err)
			case cschema == nil:
				return nil, status.Errorf(codes.InvalidArgument, "could not find schema for type %T, field %s", root, ft.Name)
			default:
				if cschema, err = resolveLeafRef(cschema); err != nil {
					return nil, status.Errorf(codes.Unknown, "failed to resolve schema for %T, field %s: %s", root, ft.Name, err)
				}
			}
		}

		schPaths, err := util.SchemaPaths(ft)
		if err != nil {
			return nil, status.Errorf(codes.Unknown, "failed to get schema paths for %T, field %s: %s", root, ft.Name, err)
		}

		for _, p := range schPaths {
			if !util.PathMatchesPrefix(path, p) {
				continue
			}
			to := len(p)
			if util.IsTypeMap(ft.Type) {
				to--
			}

			if args.modifyRoot {
				if err := util.InitializeStructField(root, ft.Name); err != nil {
					return nil, status.Errorf(codes.Unknown, "failed to initialize struct field %s in %T, child schema %v, path %v", ft.Name, root, cschema, path)
				}
			}

			// If val in args is set to a non-nil value and the path is exhausted, we
			// may be dealing with a leaf or leaf list node. We should set the val
			// to the corresponding field in GoStruct. If the field is an annotation,
			// the field doesn't have a schema, so it is handled seperately.
			if !util.IsValueNil(args.val) && len(path.Elem) == to {
				switch {
				case util.IsYgotAnnotation(ft):
					if err := util.UpdateField(root, ft.Name, args.val); err != nil {
						return nil, status.Errorf(codes.Unknown, "failed to update struct field %s in %T with value %v, because of %v", ft.Name, root, args.val, err)
					}
				case cschema.IsLeaf() || cschema.IsLeafList():
					// With GNMIEncoding, unmarshalGeneric can only unmarshal leaf or leaf list
					// nodes. Schema provided must be the schema of the leaf or leaf list node.
					// root must be the reference of container leaf/leaf list belongs to.
					if err := unmarshalGeneric(cschema, root, args.val, GNMIEncoding); err != nil {
						return nil, status.Errorf(codes.Unknown, "failed to update struct field %s in %T with value %T; %v", ft.Name, root, args.val, err)
					}
				}
			}

			np := &gpb.Path{}
			if traversedPath != nil {
				np = proto.Clone(traversedPath).(*gpb.Path)
			}
			for i := range p[0:to] {
				np.Elem = append(np.Elem, path.GetElem()[i])
			}
			return retrieveNode(cschema, fv.Interface(), util.TrimGNMIPathPrefix(path, p[0:to]), np, args)
		}
	}

	return nil, status.Errorf(codes.InvalidArgument, "no match found in %T, for path %v", root, path)
}

// retrieveNodeList is an internal function and operates on a map. It returns the nodes matching
// with keys corresponding to the key supplied in path.
// Function returns list of nodes, list of schemas and error.
func retrieveNodeList(schema *yang.Entry, root interface{}, path, traversedPath *gpb.Path, args retrieveNodeArgs) ([]*TreeNode, error) {
	rv := reflect.ValueOf(root)
	switch {
	case schema.Key == "":
		return nil, status.Errorf(codes.InvalidArgument, "unkeyed list can't be traversed, type %T, path %v", root, path)
	case len(path.GetElem()) == 0:
		return nil, status.Errorf(codes.InvalidArgument, "path length is 0, schema %v, root %v", schema, root)
	case !util.IsValueMap(rv):
		return nil, status.Errorf(codes.InvalidArgument, "root has type %T, expect map", root)
	}

	var matches []*TreeNode

	listKeyT := rv.Type().Key()
	listElemT := rv.Type().Elem()
	for _, k := range rv.MapKeys() {
		listElemV := rv.MapIndex(k)

		// Handle lists with a single key.
		if !util.IsValueStruct(k) {
			// Handle the special case that we have zero keys specified only when we are handling lists
			// with partial keys specified.
			if len(path.GetElem()[0].GetKey()) == 0 && args.partialKeyMatch {
				keys, err := ygot.PathKeyFromStruct(listElemV)
				if err != nil {
					return nil, status.Errorf(codes.Unknown, "could not get path keys at %v: %v", traversedPath, err)
				}
				nodes, err := retrieveNode(schema, listElemV.Interface(), util.PopGNMIPath(path), appendElem(traversedPath, &gpb.PathElem{Name: path.GetElem()[0].Name, Key: keys}), args)
				if err != nil {
					return nil, err
				}

				matches = append(matches, nodes...)

				continue
			}

			// Otherwise, check for equality of the key.
			pathKey, ok := path.GetElem()[0].GetKey()[schema.Key]
			if !ok {
				return nil, status.Errorf(codes.NotFound, "schema key %s is not found in gNMI path %v, root %T", schema.Key, path, root)
			}

			kv, err := getKeyValue(listElemV.Elem(), schema.Key)
			if err != nil {
				return nil, status.Errorf(codes.Unknown, "failed to get key value for %v, path %v: %v", listElemV.Interface(), path, err)
			}

			keyAsString, err := ygot.KeyValueAsString(kv)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "failed to convert %v to a string, path %v: %v", kv, path, err)
			}
			if keyAsString == pathKey {
				return retrieveNode(schema, listElemV.Interface(), util.PopGNMIPath(path), appendElem(traversedPath, path.GetElem()[0]), args)
			}
			continue
		}

		match := true
		for i := 0; i < k.NumField(); i++ {
			fieldName := listKeyT.Field(i).Name
			fieldValue := k.Field(i)
			if !fieldValue.IsValid() {
				return nil, status.Errorf(codes.InvalidArgument, "invalid field %s in %T", fieldName, k)
			}

			elemFieldT, ok := listElemT.Elem().FieldByName(fieldName)
			if !ok {
				return nil, status.Errorf(codes.NotFound, "element struct type %v does not contain key field %s", listElemT, fieldName)
			}

			schemaKey, err := directDescendantSchema(elemFieldT)
			if err != nil {
				return nil, status.Errorf(codes.Unknown, "unable to get direct descendant schema name for %v: %v", schemaKey, err)
			}

			pathKey, ok := path.GetElem()[0].GetKey()[schemaKey]
			// If key isn't found in the path key, treat it as error if partialKeyMatch is set to false.
			// Otherwise, continue searching other keys of key struct and count the value as match
			// if other keys are also match.
			switch {
			case !ok && !args.partialKeyMatch:
				return nil, status.Errorf(codes.NotFound, "gNMI path %v does not contain a map entry for schema %v, root %T", path, schemaKey, root)
			case !ok && args.partialKeyMatch:
				// If the key wasn't specified, then skip the comparison of value.
				continue
			}
			keyAsString, err := ygot.KeyValueAsString(fieldValue.Interface())
			if err != nil {
				return nil, status.Errorf(codes.Unknown, "failed to convert the field value to string, field %v: %v", fieldName, err)
			}
			if pathKey != keyAsString {
				match = false
				break
			}
		}

		if match {
			keys, err := ygot.PathKeyFromStruct(listElemV)
			if err != nil {
				return nil, status.Errorf(codes.Unknown, "could not extract keys from %v: %v", traversedPath, err)
			}
			nodes, err := retrieveNode(schema, listElemV.Interface(), util.PopGNMIPath(path), appendElem(traversedPath, &gpb.PathElem{Name: path.GetElem()[0].Name, Key: keys}), args)
			if err != nil {
				return nil, err
			}

			if nodes != nil {
				matches = append(matches, nodes...)
			}
		}
	}

	if len(matches) == 0 && args.modifyRoot {
		key, err := insertAndGetKey(schema, root, path.GetElem()[0].GetKey())
		if err != nil {
			return nil, err
		}
		nodes, err := retrieveNode(schema, rv.MapIndex(reflect.ValueOf(key)).Interface(), util.PopGNMIPath(path), appendElem(traversedPath, path.GetElem()[0]), args)
		if err != nil {
			return nil, err
		}
		matches = append(matches, nodes...)
	}

	return matches, nil
}

// GetOrCreateNode function retrieves the node specified by the supplied path from the root which must have the
// schema supplied. It strictly matches keys in the path, in other words doesn't treat partial match as match.
// However, if there is no match, a new entry in the map is created. GetOrCreateNode also initializes the nodes
// along the path if they are nil.
// Function returns the value and schema of the node as well as error.
// Note that this function may modify the supplied root even if the function fails.
func GetOrCreateNode(schema *yang.Entry, root interface{}, path *gpb.Path) (interface{}, *yang.Entry, error) {
	nodes, err := retrieveNode(schema, root, path, nil, retrieveNodeArgs{modifyRoot: true})
	if err != nil {
		return nil, nil, err
	}

	// There must be a result as this function initializes nodes along the supplied path.
	return nodes[0].Data, nodes[0].Schema, nil
}

// TreeNode wraps an individual entry within a YANG data tree to return to a caller.
type TreeNode struct {
	// Schema is the schema entry for the data tree node, specified as a goyang Entry struct.
	Schema *yang.Entry
	// Data is the data node found at the path.
	Data interface{}
	// Path is the path of the data node that is being returned.
	Path *gpb.Path
}

// GetNode retrieves the node specified by the supplied path from the specified root, whose schema must
// also be supplied. It takes a set of options which can be used to specify get behaviours, such as
// allowing partial match. If there are no matches for the path, an error is returned.
func GetNode(schema *yang.Entry, root interface{}, path *gpb.Path, opts ...GetNodeOpt) ([]*TreeNode, error) {
	return retrieveNode(schema, root, path, nil, retrieveNodeArgs{
		// We never want to modify the input root, so we specify modifyRoot.
		modifyRoot:      false,
		partialKeyMatch: hasPartialKeyMatch(opts),
	})
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

// appendElem adds the element e to the path p and returns the resulting
// path.
func appendElem(p *gpb.Path, e *gpb.PathElem) *gpb.Path {
	np := &gpb.Path{}
	if p != nil {
		np = proto.Clone(p).(*gpb.Path)
	}
	np.Elem = append(np.Elem, e)
	return np
}

// SetNode sets the value of the node specified by the supplied path from the specified root,
// whose schema must also be supplied. It takes a set of options which can be used to specify set
// behaviours, such as whether or not to ensure that the node's ancestors are initialized.
func SetNode(schema *yang.Entry, root interface{}, path *gpb.Path, val interface{}, opts ...SetNodeOpt) error {
	nodes, err := retrieveNode(schema, root, path, nil, retrieveNodeArgs{
		modifyRoot: hasInitMissingElements(opts),
		val:        val,
	})

	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		return status.Errorf(codes.NotFound, "unable to find any nodes for the given path %v", path)
	}

	return nil
}

// SetNodeOpt defines an interface that can be used to supply arguments to functions using SetNode.
type SetNodeOpt interface {
	// IsSetNodeOpt is a marker method that is used to identify an instance of SetNodeOpt.
	IsSetNodeOpt()
}

// InitMissingElements signals SetNode to initialize the node's ancestors and to ensure that keys are added
// into keyed lists(maps) if they are missing, before updating the node.
type InitMissingElements struct{}

// IsSetNodeOpt implements the SetNodeOpt interface.
func (*InitMissingElements) IsSetNodeOpt() {}

// hasInitMissingElements determines whether there is an instance of InitMissingElements within the supplied
// SetNodeOpt slice. It is used to determine whether to initialize the node's ancestors before updating the node.
func hasInitMissingElements(opts []SetNodeOpt) bool {
	for _, o := range opts {
		if _, ok := o.(*InitMissingElements); ok {
			return true
		}
	}
	return false
}
