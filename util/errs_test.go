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

package util

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"
)

var (
	testErrs = Errors{fmt.Errorf("err1"), fmt.Errorf("err2")}
	wantStr  = "err1, err2"
)

func TestError(t *testing.T) {
	if got, want := testErrs.Error(), wantStr; got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}
}

func TestString(t *testing.T) {
	if got, want := testErrs.String(), wantStr; got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}
}

func TestToString(t *testing.T) {
	if got, want := ToString(testErrs), wantStr; got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}
}

func TestNewErrs(t *testing.T) {
	var errs Errors
	errs = NewErrs(nil)
	if errs != nil {
		t.Errorf("got: %s, want: nil", errs)
	}

	errs = NewErrs(fmt.Errorf("err1"))
	if got, want := errs.String(), "err1"; got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}
}

func TestAppendErr(t *testing.T) {
	var errs Errors
	if got, want := errs.String(), ""; got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}

	errs = AppendErr(errs, nil)
	if got, want := errs.String(), ""; got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}

	errs = AppendErr(errs, fmt.Errorf("err1"))
	if got, want := errs.String(), "err1"; got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}

	errs = AppendErr(errs, nil)
	errs = AppendErr(errs, fmt.Errorf("err2"))
	if got, want := errs.String(), "err1, err2"; got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}
}

func TestAppendErrs(t *testing.T) {
	var errs Errors

	errs = AppendErrs(errs, []error{nil})
	if got, want := errs.String(), ""; got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}

	errs = AppendErrs(errs, testErrs)
	errs = AppendErrs(errs, []error{nil})
	if got, want := errs.String(), wantStr; got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}
}

func TestAppendErrsInFunction(t *testing.T) {
	myAppendErrFunc := func() (errs Errors) {
		errs = AppendErr(errs, fmt.Errorf("err1"))
		errs = AppendErr(errs, fmt.Errorf("err2"))
		return
	}
	if got, want := myAppendErrFunc().String(), wantStr; got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}

	myAppendErrsFunc := func() (errs Errors) {
		errs = AppendErrs(errs, testErrs)
		return
	}
	if got, want := myAppendErrsFunc().String(), wantStr; got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}

	myErrorSliceFunc := func() (errs []error) {
		errs = AppendErrs(errs, testErrs)
		return
	}

	if got, want := Errors(myErrorSliceFunc()).String(), wantStr; got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}
}

func TestPrefixErrors(t *testing.T) {
	tests := []struct {
		name   string
		inErrs Errors
		inPfx  string
		want   Errors
	}{{
		name: "empty",
	}, {
		name:   "prefixed",
		inErrs: Errors{errors.New("one"), errors.New("two")},
		inPfx:  "a",
		want:   Errors{errors.New("a: one"), errors.New("a: two")},
	}}

	for _, tt := range tests {
		if got := PrefixErrors(tt.inErrs, tt.inPfx); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: PrefixErrors(%v, %s): did not get expected result, got: %v, want: %v", tt.name, tt.inErrs, tt.inPfx, got, tt.want)
		}
	}
}

func TestUniqueErrors(t *testing.T) {
	tests := []struct {
		name string
		in   Errors
		want Errors
	}{{
		name: "empty",
	}, {
		name: "single error",
		in:   Errors{errors.New("one")},
		want: Errors{errors.New("one")},
	}, {
		name: "deduplicated",
		in:   Errors{errors.New("one"), errors.New("one")},
		want: Errors{errors.New("one")},
	}, {
		name: "not equal",
		in:   Errors{errors.New("one"), errors.New("two")},
		want: Errors{errors.New("one"), errors.New("two")},
	}}

	sortErrors := func(errs Errors) Errors {
		m := map[string]error{}
		keys := []string{}
		for _, err := range errs {
			k := fmt.Sprintf("%v\n", err)
			m[k] = err
			keys = append(keys, k)
		}

		sort.Strings(keys)
		var n Errors
		for _, k := range keys {
			n = append(n, m[k])
		}
		return n
	}

	for _, tt := range tests {
		if got := UniqueErrors(tt.in); !reflect.DeepEqual(sortErrors(got), sortErrors(tt.want)) {
			t.Errorf("%s: UniqueErrors(%v): did not get expected result, got: %v, want: %v", tt.name, tt.in, got, tt.want)
		}
	}
}
