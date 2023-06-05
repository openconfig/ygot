// Copyright 2020 Google Inc.
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

package schemaops_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/ygot/integration_tests/schemaops/ctestschema"
	"github.com/openconfig/ygot/ygot"
)

func TestUnorderedList(t *testing.T) {
	d := &ctestschema.Device{}
	d.NewUnorderedList("foo")
	if _, ok := d.UnorderedList["foo"]; !ok {
		t.Errorf("Expected unordered list to be populated")
	}
}

func TestOrderedMap(t *testing.T) {
	var m ctestschema.OrderedList_OrderedMap

	// Action & check: Delete prior to initialization
	if deleted, want := m.Delete("foo"), false; deleted != want {
		t.Errorf("deleted: got %v, want %v", deleted, want)
	}

	// Action: AppendNew
	fooElement, err := m.AppendNew("foo")
	if err != nil {
		t.Fatal(err)
	}
	fooElement.Value = ygot.String("value-foo")
	// Negative test
	if _, err := m.AppendNew("foo"); err == nil {
		t.Fatalf("Expected error due to duplicate, got %v", err)
	}

	// Check
	want := fooElement
	got := m.Get("foo")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}

	// Action: Get & modify
	fooElement = m.Get("foo")
	fooElement.Value = nil
	// Negative test
	if element2 := m.Get("bar"); element2 != nil {
		t.Fatalf("Expected a nil element since key doesn't exist, got %v", element2)
	}

	// Check
	want = fooElement
	got = m.Get("foo")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}

	// Action: Append
	barElement := &ctestschema.OrderedList{
		Key: ygot.String("bar"),
	}
	if err := m.Append(barElement); err != nil {
		t.Fatal(err)
	}
	// Negative test
	if err := m.Append(&ctestschema.OrderedList{
		Key: ygot.String("bar"),
	}); err == nil {
		t.Fatalf("Expected error due to duplicate element, got %v", err)
	}

	// Check
	want = barElement
	got = m.Values()[1]
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
	wantKeys := []string{"foo", "bar"}
	gotKeys := m.Keys()
	if diff := cmp.Diff(wantKeys, gotKeys); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
	if got, want := m.Len(), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	want = barElement

	// Action: Delete
	if deleted, want := m.Delete("foo"), true; deleted != want {
		t.Errorf("deleted: got %v, want %v", deleted, want)
	}
	// Negative test
	if deleted, want := m.Delete("foo"), false; deleted != want {
		t.Errorf("deleted: got %v, want %v", deleted, want)
	}

	// Check
	want = barElement
	got = m.Get("bar")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
}

func TestOrderedMapFromParent(t *testing.T) {
	m := &ctestschema.Device{}

	// Action & check: Delete prior to initialization
	if deleted, want := m.DeleteOrderedList("foo"), false; deleted != want {
		t.Errorf("deleted: got %v, want %v", deleted, want)
	}

	// Action: AppendNew
	fooElement, err := m.AppendNewOrderedList("foo")
	if err != nil {
		t.Fatal(err)
	}
	fooElement.Value = ygot.String("value-foo")
	// Negative test
	if _, err := m.AppendNewOrderedList("foo"); err == nil {
		t.Fatalf("Expected error due to duplicate, got %v", err)
	}

	// Check
	want := fooElement
	got := m.GetOrderedList("foo")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}

	// Action: Get & modify
	fooElement = m.GetOrderedList("foo")
	fooElement.Value = nil
	// Negative test
	if element2 := m.GetOrderedList("bar"); element2 != nil {
		t.Fatalf("Expected a nil element since key doesn't exist, got %v", element2)
	}

	// Check
	want = fooElement
	got = m.GetOrderedList("foo")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}

	// Action: Append
	barElement := &ctestschema.OrderedList{
		Key: ygot.String("bar"),
	}
	if err := m.AppendOrderedList(barElement); err != nil {
		t.Fatal(err)
	}
	// Negative test
	if err := m.AppendOrderedList(&ctestschema.OrderedList{
		Key: ygot.String("bar"),
	}); err == nil {
		t.Fatalf("Expected error due to duplicate element, got %v", err)
	}

	// Check
	want = barElement
	got = m.OrderedList.Values()[1]
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
	wantKeys := []string{"foo", "bar"}
	gotKeys := m.OrderedList.Keys()
	if diff := cmp.Diff(wantKeys, gotKeys); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
	if got, want := m.OrderedList.Len(), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	want = barElement

	// Action: Delete
	if deleted, want := m.DeleteOrderedList("foo"), true; deleted != want {
		t.Errorf("deleted: got %v, want %v", deleted, want)
	}
	// Negative test
	if deleted, want := m.DeleteOrderedList("foo"), false; deleted != want {
		t.Errorf("deleted: got %v, want %v", deleted, want)
	}

	// Check
	want = barElement
	got = m.GetOrderedList("bar")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
}
