package ygot

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/goyang/pkg/yang"
)

// addParents adds parent pointers for a schema tree.
func addParents(e *yang.Entry) {
	for _, c := range e.Dir {
		c.Parent = e
		addParents(c)
	}
}

// revertConfigAndAnnotation reverts all entries' Config fields to TSUnset and Annotations to empty.
func revertConfigAndAnnotation(e *yang.Entry) {
	e.Config = yang.TSUnset
	for _, s := range e.Dir {
		s.Config = yang.TSUnset
		s.Annotation = nil
		revertConfigAndAnnotation(s)
	}
}

func TestPruneConfigFalse(t *testing.T) {
	schema := &yang.Entry{
		Name: "empty-branch-test-one",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"string": {
				Name: "string",
				Kind: yang.LeafEntry,
				Type: &yang.YangType{Kind: yang.Ystring},
			},
			"maps": {
				Name: "maps",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"map": {
						Name:     "map",
						Kind:     yang.DirectoryEntry,
						ListAttr: yang.NewDefaultListAttr(),
						Key:      "string",
						Dir: map[string]*yang.Entry{
							"string": {
								Name: "string",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Ystring},
							},
							"enum": {
								Name: "enum",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Yenum},
							},
							"grand-child": {
								Name: "grand-child",
								Kind: yang.DirectoryEntry,
								Dir: map[string]*yang.Entry{
									"string": {
										Name: "string",
										Kind: yang.LeafEntry,
										Type: &yang.YangType{Kind: yang.Ystring},
									},
									"slice": {
										Name:     "slice",
										Kind:     yang.LeafEntry,
										ListAttr: yang.NewDefaultListAttr(),
										Type:     &yang.YangType{Kind: yang.Ystring},
									},
									"great-grand-child": {
										Name: "great-grand-child",
										Kind: yang.DirectoryEntry,
										Dir: map[string]*yang.Entry{
											"string": {
												Name: "string",
												Kind: yang.LeafEntry,
												Type: &yang.YangType{Kind: yang.Ystring},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"child": {
				Name: "child",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"string": {
						Name: "string",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Ystring},
					},
					"enum": {
						Name: "enum",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Yenum},
					},
					"grand-child": {
						Name: "grand-child",
						Kind: yang.DirectoryEntry,
						Dir: map[string]*yang.Entry{
							"string": {
								Name: "string",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Ystring},
							},
							"slice": {
								Name:     "slice",
								Kind:     yang.LeafEntry,
								ListAttr: yang.NewDefaultListAttr(),
								Type:     &yang.YangType{Kind: yang.Ystring},
							},
							"great-grand-child": {
								Name: "great-grand-child",
								Kind: yang.DirectoryEntry,
								Dir: map[string]*yang.Entry{
									"string": {
										Name: "string",
										Kind: yang.LeafEntry,
										Type: &yang.YangType{Kind: yang.Ystring},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	addParents(schema)

	allConfig := func() {
	}

	allState := func() {
		schema.Dir["string"].Config = yang.TSFalse
		schema.Dir["maps"].Dir["map"].Config = yang.TSFalse
		schema.Dir["child"].Config = yang.TSFalse
	}

	tests := []struct {
		desc        string
		setupSchema func()
		inStruct    GoStruct
		want        GoStruct
		wantErr     bool
	}{{
		desc:        "struct with no children",
		setupSchema: allConfig,
		inStruct:    &emptyBranchTestOne{},
		want:        &emptyBranchTestOne{},
	}, {
		desc: "top element is config false: delete all its fields",
		setupSchema: func() {
			schema.Config = yang.TSFalse
		},
		inStruct: &emptyBranchTestOne{
			String: String("hello"),
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42,
				Struct: &emptyBranchTestOneGrandchild{
					String: String("hello"),
					Slice:  []string{"one", "two"},
					Struct: &emptyBranchTestOneGreatGrandchild{
						String: String("hello"),
					},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					String:     String("hello"),
					Enumerated: 42,
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
						Struct: &emptyBranchTestOneGreatGrandchild{
							String: String("hello"),
						},
					},
				},
			},
		},
		want: &emptyBranchTestOne{},
	}, {
		desc:        "completely populated struct that is entirely config true",
		setupSchema: allConfig,
		inStruct: &emptyBranchTestOne{
			String: String("hello"),
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42,
				Struct: &emptyBranchTestOneGrandchild{
					String: String("hello"),
					Slice:  []string{"one", "two"},
					Struct: &emptyBranchTestOneGreatGrandchild{
						String: String("hello"),
					},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					String:     String("hello"),
					Enumerated: 42,
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
						Struct: &emptyBranchTestOneGreatGrandchild{
							String: String("hello"),
						},
					},
				},
			},
		},
		want: &emptyBranchTestOne{
			String: String("hello"),
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42,
				Struct: &emptyBranchTestOneGrandchild{
					String: String("hello"),
					Slice:  []string{"one", "two"},
					Struct: &emptyBranchTestOneGreatGrandchild{
						String: String("hello"),
					},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					String:     String("hello"),
					Enumerated: 42,
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
						Struct: &emptyBranchTestOneGreatGrandchild{
							String: String("hello"),
						},
					},
				},
			},
		},
	}, {
		desc:        "completely populated struct that is entirely config false",
		setupSchema: allState,
		inStruct: &emptyBranchTestOne{
			String: String("hello"),
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42,
				Struct: &emptyBranchTestOneGrandchild{
					String: String("hello"),
					Slice:  []string{"one", "two"},
					Struct: &emptyBranchTestOneGreatGrandchild{
						String: String("hello"),
					},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					String:     String("hello"),
					Enumerated: 42,
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
						Struct: &emptyBranchTestOneGreatGrandchild{
							String: String("hello"),
						},
					},
				},
			},
		},
		want: &emptyBranchTestOne{},
	}, {
		desc: "config enumerated value populated",
		setupSchema: func() {
			schema.Dir["child"].Dir["enum"].Config = yang.TSTrue
		},
		inStruct: &emptyBranchTestOne{
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42,
			},
		},
		want: &emptyBranchTestOne{
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42,
			},
		},
	}, {
		desc: "state enumerated value populated",
		setupSchema: func() {
			schema.Dir["child"].Dir["enum"].Config = yang.TSFalse
		},
		inStruct: &emptyBranchTestOne{
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42,
			},
		},
		want: &emptyBranchTestOne{
			Struct: &emptyBranchTestOneChild{
				String: String("hello"),
			},
		},
	}, {
		desc: "deep string values populated (config)",
		setupSchema: func() {
			schema.Dir["child"].Dir["grand-child"].Dir["great-grand-child"].Dir["string"].Config = yang.TSTrue
			schema.Dir["maps"].Dir["map"].Dir["grand-child"].Dir["great-grand-child"].Dir["string"].Config = yang.TSTrue
		},
		inStruct: &emptyBranchTestOne{
			Struct: &emptyBranchTestOneChild{
				Struct: &emptyBranchTestOneGrandchild{
					Struct: &emptyBranchTestOneGreatGrandchild{
						String: String("hello"),
					},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					String: String("hello"),
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
						Struct: &emptyBranchTestOneGreatGrandchild{
							String: String("hello"),
						},
					},
				},
			},
		},
		want: &emptyBranchTestOne{
			Struct: &emptyBranchTestOneChild{
				Struct: &emptyBranchTestOneGrandchild{
					Struct: &emptyBranchTestOneGreatGrandchild{
						String: String("hello"),
					},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					String: String("hello"),
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
						Struct: &emptyBranchTestOneGreatGrandchild{
							String: String("hello"),
						},
					},
				},
			},
		},
	}, {
		desc: "deep string values populated (state)",
		setupSchema: func() {
			schema.Dir["child"].Dir["grand-child"].Dir["great-grand-child"].Dir["string"].Config = yang.TSFalse
			schema.Dir["maps"].Dir["map"].Dir["grand-child"].Dir["great-grand-child"].Dir["string"].Config = yang.TSFalse
		},
		inStruct: &emptyBranchTestOne{
			Struct: &emptyBranchTestOneChild{
				Struct: &emptyBranchTestOneGrandchild{
					Struct: &emptyBranchTestOneGreatGrandchild{
						String: String("hello"),
					},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					String: String("hello"),
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Struct: &emptyBranchTestOneGreatGrandchild{
							String: String("hello"),
						},
					},
				},
			},
		},
		want: &emptyBranchTestOne{
			Struct: &emptyBranchTestOneChild{
				Struct: &emptyBranchTestOneGrandchild{
					Struct: &emptyBranchTestOneGreatGrandchild{},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					String: String("hello"),
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Struct: &emptyBranchTestOneGreatGrandchild{},
					},
				},
			},
		},
	}, {
		desc: "config slice value populated",
		setupSchema: func() {
			schema.Dir["child"].Dir["grand-child"].Dir["slice"].Config = yang.TSTrue
			schema.Dir["maps"].Dir["map"].Dir["grand-child"].Dir["slice"].Config = yang.TSTrue
		},
		inStruct: &emptyBranchTestOne{
			Struct: &emptyBranchTestOneChild{
				Struct: &emptyBranchTestOneGrandchild{
					String: String("hello"),
					Slice:  []string{"one", "two"},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
					},
				},
			},
		},
		want: &emptyBranchTestOne{
			Struct: &emptyBranchTestOneChild{
				Struct: &emptyBranchTestOneGrandchild{
					String: String("hello"),
					Slice:  []string{"one", "two"},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
					},
				},
			},
		},
	}, {
		desc: "state slice value populated",
		setupSchema: func() {
			schema.Dir["child"].Dir["grand-child"].Dir["slice"].Config = yang.TSFalse
			schema.Dir["maps"].Dir["map"].Dir["grand-child"].Dir["slice"].Config = yang.TSFalse
		},
		inStruct: &emptyBranchTestOne{
			Struct: &emptyBranchTestOneChild{
				Struct: &emptyBranchTestOneGrandchild{
					String: String("hello"),
					Slice:  []string{"one", "two"},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
					},
				},
			},
		},
		want: &emptyBranchTestOne{
			Struct: &emptyBranchTestOneChild{
				Struct: &emptyBranchTestOneGrandchild{
					String: String("hello"),
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
					},
				},
			},
		},
	}, {
		desc: "middle-level container and list are config false",
		setupSchema: func() {
			schema.Dir["child"].Dir["grand-child"].Config = yang.TSFalse
			schema.Dir["maps"].Dir["map"].Config = yang.TSFalse
		},
		inStruct: &emptyBranchTestOne{
			String: String("hello"),
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42,
				Struct: &emptyBranchTestOneGrandchild{ // config false
					String: String("hello"),
					Slice:  []string{"one", "two"},
					Struct: &emptyBranchTestOneGreatGrandchild{
						String: String("hello"),
					},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{ // config false
				"foo": {
					String:     String("hello"),
					Enumerated: 42,
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
						Struct: &emptyBranchTestOneGreatGrandchild{
							String: String("hello"),
						},
					},
				},
			},
		},
		want: &emptyBranchTestOne{
			String: String("hello"),
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42,
			},
		},
	}, {
		desc: "list container is config false",
		setupSchema: func() {
			schema.Dir["maps"].Config = yang.TSFalse
		},
		inStruct: &emptyBranchTestOne{
			String: String("hello"),
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42,
				Struct: &emptyBranchTestOneGrandchild{
					String: String("hello"),
					Slice:  []string{"one", "two"},
					Struct: &emptyBranchTestOneGreatGrandchild{
						String: String("hello"),
					},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{ // config false
				"foo": {
					String:     String("hello"),
					Enumerated: 42,
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
						Struct: &emptyBranchTestOneGreatGrandchild{
							String: String("hello"),
						},
					},
				},
			},
		},
		want: &emptyBranchTestOne{
			String: String("hello"),
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42,
				Struct: &emptyBranchTestOneGrandchild{
					String: String("hello"),
					Slice:  []string{"one", "two"},
					Struct: &emptyBranchTestOneGreatGrandchild{
						String: String("hello"),
					},
				},
			},
		},
	}, {
		desc: "random values are config false",
		setupSchema: func() {
			schema.Dir["string"].Config = yang.TSFalse
			schema.Dir["child"].Dir["enum"].Config = yang.TSFalse
			schema.Dir["child"].Dir["grand-child"].Dir["string"].Config = yang.TSFalse
			schema.Dir["child"].Dir["grand-child"].Dir["slice"].Config = yang.TSFalse
			schema.Dir["maps"].Dir["map"].Dir["string"].Config = yang.TSFalse
			schema.Dir["maps"].Dir["map"].Dir["grand-child"].Dir["great-grand-child"].Config = yang.TSFalse
		},
		inStruct: &emptyBranchTestOne{
			String: String("hello"), // config false
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42, // config false
				Struct: &emptyBranchTestOneGrandchild{
					String: String("hello"),        // config false
					Slice:  []string{"one", "two"}, // config false
					Struct: &emptyBranchTestOneGreatGrandchild{
						String: String("hello"),
					},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					String:     String("hello"), // config false
					Enumerated: 42,
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
						Struct: &emptyBranchTestOneGreatGrandchild{ // config false
							String: String("hello"),
						},
					},
				},
			},
		},
		want: &emptyBranchTestOne{
			Struct: &emptyBranchTestOneChild{
				String: String("hello"),
				Struct: &emptyBranchTestOneGrandchild{
					Struct: &emptyBranchTestOneGreatGrandchild{
						String: String("hello"),
					},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					Enumerated: 42,
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
					},
				},
			},
		},
	}, {
		desc: "random values are config false, but some are compressed",
		setupSchema: func() {
			schema.Dir["string"].Config = yang.TSFalse
			schema.Dir["string"].Annotation = map[string]interface{}{"foo-bar-baz": struct{}{}}
			schema.Dir["child"].Dir["enum"].Config = yang.TSFalse
			schema.Dir["child"].Dir["enum"].Annotation = map[string]interface{}{GoCompressedLeafAnnotation: struct{}{}}
			schema.Dir["child"].Dir["grand-child"].Dir["string"].Config = yang.TSFalse
			schema.Dir["child"].Dir["grand-child"].Dir["slice"].Config = yang.TSFalse
			schema.Dir["child"].Dir["grand-child"].Dir["slice"].Annotation = map[string]interface{}{GoCompressedLeafAnnotation: struct{}{}}
			schema.Dir["maps"].Dir["map"].Dir["string"].Config = yang.TSFalse
			schema.Dir["maps"].Dir["map"].Dir["string"].Annotation = map[string]interface{}{GoCompressedLeafAnnotation: struct{}{}}
			schema.Dir["maps"].Dir["map"].Dir["grand-child"].Dir["great-grand-child"].Config = yang.TSFalse
		},
		inStruct: &emptyBranchTestOne{
			String: String("hello"), // config false
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42, // config false but compressed
				Struct: &emptyBranchTestOneGrandchild{
					String: String("hello"),        // config false
					Slice:  []string{"one", "two"}, // config false but compressed
					Struct: &emptyBranchTestOneGreatGrandchild{
						String: String("hello"),
					},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					String:     String("hello"), // config false but compressed
					Enumerated: 42,
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
						Struct: &emptyBranchTestOneGreatGrandchild{ // config false
							String: String("hello"),
						},
					},
				},
			},
		},
		want: &emptyBranchTestOne{
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42, // config false but compressed
				Struct: &emptyBranchTestOneGrandchild{
					Slice: []string{"one", "two"}, // config false but compressed
					Struct: &emptyBranchTestOneGreatGrandchild{
						String: String("hello"),
					},
				},
			},
			StructMap: map[string]*emptyBranchTestOneChild{
				"foo": {
					String:     String("hello"), // config false but compressed
					Enumerated: 42,
					Struct: &emptyBranchTestOneGrandchild{
						String: String("hello"),
						Slice:  []string{"one", "two"},
					},
				},
			},
		},
	}, {
		desc: "bad input with config true inside config false: the inner config values are ignored",
		setupSchema: func() {
			schema.Dir["child"].Config = yang.TSTrue
			schema.Dir["child"].Dir["grand-child"].Config = yang.TSFalse
			schema.Dir["child"].Dir["grand-child"].Dir["string"].Config = yang.TSTrue
			schema.Dir["child"].Dir["grand-child"].Dir["great-grand-child"].Config = yang.TSTrue
		},
		inStruct: &emptyBranchTestOne{
			String: String("hello"),
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42,
				Struct: &emptyBranchTestOneGrandchild{ // config false
					String: String("hello"), // config true
					Slice:  []string{"one", "two"},
					Struct: &emptyBranchTestOneGreatGrandchild{ // config true
						String: String("hello"),
					},
				},
			},
		},
		want: &emptyBranchTestOne{
			String: String("hello"),
			Struct: &emptyBranchTestOneChild{
				String:     String("hello"),
				Enumerated: 42,
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			revertConfigAndAnnotation(schema)
			tt.setupSchema()
			err := PruneConfigFalse(schema, tt.inStruct)
			if (err != nil) != tt.wantErr {
				t.Errorf("Got error %v, wantErr: %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if diff := cmp.Diff(tt.inStruct, tt.want); diff != "" {
				t.Errorf("diff(-got, +want):\n%s", diff)
			}
		})
	}
}
