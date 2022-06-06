package gogen

import "testing"

// TestSafeGoEnumeratedValueName tests the safeGoEnumeratedValue function to ensure
// that enumeraton value names are correctly transformed to safe Go names.
func TestSafeGoEnumeratedValueName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"SPEED_2.5G", "SPEED_2_5G"},
		{"IPV4-UNICAST", "IPV4_UNICAST"},
		{"frameRelay", "frameRelay"},
		{"coffee", "coffee"},
		{"ethernetCsmacd", "ethernetCsmacd"},
		{"SFP+", "SFP_PLUS"},
		{"LEVEL1/2", "LEVEL1_2"},
		{"DAYS1-3", "DAYS1_3"},
		{"FISH CHIPS", "FISH_CHIPS"},
		{"FOO*", "FOO_ASTERISK"},
		{"FOO:", "FOO_COLON"},
		{",,FOO:@$,", "_COMMA_COMMAFOO_COLON_AT_DOLLAR_COMMA"},
	}

	for _, tt := range tests {
		got := safeGoEnumeratedValueName(tt.in)
		if got != tt.want {
			t.Errorf("safeGoEnumeratedValueName(%s): got: %s, want: %s", tt.in, got, tt.want)
		}
	}
}
