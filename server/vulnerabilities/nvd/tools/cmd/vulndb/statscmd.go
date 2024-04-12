// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed"
)

func init() {
	RootCmd.AddCommand(statsCmd)
}

var statsCmd = &cobra.Command{
	Use:   "stats [flags] nvd.json",
	Short: "gather stats from NVD database rules",
	Long: `
The stats command takes a NVD JSON feed file to gather and report stats information.

In instance, a NVD JSON feed file can be exported from the database with:
vulndb export --mysql <MYSQL_DSN> --format nvdcvejson [ID ...] | jq . > nvd.json

Example output: "5%: (h AND o)" indicates that 5% of AND operators are between
a piece of hardware and OS.
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("missing a NVD JSON feed file")
		}
		feedDict, err := feedLoad(args[0])
		if err != nil {
			return err
		}
		stats := cvefeed.NewStats()
		stats.Gather(feedDict)
		stats.ReportOperatorAND()
		return nil
	},
}
