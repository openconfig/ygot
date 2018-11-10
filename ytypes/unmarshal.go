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
	"errors"
	"fmt"
	"reflect"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
)

// UnmarshalOpt is an interface used for any option to be supplied
// to the Unmarshal function. Types implementing it can be used to
// control the behaviour of JSON unmarshalling.
type UnmarshalOpt interface {
	IsUnmarshalOpt()
}

// IgnoreExtraFields is an unmarshal option that controls the
// behaviour of the Unmarshal function when additional fields are
// found in the input JSON. By default, an error will be returned,
// by specifying the IgnoreExtraFields option to Unmarshal, extra
// fields will be discarded.
type IgnoreExtraFields struct{}

// IsUnmarshalOpt marks IgnoreExtraFields as a valid UnmarshalOpt.
func (*IgnoreExtraFields) IsUnmarshalOpt() {}

// AnnotationTypes specifies the Go types that are used for
// annotations. Unmarshal will attempt to unmarshal each annotation into
// the specified types.
type AnnotationTypes struct {
	Types []reflect.Type
}

// IsUnmarshalOpt marks AnnotationTypes as a valid UnmarshalOpt.
func (*AnnotationTypes) IsUnmarshalOpt() {}

// Unmarshal recursively unmarshals JSON data tree in value into the given
// parent, using the given schema. Any values already in the parent that are
// not present in value are preserved. If provided schema is a leaf or leaf
// list, parent must be referencing the parent GoStruct.
func Unmarshal(schema *yang.Entry, parent interface{}, value interface{}, opts ...UnmarshalOpt) error {
	return unmarshalGeneric(schema, parent, value, JSONEncoding, opts...)
}

// Encoding specifies how the value provided to UnmarshalGeneric function is encoded.
type Encoding int

const (
	// JSONEncoding indicates that provided value is JSON encoded.
	JSONEncoding = iota

	// GNMIEncoding indicates that provided value is gNMI TypedValue.
	GNMIEncoding
)

// unmarshalGeneric unmarshals the provided value encoded with the given
// encoding type into the parent with the provided schema. When encoding mode
// is GNMIEncoding, the schema needs to be pointing to a leaf or leaf list
// schema.
func unmarshalGeneric(schema *yang.Entry, parent interface{}, value interface{}, enc Encoding, opts ...UnmarshalOpt) error {
	util.Indent()
	defer util.Dedent()

	// Nil value means the field is unset.
	if util.IsValueNil(value) {
		return nil
	}
	if schema == nil {
		return fmt.Errorf("nil schema for parent type %T, value %v (%T)", parent, value, value)
	}
	util.DbgPrint("Unmarshal value %v, type %T, into parent type %T, schema name %s", util.ValueStrDebug(value), value, parent, schema.Name)

	if enc == GNMIEncoding && !(schema.IsLeaf() || schema.IsLeafList()) {
		return errors.New("unmarshalling a non leaf node isn't supported in GNMIEncoding mode")
	}

	switch {
	case schema.IsLeaf():
		return unmarshalLeaf(schema, parent, value, enc)
	case schema.IsLeafList():
		return unmarshalLeafList(schema, parent, value, enc)
	case schema.IsList():
		return unmarshalList(schema, parent, value, enc, opts...)
	case schema.IsChoice():
		return fmt.Errorf("cannot pass choice schema %s to Unmarshal", schema.Name)
	case schema.IsContainer():
		return unmarshalContainer(schema, parent, value, enc, opts...)
	}
	return fmt.Errorf("unknown schema type for type %T, value %v", value, value)
}

// hasIgnoreExtraFields determines whether the supplied slice of UnmarshalOpts contains
// the IgnoreExtraFields option.
func hasIgnoreExtraFields(opts []UnmarshalOpt) bool {
	for _, o := range opts {
		if _, ok := o.(*IgnoreExtraFields); ok {
			return true
		}
	}
	return false
}

func annotationTypes(opts []UnmarshalOpt) *AnnotationTypes {
	for _, o := range opts {
		if v, ok := o.(*AnnotationTypes); ok {
			return v
		}
	}
	return nil
}

func unmarshalAnnotation(parent reflect.Value, fieldName string, annContents interface{}, opts ...UnmarshalOpt) error {
	at := annotationTypes(opts)
	if at == nil {
		// If no annotationTypes were supplied, we cannot unmarshal, so return nil.
		return nil
	}

	var errs []error
	addErr := func(err error) { errs = append(errs, err) }

	jsonC, ok := annContents.([]byte)
	if !ok {
		return fmt.Errorf("invalid type %T for annotation contents", annContents)
	}
	_, _ = addErr, jsonC

	for _, t := range at.Types {
		/*if !reflect.TypeOf((*ygot.Annotation)(nil)).Implements(t) {
			return fmt.Errorf("invalid annotation type %s supplied", t.Name())
		}*/

		//target := reflect.New(t).Elem()
		//fmt.Printf("target is %T\n", target.Interface())
		//unmarshalMethod := target.MethodByName("FromJSON")
		//meth := target
		//if !util.IsValueStructPtr(target) && util.IsValuePtr(target) {
		//	meth = target.Elem()
		//}
		/*unmarshalMethod := target.MethodByName("ToJSON")
		if !meth.IsValid() {
			return fmt.Errorf("annotation type %s does not have ToJSON method", t)
		}

		cr := unmarshalMethod.Call([]reflect.Value{reflect.ValueOf(annContents)})
		if len(cr) != 1 {
			return fmt.Errorf("method UnmarshalJSON for %s returns too many values %d", t.Name(), len(cr))
		}

		err, ok := cr[0].Interface().(error)
		if !ok {
			return fmt.Errorf("method UnmarshalJSON for %s does not return an error, but a %T", t.Name(), cr[0].Interface())
		}

		if err != nil {
			fmt.Printf("error %v\n", err)
			addErr(err)
			continue
		}*/

		target := reflect.New(t)
		fmt.Printf("target is %T %v\n", target.Interface(), target.Interface())
		if err := json.Unmarshal(jsonC, target.Interface()); err != nil {
			fmt.Printf("got %v\n", err)
			addErr(err)
			// We must ignore this error since we do not know what type of annotation was used, so a failure
			// is OK.
			continue
		}

		if err := util.InsertIntoSliceStructField(parent.Interface(), fieldName, target.Elem().Interface()); err != nil {
			return fmt.Errorf("cannot insert type %s into parent struct, %v", t.Name(), err)
		}
		return nil
	}

	return fmt.Errorf("no valid type found for annotation %v, errors: %v", annContents, errs)
}
