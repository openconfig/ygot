package cmd

import (
	"fmt"
	"os"

	"github.com/openconfig/ygot/gnmidiff"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/encoding/prototext"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

func newSetRequestDiffCmd() *cobra.Command {
	setdiff := &cobra.Command{
		Use:   "setrequest",
		RunE:  setRequestDiff,
		Short: "Diffs the intent between two gNMI SetRequests.",
		Args:  cobra.MinimumNArgs(2),
	}

	setdiff.Flags().Bool("full", false, "Whether diff shows common values.")

	return setdiff
}

func setRequestFromFile(file string) (*gpb.SetRequest, error) {
	sr := &gpb.SetRequest{}
	bs, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	prototext.Unmarshal(bs, sr)

	return sr, nil
}

func setRequestDiff(cmd *cobra.Command, args []string) error {
	format := gnmidiff.Format{
		Full: viper.GetBool("full"),
	}

	srA, err := setRequestFromFile(args[0])
	if err != nil {
		return err
	}

	srB, err := setRequestFromFile(args[1])
	if err != nil {
		return err
	}

	diff, err := gnmidiff.DiffSetRequest(srA, srB, nil)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, diff.Format(format))
	return nil
}
