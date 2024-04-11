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
	RootCmd.AddCommand(snoozeCmd)
}

var snoozeCmd = &cobra.Command{
	Use:   "snooze [command]",
	Short: "manage vulnerability snooze data",
}

func init() {
	addRequiredFlags(snoozeSetCmd, "mysql", "owner", "collector", "provider")
	addOptionalFlags(snoozeSetCmd, "deadline", "metadata")
	snoozeCmd.AddCommand(snoozeSetCmd)
}

var snoozeSetCmd = &cobra.Command{
	Use:   "set [flags] [ID ...]",
	Short: "set snoozes in the vulnerability database",
	Long: `
The set command creates a snooze record in the database. These records
are useful for post-processing of vulnerability inventories to temporary
disable remediation/automation.

Snoozes are tied to specific collectors and providers, and must have an owner.

The deadline and metadata flags are optional.
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

		sc := vulndb.SnoozeCreator{
			DB:        db,
			Owner:     gFlagOwner,
			Collector: gFlagCollector,
			Provider:  gFlagProvider,
			Deadline:  gFlagDeadline.Time,
		}

		if gFlagMetadata != "" {
			sc.Metadata = []byte(gFlagMetadata)
		}

		ctx := context.Background()
		err = sc.Create(ctx, args...)
		if err != nil {
			flog.Fatalln(err)
		}
	},
}

func init() {
	addRequiredFlags(snoozeGetCmd, "mysql")
	addOptionalFlags(snoozeGetCmd, "collector", "provider", "csv_noheader")
	snoozeCmd.AddCommand(snoozeGetCmd)
}

var snoozeGetCmd = &cobra.Command{
	Use:   "get [flags]",
	Short: "get snoozes from the vulnerability database",
	Long: `
The get command returns snooze records from the database.

The --collector and --provider flags, and list of CVEs are optional filters.
`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := mysql.OpenRead(gFlagMySQL)
		if err != nil {
			flog.Fatalln("cannot open db:", err)
		}
		defer db.Close()

		sg := vulndb.SnoozeGetter{
			DB:         db,
			Collector:  gFlagCollector,
			Provider:   gFlagProvider,
			FilterCVEs: args,
		}

		ctx := context.Background()
		err = sg.CSV(ctx, os.Stdout, !gFlagCSVNoHeader)
		if err != nil {
			flog.Fatalln(err)
		}
	},
}

func init() {
	addRequiredFlags(snoozeDelCmd, "mysql", "collector", "provider", "delete_all")
	snoozeCmd.AddCommand(snoozeDelCmd)
}

var snoozeDelCmd = &cobra.Command{
	Use:   "delete [flags] [ID ...]",
	Short: "delete snoozes from the vulnerability database",
	Long: `
The delete command deletes snoozes from the database for specific providers.
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

		del := vulndb.SnoozeDeleter{
			DB:         db,
			Collector:  gFlagCollector,
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
