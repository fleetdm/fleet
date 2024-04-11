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
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/providers/lib/client"
	"github.com/facebookincubator/nvdtools/providers/snyk/schema"
)

type Client struct {
	client.Client
	baseURL    string
	consumerID string
	secret     string
}

func NewClient(c client.Client, baseURL, consumerID, secret string) *Client {
	return &Client{
		Client:     c,
		consumerID: consumerID,
		secret:     secret,
		baseURL:    baseURL,
	}
}

func (c *Client) FetchAllVulnerabilities(ctx context.Context, _ int64) (<-chan *schema.Advisory, error) {
	// since is ignored, always download all from snyk
	content, err := c.get(ctx, "application_premium")
	if err != nil {
		return nil, fmt.Errorf("can't get vulnerabilities: %v", err)
	}

	gzipRdr, err := gzip.NewReader(content)
	if err != nil {
		return nil, fmt.Errorf("can't decode gzip data: %v", err)
	}
	jsonRdr := json.NewDecoder(gzipRdr)

	output := make(chan *schema.Advisory)
	go func() {
		defer close(output)
		defer gzipRdr.Close()
		defer content.Close()
		for {
			var advisory schema.Advisory
			if err := jsonRdr.Decode(&advisory); err != nil {
				if err != io.EOF {
					flog.Errorf("can't decode content into advisories: %v", err)
					return
				}
				break
			}
			output <- &advisory
		}
	}()

	return output, nil
}

func (c *Client) get(ctx context.Context, feed string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/%s/intel_feed/%s?version=%s", c.baseURL, c.consumerID, feed, time.Now().UTC().Format("2006-01-02"))
	resp, err := client.Get(ctx, c, url, http.Header{
		"Accept":        {"application/vnd.api+json"},
		"Authorization": {"Token " + c.secret},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get vulnerabilities at %q: %v", url, err)
	}
	defer resp.Body.Close()
	var jsonResp schema.RestAPI
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return nil, fmt.Errorf("failed to decode vulnerabilities: %v", err)
	}

	url = jsonResp.Data.URL
	resp, err = client.Get(ctx, c, url, http.Header{
		"Accept-Encoding": {"gzip"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get vulnerabilities at %q: %v", url, err)
	}

	return resp.Body, nil
}
