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

// Package api implements a client to fetch vulnerabilities from vfeed.
package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/vfeed/schema"
)

const suffixPattern = "/*/CVE-*.json"

// Client holds the state.
type Client struct {
	path string
}

// NewClient creates a Client.
func NewClient(path string) *Client {
	return &Client{
		path: path,
	}
}

// FetchAllVulnerabilities will return all vfeed items. The "since" parameter is
// ignored.
func (c *Client) FetchAllVulnerabilities(_ int64) (<-chan *schema.Item, error) {
	items := make(chan *schema.Item)

	matches, err := filepath.Glob(c.path + suffixPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob source files: %v", err)
	}

	go func() {
		defer close(items)

		for _, match := range matches {

			item, err := unmarshalFile(match)
			if err != nil {
				flog.Errorf("Failed to unmarshal %s: %v", match, err)
				return
			}

			items <- item
		}
	}()

	return items, nil
}

func unmarshalFile(path string) (*schema.Item, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	item := &schema.Item{}

	if err := json.Unmarshal(data, item); err != nil {
		return nil, err
	}

	return item, nil
}
