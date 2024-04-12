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

package redhat

import (
	"sort"
	"strings"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/redhat/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/rpm"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
	"github.com/pkg/errors"
)

// packageFeed is an association between package names and the list of CVEs,
// fixed or not, that have been recording against that package.
// Packages are identified by their base package names without any epoch,
// version, release.
type packageFeed map[string][]*schema.CVE

// addPackage adds pkg to the list of packages only if not already there.
func addPackage(pkgs []string, pkg string) []string {
	for _, p := range pkgs {
		if p == pkg {
			return pkgs
		}
	}
	return append(pkgs, pkg)
}

// packageFeed transforms a Feed into a packageFeed.
func (feed *Feed) packageFeed() packageFeed {
	if feed.pkg2CVE != nil {
		return feed.pkg2CVE
	}

	pkgFeed := packageFeed{}

	for _, cve := range feed.Data {
		var pkgs []string

		// 1. look at AffectedRelease.
		for _, ar := range cve.AffectedRelease {
			if ar.Package == "" {
				continue
			}
			// Failing to parse a package isn't fatal, but we want to surface the error
			rpmPkg, err := rpm.Parse(ar.Package)
			if err != nil {
				flog.Errorf("feed: failed to parse package: %q", ar.Package)
				continue
			}
			pkgs = addPackage(pkgs, rpmPkg.Name)
		}

		// 2. look at PackageState.
		for _, ps := range cve.PackageState {
			if ps.PackageName == "" {
				continue
			}
			pkgs = addPackage(pkgs, strings.ToLower(ps.PackageName))

		}

		for _, pkg := range pkgs {
			pkgFeed[pkg] = append(pkgFeed[pkg], cve)
		}
	}

	feed.pkg2CVE = pkgFeed

	return pkgFeed
}

// ListFixedCVEs will return the list of CVEs that aren't applicable for the
// given (distro, package). Those CVEs could be not applicable for various
// reasons. For instance for packaged version isn't vulnerable or a fix has
// been backported.
// distro is a CPE identifying a distribution.
// pkg is the full package name as reported, for instance by rpm -qa.
func (feed *Feed) ListFixedCVEs(d *wfn.Attributes, p *rpm.Package) ([]string, error) {
	pkgFeed := feed.packageFeed()
	checker, err := feed.Checker()
	if err != nil {
		return nil, errors.Wrapf(err, "list")
	}

	var cves []string
	for _, cve := range pkgFeed[p.Name] {
		if checker.Check(p, d, cve.Name) {
			cves = append(cves, cve.Name)
		}
	}

	// Sort for deterministic output.
	sort.Strings(cves)

	return cves, nil
}
