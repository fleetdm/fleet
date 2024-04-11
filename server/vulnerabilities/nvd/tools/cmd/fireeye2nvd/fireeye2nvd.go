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
	"fmt"
	"io"
	"os"

	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/providers/fireeye/api"
	"github.com/facebookincubator/nvdtools/providers/fireeye/schema"
	"github.com/facebookincubator/nvdtools/providers/lib/client"
	"github.com/facebookincubator/nvdtools/providers/lib/runner"
)

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

func FetchSince(ctx context.Context, c client.Client, baseURL string, since int64) (<-chan runner.Convertible, error) {
	publicKey := os.Getenv("FIREEYE_PUBLIC_KEY")
	if publicKey == "" {
		return nil, fmt.Errorf("please set FIREEYE_PUBLIC_KEY in environment")
	}
	privateKey := os.Getenv("FIREEYE_PRIVATE_KEY")
	if privateKey == "" {
		return nil, fmt.Errorf("please set FIREEYE_PRIVATE_KEY in environment")
	}

	client := api.NewClient(c, baseURL, publicKey, privateKey)
	return client.FetchAllVulnerabilities(ctx, since)
}

func main() {
	r := runner.Runner{
		Config: runner.Config{
			BaseURL: "https://api.isightpartners.com",
			ClientConfig: client.Config{
				UserAgent: "fireeye2nvd",
			},
		},
		FetchSince: FetchSince,
		Read:       Read,
	}

	if err := r.Run(); err != nil {
		flog.Fatalln(err)
	}
}
