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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/gnmi/errdiff"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
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

type annotationParent struct {
	AnnotationField []ygot.Annotation `path:"@annotation" ygotAnnotation:"true"`
}

func (*annotationParent) IsYANGGoStruct() {}

type exAnnotation struct {
	Field1 string `json:"field1"`
}

func (a *exAnnotation) MarshalJSON() ([]byte, error) {
	return json.Marshal(*a)
}

func (a *exAnnotation) FromJSON(data []byte) error {

	return json.Unmarshal(data, a)
	// json.Unmarshal(data, a)
	/*n := &exAnnotation{}
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	a = n
	return nil*/
}

type pathWrapper struct {
	*gpb.Path
}

func (p pathWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Path)
}

func (p pathWrapper) FromJSON(data []byte) error {
	return json.Unmarshal(data, p)
}

func TestJSONUnmarshal(t *testing.T) {
	n := &exAnnotation{}
	in := []byte(`
	{
		"field1": "a value"
	}`)
	if err := json.Unmarshal(in, n); err != nil {
		t.Fatalf("can't unmarshal %v", err)
	}
}

func TestUnmarshalAnnotation(t *testing.T) {
	tests := []struct {
		name             string
		inParent         reflect.Value
		inFieldName      string
		inContents       interface{}
		inOpts           []UnmarshalOpt
		wantErrSubstring string
		wantFieldValue   []ygot.Annotation
	}{{
		name:        "simple annotation type",
		inParent:    reflect.ValueOf(&annotationParent{}),
		inFieldName: "AnnotationField",
		inContents: []byte(`
			{
				"field1": "a value"
			}
		`),
		inOpts: []UnmarshalOpt{
			&AnnotationTypes{Types: []reflect.Type{
				reflect.TypeOf(&exAnnotation{}),
			}},
		},
		wantFieldValue: []ygot.Annotation{
			&exAnnotation{
				Field1: "a value",
			},
		},
	}, {
		name:        "protobuf annotation",
		inParent:    reflect.ValueOf(&annotationParent{}),
		inFieldName: "AnnotationField",
		inContents: []byte(`{
			"elem": [
			  {
				"name": "interfaces"
			  },
			  {
				"name": "interface",
				"key": {
				  "name": "eth0"
				}
			  }
			]
		  }`),
		inOpts: []UnmarshalOpt{
			&AnnotationTypes{Types: []reflect.Type{
				reflect.TypeOf(pathWrapper{}),
			}},
		},
		wantFieldValue: []ygot.Annotation{
			pathWrapper{Path: mustPath("/interfaces/interface[name=eth0]")},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			fmt.Printf("parent is %s\n", tt.inParent.Kind())
			fmt.Printf("parent element is %s\n", tt.inParent.Elem().Kind())
			err := unmarshalAnnotation(tt.inParent, tt.inFieldName, tt.inContents, tt.inOpts...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}

			gotField := tt.inParent.Elem().FieldByName(tt.inFieldName).Interface()

			gs, ok := tt.inParent.Interface().(ygot.GoStruct)
			if !ok {
				panic(err)
			}
			j, err := ygot.ConstructIETFJSON(gs, nil)
			if err != nil {
				t.Fatalf("can't marshal to JSON: %v\n", err)
			}
			js, err := json.MarshalIndent(j, "", "  ")
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", js)
			if diff := cmp.Diff(gotField, tt.wantFieldValue, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("did not get expected value, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
