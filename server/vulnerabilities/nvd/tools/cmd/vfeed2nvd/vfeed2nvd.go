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
	"encoding/json"
	"fmt"
	"os"

	"github.com/facebookincubator/flog"
	nvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/vfeed/api"
)

const pathVar = "VFEED_REPO_PATH"

func main() {
	if err := run(); err != nil {
		flog.Fatal(err)
	}
}

func run() error {
	path := os.Getenv(pathVar)
	if path == "" {
		return fmt.Errorf("variable %s is not set", pathVar)
	}

	client := api.NewClient(path)

	// For now, FetchAllVulnerabilities disregards the "since" argument.
	items, err := client.FetchAllVulnerabilities(0)
	if err != nil {
		return fmt.Errorf("client failed to fetch vulnerabilities: %v", err)
	}

	var feed nvd.NVDCVEFeedJSON10

	for item := range items {
		nvdItem, err := item.Convert()
		if err != nil {
			return fmt.Errorf("failed to convert item %q: %v", item.ID(), err)
		}

		feed.CVEItems = append(feed.CVEItems, nvdItem)
	}

	if err := json.NewEncoder(os.Stdout).Encode(feed); err != nil {
		return fmt.Errorf("failed to encode nvd item: %v", err)
	}

	return nil
}
