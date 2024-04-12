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
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/redhat"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/rpm"
)

func main() {
	var cfg config
	cfg.addFlags()
	flog.AddFlags(flag.CommandLine, nil)
	flag.Parse()

	if err := cfg.validate(); err != nil {
		flog.Fatal(err)
	}
	if flag.NArg() != 1 {
		flog.Fatalf("expecting one argument: feed path. got %d", flag.NArg())
	}

	feed, err := redhat.LoadFeed(flag.Arg(0))
	if err != nil {
		flog.Fatal(err)
	}
	chk, err := feed.Checker()
	if err != nil {
		flog.Fatal(err)
	}

	// adjust indexes
	cfg.pkgs--
	cfg.distro--
	cfg.cve--

	if err := filter(chk, &cfg, os.Stdin, os.Stdout); err != nil {
		flog.Fatal(err)
	}
}

func filter(chk rpm.Checker, cfg *config, r io.Reader, w io.Writer) error {
	cr := csv.NewReader(r)
	cw := csv.NewWriter(w)

	for {
		// read
		row, err := cr.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// check indexes
		if err := cfg.checkIndexes(len(row)); err != nil {
			return err
		}

		// filter
		pkgs := strings.Split(row[cfg.pkgs], cfg.pkgsSep)
		filtered, err := rpm.FilterFixedPackages(chk, pkgs, row[cfg.distro], row[cfg.cve])
		if err != nil {
			return fmt.Errorf("failed to filter packages: %v", err)
		}
		row[cfg.pkgs] = strings.Join(filtered, cfg.pkgsSep)

		// write
		if err := cw.Write(row); err != nil {
			return err
		}
		cw.Flush()
		if err := cw.Error(); err != nil {
			return err
		}
	}
}

type config struct {
	pkgs, distro, cve int
	pkgsSep           string
}

func (cfg *config) addFlags() {
	flag.IntVar(&cfg.pkgs, "pkgs", 0, "csv field which holds the packages. starts with 1")
	flag.IntVar(&cfg.distro, "distro", 0, "csv field which holds the distribution CPE. starts with 1")
	flag.IntVar(&cfg.cve, "cve", 0, "csv field which holds the CVE. starts with 1")
	flag.StringVar(&cfg.pkgsSep, "pkgs-sep", "\x02", "separator to use for the packages field")
}

func (cfg *config) validate() error {
	if cfg.pkgs <= 0 || cfg.distro <= 0 || cfg.cve <= 0 {
		flog.Fatalf("indexes must be positive: distro=%d pkgs=%d cve=%d", cfg.distro, cfg.pkgs, cfg.cve)
	}
	return nil
}

func (cfg *config) checkIndexes(n int) error {
	if cfg.pkgs >= n {
		return fmt.Errorf("not enough fields. have %d fields but pkgs index is %d", n, cfg.pkgs+1)
	}
	if cfg.distro >= n {
		return fmt.Errorf("not enough fields. have %d fields but distro index is %d", n, cfg.distro+1)
	}
	if cfg.cve >= n {
		return fmt.Errorf("not enough fields. have %d fields but cve index is %d", n, cfg.cve+1)
	}
	return nil
}
