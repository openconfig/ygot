package gogen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/ygot"
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
//
// Note that the string returned by this function is not guaranteed to be
// unique among the sibling values of the enumerated type that name belongs
// to -- e.g. "FOO.BAR" and "FOO-BAR" both sanitise to "FOO_BAR". Callers
// that are generating the complete set of names for an enumerated Go type
// must instead use makeUniqueGoEnumValueNames, which resolves such
// collisions.
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

// makeUniqueGoEnumValueNames takes the complete set of ygot.EnumDefinitions
// belonging to a single generated Go enumerated type (i.e. all values of one
// YANG enumeration or identityref), and returns a map, keyed by each value's
// numeric YANG value, of the final Go identifier to use for it.
//
// safeGoEnumeratedValueName sanitises each name independently, and so two
// distinct YANG value names (e.g. "FOO.BAR" and "FOO-BAR") can sanitise to
// the same string, which would otherwise result in a duplicate constant
// definition. makeUniqueGoEnumValueNames resolves such collisions by
// suffixing colliding names, using the repository's genutil.MakeNameUnique,
// until every returned name is unique.
//
// The set of values is processed in a deterministic canonical order --
// ascending by EnumDefinition.Value, with EnumDefinition.Name as a
// tie-break -- that this function owns entirely. Any caller that
// reconstructs the same input set of values (whether at the point the
// enumerated Go type is defined, or later, when resolving a default value
// that references one of its values) is therefore guaranteed to compute the
// identical set of final names regardless of the order in which it happens
// to enumerate the input, since the input is copied and sorted before
// resolution, and is never mutated.
//
// The used-name set is seeded with "UNSET", since the generated enumerated
// Go type always reserves <EnumName>_UNSET as its zero value.
func makeUniqueGoEnumValueNames(values []ygot.EnumDefinition) map[int64]string {
	ordered := make([]ygot.EnumDefinition, len(values))
	copy(ordered, values)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Value != ordered[j].Value {
			return ordered[i].Value < ordered[j].Value
		}
		return ordered[i].Name < ordered[j].Name
	})

	usedNames := map[string]bool{"UNSET": true}
	names := make(map[int64]string, len(ordered))
	for _, v := range ordered {
		names[int64(v.Value)] = genutil.MakeNameUnique(safeGoEnumeratedValueName(v.Name), usedNames)
	}
	return names
}

// enumTypeDefinitions reconstructs the same set of ygot.EnumDefinitions (Name
// and Value pairs) that ygen/genir.go produces for the given YANG
// enumeration or identityref type. t must be a resolved (non-union)
// yang.YangType of Kind yang.Yenum or yang.Yidentityref. The returned slice's
// order is not significant -- makeUniqueGoEnumValueNames re-establishes the
// canonical (Value, Name) order before resolving any collisions -- but the
// Yidentityref case's Value assignment (alphabetical index) must remain in
// sync with ygen/genir.go's construction of
// EnumeratedYANGType.ValToYANGDetails, since that determines the actual
// numeric Value each identity is assigned.
func enumTypeDefinitions(t *yang.YangType) ([]ygot.EnumDefinition, error) {
	switch t.Kind {
	case yang.Yenum:
		valueMap := t.Enum.ValueMap()
		defs := make([]ygot.EnumDefinition, 0, len(valueMap))
		for v, n := range valueMap {
			defs = append(defs, ygot.EnumDefinition{Name: n, Value: int(v)})
		}
		return defs, nil
	case yang.Yidentityref:
		if t.IdentityBase == nil {
			return nil, fmt.Errorf("enumTypeDefinitions: identityref type %q has no base identity", t.Name)
		}
		// Identities have no explicit ordering in YANG, so, consistent
		// with ygen/genir.go, they are alphabetically sorted and assigned
		// a value corresponding to their position in that ordering.
		names := make([]string, 0, len(t.IdentityBase.Values))
		for _, v := range t.IdentityBase.Values {
			names = append(names, v.Name)
		}
		sort.Strings(names)
		defs := make([]ygot.EnumDefinition, len(names))
		for i, n := range names {
			defs[i] = ygot.EnumDefinition{Name: n, Value: i}
		}
		return defs, nil
	default:
		return nil, fmt.Errorf("enumTypeDefinitions: type %q is not an enumeration or identityref (kind: %v)", t.Name, t.Kind)
	}
}

// enumDefaultValue sanitises a default value specified for an enumeration
// which can be specified as prefix:value in the YANG schema. The baseName
// is used as the generated enumeration name stripping any prefix specified,
// (allowing removal of the enumeration type prefix if required). yangType
// must be the resolved (non-union) enumeration or identityref YangType that
// defVal is a value of; it is used to reconstruct the complete sibling set
// of values so that defVal resolves to exactly the same
// collision-disambiguated Go identifier that was chosen for it at the
// enumerated type's definition site (see makeUniqueGoEnumValueNames). The
// default value in the form <sanitised_baseName>_<sanitised_defVal> is
// returned.
func enumDefaultValue(baseName, defVal, prefix string, yangType *yang.YangType) (string, error) {
	if strings.Contains(defVal, ":") {
		defVal = strings.Split(defVal, ":")[1]
	}

	if prefix != "" {
		baseName = strings.TrimPrefix(baseName, prefix)
	}

	defs, err := enumTypeDefinitions(yangType)
	if err != nil {
		return "", err
	}
	names := makeUniqueGoEnumValueNames(defs)

	for _, d := range defs {
		if d.Name == defVal {
			return fmt.Sprintf("%s_%s", baseName, names[int64(d.Value)]), nil
		}
	}
	return "", fmt.Errorf("enumDefaultValue: default value %q not found in enumerated type %q", defVal, yangType.Name)
}
