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
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb/mysql"
)

func init() {
	RootCmd.AddCommand(vendorCmd)
}

var vendorCmd = &cobra.Command{
	Use:   "vendor [command]",
	Short: "manage vulnerability data from vendors",
}

func init() {
	addRequiredFlags(vendorImportCmd, "mysql", "owner", "provider")
	vendorCmd.AddCommand(vendorImportCmd)
}

var vendorImportCmd = &cobra.Command{
	Use:   "import [flags] [file.json[.gz] ...]",
	Short: "import vulnerability data into the database",
	Long: `
The import command imports multiple files formatted as NVD JSON 1.0 into
the database.

The database supports multiple providers, and for each provider there should
be an owner (a unixname or other form of ID). Each import requires setting
the --provider and --owner flags.

File schema: https://csrc.nist.gov/schema/nvd/feed/1.0/nvd_cve_feed_json_1.0.schema
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			os.Exit(1)
		}

		db, err := mysql.OpenWrite(gFlagMySQL)
		if err != nil {
			flog.Fatalln("cannot open db:", err)
		}
		defer db.Close()

		imp := vulndb.VendorDataImporter{
			DB:       db,
			Owner:    gFlagOwner,
			Provider: gFlagProvider,
			OnFile: func(name string) {
				flog.Infoln("importing", name)
			},
		}

		ctx := context.Background()
		vendor, err := imp.ImportFiles(ctx, args...)
		if err != nil {
			flog.Fatal(err)
		}

		flog.Infof("imported %s:%d", vendor.Provider, vendor.Version)
	},
}

func init() {
	addRequiredFlags(vendorExportCmd, "mysql", "provider", "format")
	addOptionalFlags(vendorExportCmd, "csv_noheader")
	vendorCmd.AddCommand(vendorExportCmd)
}

var vendorExportCmd = &cobra.Command{
	Use:   "export [flags] [ID ...]",
	Short: "export vulnerability data from vendors",
	Long: `
The export command returns data from the database for a specific provider.

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

		exp := vulndb.VendorDataExporter{
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
	addRequiredFlags(vendorTrimCmd, "mysql", "delete_all")
	addOptionalFlags(vendorTrimCmd, "provider")
	vendorCmd.AddCommand(vendorTrimCmd)
}

var vendorTrimCmd = &cobra.Command{
	Use:   "trim [flags]",
	Short: "delete data versions from the vulnerability database",
	Long: `
The delete command deletes versions of data from the database.

By default keeps only the latest version of data for each provider. In order
to delete data from specific providers, use --provider=name1,nameN.

To force deleting all data use --all, optionally combined with --provider.
`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := mysql.OpenWrite(gFlagMySQL)
		if err != nil {
			flog.Fatalln("cannot open db:", err)
		}
		defer db.Close()

		var providers []string
		if gFlagProvider != "" {
			providers = strings.Split(gFlagProvider, ",")
		}

		del := vulndb.VendorDataTrimmer{
			DB:                  db,
			FilterProviders:     providers,
			DeleteLatestVersion: gFlagDeleteAll,
		}

		ctx := context.Background()
		err = del.Trim(ctx)
		if err != nil {
			flog.Fatalln(err)
		}
	},
}
