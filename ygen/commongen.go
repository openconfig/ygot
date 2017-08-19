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
	"runtime"
)

// currentBinaryName returns the name of the Go binary that is currently
// running.
func callerName() string {
	// Find out the name of this binary so that it can be included in the
	// generated code for debug reasons. It is dynamically learnt based on
	// review suggestions that this code may move in the future.
	_, currentCodeFile, _, ok := runtime.Caller(0)
	if !ok {
		// In the case that we cannot determine the current running binary's name
		// this is non-fatal, so return a default string.
		return "codegen"
	}
	return currentCodeFile
}
