// Copyright 2023 Google Inc.
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

package cmd

import (
	"fmt"
	"os"

	"github.com/openconfig/ygot/gnmidiff"
	"github.com/openconfig/ygot/gnmidiff/gnmiparse"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

func setRequestDiff(cmd *cobra.Command, args []string) error {
	format := gnmidiff.Format{
		Full: viper.GetBool("full"),
	}

	srA, err := gnmiparse.SetRequestFromFile(args[0])
	if err != nil {
		return err
	}

	srB, err := gnmiparse.SetRequestFromFile(args[1])
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
