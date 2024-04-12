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
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb/mysql"
)

func init() {
	addRequiredFlags(summaryCmd, "mysql")
	addOptionalFlags(summaryCmd, "csv_noheader")
	RootCmd.AddCommand(summaryCmd)
}

var summaryCmd = &cobra.Command{
	Use:   "summary [flags]",
	Short: "export summary information from the database",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := mysql.OpenRead(gFlagMySQL)
		if err != nil {
			flog.Fatalln("cannot open db:", err)
		}
		defer db.Close()

		exp := vulndb.SummaryExporter{
			DB: db,
		}

		ctx := context.Background()
		err = exp.CSV(ctx, os.Stdout, !gFlagCSVNoHeader)
		if err != nil {
			flog.Fatalln(err)
		}
	},
}
