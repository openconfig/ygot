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
	"sort"
	"strings"

	"github.com/openconfig/ygot/util"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

// PathToString takes a gNMI Path and provides its string representation. For example,
// the path Path{Element: []string{"one", "two", "three"} is converted to the string
// "/one/two/three" and returned. Both the pre-0.4.0 "element"-based paths, and the
//  >0.4.0 paths based on "elem" are supported. In the case that post-0.4.0 paths are
// specified, keys that are specified in the path are concatenated onto the name of
// the path element using the format [name=value]. If the path specifies both pre-
// and post-0.4.0 paths, the pre-0.4.0 version is returned.
func PathToString(path *gnmipb.Path) string {
	var buf bytes.Buffer
	buf.WriteRune('/')
	if path.Element != nil {
		for i, e := range path.Element {
			buf.WriteString(e)
			if i != len(path.Element)-1 {
				buf.WriteRune('/')
			}
		}
		return buf.String()
	}

	for i, e := range path.Elem {
		buf.WriteString(e.Name)
		if len(e.Key) != 0 {
			var keys []string
			for k := range e.Key {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				buf.WriteString(fmt.Sprintf("[%s=%s]", k, e.Key[k]))
			}
		}
		if i != len(path.Elem)-1 {
			buf.WriteRune('/')
		}
	}
	return buf.String()
}

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

// StringToPath takes an input string representing a path in gNMI, and converts
// it to a gNMI Path message, populated with the specified path encodings.
func StringToPath(path string, pathTypes ...PathType) (*gnmipb.Path, util.Errors) {
	if len(pathTypes) == 0 {
		return nil, []error{errors.New("no path types specified")}
	}

	pmsg := &gnmipb.Path{}
	var errs util.Errors
	for _, p := range pathTypes {
		switch p {
		case StructuredPath:
			gp, err := StringToStructuredPath(path)
			if err != nil {
				errs = append(errs, fmt.Errorf("error building structured path: %v", err))
				continue
			}
			pmsg.Elem = gp.Elem
		case StringSlicePath:
			gp, err := StringToStringSlicePath(path)
			if err != nil {
				errs = append(errs, fmt.Errorf("error building string slice path: %v", err))
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
// to a gnmi.Path{Element: []string{"a", "b[c=d]", "e"}} which is returned.
func StringToStringSlicePath(path string) (*gnmipb.Path, error) {
	parsedPath := &gnmipb.Path{}

	var inKey, inEscape bool
	var buf bytes.Buffer
	for i, ch := range path {
		switch ch {
		case '\\':
			if !inEscape {
				inEscape = true
				continue
			}
		case '[':
			if !inKey && !inEscape {
				inKey = true
			}
		case ']':
			if !inEscape {
				inKey = false
				if !strings.Contains(buf.String(), "=") {
					return nil, fmt.Errorf("key value %s does not contain a key=value pair", buf.String())
				}
			}
		case '/':
			if i == 0 {
				// Do not take the first path element if the path starts with a "/".
				continue
			}

			if !inEscape && !inKey {
				parsedPath.Element = append(parsedPath.Element, buf.String())
				buf.Reset()
				continue
			}
		}
		if inEscape {
			// An escape lasts a single character, so leave the escape if we parsed
			// a character.
			inEscape = false
		}
		buf.WriteRune(ch)
	}
	if buf.Len() != 0 {
		parsedPath.Element = append(parsedPath.Element, buf.String())
	}

	return parsedPath, nil
}

// StringToStructuredPath takes a string representing a path, and converts it to
// a gnmi.Path, using the PathElem element message that is defined in gNMI 0.4.0.
func StringToStructuredPath(path string) (*gnmipb.Path, error) {
	parsedPath := &gnmipb.Path{}

	var inKey, inEscape bool
	var buf bytes.Buffer
	keys := map[string]string{}
	var currentKey, currentName string
	for i, ch := range path {
		switch ch {
		case '\\':
			if !inEscape {
				inEscape = true
				continue
			}
		case '[':
			// If we have a key, then record the current element's name for
			// inclusion in the PathElem message.
			if !inKey && !inEscape {
				if len(keys) == 0 {
					// The first [ means that the current
					// buffer contents are the name of the
					// element. Store this.
					currentName = buf.String()
				}
				buf.Reset()
				inKey = true
				continue
			}
		case '=':
			// When we reach an = inside a key, which is note escaped then
			// we record the key's name.
			if inKey && !inEscape {
				currentKey = buf.String()
				buf.Reset()
				continue
			}
		case ']':
			if !inEscape {
				// If this ] is not escaped, then we have reached the end of a
				// key, and record its value.
				if currentKey == "" || buf.Len() == 0 {
					return nil, fmt.Errorf("received a key with no equals sign in it, key name: %s, key value: %s", currentKey, buf.String())
				}

				inKey = false
				keys[currentKey] = buf.String()
				buf.Reset()
				currentKey = ""
				continue
			}
		case '/':
			if i == 0 {
				continue
			}

			if !inEscape && !inKey {
				parsedPath.Elem = append(parsedPath.Elem, toPathElem(&buf, currentName, keys))
				keys = map[string]string{}
				currentKey, currentName = "", ""
				buf.Reset()
				continue
			}
		}

		if inEscape {
			inEscape = false
		}
		buf.WriteRune(ch)
	}

	// Deal with the last element
	parsedPath.Elem = append(parsedPath.Elem, toPathElem(&buf, currentName, keys))

	return parsedPath, nil
}

// toPathElem takes an input buffer, current name, and key map and returns them as a gNMI
// PathElem message.
func toPathElem(buf *bytes.Buffer, currentName string, keys map[string]string) *gnmipb.PathElem {
	if len(keys) == 0 {
		return &gnmipb.PathElem{
			Name: buf.String(),
		}
	}

	return &gnmipb.PathElem{
		Name: currentName,
		Key:  keys,
	}
}
