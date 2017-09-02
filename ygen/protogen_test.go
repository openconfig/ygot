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
		name                string
		inMsg               *yangDirectory
		inMsgs              map[string]*yangDirectory
		inUniqueStructNames map[string]string
		wantMsg             protoMsg
		wantErr             bool
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
		name: "simple message with leaf-list and a message child",
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
		inUniqueStructNames: map[string]string{
			"/root/a-message/container-child": "ContainerChild",
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
				Type: "ContainerChild",
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
		s.uniqueStructNames = tt.inUniqueStructNames

		got, errs := genProtoMsg(tt.inMsg, tt.inMsgs, s)
		if (len(errs) > 0) != tt.wantErr {
			t.Errorf("%s: genProtoMsg(%#v, %#v, *genState): did not get expected error status, got: %v, wanted err: %v", tt.name, tt.inMsg, tt.inMsgs, errs, tt.wantErr)
		}

		if tt.wantErr {
			continue
		}

		if !protoMsgEq(got, tt.wantMsg) {
			diff := pretty.Compare(got, tt.wantMsg)
			t.Errorf("%s: genProtoMsg(%#v, %#v, *genState): did not get expected protobuf message definition, diff(-got,+want):\n%s", tt.name, tt.inMsg, tt.inMsgs, diff)
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
