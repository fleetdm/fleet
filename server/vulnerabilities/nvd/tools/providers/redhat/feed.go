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
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/facebookincubator/nvdtools/providers/redhat/check"
	"github.com/facebookincubator/nvdtools/providers/redhat/schema"
	"github.com/facebookincubator/nvdtools/rpm"
	"github.com/facebookincubator/nvdtools/wfn"
)

// Feed is a collection of CVEs from RedHat.
type Feed struct {
	// Data is map of CVEs as returned by the redhat API, keyed by CVE names.
	Data map[string]*schema.CVE
	// pkgCVE is a package -> CVE map produced from data and cached.
	pkg2CVE packageFeed
	// rpm.Checker for this feed, cached.
	checker rpm.Checker
}

// LoadFeed loads a Feed from a JSON file.
func LoadFeed(path string) (*Feed, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("can't open file %q: %v", path, err)
	}
	defer f.Close()
	return loadFeed(f)
}

func loadFeed(r io.Reader) (*Feed, error) {
	var feed Feed
	if err := json.NewDecoder(r).Decode(&feed.Data); err != nil {
		return nil, fmt.Errorf("can't decode feed: %v", err)
	}
	return &feed, nil
}

// Checker returns an rpm.Checker that uses the Feed.
func (feed *Feed) Checker() (rpm.Checker, error) {
	if feed.checker != nil {
		return feed.checker, nil
	}
	mc := make(mapChecker, len(feed.Data))
	var err error
	for cveid, cve := range feed.Data {
		if mc[cveid], err = check.CVEChecker(cve); err != nil {
			if err == check.ErrCheckers {
				// no checkers could be created, just skip it
				delete(mc, cveid)
				continue
			}
			return nil, fmt.Errorf("can't create a checker for %q: %v", cveid, err)
		}
	}
	feed.checker = mc
	return mc, nil
}

// cve -> checker
type mapChecker map[string]rpm.Checker

// Check is part of the rpm.Check interface
func (c mapChecker) Check(pkg *rpm.Package, distro *wfn.Attributes, cve string) bool {
	if chk, ok := c[cve]; ok {
		return chk.Check(pkg, distro, cve)
	}
	return false
}
