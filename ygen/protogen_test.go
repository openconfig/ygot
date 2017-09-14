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

func TestGenProtoMsg(t *testing.T) {
	tests := []struct {
		name                   string
		inMsg                  *yangDirectory
		inMsgs                 map[string]*yangDirectory
		inUniqueDirectoryNames map[string]string
		inCompressPaths        bool
		inBasePackage          string
		inEnumPackage          string
		wantMsgs               map[string]protoMsg
		wantErr                bool
	}{{
		name: "simple message with only scalar fields",
		inMsg: &yangDirectory{
			name: "MessageName",
			entry: &yang.Entry{
				Name: "message-name",
				Dir:  map[string]*yang.Entry{},
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
		wantMsgs: map[string]protoMsg{
			"MessageName": {
				Name:     "MessageName",
				YANGPath: "/root/message-name",
				Fields: []*protoMsgField{{
					Tag:  1,
					Name: "field_one",
					Type: "ywrapper.StringValue",
				}, {
					Tag:  1,
					Name: "field_two",
					Type: "ywrapper.IntValue",
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
		wantMsgs: map[string]protoMsg{
			"AMessage": {
				Name:     "AMessage",
				YANGPath: "/root/a-message",
				Fields: []*protoMsgField{{
					Tag:        1,
					Name:       "leaf_list",
					Type:       "ywrapper.StringValue",
					IsRepeated: true,
				}, {
					Tag:  1,
					Name: "container_child",
					Type: "a_message.ContainerChild",
				}},
			},
		},
	}, {
		name: "simple message with leaf-list and a message child, compression off",
		inMsg: &yangDirectory{
			name: "AMessage",
			entry: &yang.Entry{
				Name: "a-message",
				Dir:  map[string]*yang.Entry{},
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
		wantMsgs: map[string]protoMsg{
			"AMessage": {
				Name:     "AMessage",
				YANGPath: "/root/a-message",
				Fields: []*protoMsgField{{
					Tag:        1,
					Name:       "leaf_list",
					Type:       "ywrapper.StringValue",
					IsRepeated: true,
				}, {
					Tag:  1,
					Name: "container_child",
					Type: "root.a_message.ContainerChild",
				}},
			},
		},
	}, {
		name: "message with unimplemented list",
		inMsg: &yangDirectory{
			name: "AMessageWithAList",
			entry: &yang.Entry{
				Name: "a-message-with-a-list",
				Dir:  map[string]*yang.Entry{},
			},
			fields: map[string]*yang.Entry{
				"list": {
					Name: "list",
					Parent: &yang.Entry{
						Name: "a-message-with-a-list",
					},
					Dir: map[string]*yang.Entry{
						"key": {
							Name: "key",
							Type: &yang.YangType{Kind: yang.Ystring},
						},
					},
					Key: "key",
				},
			},
			path: []string{"", "a-messsage-with-a-list", "list"},
		},
		wantErr: true,
	}, {
		name: "message with an unimplemented mapping",
		inMsg: &yangDirectory{
			name: "MessageWithInvalidContents",
			entry: &yang.Entry{
				Name: "message-with-invalid-contents",
				Dir:  map[string]*yang.Entry{},
			},
			fields: map[string]*yang.Entry{
				"unimplemented": {
					Name: "unimplemented",
					Kind: yang.LeafEntry,
					Type: &yang.YangType{
						Kind: yang.Yunion,
						Type: []*yang.YangType{
							{Kind: yang.Ybinary},
							{Kind: yang.Yenum},
							{Kind: yang.Ybits},
							{Kind: yang.YinstanceIdentifier},
						},
					},
				},
			},
			path: []string{"", "mesassge-with-invalid-contents", "unimplemented"},
		},
		wantErr: true,
	}}

	for _, tt := range tests {
		s := newGenState()
		// Seed the state with the supplied message names that have been provided.
		s.uniqueDirectoryNames = tt.inUniqueDirectoryNames

		gotMsgs, errs := genProto3Msg(tt.inMsg, tt.inMsgs, s, tt.inCompressPaths, tt.inBasePackage, tt.inEnumPackage)
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
				t.Errorf("%s: genProtoMsg(%#v, %#v, *genState): got unexpected expected message, got: nil, want: %v", tt.name, tt.inMsg, tt.inMsgs, got.Name)
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
		name              string
		inMsg             *yangDirectory
		inMsgs            map[string]*yangDirectory
		inBasePackageName string
		inEnumPackageName string
		wantCompress      generatedProto3Message
		wantUncompress    generatedProto3Message
		wantCompressErr   bool
		wantUncompressErr bool
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
			},
			fields: map[string]*yang.Entry{
				"field-one": &yang.Entry{
					Name: "field-one",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
			path: []string{"", "module", "container", "message-name"},
		},
		wantCompress: generatedProto3Message{
			packageName: "container",
			messageCode: `
// MessageName represents the /module/container/message-name YANG schema element.
message MessageName {
  ywrapper.StringValue field_one = 1;
}
`,
		},
		wantUncompress: generatedProto3Message{
			packageName: "module.container",
			messageCode: `
// MessageName represents the /module/container/message-name YANG schema element.
message MessageName {
  ywrapper.StringValue field_one = 1;
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
		wantCompress: generatedProto3Message{
			packageName: "",
			messageCode: `
// MessageName represents the /module/message-name YANG schema element.
message MessageName {
  message_name.Child child = 1;
}
`,
		},
		wantUncompress: generatedProto3Message{
			packageName: "module",
			messageCode: `
// MessageName represents the /module/message-name YANG schema element.
message MessageName {
  module.message_name.Child child = 1;
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
  Enum enum = 1;
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
  Enum enum = 1;
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
  base.enums.TestModule_FooIdentity identityref = 1;
}
`,
		},
		wantUncompress: generatedProto3Message{
			packageName: "module",
			messageCode: `
// MessageName represents the /module-name/message-name YANG schema element.
message MessageName {
  base.enums.TestModule_FooIdentity identityref = 1;
}
`,
		},
	}}

	for _, tt := range tests {
		wantErr := map[bool]bool{true: tt.wantCompressErr, false: tt.wantUncompressErr}
		for compress, want := range map[bool]generatedProto3Message{true: tt.wantCompress, false: tt.wantUncompress} {
			s := newGenState()

			got, errs := writeProto3Msg(tt.inMsg, tt.inMsgs, s, compress, tt.inBasePackageName, tt.inEnumPackageName)
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
  EnumeratedValue_UNSET = 0;
  EnumeratedValue_VALUE_A = 1;
  EnumeratedValue_VALUE_B = 2;
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
				},
			},
		},
		wantEnums: []string{
			`
// EnumName represents an enumerated type generated for the YANG enumerated type typedef.
enum EnumName {
  EnumName_UNSET = 0;
  EnumName_SPEED_2_5G = 1;
  EnumName_SPEED_40G = 2;
}
`, `
// SecondEnum represents an enumerated type generated for the YANG enumerated type derived.
enum SecondEnum {
  SecondEnum_UNSET = 0;
  SecondEnum_VALUE_1 = 1;
  SecondEnum_VALUE_2 = 2;
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
