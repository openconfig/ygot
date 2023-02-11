package gnmidiff

import (
	"fmt"

	"github.com/derekparker/trie"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ytypes"
)

// SetToNotifsDiff contains the difference from the SetRequest to the given Notifications.
//
// - The key of the maps is the string representation of a gpb.Path constructed
// by ygot.PathToString.
// - The value of the update fields is the JSON_IETF representation of the
// value.
// TODO: Format function
type SetToNotifsDiff struct {
	// MissingUpdates are updates specified in the SetRequest missing in
	// the input Notifications.
	MissingUpdates map[string]interface{}
	// ExtraUpdates are updates not specified in the SetRequest that's
	// present in the input Notifications.
	ExtraUpdates      map[string]interface{}
	CommonUpdates     map[string]interface{}
	MismatchedUpdates map[string]MismatchedUpdate
}

// DiffSetRequestToNotifications returns a diff between a SetRequest and a
// slice of Notifications representing the state of the target after applying
// the SetRequest.
//
// newSchemaFn is intended to be provided via the function defined in generated
// ygot code (e.g. exampleoc.Schema).
// If newSchemaFn is not supplied, then any input JSON values MUST conform to the OpenConfig
// YANG style guidelines. See the following for checking compliance.
// * https://github.com/openconfig/oc-pyang
// * https://github.com/openconfig/public/blob/master/doc/openconfig_style_guide.md
func DiffSetRequestToNotifications(setreq *gpb.SetRequest, notifs []*gpb.Notification, newSchemaFn func() (*ytypes.Schema, error)) (SetToNotifsDiff, error) {
	setIntent, err := minimalSetRequestIntent(setreq, newSchemaFn)
	if err != nil {
		return SetToNotifsDiff{}, fmt.Errorf("DiffSetRequestToNotifications while calculating setIntent: %v", err)
	}
	diff := SetToNotifsDiff{
		MissingUpdates:    map[string]interface{}{},
		ExtraUpdates:      map[string]interface{}{},
		CommonUpdates:     map[string]interface{}{},
		MismatchedUpdates: map[string]MismatchedUpdate{},
	}

	updateIntent := setRequestIntent{
		Deletes: map[string]struct{}{},
		Updates: map[string]interface{}{},
	}
	// TODO: Handle prefix in notification
	for _, notif := range notifs {
		// TODO: Handle deletes in notification.
		if len(notif.Delete) > 0 {
			return SetToNotifsDiff{}, fmt.Errorf("Deletes in notifications not currently supported.")
		}
		for _, upd := range notif.Update {
			if err := populateUpdate(&updateIntent, upd, newSchemaFn, false); err != nil {
				return SetToNotifsDiff{}, err
			}
		}
	}
	updates := updateIntent.Updates

	for pathA, vA := range setIntent.Updates {
		vB, ok := updates[pathA]
		switch {
		case ok && vB != vA:
			diff.MismatchedUpdates[pathA] = MismatchedUpdate{A: vA, B: vB}
		case ok:
			diff.CommonUpdates[pathA] = vA
		default:
			diff.MissingUpdates[pathA] = vA
		}
		delete(updates, pathA)
	}

	t := trie.New()
	for pathB, vB := range updates {
		t.Add(pathB, vB)
	}
	for delPath := range setIntent.Deletes {
		// TODO: handle wildcards in delete paths (if applicable).
		for _, extraPath := range t.PrefixSearch(delPath + "/") {
			diff.ExtraUpdates[extraPath] = updates[extraPath]
		}
	}

	return diff, nil
}
