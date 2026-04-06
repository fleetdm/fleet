package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Chart datasets registered at startup.
var chartDatasets = map[string]fleet.ChartDataset{}

// RegisterChartDataset registers a chart dataset by name.
func RegisterChartDataset(ds fleet.ChartDataset) {
	chartDatasets[ds.Name()] = ds
}

type getChartDataRequest struct {
	Metric         string `url:"metric"`
	Days           int    `query:"days,optional"`
	LabelIDs       string `query:"label_ids,optional"`
	Platforms      string `query:"platforms,optional"`
	IncludeHostIDs string `query:"include_host_ids,optional"`
	ExcludeHostIDs string `query:"exclude_host_ids,optional"`
}

type getChartDataResponse struct {
	*fleet.ChartResponse
	Err error `json:"error,omitempty"`
}

func (r getChartDataResponse) Error() error { return r.Err }

func getChartDataEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getChartDataRequest)

	days := req.Days
	if days == 0 {
		days = 7
	}

	opts := fleet.ChartRequestOpts{
		Days:           days,
		LabelIDs:       parseUintList(req.LabelIDs),
		Platforms:      parseStringList(req.Platforms),
		IncludeHostIDs: parseUintList(req.IncludeHostIDs),
		ExcludeHostIDs: parseUintList(req.ExcludeHostIDs),
		DatasetFilters: map[string]string{},
	}

	resp, err := svc.GetChartData(ctx, req.Metric, opts)
	if err != nil {
		return getChartDataResponse{Err: err}, nil
	}
	return getChartDataResponse{ChartResponse: resp}, nil
}

func (svc *Service) GetChartData(ctx context.Context, metric string, opts fleet.ChartRequestOpts) (*fleet.ChartResponse, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	dataset, ok := chartDatasets[metric]
	if !ok {
		return nil, &fleet.BadRequestError{Message: fmt.Sprintf("unknown chart metric: %s", metric)}
	}

	// Validate days preset.
	validDays := map[int]bool{1: true, 7: true, 14: true, 30: true}
	if !validDays[opts.Days] {
		return nil, &fleet.BadRequestError{Message: fmt.Sprintf("invalid days value: %d (must be 1, 7, 14, or 30)", opts.Days)}
	}

	// Resolve dataset-specific filters to entity IDs.
	entityIDs, err := dataset.ResolveFilters(ctx, svc.ds, opts.DatasetFilters)
	if err != nil {
		return nil, err
	}

	// Calculate date range.
	now := time.Now().UTC()
	endDate := now
	startDate := now.AddDate(0, 0, -opts.Days)

	// Build host filter.
	var hostFilter *fleet.ChartHostFilter
	if len(opts.LabelIDs) > 0 || len(opts.Platforms) > 0 || len(opts.IncludeHostIDs) > 0 || len(opts.ExcludeHostIDs) > 0 {
		hostFilter = &fleet.ChartHostFilter{
			LabelIDs:       opts.LabelIDs,
			Platforms:      opts.Platforms,
			IncludeHostIDs: opts.IncludeHostIDs,
			ExcludeHostIDs: opts.ExcludeHostIDs,
		}
	}

	downsample := opts.Days == 30

	data, err := svc.ds.GetChartData(ctx, metric, startDate, endDate, hostFilter, entityIDs, dataset.HasEntityDimension(), downsample)
	if err != nil {
		return nil, err
	}

	totalHosts, err := svc.ds.CountHostsForChartFilter(ctx, hostFilter)
	if err != nil {
		return nil, err
	}

	// Determine resolution label.
	resolution := "hourly"
	if downsample {
		resolution = "2-hour"
	}

	// Fill in zero-value entries for time buckets with no data.
	data = fillZeroValues(data, startDate, endDate, downsample)

	return &fleet.ChartResponse{
		Metric:        metric,
		Visualization: dataset.DefaultVisualization(),
		TotalHosts:    totalHosts,
		Resolution:    resolution,
		Days:          opts.Days,
		Filters: fleet.ChartFilters{
			LabelIDs:       opts.LabelIDs,
			Platforms:      opts.Platforms,
			IncludeHostIDs: opts.IncludeHostIDs,
			ExcludeHostIDs: opts.ExcludeHostIDs,
		},
		Data: data,
	}, nil
}

// fillZeroValues fills in missing time buckets with zero values.
func fillZeroValues(data []fleet.ChartDataPoint, startDate, endDate time.Time, downsample bool) []fleet.ChartDataPoint {
	existing := make(map[time.Time]int, len(data))
	for _, dp := range data {
		existing[dp.Timestamp] = dp.Value
	}

	step := 1
	if downsample {
		step = 2
	}

	// Align start to the beginning of its day.
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	// End at the current hour of endDate.
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), endDate.Hour(), 0, 0, 0, time.UTC)

	var result []fleet.ChartDataPoint
	for t := start; !t.After(end); t = t.Add(time.Duration(step) * time.Hour) {
		val, ok := existing[t]
		if !ok {
			val = 0
		}
		result = append(result, fleet.ChartDataPoint{
			Timestamp: t,
			Value:     val,
		})
	}
	return result
}

func parseUintList(s string) []uint {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if v, err := strconv.ParseUint(p, 10, 64); err == nil {
			result = append(result, uint(v))
		}
	}
	return result
}

func parseStringList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
