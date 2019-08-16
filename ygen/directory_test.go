package ygen

import (
	"reflect"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
)

// errToString returns the string representation of err and the empty string if
// err is nil.
func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestGetOrderedFieldNames(t *testing.T) {
	tests := []struct {
		name string
		in   *Directory
		want []string
	}{{
		name: "nil directory",
		in:   nil,
		want: nil,
	}, {
		name: "empty directory",
		in: &Directory{
			Fields: map[string]*yang.Entry{},
		},
		want: []string{},
	}, {
		name: "directory with one field",
		in: &Directory{
			Fields: map[string]*yang.Entry{
				"a": {},
			},
		},
		want: []string{"a"},
	}, {
		name: "directory with multiple fields",
		in: &Directory{
			Fields: map[string]*yang.Entry{
				"a": {},
				"b": {},
				"c": {},
				"d": {},
				"e": {},
				"f": {},
				"g": {},
			},
		},
		want: []string{"a", "b", "c", "d", "e", "f", "g"},
	}, {
		name: "directory with multiple fields 2",
		in: &Directory{
			Fields: map[string]*yang.Entry{
				"the":   {},
				"quick": {},
				"brown": {},
				"fox":   {},
				"jumps": {},
				"over":  {},
				"the2":  {},
				"lazy":  {},
				"dog":   {},
			},
		},
		want: []string{"brown", "dog", "fox", "jumps", "lazy", "over", "quick", "the", "the2"},
	}}

	for _, tt := range tests {
		if got, want := GetOrderedFieldNames(tt.in), tt.want; !reflect.DeepEqual(want, got) {
			t.Errorf("%s:\nwant: %s\ngot %s", tt.name, want, got)
		}
	}
}

func TestGetOrderedDirectories(t *testing.T) {
	a := &Directory{Name: "a"}
	b := &Directory{Name: "b"}
	c := &Directory{Name: "c"}
	d := &Directory{Name: "d"}
	e := &Directory{Name: "e"}
	f := &Directory{Name: "f"}

	tests := []struct {
		name             string
		in               map[string]*Directory
		wantOrderedNames []string
		wantDirectoryMap map[string]*Directory
		wantErr          string
	}{{
		name:    "nil directory map",
		in:      nil,
		wantErr: "directory map null",
	}, {
		name:             "empty directory map",
		in:               map[string]*Directory{},
		wantOrderedNames: []string{},
		wantDirectoryMap: map[string]*Directory{},
	}, {
		name: "directory map with one directory",
		in: map[string]*Directory{
			"a/b/c": c,
		},
		wantOrderedNames: []string{"c"},
		wantDirectoryMap: map[string]*Directory{"c": c},
	}, {
		name: "directory map with multiple directories",
		in: map[string]*Directory{
			"a/b/d": d,
			"a/b/f": f,
			"a/b/c": c,
			"a/b/b": b,
			"a/b/a": a,
			"a/b/e": e,
		},
		wantOrderedNames: []string{"a", "b", "c", "d", "e", "f"},
		wantDirectoryMap: map[string]*Directory{
			"a": a,
			"b": b,
			"c": c,
			"d": d,
			"e": e,
			"f": f,
		},
	}, {
		name: "directory map with a conflict",
		in: map[string]*Directory{
			"a/b/d": d,
			"a/b/f": f,
			"a/b/c": c,
			"a/b/b": b,
			"a/b/a": a,
			"a/b/e": d,
		},
		wantErr: "directory name conflict(s) exist",
	}}

	for _, tt := range tests {
		gotOrderedNames, gotDirMap, err := GetOrderedDirectories(tt.in)
		if gotErr := errToString(err); gotErr != tt.wantErr {
			t.Errorf("%s:\nwantErr: %s\ngotErr: %s", tt.name, tt.wantErr, gotErr)
		}
		if !reflect.DeepEqual(gotOrderedNames, tt.wantOrderedNames) {
			t.Errorf("%s:\nwantOrderedNames: %s\ngotOrderedNames: %s", tt.name, tt.wantOrderedNames, gotOrderedNames)
		}
		if !reflect.DeepEqual(gotDirMap, tt.wantDirectoryMap) {
			t.Errorf("%s:\nwantDirMap: %v\ngotwantDirMap: %v", tt.name, tt.wantDirectoryMap, gotDirMap)
		}
	}
}
