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

	"github.com/openconfig/ygot/util"
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
// the fields map to, and with the parsed value from the supplied protobuf message. If the field supplied is a
func parseField(fd protoreflect.FieldDescriptor, v protoreflect.Value, vals map[*gpb.Path]interface{}, basePath *gpb.Path) error {
	if fd.IsMap() {
		return errors.New("map fields are not supported in ygen-generated protobufs")
	}

	annotatedPath, err := annotatedSchemaPath(fd)
	if err != nil {
		return err
	}
	mapPath := annotatedPath

	// Set to scalar value by default -- we extract the value from the
	// wrapper message, or child messages if required.
	val := v.Interface()

	if fd.IsList() {
		// List fields are 'repeated' in the input protobuf. We have two cases of such
		// fields:
		//  1. leaf-list fields which have scalar values - and hence are mapped in the
		//     same way as the handling of individual fields in the protobuf.
		//  2. list types in YANG - we only support keyed lists, since these have their
		//     own valid paths. For the generated protobufs we create a new XXXKey message
		//     which is the repeated type. Scalar fields within that message are the
		//     individual keys of the list (there are >= 1 of them) -- and the single
		//     message type that is included is a child message.

		// Lists cannot map to >1 different schema path in the compressed form of generated
		// protobufs.
		if len(mapPath) > 1 {
			return fmt.Errorf("invalid list, maps to >1 schema path, field: %s", fd.FullName())
		}
		listPath := mapPath[0]

		// TODO(robjs): This handles the case where we have a list - but not a leaf-list.
		l := v.List()
		var listVal proto.Message
		for i := 0; i < l.Len(); i++ {
			lv := l.Get(i)
			listMsg, ok := lv.Message().Interface().(proto.Message)
			if !ok {
				return fmt.Errorf("invalid list, value is not a proto message, %s - is %T", fd.FullName(), lv.Interface())
			}
			var listParseErr error
			listKeys := map[string]string{}

			m := listMsg.ProtoReflect()
			m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
				fmt.Printf("calling with %s\n", listPath)
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
					vals[k] = v
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

			// Handle recursing down the tree for the list that we just extracted.
			p := listPath
			if basePath != nil {
				// This is the first time that we have found a path that requires a
				// data tree path, not a schema tree path.
				rp, err := resolvedPath(basePath, listPath)
				if err != nil {
					return err
				}
				p = rp
			}

			for kn, kv := range listKeys {
				p.Elem[len(p.Elem)-1].Key[kn] = kv
			}
			fmt.Printf("recursively calling pathsFromProtoInternal with %s\n", p)
			if err := pathsFromProtoInternal(listVal, vals, p); err != nil {
				return err
			}
		}
		return nil
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
			return pathsFromProtoInternal(t, vals, basePath)
		}
	}

	// Handle cases where there is >1 path specified for a field based on
	// path compression.
	for _, path := range mapPath {
		/*if basePath != "" && !strings.Contains(path, basePath) {
			return fmt.Errorf("basePath %s doesn't contain this path %s", basePath, path)
		}*/
		tp := path
		if basePath != nil {
			var err error
			if tp, err = resolvedPath(basePath, path); err != nil {
				return err
			}
		}
		fmt.Printf("basePath: %s\nmapPath: %s\n", basePath, path)

		vals[tp] = val
	}

	return nil
}

type parsedList struct {
	keys         map[string]string
	member       proto.Message
	mappedValues map[*gpb.Path]interface{}
}

func parseListField(fd protoreflect.FieldDescriptor, v protoreflect.Value, basePath *gpb.Path) (*parsedList, error) {
	if fd.IsMap() || fd.IsList() {
		return nil, fmt.Errorf("invalid field in list, %s", fd.FullName())
	}

	if fd.Kind() == protoreflect.MessageKind {
		if t, ok := v.Message().Interface().(proto.Message); ok {
			// The only case of having proto.Message in a list key is when it is the
			// value itself, therefore return this value.
			return &parsedList{member: t}, nil
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

	p := &parsedList{
		keys:         map[string]string{keyName: kv},
		mappedValues: map[*gpb.Path]interface{}{},
	}

	for _, path := range mappedPaths {
		childElems := util.TrimGNMIPathElemPrefix(path, removePathKeys(basePath))
		newPath := &gpb.Path{Elem: append(basePath.Elem, childElems.Elem...)}
		p.mappedValues[newPath] = v.Interface()
	}
	return p, nil
}

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

func fieldName(path *gpb.Path) (string, error) {
	if len(path.Elem) == 0 || path == nil {
		return "", fmt.Errorf("invalid path %s", path)
	}
	return path.Elem[len(path.Elem)-1].Name, nil
}

func resolvedPath(dataTreePath *gpb.Path, annotatedPath *gpb.Path) (*gpb.Path, error) {
	pathSchemaParts := removePathKeys(dataTreePath)
	pathNoPrefix := util.TrimGNMIPathElemPrefix(pathSchemaParts, dataTreePath)
	np := proto.Clone(dataTreePath).(*gpb.Path)
	np.Elem = append(np.Elem, pathNoPrefix.Elem...)
	return np, nil
}

func removePathKeys(path *gpb.Path) *gpb.Path {
	np := &gpb.Path{
		Origin: path.Origin,
	}
	for _, e := range path.Elem {
		np.Elem = append(np.Elem, &gpb.PathElem{Name: e.Name})
	}
	return np
}
