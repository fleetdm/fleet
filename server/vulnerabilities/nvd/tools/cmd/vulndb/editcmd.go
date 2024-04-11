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
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/vulndb"
	"github.com/facebookincubator/nvdtools/vulndb/mysql"
)

func init() {
	addRequiredFlags(editCmd, "mysql", "owner", "provider")
	RootCmd.AddCommand(editCmd)
}

var editCmd = &cobra.Command{
	Use:   "edit [flags] [ID ...]",
	Short: "edit vulnerability data and save as custom data",
	Long: `
The edit command is a convenience shortcut for the following:

1. Extract vendor + custom records from the database
2. Edit them using $EDITOR
3. Save them in the database as custom records

This effectively allows creating custom, home-made vulnerability records.
If the custom "provider" match a vendor provider, the new custom data
overrides the vendor data when the database is exported.

See the export command for details.

The --provider parameter and a list of CVE IDs are required.

Use the JSON_INDENT environment variable to set the indentation character
for JSON output, e.g. JSON_INDENT=$'\t' or use jq.
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 || gFlagProvider == "" {
			cmd.Usage()
			os.Exit(1)
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			flog.Fatalln("$EDITOR not set, cannot edit")
		}

		db, err := mysql.OpenRead(gFlagMySQL)
		if err != nil {
			flog.Fatalln("cannot open db:", err)
		}
		defer db.Close()

		f, err := ioutil.TempFile("", "vulndb")
		if err != nil {
			flog.Fatalln("cannot open temp file:", err)
		}
		defer f.Close()

		srcmd5 := md5.New()
		mw := io.MultiWriter(f, srcmd5)

		exp := vulndb.DataExporter{
			DB:              db,
			FilterProviders: []string{gFlagProvider},
			FilterCVEs:      args,
		}

		indent := os.Getenv("JSON_INDENT")
		if indent == "" {
			indent = "  " // node-js style indentation
		}

		ctx := context.Background()
		err = exp.JSON(ctx, mw, indent)
		if err != nil {
			flog.Fatalln(err)
		}

		err = f.Sync()
		if err != nil {
			flog.Fatalln(err)
		}

		oscmd := exec.Command(editor, f.Name())
		oscmd.Stdin = os.Stdin
		oscmd.Stdout = os.Stdout
		oscmd.Stderr = os.Stderr
		err = oscmd.Run()
		if err != nil {
			os.Remove(f.Name())
			flog.Fatalln(err)
		}

		_, err = f.Seek(0, io.SeekStart)
		if err != nil {
			flog.Fatalln(err)
		}

		bb, err := ioutil.ReadAll(f)
		if err != nil {
			flog.Fatalln(err)
		}

		x, y := srcmd5.Sum(nil), md5.Sum(bb)
		if bytes.Equal(x[:], y[:]) {
			os.Remove(f.Name())
			fmt.Println("nothing to do")
			return
		}

		db, err = mysql.OpenWrite(gFlagMySQL)
		if err != nil {
			flog.Fatalln("cannot open db:", err)
		}
		defer db.Close()

		imp := vulndb.CustomDataImporter{
			DB:       db,
			Owner:    gFlagOwner,
			Provider: gFlagProvider,
		}

		err = imp.ImportFile(ctx, f.Name())
		if err != nil {
			flog.Infoln("temp file:", f.Name())
			flog.Fatalln(err)
		}

		os.Remove(f.Name()) // remove on success
	},
}
