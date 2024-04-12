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
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/flexera/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/lib/client"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/lib/runner"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Client stores information needed to access  API
// API key will be sent in the Authorization field
// rate limiter is used to enforce their api limits (so we don't go over them)
type Client struct {
	client.Client
	baseURL string
	apiKey  string
}

const (
	pageSize           = 100
	advisoriesEndpoint = "/api/advisories"
	numFetchers        = 4
)

// NewClient creates a new Client object with given properties
func NewClient(c client.Client, baseURL, apiKey string) *Client {
	return &Client{
		Client:  c,
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// FetchAllVulnerabilities will fetch all advisories since given time
// we first fetch all pages and just collect all identifiers found on them and
// push them into the `identifiers` channel. Then we start fetchers which take
// those identifiers and fetch the real advisories
func (c *Client) FetchAllVulnerabilities(ctx context.Context, since int64) (<-chan runner.Convertible, error) {
	from, to := since, time.Now().Unix()
	totalAdvisories, err := c.getNumberOfAdvisories(ctx, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get total number of advisories")
	}

	mainCtx, cancel := context.WithCancel(ctx)

	numPages := (totalAdvisories-1)/pageSize + 1
	flog.Infof("starting sync for %d advisories over %d pages\n", totalAdvisories, numPages)

	identifiers := make(chan string, totalAdvisories)
	advisories := make(chan runner.Convertible, totalAdvisories)

	identifersEg, identifiersCtx := errgroup.WithContext(mainCtx)
	for page := 0; page < numPages; page++ {
		p := page + 1
		identifersEg.Go(func() error {
			list, err := c.fetchAdvisoryList(identifiersCtx, from, to, p)
			if err != nil {
				return client.StopOrContinue(errors.Wrapf(err, "failed to fetch page %d advisory list", p))
			}
			for _, element := range list.Results {
				identifiers <- element.AdvisoryIdentifier
			}
			return nil
		})
	}

	go func() {
		if err := identifersEg.Wait(); err != nil {
			flog.Errorln(err)
			cancel()
		}
		close(identifiers)
	}()

	advisoriesEg, advisoriesCtx := errgroup.WithContext(mainCtx)
	for i := 0; i < numFetchers; i++ {
		advisoriesEg.Go(func() error {
			for identifier := range identifiers {
				advisory, err := c.Fetch(advisoriesCtx, identifier)
				if err != nil {
					return client.StopOrContinue(errors.Wrapf(err, "failed to fetch advisory %s", identifier))
				}
				advisories <- advisory
			}
			return nil
		})
	}

	go func() {
		if err := advisoriesEg.Wait(); err != nil {
			flog.Errorln(err)
			cancel()
		}
		close(advisories)
	}()

	return advisories, nil
}

// Fetch will return a channel with only one advisory in it
func (c *Client) Fetch(ctx context.Context, identifier string) (*schema.Advisory, error) {
	var advisory schema.Advisory
	endpoint := fmt.Sprintf("%s/%s", advisoriesEndpoint, identifier)
	if err := c.query(ctx, endpoint, map[string]interface{}{}, &advisory); err != nil {
		return nil, errors.Wrapf(err, "failed to query advisory details endpoint %s", endpoint)
	}
	return &advisory, nil
}

func (c *Client) fetchAdvisoryList(ctx context.Context, from, to int64, page int) (*schema.AdvisoryListResult, error) {
	var list schema.AdvisoryListResult
	params := map[string]interface{}{
		"modified__gte": from,
		"modified__lt":  to,
		"page":          page,
		"page_size":     pageSize,
	}
	if err := c.query(ctx, advisoriesEndpoint, params, &list); err != nil {
		return nil, errors.Wrapf(err, "failed to fetch page %d", page)
	}
	return &list, nil
}

func (c *Client) getNumberOfAdvisories(ctx context.Context, from, to int64) (int, error) {
	var list schema.AdvisoryListResult
	params := map[string]interface{}{
		"modified__gte": from,
		"modified__lt":  to,
		"page_size":     1,
	}
	if err := c.query(ctx, advisoriesEndpoint, params, &list); err != nil {
		return 0, errors.Wrap(err, "failed to fetch first page")
	}
	return list.Count, nil
}

func (c *Client) query(ctx context.Context, endpoint string, params map[string]interface{}, v interface{}) error {
	// setup new parameters
	u, err := url.Parse(fmt.Sprintf("%s%s", c.baseURL, endpoint))
	if err != nil {
		return errors.Wrap(err, "failed to parse client URL")
	}
	query := u.Query()
	for key, value := range params {
		query.Set(key, fmt.Sprintf("%v", value))
	}
	u.RawQuery = query.Encode()

	// execute request
	resp, err := client.Get(ctx, c, u.String(), http.Header{"Authorization": {c.apiKey}})
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// decode into json
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return errors.Wrap(err, "failed to decode response")
	}

	return nil
}
