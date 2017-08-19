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
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
)

func TestGenProtoMsg(t *testing.T) {
	tests := []struct {
		name                string
		inMsg               *yangStruct
		inMsgs              map[string]*yangStruct
		inUniqueStructNames map[string]string
		wantMsg             protoMsg
		wantErr             bool
	}{{
		name: "simple message with only scalar fields",
		inMsg: &yangStruct{
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
		inMsg: &yangStruct{
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
		inMsg: &yangStruct{
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

		if diff := pretty.Compare(got, tt.wantMsg); diff != "" {
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
		name: "contains forward slash",
		in:   "with/forwardslash",
		want: "with_forwardslash",
	}}

	for _, tt := range tests {
		if got := safeProtoFieldName(tt.in); got != tt.want {
			t.Errorf("%s: safeProtoFieldName(%s): did not get expected name, got: %v, want: %v", tt.name, tt.in, got, tt.want)
		}
	}
}
