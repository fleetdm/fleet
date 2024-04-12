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

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

// ToWFN parses CPE name from RPM package name
func ToWFN(attr *wfn.Attributes, s string) error {
	pkg, err := Parse(s)
	if err != nil {
		return fmt.Errorf("can't get fields from %q: %v", s, err)
	}

	for n, addr := range map[string]*string{
		"name":    &pkg.Name,
		"version": &pkg.Label.Version,
		"release": &pkg.Label.Release,
		"arch":    &pkg.Arch,
	} {
		if *addr, err = wfn.WFNize(*addr); err != nil {
			return fmt.Errorf("couldn't wfnize %s %q: %v", n, *addr, err)
		}
	}

	if pkg.Name == "" {
		return fmt.Errorf("no name found in RPM name %q", s)
	}
	if pkg.Label.Version == "" {
		return fmt.Errorf("no version found in RPM name %q", s)
	}
	attr.Part = "a" // TODO: figure out the way to properly detect os packages (linux_kernel or smth)
	attr.Product = pkg.Name
	attr.Version = pkg.Label.Version
	attr.Update = pkg.Label.Release
	attr.TargetHW = pkg.Arch
	return nil
}
