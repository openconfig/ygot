// Copyright 2023 Google Inc.
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
	"fmt"
	"reflect"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
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
		deletePaths := n.Delete
		if n.Atomic {
			deletePaths = append(deletePaths, &gpb.Path{})
		}
		err := UnmarshalSetRequest(schema, &gpb.SetRequest{
			Prefix: n.Prefix,
			Delete: deletePaths,
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
	preferShadowPath := hasPreferShadowPath(opts)
	ignoreExtraFields := hasIgnoreExtraFields(opts)
	bestEffortUnmarshal := hasBestEffortUnmarshal(opts)
	if req == nil {
		return nil
	}
	root := schema.Root
	rootName := reflect.TypeOf(root).Elem().Name()

	var complianceErrs *ComplianceErrors

	// Process deletes, then replace, then updates.
	if err := deletePaths(schema.SchemaTree[rootName], root, req.Prefix, req.Delete, preferShadowPath, bestEffortUnmarshal); err != nil {
		if bestEffortUnmarshal {
			complianceErrs = complianceErrs.append(err.(*ComplianceErrors).Errors...)
		} else {
			return err
		}
	}
	if err := replacePaths(schema.SchemaTree[rootName], root, req.Prefix, req.Replace, preferShadowPath, ignoreExtraFields, bestEffortUnmarshal); err != nil {
		if bestEffortUnmarshal {
			complianceErrs = complianceErrs.append(err.(*ComplianceErrors).Errors...)
		} else {
			return err
		}
	}
	if err := updatePaths(schema.SchemaTree[rootName], root, req.Prefix, req.Update, preferShadowPath, ignoreExtraFields, bestEffortUnmarshal); err != nil {
		if bestEffortUnmarshal {
			complianceErrs = complianceErrs.append(err.(*ComplianceErrors).Errors...)
		} else {
			return err
		}
	}

	if bestEffortUnmarshal && complianceErrs != nil {
		return complianceErrs
	}
	return nil
}

// deletePaths deletes a slice of paths from the given GoStruct.
func deletePaths(schema *yang.Entry, goStruct ygot.GoStruct, prefix *gpb.Path, paths []*gpb.Path, preferShadowPath, bestEffortUnmarshal bool) error {
	var dopts []DelNodeOpt
	var ce *ComplianceErrors
	if preferShadowPath {
		dopts = append(dopts, &PreferShadowPath{})
	}

	for _, path := range paths {
		if prefix != nil {
			var err error
			if path, err = util.JoinPaths(prefix, path); err != nil {
				return fmt.Errorf("cannot join prefix with deletion path: %v", err)
			}
		}
		if err := DeleteNode(schema, goStruct, path, dopts...); err != nil {
			if bestEffortUnmarshal {
				ce = ce.append(err)
				continue
			}
			return err
		}
	}

	if bestEffortUnmarshal && ce != nil {
		return ce
	}
	return nil
}

// joinPrefixToUpdate returns a new update that has the prefix joined to the path.
//
// It guarantees to not change the original update, and preserves the .Val and
// .Path values.
func joinPrefixToUpdate(prefix *gpb.Path, update *gpb.Update) (*gpb.Update, error) {
	if prefix == nil {
		return update, nil
	}
	// Here we avoid doing a deep copy for performance.
	// Currently protobuf is missing a library function for a safe
	// shallow copy: https://github.com/golang/protobuf/issues/1155
	joinedUpdate := &gpb.Update{
		Val: update.Val,
	}
	var err error
	if joinedUpdate.Path, err = util.JoinPaths(prefix, update.Path); err != nil {
		return nil, fmt.Errorf("cannot join prefix with gpb.Update path: %v", err)
	}
	return joinedUpdate, nil
}

// replacePaths unmarshals a slice of updates into the given GoStruct. It
// deletes the values at these paths before unmarshalling them. These updates
// can either by JSON-encoded or gNMI-encoded values (scalars).
func replacePaths(schema *yang.Entry, goStruct ygot.GoStruct, prefix *gpb.Path, updates []*gpb.Update, preferShadowPath, ignoreExtraFields, bestEffortUnmarshal bool) error {
	var dopts []DelNodeOpt
	var ce *ComplianceErrors
	if preferShadowPath {
		dopts = append(dopts, &PreferShadowPath{})
	}

	for _, update := range updates {
		var err error
		if update, err = joinPrefixToUpdate(prefix, update); err != nil {
			return err
		}
		if err := DeleteNode(schema, goStruct, update.Path, dopts...); err != nil {
			if bestEffortUnmarshal {
				ce = ce.append(err)
				continue
			}
			return err
		}
		if err := setNode(schema, goStruct, update, preferShadowPath, ignoreExtraFields); err != nil {
			if bestEffortUnmarshal {
				ce = ce.append(err)
				continue
			}
			return err
		}
	}
	if bestEffortUnmarshal && ce != nil {
		return ce
	}
	return nil
}

// updatePaths unmarshals a slice of updates into the given GoStruct. These
// updates can either by JSON-encoded or gNMI-encoded values (scalars).
func updatePaths(schema *yang.Entry, goStruct ygot.GoStruct, prefix *gpb.Path, updates []*gpb.Update, preferShadowPath, ignoreExtraFields, bestEffortUnmarshal bool) error {
	var ce *ComplianceErrors

	for _, update := range updates {
		var err error
		if update, err = joinPrefixToUpdate(prefix, update); err != nil {
			return err
		}
		if err := setNode(schema, goStruct, update, preferShadowPath, ignoreExtraFields); err != nil {
			if bestEffortUnmarshal {
				ce = ce.append(err)
				continue
			}
			return err
		}
	}

	if bestEffortUnmarshal && ce != nil {
		return ce
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

	if err := SetNode(schema, goStruct, update.Path, update.Val, sopts...); err != nil {
		return fmt.Errorf("setNode: %v", err)
	}
	return nil
}
