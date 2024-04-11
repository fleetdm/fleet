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

package rpm

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/facebookincubator/nvdtools/wfn"
)

// Package represents one RPM package
type Package struct {
	Name string
	Label
	Arch string
}

// Label is part of the package and allows us to commpare two RPM packages
type Label struct {
	Epoch   string
	Version string
	Release string
}

func (p *Package) String() string {
	return fmt.Sprintf("%s-%s:%s-%s.%s", p.Name, p.Epoch, p.Version, p.Release, p.Arch)
}

// Parse returns name, version, release and architecture parsed from RPM package name
// NEVRA: https://blog.jasonantman.com/2014/07/how-yum-and-rpm-compare-versions/
func Parse(pkg string) (*Package, error) {
	// pkg should be name-[epoch:]version-release.arch.rpm

	// extension
	pkg = strings.TrimSuffix(pkg, ".rpm")

	var p Package

	// arch
	if i := strings.LastIndexByte(pkg, '.'); i >= 0 {
		pkg, p.Arch = pkg[:i], pkg[i+1:]
		if p.Arch == "src" || p.Arch == "noarch" {
			p.Arch = ""
		}
	} else {
		return nil, fmt.Errorf("can't find arch in pkg %q", pkg)
	}

	// release
	if i := strings.LastIndexByte(pkg, '-'); i >= 0 {
		pkg, p.Label.Release = pkg[:i], pkg[i+1:]
	} else {
		return nil, fmt.Errorf("can't find release in pkg %q", pkg)
	}

	// version and epoch
	if i := strings.LastIndexByte(pkg, '-'); i >= 0 {
		var ver string
		pkg, ver = pkg[:i], pkg[i+1:]
		// check if there's epoch
		if i := strings.IndexByte(ver, ':'); i >= 0 {
			p.Label.Epoch, ver = ver[:i], ver[i+1:]
		}
		p.Label.Version = ver
	} else {
		return nil, fmt.Errorf("can't find version in pkg %q", pkg)
	}

	p.Name = strings.ToLower(pkg)

	return &p, nil
}

func RHELWFN(rpm *Package) (*wfn.Attributes, error) {
	var version string
	if start := strings.Index(rpm.Release, "el"); start >= 0 {
		start += 2 // skip "el"
		var end int
		for end = start; end < len(rpm.Release) && unicode.IsDigit(rune(rpm.Release[end])); end++ {
			continue
		}
		version = rpm.Release[start:end]
	}
	if version == "" {
		return nil, fmt.Errorf("can't parse rhel version from package name %q", rpm)
	}
	attr := wfn.NewAttributesWithAny()
	attr.Part = "o"
	attr.Vendor = "redhat"
	attr.Product = "enterprise_linux"
	attr.Version = version
	attr.Edition = "baseos"
	return attr, nil
}

func ParseRPMAndRHELWFN(pkg string) (*Package, *wfn.Attributes, error) {
	rpm, err := Parse(pkg)
	if err != nil {
		return nil, nil, err
	}
	d, err := RHELWFN(rpm)
	if err != nil {
		return nil, nil, err
	}
	return rpm, d, nil
}
