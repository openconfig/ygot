package ytypes

import (
	"fmt"
	"reflect"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// UnmarshalNotifications unmarshals a Notification on the root GoStruct
// specified by "schema".
//
// It does not make a copy and overwrites this value, so make a copy using
// ygot.DeepCopy() if you wish to retain the value at schema.Root prior to
// calling this function.
//
// - If preferShadowPath is specified, then the shadow path values are
// unmarshalled instead of non-shadow path values when GoStructs are generated
// with shadow paths.
// - If skipValidation is specified, then schema validation won't be performed
// after all the notifications have been unmarshalled.
func UnmarshalNotifications(schema *Schema, ns []*gpb.Notification, skipValidation bool, opts ...UnmarshalOpt) error {
	for _, n := range ns {
		err := UnmarshalSetRequest(schema, &gpb.SetRequest{
			Prefix: n.Prefix,
			Delete: n.Delete,
			Update: n.Update,
		}, true, opts...)
		if err != nil {
			return err
		}
	}

	root := schema.Root
	if !skipValidation {
		if err := validateGoStruct(root); err != nil {
			return fmt.Errorf("sum of notifications is not schema-compliant: %v", err)
		}
	}
	return nil
}

// UnmarshalSetRequest applies a SetRequest on the root GoStruct specified by
// "schema".
//
// It does not make a copy and overwrites this value, so make a copy using
// ygot.DeepCopy() if you wish to retain the value at schema.Root prior to
// calling this function.
//
// - If preferShadowPath is specified, then the shadow path values are
// unmarshalled instead of non-shadow path values when GoStructs are generated
// with shadow paths.
// - If skipValidation is specified, then schema validation won't be performed
// after the set request has been unmarshalled.
func UnmarshalSetRequest(schema *Schema, req *gpb.SetRequest, skipValidation bool, opts ...UnmarshalOpt) error {
	preferShadowPath := hasPreferShadowPath(opts)
	ignoreExtraFields := hasIgnoreExtraFields(opts)

	root := schema.Root
	node, nodeName, err := getOrCreateNode(schema.RootSchema(), root, req.Prefix, preferShadowPath)
	if err != nil {
		return err
	}

	// Process deletes, then replace, then updates.
	if err := deletePaths(schema.SchemaTree[nodeName], node, req.Delete, preferShadowPath); err != nil {
		return err
	}
	if err := replacePaths(schema.SchemaTree[nodeName], node, req.Replace, preferShadowPath, ignoreExtraFields); err != nil {
		return err
	}
	if err := updatePaths(schema.SchemaTree[nodeName], node, req.Update, preferShadowPath, ignoreExtraFields); err != nil {
		return err
	}

	if !skipValidation {
		if err := validateGoStruct(root); err != nil {
			return fmt.Errorf("SetRequest is not schema-compliant: %v", err)
		}
	}
	return nil
}

func validateGoStruct(goStruct ygot.GoStruct) error {
	vroot, ok := goStruct.(validatedGoStruct)
	if !ok {
		return fmt.Errorf("schema root cannot be validated: (%T, %v)", goStruct, goStruct)
	}
	return vroot.ΛValidate()
}

// validatedGoStruct is an interface used for validating GoStructs.
// This interface is implemented by all Go structs (YANG container or lists),
// regardless of generation flag.
type validatedGoStruct interface {
	// GoStruct ensures that the interface for a standard GoStruct
	// is embedded.
	ygot.GoStruct
	// ΛValidate compares the contents of the implementing struct against
	// the YANG schema, and returns an error if the struct's contents
	// are not valid, or nil if the struct complies with the schema.
	ΛValidate(...ygot.ValidationOption) error
}

// getOrCreateNode instantiates the node at the given path, and returns that
// node along with its name.
func getOrCreateNode(schema *yang.Entry, goStruct ygot.GoStruct, path *gpb.Path, preferShadowPath bool) (ygot.GoStruct, string, error) {
	var gcopts []GetOrCreateNodeOpt
	if preferShadowPath {
		gcopts = append(gcopts, &PreferShadowPath{})
	}
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
func deletePaths(schema *yang.Entry, goStruct ygot.GoStruct, paths []*gpb.Path, preferShadowPath bool) error {
	var dopts []DelNodeOpt
	if preferShadowPath {
		dopts = append(dopts, &PreferShadowPath{})
	}

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
func replacePaths(schema *yang.Entry, goStruct ygot.GoStruct, updates []*gpb.Update, preferShadowPath, ignoreExtraFields bool) error {
	var dopts []DelNodeOpt
	if preferShadowPath {
		dopts = append(dopts, &PreferShadowPath{})
	}

	for _, update := range updates {
		if err := DeleteNode(schema, goStruct, update.Path, dopts...); err != nil {
			return err
		}
		if err := setNode(schema, goStruct, update, preferShadowPath, ignoreExtraFields); err != nil {
			return err
		}
	}
	return nil
}

// updatePaths unmarshals a slice of updates into the given GoStruct. These
// updates can either by JSON-encoded or gNMI-encoded values (scalars).
func updatePaths(schema *yang.Entry, goStruct ygot.GoStruct, updates []*gpb.Update, preferShadowPath, ignoreExtraFields bool) error {
	for _, update := range updates {
		if err := setNode(schema, goStruct, update, preferShadowPath, ignoreExtraFields); err != nil {
			return err
		}
	}
	return nil
}

// setNode unmarshals either a JSON-encoded value or a gNMI-encoded (scalar)
// value into the given GoStruct.
func setNode(schema *yang.Entry, goStruct ygot.GoStruct, update *gpb.Update, preferShadowPath, ignoreExtraFields bool) error {
	sopts := []SetNodeOpt{&InitMissingElements{}}
	if preferShadowPath {
		sopts = append(sopts, &PreferShadowPath{})
	}
	if ignoreExtraFields {
		sopts = append(sopts, &IgnoreExtraFields{})
	}

	return SetNode(schema, goStruct, update.Path, update.Val, sopts...)
}
