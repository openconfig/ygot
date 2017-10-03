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

package ygot

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/gnmi/errlist"
	"github.com/openconfig/gnmi/value"
	"github.com/openconfig/ygot/util"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

const (
	// BinaryTypeName is the name of the type that is used for YANG
	// binary fields in the output structs.
	BinaryTypeName string = "Binary"
	// EmptyTypeName is the name of the type that is used for YANG
	// empty fields in the output structs.
	EmptyTypeName string = "YANGEmpty"
)

// path stores the elements of a path for a particular leaf,
// such that it can be used as a key for maps.
type path struct {
	p *gnmiPath
}

// gnmiPath provides a wrapper for gNMI path types, particularly
// containing the Element-based paths which are used in gNMI pre-0.3.1 and
// PathElem-based paths which are used in gNMI 0.4.0 and above.
type gnmiPath struct {
	// stringSlicePath stores a path expressed as a series of scalar elements. On output it is
	// rendered to a []string which is placed in the gNMI element field.
	stringSlicePath []string
	// pathElemPath stores a path expressed as a series of PathElem messages.
	pathElemPath []*gnmipb.PathElem
	// isAbsolute determines whether the stored path is absolute (when set), or relative
	// when unset.
	isAbsolute bool
}

// newStringSliceGNMIPath returns a new gnmiPath with a string slice path.
func newStringSliceGNMIPath(s []string) *gnmiPath {
	if s == nil {
		s = []string{}
	}
	return &gnmiPath{stringSlicePath: s}
}

// newPathElemGNMIPath returns a new gnmiPath with a PathElem path.
func newPathElemGNMIPath(e []*gnmipb.PathElem) *gnmiPath {
	if e == nil {
		e = []*gnmipb.PathElem{}
	}
	return &gnmiPath{pathElemPath: e}
}

// isValid determines whether a gnmiPath is valid by determing whether the
// elementPath and structuredPath are both set or both unset.
func (g *gnmiPath) isValid() bool {
	return (g.stringSlicePath == nil) != (g.pathElemPath == nil)
}

// isStringSlicePath determines whether the gnmiPath receiver describes a simple
// string slice path, or a structured path using gnmipb.PathElem values.
func (g *gnmiPath) isStringSlicePath() bool {
	return g.stringSlicePath != nil
}

// isPathElemPath determines whether the gnmiPath receiver describes a structured
// PathElem based gNMI path.
func (g *gnmiPath) isPathElemPath() bool {
	return g.pathElemPath != nil
}

// Copy returns a copy of the current gnmiPath.
func (g *gnmiPath) Copy() *gnmiPath {
	n := &gnmiPath{}
	if g.isStringSlicePath() {
		n.stringSlicePath = make([]string, len(g.stringSlicePath))
		copy(n.stringSlicePath, g.stringSlicePath)
		return n
	}

	n.pathElemPath = make([]*gnmipb.PathElem, len(g.pathElemPath))
	copy(n.pathElemPath, g.pathElemPath)
	return n
}

// Len returns the length of the path specified by gnmiPath.
func (g *gnmiPath) Len() int {
	if g.isStringSlicePath() {
		return len(g.stringSlicePath)
	}
	return len(g.pathElemPath)
}

// AppendName appends the string n as a new name within the gnmiPath.
// If the supplied name is nil, it is not appended.
func (g *gnmiPath) AppendName(n string) error {
	if !g.isValid() {
		return fmt.Errorf("cannot append to invalid path")
	}

	if n == "" {
		return nil
	}

	if g.isStringSlicePath() {
		g.stringSlicePath = append(g.stringSlicePath, n)
		return nil
	}
	g.pathElemPath = append(g.pathElemPath, &gnmipb.PathElem{Name: n})
	return nil
}

// LastPathElem returns the last PathElem element in the gnmiPath.
func (g *gnmiPath) LastPathElem() (*gnmipb.PathElem, error) {
	return g.PathElemAt(g.Len() - 1)
}

// LastStringElem returns the last string element of the gnmiPath.
func (g *gnmiPath) LastStringElem() (string, error) {
	return g.StringElemAt(g.Len() - 1)
}

// PathElemAt returns the PathElem at index i in the gnmiPath. It returns an error if the
// path is invalid, is not a path elem path, or the index is greater than the length
// of the path.
func (g *gnmiPath) PathElemAt(i int) (*gnmipb.PathElem, error) {
	if !g.isValid() || !g.isPathElemPath() {
		return nil, errors.New("invalid call to PathElemAt() on a non-PathElem path")
	}

	if i > g.Len() || i < 0 {
		return nil, fmt.Errorf("invalid index %d for gnmiPath len %d", i, g.Len())
	}

	return g.pathElemPath[i], nil
}

// StringElemAt returns the string element at index i in the gnmiPath. It returns an error
// if the path is invalid, is not a string slice path, or the index is greater than the
// length of the path.
func (g *gnmiPath) StringElemAt(i int) (string, error) {
	if !g.isValid() || !g.isStringSlicePath() {
		return "", errors.New("invalid call to StringElemAt() on a non-string element path")
	}

	if i > g.Len() || i < 0 {
		return "", fmt.Errorf("invalid index %d for gnmiPath len %d", i, g.Len())
	}

	return g.stringSlicePath[i], nil
}

// SetIndex sets the element at index i to the value v.
func (g *gnmiPath) SetIndex(i int, v interface{}) error {
	if i > g.Len() {
		return fmt.Errorf("invalid index, out of range, got: %d, length: %d", i, g.Len())
	}

	switch v.(type) {
	case string:
		if !g.isStringSlicePath() {
			return fmt.Errorf("cannot set index %d of %v to %v, wrong type %T, expected string", i, v, g, v)
		}
		g.stringSlicePath[i] = v.(string)
		return nil
	case *gnmipb.PathElem:
		if !g.isPathElemPath() {
			return fmt.Errorf("cannot set index %d of %v to %v, wrong type %T, expected gnmipb.PathElem", i, v, g, v)
		}
		g.pathElemPath[i] = v.(*gnmipb.PathElem)
		return nil
	}
	return fmt.Errorf("cannot set index %d of %v to %v, wrong type %T", i, v, g, v)
}

// ToProto returns the gnmiPath as a gnmi.proto Path message.
func (g *gnmiPath) ToProto() (*gnmipb.Path, error) {
	if !g.isValid() {
		return nil, errors.New("invalid path")
	}

	if g.Len() == 0 {
		return nil, nil
	}

	if g.isStringSlicePath() {
		return &gnmipb.Path{Element: g.stringSlicePath}, nil
	}
	return &gnmipb.Path{Elem: g.pathElemPath}, nil
}

// isSameType returns true if the path supplied is the same type as the
// receiver.
func (g *gnmiPath) isSameType(p *gnmiPath) bool {
	return g.isStringSlicePath() == p.isStringSlicePath()
}

// StripPrefix removes the prefix pfx from the supplied path, and returns the more
// specific path elements of the path. It returns an error if the paths are invalid,
// their types are different, or the prefix does not match the path.
func (g *gnmiPath) StripPrefix(pfx *gnmiPath) (*gnmiPath, error) {
	if !g.isSameType(pfx) {
		return nil, fmt.Errorf("mismatched path formats in prefix and path, isElementPath: %v != %v", g.isStringSlicePath(), pfx.isStringSlicePath())
	}

	if !g.isValid() || !pfx.isValid() {
		return nil, fmt.Errorf("invalid paths supplied for stripPrefix: %v, %v", g, pfx)
	}

	if pfx.isStringSlicePath() {
		for i, e := range pfx.stringSlicePath {
			if g.stringSlicePath[i] != e {
				return nil, fmt.Errorf("prefix is not a prefix of the supplied path, %v is not a subset of %v", pfx, g)
			}
		}
		return newStringSliceGNMIPath(g.stringSlicePath[len(pfx.stringSlicePath):]), nil
	}

	for i, e := range pfx.pathElemPath {
		if !proto.Equal(g.pathElemPath[i], e) {
			return nil, fmt.Errorf("prefix is not a prefix of the supplied path, %v is not a subset of %v", pfx, g)
		}
	}
	return newPathElemGNMIPath(g.pathElemPath[len(pfx.pathElemPath):]), nil
}

// GNMINotificationsConfig specifies arguments determining how the
// gNMI output should be created by ygot.
type GNMINotificationsConfig struct {
	// UsePathElem specifies whether the elem field of the gNMI Path
	// message should be used for the paths in the output notification. If
	// set to false, the element field is used.
	UsePathElem bool
	// ElementPrefix stores the prefix that should be used within the
	// Prefix field of the gNMI Notification message expressed as a slice
	// of strings as per the path definition in gNMI 0.3.1 and below.
	// Used if UsePathElem is unset.
	StringSlicePrefix []string
	// PathElemPrefix stores the prefix that should be used withinthe
	// Prefix field of the gNMI Notification message, expressed as a slice
	// of PathElem messages. This path format is used by gNMI 0.4.0 and
	// above. Used if PathElem is set.
	PathElemPrefix []*gnmipb.PathElem
}

// TogNMINotifications takes an input GoStruct and renders it to slice of
// Notification messages, marked with the specified timestamp. The configuration
// provided determines the path format utilised, and the prefix to be included
// in the message if relevant.
func TogNMINotifications(s GoStruct, ts int64, cfg GNMINotificationsConfig) ([]*gnmipb.Notification, error) {

	var pfx *gnmiPath
	if cfg.UsePathElem {
		pfx = newPathElemGNMIPath(cfg.PathElemPrefix)
	} else {
		pfx = newStringSliceGNMIPath(cfg.StringSlicePrefix)
	}

	leaves := map[*path]interface{}{}
	if err := findUpdatedLeaves(leaves, s, pfx); err != nil {
		return nil, err
	}

	msgs, err := leavesToNotifications(leaves, ts, pfx)
	if err != nil {
		return nil, err
	}

	return msgs, nil
}

// findUpdatedLeaves appends the valid leaves that are within the supplied
// GoStruct (assumed to the rooted at parentPath) to the supplied leaves map.
// If errors are encountered they are appended to the errlist.List supplied. If
// the GoStruct contains fields that are themselves structured objects (YANG
// lists, or containers - represented as maps or struct pointers), the function
// is called recursively on them.
func findUpdatedLeaves(leaves map[*path]interface{}, s GoStruct, parent *gnmiPath) error {
	var errs errlist.List

	if !parent.isValid() {
		return fmt.Errorf("invalid parent specified: %v", parent)
	}

	if s == nil {
		errs.Add(fmt.Errorf("input struct for %v was nil", parent))
		return errs.Err()
	}

	sval := reflect.ValueOf(s).Elem()
	stype := sval.Type()

	for i := 0; i < sval.NumField(); i++ {
		fval := sval.Field(i)
		ftype := stype.Field(i)

		mapPaths, err := structTagToLibPaths(ftype, parent)
		if err != nil {
			errs.Add(fmt.Errorf("%v->%s: %v", parent, ftype.Name, err))
			continue
		}

		// Handle nil values, and enumerations specifically.
		switch fval.Kind() {
		case reflect.Map, reflect.Slice, reflect.Ptr, reflect.Interface:
			if fval.IsNil() {
				continue
			}
		}

		switch fval.Kind() {
		case reflect.Map:
			// We need to map each child along with its key value.
			for _, k := range fval.MapKeys() {
				childPath, err := mapValuePath(k, fval.MapIndex(k), mapPaths[0])
				if err != nil {
					errs.Add(err)
					continue
				}

				goStruct, ok := fval.MapIndex(k).Interface().(GoStruct)
				if !ok {
					errs.Add(fmt.Errorf("%v: was not a valid GoStruct", mapPaths[0]))
					continue
				}
				errs.Add(findUpdatedLeaves(leaves, goStruct, childPath))
			}
		case reflect.Ptr:
			// Determine whether this is a pointer to a struct (another YANG container), or a leaf.
			switch fval.Elem().Kind() {
			case reflect.Struct:
				goStruct, ok := fval.Interface().(GoStruct)
				if !ok {
					errs.Add(fmt.Errorf("%v: was not a valid GoStruct", mapPaths[0]))
					continue
				}
				errs.Add(findUpdatedLeaves(leaves, goStruct, mapPaths[0]))
			default:
				for _, p := range mapPaths {
					leaves[&path{p}] = fval.Elem().Interface()
				}
			}
		case reflect.Slice:
			if fval.Type().Elem().Kind() == reflect.Ptr {
				// This is a keyless list - currently unsupported for mapping since there is
				// not an explicit path that can be used.
				errs.Add(fmt.Errorf("unimplemented: keyless list cannot be output: %v", mapPaths[0]))
				continue
			}
			// This is a leaf-list, so add it as though it were a leaf.
			for _, p := range mapPaths {
				leaves[&path{p}] = fval.Interface()
			}
		case reflect.Int64:
			name, set, err := enumFieldToString(fval, false)
			if err != nil {
				errs.Add(err)
				continue
			}

			// Skip if the enum has not been explicitly set in the schema.
			if !set {
				continue
			}

			for _, p := range mapPaths {
				leaves[&path{p}] = name
			}
			continue
		case reflect.Interface:
			// This is a union value.
			val, err := unionInterfaceValue(fval, false)
			if err != nil {
				errs.Add(err)
				continue
			}

			for _, p := range mapPaths {
				leaves[&path{p}] = val
			}
			continue
		}
	}
	return errs.Err()
}

// mapValuePath calculates the gNMI Path of a map element with the specified
// key and value. The format of the path returned depends on the input format
// of the parentPath.
func mapValuePath(key, value reflect.Value, parentPath *gnmiPath) (*gnmiPath, error) {
	childPath := &gnmiPath{}

	if parentPath == nil {
		return nil, fmt.Errorf("nil map paths supplied to mapValuePath for %v %v", key.Interface(), value.Interface())
	}

	if parentPath.isStringSlicePath() {
		keyval, err := keyValueAsString(key.Interface())
		if err != nil {
			return nil, fmt.Errorf("can't append path element key: %v", err)
		}
		// We copy the elements from the existing elementPath such that when updating
		// it, then the elements are not modified when the paths are changed.
		for _, e := range parentPath.stringSlicePath {
			childPath.stringSlicePath = append(childPath.stringSlicePath, e)
		}
		childPath.stringSlicePath = append(childPath.stringSlicePath, keyval)
		return childPath, nil
	}

	for _, e := range parentPath.pathElemPath {
		n := *e
		childPath.pathElemPath = append(childPath.pathElemPath, &n)
	}

	return appendgNMIPathElemKey(value, childPath)
}

// appendgNMIPathElemKey takes an input reflect.Value which must implement KeyHelperGoStruct
// and appends the keys from it to the last entry in the supplied mapPath, which must be a
// gNMI PathElem message.
func appendgNMIPathElemKey(v reflect.Value, p *gnmiPath) (*gnmiPath, error) {
	if p == nil {
		return nil, fmt.Errorf("nil path supplied")
	}

	if !p.isValid() {
		return nil, fmt.Errorf("invalid structured path in supplied path: %v", p)
	}

	if p.isStringSlicePath() {
		return nil, fmt.Errorf("invalid path type to append keys: %v", p)
	}

	if p.Len() == 0 {
		return nil, fmt.Errorf("invalid path element path length, can't append keys to 0 length path: %v", p.pathElemPath)
	}

	np := p.Copy()
	e, err := np.LastPathElem()
	if err != nil {
		return nil, err
	}
	newElem := *e

	if !v.IsValid() || v.IsNil() {
		return nil, fmt.Errorf("nil value received for element %v", p)
	}

	gs, ok := v.Interface().(KeyHelperGoStruct)
	if !ok {
		return nil, fmt.Errorf("cannot render to gNMI PathElem for structs that do not implement KeyHelperGoStruct, got: %T (%s)", v.Type().Name(), v.Interface())
	}

	km, err := gs.ΛListKeyMap()
	if err != nil {
		return nil, err
	}
	newElem.Key = map[string]string{}
	for kn, k := range km {
		v, err := keyValueAsString(k)
		if err != nil {
			return nil, err
		}
		newElem.Key[kn] = v
	}

	if err := np.SetIndex(np.Len()-1, &newElem); err != nil {
		return nil, err
	}
	return np, nil
}

// keyValueAsString returns a string representation of the interface{} supplied. If the
// type provided cannot be represented as a string for use in a gNMI path, an error is
// returned.
func keyValueAsString(v interface{}) (string, error) {
	kv := reflect.ValueOf(v)
	if _, isEnum := v.(GoEnum); isEnum {
		name, _, err := enumFieldToString(kv, false)
		if err != nil {
			return "", fmt.Errorf("cannot resolve enumerated type in key, got err: %v", err)
		}
		return name, nil
	}

	switch kv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v), nil
	case reflect.String:
		return v.(string), nil
	case reflect.Ptr:
		iv, err := unionPtrValue(kv, false)
		if err != nil {
			return "", err
		}
		return keyValueAsString(iv)
	}

	return "", fmt.Errorf("cannot convert type %v to a string for use in a key: %v", kv.Kind(), v)
}

// sliceToScalarArray takes an input slice of empty interfaces and converts it to
// a gNMI ScalarArray that can be populated as the leaflist_val field within a Notification
// message. Returns an error if the slice contains a type that cannot be mapped to
// a TypedValue message.
func sliceToScalarArray(v []interface{}) (*gnmipb.ScalarArray, error) {
	arr := &gnmipb.ScalarArray{}
	for _, e := range v {
		tv, err := value.FromScalar(e)
		if err != nil {
			return nil, err
		}
		arr.Element = append(arr.Element, tv)
	}
	return arr, nil
}

// leavesToNotifications takes an input map of leaves, and outputs a slice of
// notifications that corresponds to the leaf update, the supplied timestamp is
// used in the set of notifications. If an error is encountered it is returned.
// TODO(robjs): Currently, we return only a single Notification, but this is
// likely to be suboptimal since it results in very large Notifications for particular
// structs. There should be some fragmentation of Updates across Notification messages
// in a future implementation. We return a slice to keep the API stable.
func leavesToNotifications(leaves map[*path]interface{}, ts int64, pfx *gnmiPath) ([]*gnmipb.Notification, error) {
	n := &gnmipb.Notification{
		Timestamp: ts,
	}

	p, err := pfx.ToProto()
	if err != nil {
		return nil, err
	}
	n.Prefix = p

	for pk, v := range leaves {
		path, err := pk.p.StripPrefix(pfx)
		if err != nil {
			return nil, err
		}

		ppath, err := path.ToProto()
		if err != nil {
			return nil, err
		}
		u := &gnmipb.Update{Path: ppath}

		switch val := reflect.ValueOf(v); val.Kind() {
		case reflect.Slice:
			switch {
			case reflect.TypeOf(v).Name() == BinaryTypeName:
				// This is a binary type which is defined as a []byte, so
				// we encode it as bytes.
				u.Val = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_BytesVal{val.Bytes()}}
			default:
				sval, err := leaflistToSlice(val, false)
				if err != nil {
					return nil, err
				}

				arr, err := sliceToScalarArray(sval)
				if err != nil {
					return nil, err
				}
				u.Val = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_LeaflistVal{arr}}
			}
		default:
			val, err := value.FromScalar(v)
			if err != nil {
				return nil, err
			}
			u.Val = val
		}

		n.Update = append(n.Update, u)
	}

	return []*gnmipb.Notification{n}, nil
}

// leaflistToSlice takes a reflect.Value that represents a leaf list in the YANG schema
// (GoStruct) and outputs a slice of interface{} that corresponds to its contents that
// should be used within a Notification. If appendModuleName is set to true, then
// identity names are prepended with the name of the module that defines them.
func leaflistToSlice(val reflect.Value, appendModuleName bool) ([]interface{}, error) {
	sval := []interface{}{}
	for i := 0; i < val.Len(); i++ {
		e := val.Index(i)

		// Handle mapping leaf-lists. There are two cases of leaf-lists
		// within the YANG structs. The first is the simple case of having
		// a single typed leaf-list - so mapping can be done solely based
		// on the type of the elements of the slice. The second case is
		// a leaf-list of union values, which means that there may be
		// multiple types. This is represented as []interface{}
		switch e.Kind() {
		case reflect.String:
			sval = append(sval, e.String())
		case reflect.Uint8:
			sval = append(sval, uint8(e.Uint()))
		case reflect.Uint16:
			sval = append(sval, uint16(e.Uint()))
		case reflect.Uint32:
			sval = append(sval, uint32(e.Uint()))
		case reflect.Uint64, reflect.Uint:
			sval = append(sval, e.Uint())
		case reflect.Int8:
			sval = append(sval, int8(e.Int()))
		case reflect.Int16:
			sval = append(sval, int16(e.Int()))
		case reflect.Int32:
			sval = append(sval, int32(e.Int()))
		case reflect.Int:
			sval = append(sval, e.Int())
		case reflect.Int64:
			if _, ok := e.Interface().(GoEnum); ok {
				name, _, err := enumFieldToString(e, appendModuleName)
				if err != nil {
					return nil, err
				}
				sval = append(sval, name)
			} else {
				sval = append(sval, e.Int())
			}
		case reflect.Float32, reflect.Float64:
			sval = append(sval, e.Float())
		case reflect.Bool:
			sval = append(sval, e.Bool())
		case reflect.Interface:
			// Occurs in two cases:
			// 1) Where there is a leaflist of mixed types.
			// 2) Where there is a leaflist of unions.
			ival := e.Interface()
			switch reflect.TypeOf(ival).Kind() {
			case reflect.Ptr:
				uval, err := unionInterfaceValue(e, appendModuleName)
				if err != nil {
					return nil, err
				}

				sval, err = appendTypedValue(sval, reflect.ValueOf(uval), appendModuleName)
				if err != nil {
					return nil, err
				}
			default:
				var err error
				sval, err = appendTypedValue(sval, e, appendModuleName)
				if err != nil {
					return nil, err
				}
			}
		case reflect.Slice:
			// The only time we can have a slice within a leaf-list is when
			// the type of the field is a binary - such that we have a [][]byte field.
			if e.Type().Name() != BinaryTypeName {
				return nil, fmt.Errorf("unknown type within a slice: %v", e.Type().Name())
			}
			sval = append(sval, e.Bytes())
		default:
			return nil, fmt.Errorf("invalid type %s in leaflist", e.Kind())
		}
	}
	return sval, nil
}

// appendTypedValue takes an input reflect.Value and typecasts it to
// be appended the supplied slice of empty interfaces. If appendModuleName
// is set to true, the module name is prepended to the string value of
// any identity encountered when appending.
func appendTypedValue(l []interface{}, v reflect.Value, appendModuleName bool) ([]interface{}, error) {
	ival := v.Interface()
	switch reflect.TypeOf(ival).Kind() {
	case reflect.String:
		return append(l, ival.(string)), nil
	case reflect.Int8:
		return append(l, ival.(int8)), nil
	case reflect.Int16:
		return append(l, ival.(int16)), nil
	case reflect.Int32:
		return append(l, ival.(int32)), nil
	case reflect.Int:
		return append(l, ival.(int)), nil
	case reflect.Int64:
		if _, ok := ival.(GoEnum); ok {
			name, _, err := enumFieldToString(v, appendModuleName)
			if err != nil {
				return nil, err
			}
			return append(l, name), nil
		}
		return append(l, ival.(int64)), nil
	case reflect.Uint8:
		return append(l, ival.(uint8)), nil
	case reflect.Uint16:
		return append(l, ival.(uint16)), nil
	case reflect.Uint32:
		return append(l, ival.(uint32)), nil
	case reflect.Uint64, reflect.Uint:
		return append(l, ival.(uint64)), nil
	case reflect.Float32:
		return append(l, ival.(float32)), nil
	case reflect.Float64:
		return append(l, ival.(float64)), nil
	case reflect.Bool:
		return append(l, ival.(bool)), nil
	case reflect.Slice:
		if v.Type().Name() != BinaryTypeName {
			return nil, fmt.Errorf("unknown type within a slice: %v", v.Type().Name())
		}
		return append(l, v.Bytes()), nil
	}
	return nil, fmt.Errorf("unknown kind in leaflist: %v", reflect.TypeOf(v).Kind())
}

// RFC7951JSONConfig is used to control the behaviour of how
// RFC7951 JSON is output by the ygot library.
type RFC7951JSONConfig struct {
	// AppendModuleName determines whether the module name is appended to
	// elements that are defined within a different YANG module than their
	// parent.
	AppendModuleName bool
}

// ConstructIETFJSON marshals a supplied GoStruct to a map, suitable for
// handing to json.Marshal. It complies with the convention for marshalling
// to JSON described by RFC7951. The appendModName argument determines whether
// the module name should be appended to entities that are defined in a different
// module to their parent.
func ConstructIETFJSON(s GoStruct, args *RFC7951JSONConfig) (map[string]interface{}, error) {
	return constructJSON(s, "", jsonOutputConfig{
		jType:         RFC7951,
		rfc7951Config: args,
	})
}

// ConstructInternalJSON marshals a supplied GoStruct to a map, suitable for handing
// to json.Marshal. It uses the loosely specified JSON format document in
// go/yang-internal-json.
func ConstructInternalJSON(s GoStruct) (map[string]interface{}, error) {
	return constructJSON(s, "", jsonOutputConfig{
		jType: Internal,
	})
}

// jsonOutputConfig is used to determine how constructJSON should generate
// JSON.
type jsonOutputConfig struct {
	// jType specifies the format of JSON to be output (IETF RFC7951 vs. proprietary).
	jType JSONFormat
	// rfc7951Config stores the configuration to be used when outputting RFC7951
	// JSON.
	rfc7951Config *RFC7951JSONConfig
}

// constructJSON marshals a GoStruct to a map[string]interface{} which can be
// handed to JSON marshal. parentMod specifies the module that the supplied
// GoStruct is defined within such that RFC7951 format JSON is able to consider
// whether to append the name of the module to an element. The format of JSON to
// be produced and whether such module names are appended is controlled through the
// supplied jsonOutputConfig. Returns an error if the GoStruct cannot be rendered
// to JSON.
func constructJSON(s GoStruct, parentMod string, args jsonOutputConfig) (map[string]interface{}, error) {
	var errs errlist.List

	sval := reflect.ValueOf(s).Elem()
	stype := sval.Type()

	// Marshal into a map[string]interface{} which can be handed to
	// json.Marshal(Text)?
	jsonout := map[string]interface{}{}

	for i := 0; i < sval.NumField(); i++ {
		field := sval.Field(i)
		fType := stype.Field(i)

		// Determine whether we should append a module name to the path in RFC7951
		// output mode.
		var appmod string
		pmod := parentMod
		if chMod, ok := fType.Tag.Lookup("module"); ok {
			// If the child module isn't the same as the parent module,
			// then appmod stores the name of the module to prefix to paths
			// within this context.
			if chMod != parentMod {
				appmod = chMod
			}
			// Update the parent module name to be used for subsequent
			// children.
			pmod = chMod
		}

		var appendModName bool
		if args.jType == RFC7951 && args.rfc7951Config != nil && args.rfc7951Config.AppendModuleName && appmod != "" {
			appendModName = true
		}

		mapPaths, err := structTagToLibPaths(fType, newStringSliceGNMIPath([]string{}))
		if err != nil {
			errs.Add(fmt.Errorf("%s: %v", fType.Name, err))
			continue
		}

		value, err := constructJSONValue(field, pmod, args)
		if err != nil {
			errs.Add(err)
			continue
		}

		if value == nil {
			continue
		}

		if mp, ok := value.(map[string]interface{}); ok && len(mp) == 0 {
			continue
		}

		for _, p := range mapPaths {
			v, ok := value.(map[string]interface{})
			switch p.Len() {
			case 0:
				if ok {
					for mk, mv := range v {
						k := mk
						if appendModName {
							// Append the module name for the 0th element if
							// specified to do so. This is the module name of
							// the root.
							k = fmt.Sprintf("%s:%s", appmod, mk)
						}
						jsonout[k] = mv
					}
				} else {
					errs.Add(fmt.Errorf("empty path specified for non-root entity"))
					continue
				}
			case 1:
				pelem, err := p.StringElemAt(0)
				if err != nil {
					errs.Add(err)
					continue
				}
				if appendModName {
					pelem = fmt.Sprintf("%s:%s", appmod, pelem)
				}
				jsonout[pelem] = value
			default:
				var nilParent bool
				parent := jsonout
				for i := 0; i < p.Len()-1; i++ {
					k, err := p.StringElemAt(i)
					if err != nil {
						errs.Add(err)
						continue
					}

					switch {
					case (i == 0 && appendModName && !p.isAbsolute):
						// If the path is not absolute, then path compression has
						// occurred - and therefore the elements must be in the
						// same module. In this case, we append the module name
						// to the first element in the list.
						fallthrough
					case i == 0 && appendModName && parentMod == "":
						// For the first element, regardless of whether the path
						// was absolute or not, we always must append the module
						// name if there was no parent module, since this is an
						// entity at the root.
						k = fmt.Sprintf("%s:%s", appmod, k)
						nilParent = true
					}
					if _, ok := parent[k]; !ok {
						parent[k] = map[string]interface{}{}
					}
					parent = parent[k].(map[string]interface{})
				}
				k, err := p.LastStringElem()
				if err != nil {
					errs.Add(err)
					continue
				}
				if p.isAbsolute && appendModName && !nilParent {
					// If the path was not absolute, then we need to prepend the
					// module name since the last entity in the path was in a
					// different module to its parent. We do not need to check the
					// values of the module names, since in the case that the
					// module is the same appendModName is false.
					//
					// In the case that the parent was nil, then we must not
					// append the name here, since we must be within the same
					// module.
					k = fmt.Sprintf("%s:%s", appmod, k)
				}
				parent[k] = value

			}
		}
	}

	if errs.Err() != nil {
		return nil, errs.Err()
	}

	return jsonout, nil
}

// writeIETFScalarJSON takes an input scalar value, and returns it in the format
// that is expected in IETF RFC7951 JSON. Per this specification, uint64, int64
// and float64 values are represented as strings.
func writeIETFScalarJSON(i interface{}) interface{} {
	switch reflect.ValueOf(i).Kind() {
	case reflect.Uint64, reflect.Int64, reflect.Float64:
		return fmt.Sprintf("%v", i)
	}
	return i
}

// keyValue takes an input reflect.Value and returns its representation when used
// in a key for a YANG list. If the value is an enumerated type then its string
// representation is returned, otherwise the value is returned as an interface{}.
// If appendModuleName is set to true keys that are identity values in the YANG
// schema are prepended with the module that defines them.
func keyValue(v reflect.Value, appendModuleName bool) (interface{}, error) {
	if _, isEnum := v.Interface().(GoEnum); !isEnum {
		return v.Interface(), nil
	}

	name, _, err := enumFieldToString(v, appendModuleName)
	if err != nil {
		return nil, err
	}

	return name, nil
}

// constructMapJSON takes an input reflect.Value containing a map, and
// constructs the representation for JSON marshalling that corresponds to it.
// The module within which the map is defined is specified by the parentMod
// argument.
func constructMapJSON(field reflect.Value, parentMod string, args jsonOutputConfig) (interface{}, error) {
	var errs errlist.List
	mapKeyMap := map[string]reflect.Value{}
	// Order of elements determines the order in which keys will be processed.
	var mapKeys []string
	switch args.jType {
	case RFC7951:
		// YANG lists are marshalled into a JSON object array for IETF
		// JSON. We handle the keys in alphabetical order to ensure that
		// deterministic ordering is achieved in the output JSON.
		for _, k := range field.MapKeys() {
			kn := fmt.Sprintf("%v", k.Interface())
			mapKeys = append(mapKeys, kn)
			mapKeyMap[kn] = k
		}
	case Internal:
		// In non-IETF JSON, then we output a list as a JSON object. The keys
		// are stored as strings.
		for _, k := range field.MapKeys() {
			var kn string
			switch k.Kind() {
			case reflect.Struct:
				// Handle the case of a multikey list.
				var kp []string
				for j := 0; j < k.NumField(); j++ {
					keyval, err := keyValue(k.Field(j), false)
					if err != nil {
						errs.Add(fmt.Errorf("invalid enumerated key: %v", err))
						continue
					}
					kp = append(kp, fmt.Sprintf("%v", keyval))
				}
				kn = strings.Join(kp, " ")
			case reflect.Int64:
				keyval, err := keyValue(k, false)
				if err != nil {
					errs.Add(fmt.Errorf("invalid enumerated key: %v", err))
					continue
				}
				kn = fmt.Sprintf("%v", keyval)
			default:
				kn = fmt.Sprintf("%v", k.Interface())
			}
			mapKeys = append(mapKeys, kn)
			mapKeyMap[kn] = k
		}
	default:
		return nil, fmt.Errorf("unknown JSON type: %v", args.jType)
	}
	sort.Strings(mapKeys)

	if len(mapKeys) == 0 {
		return nil, nil
	}

	// Build the output that we expect. Since there is a difference between the IETF
	// and non-IETF forms, we simply choose vals to be interface{}, and then type assert
	// it later on. Since t cannot mutuate through this function we can guarantee that
	// the type assertions below will not cause panic, since we ensure that we know
	// what type of serialisation we're doing when we set the type.
	var vals interface{}
	switch args.jType {
	case RFC7951:
		vals = []interface{}{}
	case Internal:
		vals = map[string]interface{}{}
	default:
		return nil, fmt.Errorf("invalid JSON format specified: %v", args.jType)
	}
	for _, kn := range mapKeys {
		k := mapKeyMap[kn]
		goStruct, ok := field.MapIndex(k).Interface().(GoStruct)
		if !ok {
			errs.Add(fmt.Errorf("cannot map struct %v, invalid GoStruct", field))
			continue
		}

		val, err := constructJSON(goStruct, parentMod, args)
		if err != nil {
			errs.Add(err)
			continue
		}

		switch args.jType {
		case RFC7951:
			vals = append(vals.([]interface{}), val)
		case Internal:
			vals.(map[string]interface{})[kn] = val
		default:
			errs.Add(fmt.Errorf("invalid JSON type: %v", args.jType))
			continue
		}
	}

	if errs.Err() != nil {
		return nil, errs.Err()
	}
	return vals, nil
}

// constructJSONValue takes a reflect.Value which represents a struct field and
// constructs the representation that can be used to marshal the field to JSON.
// The module within which the value is defined is specified by the parentMod string,
// and the type of JSON to be rendered controlled by the value of the jsonOutputConfig
// provided. Returns an error if one occurs during the mapping process.
func constructJSONValue(field reflect.Value, parentMod string, args jsonOutputConfig) (interface{}, error) {
	var value interface{}
	var errs errlist.List

	switch field.Kind() {
	case reflect.Map, reflect.Slice, reflect.Ptr, reflect.Interface:
		if field.IsNil() {
			return nil, nil
		}
	}

	appmod := false
	if args.rfc7951Config != nil {
		appmod = args.rfc7951Config.AppendModuleName
	}

	switch field.Kind() {
	case reflect.Map:
		var err error
		value, err = constructMapJSON(field, parentMod, args)
		if err != nil {
			errs.Add(err)
		}
	case reflect.Ptr:
		switch field.Elem().Kind() {
		case reflect.Struct:
			goStruct, ok := field.Interface().(GoStruct)
			if !ok {
				return nil, fmt.Errorf("cannot map struct %v, invalid GoStruct", field)
			}

			var err error
			value, err = constructJSON(goStruct, parentMod, args)
			if err != nil {
				errs.Add(err)
			}
		default:
			value = field.Elem().Interface()
			if args.jType == RFC7951 {
				value = writeIETFScalarJSON(value)
			}
		}
	case reflect.Slice:
		var err error
		value, err = constructJSONSlice(field, parentMod, args)
		if err != nil {
			return nil, err
		}
	case reflect.Int64:
		// Enumerated values are represented as int64 in the generated Go structures.
		// For output, we map the enumerated value to the string name of the enum.
		v, set, err := enumFieldToString(field, appmod)
		if err != nil {
			return nil, err
		}

		// Skip if the enum has not been explicitly set in the schema.
		if !set {
			return nil, nil
		}
		value = v
	case reflect.Interface:
		// Union values that have more than one type are represented as a pointer to
		// an interface in the generated Go structures - extract the relevant value
		// and return this.
		var err error
		value, err = unionInterfaceValue(field, appmod)
		if err != nil {
			return nil, err
		}
		if args.jType == RFC7951 {
			value = writeIETFScalarJSON(value)
		}
	case reflect.Bool:
		// A non-pointer field of type boolean is an empty leaf within the YANG schema.
		// For RFC7951 this is represented as a null JSON array (i.e., [null]). For internal
		// JSON if the leaf is present and set, it is rendered as 'true', or as nil otherwise.
		switch {
		case args.jType == RFC7951 && field.Type().Name() == EmptyTypeName && field.Bool():
			value = []interface{}{nil}
		case field.Bool():
			value = true
		}
	default:
		return nil, fmt.Errorf("got unexpected field type, was: %v", field.Kind())
	}

	if errs.Err() != nil {
		return nil, errs.Err()
	}
	return value, nil
}

// constructJSONSlice takes an input reflect.Value containing a slice, and
// outputs the JSON that corresponds to it in the requested JSON format. In a
// GoStruct, a slice may be a binary field, leaf-list or an unkeyed list. The
// parentMod is used to track the name of the parent module in the case that
// module names should be appended.
func constructJSONSlice(field reflect.Value, parentMod string, args jsonOutputConfig) (interface{}, error) {
	if field.Type().Name() == BinaryTypeName {
		// Handle the case that that we have a Binary ([]byte) value,
		// which must be returned as a JSON string.
		return binaryBase64(field.Bytes()), nil
	}

	// In the case that the field is a slice of struct pointers then this
	// was an unkeyed YANG list.
	if c := field.Type().Elem(); util.IsTypeStructPtr(c) {
		vals := []interface{}{}
		for i := 0; i < field.Len(); i++ {
			gs, ok := field.Index(i).Interface().(GoStruct)
			if !ok {
				return nil, fmt.Errorf("invalid member of a slice, %s was not a valid GoStruct", c.Name())
			}
			j, err := constructJSON(gs, parentMod, args)
			if err != nil {
				return nil, err
			}
			vals = append(vals, j)
		}

		return vals, nil
	}

	appmod := false
	if args.rfc7951Config != nil {
		appmod = args.rfc7951Config.AppendModuleName
	}
	sl, err := leaflistToSlice(field, appmod)
	if err != nil {
		return nil, fmt.Errorf("could not map slice (leaf-list or unkeyed list): %v", err)
	}
	for j, e := range sl {
		switch {
		case reflect.TypeOf(e).Kind() == reflect.Slice:
			// This is a slice within a slice which can only be a binary value,
			// so we base64 encode it.
			sl[j] = binaryBase64(reflect.ValueOf(e).Bytes())
		case args.jType == RFC7951:
			sl[j] = writeIETFScalarJSON(e)
		}
	}
	return sl, nil
}

// unionInterfaceValue takes an input reflect.Value which must contain
// an interface Value, and resolves it from the generated union struct to
// the value which should be used for the YANG leaf.
//
// In a generated GoStruct, a union with more than one type is implemented
// as an interface which is implemented by the types that are valid for the
// union, for example:
//
//	container foo {
//		leaf bar {
//			type union {
//				type string;
//				type int32;
//			}
//		}
//	}
//
// Is mapped to:
//
//	type Foo struct {
//		Bar Foo_Bar_Union `path:"bar"`
//	}
//
//	type Foo_Bar_Union interface {
//		Is_Foo_Bar_Union()
//	}
//
//	type Foo_Bar_Union_String struct {
//		String string
//	}
//	func (*Foo_Bar_Union_String) Is_Foo_Bar_Union() {}
//
//	type Foo_Bar_Union_Int32 struct {
//		Int32 int32
//	}
//	func (*Foo_Bar_Union_Int32) Is_Foo_Bar_Union() {}
//
// This function extracts field index 0 of the struct within the interface and returns
// the value.
func unionInterfaceValue(v reflect.Value, appendModuleName bool) (interface{}, error) {
	if !util.IsValueInterfaceToStructPtr(v) {
		return nil, fmt.Errorf("received a union type which was invalid: %v", v.Kind())
	}

	s := v.Elem().Elem() // Dereference the struct ptr.

	if !util.IsStructValueWithNFields(s, 1) {
		return nil, fmt.Errorf("received a union type which did not have one field, had: %v", v.Elem().Elem().NumField())
	}

	return resolveUnionVal(s.Field(0).Interface(), appendModuleName)
}

// unionPtrValue returns the value of a union when it is stored as a pointer. The
// type of the union field is as per the description in unionInterfaceValue. Union
// pointer values are used when a list is keyed by a union.
func unionPtrValue(v reflect.Value, appendModuleName bool) (interface{}, error) {
	if !util.IsValueStructPtr(v) {
		return nil, fmt.Errorf("received a union pointer that didn't contain a struct, got: %v", v.Kind())
	}

	if !util.IsStructValueWithNFields(v.Elem(), 1) {
		return nil, fmt.Errorf("received a union pointer struct that didn't have one field, got: %v", v.Elem().NumField())
	}

	return resolveUnionVal(v.Elem().Field(0).Interface(), appendModuleName)
}

// resolveUnionVal returns the value of a field in a union, resolving it to its
// the relevant type where required.
func resolveUnionVal(v interface{}, appendModuleName bool) (interface{}, error) {
	if _, isEnum := v.(GoEnum); isEnum {
		var err error
		v, _, err = enumFieldToString(reflect.ValueOf(v), appendModuleName)
		if err != nil {
			return nil, err
		}
	}
	return v, nil
}
