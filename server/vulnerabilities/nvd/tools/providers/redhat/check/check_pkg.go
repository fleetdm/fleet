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
	"github.com/facebookincubator/nvdtools/rpm"
	"github.com/facebookincubator/nvdtools/wfn"
)

type pkgCheck interface {
	checkPkg(*rpm.Package) bool
}

type constPkgChecker bool

func (c constPkgChecker) checkPkg(_ *rpm.Package) bool {
	return bool(c)
}

// string is package name
type packageStatePkgChecker string

func (c packageStatePkgChecker) checkPkg(pkg *rpm.Package) bool {
	return pkg.Name == string(c)
}

// package is the full package to match
type affectedReleasePkgChecker rpm.Package

func (c affectedReleasePkgChecker) checkPkg(pkg *rpm.Package) bool {
	// if both have names and they're not the same, false
	if c.Name != "" && pkg.Name != "" && c.Name != pkg.Name {
		return false
	}
	// if both have archs and they're not the same, false
	if c.Arch != "" && pkg.Arch != "" && c.Arch != pkg.Arch {
		return false
	}
	if rpm.LabelCompare(pkg.Label, c.Label) < 0 {
		return false
	}

	return true
}

type singleChecker struct {
	distro     *wfn.Attributes
	pkgChecker pkgCheck
}

func (c *singleChecker) Check(pkg *rpm.Package, distro *wfn.Attributes, _ string) bool {
	if pkg == nil {
		// just a sanity check, shouldn't even be called with nil
		return false
	}
	if !c.pkgChecker.checkPkg(pkg) {
		// package doesn't match, return false
		return false
	}

	if distro != nil && c.distro != nil {
		if !wfn.Match(distro, c.distro) {
			// if they don't match, return false
			return false
		}
	}

	return true
}
