package ytypes

import (
	"fmt"
	"reflect"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// UnmarshalNotifications unmarshals a slice of Notifications on the root
// GoStruct specified by "schema". It *does not* perform validation after
// unmarshalling is complete.
//
// It does not make a copy and instead overwrites this value, so make a copy
// using ygot.DeepCopy() if you wish to retain the value at schema.Root prior
// to calling this function.
//
// If an error occurs during unmarshalling, schema.Root may already be
// modified. A rollback is not performed.
func UnmarshalNotifications(schema *Schema, ns []*gpb.Notification, opts ...UnmarshalOpt) error {
	for _, n := range ns {
		err := UnmarshalSetRequest(schema, &gpb.SetRequest{
			Prefix: n.Prefix,
			Delete: n.Delete,
			Update: n.Update,
		}, opts...)
		if err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalSetRequest applies a SetRequest on the root GoStruct specified by
// "schema". It *does not* perform validation after unmarshalling is complete.
//
// It does not make a copy and instead overwrites this value, so make a copy
// using ygot.DeepCopy() if you wish to retain the value at schema.Root prior
// to calling this function.
//
// If an error occurs during unmarshalling, schema.Root may already be
// modified. A rollback is not performed.
func UnmarshalSetRequest(schema *Schema, req *gpb.SetRequest, opts ...UnmarshalOpt) error {
	if req == nil {
		return nil
	}

	// Use option slices instead of flags to pass options down to the function calls.
	var getOrCreateOpts []GetOrCreateNodeOpt
	var deleteOpts []DelNodeOpt
	var updateOpts []SetNodeOpt

	for _, opt := range opts {
		switch o := opt.(type) {
		case *PreferShadowPath:
			getOrCreateOpts = append(getOrCreateOpts, &PreferShadowPath{})
			deleteOpts = append(deleteOpts, &PreferShadowPath{})
			updateOpts = append(updateOpts, &PreferShadowPath{})
		case *IgnoreExtraFields:
			updateOpts = append(updateOpts, &IgnoreExtraFields{})
		case *NodeCacheOpt:
			getOrCreateOpts = append(getOrCreateOpts, o)
			updateOpts = append(updateOpts, o)
		}
	}

	root := schema.Root
	node, nodeName, err := getOrCreateNode(schema.RootSchema(), root, req.Prefix, getOrCreateOpts)
	if err != nil {
		return err
	}

	// Process deletes, then replace, then updates.
	if err := deletePaths(schema.SchemaTree[nodeName], node, req.Delete, deleteOpts); err != nil {
		return err
	}
	if err := replacePaths(schema.SchemaTree[nodeName], node, req.Replace, deleteOpts, updateOpts); err != nil {
		return err
	}
	if err := updatePaths(schema.SchemaTree[nodeName], node, req.Update, updateOpts); err != nil {
		return err
	}

	return nil
}

// getOrCreateNode instantiates the node at the given path, and returns that
// node along with its name.
func getOrCreateNode(schema *yang.Entry, goStruct ygot.GoStruct, path *gpb.Path, gcopts []GetOrCreateNodeOpt) (ygot.GoStruct, string, error) {
	// Operate at the prefix level.
	nodeI, _, err := GetOrCreateNode(schema, goStruct, path, gcopts...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to GetOrCreate the prefix node: %v", err)
	}
	node, ok := nodeI.(ygot.GoStruct)
	if !ok {
		return nil, "", fmt.Errorf("prefix path points to a non-GoStruct, this is not allowed: %T, %v", nodeI, nodeI)
	}

	return node, reflect.TypeOf(nodeI).Elem().Name(), nil
}

// deletePaths deletes a slice of paths from the given GoStruct.
func deletePaths(schema *yang.Entry, goStruct ygot.GoStruct, paths []*gpb.Path, dopts []DelNodeOpt) error {
	for _, path := range paths {
		if err := DeleteNode(schema, goStruct, path, dopts...); err != nil {
			return err
		}
	}
	return nil
}

// replacePaths unmarshals a slice of updates into the given GoStruct. It
// deletes the values at these paths before unmarshalling them. These updates
// can either by JSON-encoded or gNMI-encoded values (scalars).
func replacePaths(schema *yang.Entry, goStruct ygot.GoStruct, updates []*gpb.Update, dopts []DelNodeOpt, uopts []SetNodeOpt) error {
	for _, update := range updates {
		if err := DeleteNode(schema, goStruct, update.Path, dopts...); err != nil {
			return err
		}
		if err := setNode(schema, goStruct, update, uopts); err != nil {
			return err
		}
	}
	return nil
}

// updatePaths unmarshals a slice of updates into the given GoStruct. These
// updates can either by JSON-encoded or gNMI-encoded values (scalars).
func updatePaths(schema *yang.Entry, goStruct ygot.GoStruct, updates []*gpb.Update, uopts []SetNodeOpt) error {
	for _, update := range updates {
		if err := setNode(schema, goStruct, update, uopts); err != nil {
			return err
		}
	}
	return nil
}

// setNode unmarshals either a JSON-encoded value or a gNMI-encoded (scalar)
// value into the given GoStruct.
func setNode(schema *yang.Entry, goStruct ygot.GoStruct, update *gpb.Update, uopts []SetNodeOpt) error {
	return SetNode(schema, goStruct, update.Path, update.Val, append(uopts, &InitMissingElements{})...)
}
