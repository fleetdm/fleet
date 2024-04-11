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

// vulndb command line tool.
package main

import (
	"github.com/spf13/cobra"

  "github.com/facebookincubator/flog"
)

func main() {
	if err := RootCmd.Execute(); err != nil {
		flog.Fatal(err)
	}
}

// RootCmd is the root command command used by main.
var RootCmd = &cobra.Command{
	Use:   "vulndb",
	Short: "Vulnerability Database Management Tool",
}
