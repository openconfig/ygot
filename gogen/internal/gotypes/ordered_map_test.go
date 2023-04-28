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

package gotypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/ygot/ygot"
)

func checkState(t *testing.T, got, want *RoutingPolicy_PolicyDefinition_Statement_OrderedMap) {
	t.Helper()
	if len(want.keys) != len(want.valueMap) {
		t.Fatalf("Invalid map: number of keys and values must match: %v", want)
	}

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(RoutingPolicy_PolicyDefinition_Statement_OrderedMap{})); diff != "" {
		t.Errorf("Overall: (-want, +got):\n%s", diff)
	}
	if got, want := len(got.keys), got.Len(); got != want {
		t.Errorf("Len(): Got %v, want %v", got, want)
	}
	if diff := cmp.Diff(got.keys, got.Keys()); diff != "" {
		t.Errorf("Keys(): (-want, +got):\n%s", diff)
	}
	var gotKeysInMap []string
	for _, v := range got.Values() {
		gotKeysInMap = append(gotKeysInMap, *v.Name)
	}
	if diff := cmp.Diff(got.Keys(), gotKeysInMap); diff != "" {
		t.Fatalf("Invalid map: keys and values must match in order: %v", got)
	}
}

func TestOrderedMap(t *testing.T) {
	m := &RoutingPolicy_PolicyDefinition_Statement_OrderedMap{}

	// Action & check: Delete prior to initialization
	if deleted, want := m.Delete("foo"), false; deleted != want {
		t.Errorf("deleted: got %v, want %v", deleted, want)
	}

	// Action: AppendNew
	policy, err := m.AppendNew("foo")
	if err != nil {
		t.Fatal(err)
	}
	// Negative test
	policy.DummyActions = append(policy.DummyActions, "accept all packets from Google")
	if _, err := m.AppendNew("foo"); err == nil {
		t.Fatalf("Expected error due to duplicate, got %v", err)
	}

	// Check
	checkState(t, m, &RoutingPolicy_PolicyDefinition_Statement_OrderedMap{
		keys: []string{"foo"},
		valueMap: map[string]*RoutingPolicy_PolicyDefinition_Statement{
			"foo": {
				Name:         ygot.String("foo"),
				DummyActions: []string{"accept all packets from Google"},
			},
		},
	})

	// Action: Get & modify
	policy = m.Get("foo")
	policy.DummyActions = nil
	// Negative test
	if policy2 := m.Get("bar"); policy2 != nil {
		t.Fatalf("Expected a nil policy since key doesn't exist, got %v", policy2)
	}

	// Check
	checkState(t, m, &RoutingPolicy_PolicyDefinition_Statement_OrderedMap{
		keys: []string{"foo"},
		valueMap: map[string]*RoutingPolicy_PolicyDefinition_Statement{
			"foo": {
				Name: ygot.String("foo"),
			},
		},
	})

	// Action: Append
	if err := m.Append(&RoutingPolicy_PolicyDefinition_Statement{
		Name:         ygot.String("bar"),
		DummyActions: []string{"reject all packets from Google"},
	}); err != nil {
		t.Fatal(err)
	}
	// Negative test
	if err := m.Append(&RoutingPolicy_PolicyDefinition_Statement{
		Name:         ygot.String("bar"),
		DummyActions: []string{"reject all packets from Google"},
	}); err == nil {
		t.Fatalf("Expected error due to duplicate element, got %v", err)
	}

	// Check
	checkState(t, m, &RoutingPolicy_PolicyDefinition_Statement_OrderedMap{
		keys: []string{"foo", "bar"},
		valueMap: map[string]*RoutingPolicy_PolicyDefinition_Statement{
			"foo": {
				Name: ygot.String("foo"),
			},
			"bar": {
				Name:         ygot.String("bar"),
				DummyActions: []string{"reject all packets from Google"},
			},
		},
	})

	// Action: Delete
	if deleted, want := m.Delete("foo"), true; deleted != want {
		t.Errorf("deleted: got %v, want %v", deleted, want)
	}
	// Negative test
	if deleted, want := m.Delete("foo"), false; deleted != want {
		t.Errorf("deleted: got %v, want %v", deleted, want)
	}

	// Check
	checkState(t, m, &RoutingPolicy_PolicyDefinition_Statement_OrderedMap{
		keys: []string{"bar"},
		valueMap: map[string]*RoutingPolicy_PolicyDefinition_Statement{
			"bar": {
				Name:         ygot.String("bar"),
				DummyActions: []string{"reject all packets from Google"},
			},
		},
	})
}

func TestOrderedMapFromParent(t *testing.T) {
	var m RoutingPolicy_PolicyDefinition

	// Action & check: Delete prior to initialization
	if deleted, want := m.DeleteStatement("foo"), false; deleted != want {
		t.Errorf("deleted: got %v, want %v", deleted, want)
	}

	// Action: AppendNew
	policy, err := m.AppendNewStatement("foo")
	if err != nil {
		t.Fatal(err)
	}
	// Negative test
	policy.DummyActions = append(policy.DummyActions, "accept all packets from Google")
	if _, err := m.AppendNewStatement("foo"); err == nil {
		t.Fatalf("Expected error due to duplicate, got %v", err)
	}

	// Check
	checkState(t, m.Statement, &RoutingPolicy_PolicyDefinition_Statement_OrderedMap{
		keys: []string{"foo"},
		valueMap: map[string]*RoutingPolicy_PolicyDefinition_Statement{
			"foo": {
				Name:         ygot.String("foo"),
				DummyActions: []string{"accept all packets from Google"},
			},
		},
	})

	// Action: Get & modify
	policy = m.GetStatement("foo")
	policy.DummyActions = nil
	// Negative test
	if policy2 := m.GetStatement("bar"); policy2 != nil {
		t.Fatalf("Expected a nil policy since key doesn't exist, got %v", policy2)
	}

	// Check
	checkState(t, m.Statement, &RoutingPolicy_PolicyDefinition_Statement_OrderedMap{
		keys: []string{"foo"},
		valueMap: map[string]*RoutingPolicy_PolicyDefinition_Statement{
			"foo": {
				Name: ygot.String("foo"),
			},
		},
	})

	// Action: Append
	if err := m.AppendStatement(&RoutingPolicy_PolicyDefinition_Statement{
		Name:         ygot.String("bar"),
		DummyActions: []string{"reject all packets from Google"},
	}); err != nil {
		t.Fatal(err)
	}
	// Negative test
	if err := m.AppendStatement(&RoutingPolicy_PolicyDefinition_Statement{
		Name:         ygot.String("bar"),
		DummyActions: []string{"reject all packets from Google"},
	}); err == nil {
		t.Fatalf("Expected error due to duplicate element, got %v", err)
	}

	// Check
	checkState(t, m.Statement, &RoutingPolicy_PolicyDefinition_Statement_OrderedMap{
		keys: []string{"foo", "bar"},
		valueMap: map[string]*RoutingPolicy_PolicyDefinition_Statement{
			"foo": {
				Name: ygot.String("foo"),
			},
			"bar": {
				Name:         ygot.String("bar"),
				DummyActions: []string{"reject all packets from Google"},
			},
		},
	})

	// Action: Delete
	if deleted, want := m.DeleteStatement("foo"), true; deleted != want {
		t.Errorf("deleted: got %v, want %v", deleted, want)
	}
	// Negative test
	if deleted, want := m.DeleteStatement("foo"), false; deleted != want {
		t.Errorf("deleted: got %v, want %v", deleted, want)
	}

	// Check
	checkState(t, m.Statement, &RoutingPolicy_PolicyDefinition_Statement_OrderedMap{
		keys: []string{"bar"},
		valueMap: map[string]*RoutingPolicy_PolicyDefinition_Statement{
			"bar": {
				Name:         ygot.String("bar"),
				DummyActions: []string{"reject all packets from Google"},
			},
		},
	})
}
