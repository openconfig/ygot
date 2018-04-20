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
// testing of ygot-related data.
package testutil

import (
	"fmt"
	"reflect"
	"sort"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/value"
)

// pathLess provides a function which determines whether a gNMI Path messages
// A is less than the gNMI Path message b. It can be used to allow sorting of
// gNMI path messages - for example, in cmpopts.SortSlices.
func pathLess(a, b *gnmipb.Path) bool {
	if len(a.Elem) != len(b.Elem) {
		// Less specific paths are less than more specific ones.
		return len(a.Elem) > len(b.Elem)
	}

	for i := 0; i < len(a.Elem); i++ {
		ae, be := a.Elem[i], b.Elem[i]
		if ae.Name != be.Name {
			// If the name of the path element is not equal, then use
			// string comparison to determine whether a < b.
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

	// If the origin is not equal, then use comparison between the origin
	// string.
	if a.Origin != b.Origin {
		return a.Origin < b.Origin
	}

	// If the two Path messages are entirely equal, then deterministically
	// return a < b.
	return true
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
// less than the non-nil value. If both values are nil, a is considered less
// than b.
func typedValueLess(a, b *gnmipb.TypedValue) bool {
	switch {
	case a == nil && b != nil:
		return false
	case b == nil && a != nil:
		return true
	case a == nil && b == nil:
		return true
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
