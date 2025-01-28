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

package nvd

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

var (
	e2eEnabled bool
	e2eSource  = NewSourceConfig()
	e2eTimeout = 5 * time.Minute
	e2eCVE     = cve20xmlGz
	e2eCPE     = cpe23xmlGz
)

// test: go test -v -args -v=1 -logtostderr -e2e_enabled
func init() {
	flag.BoolVar(&e2eEnabled, "e2e_enabled", e2eEnabled, "enable end-to-end test")
	flag.DurationVar(&e2eTimeout, "e2e_timeout", e2eTimeout, "timeout for end-to-end test")
	flag.Var(&e2eCVE, "e2e_cve_feed", e2eCVE.Help())
	flag.Var(&e2eCPE, "e2e_cpe_feed", e2eCPE.Help())
	e2eSource.AddFlags(flag.CommandLine)
}

func TestEndToEnd(t *testing.T) {
	if !e2eEnabled {
		t.Skip("e2e tests not enabled")
	}

	d, err := ioutil.TempDir("", "nvdsync-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(d)

	ds := Sync{
		Feeds:    []Syncer{e2eCVE, e2eCPE},
		Source:   e2eSource,
		LocalDir: d,
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eTimeout)
	defer cancel()

	if err = ds.Do(ctx); err != nil {
		t.Fatal(err)
	}
}
