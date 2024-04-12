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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/lib/client"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/redhat/api"
)

var fetchCVECmd = &cobra.Command{
	Use:   "fetch-cve CVE-XXXX-YYYY",
	Short: "fetch the latest information about a CVE",
	RunE:  fetchCVE,
}

func init() {
	rootCmd.AddCommand(fetchCVECmd)
}

func fetchCVE(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("fetch-cve: missing CVE name")
	}
	cveID := args[0]

	httpClient := client.Default()
	config := client.Config{
		UserAgent: "redhat_query",
	}
	httpClient = config.Configure(httpClient)

	feed := api.NewClient(httpClient, "https://access.redhat.com/labs/securitydataapi")
	cve, err := feed.FetchCVE(context.Background(), cveID)
	if err != nil {
		return err
	}

	output, err := json.MarshalIndent(cve, "", " ")
	if err != nil {
		return err
	}

	fmt.Println(string(output))

	return nil
}
