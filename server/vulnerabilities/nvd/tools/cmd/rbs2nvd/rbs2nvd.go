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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/providers/lib/client"
	"github.com/facebookincubator/nvdtools/providers/lib/runner"
	"github.com/facebookincubator/nvdtools/providers/rbs/api"
	"github.com/facebookincubator/nvdtools/providers/rbs/schema"
)

const (
	baseURL = "https://vulndb.cyberriskanalytics.com"
)

var tokenURL = baseURL + "/oauth/token"

func Read(r io.Reader, c chan runner.Convertible) error {
	var vulns map[string]*schema.Vulnerability
	if err := json.NewDecoder(r).Decode(&vulns); err != nil {
		return fmt.Errorf("can't decode into vulns: %v", err)
	}

	for _, vuln := range vulns {
		c <- vuln
	}

	return nil
}

func FetchSince(_ context.Context, c client.Client, baseURL string, since int64) (<-chan runner.Convertible, error) {
	clientID := os.Getenv("RBS_CLIENT_ID")
	if clientID == "" {
		return nil, fmt.Errorf("please set RBS_CLIENT_ID in environment")
	}
	clientSecret := os.Getenv("RBS_CLIENT_SECRET")
	if clientSecret == "" {
		return nil, fmt.Errorf("olease set RBS_CLIENT_SECRET in environment")
	}

	client := api.NewClient(c, baseURL, tokenURL, clientID, clientSecret)
	return client.FetchAllVulnerabilities(since)
}

func main() {
	flag.StringVar(&tokenURL, "token_url", tokenURL, "OAuth2 access token URL")

	r := runner.Runner{
		Config: runner.Config{
			BaseURL: baseURL,
			ClientConfig: client.Config{
				UserAgent: "rbs2nvd",
			},
		},
		FetchSince: FetchSince,
		Read:       Read,
	}

	if err := r.Run(); err != nil {
		flog.Errorln(err)
	}
}
