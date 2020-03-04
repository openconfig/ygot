package testutil

import (
	"strings"
	"testing"
)

func TestGenerateUnifiedDiff(t *testing.T) {
	tests := []struct {
		name           string
		inWant         string
		inGot          string
		wantDiffSubstr string
	}{{
		name:           "basic",
		inWant:         "hello, world!",
		inGot:          "Hello, world",
		wantDiffSubstr: "-hello, world!\n+Hello, world",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff, _ := GenerateUnifiedDiff(tt.inWant, tt.inGot); !strings.Contains(diff, tt.wantDiffSubstr) {
				t.Errorf("expected diff to contain %q\nbut got %q", tt.wantDiffSubstr, diff)
			}
		})
	}
}
