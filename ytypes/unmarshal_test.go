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

package ytypes

import (
	"fmt"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
)

func TestUnmarshal(t *testing.T) {
	type ParentStruct struct {
		Leaf *string `path:"leaf"`
	}
	validSchema := &yang.Entry{
		Name: "leaf",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Ystring,
		},
	}
	choiceSchema := &yang.Entry{
		Name: "choice",
		Kind: yang.ChoiceEntry,
	}
	tests := []struct {
		desc    string
		schema  *yang.Entry
		value   interface{}
		opts    []UnmarshalOpt
		wantErr string
	}{
		{
			desc:   "success nil field",
			schema: validSchema,
			value:  nil,
		},
		{
			desc:    "error nil schema",
			schema:  nil,
			value:   "{}",
			wantErr: `nil schema for parent type *ytypes.ParentStruct, value {} (string)`,
		},
		{
			desc:    "error choice schema",
			schema:  choiceSchema,
			value:   "{}",
			wantErr: `cannot pass choice schema choice to Unmarshal`,
		},
		{
			desc:   "passing options to Unmarshal",
			schema: validSchema,
			value:  nil,
			opts:   []UnmarshalOpt{&IgnoreExtraFields{}},
		},
		{
			desc:    "passing invalid option to Unmarshal",
			schema:  validSchema,
			value:   nil,
			opts:    []UnmarshalOpt{&BestEffortUnmarshal{}},
			wantErr: `unmarshalGeneric passed unsupported option BestEffortUnmarshal`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var parent ParentStruct

			err := Unmarshal(tt.schema, &parent, tt.value, tt.opts...)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: got error: %v, want error: %v", tt.desc, got, want)
			}

		})
	}
}

const (
	// maxUnmarshalAllocRegression is the maximum permitted fractional
	// increase (5%) above the Unmarshal allocation baseline.
	maxUnmarshalAllocRegression = 0.05
	// unmarshalAllocMeasureRuns is the number of runs averaged by
	// testing.AllocsPerRun after warmup.
	unmarshalAllocMeasureRuns = 5
	// wantUnmarshalKeyedListBaseAllocs is the fixed root-container
	// allocation overhead when unmarshalling ContainerStruct1.
	wantUnmarshalKeyedListBaseAllocs = 55.0
	// wantUnmarshalKeyedListAllocsPerElem is the expected allocations per
	// entry when unmarshalling a keyed list with debug logging disabled.
	wantUnmarshalKeyedListAllocsPerElem = 311.5
)

var unmarshalPerfSizes = []int{10, 100}

func newUnmarshalKeyedListWorkload(tb testing.TB, n int) func() {
	tb.Helper()
	schema := containerWithStringKey()
	addParents(schema)
	items := make([]any, n)
	for i := range n {
		items[i] = map[string]any{
			"key1": fmt.Sprintf("item-%d", i),
			"outer": map[string]any{
				"config": map[string]any{
					"inner": map[string]any{
						"int32-leaf-field": float64(i),
					},
				},
			},
		}
	}
	jsonTree := map[string]any{
		"config": map[string]any{
			"simple-key-list": items,
		},
	}
	return func() {
		root := &ContainerStruct1{}
		if err := Unmarshal(schema, root, jsonTree); err != nil {
			tb.Fatalf("Unmarshal(n=%d) failed: %v", n, err)
		}
		if len(root.StructKeyList) != n {
			tb.Fatalf("Unmarshal(n=%d) got %d entries, want %d", n, len(root.StructKeyList), n)
		}
	}
}

// TestUnmarshalKeyedListPerformance verifies that Unmarshal scales linearly in
// heap allocations across keyed lists of size n -- guarding against O(n^2)
// pretty.Sprint debug formatting regressions.
func TestUnmarshalKeyedListPerformance(t *testing.T) {
	for _, n := range unmarshalPerfSizes {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			fn := newUnmarshalKeyedListWorkload(t, n)
			got := testing.AllocsPerRun(unmarshalAllocMeasureRuns, fn)
			want := wantUnmarshalKeyedListBaseAllocs + wantUnmarshalKeyedListAllocsPerElem*float64(n)
			maxAllowed := want * (1 + maxUnmarshalAllocRegression)
			if got > maxAllowed {
				pct := ((got - want) / want) * 100
				t.Fatalf("UnmarshalKeyedList(n=%d): allocations regressed by +%.2f%% (max +%.0f%%), got: %.1f allocs/op, baseline: %.1f, max allowed: %.1f",
					n, pct, maxUnmarshalAllocRegression*100, got, want, maxAllowed)
			}
		})
	}
}

// BenchmarkUnmarshalKeyedList measures Unmarshal throughput and allocations
// across keyed lists of size n.
func BenchmarkUnmarshalKeyedList(b *testing.B) {
	for _, n := range unmarshalPerfSizes {
		fn := newUnmarshalKeyedListWorkload(b, n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				fn()
			}
		})
	}
}
