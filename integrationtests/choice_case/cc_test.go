package cc

import (
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/ygot/integrationtests/choice_case/pkg/cctest"
	"github.com/openconfig/ygot/ygot"
)

func TestUnmarshal(t *testing.T) {

	tests := []struct {
		desc             string
		in               string
		target           ygot.GoStruct
		want             ygot.GoStruct
		wantErrSubstring string
	}{{
		desc: "issue 189",
		in: `
    {
     "parent-cont": {
      "ok":[null]
     }
    }
    `,
		want: &cctest.Root{
			ParentCont: &cctest.Choicecase_ParentCont{
				Ok: cctest.YANGEmpty(true),
			},
		},
		target: &cctest.Root{},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if err := cctest.Unmarshal([]byte(tt.in), tt.target); err != nil {
				if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
					t.Fatalf("did not get expected error, %s", diff)
				}
			}
			if diff := pretty.Compare(tt.target, tt.want); diff != "" {
				t.Fatalf("did not get expected output, diff(-got,+want):\n%v", diff)
			}
		})
	}
}
