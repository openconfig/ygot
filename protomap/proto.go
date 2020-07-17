// Copyright 2020 Google Inc.
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

// Package protomap provides utilities that map ygen-generated protobuf
// messages to and from other types (e.g., gNMI Notification messages,
// or ygen-generated GoStructs).
package protomap

import (
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/openconfig/ygot/ygot"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	yextpb "github.com/openconfig/ygot/proto/yext"
	wpb "github.com/openconfig/ygot/proto/ywrapper"
)

// pathsFromProto returns, from a populated proto, a map between the YANG schema
// path (as specified in the yext.schemapath extension) and the value populated in
// the message.
func pathsFromProto(p proto.Message) (map[*gpb.Path]interface{}, error) {
	pp := map[*gpb.Path]interface{}{}
	if err := pathsFromProtoInternal(p, pp, nil); err != nil {
		return nil, err
	}
	return pp, nil
}

// pathsFromProtoInternal is called recursively for each proto.Message field that
// is found within an input protobuf message. It extracts the fields that are specified
// within the
func pathsFromProtoInternal(p proto.Message, vals map[*gpb.Path]interface{}, basePath *gpb.Path) error {
	m := p.ProtoReflect()
	var rangeErr error
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if err := parseField(fd, v, vals, basePath); err != nil {
			rangeErr = err
			return false
		}
		return true
	})

	if rangeErr != nil {
		return rangeErr
	}

	return nil
}

// parseField handles a single field of a protobuf message, as described by the supplied descriptor, and
// with the specified value. It appends entries to the supplied vals map, keyed by the data tree path that
// the fields map to, and with the parsed value from the supplied protobuf message.
func parseField(fd protoreflect.FieldDescriptor, v protoreflect.Value, vals map[*gpb.Path]interface{}, basePath *gpb.Path) error {
	if fd.IsMap() {
		return errors.New("map fields are not supported in ygen-generated protobufs")
	}

	annotatedPath, err := annotatedSchemaPath(fd)
	if err != nil {
		return err
	}

	// Set to scalar value by default -- we extract the value from the
	// wrapper message, or child messages if required.
	val := v.Interface()

	if fd.IsList() {
		return parseList(fd, v, vals, basePath, annotatedPath)
	}

	// Handle messages that are field values
	if fd.Kind() == protoreflect.MessageKind {
		switch t := v.Message().Interface().(type) {
		case *wpb.BoolValue:
			val = t.GetValue()
		case *wpb.BytesValue:
			val = t.GetValue()
		case *wpb.Decimal64Value:
			return fmt.Errorf("unhandled type, decimal64")
		case *wpb.IntValue:
			val = t.GetValue()
		case *wpb.StringValue:
			val = t.GetValue()
		case *wpb.UintValue:
			val = t.GetValue()
		case proto.Message:
			if len(annotatedPath) != 1 {
				return fmt.Errorf("invalid container, maps to >1 schema path, field: %s", fd.FullName())
			}
			return pathsFromProtoInternal(t, vals, basePath)
		}
	}

	// Handle cases where there is >1 path specified for a field based on
	// path compression.
	for _, path := range annotatedPath {
		vals[resolvedPath(basePath, path)] = val
	}

	return nil
}

// Modify the Range function for a protoreflect.Message to be able to cover fields that
// are not populated, since we need to be able to support scalar fields in our ranges.
//
// This code is taken from the updated protojson package - and is used because we need
// to range over all scalar fields within the populated key messages for a list - since
// we should include the values even if they are set to the Go default value (e.g., a uint32
// is set to 0).
type unpopRange struct{ protoreflect.Message }

// Range wraps the protomessage.Range, and sets fields to be marked as non-nil even if they
// are set to the Go default value. This means that we will output fields that are unset as
// their nil values, which is required for list keys within these messages.
func (m unpopRange) Range(f func(protoreflect.FieldDescriptor, protoreflect.Value) bool) {
	fds := m.Descriptor().Fields()
	for i := 0; i < fds.Len(); i++ {
		fd := fds.Get(i)
		if m.Has(fd) || fd.ContainingOneof() != nil {
			continue // ignore populated fields and fields within a oneofs
		}

		v := m.Get(fd)
		isProto2Scalar := fd.Syntax() == protoreflect.Proto2 && fd.Default().IsValid()
		isSingularMessage := fd.Cardinality() != protoreflect.Repeated && fd.Kind() == protoreflect.MessageKind
		if isProto2Scalar || isSingularMessage {
			v = protoreflect.Value{} // use invalid value to emit null
		}
		if !f(fd, v) {
			return
		}
	}
	m.Message.Range(f)
}

// parseList parses the field described by fd, with value v - which must be a repeated field in
// the protobuf, and appends its values to the value map - using the supplied base and mapped paths
// to determine the data tree paths of the populated fields. It returns an error if values cannot
// be extracted,
//
// List fields are 'repeated' in the input protobuf. We have two cases of such
// fields:
//  1. leaf-list fields which have scalar values - and hence are mapped in the
//     same way as the handling of individual fields in the protobuf.
//  2. list types in YANG - we only support keyed lists, since these have their
//     own valid paths. For the generated protobufs we create a new XXXKey message
//     which is the repeated type. Scalar fields within that message are the
//     individual keys of the list (there are >= 1 of them) -- and the single
//     message type that is included is a child message.
func parseList(fd protoreflect.FieldDescriptor, v protoreflect.Value, vals map[*gpb.Path]interface{}, basePath *gpb.Path, mapPath []*gpb.Path) error {
	// Lists cannot map to >1 different schema path in the compressed form of generated
	// protobufs.
	if len(mapPath) != 1 {
		return fmt.Errorf("invalid list, does not map to 1 schema path, field: %s", fd.FullName())
	}
	listPath := mapPath[0]

	// TODO(robjs): This handles the case where we have a list - but not a leaf-list.
	//              Add implementation to handle leaf lists.
	l := v.List()
	if fd.Kind() != protoreflect.MessageKind {
		return fmt.Errorf("invalid list, value is not a proto message, %s - is %T", fd.FullName(), l.NewElement())
	}
	var listVal proto.Message
	for i := 0; i < l.Len(); i++ {
		listMsg := l.Get(i).Message().Interface().(proto.Message)

		var listParseErr error
		listKeys := map[string]string{}
		mappedValues := map[*gpb.Path]interface{}{}

		unpopRange{listMsg.ProtoReflect()}.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			pl, err := parseListField(fd, v, listPath)
			if err != nil {
				listParseErr = err
				return false
			}

			// If this field was not the value of the list then we receive a populated
			// single key back from the parseListField call. We don't check for nil
			// since if it is not populated then we'll simply never do this mapping.
			for k, v := range pl.keys {
				listKeys[k] = v
			}

			// When there are keys in the list, we'll also have fields that they map to
			// in the output set of paths, so add these to the values that we're receiving.
			for k, v := range pl.mappedValues {
				mappedValues[k] = v
			}

			// If this was the list value, then we return the value of the list,
			// which is always a protobuf message back.
			if pl.member != nil {
				listVal = pl.member
			}
			return true
		})

		if listParseErr != nil {
			return fmt.Errorf("could not parse a field within the list %s , %v", fd.FullName(), listParseErr)
		}

		// This is the first time that we have found a path that requires a
		// data tree path, not a schema tree path.
		p := resolvedPath(basePath, listPath)

		for kn, kv := range listKeys {
			le := p.Elem[len(p.Elem)-1]
			if le.Key == nil {
				le.Key = map[string]string{}
			}
			le.Key[kn] = kv
		}

		for path, value := range mappedValues {
			vals[resolvedPath(p, path)] = value
		}

		if err := pathsFromProtoInternal(listVal, vals, p); err != nil {
			return err
		}
	}
	return nil
}

// parsedListField returns the details of an individual field of a message
// which corresponds to a YANG list (is 'repeated').
type parsedListField struct {
	// keys is a map of the keys using the gNMI path format for keys - where all
	// values are mapped to strings. Since the parsed field contains only one field
	// then only one key will ever be populated. This field is only set to a non-nil
	// value if the field corresponds to a key.
	keys map[string]string
	// mappedValues stores the values that are contained in the input protobuf, and is
	// populated only when the field supplied is a key. Since the key fields are removed
	// from the 'value' protobuf, their values can only be extracted from the repeated 'Key'
	// message. A single field may result in >1 populated mappedValues in the case that there
	// are multiple paths within the schemapath annotation.
	mappedValues map[*gpb.Path]interface{}
	// member is populated when the field parsed is the member of the list - i.e., the
	// repeated proto.Message which corresponds to the subtree under the list at a particular
	// key.
	member proto.Message
}

// parseListField parses an individual field within a 'repeated' message representing a YANG list, as
// described by fd, and with value v. The supplied basePath is used for the base data tree path for field
// specified. It returns a parsedListField describing the individual field supplied.
func parseListField(fd protoreflect.FieldDescriptor, v protoreflect.Value, basePath *gpb.Path) (*parsedListField, error) {
	if fd.IsMap() || fd.IsList() {
		return nil, fmt.Errorf("list field is of unexpected map or list type: %q", fd.FullName())
	}

	if fd.Kind() == protoreflect.MessageKind {
		if t, ok := v.Message().Interface().(proto.Message); ok {
			// The only case of having proto.Message in a list key is when the field
			// represents the list's value portion, therefore return this value.
			return &parsedListField{member: t}, nil
		}
	}

	mapPaths, err := annotatedSchemaPath(fd)
	if err != nil {
		return nil, err
	}

	var keyName string
	mappedPaths := []*gpb.Path{}
	for _, p := range mapPaths {
		n, err := fieldName(p)
		if err != nil {
			return nil, err
		}
		switch {
		case keyName == "":
			keyName = n
		case n != keyName:
			return nil, fmt.Errorf("received list key with leaf names that do not match, %s != %s", keyName, n)
		}
		mappedPaths = append(mappedPaths, p)
	}

	kv, err := ygot.KeyValueAsString(v.Interface())
	if err != nil {
		return nil, fmt.Errorf("cannot map list key %v, %v", v.Interface(), err)
	}

	p := &parsedListField{
		keys:         map[string]string{keyName: kv},
		mappedValues: map[*gpb.Path]interface{}{},
	}

	for _, path := range mappedPaths {
		p.mappedValues[resolvedPath(basePath, path)] = v.Interface()
	}
	return p, nil
}

// annotatedSchemaPath extracts the schemapath annotation from the supplied field descriptor,
// parsing the included string paths into a slice of gNMI 'Path' messages.
func annotatedSchemaPath(fd protoreflect.FieldDescriptor) ([]*gpb.Path, error) {
	po := fd.Options().(*descriptorpb.FieldOptions)
	ex := proto.GetExtension(po, yextpb.E_Schemapath).(string)
	if ex == "" {
		return nil, fmt.Errorf("received field with invalid annotation, field: %s", fd.FullName())
	}

	paths := []*gpb.Path{}
	for _, p := range strings.Split(ex, "|") {
		pp, err := ygot.StringToStructuredPath(p)
		if err != nil {
			return nil, fmt.Errorf("received invalid annotated path, %s, %v", ex, err)
		}
		paths = append(paths, pp)
	}
	return paths, nil
}

// fieldName returns the name last element of the path supplied - corresponding
// to the field that is being described by the specified path.
func fieldName(path *gpb.Path) (string, error) {
	if len(path.Elem) == 0 || path == nil || path.Elem[len(path.Elem)-1].Name == "" {
		return "", fmt.Errorf("invalid path %s", path)
	}
	return path.Elem[len(path.Elem)-1].Name, nil
}

// resolvedPath fully resolves a path of an element with the annotation
// supplied in the annotatedPath, from the supplied basePath - which
// is a resolved data tree path (which may include list keys).
func resolvedPath(basePath, annotatedPath *gpb.Path) *gpb.Path {
	if basePath == nil {
		return annotatedPath
	}
	np := proto.Clone(basePath).(*gpb.Path)
	np.Elem = append(np.Elem, annotatedPath.Elem[len(basePath.Elem):]...)
	return np
}
