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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(diffCmd)
}

func feedLoad(file string) (cvefeed.Dictionary, error) {
	flog.Infof("loading %s\n", file)
	dict, err := cvefeed.LoadJSONDictionary(file)
	if err != nil {
		return nil, fmt.Errorf("failed to load dictionary %s: %v", file, err)
	}
	return dict, nil
}

func feedName(file string) string {
	return filepath.Base(file[:len(file)-5])
}

func printArraySorted(a []string, indent string, n int) {
	min := func(a, b int) int {
		if a <= b {
			return a
		}
		return b
	}

	sort.Strings(a)
	for i := 0; i < min(len(a), n); i++ {
		fmt.Printf("%s%s\n", indent, a[i])
	}
	if len(a) > n {
		fmt.Printf("%s... (%d more)\n", indent, len(a)-10)
	}
}

var diffCmd = &cobra.Command{
	Use:   "diff [flags] a.json b.json",
	Short: "diff two vulnerability feeds",
	RunE: func(cmd *cobra.Command, args []string) error {
		percentInt := func(a, b int) float64 {
			return float64(a) / float64(b) * 100
		}

		if len(args) != 2 {
			return errors.New("missing JSON export files")
		}

		aDict, err := feedLoad(args[0])
		if err != nil {
			return err
		}

		bDict, err := feedLoad(args[1])
		if err != nil {
			return err
		}

		flog.Infoln("computing stats")

		a := feedName(args[0])
		b := feedName(args[1])

		stats := cvefeed.Diff(a, aDict, b, bDict)

		fmt.Printf("Num vulnerabilities in %s: %d\n", a, stats.NumVulnsA())
		fmt.Printf("Num vulnerabilities in %s: %d\n", b, stats.NumVulnsB())
		fmt.Printf("Num vulnerabilities in %s but not in %s: %d\n", a, b, stats.NumVulnsANotB())
		printArraySorted(stats.VulnsANotB(), "    ", 10)
		fmt.Printf("Num vulnerabilities in %s but not in %s: %d\n", b, a, stats.NumVulnsBNotA())
		printArraySorted(stats.VulnsBNotA(), "    ", 10)
		fmt.Println()
		fmt.Printf("Different vulnerabilities: %d\n", stats.NumDiffVulns())
		fmt.Printf("    different descriptions: %d (%.2f%%, total %.2f%%)\n",
			stats.NumChunk(cvefeed.ChunkDescription), stats.PercentChunk(cvefeed.ChunkDescription),
			percentInt(stats.NumChunk(cvefeed.ChunkDescription), stats.NumVulnsA()))
		fmt.Printf("    different scores      : %d (%.2f%%, total %.2f%%)\n",
			stats.NumChunk(cvefeed.ChunkScore), stats.PercentChunk(cvefeed.ChunkScore),
			percentInt(stats.NumChunk(cvefeed.ChunkScore), stats.NumVulnsA()))

		flog.Infoln("writing differences to stats.json")

		data, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode stats to JSON: %w", err)
		}
		if err := ioutil.WriteFile("stats.json", data, 0o644); err != nil {
			return fmt.Errorf("failed to write stats file: %w", err)
		}

		return nil
	},
}
