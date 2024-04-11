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
	"strings"

	"github.com/spf13/cobra"

	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/vulndb"
	"github.com/facebookincubator/nvdtools/vulndb/mysql"
)

func init() {
	addRequiredFlags(exportCmd, "mysql", "format")
	addOptionalFlags(exportCmd, "provider", "csv_noheader")
	RootCmd.AddCommand(exportCmd)
}

var exportCmd = &cobra.Command{
	Use:   "export [flags] [ID ...]",
	Short: "export vulnerability data from the database",
	Long: `
The export command returns data from all vendors + overrides.

The --format flag is required and must be one of csv or nvdcvejson.

The --provider flag is optional, as well as the list of IDs to filter in.

Use the JSON_INDENT environment variable to set the indentation character
for JSON output, e.g. JSON_INDENT=$'\t' or use jq.
`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := mysql.OpenRead(gFlagMySQL)
		if err != nil {
			flog.Fatalln("cannot open db:", err)
		}
		defer db.Close()

		var providers []string
		if len(gFlagProvider) > 0 {
			providers = strings.Split(gFlagProvider, ",")
		}

		exp := vulndb.DataExporter{
			DB:              db,
			FilterProviders: providers,
			FilterCVEs:      args,
		}

		ctx := context.Background()

		switch gFlagFormat {
		case "csv":
			err = exp.CSV(ctx, os.Stdout, !gFlagCSVNoHeader)
		case "nvdcvejson":
			err = exp.JSON(ctx, os.Stdout, os.Getenv("JSON_INDENT"))
		default:
			flog.Fatalln("unsupported format:", gFlagFormat)
		}
		if err != nil {
			flog.Fatalln(err)
		}
	},
}
