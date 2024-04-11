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
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var options struct {
	feed   string
	distro string
}

var rootCmd = &cobra.Command{
	Use:           "redhat_query",
	Short:         "redhat_query performs various queries on the redhat CVE feed",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&options.feed, "feed", "f", "redhat-feed.json", "path to the feed JSON file as retrieved by redhat2nvd")
	rootCmd.PersistentFlags().StringVarP(&options.distro, "distribution", "d", "cpe:/o:redhat:enterprise_linux:7", "CPE identifying the distribution")
	rootCmd.MarkFlagRequired("feed")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
