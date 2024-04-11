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
	"strings"

	"github.com/facebookincubator/nvdtools/providers/redhat"
	"github.com/facebookincubator/nvdtools/rpm"
	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

var fixedCVEsCmd = &cobra.Command{
	Use:   "fixed-cves PACKAGENAME [PACKAGENAME...]",
	Short: "list the fixed/non applicable CVEs for a given package",
	RunE:  fixedCVEs,
}

func init() {
	rootCmd.AddCommand(fixedCVEsCmd)
}

func fixedCVEs(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("fixed-cves: missing package name(s)")
	}

	feed, err := redhat.LoadFeed(options.feed)
	if err != nil {
		return errors.Wrap(err, "fixed-cves")
	}

	distro, err := wfn.Parse(options.distro)
	if err != nil {
		return fmt.Errorf("fixed-cves: can't parse distro cpe %q: %v", distro, err)
	}

	for _, pkg := range args {
		rpmPkg, err := rpm.Parse(pkg)
		if err != nil {
			return fmt.Errorf("fixed-cves: can't parse package %q: %v", pkg, err)
		}

		cves, err := feed.ListFixedCVEs(distro, rpmPkg)
		if err != nil {
			return errors.Wrap(err, "fixed-cves")
		}

		if len(cves) == 0 {
			fmt.Printf("%s: <no fixed CVE found>\n", pkg)
			continue
		}

		fmt.Printf("%s: %s\n", pkg, strings.Join(cves, ","))
	}

	return nil
}
