package integration_tests

import (
	"path/filepath"
	"testing"

	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/ygot/gogen"
	"github.com/openconfig/ygot/protogen"
	"github.com/openconfig/ygot/ygen"
)

func TestGenerateErrs(t *testing.T) {
	tests := []struct {
		name                  string
		inFiles               []string
		inPath                []string
		inIROptions           ygen.IROptions
		wantGoOK              bool
		wantGoErrSubstring    string
		wantProtoOK           bool
		wantProtoErrSubstring string
		wantSameErrSubstring  bool
	}{{
		name:                 "missing YANG file",
		inFiles:              []string{filepath.Join("testdata", "errors", "doesnt-exist.yang")},
		wantGoErrSubstring:   "no such file",
		wantSameErrSubstring: true,
	}, {
		name:                 "bad YANG file",
		inFiles:              []string{filepath.Join("testdata", "errors", "bad-module.yang")},
		wantGoErrSubstring:   "syntax error",
		wantSameErrSubstring: true,
	}, {
		name:                 "missing import due to path",
		inFiles:              []string{filepath.Join("testdata", "errors", "missing-import.yang")},
		wantGoErrSubstring:   "no such module",
		wantSameErrSubstring: true,
	}, {
		name:        "import satisfied due to path",
		inFiles:     []string{filepath.Join("testdata", "errors", "missing-import.yang")},
		inPath:      []string{filepath.Join("testdata", "errors", "subdir")},
		wantGoOK:    true,
		wantProtoOK: true,
	}}

	for _, tt := range tests {
		gcg := gogen.New("", tt.inIROptions, gogen.GoOpts{})

		_, goErr := gcg.Generate(tt.inFiles, tt.inPath)
		switch {
		case tt.wantGoOK && goErr != nil:
			t.Errorf("%s: gcg.GenerateGoCode(%v, %v): got unexpected error, got: %v, want: nil", tt.name, tt.inFiles, tt.inPath, goErr)
		case tt.wantGoOK:
		default:
			if diff := errdiff.Substring(goErr, tt.wantGoErrSubstring); diff != "" {
				t.Errorf("%s: gcg.GenerateGoCode(%v, %v): %v", tt.name, tt.inFiles, tt.inPath, diff)
			}
		}

		pcg := protogen.New("", tt.inIROptions, protogen.ProtoOpts{})

		if tt.wantSameErrSubstring {
			tt.wantProtoErrSubstring = tt.wantGoErrSubstring
		}

		_, protoErr := pcg.Generate(tt.inFiles, tt.inPath)
		switch {
		case tt.wantProtoOK && protoErr != nil:
			t.Errorf("%s: pcg.Generate(%v, %v): got unexpected error, got: %v, want: nil", tt.name, tt.inFiles, tt.inPath, protoErr)
		case tt.wantProtoOK:
		default:
			if diff := errdiff.Substring(protoErr, tt.wantProtoErrSubstring); diff != "" {
				t.Errorf("%s: pcg.Generate(%v, %v): %v", tt.name, tt.inFiles, tt.inPath, diff)
			}
		}

	}
}
