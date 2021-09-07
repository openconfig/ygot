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

// Package testutil contains a set of utilities that are useful within
// tests of ygot-related data.
package testutil

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/gnmi/value"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

// ComparerOpt is an interface that all comparison options must implement.
type ComparerOpt interface {
	IsComparerOpt()
}

// IgnoreTimestamp is a comparison option that ignores timestamp values in
// gNMI messages.
type IgnoreTimestamp struct{}

// IsComparerOpt marks IgnoreTimestamp as a ComparerOpt.
func (IgnoreTimestamp) IsComparerOpt() {}

// hasIgnoreTimestamp determines whether the opt slice contains at least one
// instance of the IgnoreTimestamp option.
func hasIgnoreTimestamp(opts []ComparerOpt) bool {
	for _, o := range opts {
		if _, ok := o.(IgnoreTimestamp); ok {
			return true
		}
	}
	return false
}

// CustomComparer allows for a comparer for a particular type to be
// overloaded such that an external caller can inject a new way to
// compare a specific field of a gNMI message. It is a map, keyed by
// a reflect.Type of the message field, with a value of a cmp.Option
// produced by cmp.Comparer().
// NOTE: Proto comparison is done through protocmp -- this means any custom
// comparison of a proto field can only be done by supplying a protocmp-created
// cmp.Option.
type CustomComparer map[reflect.Type]cmp.Option

// IsComparerOpt marks CustomComparer as a ComparerOpt.
func (CustomComparer) IsComparerOpt() {}

// comparers resolves the comparers that are to be used for a particular
// operation -- it uses a default set and augments or replaces entries
// with those in any CustomComparer that is found within the opts slice.
func comparers(opts []ComparerOpt) []cmp.Option {
	cmps := map[reflect.Type]cmp.Option{
		reflect.TypeOf(&gnmipb.TypedValue{}): protocmp.FilterMessage(&gnmipb.TypedValue{},
			cmp.Comparer(func(a, b protocmp.Message) bool {
				if _, ok := a["json_ietf_val"]; ok {
					return JSONIETFComparer(a["json_ietf_val"].([]byte), b["json_ietf_val"].([]byte))
				}
				return cmp.Equal(a, b)
			})),
	}

	for _, o := range opts {
		if cc, ok := o.(CustomComparer); ok {
			for t, v := range cc {
				cmps[t] = v
			}
		}
	}

	currCmps := []cmp.Option{}
	for _, o := range cmps {
		currCmps = append(currCmps, o)
	}
	return currCmps
}

// GetResponseEqual compares the contents of a and b and returns true if they
// are equal. Extensions in the GetResponse are ignored. The supplied ComparerOpt
// options are used to influnce the equality comparison between a and b.
func GetResponseEqual(a, b *gnmipb.GetResponse, opts ...ComparerOpt) bool {
	return NotificationSetEqual(a.Notification, b.Notification, opts...)
}

// SubscribeResponseEqual compares the contents of a and b and returns true if
// they are equal. Extensions in the SubscribeResponse are ignored.
func SubscribeResponseEqual(a, b *gnmipb.SubscribeResponse) bool {
	switch {
	case a.GetUpdate() != nil && b.GetUpdate() == nil, b.GetUpdate() != nil && a.GetUpdate() == nil:
		return false
	case a.GetUpdate() != nil && b.GetUpdate() != nil:
		return NotificationSetEqual([]*gnmipb.Notification{a.GetUpdate()}, []*gnmipb.Notification{b.GetUpdate()})
	default:
		return a.GetSyncResponse() == b.GetSyncResponse()
	}
}

// SubscribeResponseSetEqual compares the contents of the slices of SubscribeResponse
// messages in a and b and returns true if they are equal. Order of the slices is
// ignored.
func SubscribeResponseSetEqual(a, b []*gnmipb.SubscribeResponse) bool {
	for _, ar := range a {
		var matched bool
		for _, br := range b {
			if SubscribeResponseEqual(ar, br) {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// NotificationSetEqual compares the contents of a and b and returns true if
// they are equal. Order of the slices is ignored. The set of ComparerOpts
// supplied are used to influnce the equality comparison between members
// of a and b.
func NotificationSetEqual(a, b []*gnmipb.Notification, opts ...ComparerOpt) bool {
	ignoreTS := hasIgnoreTimestamp(opts)
	cmps := comparers(opts)
	cmps = append(cmps, []cmp.Option{cmpopts.SortSlices(UpdateLess), cmpopts.EquateEmpty(), protocmp.Transform()}...)

	for _, an := range a {
		var matched bool
		for _, bn := range b {
			n := &notificationMatch{
				timestamp: true,
				prefix:    proto.Equal(an.GetPrefix(), bn.GetPrefix()),
				update: cmp.Equal(an.GetUpdate(), bn.GetUpdate(),
					cmps...,
				),
				delete: cmp.Equal(an.GetDelete(), bn.GetDelete(), cmpopts.SortSlices(PathLess), cmpopts.EquateEmpty(), protocmp.Transform()),
			}

			if !ignoreTS {
				n.timestamp = (an.GetTimestamp() == bn.GetTimestamp())
			}

			if n.matched() {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// JSONIETFComparer compares the two provided JSON IETF TypedValues to
// determine whether their contents are the same. If either value is
// invalid JSON, the function returns false.
func JSONIETFComparer(a, b []byte) bool {
	aj, bj := map[string]interface{}{}, map[string]interface{}{}
	if err := json.Unmarshal(a, &aj); err != nil {
		return false
	}

	if err := json.Unmarshal(b, &bj); err != nil {
		return false
	}

	return cmp.Equal(aj, bj)
}

// notificationMatch tracks whether a gNMI notification pair has matched.
type notificationMatch struct {
	timestamp bool
	prefix    bool
	update    bool
	delete    bool
}

// matched determines whether the receiver notificationMatch n represents
// a matched pair.
func (n *notificationMatch) matched() bool {
	return n.timestamp && n.prefix && n.update && n.delete
}

// UpdateSetEqual compares the contents of a and b and returns true if they are
// equal. Order of the slices is ignored.
func UpdateSetEqual(a, b []*gnmipb.Update) bool {
	return cmp.Equal(a, b, cmpopts.SortSlices(UpdateLess), cmpopts.EquateEmpty(), protocmp.Transform())
}

// updateSet is an alias for a slice of gNMI Update messages.
type updateSet []*gnmipb.Update

// Len, Less, and Swap implement the sort.Interface interface.
func (u updateSet) Len() int           { return len(u) }
func (u updateSet) Less(i, j int) bool { return UpdateLess(u[i], u[j]) }
func (u updateSet) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }

// pathSet is an alias for a slice of gNMI Path messages.
type pathSet []*gnmipb.Path

// Len, Less, and Swap implement the sort.Interface interface.
func (p pathSet) Len() int           { return len(p) }
func (p pathSet) Less(i, j int) bool { return PathLess(p[i], p[j]) }
func (p pathSet) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// NotificationLess compares the two notifications a and b, returning true if
// a is less than b, and false if not. Less is defined by:
//  - Comparing the timestamp.
//  - If equal timestamps, comparing the prefix using PathLess.
//  - If equal prefixes, comparing the Updates using UpdateLess.
//  - If equal updates, comparing the Deletes using deleteLess.
// If all fields are equal, the function returns false to ensure that the
// irreflexive property required by cmpopts.SortSlices is implemented.
func NotificationLess(a, b *gnmipb.Notification) bool {
	switch {
	case a == nil && b != nil:
		return true
	case a == nil && b == nil:
		// Ensure notification less meets the irreflexive property required by
		// cmpopts.
		return false
	case b == nil && a != nil:
		return false
	}

	if proto.Equal(a, b) {
		return false
	}

	if a.Timestamp != b.Timestamp {
		return a.Timestamp < b.Timestamp
	}

	if !proto.Equal(a.Prefix, b.Prefix) {
		return PathLess(a.Prefix, b.Prefix)
	}

	if !cmp.Equal(a.Update, b.Update, cmpopts.SortSlices(UpdateLess), cmpopts.EquateEmpty(), protocmp.Transform()) {
		if len(a.Update) < len(b.Update) {
			return true
		}
		if len(b.Update) < len(a.Update) {
			return false
		}

		// Don't modify the original data.
		sortedA, sortedB := proto.Clone(a).(*gnmipb.Notification), proto.Clone(b).(*gnmipb.Notification)
		sort.Sort(updateSet(sortedA.Update))
		sort.Sort(updateSet(sortedB.Update))

		for _, uA := range sortedA.Update {
			for _, uB := range sortedB.Update {
				if !proto.Equal(uA, uB) {
					return UpdateLess(uA, uB)
				}
			}
		}
	}

	if !cmp.Equal(a.Delete, b.Delete, cmpopts.SortSlices(PathLess), cmpopts.EquateEmpty(), protocmp.Transform()) {
		if len(a.Delete) < len(b.Delete) {
			return true
		}

		if len(b.Delete) < len(a.Delete) {
			return false
		}

		// Again, don't modify the original data.
		sortedA, sortedB := proto.Clone(a).(*gnmipb.Notification), proto.Clone(b).(*gnmipb.Notification)
		sort.Sort(pathSet(sortedA.Delete))
		sort.Sort(pathSet(sortedB.Delete))

		for _, dA := range sortedA.Delete {
			for _, dB := range sortedB.Delete {
				if !proto.Equal(dA, dB) {
					return PathLess(dA, dB)
				}
			}
		}
	}

	return true
}

// UpdateLess compares two gNMI Update messages and returns true if a < b.
// The less-than comparison is done by first comparing the paths of the updates,
// and subquently comparing the typedValue fields of the updates, followed by
// the duplicates fields. If all fields are equal, returns false.
func UpdateLess(a, b *gnmipb.Update) bool {
	if proto.Equal(a, b) {
		// If the two values are equal, return true to avoid the expense of checking
		// each field.
		return false
	}

	if !proto.Equal(a.Path, b.Path) {
		return PathLess(a.Path, b.Path)
	}

	if !proto.Equal(a.Val, b.Val) {
		return typedValueLess(a.Val, b.Val)
	}

	return a.Duplicates < b.Duplicates
}

// PathLess provides a function which determines whether a gNMI Path messages
// A is less than the gNMI Path message b. It can be used to allow sorting of
// gNMI path messages - for example, in cmpopts.SortSlices.
func PathLess(a, b *gnmipb.Path) bool {
	switch {
	case a == nil && b == nil:
		return false
	case a == nil && b != nil:
		return true
	case b == nil && a != nil:
		return false
	}

	if proto.Equal(a, b) {
		return false
	}

	if len(a.Elem) != len(b.Elem) {
		// Less specific paths are less than more specific ones.
		return len(a.Elem) > len(b.Elem)
	}

	for i := 0; i < len(a.Elem); i++ {
		ae, be := a.Elem[i], b.Elem[i]
		if ae.Name != be.Name {
			// If the name of the path element is not equal, then use
			// string comparison to determine whether a < b
			return ae.Name < be.Name
		}

		aKeys, bKeys := stringKeys(ae.Key), stringKeys(be.Key)
		sort.Strings(aKeys)
		sort.Strings(bKeys)

		if len(aKeys) != len(bKeys) {
			// Paths with more keys are considered less than paths
			// with fewer.
			return len(aKeys) < len(bKeys)
		}

		for j := 0; j < len(aKeys); j++ {
			ak, bk := aKeys[j], bKeys[j]
			if ak != bk {
				// If the sorted list of keys is not equal, then use string
				// comparison between the key names.
				return ak < bk
			}

			av, bv := ae.Key[ak], be.Key[bk]
			if av != bv {
				// If the key names match, use the value of the key to determine
				// equality.
				return av < bv
			}
		}
	}

	// Handle comparison of paths that are based on the "element" rather than
	// "elem".
	// TODO(robjs): Remove handling of element fields when they are fully
	// deprecated in gNMI.
	//
	//lint:ignore SA1019 Specifically handling deprecated gNMI Element fields.
	for len(a.Element) != len(b.Element) {
		//lint:ignore SA1019 Specifically handling deprecated gNMI Element fields.
		return len(a.Element) > len(b.Element)
	}

	//lint:ignore SA1019 Specifically handling deprecated gNMI Element fields.
	for i := 0; i < len(a.Element); i++ {
		//lint:ignore SA1019 Specifically handling deprecated gNMI Element fields.
		if ae, be := a.Element[i], b.Element[i]; ae != be {
			return ae < be
		}
	}

	// If the origin is not equal, then use comparison between the origin
	// string.
	if a.Origin != b.Origin {
		return a.Origin < b.Origin
	}

	// If the two Path messages are entirely equal, then deterministically
	// return b < a per the irreflexive property.
	return false
}

// stringKeys returns a slice of the keys of the supplied map m.
func stringKeys(m map[string]string) []string {
	ss := []string{}
	for k := range m {
		ss = append(ss, k)
	}
	return ss
}

// typedValueLess compares the value of the gNMI TypedValues a and b. If a < b,
// it returns true, otherwise it returns false. It can be used when comparing
// typed values for sorting purposes. If the value within the TypedValue message
// is not directly comparable, it formats it as a string and compares the two
// strings specified.
//
// If nil input is provided for either a or b, the nil value is considered
// less than the non-nil value. If both values are nil, b is considered less
// than a to implement the irreflexive property required by cmpopts.
func typedValueLess(a, b *gnmipb.TypedValue) bool {
	switch {
	case a == nil && b != nil:
		return false
	case b == nil && a != nil:
		return true
	case a == nil && b == nil:
		return false
	}

	// If the two types are not the same, then use their string representations
	// to make them comparable.
	aVal, bVal := a.GetValue(), b.GetValue()
	aType, bType := reflect.TypeOf(aVal), reflect.TypeOf(bVal)
	if aType != bType {
		return typedValueStringLess(reflect.ValueOf(aVal), reflect.ValueOf(bVal), aType, bType)
	}

	// Since a comparison method cannot return an error, we must handle all cases
	// where the type is not a scalar type - we do this be reverting to using
	// the string representation.
	canScalar := true
	aScalar, err := value.ToScalar(a)
	if err != nil {
		canScalar = false
	}

	bScalar, err := value.ToScalar(b)
	if err != nil {
		canScalar = false
	}

	if !canScalar {
		return typedValueStringLess(reflect.ValueOf(aVal), reflect.ValueOf(bVal), aType, bType)
	}

	switch aScalar.(type) {
	case string:
		return aScalar.(string) < bScalar.(string)
	case float32:
		return aScalar.(float32) < bScalar.(float32)
	case int64:
		return aScalar.(int64) < bScalar.(int64)
	case uint64:
		return aScalar.(uint64) < bScalar.(uint64)
	case bool:
		return boolLess(aScalar.(bool), bScalar.(bool))
	default:
		return typedValueStringLess(reflect.ValueOf(aVal), reflect.ValueOf(bVal), aType, bType)
	}
}

// typedValueStringLess takes two gNMI TypedValue.Value fields as their reflect.Value
// and reflect.Type representations and converts them to a string to compare them. It
// returns the value of the string less-than between the stringified a and b.
func typedValueStringLess(av, bv reflect.Value, at, bt reflect.Type) bool {
	ai, bi := av.Interface(), bv.Interface()
	if at.Kind() == reflect.Ptr {
		ai = av.Elem().Interface()
	}
	if bt.Kind() == reflect.Ptr {
		bi = bv.Elem().Interface()
	}

	return fmt.Sprintf("%v", ai) < fmt.Sprintf("%v", bi)
}

// boolLess implements a comparison  of the bools a and b. It returns true
// if a < b. The bool set to false is considered to be less than a bool set
// to true. If the values are equal, a is considered less than b.
func boolLess(a, b bool) bool {
	switch {
	case a && b, !a && !b:
		return true
	case a && !b:
		return false
	}
	return true
}
