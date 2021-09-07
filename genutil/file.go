// Copyright 2019 Google Inc.
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

package genutil

import (
	"os"

	log "github.com/golang/glog"
)

// OpenFile opens a file with the supplied name, logging and exiting if it cannot
// be opened.
func OpenFile(fn string) *os.File {
	fileOut, err := os.Create(fn)
	if err != nil {
		log.Exitf("Error: could not open output file: %v\n", err)
	}
	return fileOut
}

// SyncFile synchronises the supplied os.File and closes it.
func SyncFile(fh *os.File) {
	if err := fh.Sync(); err != nil {
		log.Exitf("Error: could not sync file output: %v\n", err)
	}

	if err := fh.Close(); err != nil {
		log.Exitf("Error: could not close output file: %v\n", err)
	}
}
