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
package ygen

import (
	"reflect"
	"sort"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
)

func protoMsgEq(a, b protoMsg) bool {
	if a.Name != b.Name {
		return false
	}

	if a.YANGPath != b.YANGPath {
		return false
	}

	if a.Imports != nil && b.Imports != nil && !reflect.DeepEqual(a.Imports, b.Imports) {
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

	if !reflect.DeepEqual(fieldMap(a.Fields), fieldMap(b.Fields)) {
		return false
	}

	return true
}

func TestGenProto3Msg(t *testing.T) {
	tests := []struct {
		name                   string
		inMsg                  *yangDirectory
		inMsgs                 map[string]*yangDirectory
		inUniqueDirectoryNames map[string]string
		inCompressPaths        bool
		inBasePackage          string
		inEnumPackage          string
		inBaseImportPath       string
		inAnnotateSchemaPaths  bool
		wantMsgs               map[string]protoMsg
		wantErr                bool
	}{{
		name: "simple message with only scalar fields",
		inMsg: &yangDirectory{
			name: "MessageName",
			entry: &yang.Entry{
				Name: "message-name",
				Dir:  map[string]*yang.Entry{},
				Kind: yang.DirectoryEntry,
			},
			fields: map[string]*yang.Entry{
				"field-one": {
					Name: "field-one",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
				"field-two": {
					Name: "field-two",
					Type: &yang.YangType{Kind: yang.Yint8},
				},
			},
			path: []string{"", "root", "message-name"},
		},
		inBasePackage: "base",
		inEnumPackage: "enums",
		wantMsgs: map[string]protoMsg{
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
		name: "simple message with union leaf and leaf-list",
		inMsg: &yangDirectory{
			name: "MessageName",
			entry: &yang.Entry{
				Name: "message-name",
				Dir:  map[string]*yang.Entry{},
				Kind: yang.DirectoryEntry,
			},
			fields: map[string]*yang.Entry{
				"field-one": {
					Name: "field-one",
					Type: &yang.YangType{
						Kind: yang.Yunion,
						Type: []*yang.YangType{
							{Kind: yang.Ystring},
							{Kind: yang.Yint8},
						},
					},
				},
				"field-two": {
					Name:     "field-two",
					ListAttr: &yang.ListAttr{},
					Type: &yang.YangType{
						Kind: yang.Yunion,
						Type: []*yang.YangType{
							{Kind: yang.Yint32},
							{
								Kind: yang.Yenum,
								Name: "derived-enum",
								Enum: &yang.EnumType{},
							},
						},
					},
					Parent: &yang.Entry{Name: "parent"},
					Node: &yang.Leaf{
						Name: "leaf",
						Parent: &yang.Module{
							Name: "base",
						},
					},
				},
			},
			path: []string{"", "root", "message-name"},
		},
		inBasePackage: "base",
		inEnumPackage: "enums",
		wantMsgs: map[string]protoMsg{
			"MessageName": {
				Name:     "MessageName",
				YANGPath: "/root/message-name",
				Imports:  []string{"base/enums"},
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
					Type:       "ParentFieldTwoUnion",
					IsRepeated: true,
				}},
			},
			"ParentFieldTwoUnion": {
				Name:     "ParentFieldTwoUnion",
				YANGPath: "/parent/field-two union field field-two",
				Fields: []*protoMsgField{{
					Tag:  305727351,
					Name: "field_two_basederivedenumenum",
					Type: "base.enums.BaseDerivedEnumEnum",
				}, {
					Tag:  226381575,
					Name: "field_two_sint64",
					Type: "sint64",
				}},
			},
		},
	}, {
		name: "simple message with leaf-list and a message child, compression on",
		inMsg: &yangDirectory{
			name: "AMessage",
			entry: &yang.Entry{
				Name: "a-message",
				Dir:  map[string]*yang.Entry{},
				Kind: yang.DirectoryEntry,
			},
			fields: map[string]*yang.Entry{
				"leaf-list": {
					Name:     "leaf-list",
					Type:     &yang.YangType{Kind: yang.Ystring},
					ListAttr: &yang.ListAttr{},
				},
				"container-child": {
					Name: "container-child",
					Dir:  map[string]*yang.Entry{},
					Kind: yang.DirectoryEntry,
					Parent: &yang.Entry{
						Name: "a-message",
						Parent: &yang.Entry{
							Name: "root",
						},
					},
				},
			},
			path: []string{"", "root", "a-message"},
		},
		inMsgs: map[string]*yangDirectory{
			"/root/a-message/container-child": {
				name: "ContainerChild",
				entry: &yang.Entry{
					Name: "container-child",
					Parent: &yang.Entry{
						Name: "a-message",
						Parent: &yang.Entry{
							Name: "root",
						},
					},
				},
			},
		},
		inCompressPaths: true,
		inBasePackage:   "base",
		inEnumPackage:   "enums",
		wantMsgs: map[string]protoMsg{
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
					Type: "base.a_message.ContainerChild",
				}},
				Imports: []string{"base/a_message"},
			},
		},
	}, {
		name: "simple message with leaf-list and a message child, compression off",
		inMsg: &yangDirectory{
			name: "AMessage",
			entry: &yang.Entry{
				Name: "a-message",
				Dir:  map[string]*yang.Entry{},
				Kind: yang.DirectoryEntry,
			},
			fields: map[string]*yang.Entry{
				"leaf-list": {
					Name:     "leaf-list",
					Type:     &yang.YangType{Kind: yang.Ystring},
					ListAttr: &yang.ListAttr{},
				},
				"container-child": {
					Name: "container-child",
					Dir:  map[string]*yang.Entry{},
					Kind: yang.DirectoryEntry,
					Parent: &yang.Entry{
						Name: "a-message",
						Parent: &yang.Entry{
							Name: "root",
						},
					},
				},
			},
			path: []string{"", "root", "a-message"},
		},
		inMsgs: map[string]*yangDirectory{
			"/root/a-message/container-child": {
				name: "ContainerChild",
				entry: &yang.Entry{
					Name: "container-child",
					Parent: &yang.Entry{
						Name: "a-message",
						Parent: &yang.Entry{
							Name: "root",
						},
					},
				},
			},
		},
		inBasePackage: "base",
		inEnumPackage: "enums",
		wantMsgs: map[string]protoMsg{
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
					Type: "base.root.a_message.ContainerChild",
				}},
				Imports: []string{"base/root/a_message"},
			},
		},
	}, {
		name: "message with list",
		inMsg: &yangDirectory{
			name: "AMessageWithAList",
			entry: &yang.Entry{
				Name: "a-message-with-a-list",
				Dir:  map[string]*yang.Entry{},
				Kind: yang.DirectoryEntry,
			},
			fields: map[string]*yang.Entry{
				"list": {
					Name: "list",
					Parent: &yang.Entry{
						Name: "a-message-with-a-list",
					},
					Kind: yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"key": {
							Name: "key",
							Type: &yang.YangType{Kind: yang.Ystring},
						},
					},
					Key:      "key",
					ListAttr: &yang.ListAttr{},
				},
			},
			path: []string{"", "a-message-with-a-list", "list"},
		},
		inBasePackage: "base",
		inEnumPackage: "enums",
		inUniqueDirectoryNames: map[string]string{
			"/a-message-with-a-list/list": "List",
		},
		inMsgs: map[string]*yangDirectory{
			"/a-message-with-a-list/list": &yangDirectory{
				name: "List",
				entry: &yang.Entry{
					Name: "list",
					Parent: &yang.Entry{
						Name: "a-message-with-a-list",
					},
					Kind: yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"key": {
							Name: "key",
							Type: &yang.YangType{Kind: yang.Ystring},
						},
					},
					Key:      "key",
					ListAttr: &yang.ListAttr{},
				},
				fields: map[string]*yang.Entry{
					"key": {
						Name: "key",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
				},
			},
		},
		wantMsgs: map[string]protoMsg{
			"AMessageWithAList": protoMsg{
				Name:     "AMessageWithAList",
				YANGPath: "/a-message-with-a-list/list",
				Fields: []*protoMsgField{{
					Name:       "list",
					Type:       "ListKey",
					Tag:        200573382,
					IsRepeated: true,
				}},
			},
			"ListKey": protoMsg{
				Name:     "ListKey",
				YANGPath: "/a-message-with-a-list/list",
				Fields: []*protoMsgField{{
					Tag:        1,
					Name:       "key",
					Type:       "string",
					IsRepeated: false,
				}, {
					Tag:  2,
					Name: "list",
					Type: "base.a_message_with_a_list.List",
				}},
				Imports: []string{"base/a_message_with_a_list"},
			},
		},
	}, {
		name: "message with missing directory",
		inMsg: &yangDirectory{
			name:  "foo",
			entry: &yang.Entry{Name: "foo"},
			fields: map[string]*yang.Entry{
				"bar": {
					Name: "bar",
					Kind: yang.DirectoryEntry,
					Dir:  map[string]*yang.Entry{},
				},
			},
		},
		wantErr: true,
	}, {
		name: "message with an unimplemented mapping",
		inMsg: &yangDirectory{
			name: "MessageWithInvalidContents",
			entry: &yang.Entry{
				Name: "message-with-invalid-contents",
				Dir:  map[string]*yang.Entry{},
				Kind: yang.DirectoryEntry,
			},
			fields: map[string]*yang.Entry{
				"unimplemented": {
					Name: "unimplemented",
					Kind: yang.LeafEntry,
					Type: &yang.YangType{
						Kind: yang.Yunion,
						Type: []*yang.YangType{
							{Kind: yang.Ybinary},
							{Kind: yang.Ybits},
							{Kind: yang.YinstanceIdentifier},
						},
					},
				},
			},
			path: []string{"", "messasge-with-invalid-contents", "unimplemented"},
		},
		wantErr: true,
	}, {
		name: "message with any anydata field",
		inMsg: &yangDirectory{
			name: "MessageWithAnydata",
			entry: &yang.Entry{
				Name: "message-with-anydata",
				Kind: yang.DirectoryEntry,
				Dir:  map[string]*yang.Entry{},
			},
			fields: map[string]*yang.Entry{
				"any-data": {
					Name: "any-data",
					Kind: yang.AnyDataEntry,
				},
				"leaf": {
					Name: "leaf",
					Kind: yang.LeafEntry,
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
			path: []string{"", "message-with-anydata"},
		},
		inBasePackage: "base",
		inEnumPackage: "enums",
		wantMsgs: map[string]protoMsg{
			"MessageWithAnydata": {
				Name:     "MessageWithAnydata",
				YANGPath: "/message-with-anydata",
				Imports:  []string{"google/protobuf/any"},
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
		inMsg: &yangDirectory{
			name: "MessageWithAnnotations",
			entry: &yang.Entry{
				Name: "message-with-annotations",
				Kind: yang.DirectoryEntry,
				Dir:  map[string]*yang.Entry{},
				Parent: &yang.Entry{
					Name: "two",
					Parent: &yang.Entry{
						Name: "one",
					},
				},
			},
			fields: map[string]*yang.Entry{
				"leaf": {
					Name: "leaf",
					Kind: yang.LeafEntry,
					Type: &yang.YangType{Kind: yang.Ystring},
					Parent: &yang.Entry{
						Name: "two",
						Parent: &yang.Entry{
							Name: "one",
						},
					},
				},
			},
			path: []string{"", "one", "two"},
		},
		inBasePackage:         "base",
		inEnumPackage:         "enums",
		inAnnotateSchemaPaths: true,
		wantMsgs: map[string]protoMsg{
			"MessageWithAnnotations": protoMsg{
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
		s := newGenState()
		// Seed the state with the supplied message names that have been provided.
		s.uniqueDirectoryNames = tt.inUniqueDirectoryNames

		gotMsgs, errs := genProto3Msg(tt.inMsg, tt.inMsgs, s, protoMsgConfig{
			compressPaths:       tt.inCompressPaths,
			basePackageName:     tt.inBasePackage,
			enumPackageName:     tt.inEnumPackage,
			baseImportPath:      tt.inBaseImportPath,
			annotateSchemaPaths: tt.inAnnotateSchemaPaths,
		})

		if (errs != nil) != tt.wantErr {
			t.Errorf("s: genProtoMsg(%#v, %#v, *genState, %v, %v, %s, %s): did not get expected error status, got: %v, wanted err: %v", tt.name, tt.inMsg, tt.inMsgs, tt.inCompressPaths, tt.inBasePackage, tt.inEnumPackage, errs, tt.wantErr)
		}

		if tt.wantErr {
			continue
		}

		notSeen := map[string]bool{}
		for _, w := range tt.wantMsgs {
			notSeen[w.Name] = true
		}

		for _, got := range gotMsgs {
			want, ok := tt.wantMsgs[got.Name]
			if !ok {
				t.Errorf("%s: genProtoMsg(%#v, %#v, *genState): got unexpected message, got: %v, want: %v", tt.name, tt.inMsg, tt.inMsgs, got.Name, tt.wantMsgs)
				continue
			}
			delete(notSeen, got.Name)

			if !protoMsgEq(got, want) {
				diff := pretty.Compare(got, want)
				t.Errorf("%s: genProtoMsg(%#v, %#v, *genState): did not get expected protobuf message definition, diff(-got,+want):\n%s", tt.name, tt.inMsg, tt.inMsgs, diff)
			}
		}

		if len(notSeen) != 0 {
			t.Errorf("%s: genProtoMsg(%#v, %#v, *genState); did not test all returned messages, got remaining messages: %v, want: none", tt.name, tt.inMsg, tt.inMsgs, notSeen)
		}
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
	// A definition of an enumerated type.
	enumeratedLeafDef := yang.NewEnumType()
	enumeratedLeafDef.Set("ONE", int64(1))
	enumeratedLeafDef.Set("FORTYTWO", int64(42))

	tests := []struct {
		name                   string
		inMsg                  *yangDirectory
		inMsgs                 map[string]*yangDirectory
		inBasePackageName      string
		inEnumPackageName      string
		inBaseImportPath       string
		inUniqueDirectoryNames map[string]string
		wantCompress           generatedProto3Message
		wantUncompress         generatedProto3Message
		wantCompressErr        bool
		wantUncompressErr      bool
	}{{
		name: "simple message with scalar fields",
		inMsg: &yangDirectory{
			name: "MessageName",
			entry: &yang.Entry{
				Name: "message-name",
				Kind: yang.DirectoryEntry,
				Dir:  map[string]*yang.Entry{},
				Parent: &yang.Entry{
					Name: "container",
					Kind: yang.DirectoryEntry,
					Dir:  map[string]*yang.Entry{},
					Parent: &yang.Entry{
						Name: "module",
						Kind: yang.DirectoryEntry,
						Dir:  map[string]*yang.Entry{},
					},
				},
				Node: &yang.Container{Name: "message-name"},
			},
			fields: map[string]*yang.Entry{
				"field-one": &yang.Entry{
					Name: "field-one",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
			path: []string{"", "module", "container", "message-name"},
		},
		inBasePackageName: "base",
		inEnumPackageName: "enums",
		wantCompress: generatedProto3Message{
			packageName: "container",
			messageCode: `
// MessageName represents the /module/container/message-name YANG schema element.
message MessageName {
  ywrapper.StringValue field_one = 410095931;
}
`,
		},
		wantUncompress: generatedProto3Message{
			packageName: "module.container",
			messageCode: `
// MessageName represents the /module/container/message-name YANG schema element.
message MessageName {
  ywrapper.StringValue field_one = 410095931;
}
`,
		},
	}, {
		name: "simple message with other messages embedded",
		inMsg: &yangDirectory{
			name: "MessageName",
			entry: &yang.Entry{
				Name: "message-name",
				Kind: yang.DirectoryEntry,
				Parent: &yang.Entry{
					Name: "module",
					Kind: yang.DirectoryEntry,
				},
			},
			fields: map[string]*yang.Entry{
				"child": &yang.Entry{
					Name: "child",
					Kind: yang.DirectoryEntry,
					Dir:  map[string]*yang.Entry{},
					Parent: &yang.Entry{
						Name: "message-name",
						Kind: yang.DirectoryEntry,
						Parent: &yang.Entry{
							Name: "module",
							Kind: yang.DirectoryEntry,
						},
					},
				},
			},
			path: []string{"", "module", "message-name"},
		},
		inMsgs: map[string]*yangDirectory{
			"/module/message-name/child": &yangDirectory{
				name: "Child",
				entry: &yang.Entry{
					Name: "child",
					Kind: yang.DirectoryEntry,
					Parent: &yang.Entry{
						Name: "message-name",
						Kind: yang.DirectoryEntry,
						Parent: &yang.Entry{
							Name: "module",
							Kind: yang.DirectoryEntry,
						},
					},
				},
			},
		},
		inBasePackageName: "base",
		inEnumPackageName: "enums",
		wantCompress: generatedProto3Message{
			packageName: "",
			messageCode: `
// MessageName represents the /module/message-name YANG schema element.
message MessageName {
  base.message_name.Child child = 399980855;
}
`,
		},
		wantUncompress: generatedProto3Message{
			packageName: "module",
			messageCode: `
// MessageName represents the /module/message-name YANG schema element.
message MessageName {
  base.module.message_name.Child child = 399980855;
}
`,
		},
	}, {
		name: "simple message with an enumeration leaf",
		inMsg: &yangDirectory{
			name: "MessageName",
			entry: &yang.Entry{
				Name: "message-name",
				Kind: yang.DirectoryEntry,
				Parent: &yang.Entry{
					Name: "module",
					Kind: yang.DirectoryEntry,
				},
			},
			fields: map[string]*yang.Entry{
				"enum": &yang.Entry{
					Name: "enum",
					Kind: yang.LeafEntry,
					Parent: &yang.Entry{
						Name: "message-name",
						Parent: &yang.Entry{
							Name: "module",
						},
					},
					Type: &yang.YangType{
						Name: "enumeration",
						Kind: yang.Yenum,
						Enum: enumeratedLeafDef,
					},
				},
			},
			path: []string{"", "module", "message-name"},
		},
		inBasePackageName: "base",
		inEnumPackageName: "enums",
		wantCompress: generatedProto3Message{
			packageName: "",
			messageCode: `
// MessageName represents the /module/message-name YANG schema element.
message MessageName {
  enum Enum {
    Enum_UNSET = 0;
    Enum_ONE = 2;
    Enum_FORTYTWO = 43;
  }
  Enum enum = 278979784;
}
`,
		},
		wantUncompress: generatedProto3Message{
			packageName: "module",
			messageCode: `
// MessageName represents the /module/message-name YANG schema element.
message MessageName {
  enum Enum {
    Enum_UNSET = 0;
    Enum_ONE = 2;
    Enum_FORTYTWO = 43;
  }
  Enum enum = 278979784;
}
`,
		},
	}, {
		name: "simple message with a list",
		inMsg: &yangDirectory{
			name: "MessageName",
			entry: &yang.Entry{
				Name: "message-name",
				Kind: yang.DirectoryEntry,
				Parent: &yang.Entry{
					Name: "module",
					Kind: yang.DirectoryEntry,
				},
			},
			fields: map[string]*yang.Entry{
				"list": &yang.Entry{
					Name: "list",
					Kind: yang.DirectoryEntry,
					Parent: &yang.Entry{
						Name: "message-name",
						Parent: &yang.Entry{
							Name: "module",
							Kind: yang.DirectoryEntry,
						},
					},
					Key:      "keyfield",
					ListAttr: &yang.ListAttr{},
					Dir: map[string]*yang.Entry{
						"keyfield": {
							Name: "keyfield",
							Type: &yang.YangType{
								Kind: yang.Ystring,
							},
						},
					},
				},
			},
		},
		inMsgs: map[string]*yangDirectory{
			"/module/message-name/list": {
				name: "ListMessageName",
				entry: &yang.Entry{
					Name: "list",
					Kind: yang.DirectoryEntry,
					Parent: &yang.Entry{
						Name: "message-name",
						Parent: &yang.Entry{
							Name: "module",
							Kind: yang.DirectoryEntry,
						},
					},
					Key:      "keyfield",
					ListAttr: &yang.ListAttr{},
					Dir: map[string]*yang.Entry{
						"keyfield": {
							Name: "keyfield",
							Type: &yang.YangType{
								Kind: yang.Ystring,
							},
						},
					},
				},
				fields: map[string]*yang.Entry{
					"keyfield": {
						Name: "keyfield",
						Type: &yang.YangType{
							Kind: yang.Ystring,
						},
					},
				},
			},
		},
		inUniqueDirectoryNames: map[string]string{"/module/message-name/list": "List"},
		inBasePackageName:      "base",
		inEnumPackageName:      "enums",
		wantCompress: generatedProto3Message{
			packageName: "message_name",
			messageCode: `
// ListKey represents the /module/message-name/list YANG schema element.
message ListKey {
  string keyfield = 1;
  base.message_name.List list = 2;
}

// MessageName represents the  YANG schema element.
message MessageName {
  repeated ListKey list = 140998691;
}
`,
		},
		wantUncompress: generatedProto3Message{
			packageName: "module",
			messageCode: `
// ListKey represents the /module/message-name/list YANG schema element.
message ListKey {
  string keyfield = 1;
  base.module.message_name.List list = 2;
}

// MessageName represents the  YANG schema element.
message MessageName {
  repeated ListKey list = 140998691;
}
`,
		},
	}, {
		name: "simple message with an identityref leaf",
		inMsg: &yangDirectory{
			name: "MessageName",
			entry: &yang.Entry{
				Name: "message-name",
				Kind: yang.DirectoryEntry,
				Parent: &yang.Entry{
					Name: "module",
					Kind: yang.DirectoryEntry,
				},
			},
			fields: map[string]*yang.Entry{
				"identityref": &yang.Entry{
					Name: "identityref",
					Kind: yang.LeafEntry,
					Parent: &yang.Entry{
						Name: "message-name",
						Parent: &yang.Entry{
							Name: "module",
						},
					},
					Type: &yang.YangType{
						Name: "identityref",
						Kind: yang.Yidentityref,
						IdentityBase: &yang.Identity{
							Name: "foo-identity",
							Values: []*yang.Identity{
								{Name: "ONE"},
								{Name: "TWO"},
							},
							Parent: &yang.Module{
								Name: "test-module",
							},
						},
					},
				},
			},
			path: []string{"", "module-name", "message-name"},
		},
		inBasePackageName: "base",
		inEnumPackageName: "enums",
		wantCompress: generatedProto3Message{
			packageName: "",
			messageCode: `
// MessageName represents the /module-name/message-name YANG schema element.
message MessageName {
  base.enums.TestModuleFooIdentity identityref = 518954308;
}
`,
		},
		wantUncompress: generatedProto3Message{
			packageName: "module",
			messageCode: `
// MessageName represents the /module-name/message-name YANG schema element.
message MessageName {
  base.enums.TestModuleFooIdentity identityref = 518954308;
}
`,
		},
	}}

	for _, tt := range tests {
		wantErr := map[bool]bool{true: tt.wantCompressErr, false: tt.wantUncompressErr}
		for compress, want := range map[bool]generatedProto3Message{true: tt.wantCompress, false: tt.wantUncompress} {
			s := newGenState()
			// Seed the message names with the supplied input.
			s.uniqueDirectoryNames = tt.inUniqueDirectoryNames

			got, errs := writeProto3Msg(tt.inMsg, tt.inMsgs, s, protoMsgConfig{
				compressPaths:   compress,
				basePackageName: tt.inBasePackageName,
				enumPackageName: tt.inEnumPackageName,
				baseImportPath:  tt.inBaseImportPath,
			})

			if (errs != nil) != wantErr[compress] {
				t.Errorf("%s: writeProto3Msg(%v, %v, %v, %v): did not get expected error return status, got: %v, wanted error: %v", tt.name, tt.inMsg, tt.inMsgs, s, compress, errs, wantErr[compress])
			}

			if errs != nil {
				continue
			}

			if got.packageName != want.packageName {
				t.Errorf("%s: writeProto3Msg(%v, %v, %v, %v): did not get expected package name, got: %v, want: %v", tt.name, tt.inMsg, tt.inMsgs, s, compress, got.packageName, want.packageName)
			}

			if reflect.DeepEqual(got.requiredImports, want.requiredImports) {
				t.Errorf("%s: writeProto3Msg(%v, %v, %v, %v): did not get expected set of imports, got: %v, want: %v", tt.name, tt.inMsg, tt.inMsgs, s, compress, got.requiredImports, want.requiredImports)
			}

			if diff := pretty.Compare(got.messageCode, want.messageCode); diff != "" {
				if diffl, err := generateUnifiedDiff(got.messageCode, want.messageCode); err == nil {
					diff = diffl
				}
				t.Errorf("%s: writeProto3Msg(%v, %v, %v, %v): did not get expected message returned, diff(-got,+want):\n%s", tt.name, tt.inMsg, tt.inMsgs, s, compress, diff)
			}
		}
	}
}

func TestGenListKeyProto(t *testing.T) {
	tests := []struct {
		name          string
		inListPackage string
		inListName    string
		inArgs        protoDefinitionArgs
		wantMsg       *protoMsg
		wantErr       bool
	}{{
		name:          "simple list key proto",
		inListPackage: "pkg",
		inListName:    "list",
		inArgs: protoDefinitionArgs{
			field: &yang.Entry{
				Name:     "list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{},
				Key:      "key",
				Dir:      map[string]*yang.Entry{},
			},
			directory: &yangDirectory{
				name: "List",
				fields: map[string]*yang.Entry{
					"key": {
						Name: "key",
						Type: &yang.YangType{
							Kind: yang.Ystring,
						},
					},
				},
			},
			definedDirectories: map[string]*yangDirectory{},
			state: &genState{
				uniqueDirectoryNames: map[string]string{
					"/list": "List",
				},
			},
			compressPaths:   false,
			basePackageName: "base",
			baseImportPath:  "base/path",
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
				Type: "base.pkg.list",
			}},
			Imports: []string{"base/path/base/pkg"},
		},
	}, {
		name:          "list with union key - string and int",
		inListPackage: "pkg",
		inListName:    "list",
		inArgs: protoDefinitionArgs{
			field: &yang.Entry{
				Name:     "list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{},
				Key:      "key",
				Dir:      map[string]*yang.Entry{},
			},
			directory: &yangDirectory{
				name: "List",
				fields: map[string]*yang.Entry{
					"key": {
						Name: "key",
						Type: &yang.YangType{
							Kind: yang.Yunion,
							Type: []*yang.YangType{
								{Kind: yang.Ystring},
								{Kind: yang.Yint8},
							},
						},
					},
				},
			},
			definedDirectories: map[string]*yangDirectory{},
			state: &genState{
				uniqueDirectoryNames: map[string]string{
					"/list": "List",
				},
			},
			compressPaths:   false,
			basePackageName: "base",
			baseImportPath:  "base/path",
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
				Type: "base.pkg.list",
			}},
			Imports: []string{"base/path/base/pkg"},
		},
	}, {
		name:          "list with union key - two string",
		inListPackage: "pkg",
		inListName:    "list",
		inArgs: protoDefinitionArgs{
			field: &yang.Entry{
				Name:     "list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{},
				Key:      "key",
				Dir:      map[string]*yang.Entry{},
			},
			directory: &yangDirectory{
				name: "List",
				fields: map[string]*yang.Entry{
					"key": {
						Name: "key",
						Type: &yang.YangType{
							Kind: yang.Yunion,
							Type: []*yang.YangType{
								{Kind: yang.Ystring, Pattern: []string{"b.*"}},
								{Kind: yang.Ystring, Pattern: []string{"a.*"}},
							},
						},
					},
				},
			},
			definedDirectories: map[string]*yangDirectory{},
			state: &genState{
				uniqueDirectoryNames: map[string]string{
					"/list": "List",
				},
			},
			compressPaths:   false,
			basePackageName: "base",
			baseImportPath:  "base/path",
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
				Type: "base.pkg.list",
			}},
			Imports: []string{"base/path/base/pkg"},
		},
	}}

	for _, tt := range tests {
		got, err := genListKeyProto(tt.inListPackage, tt.inListName, tt.inArgs)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: genListKeyProto(%s, %s, %#v): got unexpected error returned, got: %v, want err: %v", tt.name, tt.inListPackage, tt.inListName, tt.inArgs, err, tt.wantErr)
		}

		if diff := pretty.Compare(got, tt.wantMsg); diff != "" {
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
		name      string
		inEnums   map[string]*yangEnum
		wantEnums []string
		wantErr   bool
	}{{
		name: "skipped enumeration type",
		inEnums: map[string]*yangEnum{
			"e": &yangEnum{
				name: "e",
				entry: &yang.Entry{
					Name: "e",
					Type: &yang.YangType{
						Name: "enumeration",
						Kind: yang.Yenum,
					},
				},
			},
		},
		wantEnums: []string{},
	}, {
		name: "enum for identityref",
		inEnums: map[string]*yangEnum{
			"EnumeratedValue": {
				name: "EnumeratedValue",
				entry: &yang.Entry{
					Type: &yang.YangType{
						IdentityBase: &yang.Identity{
							Name: "IdentityValue",
							Values: []*yang.Identity{
								{Name: "VALUE_A", Parent: &yang.Module{Name: "mod"}},
								{Name: "VALUE_B", Parent: &yang.Module{Name: "mod2"}},
							},
						},
					},
				},
			},
		},
		wantEnums: []string{
			`
// EnumeratedValue represents an enumerated type generated for the YANG identity IdentityValue.
enum EnumeratedValue {
  ENUMERATEDVALUE_UNSET = 0;
  ENUMERATEDVALUE_VALUE_A = 1;
  ENUMERATEDVALUE_VALUE_B = 2;
}
`,
		},
	}, {
		name: "enum for typedef enumeration",
		inEnums: map[string]*yangEnum{
			"e": &yangEnum{
				name: "EnumName",
				entry: &yang.Entry{
					Name: "e",
					Type: &yang.YangType{
						Name: "typedef",
						Kind: yang.Yenum,
						Enum: testYANGEnums["enumOne"],
					},
					Annotation: map[string]interface{}{
						"valuePrefix": []string{"enum-name"},
					},
				},
			},
			"f": &yangEnum{
				name: "SecondEnum",
				entry: &yang.Entry{
					Name: "f",
					Type: &yang.YangType{
						Name: "derived",
						Kind: yang.Yenum,
						Enum: testYANGEnums["enumTwo"],
					},
					Annotation: map[string]interface{}{
						"valuePrefix": []string{"secondenum"},
					},
				},
			},
		},
		wantEnums: []string{
			`
// EnumName represents an enumerated type generated for the YANG enumerated type typedef.
enum EnumName {
  ENUMNAME_UNSET = 0;
  ENUMNAME_SPEED_2_5G = 1;
  ENUMNAME_SPEED_40G = 2;
}
`, `
// SecondEnum represents an enumerated type generated for the YANG enumerated type derived.
enum SecondEnum {
  SECONDENUM_UNSET = 0;
  SECONDENUM_VALUE_1 = 1;
  SECONDENUM_VALUE_2 = 2;
}
`,
		},
	}}

	for _, tt := range tests {
		got, err := writeProtoEnums(tt.inEnums)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: writeProtoEnums(%v): did not get expected error, got: %v", tt.name, tt.inEnums, err)
		}

		if err != nil {
			continue
		}

		// Sort the returned output to avoid test flakes.
		sort.Strings(got)
		if diff := pretty.Compare(got, tt.wantEnums); diff != "" {
			t.Errorf("%s: writeProtoEnums(%v): did not get expected output, diff(-got,+want):\n%s", tt.name, tt.inEnums, diff)
		}
	}
}

func TestUnionFieldToOneOf(t *testing.T) {
	// Create mock enumerations within goyang since we cannot create them in-line.
	testEnums := map[string][]string{
		"enumOne": {"SPEED_2.5G", "SPEED_40G"},
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
		name            string
		inName          string
		inEntry         *yang.Entry
		inMappedType    *mappedType
		wantFields      []*protoMsgField
		wantEnums       map[string]*protoMsgEnum
		wantRepeatedMsg *protoMsg
		wantErr         bool
	}{{
		name:   "simple string union",
		inName: "FieldName",
		inEntry: &yang.Entry{
			Name: "field-name",
			Type: &yang.YangType{
				Type: []*yang.YangType{
					{Kind: yang.Ystring},
					{Kind: yang.Yint8},
				},
			},
		},
		inMappedType: &mappedType{
			unionTypes: map[string]int{
				"string": 0,
				"sint64": 0,
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
		name:   "decimal64 union",
		inName: "FieldName",
		inEntry: &yang.Entry{
			Name: "field-name",
			Type: &yang.YangType{
				Type: []*yang.YangType{
					{Kind: yang.Ystring},
					{Kind: yang.Ydecimal64},
				},
			},
		},
		inMappedType: &mappedType{
			unionTypes: map[string]int{
				"string":             0,
				"ywrapper.Decimal64": 1,
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
		inEntry: &yang.Entry{
			Name: "field-name",
			Type: &yang.YangType{
				Type: []*yang.YangType{
					{
						Name: "enumeration",
						Kind: yang.Yenum,
						Enum: testYANGEnums["enumOne"],
					},
					{Kind: yang.Ystring},
				},
			},
		},
		inMappedType: &mappedType{
			unionTypes: map[string]int{
				"SomeEnumType": 0,
				"string":       1,
			},
		},
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
			"FieldName": &protoMsgEnum{
				Values: map[int64]string{
					0: "UNSET",
					1: "SPEED_2_5G",
					2: "SPEED_40G",
				},
			},
		},
	}, {
		name:   "leaflist of union",
		inName: "FieldName",
		inEntry: &yang.Entry{
			Name: "field-name",
			Type: &yang.YangType{
				Type: []*yang.YangType{
					{Kind: yang.Ystring},
					{Kind: yang.Yuint8},
				},
			},
			Parent:   &yang.Entry{Name: "parent"},
			ListAttr: &yang.ListAttr{},
		},
		inMappedType: &mappedType{
			unionTypes: map[string]int{
				"string": 0,
				"uint64": 1,
			},
		},
		wantRepeatedMsg: &protoMsg{
			Name:     "ParentFieldNameUnion",
			YANGPath: "/parent/field-name union field field-name",
			Fields: []*protoMsgField{{
				Tag:  85114709,
				Name: "FieldName_string",
				Type: "string",
			}, {
				Tag:  192993976,
				Name: "FieldName_uint64",
				Type: "uint64",
			}},
		},
	}}

	for _, tt := range tests {
		got, err := unionFieldToOneOf(tt.inName, tt.inEntry, tt.inMappedType)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: unionFieldToOneOf(%s, %v, %v): did not get expected error, got: %v, wanted err: %v", tt.name, tt.inName, tt.inEntry, tt.inMappedType, err, tt.wantErr)
		}

		if err != nil {
			continue
		}

		if diff := pretty.Compare(got.oneOfFields, tt.wantFields); diff != "" {
			t.Errorf("%s: unionFieldToOneOf(%s, %v, %v): did not get expected set of fields, diff(-got,+want):\n%s", tt.name, tt.inName, tt.inEntry, tt.inMappedType, diff)
		}

		if diff := pretty.Compare(got.enums, tt.wantEnums); diff != "" {
			t.Errorf("%s: unionFieldToOneOf(%s, %v, %v): did not get expected set of enums, diff(-got,+want):\n%s", tt.name, tt.inName, tt.inEntry, tt.inMappedType, diff)
		}

		if diff := pretty.Compare(got.repeatedMsg, tt.wantRepeatedMsg); diff != "" {
			t.Errorf("%s: unionFieldToOneOf(%s, %v, %v): did not get expected repeated message, diff(-got,+want):\n%s", tt.name, tt.inName, tt.inEntry, tt.inMappedType, diff)
		}
	}
}
