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

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/idefense/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/lib/client"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/lib/runner"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Client struct
type Client struct {
	client.Client
	baseURL string
	apiKey  string
}

const (
	pageSize              = 200
	vulnerabilityEndpoint = "/rest/vulnerability/v0"
)

// NewClient creates an object which is used to query the iDefense API
func NewClient(c client.Client, baseURL, apiKey string) *Client {
	return &Client{
		Client:  c,
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// FetchAllVulnerabilities will fetch all vulnerabilities from iDefense API
func (c *Client) FetchAllVulnerabilities(ctx context.Context, since int64) (<-chan runner.Convertible, error) {
	sinceStr := time.Unix(since, 0).Format("2006-01-02T15:04:05.000Z")

	result, err := c.queryVulnerabilities(ctx, map[string]interface{}{
		"last_modified.from":      sinceStr,
		"last_modified.inclusive": "true",
		"page_size":               0,
	})
	if err != nil {
		return nil, err
	}

	totalVulns := result.TotalSize
	if totalVulns == 0 {
		return nil, errors.New("no vulnerabilities found in given window")
	}

	output := make(chan runner.Convertible)
	numPages := (totalVulns-1)/pageSize + 1

	// fetch pages concurrently
	flog.Infof("starting sync for %d vulnerabilities over %d pages\n", totalVulns, numPages)
	eg, ctx := errgroup.WithContext(ctx)
	for page := 1; page <= numPages; page++ {
		page := page
		eg.Go(func() error {
			result, err := c.queryVulnerabilities(ctx, map[string]interface{}{
				"last_modified.from":      sinceStr,
				"last_modified.inclusive": "true",
				"page_size":               pageSize,
				"page":                    page,
			})
			if err != nil {
				return client.StopOrContinue(errors.Wrapf(err, "failed to get page %d: %v", page, err))
			}
			for _, vuln := range result.Results {
				if vuln != nil {
					output <- vuln
				}
			}
			return nil
		})
	}

	go func() {
		if err := eg.Wait(); err != nil {
			flog.Errorln(err)
		}
		close(output)
	}()

	return output, nil
}

func (c *Client) queryVulnerabilities(ctx context.Context, params map[string]interface{}) (*schema.VulnerabilitySearchResults, error) {
	u, err := url.Parse(c.baseURL + vulnerabilityEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse url")
	}
	query := url.Values{}
	for key, value := range params {
		query.Set(key, fmt.Sprintf("%v", value))
	}
	u.RawQuery = query.Encode()

	resp, err := client.Get(ctx, c, u.String(), http.Header{
		"Auth-Token": {c.apiKey},
	})
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// decode into json
	var result schema.VulnerabilitySearchResults
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode response")
	}

	return &result, nil
}
