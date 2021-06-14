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

	"github.com/openconfig/gnmi/value"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	yextpb "github.com/openconfig/ygot/proto/yext"
	wpb "github.com/openconfig/ygot/proto/ywrapper"
)

// PathsFromProto returns, from a populated proto, a map between the YANG schema
// path (as specified in the yext.schemapath extension) and the value populated in
// the message.
func PathsFromProto(p proto.Message) (map[*gpb.Path]interface{}, error) {
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
		// Deal with the case where the list member is set to nil.
		if !v.IsValid() {
			return nil, fmt.Errorf("nil list member in field %s, %v", fd.FullName(), v)
		}

		// The only case of having proto.Message in a list key is when the field
		// represents the list's value portion, therefore return this value.
		return &parsedListField{member: v.Message().Interface()}, nil

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

// UnmapOpt marks that a particular option can be supplied as an argument
// to the ProtoFromPaths function.
type UnmapOpt interface {
	isUnmapOpt()
}

// IgnoreExtraPaths indicates that unmapping should ignore any additional
// paths that are found in the gNMI Notifications that do not have corresponding
// fields in the protobuf.
//
// This option is typically used in conjunction with path compression where there
// are some leaves that do not have corresponding fields.
func IgnoreExtraPaths() *ignoreExtraPaths { return &ignoreExtraPaths{} }

type ignoreExtraPaths struct{}

// isUnmapOpt marks ignoreExtraPaths as an unmap option.
func (*ignoreExtraPaths) isUnmapOpt() {}

// ValuePathPrefix indicates that the values in the supplied map have a prefix which
// is equal to the supplied path. The prefix plus each path in the vals map must be
// equal to the absolute path for the supplied values.
func ValuePathPrefix(path *gpb.Path) *valuePathPrefix {
	return &valuePathPrefix{p: path}
}

type valuePathPrefix struct{ p *gpb.Path }

// isUnmapOpt marks valuePathPrefix as an unmap option.
func (*valuePathPrefix) isUnmapOpt() {}

// ProtobufMessagePrefix specifies the path that the protobuf message supplied to ProtoFromPaths
// makes up. This is used in cases where the message itself is not the root - and hence unmarshalling
// should look for paths relative to the specified path in the vals map.
func ProtobufMessagePrefix(path *gpb.Path) *protoMsgPrefix {
	return &protoMsgPrefix{p: path}
}

type protoMsgPrefix struct{ p *gpb.Path }

// isUnmapOpt marks protoMsgPrefix as an unmap option.
func (*protoMsgPrefix) isUnmapOpt() {}

// ProtoFromPaths takes an input ygot-generated protobuf and unmarshals the values provided in vals into the map.
// The vals map must be keyed by the gNMI path to the leaf, with the interface{} value being the value that the
// leaf at the field should be set to.
//
// The protobuf p is modified in place to add the values.
//
// The set of UnmapOpts that are provided (opt) are used to control the behaviour of unmarshalling the specified data.
//
// ProtoFromPaths returns an error if the data cannot be unmarshalled.
func ProtoFromPaths(p proto.Message, vals map[*gpb.Path]interface{}, opt ...UnmapOpt) error {
	if p == nil {
		return errors.New("nil protobuf supplied")
	}

	valPrefix, err := hasValuePathPrefix(opt)
	if err != nil {
		return fmt.Errorf("invalid value prefix supplied, %v", err)
	}

	protoPrefix, err := hasProtoMsgPrefix(opt)
	if err != nil {
		return fmt.Errorf("invalid protobuf message prefix supplied in options, %v", err)
	}

	schemaPath := func(p *gpb.Path) *gpb.Path {
		np := proto.Clone(p).(*gpb.Path)
		for _, e := range np.Elem {
			e.Key = nil
		}
		return np
	}

	// directCh is a map between the absolute schema path for a particular value, and
	// the value specified.
	directCh := map[*gpb.Path]interface{}{}
	for p, v := range vals {
		absPath := &gpb.Path{
			Elem: append(append([]*gpb.PathElem{}, schemaPath(valPrefix).Elem...), p.Elem...),
		}

		if !util.PathMatchesPathElemPrefix(absPath, protoPrefix) {
			return fmt.Errorf("invalid path provided, absolute paths must be used, %s does not have prefix %s", absPath, protoPrefix)
		}

		// make the path absolute, and a schema path.
		pp := util.TrimGNMIPathElemPrefix(absPath, protoPrefix)

		if len(pp.GetElem()) == 1 {
			directCh[pp] = v
		}
		// TODO(robjs): it'd be good to have something here that tells us whether we are in
		// a compressed schema. Potentially we should add something to the generated protobuf
		// as a fileoption that would give us this indication.
		if len(pp.Elem) == 2 {
			if pp.Elem[len(pp.Elem)-2].Name == "config" || pp.Elem[len(pp.Elem)-2].Name == "state" {
				directCh[pp] = v
			}
		}
	}

	mapped := map[*gpb.Path]bool{}
	m := p.ProtoReflect()
	var rangeErr error
	unpopRange{m}.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		annotatedPath, err := annotatedSchemaPath(fd)
		if err != nil {
			rangeErr = err
			return false
		}

		for _, ap := range annotatedPath {
			if !util.PathMatchesPathElemPrefix(ap, protoPrefix) {
				rangeErr = fmt.Errorf("annotation %s does not match the supplied prefix %s", ap, protoPrefix)
				return false
			}
			trimmedAP := util.TrimGNMIPathElemPrefix(ap, protoPrefix)
			for chp, chv := range directCh {
				if proto.Equal(trimmedAP, chp) {
					switch fd.Kind() {
					case protoreflect.MessageKind:
						v, isWrap, err := makeWrapper(m, fd, chv)
						if err != nil {
							rangeErr = err
							return false
						}
						if !isWrap {
							// TODO(robjs): recurse into the message if it wasn't a wrapper
							// type.
							rangeErr = fmt.Errorf("unimplemented: child messages, field %s", fd.FullName())
							return false
						}
						mapped[chp] = true
						m.Set(fd, protoreflect.ValueOfMessage(v))
					case protoreflect.EnumKind:
						v, err := enumValue(fd, chv)
						if err != nil {
							rangeErr = err
							return false
						}
						mapped[chp] = true
						m.Set(fd, v)
					default:
						rangeErr = fmt.Errorf("unknown field kind %s for %s", fd.Kind(), fd.FullName())
						return false
					}
				}
			}
		}
		return true
	})

	if rangeErr != nil {
		return rangeErr
	}

	if !hasIgnoreExtraPaths(opt) {
		for chp := range directCh {
			if !mapped[chp] {
				return fmt.Errorf("did not map path %s to a proto field", chp)
			}
		}
	}

	return nil
}

// hasIgnoreExtraPaths checks whether the supplied opts slice contains the
// ignoreExtraPaths option.
func hasIgnoreExtraPaths(opts []UnmapOpt) bool {
	for _, o := range opts {
		if _, ok := o.(*ignoreExtraPaths); ok {
			return true
		}
	}
	return false
}

// hasProtoMsgPrefix checks whether the supplied opts slice contains the
// protoMsgPrefix option, and validates and returns the path it contains.
func hasProtoMsgPrefix(opts []UnmapOpt) (*gpb.Path, error) {
	for _, o := range opts {
		if v, ok := o.(*protoMsgPrefix); ok {
			if v.p == nil {
				return nil, fmt.Errorf("invalid protobuf prefix supplied, %+v", v)
			}
			return v.p, nil
		}
	}
	return &gpb.Path{}, nil
}

// hasValuePathPrefix checks whether the supplied opts slice contains
// the valuePathPrefix option, and validates and returns the apth it contains.
func hasValuePathPrefix(opts []UnmapOpt) (*gpb.Path, error) {
	for _, o := range opts {
		if v, ok := o.(*valuePathPrefix); ok {
			if v.p == nil {
				return nil, fmt.Errorf("invalid protobuf prefix supplied, %+v", v)
			}
			return v.p, nil
		}
	}
	return &gpb.Path{}, nil
}

// makeWrapper generates a new message for field fd of the proto message msg with the value set to val.
// The field fd must describe a field that has a message type. An error is returned if the wrong
// type of payload is provided for the message. The second, boolean, return argument specifies whether
// the message provided was a known wrapper type.
func makeWrapper(msg protoreflect.Message, fd protoreflect.FieldDescriptor, val interface{}) (protoreflect.Message, bool, error) {
	var wasTypedVal bool
	if tv, ok := val.(*gpb.TypedValue); ok {
		pv, err := value.ToScalar(tv)
		if err != nil {
			return nil, false, fmt.Errorf("cannot convert TypedValue to scalar, %s", tv)
		}
		val = pv
		wasTypedVal = true
	}

	newV := msg.NewField(fd)
	switch newV.Message().Interface().(type) {
	case *wpb.StringValue:
		nsv, ok := val.(string)
		if !ok {
			return nil, false, fmt.Errorf("got non-string value for string field, field: %s, value: %v", fd.FullName(), val)
		}
		return (&wpb.StringValue{Value: nsv}).ProtoReflect(), true, nil
	case *wpb.UintValue:
		var nsv uint64
		switch {
		case wasTypedVal:
			nsv = val.(uint64)
		default:
			iv, ok := val.(uint)
			if !ok {
				return nil, false, fmt.Errorf("got non-uint value for uint field, field: %s, value: %v", fd.FullName(), val)
			}
			nsv = uint64(iv)
		}

		return (&wpb.UintValue{Value: nsv}).ProtoReflect(), true, nil
	case *wpb.BytesValue:
		bv, ok := val.([]byte)
		if !ok {
			return nil, false, fmt.Errorf("got non-byte slice value for bytes field, field: %s, value: %v", fd.FullName(), val)
		}
		return (&wpb.BytesValue{Value: bv}).ProtoReflect(), true, nil
	default:
		return nil, false, nil
	}
}

// enumValue returns the concrete implementation of the enumeration with the yang_name annotation set
// to the string contained in val of the enumeration within the field descriptor fd. It returns an
// error if the value cannot be found, or the input value is not valid.
func enumValue(fd protoreflect.FieldDescriptor, val interface{}) (protoreflect.Value, error) {
	var setVal string
	switch inVal := val.(type) {
	case string:
		setVal = inVal
	case *gpb.TypedValue:
		tv, err := value.ToScalar(inVal)
		if err != nil {
			return protoreflect.ValueOf(nil), fmt.Errorf("cannot convert supplied TypedValue to scalar, %v", err)
		}
		s, ok := tv.(string)
		if !ok {
			return protoreflect.ValueOf(nil), fmt.Errorf("supplied TypedValue for enumeration must be a string, got: %T", tv)
		}
		setVal = s
	default:
		return protoreflect.ValueOf(nil), fmt.Errorf("got unknown type for enumeration, %T", inVal)
	}

	evals := map[string]protoreflect.EnumValueDescriptor{}
	for i := 0; i < fd.Enum().Values().Len(); i++ {
		tv := fd.Enum().Values().Get(i)
		yn, ok, err := enumYANGName(fd.Enum().Values().Get(i))
		if err != nil {
			return protoreflect.ValueOf(nil), fmt.Errorf("error with enumeration value %s", fd.FullName())
		}
		if !ok {
			continue
		}
		evals[yn] = tv
	}

	setEnumVal, ok := evals[setVal]
	if !ok {
		return protoreflect.ValueOf(nil), fmt.Errorf("got unknown value in enumeration %s, %s", fd.FullName(), setVal)
	}

	return protoreflect.ValueOfEnum(setEnumVal.Number()), nil
}

// enumYANGName returns the value of the yang_name annotation to a protobuf enumeration
// value. It reads from the supplied enum value descriptor (ed), which must be an
// enumeration descriptor. It returns the found annotation, a bool indicating whether the
// annotation existed, and an error.
//
// The bool indicating whether the annotation exists is used because unset values within
// an enumeration that do not have a real YANG value simply omit the annotation.
func enumYANGName(ed protoreflect.EnumValueDescriptor) (string, bool, error) {
	eo := ed.Options().(*descriptorpb.EnumValueOptions)
	ex := proto.GetExtension(eo, yextpb.E_YangName).(string)
	if ex == "" {
		// this is an unset value, so mark that the caller doesn't need to handle
		// this.
		return "", false, nil
	}
	return ex, true, nil
}
