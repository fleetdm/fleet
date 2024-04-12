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
	"sync"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/fireeye/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/stats"
)

// FetchAllThreatReportsSince will fetch all vulnerabilities with specified parameters
func (c *Client) FetchAllThreatReportsSince(ctx context.Context, since int64) (<-chan *schema.Report, error) {
	parameters := newParametersSince(since)
	if err := parameters.validate(); err != nil {
		return nil, err
	}

	// fetch indexes

	reportIDs := make(chan string)
	wgReportIDs := sync.WaitGroup{}

	for _, params := range parameters.batchBy(ninetyDays) {
		wgReportIDs.Add(1)
		params := params
		go func() {
			defer wgReportIDs.Done()
			flog.Infof("Fetching: %s\n", params)
			if rIDs, err := c.fetchReportIDs(ctx, params); err == nil {
				for _, rID := range rIDs {
					reportIDs <- rID
				}
			} else {
				flog.Errorln(err)
			}
		}()
	}

	go func() {
		wgReportIDs.Wait()
		close(reportIDs)
	}()

	// fetch reports

	reports := make(chan *schema.Report)
	wgReports := sync.WaitGroup{}

	for rID := range reportIDs {
		wgReports.Add(1)
		rID := rID
		go func() {
			defer wgReports.Done()
			if report, err := c.fetchReport(ctx, rID); err == nil {
				stats.IncrementCounter("report.success")
				reports <- report
			} else {
				stats.IncrementCounter("report.error")
				flog.Errorln(err)
			}
		}()
	}

	go func() {
		wgReports.Wait()
		close(reports)
	}()

	return reports, nil
}

func (c *Client) fetchReportIDs(ctx context.Context, parameters timeRangeParameters) ([]string, error) {
	resp, err := c.Request(ctx, fmt.Sprintf("/report/index?intelligenceType=threat&%s", parameters.query()))
	if err != nil {
		return nil, err
	}

	var reportIndex []*schema.ReportIndexItem
	if err := json.NewDecoder(resp).Decode(&reportIndex); err != nil {
		return nil, err
	}

	reportIDs := make([]string, len(reportIndex))
	for i := 0; i < len(reportIndex); i++ {
		reportIDs[i] = reportIndex[i].ReportID
	}

	return reportIDs, nil
}

func (c *Client) fetchReport(ctx context.Context, reportID string) (*schema.Report, error) {
	resp, err := c.Request(ctx, fmt.Sprintf("/report/%s?detail=full", reportID))
	if err != nil {
		return nil, err
	}

	var wrapper schema.ReportWrapper
	if err := json.NewDecoder(resp).Decode(&wrapper); err != nil {
		return nil, err
	}

	return &wrapper.Report, nil
}
