// Copyright 2018 Google Inc.
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

// Package testcmp contains a functions that can be used alongside the Go cmp
// or ygot testutil packages to provide comparisons between particular gNMI
// or ygot data structures with more intelligence than the base cmp or proto.Equal
// functions.
package testcmp

import (
	"fmt"
	"reflect"

	log "github.com/golang/glog"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

// UpdateComparer returns a testutil.CustomComparer map that contains a comparer that
// binds the GNMIUpdateComparer function to be used to compare gNMI Update messages. It
// takes an argument of the Schema() function within a generated Go package which contains
// the relevant details to unmarshal JSON within the package.
func UpdateComparer(schemaFunc func() (*ytypes.Schema, error)) (testutil.CustomComparer, error) {
	schema, err := schemaFunc()
	if err != nil {
		return nil, fmt.Errorf("cannot extract schema from ygot package, %v", err)
	}

	return testutil.CustomComparer{
		reflect.TypeOf(&gnmipb.Update{}): cmp.Comparer(func(a, b *gnmipb.Update) bool {
			_, equal, err := GNMIUpdateComparer(a, b, schema)
			if err != nil {
				log.Errorf("cannot compare updates for Notifications, got err: %v", err)
			}
			return equal
		}),
	}, nil
}

// GNMIUpdateComparer compares the two gNMI Update messages, a and b, supplied. It takes the a ytypes.Schema
// definition of a generated Go package, which provides the fields required to unmarshal IETF JSON contents.
// It returns a gNMI Notification which reflects a diff between a and b (if it can be calculated), a bool
// indicating whether a == b and any error that is encountered.
func GNMIUpdateComparer(a, b *gnmipb.Update, jsonSpec *ytypes.Schema) (*gnmipb.Notification, bool, error) {
	if jsonSpec == nil || !jsonSpec.IsValid() {
		return nil, false, fmt.Errorf("JSON specification is not valid, %v", jsonSpec)
	}

	av, bv := a.GetVal().GetValue(), b.GetVal().GetValue()
	switch {
	case av == nil && bv == nil:
		// Equal, since both values are nil
		return nil, true, nil
	case av == nil, bv == nil:
		// Not equal, since one value is nil and the other is not.
		return nil, false, nil
	}

	if reflect.TypeOf(av) != reflect.TypeOf(bv) {
		// Not equal due to the type of TypedValue specified being different.
		return nil, false, nil
	}

	_, aOK := av.(*gnmipb.TypedValue_JsonIetfVal)
	_, bOK := av.(*gnmipb.TypedValue_JsonIetfVal)
	if !aOK || !bOK {
		// One or both of the updates doesn't contain a JSON IETF typed value
		// so revert to using cmp.Equal to test their equality.
		return nil, cmp.Equal(a, b, cmpopts.EquateEmpty()), nil
	}

	// Create a new root, since GetOrCreateNode can modify the root even during
	// a failure.
	rootA, err := newStruct(jsonSpec.Root)
	if err != nil {
		return nil, false, fmt.Errorf("cannot create new root struct, got err: %v", err)
	}

	aInterface, _, err := ytypes.GetOrCreateNode(jsonSpec.RootSchema(), rootA, a.Path)
	if err != nil {
		return nil, false, fmt.Errorf("cannot retrieve struct for path %s, err: %v", a.Path, err)
	}

	aStruct, ok := aInterface.(ygot.GoStruct)
	if !ok {
		return nil, false, fmt.Errorf("path %s with IETF JSON does not correspond to a struct", a.Path)
	}

	rootB, err := newStruct(jsonSpec.Root)
	if err != nil {
		return nil, false, fmt.Errorf("cannot create new root struct, got err: %v", err)
	}

	bInterface, _, err := ytypes.GetOrCreateNode(jsonSpec.RootSchema(), rootB, b.Path)
	if err != nil {
		return nil, false, fmt.Errorf("cannot retrieve struct for path %s, err: %v", b.Path, err)
	}

	bStruct, ok := bInterface.(ygot.GoStruct)
	if !ok {
		return nil, false, fmt.Errorf("path %s with IETF JSON does not correspond to a struct", b.Path)
	}

	if err := unmarshalStruct(a.GetVal(), aStruct, jsonSpec.Unmarshal); err != nil {
		return nil, false, fmt.Errorf("cannot unmarshal JSON for struct A, got err: %v", err)
	}

	if err := unmarshalStruct(b.GetVal(), bStruct, jsonSpec.Unmarshal); err != nil {
		return nil, false, fmt.Errorf("cannot unmarshal JSON for struct B, got err: %v", err)
	}

	diff, err := ygot.Diff(rootA, rootB)
	if err != nil {
		return nil, false, fmt.Errorf("cannot diff structs after unmarshalling, got err: %v", err)
	}

	if len(diff.Update) == 0 && len(diff.Delete) == 0 {
		// No diffs after unmarshalling -- so these values are equal.
		return nil, true, nil
	}

	return diff, false, nil
}

// newStruct returns a new copy of the supplied ygot.GoStruct.
func newStruct(t ygot.GoStruct) (ygot.GoStruct, error) {
	ni := reflect.New(reflect.TypeOf(t).Elem())
	n, ok := ni.Interface().(ygot.GoStruct)
	if !ok {
		return nil, fmt.Errorf("cannot create new instance of %T", t)
	}
	return n, nil
}

// unmarshalStruct unmarshal the JSON IETF field of the supplied TypedValue into
// the dst GoStruct using the supplied Unmarshal function.
func unmarshalStruct(v *gnmipb.TypedValue, dst ygot.GoStruct, ufn ytypes.UnmarshalFunc) error {
	jsonval, ok := v.GetValue().(*gnmipb.TypedValue_JsonIetfVal)
	if !ok {
		return fmt.Errorf("value did not contain IETF JSON")
	}
	if err := ufn(jsonval.JsonIetfVal, dst, &ytypes.IgnoreExtraFields{}); err != nil {
		return fmt.Errorf("cannot unmarshal %v", err)
	}
	return nil
}
