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
	"strconv"
	"sync"
	"time"

	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/providers/lib/client"
	"github.com/facebookincubator/nvdtools/providers/lib/runner"
	"github.com/facebookincubator/nvdtools/providers/redhat/schema"
)

const (
	perPage = 50
)

// Client struct
type Client struct {
	client.Client
	baseURL string
}

// NewClient creates an object which is used to query the RedHat API
func NewClient(c client.Client, baseURL string) *Client {
	return &Client{
		Client:  c,
		baseURL: baseURL,
	}
}

// FetchAllCVEs will fetch all vulnerabilities
func (c *Client) FetchAllCVEs(ctx context.Context, since int64) (<-chan runner.Convertible, error) {
	output := make(chan runner.Convertible)
	wg := sync.WaitGroup{}

	for page := range c.fetchAllPages(ctx, since) {
		for _, cveItem := range *page {
			wg.Add(1)
			go func(cveid string) {
				defer wg.Done()
				flog.Infof("\tfetching cve %s", cveid)
				cve, err := c.FetchCVE(ctx, cveid)
				if err != nil {
					flog.Errorf("error while fetching cve %s: %v", cveid, err)
					return
				}
				output <- cve
			}(cveItem.CVE)
		}
	}

	go func() {
		wg.Wait()
		close(output)
	}()

	return output, nil
}

// FetchCVE retrieves a single CVE.
func (c *Client) FetchCVE(ctx context.Context, cveid string) (*schema.CVE, error) {
	resp, err := c.queryPath(ctx, fmt.Sprintf("/cve/%s.json", cveid))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from feed: %v", err)
	}
	defer resp.Body.Close()

	var cve schema.CVE
	if err := json.NewDecoder(resp.Body).Decode(&cve); err != nil {
		return nil, fmt.Errorf("failed to decode cve response into a cve: %v", err)
	}

	return &cve, nil
}

func (c *Client) fetchAllPages(ctx context.Context, since int64) <-chan *schema.CVEList {
	output := make(chan *schema.CVEList)
	go func() {
		defer close(output)
		for page := 1; ; page++ {
			flog.Infof("fetching page %d", page)
			if list, err := c.fetchListPage(ctx, since, page); err == nil {
				output <- list
				if len(*list) < perPage {
					break
				}
			} else {
				flog.Errorf("can't fetch page %d: %v", page, err)
				break
			}
		}
	}()
	return output
}

func (c *Client) fetchListPage(ctx context.Context, since int64, page int) (*schema.CVEList, error) {
	params := url.Values{}
	params.Add("per_page", strconv.Itoa(perPage))
	params.Add("page", strconv.Itoa(page))
	params.Add("after", time.Unix(since, 0).Format("2006-01-02")) // YYYY-MM-DD

	resp, err := c.queryPath(ctx, "/cve.json?"+params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cve list: %v", err)
	}
	defer resp.Body.Close()

	var cveList schema.CVEList
	if err := json.NewDecoder(resp.Body).Decode(&cveList); err != nil {
		return nil, fmt.Errorf("failed to decode response into a list of cves: %v", err)
	}
	return &cveList, nil
}

func (c *Client) queryPath(ctx context.Context, path string) (*http.Response, error) {
	return client.Get(ctx, c, c.baseURL+path, http.Header{})
}
