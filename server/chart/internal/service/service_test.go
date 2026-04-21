package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/chart/internal/types"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAuthorizer always allows access.
type mockAuthorizer struct{}

func (m *mockAuthorizer) Authorize(_ context.Context, _ platform_authz.AuthzTyper, _ platform_authz.Action) error {
	return nil
}

// mockDatastore implements types.Datastore for unit tests.
type mockDatastore struct {
	getSCDDataFunc             func(ctx context.Context, dataset string, startDate, endDate time.Time, bucketSize time.Duration, strategy api.SampleStrategy, hostFilter *types.HostFilter, entityIDs []string) ([]api.DataPoint, error)
	getHostIDsForFilterFunc    func(ctx context.Context, hostFilter *types.HostFilter) ([]uint, error)
	countHostsForChartFilterFn func(ctx context.Context, hostFilter *types.HostFilter) (int, error)
	findRecentlySeenHostIDsFn  func(ctx context.Context, lookback time.Duration) ([]uint, error)
	recordBucketDataFn         func(ctx context.Context, dataset string, bucketStart time.Time, bucketSize time.Duration, strategy api.SampleStrategy, entityBitmaps map[string][]byte) error
	recordBucketDataInvoked    bool
}

func (m *mockDatastore) FindRecentlySeenHostIDs(ctx context.Context, lookback time.Duration) ([]uint, error) {
	if m.findRecentlySeenHostIDsFn != nil {
		return m.findRecentlySeenHostIDsFn(ctx, lookback)
	}
	return nil, nil
}

func (m *mockDatastore) RecordBucketData(ctx context.Context, dataset string, bucketStart time.Time, bucketSize time.Duration, strategy api.SampleStrategy, entityBitmaps map[string][]byte) error {
	m.recordBucketDataInvoked = true
	if m.recordBucketDataFn != nil {
		return m.recordBucketDataFn(ctx, dataset, bucketStart, bucketSize, strategy, entityBitmaps)
	}
	return nil
}

func (m *mockDatastore) GetSCDData(ctx context.Context, dataset string, startDate, endDate time.Time, bucketSize time.Duration, strategy api.SampleStrategy, hostFilter *types.HostFilter, entityIDs []string) ([]api.DataPoint, error) {
	if m.getSCDDataFunc != nil {
		return m.getSCDDataFunc(ctx, dataset, startDate, endDate, bucketSize, strategy, hostFilter, entityIDs)
	}
	return nil, nil
}

func (m *mockDatastore) GetHostIDsForFilter(ctx context.Context, hostFilter *types.HostFilter) ([]uint, error) {
	if m.getHostIDsForFilterFunc != nil {
		return m.getHostIDsForFilterFunc(ctx, hostFilter)
	}
	return nil, nil
}

func (m *mockDatastore) CountHostsForChartFilter(ctx context.Context, hostFilter *types.HostFilter) (int, error) {
	if m.countHostsForChartFilterFn != nil {
		return m.countHostsForChartFilterFn(ctx, hostFilter)
	}
	return 0, nil
}

func (m *mockDatastore) CleanupSCDData(_ context.Context, _ int) error {
	return nil
}

func TestGetChartDataUnknownMetric(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, nil)

	_, err := svc.GetChartData(t.Context(), "nonexistent", api.RequestOpts{Days: 7})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown chart metric")
}

func TestGetChartDataInvalidDays(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 5})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid days value")
}

func TestGetChartDataInvalidDownsample(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	cases := []struct {
		name       string
		downsample int
	}{
		{"not a divisor of 24", 5},
		{"negative divisor of 24", -2},
		{"negative non-divisor", -5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 7, Downsample: tc.downsample})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid downsample value")
		})
	}
}

func TestGetChartDataHourlyPassesThrough(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	var gotBucketSize time.Duration
	var gotStart, gotEnd time.Time
	var gotStrategy api.SampleStrategy
	ds.getSCDDataFunc = func(_ context.Context, dataset string, start, end time.Time, bucketSize time.Duration, strategy api.SampleStrategy, _ *types.HostFilter, _ []string) ([]api.DataPoint, error) {
		assert.Equal(t, "uptime", dataset)
		gotBucketSize = bucketSize
		gotStart = start
		gotEnd = end
		gotStrategy = strategy
		return []api.DataPoint{{Timestamp: start, Value: 42}}, nil
	}
	ds.countHostsForChartFilterFn = func(_ context.Context, _ *types.HostFilter) (int, error) {
		return 200, nil
	}

	resp, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 7})
	require.NoError(t, err)
	assert.Equal(t, "uptime", resp.Metric)
	assert.Equal(t, "checkerboard", resp.Visualization)
	assert.Equal(t, "hourly", resp.Resolution)
	assert.Equal(t, 200, resp.TotalHosts)
	assert.Equal(t, 7, resp.Days)
	assert.Equal(t, time.Hour, gotBucketSize)
	assert.Equal(t, api.SampleStrategyAccumulate, gotStrategy)
	// Span must be exactly 7 days.
	assert.Equal(t, 7*24*time.Hour, gotEnd.Sub(gotStart))
}

func TestGetChartDataDownsampleResolution(t *testing.T) {
	for _, tc := range []struct {
		downsample int
		resolution string
		bucketSize time.Duration
	}{
		{0, "hourly", time.Hour},
		{1, "hourly", time.Hour},
		{2, "2-hour", 2 * time.Hour},
		{4, "4-hour", 4 * time.Hour},
		{8, "8-hour", 8 * time.Hour},
	} {
		t.Run(tc.resolution, func(t *testing.T) {
			ds := &mockDatastore{}
			svc := NewService(&mockAuthorizer{}, ds, nil)
			svc.RegisterDataset(&chart.UptimeDataset{})

			var gotBucketSize time.Duration
			ds.getSCDDataFunc = func(_ context.Context, _ string, _, _ time.Time, bucketSize time.Duration, _ api.SampleStrategy, _ *types.HostFilter, _ []string) ([]api.DataPoint, error) {
				gotBucketSize = bucketSize
				return nil, nil
			}

			resp, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 30, Downsample: tc.downsample})
			require.NoError(t, err)
			assert.Equal(t, tc.resolution, resp.Resolution)
			assert.Equal(t, tc.bucketSize, gotBucketSize)
		})
	}
}

func TestGetChartDataDailyIgnoresDownsample(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, nil)
	svc.RegisterDataset(&chart.CVEDataset{})

	var gotBucketSize time.Duration
	var gotStrategy api.SampleStrategy
	ds.getSCDDataFunc = func(_ context.Context, _ string, _, _ time.Time, bucketSize time.Duration, strategy api.SampleStrategy, _ *types.HostFilter, _ []string) ([]api.DataPoint, error) {
		gotBucketSize = bucketSize
		gotStrategy = strategy
		return nil, nil
	}

	resp, err := svc.GetChartData(t.Context(), "cve", api.RequestOpts{Days: 30, Downsample: 4})
	require.NoError(t, err)
	assert.Equal(t, "daily", resp.Resolution)
	assert.Equal(t, 24*time.Hour, gotBucketSize)
	assert.Equal(t, api.SampleStrategySnapshot, gotStrategy)
}

func TestGetChartDataWithHostFilters(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	ds.getSCDDataFunc = func(_ context.Context, _ string, _, _ time.Time, _ time.Duration, _ api.SampleStrategy, hostFilter *types.HostFilter, _ []string) ([]api.DataPoint, error) {
		require.NotNil(t, hostFilter)
		assert.Equal(t, []uint{1, 2}, hostFilter.LabelIDs)
		assert.Equal(t, []string{"darwin"}, hostFilter.Platforms)
		return []api.DataPoint{{Value: 2}}, nil
	}
	ds.countHostsForChartFilterFn = func(_ context.Context, hostFilter *types.HostFilter) (int, error) {
		require.NotNil(t, hostFilter)
		return 2, nil
	}

	resp, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{
		Days:      7,
		LabelIDs:  []uint{1, 2},
		Platforms: []string{"darwin"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, resp.TotalHosts)
	assert.Equal(t, []uint{1, 2}, resp.Filters.LabelIDs)
	assert.Equal(t, []string{"darwin"}, resp.Filters.Platforms)
}

func TestComputeBucketRange(t *testing.T) {
	t.Run("hourly UTC", func(t *testing.T) {
		now := time.Date(2026, 4, 8, 14, 37, 12, 0, time.UTC)
		start, end := computeBucketRange(now, time.Hour, 1, 0)
		assert.Equal(t, time.Date(2026, 4, 8, 14, 0, 0, 0, time.UTC), end)
		assert.Equal(t, time.Date(2026, 4, 7, 14, 0, 0, 0, time.UTC), start)
	})

	t.Run("hourly with downsample aligns to step", func(t *testing.T) {
		now := time.Date(2026, 4, 8, 15, 30, 0, 0, time.UTC)
		_, end := computeBucketRange(now, 4*time.Hour, 1, 0)
		// 15 / 4 * 4 = 12 — aligned to nearest step hour within the day.
		assert.Equal(t, time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC), end)
	})

	t.Run("hourly with tz offset aligns to local hour", func(t *testing.T) {
		// 14:37 UTC = 07:37 PDT (offset +420 minutes). Local hour 07 → end at 07:00 PDT = 14:00 UTC.
		now := time.Date(2026, 4, 8, 14, 37, 0, 0, time.UTC)
		_, end := computeBucketRange(now, time.Hour, 1, 420)
		assert.Equal(t, time.Date(2026, 4, 8, 14, 0, 0, 0, time.UTC), end)
	})

	t.Run("daily with tz offset aligns to local midnight", func(t *testing.T) {
		// 14:37 UTC = 07:37 PDT. Local midnight = 00:00 PDT = 07:00 UTC.
		now := time.Date(2026, 4, 8, 14, 37, 0, 0, time.UTC)
		start, end := computeBucketRange(now, 24*time.Hour, 7, 420)
		assert.Equal(t, time.Date(2026, 4, 8, 7, 0, 0, 0, time.UTC), end)
		assert.Equal(t, time.Date(2026, 4, 1, 7, 0, 0, 0, time.UTC), start)
	})
}

func TestCollectDatasetsUptime(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	now := time.Date(2026, 4, 8, 14, 37, 0, 0, time.UTC)
	wantBucketStart := time.Date(2026, 4, 8, 14, 0, 0, 0, time.UTC)

	ds.findRecentlySeenHostIDsFn = func(_ context.Context, lookback time.Duration) ([]uint, error) {
		assert.Equal(t, 10*time.Minute, lookback)
		return []uint{1, 2, 3}, nil
	}
	ds.recordBucketDataFn = func(_ context.Context, dataset string, bucketStart time.Time, bucketSize time.Duration, strategy api.SampleStrategy, entityBitmaps map[string][]byte) error {
		assert.Equal(t, "uptime", dataset)
		assert.Equal(t, wantBucketStart, bucketStart)
		assert.Equal(t, time.Hour, bucketSize)
		assert.Equal(t, api.SampleStrategyAccumulate, strategy)
		require.Len(t, entityBitmaps, 1)
		assert.NotEmpty(t, entityBitmaps[""])
		return nil
	}

	err := svc.CollectDatasets(t.Context(), now)
	require.NoError(t, err)
	assert.True(t, ds.recordBucketDataInvoked)
}

func TestUptimeDatasetMetadata(t *testing.T) {
	d := &chart.UptimeDataset{}
	assert.Equal(t, "uptime", d.Name())
	assert.Equal(t, time.Hour, d.BucketSize())
	assert.Equal(t, api.SampleStrategyAccumulate, d.SampleStrategy())
	assert.Equal(t, "checkerboard", d.DefaultVisualization())
	assert.False(t, d.HasEntityDimension())
	assert.Nil(t, d.SupportedFilters())

	entityIDs, err := d.ResolveFilters(t.Context(), nil, nil)
	require.NoError(t, err)
	assert.Nil(t, entityIDs)
}

func TestCVEDatasetMetadata(t *testing.T) {
	d := &chart.CVEDataset{}
	assert.Equal(t, "cve", d.Name())
	assert.Equal(t, 24*time.Hour, d.BucketSize())
	assert.Equal(t, api.SampleStrategySnapshot, d.SampleStrategy())
	assert.Equal(t, "line", d.DefaultVisualization())
	assert.True(t, d.HasEntityDimension())
}
