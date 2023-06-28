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

// Package gotypes is a playground package for demonstrating types that are
// used in the generated Go code.
package gotypes

import (
	"errors"
	"fmt"
)

// RoutingPolicy_PolicyDefinition is the parent for the policy statement, used
// a demonstration of an ordered map (for an `ordered-by user` YANG list).
type RoutingPolicy_PolicyDefinition struct {
	// Statement is an ordered map of policy statements.
	//
	// Note that the design here is to not use a pointer so that the empty
	// value is usable. This is unlike the regular unordered lists where
	// the various helpers need to reside on the parent struct in order to
	// avoid a nil pointer exception on the raw map type.
	Statement *RoutingPolicy_PolicyDefinition_Statement_OrderedMap
}

// AppendNewStatement appends a new policy statement with the given key.
func (s *RoutingPolicy_PolicyDefinition) AppendNewStatement(key string) (*RoutingPolicy_PolicyDefinition_Statement, error) {
	if s.Statement == nil {
		s.Statement = &RoutingPolicy_PolicyDefinition_Statement_OrderedMap{}
	}
	return s.Statement.AppendNew(key)
}

// AppendStatement appends the given statement, extracting the key from the value.
func (s *RoutingPolicy_PolicyDefinition) AppendStatement(statement *RoutingPolicy_PolicyDefinition_Statement) error {
	if s.Statement == nil {
		s.Statement = &RoutingPolicy_PolicyDefinition_Statement_OrderedMap{}
	}
	return s.Statement.Append(statement)
}

// GetStatement gets the statement with the given key.
func (s *RoutingPolicy_PolicyDefinition) GetStatement(key string) *RoutingPolicy_PolicyDefinition_Statement {
	return s.Statement.Get(key)
}

// DeleteStatement deletes the statement with the given key.
func (s *RoutingPolicy_PolicyDefinition) DeleteStatement(key string) bool {
	return s.Statement.Delete(key)
}

// RoutingPolicy_PolicyDefinition_Statement represents an ordered-map element.
type RoutingPolicy_PolicyDefinition_Statement struct {
	// DummyActions are dummy BGP policy actions intended to represent that
	// ordered maps can have values alongside keys.
	DummyActions []string
	Name         *string
}

// RoutingPolicy_PolicyDefinition_Statement_OrderedMap is a candidate ordered-map
// implementation.
type RoutingPolicy_PolicyDefinition_Statement_OrderedMap struct {
	// TODO: Add a mutex here and add race tests after implementing
	// ygot.Equal and evaluating the thread-safety of ygot.
	// mu       sync.RWmutex
	// keys contain the key order of the map.
	keys []string
	// valueMap contains the mapping from the statement key to each of the
	// policy statements.
	valueMap map[string]*RoutingPolicy_PolicyDefinition_Statement
}

// init initializes any uninitialized values.
func (o *RoutingPolicy_PolicyDefinition_Statement_OrderedMap) init() {
	if o == nil {
		return
	}
	if o.valueMap == nil {
		o.valueMap = map[string]*RoutingPolicy_PolicyDefinition_Statement{}
	}
}

// Keys returns a copy of the list's keys.
func (o *RoutingPolicy_PolicyDefinition_Statement_OrderedMap) Keys() []string {
	if o == nil {
		return nil
	}
	return append([]string{}, o.keys...)
}

// Values returns the current set of the list's values in order.
func (o *RoutingPolicy_PolicyDefinition_Statement_OrderedMap) Values() []*RoutingPolicy_PolicyDefinition_Statement {
	if o == nil {
		return nil
	}
	var values []*RoutingPolicy_PolicyDefinition_Statement
	for _, key := range o.keys {
		values = append(values, o.valueMap[key])
	}
	return values
}

// Len returns a size of RoutingPolicy_PolicyDefinition_Statement_OrderedMap
func (o *RoutingPolicy_PolicyDefinition_Statement_OrderedMap) Len() int {
	if o == nil {
		return 0
	}
	return len(o.keys)
}

// Get returns the value corresponding to the key. If the key is not found, nil
// is returned.
func (o *RoutingPolicy_PolicyDefinition_Statement_OrderedMap) Get(key string) *RoutingPolicy_PolicyDefinition_Statement {
	if o == nil {
		return nil
	}
	val, _ := o.valueMap[key]
	return val
}

// Delete deletes an element -- this is O(n) to keep the simple implementation.
func (o *RoutingPolicy_PolicyDefinition_Statement_OrderedMap) Delete(key string) bool {
	if o == nil {
		return false
	}
	if _, ok := o.valueMap[key]; !ok {
		return false
	}
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			delete(o.valueMap, key)
			return true
		}
	}
	return false
}

// Append appends a policy statement, returning an error if the statement
// already exists or if the key is unspecified.
func (o *RoutingPolicy_PolicyDefinition_Statement_OrderedMap) Append(statement *RoutingPolicy_PolicyDefinition_Statement) error {
	if o == nil {
		return errors.New("nil ordered map, cannot append statement")
	}
	if statement == nil {
		return errors.New("nil statement")
	}
	if statement.Name == nil {
		return errors.New("nil key Name")
	}
	key := *statement.Name
	if _, ok := o.valueMap[key]; ok {
		return fmt.Errorf("duplicate key for list Statement %v", key)
	}
	o.keys = append(o.keys, key)
	o.init()
	o.valueMap[key] = statement
	return nil
}

// AppendNew creates and appends a new policy statement, returning the
// newly-initialized statement. It returns an error if the statement already
// exists.
func (o *RoutingPolicy_PolicyDefinition_Statement_OrderedMap) AppendNew(key string) (*RoutingPolicy_PolicyDefinition_Statement, error) {
	if o == nil {
		return nil, errors.New("nil ordered map, cannot append new statement")
	}
	if _, ok := o.valueMap[key]; ok {
		return nil, fmt.Errorf("duplicate key for list Statement %v", key)
	}
	o.keys = append(o.keys, key)
	newElement := &RoutingPolicy_PolicyDefinition_Statement{
		Name: &key,
	}
	o.init()
	o.valueMap[key] = newElement
	return newElement, nil
}
