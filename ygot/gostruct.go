package ygot

import (
	"reflect"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
)

const (
	// GoCompressedLeafAnnotation is the yang.Entry annotation name to
	// indicate that a particular leaf entry has a sibling that is
	// compressed out.
	GoCompressedLeafAnnotation = "ygot-oc-compressed-leaf"
)

// PruneConfigFalse in-place removes branches or leaf nodes that represent
// derived state for compressed GoStructs, and branches or leaf nodes that
// contain "config false" data for uncompressed GoStructs.
//
// Derived state is the subset of all operational state, or equivalently,
// "config false" nodes in the YANG definition, where the data is generated as
// part of the system's own interactions, rather than to reflect under what
// configuration the system is operating, the latter also known as applied
// configuration. These nodes are identifiable in YANG as the subset of "config
// false" nodes that do not have a sibling "config true" node in OpenConfig
// YANG models.
//
// The distinction between compressed and uncompressed GoStructs is due to the
// intrinsic non-existence of either intended configuration or applied
// configuration in the GoStruct, making pruning derived state the more useful
// operation for compressed GoStructs.
//
// The behaviour of this function is the same between compressed GoStructs
// generated using PreferIntendedConfig or PreferOperationalState, since
// compression behaviour doesn't affect derived state data.
//
// If the input GoStruct is itself to be entirely pruned, then instead, all of
// its fields will be removed.
//
// This function assumes that there should not be "config true" leaves
// underneath a "config false" branch, per RFC7950
// (https://datatracker.ietf.org/doc/html/rfc7950#section-7.21.1).
func PruneConfigFalse(schema *yang.Entry, s GoStruct) error {
	pruneReadOnlyIterFunc := func(ni *util.NodeInfo, in, out interface{}) util.Errors {
		if ni == nil || util.IsNilOrInvalidValue(ni.FieldValue) || ni.FieldValue.IsZero() {
			return nil
		}
		if util.IsConfig(ni.Schema) {
			return nil
		}
		if ni.Schema.Annotation[GoCompressedLeafAnnotation] != nil {
			return nil
		}
		// The top-level GoStruct cannot be written to since it is
		// unaddressable, so the best we can do is to skip writing to
		// it, and prune its children.
		if ni.Parent == nil {
			return nil
		}
		ni.FieldValue.Set(reflect.Zero(ni.FieldValue.Type()))
		return nil
	}
	if errs := util.ForEachField(schema, s, nil, nil, pruneReadOnlyIterFunc); errs != nil {
		return errs
	}
	return nil
}
