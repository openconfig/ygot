package main

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/openconfig/ygot/ygen"
)

// Simple test case that ensures that the end-to-end ygot pipeline works
// correctly. This is a smoke-test for the ygot package.

// The path to this directory within the test package.
var TestRoot string

func TestGenerateCode(t *testing.T) {
	tests := []struct {
		name     string
		inConfig *ygen.GeneratorConfig
		inFiles  []string
		inPaths  []string
	}{{
		name: "openconfig interfaces",
		inConfig: &ygen.GeneratorConfig{
			CompressOCPaths:    true,
			ExcludeModules:     []string{"ietf-interfaces"},
			GenerateFakeRoot:   true,
			GenerateJSONSchema: true,
		},
		inFiles: []string{
			filepath.Join(TestRoot, "yang", "openconfig-interfaces.yang"),
			filepath.Join(TestRoot, "yang", "openconfig-if-ip.yang"),
		},
		inPaths: []string{filepath.Join(TestRoot, "yang")},
	}, {
		name: "openconfig interfaces with no compression",
		inConfig: &ygen.GeneratorConfig{
			CompressOCPaths:    false,
			ExcludeModules:     []string{"ietf-interfaces"},
			GenerateFakeRoot:   true,
			GenerateJSONSchema: true,
		},
		inFiles: []string{
			filepath.Join(TestRoot, "yang", "openconfig-interfaces.yang"),
			filepath.Join(TestRoot, "yang", "openconfig-if-ip.yang"),
		},
		inPaths: []string{filepath.Join(TestRoot, "yang")},
	}}

	for _, tt := range tests {
		cg := ygen.NewYANGCodeGenerator(tt.inConfig)
		got, err := cg.GenerateGoCode(tt.inFiles, tt.inPaths)
		if err != nil {
			t.Errorf("%s: GenerateGoCode(%v, %v): Config: %v, got unexpected error: %v", tt.name, tt.inFiles, tt.inPaths, tt.inConfig, err)
			continue
		}

		var b bytes.Buffer
		fmt.Fprintf(&b, got.Header)
		for _, s := range got.Structs {
			fmt.Fprintf(&b, s)
		}

		for _, e := range got.Enums {
			fmt.Fprintf(&b, e)
		}

		fmt.Fprintf(&b, got.EnumMap)
		fmt.Fprintf(&b, got.JSONSchemaCode)

		// Parse the generated code using the Go parser and check whether any errors
		// are returned.
		fset := token.NewFileSet()
		if _, err := parser.ParseFile(fset, "", b.String(), parser.AllErrors); err != nil {
			t.Errorf("%s: could not parse generated Go code: %v\n\n%s", tt.name, err, b.String())
		}
	}
}
