package gnmidiff

import (
	"fmt"
	"reflect"

	"github.com/derekparker/trie"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ytypes"
)

// SetToNotifsDiff contains the difference from the SetRequest to the given Notifications.
type SetToNotifsDiff UpdateDiff

// Format outputs the SetToNotifsDiff in human-readable format.
//
// NOTE: Do not depend on the output of this being stable.
func (diff SetToNotifsDiff) Format(f Format) string {
	f.title = "SetToNotifsDiff"
	f.aName = "want/SetRequest"
	f.bName = "got/Notifications"
	return StructuredDiff{UpdateDiff: UpdateDiff(diff)}.Format(f)
}

// DiffSetRequestToNotifications returns a diff between a SetRequest and a
// slice of Notifications representing the state of the target after applying
// the SetRequest.
//
// schema is intended to be provided via the function defined in generated
// ygot code (e.g. exampleoc.Schema).
// If schema is not supplied, then any input JSON values MUST conform to the OpenConfig
// YANG style guidelines. See the following for checking compliance.
// * https://github.com/openconfig/oc-pyang
// * https://github.com/openconfig/public/blob/master/doc/openconfig_style_guide.md
func DiffSetRequestToNotifications(setreq *gpb.SetRequest, notifs []*gpb.Notification, schema *ytypes.Schema) (SetToNotifsDiff, error) {
	setIntent, err := minimalSetRequestIntent(setreq, schema)
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
	for _, notif := range notifs {
		// TODO: Handle deletes in notification.
		if len(notif.Delete) > 0 {
			return SetToNotifsDiff{}, fmt.Errorf("Deletes in notifications not currently supported.")
		}
		prefix, err := prefixStr(notif.Prefix)
		if err != nil {
			return SetToNotifsDiff{}, fmt.Errorf("gnmidiff: %v", err)
		}
		for _, upd := range notif.Update {
			path, err := fullPathStr(prefix, upd.Path)
			if err != nil {
				return SetToNotifsDiff{}, err
			}
			if err := updateIntent.populateUpdate(path, upd.Val, schema, false); err != nil {
				return SetToNotifsDiff{}, err
			}
		}
	}
	updates := updateIntent.Updates

	for pathA, vA := range setIntent.Updates {
		vB, ok := updates[pathA]
		switch {
		case ok && !reflect.DeepEqual(vA, vB):
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
