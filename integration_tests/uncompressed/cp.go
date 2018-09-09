// Copyright 2018 Google Inc.
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

// Package uncompressed is an integration test for ygot that tests features of uncompressed
// schemas using real YANG schemas.
package uncompressed

//go:generate sh -c "go run ../../generator/generator.go -path=yang -output_file=cschema/structs.go -package_name=cschema -generate_fakeroot -fakeroot_name=root -generate_getters -compress_paths yang/uncompressed.yang && go run ../../generator/generator.go -path=yang -output_file=uschema/structs.go -package_name=uschema -generate_fakeroot -fakeroot_name=root -generate_getters yang/uncompressed.yang"
