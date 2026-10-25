package gogen

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

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

// TestMakeUniqueGoEnumValueNames validates that makeUniqueGoEnumValueNames
// preserves the existing sanitised name for non-colliding sibling values
// (backward compatibility, see https://github.com/openconfig/ygot/issues/1083),
// and disambiguates colliding sibling values -- including second-order
// collisions with an already-uniquified name, and a value literally named
// "UNSET" -- into a set of names that are all unique.
func TestMakeUniqueGoEnumValueNames(t *testing.T) {
	tests := []struct {
		name string
		in   []ygot.EnumDefinition
		want map[int64]string
	}{{
		name: "no collisions -- ordinary sanitised names are unchanged",
		in: []ygot.EnumDefinition{
			{Name: "SPEED_2.5G", Value: 0},
			{Name: "IPV4-UNICAST", Value: 1},
			{Name: "LEVEL1/2", Value: 2},
		},
		want: map[int64]string{
			0: "SPEED_2_5G",
			1: "IPV4_UNICAST",
			2: "LEVEL1_2",
		},
	}, {
		name: "-, ., / and space all collide with a literal underscore",
		in: []ygot.EnumDefinition{
			{Name: "a-b", Value: 0},
			{Name: "a.b", Value: 1},
			{Name: "a/b", Value: 2},
			{Name: "a b", Value: 3},
			{Name: "a_b", Value: 4},
		},
		want: map[int64]string{
			0: "a_b",
			1: "a_b_",
			2: "a_b__",
			3: "a_b___",
			4: "a_b____",
		},
	}, {
		name: "second-order collision: a sibling already named like a uniquified result",
		in: []ygot.EnumDefinition{
			{Name: "a-b", Value: 0},
			{Name: "a.b", Value: 1},
			{Name: "a_b_", Value: 2},
		},
		want: map[int64]string{
			0: "a_b",
			1: "a_b_",
			// The literal "a_b_" also collides with the name assigned
			// to "a.b", so it must be pushed one underscore further.
			2: "a_b__",
		},
	}, {
		name: "a value literally named UNSET collides with the reserved zero value",
		in: []ygot.EnumDefinition{
			{Name: "UNSET", Value: 0},
			{Name: "RUNNING", Value: 1},
		},
		want: map[int64]string{
			0: "UNSET_",
			1: "RUNNING",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run against both the given order, and a reversed copy,
			// to demonstrate that the canonical (Value, Name) ordering
			// is owned by makeUniqueGoEnumValueNames itself, and does
			// not depend on the order in which the caller happens to
			// enumerate the sibling values.
			reversed := make([]ygot.EnumDefinition, len(tt.in))
			for i, v := range tt.in {
				reversed[len(tt.in)-1-i] = v
			}

			for _, in := range [][]ygot.EnumDefinition{tt.in, reversed} {
				inCopy := make([]ygot.EnumDefinition, len(in))
				copy(inCopy, in)

				got := makeUniqueGoEnumValueNames(inCopy)
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("makeUniqueGoEnumValueNames(%v): (-want, +got):\n%s", in, diff)
				}

				if diff := cmp.Diff(in, inCopy); diff != "" {
					t.Errorf("makeUniqueGoEnumValueNames(%v) mutated its input, diff(-orig, +after):\n%s", in, diff)
				}

				seen := map[string]bool{}
				for _, name := range got {
					if seen[name] {
						t.Errorf("makeUniqueGoEnumValueNames(%v): duplicate generated name %q in %v", in, name, got)
					}
					seen[name] = true
				}
			}
		})
	}
}

// TestEnumTypeDefinitions validates that enumTypeDefinitions reconstructs the
// same set of (Name, Value) pairs -- in particular, the same Value for each
// identityref Name -- that ygen/genir.go assigns when building
// EnumeratedYANGType.ValToYANGDetails. enumTypeDefinitions does not promise
// anything about the order of its returned slice, so this test sorts before
// comparing.
func TestEnumTypeDefinitions(t *testing.T) {
	enumType := yang.NewEnumType()
	enumType.Set("LOW", 5)
	enumType.Set("HIGH", 10)

	identityBase := &yang.Identity{
		Name: "base-identity",
		Values: []*yang.Identity{
			{Name: "FOO.BAR"},
			{Name: "ALPHA"},
			{Name: "FOO-BAR"},
		},
	}

	tests := []struct {
		name    string
		in      *yang.YangType
		want    []ygot.EnumDefinition
		wantErr bool
	}{{
		name: "enumeration",
		in:   &yang.YangType{Kind: yang.Yenum, Enum: enumType},
		want: []ygot.EnumDefinition{
			{Name: "LOW", Value: 5},
			{Name: "HIGH", Value: 10},
		},
	}, {
		name: "identityref -- alphabetically ordered, zero-indexed",
		in:   &yang.YangType{Kind: yang.Yidentityref, IdentityBase: identityBase},
		want: []ygot.EnumDefinition{
			{Name: "ALPHA", Value: 0},
			{Name: "FOO-BAR", Value: 1},
			{Name: "FOO.BAR", Value: 2},
		},
	}, {
		name:    "identityref without a base identity",
		in:      &yang.YangType{Kind: yang.Yidentityref},
		wantErr: true,
	}, {
		name:    "unsupported kind",
		in:      &yang.YangType{Kind: yang.Ystring},
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := enumTypeDefinitions(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("enumTypeDefinitions(%v): got error: %v, wantErr: %v", tt.in, err, tt.wantErr)
			}
			if err != nil {
				return
			}

			// enumTypeDefinitions does not guarantee an order, so sort
			// before comparing -- this is a test-comparison
			// convenience, not a claim about the function's contract.
			sort.Slice(got, func(i, j int) bool { return got[i].Value < got[j].Value })
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("enumTypeDefinitions(%v): (-want, +got):\n%s", tt.in, diff)
			}
		})
	}
}

// TestEnumDefaultValue validates that enumDefaultValue resolves a default
// value to the same collision-disambiguated Go identifier that
// makeUniqueGoEnumValueNames would choose for it at the enum's definition
// site.
func TestEnumDefaultValue(t *testing.T) {
	collidingEnum := yang.NewEnumType()
	collidingEnum.Set("SPEED.1", 0)
	collidingEnum.Set("SPEED-1", 1)

	tests := []struct {
		name       string
		inBaseName string
		inDefVal   string
		inPrefix   string
		inYANGType *yang.YangType
		want       string
		wantErr    bool
	}{{
		name:       "first of two colliding values keeps the plain sanitised name",
		inBaseName: "MyEnum",
		inDefVal:   "SPEED.1",
		inYANGType: &yang.YangType{Kind: yang.Yenum, Enum: collidingEnum},
		want:       "MyEnum_SPEED_1",
	}, {
		name:       "second of two colliding values is disambiguated",
		inBaseName: "MyEnum",
		inDefVal:   "SPEED-1",
		inYANGType: &yang.YangType{Kind: yang.Yenum, Enum: collidingEnum},
		want:       "MyEnum_SPEED_1_",
	}, {
		name:       "module-prefixed default value",
		inBaseName: "MyEnum",
		inDefVal:   "some-module:SPEED-1",
		inYANGType: &yang.YangType{Kind: yang.Yenum, Enum: collidingEnum},
		want:       "MyEnum_SPEED_1_",
	}, {
		name:       "prefix is stripped from the base name",
		inBaseName: "E_MyEnum",
		inPrefix:   "E_",
		inDefVal:   "SPEED-1",
		inYANGType: &yang.YangType{Kind: yang.Yenum, Enum: collidingEnum},
		want:       "MyEnum_SPEED_1_",
	}, {
		name:       "unknown default value",
		inBaseName: "MyEnum",
		inDefVal:   "NOT-A-VALUE",
		inYANGType: &yang.YangType{Kind: yang.Yenum, Enum: collidingEnum},
		wantErr:    true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := enumDefaultValue(tt.inBaseName, tt.inDefVal, tt.inPrefix, tt.inYANGType)
			if (err != nil) != tt.wantErr {
				t.Fatalf("enumDefaultValue(%q, %q, %q): got error: %v, wantErr: %v", tt.inBaseName, tt.inDefVal, tt.inPrefix, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got != tt.want {
				t.Errorf("enumDefaultValue(%q, %q, %q): got: %q, want: %q", tt.inBaseName, tt.inDefVal, tt.inPrefix, got, tt.want)
			}
		})
	}
}
