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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
)

var validListSchema = &yang.Entry{
	Name:     "valid-list-schema",
	Kind:     yang.DirectoryEntry,
	ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
	Key:      "key_field_name",
	Config:   yang.TSTrue,
	Dir: map[string]*yang.Entry{
		"key_field_name": {
			Kind: yang.LeafEntry,
			Name: "key_field_name",
			Type: &yang.YangType{Kind: yang.Ystring},
		},
	},
}

func TestValidateListSchema(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validListSchema,
		},
		{
			desc:    "nil schema",
			schema:  nil,
			wantErr: true,
		},
		{
			desc:    "bad schema type",
			schema:  &yang.Entry{Name: "nil-type-schema", Kind: yang.LeafEntry},
			wantErr: true,
		},
		{
			desc: "missing dir",
			schema: &yang.Entry{
				Name:   "missing-dir-schema",
				Kind:   yang.DirectoryEntry,
				Key:    "key_field_name",
				Config: yang.TSTrue,
			},
			wantErr: true,
		},
		{
			desc: "missing key field",
			schema: &yang.Entry{
				Name:     "missing-key-field-schema",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key_field_name": {
						Kind: yang.LeafEntry,
						Name: "key_field_name",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
				},
			},
			wantErr: true,
		},
		{
			desc: "missing key leaf",
			schema: &yang.Entry{
				Name:     "missing-key-leaf-schema",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key_field_name",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"other_name": {
						Kind: yang.LeafEntry,
						Name: "other_name",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateListSchema(tt.schema)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateListSchema(%v) got error: %v, want error? %v", tt.desc, tt.schema, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateList(t *testing.T) {
	// nil value
	if got := validateList(nil, nil); got != nil {
		t.Errorf("nil value: Unmarshal got error: %v, want error: nil", got)
	}

	// nil schema
	err := util.Errors(validateList(nil, &struct{}{})).Error()
	wantErr := `list schema is nil`
	if got, want := err, wantErr; got != want {
		t.Errorf("nil schema: Unmarshal got error: %v, want error: %v", got, want)
	}

	// bad value type
	err = util.Errors(validateList(validListSchema, struct{}{})).Error()
	wantErr = `validateList expected map/slice type for valid-list-schema, got struct {}`
	if got, want := err, wantErr; got != want {
		t.Errorf("nil schema: Unmarshal got error: %v, want error: %v", got, want)
	}
}

func TestValidateListNoKey(t *testing.T) {
	listSchema := &yang.Entry{
		Name:     "list-schema",
		Kind:     yang.DirectoryEntry,
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
		Dir: map[string]*yang.Entry{
			"leaf-name": {
				Kind: yang.LeafEntry,
				Name: "LeafName",
				Type: &yang.YangType{Kind: yang.Ystring},
			},
		},
	}

	type StringListElemStruct struct {
		LeafName   *string `path:"leaf-name"`
		Annotation *string `ygotAnnotation:"true"`
	}
	type BadElemStruct struct {
		UnknownName *string `path:"unknown-name"`
	}

	tests := []struct {
		desc    string
		schema  *yang.Entry
		val     interface{}
		wantErr string
	}{
		{
			desc:   "success with nil value",
			schema: listSchema,
			val:    nil,
		},
		{
			desc:   "success",
			schema: listSchema,
			val:    []*StringListElemStruct{{LeafName: ygot.String("elem1_leaf_name")}},
		},
		{
			desc:   "success with list element",
			schema: listSchema,
			val:    &StringListElemStruct{LeafName: ygot.String("elem1_leaf_name")},
		},
		{
			desc:    "nil schema",
			schema:  nil,
			val:     1,
			wantErr: `nil schema for type int, value 1`,
		},
		{
			desc:    "bad field",
			schema:  listSchema,
			val:     []*BadElemStruct{{UnknownName: ygot.String("elem1_leaf_name")}},
			wantErr: `child schema not found for struct list-schema field UnknownName`,
		},
		{
			desc:   "failure with list element",
			schema: listSchema,
			val:    &StringListElemStruct{LeafName: ygot.String("elem1_leaf_name")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			errs := Validate(tt.schema, tt.val)
			if got, want := errs.String(), tt.wantErr; got != want {
				t.Errorf("%s: Validate got error: %v, want error: %v", tt.desc, got, want)
			}
			testErrLog(t, tt.desc, errs)
		})
	}
}

func TestValidateListSimpleKey(t *testing.T) {
	listSchema := &yang.Entry{
		Name:     "list-schema",
		Kind:     yang.DirectoryEntry,
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
		Key:      "keyfield-name",
		Config:   yang.TSTrue,
		Dir: map[string]*yang.Entry{
			"keyfield-name": {
				Kind: yang.LeafEntry,
				Name: "keyfield-name",
				Type: &yang.YangType{Kind: yang.Ystring},
			},
			"leaf-name": {
				Kind: yang.LeafEntry,
				Name: "leaf-name",
				Type: &yang.YangType{Kind: yang.Ystring},
			},
		},
	}

	type StringListElemStruct struct {
		KeyFieldName *string `path:"keyfield-name"`
		LeafName     *string `path:"leaf-name"`
		Annotation   *string `ygotAnnotation:"true"`
	}
	type BadElemStruct struct {
		LeafName *string
	}

	tests := []struct {
		desc    string
		val     interface{}
		wantErr bool
	}{
		{
			desc: "success",
			val: map[string]*StringListElemStruct{
				"elem1_key_val": {
					KeyFieldName: ygot.String("elem1_key_val"),
					LeafName:     ygot.String("elem1_leaf_name"),
				},
			},
		},
		{
			desc: "missing key",
			val: map[string]*BadElemStruct{
				"elem1": {
					LeafName: ygot.String("elem1_leaf_name"),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			errs := Validate(listSchema, tt.val)
			if got, want := (errs != nil), tt.wantErr; got != want {
				t.Errorf("%s: b.Validate(%v) got error: %v, want error? %v", tt.desc, tt.val, errs, tt.wantErr)
			}
			testErrLog(t, tt.desc, errs)
		})
	}
}

func TestValidateListStructKey(t *testing.T) {
	listSchemaStructKey := &yang.Entry{
		Name:     "list-schema-struct-key",
		Kind:     yang.DirectoryEntry,
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
		Key:      "Key1 Key2",
		Config:   yang.TSTrue,
		Dir: map[string]*yang.Entry{
			"key1": {
				Kind: yang.LeafEntry,
				Name: "Key1",
				Type: &yang.YangType{Kind: yang.Ystring},
			},
			"key2": {
				Kind: yang.LeafEntry,
				Name: "Key2",
				Type: &yang.YangType{Kind: yang.Yint32},
			},
			"leaf-name": {
				Kind: yang.LeafEntry,
				Name: "LeafName",
				Type: &yang.YangType{Kind: yang.Ystring},
			},
		},
	}

	type KeyStruct struct {
		Key1 string
		Key2 int32
	}
	type StringListElemStruct struct {
		Key1     *string `path:"key1"`
		Key2     *int32  `path:"key2"`
		LeafName *string `path:"leaf-name"`
	}
	type BadElemStruct1 struct {
		Key1     *string `path:"key1"`
		LeafName *string `path:"leaf-name"`
	}
	type BadElemStruct2 struct {
		Key1       *string `path:"key1"`
		Key2       *int32  `path:"key2"`
		ExtraField *string `path:"extra-name"`
		LeafName   *string `path:"leaf-name"`
	}

	tests := []struct {
		desc    string
		val     interface{}
		wantErr bool
	}{
		{
			desc: "success",
			val: map[KeyStruct]*StringListElemStruct{
				{"elem1_key_val", 1}: {
					Key1:     ygot.String("elem1_key_val"),
					Key2:     ygot.Int32(1),
					LeafName: ygot.String("elem1_leaf_name"),
				},
			},
		},
		{
			desc: "bad key value",
			val: map[KeyStruct]*StringListElemStruct{
				{"elem1_key_val", 1}: {
					Key1:     ygot.String("elem1_key_val"),
					Key2:     ygot.Int32(2),
					LeafName: ygot.String("elem1_leaf_name"),
				},
			},
			wantErr: true,
		},
		{
			desc: "missing key",
			val: map[KeyStruct]*BadElemStruct1{
				{"elem1_key_val", 0}: {
					Key1:     ygot.String("elem1_key_val"),
					LeafName: ygot.String("elem1_leaf_name"),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			errs := Validate(listSchemaStructKey, tt.val)
			if got, want := (errs != nil), tt.wantErr; got != want {
				t.Errorf("%s: b.Validate(%v) got error: %v, want error? %v", tt.desc, tt.val, errs, tt.wantErr)
			}
			testErrLog(t, tt.desc, errs)
		})
	}
}

func TestUnmarshalList(t *testing.T) {
	// nil value
	if got := unmarshalList(nil, nil, nil); got != nil {
		t.Errorf("nil value: Unmarshal got error: %v, want error: nil", got)
	}

	// nil schema
	wantErr := `list schema is nil`
	if got, want := errToString(unmarshalList(nil, nil, []struct{}{})), wantErr; got != want {
		t.Errorf("nil schema: Unmarshal got error: %v, want error: %v", got, want)
	}

	// bad parent type
	wantErr = `unmarshalList for valid-list-schema got parent type struct, expect map, slice ptr or struct ptr`
	if got, want := errToString(unmarshalList(validListSchema, struct{}{}, []interface{}{})), wantErr; got != want {
		t.Errorf("nil schema: Unmarshal got error: %v, want error: %v", got, want)
	}

	// bad value type
	wantErr = `unmarshalContainer for schema valid-list-schema: jsonTree 42 (int): got type int inside container, expect map[string]interface{}`
	if got, want := errToString(unmarshalList(validListSchema, &struct{}{}, int(42))), wantErr; got != want {
		t.Errorf("nil schema: Unmarshal got error: %v, want error: %v", got, want)
	}

	// bad parent type for unmarshalContainerWithListSchema
	wantErr = `unmarshalContainerWithListSchema value [], type []interface {}, into parent type struct {}, schema name valid-list-schema: parent must be a struct ptr`
	if got, want := errToString(unmarshalContainerWithListSchema(validListSchema, struct{}{}, []interface{}{})), wantErr; got != want {
		t.Errorf("nil schema: Unmarshal got error: %v, want error: %v", got, want)
	}
}

func TestUnmarshalUnkeyedList(t *testing.T) {
	containerWithLeafListSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"struct-list": {
				Name:     "struct-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Dir: map[string]*yang.Entry{
					"leaf-field": {
						Kind: yang.LeafEntry,
						Name: "leaf-field",
						Type: &yang.YangType{Kind: yang.Yint32},
					},
					"enum-leaf-field": {
						Kind: yang.LeafEntry,
						Name: "enum-leaf-field",
						Type: &yang.YangType{Kind: yang.Yenum},
					},
				},
			},
		},
	}

	type ListElemStruct struct {
		LeafName *int32   `path:"leaf-field"`
		EnumLeaf EnumType `path:"enum-leaf-field"`
	}
	type ContainerStruct struct {
		StructList []*ListElemStruct `path:"struct-list"`
	}

	tests := []struct {
		desc    string
		schema  *yang.Entry
		json    string
		want    ContainerStruct
		wantErr string
	}{
		{
			desc:   "success with nil value",
			schema: containerWithLeafListSchema,
			json:   ``,
			want:   ContainerStruct{},
		},
		{
			desc:   "success",
			schema: containerWithLeafListSchema,
			json:   `{"struct-list" : [ { "leaf-field" : 42, "enum-leaf-field" : "E_VALUE_FORTY_TWO"} ] }`,
			want: ContainerStruct{
				StructList: []*ListElemStruct{
					{
						LeafName: ygot.Int32(42),
						EnumLeaf: 42,
					},
				},
			},
		},
		{
			desc:    "nil schema error",
			schema:  nil,
			json:    `{}`,
			want:    ContainerStruct{},
			wantErr: `nil schema for parent type *ytypes.ContainerStruct, value map[] (map[string]interface {})`,
		},
		{
			desc:    "bad value type",
			schema:  containerWithLeafListSchema,
			json:    `{"struct-list" : { "leaf-field" : 42 } }`,
			wantErr: `unmarshalList for schema struct-list: jsonList map[leaf-field:42] (map): got type map[string]interface {}, expect []interface{}`,
		},
	}

	var jsonTree interface{}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var parent ContainerStruct

			if tt.json != "" {
				if err := json.Unmarshal([]byte(tt.json), &jsonTree); err != nil {
					t.Fatal(fmt.Sprintf("%s : %s", tt.desc, err))
				}
			}

			err := Unmarshal(tt.schema, &parent, jsonTree)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: Unmarshal got error: %v, want error: %v", tt.desc, got, want)
			}
			testErrLog(t, tt.desc, err)
			if err == nil {
				if got, want := parent, tt.want; !reflect.DeepEqual(got, want) {
					t.Errorf("%s: Unmarshal got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
				}
			}
		})
	}
}

func TestUnmarshalKeyedList(t *testing.T) {
	containerWithLeafListSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"key-list": {
				Name:     "key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key": {
						Kind: yang.LeafEntry,
						Name: "key",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
					"leaf-field": {
						Kind: yang.LeafEntry,
						Name: "leaf-field",
						Type: &yang.YangType{Kind: yang.Yint32},
					},
				},
			},
		},
	}

	type ListElemStruct struct {
		Key       *string `path:"key"`
		LeafField *int32  `path:"leaf-field"`
	}
	type ContainerStruct struct {
		KeyList map[string]*ListElemStruct `path:"key-list"`
	}

	tests := []struct {
		desc    string
		json    string
		want    ContainerStruct
		opts    []UnmarshalOpt
		wantErr string
	}{
		{
			desc: "success",
			json: `{ "key-list" : [ { "key" : "forty-two", "leaf-field" : 42} ] }`,
			want: ContainerStruct{
				KeyList: map[string]*ListElemStruct{
					"forty-two": {
						Key:       ygot.String("forty-two"),
						LeafField: ygot.Int32(42),
					},
				},
			},
		},
		{
			desc:    "bad field",
			json:    `{ "key-list" : [ { "key" : "forty-two", "bad-field" : 42} ] }`,
			wantErr: `parent container key-list (type *ytypes.ListElemStruct): JSON contains unexpected field bad-field`,
		},
		{
			desc: "ignore unknown field",
			json: `{ "key-list" : [ { "key" : "forty-two", "bad-field" : 42} ] }`,
			opts: []UnmarshalOpt{&IgnoreExtraFields{}},
			want: ContainerStruct{
				KeyList: map[string]*ListElemStruct{
					"forty-two": {
						Key: ygot.String("forty-two"),
					},
				},
			},
		},
	}

	var jsonTree interface{}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var parent ContainerStruct

			if err := json.Unmarshal([]byte(tt.json), &jsonTree); err != nil {
				t.Fatal(fmt.Sprintf("%s : %s", tt.desc, err))
			}

			err := Unmarshal(containerWithLeafListSchema, &parent, jsonTree, tt.opts...)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: Unmarshal got error: %v, want error: %v", tt.desc, got, want)
			}
			testErrLog(t, tt.desc, err)
			if err == nil {
				if got, want := parent, tt.want; !reflect.DeepEqual(got, want) {
					t.Errorf("%s: Unmarshal got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
				}
			}
		})
	}
}

func TestUnmarshalStructKeyedList(t *testing.T) {
	containerWithLeafListSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"struct-key-list": {
				Name:     "struct-key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key1 key2 key3",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key1": {
						Kind: yang.LeafEntry,
						Name: "key1",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
					"key2": {
						Kind: yang.LeafEntry,
						Name: "key2",
						Type: &yang.YangType{Kind: yang.Yint32},
					},
					"key3": {
						Kind: yang.LeafEntry,
						Name: "key3",
						Type: &yang.YangType{Kind: yang.Yenum},
					},
					"leaf-field": {
						Kind: yang.LeafEntry,
						Name: "leaf-field",
						Type: &yang.YangType{Kind: yang.Yint32},
					},
				},
			},
		},
	}

	type KeyStruct struct {
		Key1    string
		Key2    int32
		EnumKey EnumType
	}
	type ListElemStruct struct {
		Key1     *string  `path:"key1"`
		Key2     *int32   `path:"key2"`
		EnumKey  EnumType `path:"key3"`
		LeafName *int32   `path:"leaf-field"`
	}
	type ContainerStruct struct {
		StructKeyList map[KeyStruct]*ListElemStruct `path:"struct-key-list"`
	}

	tests := []struct {
		desc    string
		json    string
		want    ContainerStruct
		wantErr string
	}{
		{
			desc: "success",
			json: `{ "struct-key-list" : [ { "key1" : "forty-two", "key2" : 42, "key3" : "E_VALUE_FORTY_TWO", "leaf-field" : 43} ] }`,
			want: ContainerStruct{
				StructKeyList: map[KeyStruct]*ListElemStruct{
					{"forty-two", 42, 42}: {
						Key1:     ygot.String("forty-two"),
						Key2:     ygot.Int32(42),
						EnumKey:  42,
						LeafName: ygot.Int32(43),
					},
				},
			},
		},
	}

	var jsonTree interface{}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var parent ContainerStruct

			if err := json.Unmarshal([]byte(tt.json), &jsonTree); err != nil {
				t.Fatal(fmt.Sprintf("%s : %s", tt.desc, err))
			}

			err := Unmarshal(containerWithLeafListSchema, &parent, jsonTree)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: Unmarshal got error: %v, want error: %v", tt.desc, got, want)
			}
			testErrLog(t, tt.desc, err)
			if err == nil {
				if got, want := parent, tt.want; !reflect.DeepEqual(got, want) {
					t.Errorf("%s: Unmarshal got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
				}
			}
		})
	}
}

func TestUnmarshalSingleListElement(t *testing.T) {
	listSchema := &yang.Entry{
		Name:     "struct-list",
		Kind:     yang.DirectoryEntry,
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
		Dir: map[string]*yang.Entry{
			"leaf-field": {
				Kind: yang.LeafEntry,
				Name: "leaf-field",
				Type: &yang.YangType{Kind: yang.Yint32},
			},
			"enum-leaf-field": {
				Kind: yang.LeafEntry,
				Name: "enum-leaf-field",
				Type: &yang.YangType{Kind: yang.Yenum},
			},
		},
	}

	type ListElemStruct struct {
		LeafName *int32   `path:"leaf-field"`
		EnumLeaf EnumType `path:"enum-leaf-field"`
	}

	tests := []struct {
		desc    string
		json    string
		want    ListElemStruct
		wantErr string
	}{
		{
			desc: "success",
			json: `{ "leaf-field" : 42, "enum-leaf-field" : "E_VALUE_FORTY_TWO"}`,
			want: ListElemStruct{
				LeafName: ygot.Int32(42),
				EnumLeaf: 42,
			},
		},
		{
			desc:    "bad field",
			json:    `{ "leaf-field" : 42, "bad-field" : "E_VALUE_FORTY_TWO"}`,
			wantErr: `parent container struct-list (type *ytypes.ListElemStruct): JSON contains unexpected field bad-field`,
		},
	}

	var jsonTree interface{}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var parent ListElemStruct

			if err := json.Unmarshal([]byte(tt.json), &jsonTree); err != nil {
				t.Fatal(fmt.Sprintf("%s : %s", tt.desc, err))
			}

			err := Unmarshal(listSchema, &parent, jsonTree)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: Unmarshal got error: %v, want error: %v", tt.desc, got, want)
			}
			testErrLog(t, tt.desc, err)
			if err == nil {
				if got, want := parent, tt.want; !reflect.DeepEqual(got, want) {
					t.Errorf("%s: Unmarshal got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
				}
			}
		})
	}
}

func TestStructMapKeyValueCreation(t *testing.T) {
	containerWithLeafListSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"struct-key-list": {
				Name:     "struct-key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key1 key2 key3",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key1": {
						Kind: yang.LeafEntry,
						Name: "key1",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
					"key2": {
						Kind: yang.LeafEntry,
						Name: "key2",
						Type: &yang.YangType{Kind: yang.Yint32},
					},
					"key3": {
						Kind: yang.LeafEntry,
						Name: "key3",
						Type: &yang.YangType{Kind: yang.Yenum},
					},
					"leaf-field": {
						Kind: yang.LeafEntry,
						Name: "leaf-field",
						Type: &yang.YangType{Kind: yang.Yint32},
					},
				},
			},
		},
	}

	type KeyStruct struct {
		Key1    string   `path:"key1"`
		Key2    int32    `path:"key2"`
		EnumKey EnumType `path:"key3"`
	}

	type ListElemStruct struct {
		Key1     *string  `path:"key1"`
		Key2     *int32   `path:"key2"`
		EnumKey  EnumType `path:"key3"`
		LeafName *int32   `path:"leaf-field"`
	}

	type ContainerStruct struct {
		StructKeyList map[KeyStruct]*ListElemStruct `path:"struct-key-list"`
	}

	tests := []struct {
		desc         string
		keys         map[string]string
		want         KeyStruct
		errSubstring string
	}{
		{
			desc: "success",
			keys: map[string]string{"key1": "int0", "key2": "42", "key3": "E_VALUE_FORTY_TWO"},
			want: KeyStruct{Key1: "int0", Key2: 42, EnumKey: 42},
		},
		// note that an extra key in the map is just ignored as long as the mandatory keys present.
		{
			desc:         "not existing key",
			keys:         map[string]string{"key4": "int0", "key2": "42", "key3": "E_VALUE_FORTY_TWO"},
			errSubstring: "missing key1",
		},
		{
			desc:         "overflowing key",
			keys:         map[string]string{"key1": "int0", "key2": "14294967296", "key3": "E_VALUE_FORTY_TWO"},
			errSubstring: "unable to convert",
		},
		{
			desc:         "upper case key",
			keys:         map[string]string{"Key1": "int0", "key2": "14294967296", "key3": "E_VALUE_FORTY_TWO"},
			errSubstring: "missing key1",
		},
		{
			desc:         "missing key",
			keys:         map[string]string{"key2": "42", "key3": "E_VALUE_FORTY_TWO"},
			errSubstring: "missing key1",
		},
		{
			desc:         "incorrect type for key2",
			keys:         map[string]string{"key1": "int0", "key2": "forty_two", "key3": "E_VALUE_FORTY_TWO"},
			errSubstring: "unable to convert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			parent := &ContainerStruct{}
			util.InitializeStructField(parent, "StructKeyList")
			v, e := makeValForInsert(containerWithLeafListSchema, parent.StructKeyList, tt.keys)
			if diff := errdiff.Substring(e, tt.errSubstring); diff != "" {
				t.Fatalf("got %v, want error %v", e, tt.errSubstring)
			}
			if e != nil {
				return
			}
			k, e := makeKeyForInsert(containerWithLeafListSchema, parent.StructKeyList, v)
			if diff := errdiff.Substring(e, tt.errSubstring); diff != "" {
				t.Fatalf("got %v, want error %v", e, tt.errSubstring)
			}
			if e != nil {
				return
			}
			if diff := cmp.Diff(k.Interface(), tt.want); diff != "" {
				t.Errorf("got %v, want %v: diff %v", k, tt.want, diff)
			}
		})
	}
}

type simpleStruct struct {
	KeyList interface{} `path:"key-list"`
}

type ListUintStruct struct {
	Key *uint32 `path:"key"`
}

type ListStringStruct struct {
	Key *string `path:"key"`
}

func (l *ListUintStruct) String() string {
	return fmt.Sprintf("Key: %d", *l.Key)
}

func TestSimpleMapKeyValueCreation(t *testing.T) {
	tests := []struct {
		desc         string
		keys         map[string]string
		inSchema     *yang.Entry
		container    *simpleStruct
		want         interface{}
		errSubstring string
	}{
		{
			desc: "success - uint32 <key,value> creation",
			keys: map[string]string{"key": "42"},
			inSchema: &yang.Entry{
				Name:     "key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key": {
						Kind: yang.LeafEntry,
						Name: "key",
						Type: &yang.YangType{Kind: yang.Yuint32},
					},
				},
			},
			container: &simpleStruct{KeyList: map[uint32]*ListUintStruct{}},
			want:      uint32(42),
		},
		{
			desc: "incorrect type - uint32 <key,value> creation",
			keys: map[string]string{"key": "-42"},
			inSchema: &yang.Entry{
				Name:     "key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key": {
						Kind: yang.LeafEntry,
						Name: "key",
						Type: &yang.YangType{Kind: yang.Yuint32},
					},
				},
			},
			container:    &simpleStruct{KeyList: map[uint32]*ListUintStruct{}},
			errSubstring: "unable to convert",
		},
		{
			desc: "overflowing type - uint32 <key,value> creation",
			keys: map[string]string{"key": "14294967296"},
			inSchema: &yang.Entry{
				Name:     "key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key": {
						Kind: yang.LeafEntry,
						Name: "key",
						Type: &yang.YangType{Kind: yang.Yuint32},
					},
				},
			},
			container:    &simpleStruct{KeyList: map[uint32]*ListUintStruct{}},
			errSubstring: "unable to convert",
		},
		{
			desc: "incorrect type - uint32 <key,value> creation",
			keys: map[string]string{"key": "test"},
			inSchema: &yang.Entry{
				Name:     "key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key": {
						Kind: yang.LeafEntry,
						Name: "key",
						Type: &yang.YangType{Kind: yang.Yuint32},
					},
				},
			},
			container:    &simpleStruct{KeyList: map[uint32]*ListUintStruct{}},
			errSubstring: "unable to convert",
		},
		{
			desc: "success - string <key,value> creation",
			keys: map[string]string{"key": "test0"},
			inSchema: &yang.Entry{
				Name:     "key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key": {
						Kind: yang.LeafEntry,
						Name: "key",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
					"leaf-field": {
						Kind: yang.LeafEntry,
						Name: "leaf-field",
						Type: &yang.YangType{Kind: yang.Yint32},
					},
				},
			},
			container: &simpleStruct{KeyList: map[string]*ListStringStruct{}},
			want:      "test0",
		},
		{
			desc: "missing key - string <key,value> creation",
			keys: map[string]string{"missing_key": "test0"},
			inSchema: &yang.Entry{
				Name:     "key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key": {
						Kind: yang.LeafEntry,
						Name: "key",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
					"leaf-field": {
						Kind: yang.LeafEntry,
						Name: "leaf-field",
						Type: &yang.YangType{Kind: yang.Yint32},
					},
				},
			},
			container:    &simpleStruct{KeyList: map[string]*ListStringStruct{}},
			errSubstring: "missing key",
		},
		{
			desc:         "parent is not reflect.Map kind",
			container:    &simpleStruct{KeyList: int32(42)},
			errSubstring: "int32 is not a reflect.Map kind",
		},
		{
			desc:         "map value is not pointer type",
			container:    &simpleStruct{KeyList: map[string]string{}},
			errSubstring: "string is not a pointer to a struct",
		},
		{
			desc: "fail map value doesn't have the key with the tag specified in path",
			keys: map[string]string{"missing-key": "42"},
			inSchema: &yang.Entry{
				Name:     "key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "missing-key",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"missing-key": {
						Kind: yang.LeafEntry,
						Name: "missing-key",
						Type: &yang.YangType{Kind: yang.Yuint32},
					},
				},
			},
			container:    &simpleStruct{KeyList: map[uint32]*ListUintStruct{}},
			errSubstring: "does not contain a field with tag missing-key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			util.InitializeStructField(tt.container, "KeyList")
			v, e := makeValForInsert(tt.inSchema, tt.container.KeyList, tt.keys)
			if diff := errdiff.Substring(e, tt.errSubstring); diff != "" {
				t.Fatalf("got %v, want error %v", e, tt.errSubstring)
			}
			if e != nil {
				return
			}
			k, e := makeKeyForInsert(tt.inSchema, tt.container.KeyList, v)
			if diff := errdiff.Substring(e, tt.errSubstring); diff != "" {
				t.Fatalf("got %v, want error %v", e, tt.errSubstring)
			}
			if e != nil {
				return
			}
			if k.Interface() != tt.want {
				t.Errorf("got %v, want %v", k.Interface(), tt.want)
			}
		})
	}
}

func TestInsertAndGetKey(t *testing.T) {
	type KeyStruct struct {
		Key1    int32    `path:"key1"` // Key1 type doesn't match with the type of Key1 in ListElemStruct
		Key2    int32    `path:"key2"`
		EnumKey EnumType `path:"key3"`
	}

	type ListElemStruct struct {
		Key1    *string  `path:"key1"`
		Key2    *int32   `path:"key2"`
		EnumKey EnumType `path:"key3"`
	}

	tests := []struct {
		inDesc           string
		inSchema         *yang.Entry
		inParent         interface{}
		inKeys           map[string]string
		want             interface{}
		wantErrSubstring string
	}{
		{
			inDesc: "success creating key and value for uint32 key type",
			inSchema: &yang.Entry{
				Name:     "key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key": {
						Kind: yang.LeafEntry,
						Name: "key",
						Type: &yang.YangType{Kind: yang.Yuint32},
					},
				},
			},
			inParent: map[uint32]*ListUintStruct{},
			inKeys:   map[string]string{"key": "42"},
			want:     &ListUintStruct{Key: ygot.Uint32(42)},
		},
		{
			inDesc: "fail missing key in the schema",
			inSchema: &yang.Entry{
				Name:     "key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Config:   yang.TSTrue,
				Dir:      map[string]*yang.Entry{},
			},
			wantErrSubstring: "unkeyed list can't be traversed",
		},
		{
			inDesc: "fail non-map root",
			inSchema: &yang.Entry{
				Name:     "key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key": {
						Kind: yang.LeafEntry,
						Name: "key",
						Type: &yang.YangType{Kind: yang.Yuint32},
					},
				},
			},
			inParent:         []*ListUintStruct{},
			wantErrSubstring: "root has type []*ytypes.ListUintStruct, want map",
		},
		{
			inDesc: "fail missing key in keys map",
			inSchema: &yang.Entry{
				Name:     "key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "missing-key",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"missing-key": {
						Kind: yang.LeafEntry,
						Name: "missing-key",
						Type: &yang.YangType{Kind: yang.Yuint32},
					},
				},
			},
			inParent:         map[uint32]*ListUintStruct{},
			wantErrSubstring: "missing missing-key key in map[]",
		},
		{
			inDesc: "fail creating key due to not matching type",
			inSchema: &yang.Entry{
				Name:     "key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key": {
						Kind: yang.LeafEntry,
						Name: "key",
						Type: &yang.YangType{Kind: yang.Yuint32},
					},
				},
			},
			inParent:         map[string]*ListUintStruct{},
			inKeys:           map[string]string{"key": "42"},
			wantErrSubstring: "uint32 is not assignable to string",
		},
		{
			inDesc: "fail creating key due to not maching key type - struct key",
			inSchema: &yang.Entry{
				Name:     "struct-key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key1 key2 key3",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key1": {
						Kind: yang.LeafEntry,
						Name: "key1",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
					"key2": {
						Kind: yang.LeafEntry,
						Name: "key2",
						Type: &yang.YangType{Kind: yang.Yint32},
					},
					"key3": {
						Kind: yang.LeafEntry,
						Name: "key3",
						Type: &yang.YangType{Kind: yang.Yenum},
					},
				},
			},
			inParent:         map[KeyStruct]*ListElemStruct{},
			inKeys:           map[string]string{"key1": "42", "key2": "42", "key3": "E_VALUE_FORTY_TWO"},
			wantErrSubstring: "string is not assignable to int32",
		},
	}

	for _, tt := range tests {
		t.Run(tt.inDesc, func(t *testing.T) {
			got, err := insertAndGetKey(tt.inSchema, tt.inParent, tt.inKeys)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("got %v, want error %v", err, tt.wantErrSubstring)
			}
			if err != nil {
				return
			}
			val := reflect.ValueOf(tt.inParent).MapIndex(reflect.ValueOf(got)).Interface()
			if !reflect.DeepEqual(val, tt.want) {
				t.Errorf("got %v, want %v", val, tt.want)
			}
		})
	}
}
