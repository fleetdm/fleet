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
	"reflect"
	"strings"
	"testing"

	"github.com/facebookincubator/nvdtools/rpm"
	"github.com/facebookincubator/nvdtools/wfn"
)

func TestFilter(t *testing.T) {
	cfg := config{
		pkgs:    0,
		distro:  1,
		cve:     2,
		pkgsSep: ",",
	}
	chk := testChecker("foo")

	for i, tc := range []struct {
		in, out []string
		fail    bool
	}{
		// pkgs,distro,cve
		{
			in:  []string{`name-epoch:version-release.arch.src,cpe:/a:redhat:enterprise_linux:7,CVE-X-Y`},
			out: []string{`name-epoch:version-release.arch.src,cpe:/a:redhat:enterprise_linux:7,CVE-X-Y`},
		},
		{
			in:  []string{`"foo-epoch:version-release.arch.src,bar-epoch:version-release.arch.src",cpe:/a:redhat:enterprise_linux:7,CVE-X-Y`},
			out: []string{`bar-epoch:version-release.arch.src,cpe:/a:redhat:enterprise_linux:7,CVE-X-Y`},
		},
		{
			in:  []string{`"foo-epoch:version-release.arch.src,not a rpm package",cpe:/a:redhat:enterprise_linux:7,CVE-X-Y`},
			out: []string{`not a rpm package,cpe:/a:redhat:enterprise_linux:7,CVE-X-Y`},
		},
		{
			in: []string{
				`"foo-epoch:version-release.arch.src,bar-epoch:version-release.arch.src",cpe:/a:redhat:enterprise_linux:7,CVE-X-Y`,
				`"foo-epoch:version-release.arch.src,not a rpm package",cpe:/a:redhat:enterprise_linux:7,CVE-X-Y`,
			},
			out: []string{
				`bar-epoch:version-release.arch.src,cpe:/a:redhat:enterprise_linux:7,CVE-X-Y`,
				`not a rpm package,cpe:/a:redhat:enterprise_linux:7,CVE-X-Y`,
			},
		},
		{
			in:   []string{`package,not a valid cpe so should fail,some cve`},
			fail: true,
		},
		{
			in:   []string{`wrong number of fields,should be 3 but have 2`},
			fail: true,
		},
		{
			in:  []string{`package,cpe:/,cve,some,additional,things,shouldn't change anything!`},
			out: []string{`package,cpe:/,cve,some,additional,things,shouldn't change anything!`},
		},
	} {
		t.Run(fmt.Sprintf("case-%d", i+1), func(t *testing.T) {
			r := strings.NewReader(strings.Join(tc.in, "\n"))
			var w strings.Builder

			if err := filter(chk, &cfg, r, &w); err != nil {
				if !tc.fail {
					t.Fatalf("shouldn't have failed for %v, but did: %v", tc.in, err)
				}
				return
			}
			if tc.fail {
				t.Fatalf("should've failed for %v, but didn't", tc.in)
			}

			want := tc.out
			if have := strings.Split(strings.TrimSpace(w.String()), "\n"); !reflect.DeepEqual(have, want) {
				t.Fatalf("wrong output. have: %v, want: %v", have, want)
			}
		})
	}
}

// fixed packages are the ones with this name
type testChecker string

func (name testChecker) Check(pkg *rpm.Package, _ *wfn.Attributes, _ string) bool {
	return pkg.Name == string(name)
}
