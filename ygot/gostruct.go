package ygot

import (
	"fmt"
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

// PruneReadOnly removes branches or leaf nodes that contain "config false" data in-place.
//
// Note that the input GoStruct MUST NOT be read-only, since only anchored
// pointers can be written to; otherwise an error will be returned.
//
// The behaviour of this function is the same between GoStructs generated using
// prefer_operational_state=true or prefer_operational_state=false.
//
// Where a read-only branch is encountered, the entire branch is pruned. Since
// there should not be non-read-only leaves underneath a read-only branch, they
// are treated as read-only by PruneReadOnly
// (https://datatracker.ietf.org/doc/html/rfc7950#section-7.21.1).
func PruneReadOnly(schema *yang.Entry, s GoStruct) error {
	pruneReadOnlyIterFunc := func(ni *util.NodeInfo, in, out interface{}) util.Errors {
		if ni == nil || util.IsNilOrInvalidValue(ni.FieldValue) || ni.FieldValue.IsZero() {
			return nil
		}
		if !ni.Schema.ReadOnly() {
			return nil
		}
		if ni.Schema.Annotation[GoCompressedLeafAnnotation] != nil {
			return nil
		}
		if ni.Parent == nil {
			return util.NewErrs(fmt.Errorf("read-only node doesn't have a parent node: %s", ni.Schema.Path()))
		}
		ni.FieldValue.Set(reflect.Zero(ni.FieldValue.Type()))
		return nil
	}
	if errs := util.ForEachField(schema, s, nil, nil, pruneReadOnlyIterFunc); errs != nil {
		return errs
	}
	return nil
}
