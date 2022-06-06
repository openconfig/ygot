package gogen

import (
	"fmt"
	"strings"
)

// safeGoEnumeratedValueName takes an input string, which is the name of an
// enumerated value from a YANG schema, and ensures that it is safe to be
// output as part of the name of the enumerated value in the Go code. The
// sanitised value is returned.  Per RFC6020 Section 9.6.4,
// "The enum Statement [...] takes as an argument a string which is the
// assigned name. The string MUST NOT be empty and MUST NOT have any
// leading or trailing whitespace characters. The use of Unicode control
// codes SHOULD be avoided."
// Note: this rule is distinct and looser than the rule for YANG identifiers.
// The implementation used here replaces some (not all) characters allowed
// in a YANG enum assigned name but not in Go code. Current support is based
// on real-world feedback e.g. in OpenConfig schemas, there are currently
// a small number of identity values that contain "." and hence
// must be specifically handled.
func safeGoEnumeratedValueName(name string) string {
	// NewReplacer takes pairs of strings to be replaced in the form
	// old, new.
	replacer := strings.NewReplacer(
		".", "_",
		"-", "_",
		"/", "_",
		"+", "_PLUS",
		",", "_COMMA",
		"@", "_AT",
		"$", "_DOLLAR",
		"*", "_ASTERISK",
		":", "_COLON",
		" ", "_")
	return replacer.Replace(name)
}

// enumDefaultValue sanitises a default value specified for an enumeration
// which can be specified as prefix:value in the YANG schema. The baseName
// is used as the generated enumeration name stripping any prefix specified,
// (allowing removal of the enumeration type prefix if required). The default
// value in the form <sanitised_baseName>_<sanitised_defVal> is returned as
// a pointer.
func enumDefaultValue(baseName, defVal, prefix string) string {
	if strings.Contains(defVal, ":") {
		defVal = strings.Split(defVal, ":")[1]
	}

	if prefix != "" {
		baseName = strings.TrimPrefix(baseName, prefix)
	}

	defVal = safeGoEnumeratedValueName(defVal)

	return fmt.Sprintf("%s_%s", baseName, defVal)
}
