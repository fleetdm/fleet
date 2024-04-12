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

package check

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/redhat/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/rpm"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

var ErrCheckers = errors.New("no applicable checkers")

func CVEChecker(cve *schema.CVE) (rpm.Checker, error) {
	var chks []rpm.Checker

	archks, err := affectedReleaseCheckers(cve)
	if err != nil {
		return nil, fmt.Errorf("can't construct checkers for affected release: %v", err)
	}
	chks = archks

	pschks, err := packageStateCheckers(cve)
	if err != nil {
		return nil, fmt.Errorf("can't construct checkers for package state: %v", err)
	}
	chks = append(chks, pschks...)

	if len(chks) == 0 {
		return nil, ErrCheckers
	}

	return &cveChecker{cve.Name, chks}, nil
}

func affectedReleaseCheckers(cve *schema.CVE) ([]rpm.Checker, error) {
	var chks []rpm.Checker

	for _, ar := range cve.AffectedRelease {
		if ar.CPE == "" {
			continue
		}

		d, err := wfn.Parse(ar.CPE)
		if err != nil {
			return nil, fmt.Errorf("can't parse distro cpe %q: %v", ar.CPE, err)
		}
		// XXX we need to do this because RedHat sometimes sets `a` as part for RHEL-X, when it should be `o`
		d.Part = wfn.Any

		var pc pkgCheck = constPkgChecker(true) // match all packages
		if ar.Package != "" {
			// add .src to parse it correctly, they're all src rpms
			if p, err := rpm.Parse(ar.Package + ".src"); err == nil {
				pc = affectedReleasePkgChecker(*p)
			}
		}

		chks = append(chks, &singleChecker{d, pc})
	}

	return chks, nil
}

func packageStateCheckers(cve *schema.CVE) ([]rpm.Checker, error) {
	var chks []rpm.Checker

	for _, ps := range cve.PackageState {
		if ps.FixState != "" && !schema.IsFixed(ps.FixState) {
			// if the package hasn't been fixed, continue
			continue
		}

		if ps.CPE == "" {
			continue
		}

		d, err := wfn.Parse(ps.CPE)
		if err != nil {
			return nil, fmt.Errorf("can't parse distro cpe %q: %v", ps.CPE, err)
		}
		// XXX we need to do this because RedHat sometimes sets `a` as part for RHEL-X, when it should be `o`
		d.Part = wfn.Any

		var pc pkgCheck = constPkgChecker(true) // match all packages
		if ps.PackageName != "" {
			pc = packageStatePkgChecker(strings.ToLower(ps.PackageName))
		}

		chks = append(chks, &singleChecker{d, pc})
	}

	return chks, nil
}

// this is an or checker, if any of checkers returns true, result is true
type cveChecker struct {
	cve  string
	chks []rpm.Checker
}

// Check is part of the rpm.Check interface
func (c *cveChecker) Check(pkg *rpm.Package, distro *wfn.Attributes, cve string) bool {
	if cve != c.cve {
		return false
	}
	for _, chk := range c.chks {
		if chk.Check(pkg, distro, cve) {
			return true
		}
	}
	return false
}
