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
		wantMsg                protoMsg
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
		wantMsg: protoMsg{
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
		wantMsg: protoMsg{
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
		wantMsg: protoMsg{
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

		got, errs := genProto3Msg(tt.inMsg, tt.inMsgs, s, tt.inCompressPaths)
		if (len(errs) > 0) != tt.wantErr {
			t.Errorf("s: genProto3Msg(%#v, %#v, *genState, %v): did not get expected error status, got: %v, wanted err: %v", tt.name, tt.inMsg, tt.inMsgs, tt.inCompressPaths, errs, tt.wantErr)
		}

		if tt.wantErr {
			continue
		}

		if !protoMsgEq(got, tt.wantMsg) {
			diff := pretty.Compare(got, tt.wantMsg)
			t.Errorf("%s: genProto3Msg(%#v, %#v, *genState): did not get expected protobuf message definition, diff(-got,+want):\n%s", tt.name, tt.inMsg, tt.inMsgs, diff)
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
		if got := safeProtoFieldName(tt.in); got != tt.want {
			t.Errorf("%s: safeProtoFieldName(%s): did not get expected name, got: %v, want: %v", tt.name, tt.in, got, tt.want)
		}
	}
}

// writeProtoTestResult stores the result of a test for writeProto3Msg.
type writeProto3MsgTestResult struct {
	pkg     string   // pkg stores the expected package returned from writeProto3Msg.
	msg     string   // msg stores the expected message code returned.
	imports []string // imports stores the expected set of imports for this message.
	err     bool     // err stores whether there are expected to be returned errors.
}

func TestWriteProtoMsg(t *testing.T) {
	tests := []struct {
		name           string
		inMsg          *yangDirectory
		inMsgs         map[string]*yangDirectory
		wantCompress   writeProto3MsgTestResult
		wantUncompress writeProto3MsgTestResult
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
		wantCompress: writeProto3MsgTestResult{
			pkg: "container",
			msg: `
// MessageName represents the /module/container/message-name YANG schema element.
message MessageName {
  ywrapper.StringValue field_one = 1;
}`,
		},
		wantUncompress: writeProto3MsgTestResult{
			pkg: "module.container",
			msg: `
// MessageName represents the /module/container/message-name YANG schema element.
message MessageName {
  ywrapper.StringValue field_one = 1;
}`,
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
		wantCompress: writeProto3MsgTestResult{
			pkg: "",
			msg: `
// MessageName represents the /module/message-name YANG schema element.
message MessageName {
  message_name.Child child = 1;
}`,
		},
		wantUncompress: writeProto3MsgTestResult{
			pkg: "module",
			msg: `
// MessageName represents the /module/message-name YANG schema element.
message MessageName {
  module.message_name.Child child = 1;
}`,
		},
	}}

	for _, tt := range tests {
		for compress, want := range map[bool]writeProto3MsgTestResult{true: tt.wantCompress, false: tt.wantUncompress} {
			s := newGenState()
			gotPkg, gotMsg, gotImports, errs := writeProto3Msg(tt.inMsg, tt.inMsgs, s, compress)
			if (len(errs) > 0) != want.err {
				t.Errorf("%s: writeProto3Msg(%v, %v, %v, %v): did not get expected error return status, got: %v, wanted error: %v", tt.name, tt.inMsg, tt.inMsgs, s, compress, errs, want.err)
			}

			if len(errs) > 0 {
				continue
			}

			if gotPkg != want.pkg {
				t.Errorf("%s: writeProto3Msg(%v, %v, %v, %v): did not get expected package name, got: %v, want: %v", tt.name, tt.inMsg, tt.inMsgs, s, compress, gotPkg, want.pkg)
			}

			if reflect.DeepEqual(gotImports, want.imports) {
				t.Errorf("%s: writeProto3Msg(%v, %v, 5v, %v): did not get expected set of imports, got: %v, want: %v", tt.name, tt.inMsg, tt.inMsgs, s, compress, gotImports, want.imports)
			}

			if diff := pretty.Compare(gotMsg, want.msg); diff != "" {
				if diffl, err := generateUnifiedDiff(gotMsg, want.msg); err == nil {
					diff = diffl
				}
				t.Errorf("%s: writeProto3Msg(%v, %v, %v, %v): did not get expected message returned, diff(-got,+want):\n%s", tt.name, tt.inMsg, tt.inMsgs, s, compress, diff)
			}
		}
	}
}
