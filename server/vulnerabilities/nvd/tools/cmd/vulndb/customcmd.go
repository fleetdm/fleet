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
	"github.com/facebookincubator/nvdtools/vulndb"
	"github.com/facebookincubator/nvdtools/vulndb/mysql"
)

func init() {
	RootCmd.AddCommand(customCmd)
}

var customCmd = &cobra.Command{
	Use:   "custom [command]",
	Short: "manage custom vulnerability data",
}

func init() {
	addRequiredFlags(customImportCmd, "mysql", "owner", "provider")
	customCmd.AddCommand(customImportCmd)
}

var customImportCmd = &cobra.Command{
	Use:   "import [flags] file.json[.gz]",
	Short: "import custom vulnerability into the database",
	Long: `
The import command imports a file formatted as NVD JSON 1.0 into the database.

The database supports multiple providers, and for each provider there should
be an owner (a unixname or other form of ID). Each import requires setting
the --provider and --owner flags.

File schema: https://csrc.nist.gov/schema/nvd/feed/1.0/nvd_cve_feed_json_1.0.schema
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			cmd.Usage()
			os.Exit(1)
		}

		db, err := mysql.OpenWrite(gFlagMySQL)
		if err != nil {
			flog.Fatalln("cannot open db:", err)
		}
		defer db.Close()

		imp := vulndb.CustomDataImporter{
			DB:       db,
			Owner:    gFlagOwner,
			Provider: gFlagProvider,
		}

		ctx := context.Background()
		err = imp.ImportFile(ctx, args[0])
		if err != nil {
			flog.Fatal(err)
		}
	},
}

func init() {
	addRequiredFlags(customExportCmd, "mysql", "provider", "format")
	addOptionalFlags(customExportCmd, "csv_noheader")
	customCmd.AddCommand(customExportCmd)
}

var customExportCmd = &cobra.Command{
	Use:   "export [flags] [ID ...]",
	Short: "export custom vulnerability from the database",
	Long: `
The export command returns custom data from the database for specific providers.

The --format flag is required and must be one of csv or nvdcvejson.

Use the JSON_INDENT environment variable to set the indentation character
for JSON output, e.g. JSON_INDENT=$'\t' or use jq.
`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := mysql.OpenRead(gFlagMySQL)
		if err != nil {
			flog.Fatalln("cannot open db:", err)
		}
		defer db.Close()

		exp := vulndb.CustomDataExporter{
			DB:         db,
			Provider:   gFlagProvider,
			FilterCVEs: args,
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

func init() {
	addRequiredFlags(customDeleteCmd, "mysql", "provider", "delete_all")
	customCmd.AddCommand(customDeleteCmd)
}

var customDeleteCmd = &cobra.Command{
	Use:   "delete [flags] [ID ...]",
	Short: "delete custom vulnerability from the database",
	Long: `
The delete command deletes custom records from the database,
for specific providers. Requires a list of CVE ID to delete, or
--all to delete all records from the given provider.
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && !gFlagDeleteAll {
			cmd.Usage()
			os.Exit(1)
		}

		db, err := mysql.OpenWrite(gFlagMySQL)
		if err != nil {
			flog.Fatalln("cannot open db:", err)
		}
		defer db.Close()

		del := vulndb.CustomDataDeleter{
			DB:         db,
			Provider:   gFlagProvider,
			FilterCVEs: args,
		}

		ctx := context.Background()
		err = del.Delete(ctx)
		if err != nil {
			flog.Fatalln(err)
		}
	},
}
