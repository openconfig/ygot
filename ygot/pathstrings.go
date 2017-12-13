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
	"bytes"
	"errors"
	"fmt"
	stdpath "path"
	"sort"
	"strings"

	"github.com/openconfig/ygot/util"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

// PathType is used to indicate a gNMI path type.
type PathType int64

const (
	// StructuredPath represents a Path using the structured 'PathElem' message in
	// the 'elem' field of the gNMI Path message.
	StructuredPath PathType = iota
	// StringSlicePath represents a Path using the 'Element' repeated string field
	// of the gNMI path message.
	StringSlicePath
)

// PathToString is like PathToStrings, but returns a single formatted string
// representing the path. Path is always treated as absolute.
func PathToString(path *gnmipb.Path) (string, error) {
	s, err := PathToStrings(path)
	return "/" + stdpath.Join(s...), err
}

// PathToStrings takes a gNMI Path and provides its string representation. For example,
// the path Path{Element: []string{"one", "two", "three"} is converted to the slice
// ["one", "two", "three"] and returned. Both the pre-0.4.0 "element"-based paths, and the
//  >0.4.0 paths based on "elem" are supported. In the case that post-0.4.0 paths are
// specified, keys that are specified in the path are concatenated onto the name of
// the path element using the format [name=value]. If the path specifies both pre-
// and post-0.4.0 paths, the pre-0.4.0 version is returned.
func PathToStrings(path *gnmipb.Path) ([]string, error) {
	var p []string
	if path.Element != nil {
		for i, e := range path.Element {
			if e == "" {
				return nil, fmt.Errorf("empty element at index %d in %v", i, path.Element)
			}
			p = append(p, e)
		}
		return p, nil
	}

	for i, e := range path.Elem {
		if e.Name == "" {
			return nil, fmt.Errorf("empty name for PathElem at index %d", i)
		}

		elem, err := elemToString(e.Name, e.Key)
		if err != nil {
			return nil, fmt.Errorf("failed formatting PathElem at index %d: %v", i, err)
		}
		p = append(p, elem)
	}
	return p, nil
}

// elemToString returns a formatted string representation of a single Path.Elem
// item. name and kv correspond to PathElem.Name and PathElem.Key.
func elemToString(name string, kv map[string]string) (string, error) {
	if name == "" {
		return "", errors.New("empty name for PathElem")
	}
	if len(kv) == 0 {
		return name, nil
	}

	var keys []string
	for k, v := range kv {
		if k == "" {
			return "", fmt.Errorf("empty key name (value: %s) in element %s", v, name)
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := strings.Replace(kv[k], `=`, `\=`, -1)
		v = strings.Replace(v, `]`, `\]`, -1)
		name = fmt.Sprintf("%s[%s=%s]", name, k, v)
	}

	return name, nil
}

// StringToPath takes an input string representing a path in gNMI, and converts
// it to a gNMI Path message, populated with the specified path encodings.
func StringToPath(path string, pathTypes ...PathType) (*gnmipb.Path, error) {
	var errs util.Errors
	if len(pathTypes) == 0 {
		return nil, util.AppendErr(errs, errors.New("no path types specified"))
	}

	pmsg := &gnmipb.Path{}
	for _, p := range pathTypes {
		switch p {
		case StructuredPath:
			gp, err := StringToStructuredPath(path)
			if err != nil {
				errs = util.AppendErr(errs, fmt.Errorf("error building structured path: %v", err))
				continue
			}
			pmsg.Elem = gp.Elem
		case StringSlicePath:
			gp, err := StringToStringSlicePath(path)
			if err != nil {
				errs = util.AppendErr(errs, fmt.Errorf("error building string slice path: %v", err))
				continue
			}
			pmsg.Element = gp.Element
		}
	}

	if errs != nil {
		return nil, errs
	}

	return pmsg, nil
}

// StringToStringSlicePath takes a string representing a path, and converts it into a
// gnmi.Path. For example, if the Path "/a/b[c=d]/e" is input, it is converted
// to a gnmi.Path{Element: []string{"a", "b[c=d]", "e"}} which is returned. Where there
// are complex predicates that are used within an element, they are not parsed and the
// contents is left unchanged. This implements the legacy string slice path that are
// used in gNMI pre-0.4.0. The specification for these paths is at https://goo.gl/uD6g6z.
func StringToStringSlicePath(path string) (*gnmipb.Path, error) {
	parts := pathStringToElements(path)
	gpath := new(gnmipb.Path)
	for _, p := range parts {
		// Run through extractKV to ensure that the path is valid.
		name, kv, err := extractKV(p)
		if err != nil {
			return nil, fmt.Errorf("error parsing path %q: %v", path, err)
		}
		fpath, err := elemToString(name, kv)
		if err != nil {
			return nil, fmt.Errorf("error formatting path %q: %v", path, err)
		}
		gpath.Element = append(gpath.Element, fpath)
	}

	return gpath, nil
}

// StringToStructuredPath takes a string representing a path, and converts it to
// a gnmi.Path, using the PathElem element message that is defined in gNMI 0.4.0.
func StringToStructuredPath(path string) (*gnmipb.Path, error) {
	parts := pathStringToElements(path)

	gpath := &gnmipb.Path{}
	for _, p := range parts {
		name, kv, err := extractKV(p)
		if err != nil {
			return nil, fmt.Errorf("error parsing path %s: %v", path, err)
		}
		gpath.Elem = append(gpath.Elem, &gnmipb.PathElem{
			Name: name,
			Key:  kv,
		})
	}
	return gpath, nil
}

// pathStringToElements splits the string s, which represents a gNMI string
// path into its constituent elements. It does not parse keys, which are left
// unchanged within the path - but removes escape characters from element
// names. The path returned omits any leading empty elements when splitting
// on the / character.
func pathStringToElements(s string) []string {
	var parts []string
	var buf bytes.Buffer

	var inKey, inEscape bool

	for _, ch := range s {
		switch {
		case ch == '[' && !inEscape:
			inKey = true
		case ch == ']' && !inEscape:
			inKey = false
		case ch == '\\' && !inEscape && !inKey:
			inEscape = true
			continue
		case ch == '/' && !inEscape && !inKey:
			parts = append(parts, buf.String())
			buf.Reset()
			continue
		}

		buf.WriteRune(ch)
		inEscape = false
	}

	if buf.Len() != 0 {
		parts = append(parts, buf.String())
	}

	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}

	return parts
}

// extractKV extracts key value predicates from the input string in. It returns
// the name of the element, a map keyed by key name with values of the predicates
// specified. It removes escape characters from keys and values where they are
// specified.
func extractKV(in string) (string, map[string]string, error) {
	var inEscape, inKey, inValue bool
	var name, currentKey string
	var buf bytes.Buffer
	keys := map[string]string{}

	for _, ch := range in {
		switch {
		case ch == '[' && !inEscape && !inValue && inKey:
			return "", nil, fmt.Errorf("received an unescaped [ in key of element %s", name)
		case ch == '[' && !inEscape && !inKey:
			inKey = true
			if len(keys) == 0 {
				if buf.Len() == 0 {
					return "", nil, errors.New("received a value when the element name was null")
				}
				name = buf.String()
				buf.Reset()
			}
			continue
		case ch == ']' && !inEscape && !inKey:
			return "", nil, fmt.Errorf("received an unescaped ] when not in a key for element %s", buf.String())
		case ch == ']' && !inEscape:
			inKey = false
			inValue = false
			if err := addKey(keys, name, currentKey, buf.String()); err != nil {
				return "", nil, err
			}
			buf.Reset()
			currentKey = ""
			continue
		case ch == '\\' && !inEscape:
			inEscape = true
			continue
		case ch == '=' && inKey && !inEscape && !inValue:
			currentKey = buf.String()
			buf.Reset()
			inValue = true
			continue
		}

		buf.WriteRune(ch)
		inEscape = false
	}

	if len(keys) == 0 {
		name = buf.String()
	}

	if len(keys) != 0 && buf.Len() != 0 {
		// In this case, we have trailing garbage following the key.
		return "", nil, fmt.Errorf("trailing garbage following keys in element %s, got: %v", name, buf.String())
	}

	if strings.Contains(name, " ") {
		return "", nil, fmt.Errorf("invalid space character included in element name '%s'", name)
	}

	return name, keys, nil
}

// addKey adds key k with value v to the key's map. The key, value pair is specified
// to be for an element named e.
func addKey(keys map[string]string, e, k, v string) error {
	switch {
	case strings.Contains(k, " "):
		return fmt.Errorf("received an invalid space in element %s key name '%s'", e, k)
	case e == "":
		return fmt.Errorf("received null element value with key and value %s=%s", k, v)
	case k == "":
		return fmt.Errorf("received null key name for element %s", e)
	case v == "":
		return fmt.Errorf("received null value for key %s of element %s", k, e)
	}
	keys[k] = v
	return nil
}
