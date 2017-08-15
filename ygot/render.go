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
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/openconfig/gnmi/errlist"
	"github.com/openconfig/gnmi/value"

	log "github.com/golang/glog"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

const (
	// BinaryTypeName is the name of the type that is used for YANG
	// binary fields in the output structs.
	BinaryTypeName string = "Binary"
)

// path stores the elements of a path for a particular leaf,
// such that it can be used as a key for maps.
type path struct {
	p []interface{}
}

// TogNMINotifications takes an input GoStruct and renders it to slice of
// Notification messages, marked with the specified timestamp. If a parentPath
// is used, it is used as a prefix path for the notifications returned.
func TogNMINotifications(s GoStruct, ts int64, prefix []interface{}) ([]*gnmipb.Notification, error) {
	leaves := map[*path]interface{}{}
	if err := findUpdatedLeaves(leaves, s, prefix); err != nil {
		return nil, err
	}

	msgs, err := leavesToNotifications(leaves, ts, prefix)
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
func findUpdatedLeaves(leaves map[*path]interface{}, s GoStruct, parentPath []interface{}) error {
	var errs errlist.List

	if s == nil {
		errs.Add(fmt.Errorf("input struct for %v was nil", parentPath))
		return errs.Err()
	}

	sval := reflect.ValueOf(s).Elem()
	stype := sval.Type()

	for i := 0; i < sval.NumField(); i++ {
		fval := sval.Field(i)
		ftype := stype.Field(i)

		mapPaths, err := structTagToLibPaths(ftype, parentPath)
		if err != nil {
			errs.Add(fmt.Errorf("%s->%s: %v", parentPath, ftype.Name, err))
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
				// mapPaths can only be one element long for a YANG list.
				keyval := k.Interface()
				if _, isEnum := keyval.(GoEnum); isEnum {
					name, _, err := enumFieldToString(k, false)
					if err != nil {
						errs.Add(fmt.Errorf("invalid enumerated key for %v: %v", mapPaths[0], err))
						continue
					}
					keyval = interface{}(name)
				}
				childPath := append(mapPaths[0], keyval)
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
			val, err := unionInterfaceValue(fval)
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

// interfacePathAsgNMIPath takes a path that is specified as a slice of empty interfaces
// and populates a gNMI path message with the string components. It should be noted that this
// functionality does not comply with the gNMI specification, and should be updated in the
// future.
// TODO(robjs): Update this functionality based on adoption of the gNMI structured paths
// when Pictor is ready to support these.
func interfacePathAsgNMIPath(p []interface{}) *gnmipb.Path {
	pfx := &gnmipb.Path{}
	for _, e := range p {
		pfx.Element = append(pfx.Element, fmt.Sprintf("%v", e))
	}
	return pfx
}

// stripPrefix removes the specified prefix from the provided path. Returns an error if
// the prefix is not a valid prefix of path.
func stripPrefix(path []interface{}, prefix []interface{}) ([]interface{}, error) {
	for i := range prefix {
		if path[i] != prefix[i] {
			return nil, fmt.Errorf("path %v does not have prefix %v", path, prefix)
		}
	}

	return path[len(prefix):], nil
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
func leavesToNotifications(leaves map[*path]interface{}, ts int64, prefix []interface{}) ([]*gnmipb.Notification, error) {
	n := &gnmipb.Notification{
		Timestamp: ts,
	}

	pfx := interfacePathAsgNMIPath(prefix)
	if len(pfx.Element) > 0 {
		n.Prefix = pfx
	}

	for p, v := range leaves {
		path, err := stripPrefix(p.p, prefix)

		for _, p := range path {
			if p == nil {
				log.Infof("leavesToNotifications got nil in path: %v", path)
			}
		}

		if err != nil {
			return nil, err
		}

		u := &gnmipb.Update{
			Path: interfacePathAsgNMIPath(path),
		}

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
				uval, err := unionInterfaceValue(e)
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

		mapPaths, err := structTagToLibPaths(fType, nil)
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
			switch len(p) {
			case 0:
				if ok {
					// Handle the case that the path is empty, used by the default
					// struct.
					for mk, mv := range v {
						k := mk
						if appendModName {
							k = fmt.Sprintf("%s:%s", appmod, mk)
						}
						jsonout[k] = mv
					}
				} else {
					errs.Add(fmt.Errorf("empty path specified for non-root entity"))
					continue
				}
			case 1:
				pelem, ok := p[0].(string)
				if !ok {
					errs.Add(fmt.Errorf("could not convert path %v into a string", p))
					continue
				}
				if appendModName {
					pelem = fmt.Sprintf("%s:%s", appmod, pelem)
				}
				jsonout[pelem] = value
			default:
				parent := jsonout
				for i := 0; i < len(p)-1; i++ {
					k, ok := p[i].(string)
					if !ok {
						errs.Add(fmt.Errorf("could not convert path %v into a string for %v", p[i], p))
						continue
					}

					// For the 0th element, append the module name if it differs to the
					// parent. All schema compression that is within a GoStruct is intra-module.
					if i == 0 && appendModName {
						k = fmt.Sprintf("%s:%s", appmod, k)
					}
					if _, ok := parent[k]; !ok {
						parent[k] = map[string]interface{}{}
					}
					parent = parent[k].(map[string]interface{})
				}
				k, ok := p[len(p)-1].(string)
				if !ok {
					errs.Add(fmt.Errorf("could not convert path element %v into a string for %v", p[len(p)-1], p))
					continue
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
		value, err = constructJSONSlice(field, args)
		if err != nil {
			return nil, err
		}
	case reflect.Int64:
		// Enumerated values are represented as int64 in the generated Go structures.
		// For output, we map the enumerated value to the string name of the enum.
		appmod := false
		if args.rfc7951Config != nil {
			appmod = args.rfc7951Config.AppendModuleName
		}
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
		value, err = unionInterfaceValue(field)
		if err != nil {
			return nil, err
		}
		if args.jType == RFC7951 {
			value = writeIETFScalarJSON(value)
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
// GoStruct, a slice may be a binary field, leaf-list or an unkeyed list.
func constructJSONSlice(field reflect.Value, args jsonOutputConfig) (interface{}, error) {
	if field.Type().Name() == BinaryTypeName {
		// Handle the case that that we have a Binary ([]byte) value,
		// which must be returned as a JSON string.
		return binaryBase64(field.Bytes()), nil
	}

	// TODO(robjs): Check for the case whereby we have an unkeyed list
	// and the child is a struct.
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
func unionInterfaceValue(v reflect.Value) (interface{}, error) {
	switch {
	case v.Kind() != reflect.Ptr && v.Kind() != reflect.Interface:
		return nil, fmt.Errorf("received a union type which was invalid: %v", v.Kind())
	case v.Elem().Kind() != reflect.Ptr:
		return nil, fmt.Errorf("received a union type which was not a pointer: %v", v.Kind())
	case v.Elem().Elem().Kind() != reflect.Struct:
		return nil, fmt.Errorf("received a union type that did not contain a struct: %v", v.Kind())
	case v.Elem().Elem().NumField() != 1:
		return nil, fmt.Errorf("received a union type which did not have one field, had: %v", v.Elem().Elem().NumField())
	}

	return v.Elem().Elem().Field(0).Interface(), nil
}
