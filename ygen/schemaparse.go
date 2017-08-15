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
	"fmt"
)

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

	if bstr.Len() != 0 {
		lines = append(lines, bstr.String())
	}

	return lines
}
