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
	"fmt"
	"testing"

	"github.com/facebookincubator/nvdtools/rpm"
	"github.com/facebookincubator/nvdtools/wfn"
)

var (
	pkg *rpm.Package
)

func init() {
	pkg, _ = rpm.Parse("name-1:v2-rel.arch.rpm")
	// pkg = {
	// 	Name: "name",
	// 	Label: {
	// 		Epoch: "1",
	// 		Version: "v2",
	// 		Release: "rel",
	// 	},
	// 	Arch: "arch",
	// }
}

func TestConstPkgChecker(t *testing.T) {
	for i, tc := range []struct {
		conf   bool
		expect bool
	}{
		{true, true},
		{false, false},
	} {
		t.Run(fmt.Sprintf("case-%d", i+1), func(t *testing.T) {
			if got := constPkgChecker(tc.conf).checkPkg(pkg); got != tc.expect {
				t.Fatalf("pkg checker (%v) should return %v, got %v", tc.conf, tc.expect, got)
			}
		})
	}
}

func TestPackageStatePkgChecker(t *testing.T) {
	for i, tc := range []struct {
		conf   string
		expect bool
	}{
		{"name", true},
		{"something", false},
	} {
		t.Run(fmt.Sprintf("case-%d", i+1), func(t *testing.T) {
			if got := packageStatePkgChecker(tc.conf).checkPkg(pkg); got != tc.expect {
				t.Fatalf("pkg checker (%v) should return %v, got %v", tc.conf, tc.expect, got)
			}
		})
	}
}

func TestAffectedReleasePkgChecker(t *testing.T) {
	for i, tc := range []struct {
		conf   rpm.Package
		expect bool
	}{
		// config version is higher, so it means pkg wasn't fixed
		{rpm.Package{Name: "name", Label: rpm.Label{Epoch: "1", Version: "v3"}}, false},
		// config version is lower, so it means pkg was fixed
		{rpm.Package{Name: "name", Label: rpm.Label{Epoch: "1", Version: "v1"}}, true},
		// name is not the same, wasn't fixed
		{rpm.Package{Name: "name2"}, false},
		// arch is not the same, wasn't fixed
		{rpm.Package{Name: "name", Arch: "aaaa"}, false},
	} {
		t.Run(fmt.Sprintf("case-%d", i+1), func(t *testing.T) {
			if got := affectedReleasePkgChecker(tc.conf).checkPkg(pkg); got != tc.expect {
				t.Fatalf("pkg checker (%v) should return %v, got %v", tc.conf, tc.expect, got)
			}
		})
	}
}

func TestSingleChecker(t *testing.T) {
	// since we have unit tests for all other pkg checkers, we're just gonna use the const one for simplicity
	pc := constPkgChecker(true)
	cfgDistro := wfn.Attributes{
		Part:    "o",
		Vendor:  "vendor",
		Product: "product",
		Version: "4",
	}
	chk := &singleChecker{&cfgDistro, pc}

	for i, tc := range []struct {
		distro wfn.Attributes
		expect bool
	}{
		// version is the same as the fixed one, expecting true
		{wfn.Attributes{Part: "o", Vendor: "vendor", Product: "product", Version: "4"}, true},
		// part is different, expecting false
		{wfn.Attributes{Part: "a", Vendor: "vendor", Product: "product", Version: "4"}, false},
		// product name is different, expecting false
		{wfn.Attributes{Part: "a", Vendor: "vendor", Product: "producttt", Version: "4"}, false},
	} {
		t.Run(fmt.Sprintf("case-%d", i+1), func(t *testing.T) {
			// pkg can be anything other than nil since we're using const pkg checker
			if have := chk.Check(pkg, &tc.distro, ""); have != tc.expect {
				t.Fatalf("wrong check result for %q: have %v, expect %v", tc.distro.BindToURI(), have, tc.expect)
			}
		})
	}
}
