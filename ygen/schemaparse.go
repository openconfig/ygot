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

package ygen

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
)

// buildJSON tree takes an input set of modules expressed as a slice of yang.Entry
// structs, the names of directories within the generated code, and a yang.Entry
// which corresponds to the fake root, and a boolean indicating whether the current
// code generation is compressing paths, and returns a JSON serialised tree of the
// schema for the set of modules. The JSON document that is returned is always
// rooted on a yang.Entry which corresponds to the root item, and stores all
// root-level enties (and their subtrees) within the input module set. All
// YANG directories are annotated in the output JSON with the name of the type
// they correspond to in the generated code, and the absolute schema path that
// the entry corresponds to. In the case that the fake root struct that is provided
// is nil, a synthetic root entry is used to store the schema tree.
func buildJSONTree(ms []*yang.Entry, dn map[string]string, fakeroot *yang.Entry, compressed bool) ([]byte, error) {
	rootEntry := &yang.Entry{
		Dir:        map[string]*yang.Entry{},
		Annotation: map[string]interface{}{},
	}
	for _, m := range ms {
		annotateChildren(m, dn)
		for _, ch := range children(m) {
			if _, ex := rootEntry.Dir[ch.Name]; ex {
				return nil, fmt.Errorf("overlapping root children for key %s", ch.Name)
			}
			rootEntry.Dir[ch.Name] = ch
		}
	}

	if fakeroot != nil {
		rootEntry.Name = fakeroot.Name
		rootEntry.Annotation["schemapath"] = "/"
		rootEntry.Annotation["structname"] = dn[fakeroot.Path()]
		rootEntry.Kind = yang.DirectoryEntry
	}
	// Always annotate the root as a fake root, so that it is not treated
	// as a path element in ytypes.
	rootEntry.Annotation["isFakeRoot"] = true

	// Annotate the root indicating that compression was enabled.
	if compressed {
		rootEntry.Annotation[util.CompressedSchemaAnnotation] = compressed
	}

	j, err := json.MarshalIndent(rootEntry, "", strings.Repeat(" ", 4))
	if err != nil {
		return nil, fmt.Errorf("JSON marshalling error: %v", err)
	}
	if len(j) == 0 {
		return nil, nil
	}
	return j, nil
}

// annotateChildren annotates the children of e with their schema path, and the value corresponding
// to its path in the supplied dn map. The dn map is assumed to contain the
// names of unique directories that are generated within the code to be output.
// The children of e are recursively annotated.
func annotateChildren(e *yang.Entry, dn map[string]string) {
	annotateEntry(e, dn)
	for _, ch := range children(e) {
		annotateEntry(ch, dn)
		if ch.IsDir() {
			ch.Annotation["schemapath"] = ch.Path()
			// Recurse to annotate the children of this entry.
			annotateChildren(ch, dn)
		}
	}
}

// annotateEntry modifies the yang.Entry e to:
//  - set the description to be the nil string to reduce
//    filesizes of serialised schemas.
//  - add the struct name corresponding to the path of the entry
//    in the supplied dn map to the annotations.
//  - add the YANG schema path to the annotations, where e
//    corresponds to a YANG directory.
func annotateEntry(e *yang.Entry, dn map[string]string) {
	e.Description = ""
	if e.Annotation == nil {
		e.Annotation = map[string]interface{}{}
	}
	if n, ok := dn[e.Path()]; ok {
		e.Annotation["structname"] = n
	}
	if e.IsDir() {
		e.Annotation["schemapath"] = e.Path()
	}
}

// WriteGzippedByteSlice takes an input slice of bytes, gzips it
// and returns the resulting compressed output as a byte slice.
func WriteGzippedByteSlice(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}

	if _, err := gzw.Write(b); err != nil {
		return nil, err
	}
	gzw.Flush()
	gzw.Close()

	return buf.Bytes(), nil
}

// BytesToGoByteSlice takes an input slice of bytes and outputs it
// as a slice of strings corresponding to lines of Go source code
// that define the byte slice. Each string within the slice contains
// up to 16 bytes of the output byte array, with each byte represented
// as a two digit hex character.
func BytesToGoByteSlice(b []byte) []string {
	var lines []string
	var bstr bytes.Buffer
	for i := 0; i < len(b); i += 16 {
		e := i + 16
		if e > len(b) {
			e = len(b)
		}
		for j, c := range b[i:e] {
			bstr.WriteString(fmt.Sprintf("0x%02x,", c))
			// Only write a space if we are not at the end of a line, or
			// at the very end of the byte slice.
			if j != 15 && i+j != len(b)-1 {
				bstr.WriteString(" ")
			}
		}
		lines = append(lines, bstr.String())
		bstr.Reset()
	}

	return lines
}
