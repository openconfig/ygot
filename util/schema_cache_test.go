// Copyright 2026 Google Inc.
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

package util

import (
	"reflect"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/goyang/pkg/yang"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// TestChildSchemaAnnotationGuard verifies that ChildSchema,
// ChildSchemaPreferShadow, and childSchema return (nil, nil) with zero
// allocations for ygotAnnotation fields regardless of whether a path tag or
// choice/case schema subtree is present.
func TestChildSchemaAnnotationGuard(t *testing.T) {
	type annotatedContainer struct {
		// MetadataWithPath is a container-level annotation field carrying a path tag.
		MetadataWithPath []any `path:"@" ygotAnnotation:"true"`
		// MetadataNoPath is an annotation field without a path tag.
		MetadataNoPath []any `ygotAnnotation:"true"`
		// LeafAnnotWithShadow is a leaf annotation field carrying both path and shadow-path tags.
		LeafAnnotWithShadow []any `path:"config/@leaf" shadow-path:"state/@leaf" ygotAnnotation:"true"`
		// ValidLeaf is a regular schema-backed leaf field.
		ValidLeaf *string `path:"config/leaf|leaf" shadow-path:"state/leaf|leaf"`
	}

	schema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"config": {
				Name: "config",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"leaf": {Name: "leaf", Kind: yang.LeafEntry, Type: &yang.YangType{Kind: yang.Ystring}},
				},
			},
			"state": {
				Name: "state",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"leaf": {Name: "leaf", Kind: yang.LeafEntry, Type: &yang.YangType{Kind: yang.Ystring}},
				},
			},
			"choice": {
				Name: "choice",
				Kind: yang.ChoiceEntry,
				Dir: map[string]*yang.Entry{
					"case": {
						Name: "case",
						Kind: yang.CaseEntry,
						Dir: map[string]*yang.Entry{
							"choice-leaf": {Name: "choice-leaf", Kind: yang.LeafEntry, Type: &yang.YangType{Kind: yang.Ystring}},
						},
					},
				},
			},
		},
	}
	addParents(schema)

	st := reflect.TypeOf(annotatedContainer{})
	tests := []struct {
		desc       string
		inField    string
		inShadow   bool
		inDbg      bool
		wantSchema *yang.Entry
		wantErr    bool
	}{{
		desc:    "annotation with path tag",
		inField: "MetadataWithPath",
	}, {
		desc:     "annotation with path tag and preferShadowPath",
		inField:  "MetadataWithPath",
		inShadow: true,
	}, {
		desc:    "annotation without path tag",
		inField: "MetadataNoPath",
	}, {
		desc:     "annotation without path tag and preferShadowPath",
		inField:  "MetadataNoPath",
		inShadow: true,
	}, {
		desc:    "annotation without path tag with debugSchema enabled",
		inField: "MetadataNoPath",
		inDbg:   true,
	}, {
		desc:     "leaf annotation with shadow-path and debugSchema enabled",
		inField:  "LeafAnnotWithShadow",
		inShadow: true,
		inDbg:    true,
	}, {
		desc:       "valid leaf primary path",
		inField:    "ValidLeaf",
		wantSchema: schema.Dir["config"].Dir["leaf"],
	}, {
		desc:       "valid leaf shadow path",
		inField:    "ValidLeaf",
		inShadow:   true,
		wantSchema: schema.Dir["state"].Dir["leaf"],
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			saveDebugVals()
			defer restoreDebugVals()
			debugSchema = tt.inDbg

			ft, ok := st.FieldByName(tt.inField)
			if !ok {
				t.Fatalf("FieldByName(%q) not found", tt.inField)
			}
			var got *yang.Entry
			var err error
			if tt.inShadow {
				got, err = ChildSchemaPreferShadow(schema, ft)
			} else {
				got, err = ChildSchema(schema, ft)
			}
			if (err != nil) != tt.wantErr {
				t.Fatalf("ChildSchema(%s, shadow=%v): did not get expected error, got: %v, wantErr? %v", tt.inField, tt.inShadow, err, tt.wantErr)
			}
			if got != tt.wantSchema {
				t.Fatalf("ChildSchema(%s, shadow=%v): got %v, want %v", tt.inField, tt.inShadow, got, tt.wantSchema)
			}
		})
	}

	annotField, _ := st.FieldByName("MetadataWithPath")
	if allocs := testing.AllocsPerRun(20, func() {
		_, _ = ChildSchema(schema, annotField)
	}); allocs != 0 {
		t.Fatalf("ChildSchema(annotation): got %.1f allocs/op, want 0", allocs)
	}
}

// TestPathSliceMutationSafety verifies that mutating or appending to slices
// returned by SchemaPaths, ShadowSchemaPaths, RelativeSchemaPath, and
// RelativeSchemaPathPreferShadow never corrupts subsequent calls or cached
// StructDesc metadata.
func TestPathSliceMutationSafety(t *testing.T) {
	type sampleStruct struct {
		// Leaf is a compressed leaf field with module-prefixed path and shadow-path tags.
		Leaf *string `path:"mod:config/mod:leaf-a|mod:leaf-a" shadow-path:"mod:state/mod:leaf-a|mod:leaf-a"`
	}

	st := reflect.TypeOf(sampleStruct{})
	ft := st.Field(0)
	sd := DescribeStructType(st)

	wantSchemaPaths := [][]string{{"config", "leaf-a"}, {"leaf-a"}}
	wantShadowPaths := [][]string{{"state", "leaf-a"}, {"leaf-a"}}
	wantRelPath := []string{"mod:config", "mod:leaf-a"}
	wantRelShadowPath := []string{"mod:state", "mod:leaf-a"}

	sp, err := SchemaPaths(ft)
	if err != nil {
		t.Fatalf("SchemaPaths() unexpected error: %v", err)
	}
	sp[0][0] = "mutated"
	sp[0] = append(sp[0], "extra-elem")
	_ = append(sp, []string{"extra-path"})

	ssp := ShadowSchemaPaths(ft)
	ssp[0][0] = "mutated-shadow"
	ssp[0] = append(ssp[0], "extra-shadow-elem")

	rp, err := RelativeSchemaPath(ft)
	if err != nil {
		t.Fatalf("RelativeSchemaPath() unexpected error: %v", err)
	}
	rp[0] = "mutated-rel"
	_ = append(rp, "extra-rel")

	rsp, err := RelativeSchemaPathPreferShadow(ft)
	if err != nil {
		t.Fatalf("RelativeSchemaPathPreferShadow() unexpected error: %v", err)
	}
	rsp[0] = "mutated-rel-shadow"

	gotSP, err := SchemaPaths(ft)
	if err != nil {
		t.Fatalf("second SchemaPaths() unexpected error: %v", err)
	}
	if diff := cmp.Diff(gotSP, wantSchemaPaths); diff != "" {
		t.Fatalf("SchemaPaths() after caller mutation: diff(-got,+want):\n%s", diff)
	}
	if diff := cmp.Diff(sd.Fields[0].SchemaPaths, wantSchemaPaths); diff != "" {
		t.Fatalf("StructDesc.SchemaPaths after caller mutation: diff(-got,+want):\n%s", diff)
	}

	gotSSP := ShadowSchemaPaths(ft)
	if diff := cmp.Diff(gotSSP, wantShadowPaths); diff != "" {
		t.Fatalf("ShadowSchemaPaths() after caller mutation: diff(-got,+want):\n%s", diff)
	}
	if diff := cmp.Diff(sd.Fields[0].ShadowSchemaPaths, wantShadowPaths); diff != "" {
		t.Fatalf("StructDesc.ShadowSchemaPaths after caller mutation: diff(-got,+want):\n%s", diff)
	}

	gotRP, err := RelativeSchemaPath(ft)
	if err != nil {
		t.Fatalf("second RelativeSchemaPath() unexpected error: %v", err)
	}
	if diff := cmp.Diff(gotRP, wantRelPath); diff != "" {
		t.Fatalf("RelativeSchemaPath() after caller mutation: diff(-got,+want):\n%s", diff)
	}

	gotRSP, err := RelativeSchemaPathPreferShadow(ft)
	if err != nil {
		t.Fatalf("second RelativeSchemaPathPreferShadow() unexpected error: %v", err)
	}
	if diff := cmp.Diff(gotRSP, wantRelShadowPath); diff != "" {
		t.Fatalf("RelativeSchemaPathPreferShadow() after caller mutation: diff(-got,+want):\n%s", diff)
	}
}

// TestStructDescMetadataAndConcurrency verifies StructDesc field indexing,
// embedded struct fallback, shallow-copied list schema canonicalisation, warm
// zero-allocation lookups, and concurrent safety under -race.
func TestStructDescMetadataAndConcurrency(t *testing.T) {
	type embeddedBase struct {
		// Promoted is an embedded field accessible via promoted lookup.
		Promoted *string `path:"promoted"`
		// Shadowed is overridden by the outer struct's Shadowed field.
		Shadowed *int32 `path:"base-shadowed"`
	}
	type outerWithEmbed struct {
		embeddedBase
		// Shadowed overrides embeddedBase.Shadowed at the top struct level.
		Shadowed *string `path:"outer-shadowed"`
	}
	type emptyPrefixStruct struct {
		// RootSlash trims trailing empty segments to an empty prefix.
		RootSlash *string `path:"/"`
	}

	outerVal := outerWithEmbed{
		embeddedBase: embeddedBase{Promoted: String("p")},
		Shadowed:     String("s"),
	}
	sd := DescribeStructType(reflect.TypeOf(outerVal))
	if sd != DescribeStructType(reflect.TypeOf(outerVal)) {
		t.Fatalf("DescribeStructType did not return cached *StructDesc pointer")
	}

	sv := reflect.ValueOf(outerVal)
	ft, fv, ok := sd.FieldValueByName(sv, "Promoted")
	if !ok || ft.Name != "Promoted" || fv.Elem().String() != "p" {
		t.Fatalf("FieldValueByName(Promoted) = (%v, %v, %v), want Promoted=\"p\"", ft.Name, fv, ok)
	}
	ft, fv, ok = sd.FieldValueByName(sv, "Shadowed")
	if !ok || ft.Type != reflect.TypeOf((*string)(nil)) || fv.Elem().String() != "s" {
		t.Fatalf("FieldValueByName(Shadowed) did not resolve outer field: got (%v, %v, %v)", ft.Type, fv, ok)
	}
	if _, _, ok := sd.FieldValueByName(sv, "NonExistent"); ok {
		t.Fatalf("FieldValueByName(NonExistent) = true, want false")
	}

	if !DescribeStructType(reflect.TypeOf(emptyPrefixStruct{})).HasEmptyPrefix() {
		t.Fatalf("HasEmptyPrefix() = false for path:\"/\", want true")
	}

	// Verify shallow copies of a list *yang.Entry with ListAttr cleared reuse
	// the canonical parent-directory cache entry rather than allocating new
	// StructSchemaDesc instances per copy.
	schema, benchType := buildWideBenchSchemaAndType(16)
	listSchema := &yang.Entry{
		Name:     "item",
		Kind:     yang.DirectoryEntry,
		ListAttr: yang.NewDefaultListAttr(),
		Dir:      schema.Dir,
	}
	parentSchema := &yang.Entry{
		Name: "items",
		Kind: yang.DirectoryEntry,
		Dir:  map[string]*yang.Entry{"item": listSchema},
	}
	listSchema.Parent = parentSchema

	benchSD := DescribeStructType(benchType)
	shallow1 := *listSchema
	shallow1.ListAttr = nil
	shallow2 := *listSchema
	shallow2.ListAttr = nil
	ssd1 := benchSD.Schema(&shallow1)
	ssd2 := benchSD.Schema(&shallow2)
	if ssd1 != ssd2 {
		t.Fatalf("Schema() on shallow list-element copies returned distinct descriptors %p vs %p", ssd1, ssd2)
	}

	// Warm lookup on canonical schema must allocate zero heap objects.
	_ = benchSD.Schema(schema)
	if allocs := testing.AllocsPerRun(20, func() {
		_ = DescribeStructType(benchType).Schema(schema)
	}); allocs != 0 {
		t.Fatalf("warm DescribeStructType().Schema(): got %.1f allocs/op, want 0", allocs)
	}

	// Exercise concurrent cache reads across goroutines under -race.
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				d := DescribeStructType(benchType)
				s := d.Schema(schema)
				if s.FirstErrIdx != len(d.Fields) {
					t.Errorf("FirstErrIdx = %d, want %d", s.FirstErrIdx, len(d.Fields))
					return
				}
			}
		}()
	}
	wg.Wait()
}

// TestCandidateCursorAndTwoSegmentIndex verifies that StructDesc.Candidates
// merges single-segment and two-segment path and shadow-path indices in
// strictly ascending deduplicated order below limit with zero heap allocations.
func TestCandidateCursorAndTwoSegmentIndex(t *testing.T) {
	type compressedStruct struct {
		// Metadata is a container-level annotation field with a single-segment path.
		Metadata []any `path:"@" ygotAnnotation:"true"`
		// LeafA is a compressed leaf field matching config/leaf-a, state/leaf-a, or leaf-a.
		LeafA *string `path:"mod:config/mod:leaf-a|leaf-a" shadow-path:"state/leaf-a|leaf-a"`
		// LeafB is a second compressed leaf sharing the config and state prefixes.
		LeafB *string `path:"config/leaf-b|leaf-b" shadow-path:"state/leaf-b|leaf-b"`
		// WildcardConfig has a single-segment config prefix that overlaps two-segment config/* paths.
		WildcardConfig *string `path:"config"`
		// DeepList has a three-segment compressed path indexed by its first two segments.
		DeepList *string `path:"outer/items/item" shadow-path:"shadow-outer/items/item"`
	}

	sd := DescribeStructType(reflect.TypeOf(compressedStruct{}))
	collect := func(p *gpb.Path, includeShadow bool, limit int) []int {
		var out []int
		cur := sd.Candidates(p, includeShadow)
		for idx, ok := cur.Next(limit); ok; idx, ok = cur.Next(limit) {
			out = append(out, idx)
		}
		return out
	}

	tests := []struct {
		desc          string
		inPath        *gpb.Path
		includeShadow bool
		inLimit       int
		want          []int
	}{{
		desc:          "empty path returns no candidates",
		inPath:        &gpb.Path{},
		includeShadow: true,
		inLimit:       len(sd.Fields),
	}, {
		desc:          "single-segment config path matches all config-prefixed fields",
		inPath:        &gpb.Path{Elem: []*gpb.PathElem{{Name: "config"}}},
		includeShadow: true,
		inLimit:       len(sd.Fields),
		want:          []int{1, 2, 3},
	}, {
		desc:          "two-segment config/leaf-b isolates LeafB and single-segment WildcardConfig",
		inPath:        &gpb.Path{Elem: []*gpb.PathElem{{Name: "config"}, {Name: "leaf-b"}}},
		includeShadow: true,
		inLimit:       len(sd.Fields),
		want:          []int{2, 3},
	}, {
		desc:          "two-segment module-prefixed path matches LeafA",
		inPath:        &gpb.Path{Elem: []*gpb.PathElem{{Name: "mod:config"}, {Name: "mod:leaf-a"}}},
		includeShadow: false,
		inLimit:       len(sd.Fields),
		want:          []int{1, 3},
	}, {
		desc:          "two-segment shadow path included only when includeShadow is true",
		inPath:        &gpb.Path{Elem: []*gpb.PathElem{{Name: "state"}, {Name: "leaf-b"}}},
		includeShadow: true,
		inLimit:       len(sd.Fields),
		want:          []int{2},
	}, {
		desc:          "two-segment shadow path excluded when includeShadow is false",
		inPath:        &gpb.Path{Elem: []*gpb.PathElem{{Name: "state"}, {Name: "leaf-b"}}},
		includeShadow: false,
		inLimit:       len(sd.Fields),
	}, {
		desc:          "duplicate primary and shadow single-segment leaf-a deduplicated",
		inPath:        &gpb.Path{Elem: []*gpb.PathElem{{Name: "leaf-a"}, {Name: "sub"}}},
		includeShadow: true,
		inLimit:       len(sd.Fields),
		want:          []int{1},
	}, {
		desc:          "limit truncates candidates at or above limit",
		inPath:        &gpb.Path{Elem: []*gpb.PathElem{{Name: "config"}, {Name: "leaf-b"}}},
		includeShadow: true,
		inLimit:       3,
		want:          []int{2},
	}, {
		desc:          "three-segment field matched by two-element partial prefix",
		inPath:        &gpb.Path{Elem: []*gpb.PathElem{{Name: "outer"}, {Name: "items"}}},
		includeShadow: false,
		inLimit:       len(sd.Fields),
		want:          []int{4},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := collect(tt.inPath, tt.includeShadow, tt.inLimit)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Fatalf("Candidates(%v, shadow=%v, limit=%d): diff(-got,+want):\n%s", tt.inPath, tt.includeShadow, tt.inLimit, diff)
			}
		})
	}

	probePath := &gpb.Path{Elem: []*gpb.PathElem{{Name: "config"}, {Name: "leaf-b"}}}
	if allocs := testing.AllocsPerRun(20, func() {
		cur := sd.Candidates(probePath, true)
		for _, ok := cur.Next(len(sd.Fields)); ok; _, ok = cur.Next(len(sd.Fields)) {
		}
	}); allocs != 0 {
		t.Fatalf("Candidates().Next(): got %.1f allocs/op, want 0", allocs)
	}
}
