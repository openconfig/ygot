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

package protogen

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygen"
	"github.com/openconfig/ygot/ygot"
	"google.golang.org/protobuf/testing/protocmp"
)

func protoMsgEq(a, b *protoMsg) bool {
	if a.Name != b.Name {
		return false
	}

	if a.YANGPath != b.YANGPath {
		return false
	}

	if a.Imports != nil && b.Imports != nil && !cmp.Equal(a.Imports, b.Imports) {
		return false
	}

	// Avoid flakes by comparing the fields in an unordered data structure.
	fieldMap := func(s []*protoMsgField) map[string]*protoMsgField {
		e := map[string]*protoMsgField{}
		for _, m := range s {
			e[m.Name] = m
		}
		return e
	}

	return cmp.Equal(fieldMap(a.Fields), fieldMap(b.Fields))
}

func TestGenProto3Msg(t *testing.T) {
	modules := yang.NewModules()
	modules.Modules["mod"] = &yang.Module{
		Name: "mod",
		Namespace: &yang.Value{
			Name: "u:mod",
		},
	}

	tests := []struct {
		name                  string
		inMsg                 *ygen.ParsedDirectory
		inIR                  *ygen.IR
		inCompressPaths       bool
		inBasePackage         string
		inEnumPackage         string
		inBaseImportPath      string
		inAnnotateSchemaPaths bool
		inParentPackage       string
		inChildMsgs           []*generatedProto3Message
		wantMsgs              map[string]*protoMsg
		wantErr               bool
	}{{
		name: "simple message with only scalar fields",
		inMsg: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"field-one": {
					Name: "field_one",
					Type: ygen.LeafNode,
					LangType: &ygen.MappedType{
						NativeType: "ywrapper.StringValue",
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "field-one",
						Path: "/field-one",
					},
				},
				"field-two": {
					Name: "field_two",
					Type: ygen.LeafNode,
					LangType: &ygen.MappedType{
						NativeType: "ywrapper.IntValue",
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "field-two",
						Path: "/field-two",
					},
				},
			},
			Path: "/root/message-name",
		},
		inBasePackage: "base",
		inEnumPackage: "enums",
		wantMsgs: map[string]*protoMsg{
			"MessageName": {
				Name:     "MessageName",
				YANGPath: "/root/message-name",
				Fields: []*protoMsgField{{
					Tag:  410095931,
					Name: "field_one",
					Type: "ywrapper.StringValue",
				}, {
					Tag:  25944937,
					Name: "field_two",
					Type: "ywrapper.IntValue",
				}},
			},
		},
	}, {
		name: "simple message with child messages, ensure no difference in logic",
		inMsg: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"field-one": {
					Name: "field_one",
					Type: ygen.LeafNode,
					LangType: &ygen.MappedType{
						NativeType: "ywrapper.StringValue",
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "field-one",
						Path: "/field-one",
					},
				},
			},
			Path: "/root/message-name",
		},
		inBasePackage: "base",
		inEnumPackage: "enums",
		inChildMsgs: []*generatedProto3Message{{
			PackageName:     "none",
			MessageCode:     "test-code",
			RequiredImports: []string{"should-be-ignored"},
		}},
		wantMsgs: map[string]*protoMsg{
			"MessageName": {
				Name:     "MessageName",
				YANGPath: "/root/message-name",
				Fields: []*protoMsgField{{
					Tag:  410095931,
					Name: "field_one",
					Type: "ywrapper.StringValue",
				}},
			},
		},
	}, {
		name: "simple message with union leaf and leaf-list",
		inMsg: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"field-one": {
					Name: "field_one",
					Type: ygen.LeafNode,
					LangType: &ygen.MappedType{
						UnionTypes: map[string]ygen.MappedUnionSubtype{
							"string": {
								Index: 0,
							},
							"sint64": {
								Index: 1,
							},
						},
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "field-one",
						Path: "/field-one",
					},
				},
				"field-two": {
					Name: "field_two",
					Type: ygen.LeafListNode,
					LangType: &ygen.MappedType{
						UnionTypes: map[string]ygen.MappedUnionSubtype{
							"sint64": {
								Index: 0,
							},
							"base.enums.BaseDerivedEnum": {
								Index:                 1,
								EnumeratedYANGTypeKey: "/root/derived-enum",
							},
						},
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "field-two",
						Path: "/parent/field-two",
					},
				},
			},
			Path: "/root/message-name",
		},
		inIR: &ygen.IR{
			Enums: map[string]*ygen.EnumeratedYANGType{
				"/root/derived-enum": {
					Name:     "BaseDerivedEnum",
					Kind:     ygen.DerivedEnumerationType,
					TypeName: "derived-enum",
					ValToYANGDetails: []ygot.EnumDefinition{
						{
							Name:           "NORMAL",
							DefiningModule: "base",
							Value:          0,
						},
					},
				},
			},
		},
		inBasePackage: "base",
		inEnumPackage: "enums",
		wantMsgs: map[string]*protoMsg{
			"MessageName": {
				Name:     "MessageName",
				YANGPath: "/root/message-name",
				Imports:  []string{"base/enums/enums.proto"},
				Fields: []*protoMsgField{{
					Tag:     410095931,
					Name:    "field_one",
					Type:    "",
					IsOneOf: true,
					OneOfFields: []*protoMsgField{{
						Tag:  225170402,
						Name: "field_one_sint64",
						Type: "sint64",
					}, {
						Tag:  299030977,
						Name: "field_one_string",
						Type: "string",
					}},
				}, {
					Tag:        332121324,
					Name:       "field_two",
					Type:       "FieldTwoUnion",
					IsRepeated: true,
				}},
			},
			"FieldTwoUnion": {
				Name:     "FieldTwoUnion",
				YANGPath: "/parent/field-two union field field-two",
				Fields: []*protoMsgField{{
					Tag:  350335944,
					Name: "field_two_basederivedenum",
					Type: "base.enums.BaseDerivedEnum",
				}, {
					Tag:  226381575,
					Name: "field_two_sint64",
					Type: "sint64",
				}},
			},
		},
	}, {
		name: "union leaf with annotate schema paths enabled",
		inMsg: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"field-one": {
					Name:        "field_one",
					Type:        ygen.LeafNode,
					MappedPaths: [][]string{{"", "field-one"}},
					LangType: &ygen.MappedType{
						UnionTypes: map[string]ygen.MappedUnionSubtype{
							"string": {
								Index: 0,
							},
							"sint64": {
								Index: 1,
							},
						},
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "field-one",
						Path: "/field-one",
					},
				},
			},
			Path: "/root/message-name",
		},
		inIR: &ygen.IR{
			Enums: map[string]*ygen.EnumeratedYANGType{},
		},
		inBasePackage:         "base",
		inEnumPackage:         "enums",
		inAnnotateSchemaPaths: true,
		wantMsgs: map[string]*protoMsg{
			"MessageName": {
				Name:     "MessageName",
				YANGPath: "/root/message-name",
				Fields: []*protoMsgField{{
					Tag:     410095931,
					Name:    "field_one",
					Type:    "",
					IsOneOf: true,
					Options: []*protoOption{{
						Name:  "(yext.schemapath)",
						Value: `"/field-one"`,
					}},
					OneOfFields: []*protoMsgField{{
						Tag:  225170402,
						Name: "field_one_sint64",
						Type: "sint64",
						Options: []*protoOption{{
							Name:  "(yext.schemapath)",
							Value: `"/field-one"`,
						}},
					}, {
						Tag:  299030977,
						Name: "field_one_string",
						Type: "string",
						Options: []*protoOption{{
							Name:  "(yext.schemapath)",
							Value: `"/field-one"`,
						}},
					}},
				}},
			},
		},
	}, {
		name: "simple message with leaf-list and a message child, compression on",
		inMsg: &ygen.ParsedDirectory{
			Name: "AMessage",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"leaf-list": {
					Name: "leaf_list",
					Type: ygen.LeafListNode,
					LangType: &ygen.MappedType{
						NativeType: "ywrapper.StringValue",
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "leaf-list",
						Path: "/leaf-list",
					},
				},
				"container-child": {
					Name: "container_child",
					Type: ygen.ContainerNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "container-child",
						Path: "/root/a-message/container-child",
					},
				},
			},
			Path: "/root/a-message",
		},
		inIR: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/root/a-message/container-child": {
					Name:        "ContainerChild",
					Type:        ygen.Container,
					Path:        "/root/a-message/container-child",
					PackageName: "a_message",
				},
			},
		},
		inCompressPaths: true,
		inBasePackage:   "base",
		inEnumPackage:   "enums",
		wantMsgs: map[string]*protoMsg{
			"AMessage": {
				Name:     "AMessage",
				YANGPath: "/root/a-message",
				Fields: []*protoMsgField{{
					Tag:        299656613,
					Name:       "leaf_list",
					Type:       "ywrapper.StringValue",
					IsRepeated: true,
				}, {
					Tag:  17594927,
					Name: "container_child",
					Type: "a_message.ContainerChild",
				}},
				Imports: []string{"base/a_message/a_message.proto"},
			},
		},
	}, {
		name: "simple message with leaf-list and a message child, compression off",
		inMsg: &ygen.ParsedDirectory{
			Name: "AMessage",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"leaf-list": {
					Name: "leaf_list",
					Type: ygen.LeafListNode,
					LangType: &ygen.MappedType{
						NativeType: "ywrapper.StringValue",
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "leaf-list",
						Path: "/leaf-list",
					},
				},
				"container-child": {
					Name: "container_child",
					Type: ygen.ContainerNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "container-child",
						Path: "/root/a-message/container-child",
					},
				},
			},
			Path: "/root/a-message",
		},
		inIR: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/root/a-message/container-child": {
					Name:        "ContainerChild",
					Type:        ygen.Container,
					Path:        "/root/a-message/container-child",
					PackageName: "root.a_message",
				},
			},
		},
		inBasePackage: "base",
		inEnumPackage: "enums",
		wantMsgs: map[string]*protoMsg{
			"AMessage": {
				Name:     "AMessage",
				YANGPath: "/root/a-message",
				Fields: []*protoMsgField{{
					Tag:        299656613,
					Name:       "leaf_list",
					Type:       "ywrapper.StringValue",
					IsRepeated: true,
				}, {
					Tag:  17594927,
					Name: "container_child",
					Type: "root.a_message.ContainerChild",
				}},
				Imports: []string{"base/root/a_message/a_message.proto"},
			},
		},
	}, {
		name: "message with list",
		inMsg: &ygen.ParsedDirectory{
			Name: "AMessageWithAList",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"list": {
					Name: "list",
					Type: ygen.ListNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "list",
						Path: "/a-message-with-a-list/list",
					},
				},
			},
			Path: "/a-message-with-a-list/list",
		},
		inIR: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/a-message-with-a-list/list": {
					Name:        "ygen.List",
					Type:        ygen.List,
					Path:        "/a-message-with-a-list/list",
					PackageName: "a_message_with_a_list",
					Fields: map[string]*ygen.NodeDetails{
						"key": {
							Name: "key",
							Type: ygen.LeafNode,
							YANGDetails: ygen.YANGNodeDetails{
								Name: "key",
								Path: "/key",
							},
						},
					},
					ListKeys: map[string]*ygen.ListKey{
						"key": {
							Name: "key",
							LangType: &ygen.MappedType{
								NativeType: "string",
							},
						},
					},
				},
			},
		},
		inBasePackage: "base",
		inEnumPackage: "enums",
		wantMsgs: map[string]*protoMsg{
			"AMessageWithAList": {
				Name:     "AMessageWithAList",
				YANGPath: "/a-message-with-a-list/list",
				Fields: []*protoMsgField{{
					Name:       "list",
					Type:       "ygen.ListKey",
					Tag:        200573382,
					IsRepeated: true,
				}},
			},
			"ygen.ListKey": {
				Name:     "ygen.ListKey",
				YANGPath: "/a-message-with-a-list/list",
				Fields: []*protoMsgField{{
					Tag:        1,
					Name:       "key",
					Type:       "string",
					IsRepeated: false,
				}, {
					Tag:  2,
					Name: "list",
					Type: "a_message_with_a_list.ygen.List",
				}},
				Imports: []string{"base/a_message_with_a_list/a_message_with_a_list.proto"},
			},
		},
	}, {
		name: "message with list, where the key has the same name as list",
		inMsg: &ygen.ParsedDirectory{
			Name: "AMessageWithAList",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"list": {
					Name: "list",
					Type: ygen.ListNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "list",
						Path: "/a-message-with-a-list/list",
					},
				},
			},
			Path: "/a-message-with-a-list/list",
		},
		inIR: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/a-message-with-a-list/list": {
					Name:        "ygen.List",
					Type:        ygen.List,
					Path:        "/a-message-with-a-list/list",
					PackageName: "a_message_with_a_list",
					Fields: map[string]*ygen.NodeDetails{
						"list": {
							Name: "list",
							Type: ygen.LeafNode,
							YANGDetails: ygen.YANGNodeDetails{
								Name: "list",
								Path: "/list",
							},
						},
					},
					ListKeys: map[string]*ygen.ListKey{
						"list": {
							Name: "list",
							LangType: &ygen.MappedType{
								NativeType: "string",
							},
						},
					},
				},
			},
		},
		inBasePackage: "base",
		inEnumPackage: "enums",
		wantMsgs: map[string]*protoMsg{
			"AMessageWithAList": {
				Name:     "AMessageWithAList",
				YANGPath: "/a-message-with-a-list/list",
				Fields: []*protoMsgField{{
					Name:       "list",
					Type:       "ygen.ListKey",
					Tag:        200573382,
					IsRepeated: true,
				}},
			},
			"ygen.ListKey": {
				Name:     "ygen.ListKey",
				YANGPath: "/a-message-with-a-list/list",
				Fields: []*protoMsgField{{
					Tag:        1,
					Name:       "list_key",
					Type:       "string",
					IsRepeated: false,
				}, {
					Tag:  2,
					Name: "list",
					Type: "a_message_with_a_list.ygen.List",
				}},
				Imports: []string{"base/a_message_with_a_list/a_message_with_a_list.proto"},
			},
		},
	}, {
		name: "message with missing directory",
		inMsg: &ygen.ParsedDirectory{
			Name: "Foo",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"bar": {
					Name: "bar",
					Type: ygen.ContainerNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "bar",
						Path: "/bar",
					},
				},
			},
			Path: "/foo",
		},
		inIR: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{},
		},
		wantErr: true,
	}, {
		name: "message with any anydata field",
		inMsg: &ygen.ParsedDirectory{
			Name: "MessageWithAnydata",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"any-data": {
					Name:     "any_data",
					Type:     ygen.AnyDataNode,
					LangType: nil,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "any-data",
						Path: "/any-data",
					},
				},
				"leaf": {
					Name: "leaf",
					Type: ygen.LeafNode,
					LangType: &ygen.MappedType{
						NativeType: "ywrapper.StringValue",
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "leaf",
						Path: "/leaf",
					},
				},
			},
			Path: "/message-with-anydata",
		},
		inBasePackage: "base",
		inEnumPackage: "enums",
		wantMsgs: map[string]*protoMsg{
			"MessageWithAnydata": {
				Name:     "MessageWithAnydata",
				YANGPath: "/message-with-anydata",
				Imports:  []string{"google/protobuf/any.proto"},
				Fields: []*protoMsgField{{
					Tag:  453452743,
					Name: "any_data",
					Type: "google.protobuf.Any",
				}, {
					Tag:  463279904,
					Name: "leaf",
					Type: "ywrapper.StringValue",
				}},
			},
		},
	}, {
		name: "message with annotate schema paths enabled",
		inMsg: &ygen.ParsedDirectory{
			Name: "MessageWithAnnotations",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"leaf": {
					Name: "leaf",
					Type: ygen.LeafNode,
					LangType: &ygen.MappedType{
						NativeType: "ywrapper.StringValue",
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "leaf",
						Path: "/one/two/leaf",
					},
					MappedPaths: [][]string{{
						"",
						"two",
						"leaf",
					}},
				},
			},
			Path: "/one/two",
		},
		inBasePackage:         "base",
		inEnumPackage:         "enums",
		inAnnotateSchemaPaths: true,
		wantMsgs: map[string]*protoMsg{
			"MessageWithAnnotations": {
				Name:     "MessageWithAnnotations",
				YANGPath: "/one/two",
				Fields: []*protoMsgField{{
					Name: "leaf",
					Tag:  60047678,
					Type: "ywrapper.StringValue",
					Options: []*protoOption{{
						Name:  "(yext.schemapath)",
						Value: `"/two/leaf"`,
					}},
				}},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMsgs, errs := genProto3Msg(tt.inMsg, tt.inIR, &protoMsgConfig{
				compressPaths:       tt.inCompressPaths,
				basePackageName:     tt.inBasePackage,
				enumPackageName:     tt.inEnumPackage,
				baseImportPath:      tt.inBaseImportPath,
				annotateSchemaPaths: tt.inAnnotateSchemaPaths,
			}, tt.inParentPackage, tt.inChildMsgs)

			if (errs != nil) != tt.wantErr {
				t.Errorf("%s: genProtoMsg(%#v, %#v, %v, %s, %s): did not get expected error status, got: %v, wanted err: %v", tt.name, tt.inMsg, tt.inIR, tt.inCompressPaths, tt.inBasePackage, tt.inEnumPackage, errs, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			notSeen := map[string]bool{}
			for _, w := range tt.wantMsgs {
				notSeen[w.Name] = true
			}

			for _, got := range gotMsgs {
				want, ok := tt.wantMsgs[got.Name]
				if !ok {
					t.Errorf("%s: genProtoMsg(%#v, %#v): got unexpected message, got: %v, want: %v", tt.name, tt.inMsg, tt.inIR, got.Name, tt.wantMsgs)
					continue
				}
				delete(notSeen, got.Name)

				if !protoMsgEq(got, want) {
					diff := cmp.Diff(got, want, cmpopts.EquateEmpty(), protocmp.Transform())
					t.Errorf("%s: genProtoMsg(%#v, %#v): did not get expected protobuf message definition, diff(-got,+want):\n%s", tt.name, tt.inMsg, tt.inIR, diff)
				}
			}

			if len(notSeen) != 0 {
				t.Errorf("%s: genProtoMsg(%#v, %#v); did not test all returned messages, got remaining messages: %v, want: none", tt.name, tt.inMsg, tt.inIR, notSeen)
			}
		})
	}
}

func TestSafeProtoName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{{
		name: "contains hyphen",
		in:   "with-hyphen",
		want: "with_hyphen",
	}, {
		name: "contains period",
		in:   "with.period",
		want: "with_period",
	}, {
		name: "contains plus",
		in:   "with+plus",
		want: "with_plus",
	}, {
		name: "contains slash",
		in:   "with/slash",
		want: "with_slash",
	}, {
		name: "contains space",
		in:   "with space",
		want: "with_space",
	}, {
		name: "contains numbers",
		in:   "with1_numbers234",
		want: "with1_numbers234",
	}, {
		name: "unchanged",
		in:   "unchanged",
		want: "unchanged",
	}}

	for _, tt := range tests {
		if got := safeProtoIdentifierName(tt.in); got != tt.want {
			t.Errorf("%s: safeProtoFieldName(%s): did not get expected name, got: %v, want: %v", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestWriteProtoMsg(t *testing.T) {
	tests := []struct {
		name string
		// inMsg should be used if msg is the same between compressed and uncompressed.
		inMsg           *ygen.ParsedDirectory
		inMsgCompress   *ygen.ParsedDirectory
		inMsgUncompress *ygen.ParsedDirectory
		inIR            *ygen.IR
		// inIR should be used if ygen.IR is the same between compressed and uncompressed.
		inIRCompress      *ygen.IR
		inIRUncompress    *ygen.IR
		inBasePackageName string
		inEnumPackageName string
		inBaseImportPath  string
		inNestedMessages  bool
		wantCompress      *generatedProto3Message
		wantUncompress    *generatedProto3Message
		wantCompressErr   bool
		wantUncompressErr bool
	}{{
		name: "simple message with scalar fields",
		inMsgCompress: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"field-one": {
					Name: "field_one",
					Type: ygen.LeafNode,
					LangType: &ygen.MappedType{
						NativeType: "ywrapper.StringValue",
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "field-one",
						Path: "/field-one",
					},
				},
			},
			PackageName: "container",
			Path:        "/module/container/message-name",
		},
		inMsgUncompress: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"field-one": {
					Name: "field_one",
					Type: ygen.LeafNode,
					LangType: &ygen.MappedType{
						NativeType: "ywrapper.StringValue",
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "field-one",
						Path: "/field-one",
					},
				},
			},
			PackageName: "module.container",
			Path:        "/module/container/message-name",
		},
		inBasePackageName: "base",
		inEnumPackageName: "enums",
		wantCompress: &generatedProto3Message{
			PackageName: "container",
			MessageCode: `
// MessageName represents the /module/container/message-name YANG schema element.
message MessageName {
  ywrapper.StringValue field_one = 410095931;
}`,
		},
		wantUncompress: &generatedProto3Message{
			PackageName: "module.container",
			MessageCode: `
// MessageName represents the /module/container/message-name YANG schema element.
message MessageName {
  ywrapper.StringValue field_one = 410095931;
}`,
		},
	}, {
		name: "simple message with other messages embedded",
		inMsgCompress: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"child": {
					Name: "child",
					Type: ygen.ContainerNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "child",
						Path: "/module/message-name/child",
					},
				},
			},
			Path:        "/module/message-name",
			PackageName: "",
		},
		inMsgUncompress: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"child": {
					Name: "child",
					Type: ygen.ContainerNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "child",
						Path: "/module/message-name/child",
					},
				},
			},
			Path:        "/module/message-name",
			PackageName: "module",
		},
		inIRCompress: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/module/message-name/child": {
					Name:        "Child",
					Type:        ygen.Container,
					Path:        "/module/message-name/child",
					PackageName: "message_name",
				},
			},
		},
		inIRUncompress: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/module/message-name/child": {
					Name:        "Child",
					Type:        ygen.Container,
					Path:        "/module/message-name/child",
					PackageName: "module.message_name",
				},
			},
		},
		inBasePackageName: "base",
		inEnumPackageName: "enums",
		wantCompress: &generatedProto3Message{
			PackageName: "",
			MessageCode: `
// MessageName represents the /module/message-name YANG schema element.
message MessageName {
  message_name.Child child = 399980855;
}`,
			RequiredImports: []string{"base/message_name/message_name.proto"},
		},
		wantUncompress: &generatedProto3Message{
			PackageName: "module",
			MessageCode: `
// MessageName represents the /module/message-name YANG schema element.
message MessageName {
  message_name.Child child = 399980855;
}`,
			RequiredImports: []string{"base/module/message_name/message_name.proto"},
		},
	}, {
		name: "simple message with other messages embedded - with nested messages",
		inMsgCompress: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"child": {
					Name: "child",
					Type: ygen.ContainerNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "child",
						Path: "/module/message-name/child",
					},
				},
			},
			Path:        "/module/message-name",
			PackageName: "",
		},
		inIRCompress: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/module/message-name/child": {
					Name: "Child",
					Type: ygen.Container,
					Fields: map[string]*ygen.NodeDetails{
						"leaf": {
							Name: "leaf",
							Type: ygen.LeafNode,
							LangType: &ygen.MappedType{
								NativeType: "ywrapper.StringValue",
							},
							YANGDetails: ygen.YANGNodeDetails{
								Name: "leaf",
								Path: "/leaf",
							},
						},
					},
					Path:        "/module/message-name/child",
					PackageName: "",
				},
			},
		},
		inMsgUncompress: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"child": {
					Name: "child",
					Type: ygen.ContainerNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "child",
						Path: "/module/message-name/child",
					},
				},
			},
			Path:        "/module/message-name",
			PackageName: "module",
		},
		inIRUncompress: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/module/message-name/child": {
					Name: "Child",
					Type: ygen.Container,
					Fields: map[string]*ygen.NodeDetails{
						"leaf": {
							Name: "leaf",
							Type: ygen.LeafNode,
							LangType: &ygen.MappedType{
								NativeType: "ywrapper.StringValue",
							},
							YANGDetails: ygen.YANGNodeDetails{
								Name: "leaf",
								Path: "/leaf",
							},
						},
					},
					Path:        "/module/message-name/child",
					PackageName: "module",
				},
			},
		},
		inBasePackageName: "base",
		inEnumPackageName: "enums",
		inNestedMessages:  true,
		wantCompress: &generatedProto3Message{
			PackageName: "",
			MessageCode: `
message MessageName {
  message Child {
    ywrapper.StringValue leaf = 463279904;
  }
  Child child = 399980855;
}`,
		},
		wantUncompress: &generatedProto3Message{
			PackageName: "module",
			MessageCode: `
message MessageName {
  message Child {
    ywrapper.StringValue leaf = 463279904;
  }
  Child child = 399980855;
}`,
		},
	}, {
		name: "simple message with an enumeration leaf",
		inMsgCompress: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"enum": {
					Name: "enum",
					Type: ygen.LeafNode,
					LangType: &ygen.MappedType{
						NativeType:            "Enum",
						IsEnumeratedValue:     true,
						EnumeratedYANGTypeKey: "/module/message-name/enum",
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "enum",
						Path: "/module/message-name/enum",
					},
				},
			},
			Path:        "/module/message-name",
			PackageName: "",
		},
		inMsgUncompress: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"enum": {
					Name: "enum",
					Type: ygen.LeafNode,
					LangType: &ygen.MappedType{
						NativeType:            "Enum",
						IsEnumeratedValue:     true,
						EnumeratedYANGTypeKey: "/module/message-name/enum",
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "enum",
						Path: "/module/message-name/enum",
					},
				},
			},
			Path:        "/module/message-name",
			PackageName: "module",
		},
		inIR: &ygen.IR{
			Enums: map[string]*ygen.EnumeratedYANGType{
				"/module/message-name/enum": {
					Name:     "ModuleMessageNameEnum",
					Kind:     ygen.SimpleEnumerationType,
					TypeName: "enumeration",
					ValToYANGDetails: []ygot.EnumDefinition{
						{
							Name:  "ONE",
							Value: 1,
						},
						{
							Name:  "FORTYTWO",
							Value: 42,
						},
					},
				},
			},
		},
		inBasePackageName: "base",
		inEnumPackageName: "enums",
		wantCompress: &generatedProto3Message{
			PackageName: "",
			MessageCode: `
// MessageName represents the /module/message-name YANG schema element.
message MessageName {
  enum Enum {
    ENUM_UNSET = 0;
    ENUM_ONE = 2;
    ENUM_FORTYTWO = 43;
  }
  Enum enum = 278979784;
}`,
		},
		wantUncompress: &generatedProto3Message{
			PackageName: "module",
			MessageCode: `
// MessageName represents the /module/message-name YANG schema element.
message MessageName {
  enum Enum {
    ENUM_UNSET = 0;
    ENUM_ONE = 2;
    ENUM_FORTYTWO = 43;
  }
  Enum enum = 278979784;
}`,
		},
	}, {
		name: "simple message with a list",
		inMsgUncompress: &ygen.ParsedDirectory{
			Name: "AMessage",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"list": {
					Name: "list",
					Type: ygen.ListNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "list",
						Path: "/module/a-message/surrounding-container/list",
					},
				},
			},
			Path:        "/module/a-message",
			PackageName: "module",
		},
		inIRUncompress: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/module/a-message/surrounding-container/list": {
					Name: "ygen.List",
					Type: ygen.List,
					Fields: map[string]*ygen.NodeDetails{
						"keyfield": {
							Name: "keyfield",
							Type: ygen.LeafNode,
							YANGDetails: ygen.YANGNodeDetails{
								Name: "keyfield",
								Path: "/keyfield",
							},
						},
					},
					ListKeys: map[string]*ygen.ListKey{
						"keyfield": {
							Name: "keyfield",
							LangType: &ygen.MappedType{
								NativeType: "string",
							},
						},
					},
					Path:        "/module/a-message/surrounding-container/list",
					PackageName: "module.a_message.surrounding_container",
				},
			},
		},
		inMsgCompress: &ygen.ParsedDirectory{
			Name: "AMessage",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"list": {
					Name: "list",
					Type: ygen.ListNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "list",
						Path: "/module/a-message/surrounding-container/list",
					},
				},
			},
			Path:        "/module/a-message",
			PackageName: "",
		},
		inIRCompress: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/module/a-message/surrounding-container/list": {
					Name: "ygen.List",
					Type: ygen.List,
					Fields: map[string]*ygen.NodeDetails{
						"keyfield": {
							Name: "keyfield",
							Type: ygen.LeafNode,
							YANGDetails: ygen.YANGNodeDetails{
								Name: "keyfield",
								Path: "/keyfield",
							},
						},
					},
					ListKeys: map[string]*ygen.ListKey{
						"keyfield": {
							Name: "keyfield",
							LangType: &ygen.MappedType{
								NativeType: "string",
							},
						},
					},
					Path:        "/module/a-message/surrounding-container/list",
					PackageName: "a_message",
				},
			},
		},
		inBasePackageName: "base",
		inEnumPackageName: "enums",
		wantCompress: &generatedProto3Message{
			PackageName: "",
			MessageCode: `
// ygen.ListKey represents the /module/a-message/surrounding-container/list YANG schema element.
message ygen.ListKey {
  string keyfield = 1;
  a_message.ygen.List list = 2;
}

// AMessage represents the /module/a-message YANG schema element.
message AMessage {
  repeated ygen.ListKey list = 486198550;
}`,
			RequiredImports: []string{"base/a_message/a_message.proto"},
		},
		wantUncompress: &generatedProto3Message{
			PackageName: "module",
			MessageCode: `
// ygen.ListKey represents the /module/a-message/surrounding-container/list YANG schema element.
message ygen.ListKey {
  string keyfield = 1;
  a_message.surrounding_container.ygen.List list = 2;
}

// AMessage represents the /module/a-message YANG schema element.
message AMessage {
  repeated ygen.ListKey list = 486198550;
}`,
			RequiredImports: []string{"base/module/a_message/surrounding_container/surrounding_container.proto"},
		},
	}, {
		name: "simple message with a list - nested messages",
		inMsgUncompress: &ygen.ParsedDirectory{
			Name: "AMessage",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"list": {
					Name: "list",
					Type: ygen.ListNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "list",
						Path: "/module/a-message/surrounding-container/list",
					},
				},
			},
			Path:        "/module/a-message",
			PackageName: "module",
		},
		inIRUncompress: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/module/a-message/surrounding-container/list": {
					Name: "ygen.List",
					Type: ygen.List,
					Fields: map[string]*ygen.NodeDetails{
						"keyfield": {
							Name: "keyfield",
							Type: ygen.LeafNode,
							LangType: &ygen.MappedType{
								NativeType: "ywrapper.StringValue",
							},
							YANGDetails: ygen.YANGNodeDetails{
								Name: "keyfield",
								Path: "/keyfield",
							},
						},
					},
					ListKeys: map[string]*ygen.ListKey{
						"keyfield": {
							Name: "keyfield",
							LangType: &ygen.MappedType{
								NativeType: "string",
							},
						},
					},
					Path:        "/module/a-message/surrounding-container/list",
					PackageName: "module",
				},
			},
		},
		inMsgCompress: &ygen.ParsedDirectory{
			Name: "AMessage",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"list": {
					Name: "list",
					Type: ygen.ListNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "list",
						Path: "/module/a-message/surrounding-container/list",
					},
				},
			},
			Path:        "/module/a-message",
			PackageName: "",
		},
		inIRCompress: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/module/a-message/surrounding-container/list": {
					Name: "ygen.List",
					Type: ygen.List,
					Fields: map[string]*ygen.NodeDetails{
						"keyfield": {
							Name: "keyfield",
							Type: ygen.LeafNode,
							LangType: &ygen.MappedType{
								NativeType: "ywrapper.StringValue",
							},
							YANGDetails: ygen.YANGNodeDetails{
								Name: "keyfield",
								Path: "/keyfield",
							},
						},
					},
					ListKeys: map[string]*ygen.ListKey{
						"keyfield": {
							Name: "keyfield",
							LangType: &ygen.MappedType{
								NativeType: "string",
							},
						},
					},
					Path:        "/module/a-message/surrounding-container/list",
					PackageName: "a_message",
				},
			},
		},
		inBasePackageName: "base",
		inEnumPackageName: "enums",
		inNestedMessages:  true,
		wantCompress: &generatedProto3Message{
			PackageName: "",
			MessageCode: `
message AMessage {
  message ygen.List {
  }
  message ygen.ListKey {
    string keyfield = 1;
    ygen.List list = 2;
  }
  repeated ygen.ListKey list = 486198550;
}`,
		},
		wantUncompress: &generatedProto3Message{
			PackageName: "module",
			MessageCode: `
message AMessage {
  message ygen.List {
  }
  message ygen.ListKey {
    string keyfield = 1;
    ygen.List list = 2;
  }
  repeated ygen.ListKey list = 486198550;
}`,
		},
	}, {
		name: "simple message with unkeyed list - nested messages",
		inMsgUncompress: &ygen.ParsedDirectory{
			Name: "AMessage",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"list": {
					Name: "list",
					Type: ygen.ListNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "list",
						Path: "/module/a-message/surrounding-container/list",
					},
				},
			},
			Path:        "/module/a-message",
			PackageName: "module",
		},
		inIRUncompress: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/module/a-message/surrounding-container/list": {
					Name: "ygen.List",
					Type: ygen.List,
					Fields: map[string]*ygen.NodeDetails{
						"keyfield": {
							Name: "keyfield",
							Type: ygen.LeafNode,
							LangType: &ygen.MappedType{
								NativeType: "ywrapper.StringValue",
							},
							YANGDetails: ygen.YANGNodeDetails{
								Name: "keyfield",
								Path: "/keyfield",
							},
						},
					},
					Path:        "/module/a-message/surrounding-container/list",
					PackageName: "module.a_message.surrounding_container",
				},
			},
		},
		inMsgCompress: &ygen.ParsedDirectory{
			Name: "AMessage",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"list": {
					Name: "list",
					Type: ygen.ListNode,
					YANGDetails: ygen.YANGNodeDetails{
						Name: "list",
						Path: "/module/a-message/surrounding-container/list",
					},
				},
			},
			Path:        "/module/a-message",
			PackageName: "",
		},
		inIRCompress: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/module/a-message/surrounding-container/list": {
					Name: "ygen.List",
					Type: ygen.List,
					Fields: map[string]*ygen.NodeDetails{
						"keyfield": {
							Name: "keyfield",
							Type: ygen.LeafNode,
							LangType: &ygen.MappedType{
								NativeType: "ywrapper.StringValue",
							},
							YANGDetails: ygen.YANGNodeDetails{
								Name: "keyfield",
								Path: "/keyfield",
							},
						},
					},
					Path:        "/module/a-message/surrounding-container/list",
					PackageName: "a_message",
				},
			},
		},
		inBasePackageName: "base",
		inEnumPackageName: "enums",
		inNestedMessages:  true,
		wantCompress: &generatedProto3Message{
			PackageName: "",
			MessageCode: `
message AMessage {
  message ygen.List {
    ywrapper.StringValue keyfield = 411968747;
  }
  repeated ygen.List list = 486198550;
}`,
		},
		wantUncompress: &generatedProto3Message{
			PackageName: "module",
			MessageCode: `
message AMessage {
  message ygen.List {
    ywrapper.StringValue keyfield = 411968747;
  }
  repeated ygen.List list = 486198550;
}`,
		},
	}, {
		name: "message skipped due to path length",
		inMsg: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Path: "one/two",
		},
		inBasePackageName: "base",
		inEnumPackageName: "enums",
		inNestedMessages:  true,
		wantCompress:      nil,
		wantUncompress:    nil,
	}, {
		name: "simple message with an identityref leaf",
		inMsgUncompress: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"identityref": {
					Name: "identityref",
					Type: ygen.LeafNode,
					LangType: &ygen.MappedType{
						NativeType:            "base.enums.TestModuleFooIdentity",
						IsEnumeratedValue:     true,
						EnumeratedYANGTypeKey: "/module/foo-identity",
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "identityref",
						Path: "/module/message-name/identityref",
					},
				},
			},
			PackageName: "module",
			Path:        "/module-name/message-name",
		},
		inMsgCompress: &ygen.ParsedDirectory{
			Name: "MessageName",
			Type: ygen.Container,
			Fields: map[string]*ygen.NodeDetails{
				"identityref": {
					Name: "identityref",
					Type: ygen.LeafNode,
					LangType: &ygen.MappedType{
						NativeType:            "base.enums.TestModuleFooIdentity",
						IsEnumeratedValue:     true,
						EnumeratedYANGTypeKey: "/module/foo-identity",
					},
					YANGDetails: ygen.YANGNodeDetails{
						Name: "identityref",
						Path: "/module/message-name/identityref",
					},
				},
			},
			PackageName: "",
			Path:        "/module-name/message-name",
		},
		inIR: &ygen.IR{
			Enums: map[string]*ygen.EnumeratedYANGType{
				"/module/foo-identity": {
					Name:     "TestModuleFooIdentity",
					Kind:     ygen.IdentityType,
					TypeName: "identityref",
					ValToYANGDetails: []ygot.EnumDefinition{
						{
							Name:           "ONE",
							DefiningModule: "test-module",
						},
						{
							Name:           "TWO",
							DefiningModule: "test-module",
						},
					},
				},
			},
		},
		inBasePackageName: "base",
		inEnumPackageName: "enums",
		wantCompress: &generatedProto3Message{
			PackageName: "",
			MessageCode: `
// MessageName represents the /module-name/message-name YANG schema element.
message MessageName {
  base.enums.TestModuleFooIdentity identityref = 518954308;
}`,
			RequiredImports: []string{"base/enums/enums.proto"},
		},
		wantUncompress: &generatedProto3Message{
			PackageName: "module",
			MessageCode: `
// MessageName represents the /module-name/message-name YANG schema element.
message MessageName {
  base.enums.TestModuleFooIdentity identityref = 518954308;
}`,
			RequiredImports: []string{"base/enums/enums.proto"},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, compress := range []bool{false, true} {
				inMsg := tt.inMsg
				inIR := tt.inIR
				want := tt.wantUncompress
				wantErr := tt.wantUncompressErr
				if compress {
					if inMsg == nil {
						inMsg = tt.inMsgCompress
					}
					if inIR == nil {
						inIR = tt.inIRCompress
					}
					want = tt.wantCompress
					wantErr = tt.wantCompressErr
				} else {
					if inMsg == nil {
						inMsg = tt.inMsgUncompress
					}
					if inIR == nil {
						inIR = tt.inIRUncompress
					}
				}

				got, errs := writeProto3Msg(inMsg, inIR, &protoMsgConfig{
					compressPaths:   compress,
					basePackageName: tt.inBasePackageName,
					enumPackageName: tt.inEnumPackageName,
					baseImportPath:  tt.inBaseImportPath,
					nestedMessages:  tt.inNestedMessages,
				})

				if (errs != nil) != wantErr {
					t.Errorf("%s: writeProto3Msg(%v, %v, %v): did not get expected error return status, got: %v, wanted error: %v", tt.name, inMsg, inIR, compress, errs, wantErr)
				}

				if errs != nil || got == nil {
					continue
				}

				if got.PackageName != want.PackageName {
					t.Errorf("%s: writeProto3Msg(%v, %v, %v): did not get expected package name, got: %v, want: %v", tt.name, inMsg, inIR, compress, got.PackageName, want.PackageName)
				}

				if diff := cmp.Diff(want.RequiredImports, got.RequiredImports); diff != "" {
					t.Errorf("%s: writeProto3Msg(%v, %v, %v): did not get expected set of imports, (-want, +got,):\n%s", tt.name, inMsg, inIR, compress, diff)
				}

				if diff := cmp.Diff(got.MessageCode, want.MessageCode); diff != "" {
					if diffl, err := testutil.GenerateUnifiedDiff(want.MessageCode, got.MessageCode); err == nil {
						diff = diffl
					}
					t.Errorf("%s: writeProto3Msg(%v, %v, %v): did not get expected message returned, diff(-want, +got):\n%s", tt.name, inMsg, inIR, compress, diff)
				}
			}
		})
	}
}

func TestGenListKeyProto(t *testing.T) {
	tests := []struct {
		name          string
		inListPackage string
		inListName    string
		inArgs        *protoDefinitionArgs
		wantMsg       *protoMsg
		wantErr       bool
	}{{
		name:          "simple list key proto",
		inListPackage: "pkg",
		inListName:    "list",
		inArgs: &protoDefinitionArgs{
			field: &ygen.NodeDetails{
				Name: "list",
				Type: ygen.ListNode,
				YANGDetails: ygen.YANGNodeDetails{
					Path: "/list",
				},
			},
			directory: &ygen.ParsedDirectory{
				Name: "ygen.List",
				Type: ygen.List,
				Fields: map[string]*ygen.NodeDetails{
					"key": {
						Name: "key",
						LangType: &ygen.MappedType{
							NativeType: "string",
						},
						YANGDetails: ygen.YANGNodeDetails{
							Path: "/list/key",
						},
					},
				},
				ListKeys: map[string]*ygen.ListKey{
					"key": {
						Name: "key",
						LangType: &ygen.MappedType{
							NativeType: "string",
						},
					},
				},
			},
			cfg: &protoMsgConfig{
				compressPaths:   false,
				basePackageName: "base",
				baseImportPath:  "base/path",
			},
		},
		wantMsg: &protoMsg{
			Name:     "listKey",
			YANGPath: "/list",
			Fields: []*protoMsgField{{
				Tag:  1,
				Name: "key",
				Type: "string",
			}, {
				Tag:  2,
				Name: "list",
				Type: "pkg.list",
			}},
			Imports: []string{"base/path/base/pkg/pkg.proto"},
		},
	}, {
		name:          "list with union key - string and int",
		inListPackage: "pkg",
		inListName:    "list",
		inArgs: &protoDefinitionArgs{
			field: &ygen.NodeDetails{
				Name: "list",
				Type: ygen.ListNode,
				YANGDetails: ygen.YANGNodeDetails{
					Path: "/list",
				},
			},
			directory: &ygen.ParsedDirectory{
				Name: "ygen.List",
				Type: ygen.List,
				Fields: map[string]*ygen.NodeDetails{
					"key": {
						Name: "key",
						LangType: &ygen.MappedType{
							UnionTypes: map[string]ygen.MappedUnionSubtype{
								"string": {
									Index: 0,
								},
								"sint64": {
									Index: 1,
								},
							},
						},
						YANGDetails: ygen.YANGNodeDetails{
							Path: "/key",
						},
					},
				},
				ListKeys: map[string]*ygen.ListKey{
					"key": {
						Name: "key",
						LangType: &ygen.MappedType{
							UnionTypes: map[string]ygen.MappedUnionSubtype{
								"string": {
									Index: 0,
								},
								"sint64": {
									Index: 1,
								},
							},
						},
					},
				},
			},
			ir: &ygen.IR{},
			cfg: &protoMsgConfig{
				compressPaths:   false,
				basePackageName: "base",
				baseImportPath:  "base/path",
			},
		},
		wantMsg: &protoMsg{
			Name:     "listKey",
			YANGPath: "/list",
			Fields: []*protoMsgField{{
				Tag:     1,
				Name:    "key",
				IsOneOf: true,
				OneOfFields: []*protoMsgField{{
					Tag:  232819104,
					Name: "key_sint64",
					Type: "sint64",
				}, {
					Tag:  470483267,
					Name: "key_string",
					Type: "string",
				}},
			}, {
				Tag:  2,
				Name: "list",
				Type: "pkg.list",
			}},
			Imports: []string{"base/path/base/pkg/pkg.proto"},
		},
	}}

	for _, tt := range tests {
		got, err := genListKeyProto(tt.inListPackage, tt.inListName, tt.inArgs)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: genListKeyProto(%s, %s, %#v): got unexpected error returned, got: %v, want err: %v", tt.name, tt.inListPackage, tt.inListName, tt.inArgs, err, tt.wantErr)
		}

		if diff := cmp.Diff(got, tt.wantMsg, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("%s: genListKeyProto(%s, %s, %#v): did not get expected return message, diff(-got,+want):\n%s", tt.name, tt.inListPackage, tt.inListName, tt.inArgs, diff)
		}
	}
}

func TestWriteProtoEnums(t *testing.T) {
	// Create mock enumerations within goyang since we cannot create them in-line.
	testEnums := map[string][]string{
		"enumOne": {"SPEED_2.5G", "SPEED_40G"},
		"enumTwo": {"VALUE_1", "VALUE_2"},
	}
	testYANGEnums := map[string]*yang.EnumType{}

	for name, values := range testEnums {
		enum := yang.NewEnumType()
		for i, v := range values {
			enum.Set(v, int64(i))
		}
		testYANGEnums[name] = enum
	}

	tests := []struct {
		name                string
		inEnums             map[string]*ygen.EnumeratedYANGType
		inAnnotateEnumNames bool
		wantEnums           []string
		wantErr             bool
	}{{
		name: "skipped enumeration type",
		inEnums: map[string]*ygen.EnumeratedYANGType{
			"/field-name|enum": {
				Name:     "SomeEnumType",
				Kind:     ygen.SimpleEnumerationType,
				TypeName: "enumeration",
				ValToYANGDetails: []ygot.EnumDefinition{{
					Name:  "SPEED_2.5G",
					Value: 0,
				}, {
					Name:  "SPEED_40G",
					Value: 1,
				}},
			},
		},
		wantEnums: []string{},
	}, {
		name: "enum for identityref",
		inEnums: map[string]*ygen.EnumeratedYANGType{
			"/field-name|enum": {
				Name:             "EnumeratedValue",
				Kind:             ygen.IdentityType,
				IdentityBaseName: "IdentityValue",
				ValToYANGDetails: []ygot.EnumDefinition{{
					Name:           "VALUE_A",
					DefiningModule: "mod",
				}, {
					Name:           "VALUE_B",
					DefiningModule: "mod2",
				}},
			},
		},
		wantEnums: []string{
			`
// EnumeratedValue represents an enumerated type generated for the YANG identity IdentityValue.
enum EnumeratedValue {
  ENUMERATEDVALUE_UNSET = 0;
  ENUMERATEDVALUE_VALUE_A = 321526273;
  ENUMERATEDVALUE_VALUE_B = 321526274;
}
`,
		},
	}, {
		name: "enum for typedef enumeration",
		inEnums: map[string]*ygen.EnumeratedYANGType{
			"e": {
				Name:     "EnumName",
				Kind:     ygen.DerivedEnumerationType,
				TypeName: "typedef",
				ValToYANGDetails: []ygot.EnumDefinition{{
					Name:  "SPEED_2.5G",
					Value: 0,
				}, {
					Name:  "SPEED_40G",
					Value: 1,
				}},
			},
			"f": {
				Name:     "SecondEnum",
				Kind:     ygen.DerivedEnumerationType,
				TypeName: "derived",
				ValToYANGDetails: []ygot.EnumDefinition{{
					Name:  "VALUE_1",
					Value: 0,
				}, {
					Name:  "VALUE_2",
					Value: 1,
				}},
			},
		},
		inAnnotateEnumNames: true,
		wantEnums: []string{
			`
// EnumName represents an enumerated type generated for the YANG enumerated type typedef.
enum EnumName {
  ENUMNAME_UNSET = 0;
  ENUMNAME_SPEED_2_5G = 1 [(yext.yang_name) = "SPEED_2.5G"];
  ENUMNAME_SPEED_40G = 2 [(yext.yang_name) = "SPEED_40G"];
}
`, `
// SecondEnum represents an enumerated type generated for the YANG enumerated type derived.
enum SecondEnum {
  SECONDENUM_UNSET = 0;
  SECONDENUM_VALUE_1 = 1 [(yext.yang_name) = "VALUE_1"];
  SECONDENUM_VALUE_2 = 2 [(yext.yang_name) = "VALUE_2"];
}
`,
		},
	}}

	for _, tt := range tests {
		got, err := writeProtoEnums(tt.inEnums, tt.inAnnotateEnumNames)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: writeProtoEnums(%v): did not get expected error, got: %v", tt.name, tt.inEnums, err)
		}

		if err != nil {
			continue
		}

		// Sort the returned output to avoid test flakes.
		sort.Strings(got)
		if diff := cmp.Diff(got, tt.wantEnums, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("%s: writeProtoEnums(%v): did not get expected output, diff(-got,+want):\n%s", tt.name, tt.inEnums, diff)
		}
	}
}

func TestUnionFieldToOneOf(t *testing.T) {
	tests := []struct {
		name    string
		inName  string
		inField *ygen.NodeDetails
		// inPath is populated with field.YANGDetails.Path if not set.
		inPath                string
		inMappedType          *ygen.MappedType
		inEnums               map[string]*ygen.EnumeratedYANGType
		inAnnotateEnumNames   bool
		inAnnotateSchemaPaths bool
		wantFields            []*protoMsgField
		wantEnums             map[string]*protoMsgEnum
		wantRepeatedMsg       *protoMsg
		wantErr               bool
	}{{
		name:   "simple string union",
		inName: "FieldName",
		inField: &ygen.NodeDetails{
			Name: "field-name",
			Type: ygen.LeafNode,
			YANGDetails: ygen.YANGNodeDetails{
				Path: "/field-name",
			},
			LangType: &ygen.MappedType{
				UnionTypes: map[string]ygen.MappedUnionSubtype{
					"string": {
						Index: 0,
					},
					"sint64": {
						Index: 1,
					},
				},
			},
		},
		inMappedType: &ygen.MappedType{
			UnionTypes: map[string]ygen.MappedUnionSubtype{
				"string": {
					Index: 0,
				},
				"sint64": {
					Index: 1,
				},
			},
		},
		wantFields: []*protoMsgField{{
			Tag:  171677331,
			Name: "FieldName_sint64",
			Type: "sint64",
		}, {
			Tag:  173535000,
			Name: "FieldName_string",
			Type: "string",
		}},
		wantEnums: map[string]*protoMsgEnum{},
	}, {
		name:   "simple string union with a non-empty path argument",
		inName: "FieldName",
		inField: &ygen.NodeDetails{
			Name: "field-name",
			Type: ygen.LeafNode,
			LangType: &ygen.MappedType{
				UnionTypes: map[string]ygen.MappedUnionSubtype{
					"string": {
						Index: 0,
					},
					"sint64": {
						Index: 1,
					},
				},
			},
		},
		inMappedType: &ygen.MappedType{
			UnionTypes: map[string]ygen.MappedUnionSubtype{
				"string": {
					Index: 0,
				},
				"sint64": {
					Index: 1,
				},
			},
		},
		inPath: "a/b/c/d",
		wantFields: []*protoMsgField{{
			Tag:  352411621,
			Name: "FieldName_sint64",
			Type: "sint64",
		}, {
			Tag:  156680110,
			Name: "FieldName_string",
			Type: "string",
		}},
		wantEnums: map[string]*protoMsgEnum{},
	}, {
		name:   "decimal64 union",
		inName: "FieldName",
		inField: &ygen.NodeDetails{
			Name: "field-name",
			Type: ygen.LeafNode,
			YANGDetails: ygen.YANGNodeDetails{
				Path: "/field-name",
			},
			LangType: &ygen.MappedType{
				UnionTypes: map[string]ygen.MappedUnionSubtype{
					"string": {
						Index: 0,
					},
					"ywrapper.Decimal64": {
						Index: 1,
					},
				},
			},
		},
		inMappedType: &ygen.MappedType{
			UnionTypes: map[string]ygen.MappedUnionSubtype{
				"string": {
					Index: 0,
				},
				"ywrapper.Decimal64": {
					Index: 1,
				},
			},
		},
		wantFields: []*protoMsgField{{
			Tag:  173535000,
			Name: "FieldName_string",
			Type: "string",
		}, {
			Tag:  328616554,
			Name: "FieldName_decimal64",
			Type: "ywrapper.Decimal64",
		}},
		wantEnums: map[string]*protoMsgEnum{},
	}, {
		name:   "union with an enumeration",
		inName: "FieldName",
		inField: &ygen.NodeDetails{
			Name: "field-name",
			Type: ygen.LeafNode,
			YANGDetails: ygen.YANGNodeDetails{
				Path: "/field-name",
			},
			LangType: &ygen.MappedType{
				UnionTypes: map[string]ygen.MappedUnionSubtype{
					"SomeEnumType": {
						Index:                 0,
						EnumeratedYANGTypeKey: "/field-name|enum",
					},
					"string": {
						Index: 1,
					},
				},
			},
		},
		inMappedType: &ygen.MappedType{
			UnionTypes: map[string]ygen.MappedUnionSubtype{
				"SomeEnumType": {
					Index:                 0,
					EnumeratedYANGTypeKey: "/field-name|enum",
				},
				"string": {
					Index: 1,
				},
			},
		},
		inEnums: map[string]*ygen.EnumeratedYANGType{
			"/field-name|enum": {
				Name:     "SomeEnumType",
				Kind:     ygen.SimpleEnumerationType,
				TypeName: "enumeration",
				ValToYANGDetails: []ygot.EnumDefinition{{
					Name:  "SPEED_2.5G",
					Value: 0,
				}, {
					Name:  "SPEED_40G",
					Value: 1,
				}},
			},
		},
		inAnnotateEnumNames: true,
		wantFields: []*protoMsgField{{
			Tag:  29065580,
			Name: "FieldName_someenumtype",
			Type: "SomeEnumType",
		}, {
			Tag:  173535000,
			Name: "FieldName_string",
			Type: "string",
		}},
		wantEnums: map[string]*protoMsgEnum{
			"SomeEnumType": {
				Values: map[int64]protoEnumValue{
					0: {ProtoLabel: "UNSET"},
					1: {ProtoLabel: "SPEED_2_5G", YANGLabel: "SPEED_2.5G"},
					2: {ProtoLabel: "SPEED_40G", YANGLabel: "SPEED_40G"},
				},
			},
		},
	}, {
		name:   "union with an enumeration, but union is typedef",
		inName: "FieldName",
		inField: &ygen.NodeDetails{
			Name: "field-name",
			Type: ygen.LeafNode,
			YANGDetails: ygen.YANGNodeDetails{
				Path: "/field-name",
			},
			LangType: &ygen.MappedType{
				UnionTypes: map[string]ygen.MappedUnionSubtype{
					"SomeEnumType": {
						Index:                 0,
						EnumeratedYANGTypeKey: "/field-name|enum",
					},
					"string": {
						Index: 1,
					},
				},
			},
		},
		inMappedType: &ygen.MappedType{
			UnionTypes: map[string]ygen.MappedUnionSubtype{
				"SomeEnumType": {
					Index:                 0,
					EnumeratedYANGTypeKey: "/field-name|enum",
				},
				"string": {
					Index: 1,
				},
			},
		},
		inEnums: map[string]*ygen.EnumeratedYANGType{
			"/field-name|enum": {
				Name:     "SomeEnumType",
				Kind:     ygen.DerivedUnionEnumerationType,
				TypeName: "derived-union",
				ValToYANGDetails: []ygot.EnumDefinition{{
					Name:  "SPEED_2.5G",
					Value: 0,
				}, {
					Name:  "SPEED_40G",
					Value: 1,
				}},
			},
		},
		inAnnotateEnumNames: true,
		wantFields: []*protoMsgField{{
			Tag:  29065580,
			Name: "FieldName_someenumtype",
			Type: "SomeEnumType",
		}, {
			Tag:  173535000,
			Name: "FieldName_string",
			Type: "string",
		}},
		wantEnums: nil,
	}, {
		name:   "leaflist of union",
		inName: "FieldName",
		inField: &ygen.NodeDetails{
			Name: "field-name",
			Type: ygen.LeafListNode,
			YANGDetails: ygen.YANGNodeDetails{
				Name: "field-name",
				Path: "/parent/field-name",
			},
			MappedPaths: [][]string{{"", "parent", "field-name"}},
			LangType: &ygen.MappedType{
				UnionTypes: map[string]ygen.MappedUnionSubtype{
					"string": {
						Index: 0,
					},
					"uint64": {
						Index: 1,
					},
				},
			},
		},
		inMappedType: &ygen.MappedType{
			UnionTypes: map[string]ygen.MappedUnionSubtype{
				"string": {
					Index: 0,
				},
				"uint64": {
					Index: 1,
				},
			},
		},
		inAnnotateSchemaPaths: true,
		wantRepeatedMsg: &protoMsg{
			Name:     "FieldNameUnion",
			YANGPath: "/parent/field-name union field field-name",
			Fields: []*protoMsgField{{
				Tag:  85114709,
				Name: "FieldName_string",
				Type: "string",
				Options: []*protoOption{{
					Name:  "(yext.schemapath)",
					Value: `"/parent/field-name"`,
				}},
			}, {
				Tag:  192993976,
				Name: "FieldName_uint64",
				Type: "uint64",
				Options: []*protoOption{{
					Name:  "(yext.schemapath)",
					Value: `"/parent/field-name"`,
				}},
			}},
		},
	}}

	for _, tt := range tests {
		if tt.inPath == "" {
			tt.inPath = tt.inField.YANGDetails.Path
		}
		got, err := unionFieldToOneOf(tt.inName, tt.inField, tt.inPath, tt.inMappedType, tt.inEnums, tt.inAnnotateEnumNames, tt.inAnnotateSchemaPaths)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: unionFieldToOneOf(%s, %v, %v, %v): did not get expected error, got: %v, wanted err: %v", tt.name, tt.inName, tt.inField, tt.inMappedType, tt.inAnnotateEnumNames, err, tt.wantErr)
		}

		if err != nil {
			continue
		}

		if diff := cmp.Diff(got.oneOfFields, tt.wantFields); diff != "" {
			t.Errorf("%s: unionFieldToOneOf(%s, %v, %v, %v): did not get expected set of fields, diff(-got,+want):\n%s", tt.name, tt.inName, tt.inField, tt.inMappedType, tt.inAnnotateEnumNames, diff)
		}

		if diff := cmp.Diff(got.enums, tt.wantEnums, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("%s: unionFieldToOneOf(%s, %v, %v, %v): did not get expected set of enums, diff(-got,+want):\n%s", tt.name, tt.inName, tt.inField, tt.inMappedType, tt.inAnnotateEnumNames, diff)
		}

		if diff := cmp.Diff(got.repeatedMsg, tt.wantRepeatedMsg); diff != "" {
			t.Errorf("%s: unionFieldToOneOf(%s, %v, %v, %v): did not get expected repeated message, diff(-got,+want):\n%s", tt.name, tt.inName, tt.inField, tt.inMappedType, tt.inAnnotateEnumNames, diff)
		}
	}
}

func TestStripPackagePrefix(t *testing.T) {
	tests := []struct {
		name         string
		inPrefix     string
		inPath       string
		want         string
		wantStripped bool
	}{{
		name:         "invalid prefix at element one",
		inPrefix:     "one.two",
		inPath:       "two.four",
		want:         "two.four",
		wantStripped: false,
	}, {
		name:         "single element prefix",
		inPrefix:     "one",
		inPath:       "one.three",
		want:         "three",
		wantStripped: true,
	}, {
		name:         "longer prefix",
		inPrefix:     "one.two.three",
		inPath:       "one.two.three.five",
		want:         "five",
		wantStripped: true,
	}}

	for _, tt := range tests {
		got, stripped := stripPackagePrefix(tt.inPrefix, tt.inPath)
		if got != tt.want {
			t.Errorf("%s: stripPackagePrefix(%s, %s): did not get expected output, got: %s, want: %s", tt.name, tt.inPrefix, tt.inPath, got, tt.want)
		}

		if stripped != tt.wantStripped {
			t.Errorf("%s: stripPackagePrefix(%s, %s): did not get expected stipped status, got: %v, want: %v", tt.name, tt.inPrefix, tt.inPath, stripped, tt.wantStripped)
		}
	}

}
