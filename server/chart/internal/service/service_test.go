package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
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
	getBlobDataFunc            func(ctx context.Context, dataset string, startDate, endDate time.Time, entityIDs []string) ([]chart.BlobDataPoint, error)
	getHostIDsForFilterFunc    func(ctx context.Context, hostFilter *chart.HostFilter) ([]uint, error)
	countHostsForChartFilterFn func(ctx context.Context, hostFilter *chart.HostFilter) (int, error)
	collectUptimeFn            func(ctx context.Context, now time.Time) error
	collectUptimeInvoked       bool
}

func (m *mockDatastore) CountHostsForChartFilter(ctx context.Context, hostFilter *chart.HostFilter) (int, error) {
	return m.countHostsForChartFilterFn(ctx, hostFilter)
}

func (m *mockDatastore) CollectUptimeChartData(ctx context.Context, now time.Time) error {
	m.collectUptimeInvoked = true
	if m.collectUptimeFn != nil {
		return m.collectUptimeFn(ctx, now)
	}
	return nil
}

func (m *mockDatastore) CollectPolicyFailingChartData(_ context.Context, _ time.Time) error {
	return nil
}

func (m *mockDatastore) GetPolicyFailingSnapshot(_ context.Context) ([]chart.PolicyFailingSnapshot, error) {
	return nil, nil
}

func (m *mockDatastore) GetPoliciesMetadata(_ context.Context) ([]chart.PolicyMeta, error) {
	return nil, nil
}

func (m *mockDatastore) GetTeamsMetadata(_ context.Context) ([]chart.TeamMeta, error) {
	return nil, nil
}

func (m *mockDatastore) GetHostTeamAssignments(_ context.Context) ([]chart.HostTeam, error) {
	return nil, nil
}

func (m *mockDatastore) GetPolicyFailingByTeamTrend(_ context.Context, _, _ time.Time) ([]chart.TeamTrendPoint, error) {
	return nil, nil
}

func (m *mockDatastore) GetTopNonCompliantHosts(_ context.Context, _ int) ([]chart.HostFailingSummary, error) {
	return nil, nil
}

func (m *mockDatastore) GetBlobData(ctx context.Context, dataset string, startDate, endDate time.Time, entityIDs []string) ([]chart.BlobDataPoint, error) {
	if m.getBlobDataFunc != nil {
		return m.getBlobDataFunc(ctx, dataset, startDate, endDate, entityIDs)
	}
	return nil, nil
}

func (m *mockDatastore) GetHostIDsForFilter(ctx context.Context, hostFilter *chart.HostFilter) ([]uint, error) {
	if m.getHostIDsForFilterFunc != nil {
		return m.getHostIDsForFilterFunc(ctx, hostFilter)
	}
	return nil, nil
}

func (m *mockDatastore) CleanupBlobData(_ context.Context, _ int) error {
	return nil
}

func (m *mockDatastore) RecordSCDData(_ context.Context, _ string, _ map[string][]byte, _ time.Time) error {
	return nil
}

func (m *mockDatastore) GetSCDData(_ context.Context, _ string, _, _ time.Time, _ *chart.HostFilter, _ []string) ([]chart.DataPoint, error) {
	return nil, nil
}

func (m *mockDatastore) CleanupSCDData(_ context.Context, _ int) error {
	return nil
}

func TestGetChartDataUnknownMetric(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, nil)

	_, err := svc.GetChartData(t.Context(), "nonexistent", chart.RequestOpts{Days: 7})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown chart metric")
}

func TestGetChartDataInvalidDays(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	_, err := svc.GetChartData(t.Context(), "uptime", chart.RequestOpts{Days: 5})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid days value")
}

func TestGetChartDataInvalidDownsample(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	_, err := svc.GetChartData(t.Context(), "uptime", chart.RequestOpts{Days: 7, Downsample: 3})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid downsample value")
}

func TestGetChartDataBlobHourly(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	ds.getBlobDataFunc = func(ctx context.Context, dataset string, startDate, endDate time.Time, entityIDs []string) ([]chart.BlobDataPoint, error) {
		assert.Equal(t, "uptime", dataset)
		assert.Nil(t, entityIDs)
		// Return a blob for one hour: hosts 1, 2, 3 were online.
		return []chart.BlobDataPoint{
			{ChartDate: time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC), Hour: 10, HostBitmap: chart.HostIDsToBlob([]uint{1, 2, 3})},
		}, nil
	}
	ds.countHostsForChartFilterFn = func(ctx context.Context, hostFilter *chart.HostFilter) (int, error) {
		assert.Nil(t, hostFilter)
		return 200, nil
	}

	resp, err := svc.GetChartData(t.Context(), "uptime", chart.RequestOpts{Days: 7})
	require.NoError(t, err)
	assert.Equal(t, "uptime", resp.Metric)
	assert.Equal(t, "checkerboard", resp.Visualization)
	assert.Equal(t, "hourly", resp.Resolution)
	assert.Equal(t, 200, resp.TotalHosts)
	assert.Equal(t, 7, resp.Days)

	// Verify that the hour 10 data point has value 3 (three hosts).
	var found bool
	for _, dp := range resp.Data {
		if dp.Timestamp.Hour() == 10 && dp.Timestamp.Day() == 7 {
			assert.Equal(t, 3, dp.Value)
			found = true
		}
	}
	assert.True(t, found, "expected data point at hour 10")
}

func TestGetChartDataBlobDownsample(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	for _, tc := range []struct {
		downsample int
		resolution string
	}{
		{2, "2-hour"},
		{4, "4-hour"},
		{8, "8-hour"},
	} {
		t.Run(tc.resolution, func(t *testing.T) {
			// Blobs land on a day guaranteed to fall inside the 30-day window the
			// service computes from time.Now(), regardless of UTC hour.
			blobDay := time.Now().UTC().AddDate(0, 0, -5).Truncate(24 * time.Hour)
			// Return blobs for hours 0 and 1 with different hosts — downsampling should OR them.
			ds.getBlobDataFunc = func(ctx context.Context, dataset string, startDate, endDate time.Time, entityIDs []string) ([]chart.BlobDataPoint, error) {
				return []chart.BlobDataPoint{
					{ChartDate: blobDay, Hour: 0, HostBitmap: chart.HostIDsToBlob([]uint{1, 2})},
					{ChartDate: blobDay, Hour: 1, HostBitmap: chart.HostIDsToBlob([]uint{2, 3})},
				}, nil
			}
			ds.countHostsForChartFilterFn = func(ctx context.Context, hostFilter *chart.HostFilter) (int, error) {
				return 100, nil
			}

			resp, err := svc.GetChartData(t.Context(), "uptime", chart.RequestOpts{Days: 30, Downsample: tc.downsample})
			require.NoError(t, err)
			assert.Equal(t, tc.resolution, resp.Resolution)

			// The hour-0 bucket on blobDay should have 3 hosts (OR of {1,2} and {2,3} = {1,2,3}).
			var found bool
			for _, dp := range resp.Data {
				if dp.Timestamp.Year() == blobDay.Year() &&
					dp.Timestamp.Month() == blobDay.Month() &&
					dp.Timestamp.Day() == blobDay.Day() &&
					dp.Timestamp.Hour() == 0 {
					assert.Equal(t, 3, dp.Value)
					found = true
				}
			}
			assert.True(t, found, "expected data point for hour 0 on %s", blobDay.Format("2006-01-02"))
		})
	}
}

func TestGetChartDataBlobWithHostFilters(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	// Blob has hosts 1, 2, 3 online.
	ds.getBlobDataFunc = func(ctx context.Context, dataset string, startDate, endDate time.Time, entityIDs []string) ([]chart.BlobDataPoint, error) {
		return []chart.BlobDataPoint{
			{ChartDate: time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC), Hour: 10, HostBitmap: chart.HostIDsToBlob([]uint{1, 2, 3})},
		}, nil
	}
	// Filter returns only hosts 1, 3 (simulating a label filter).
	ds.getHostIDsForFilterFunc = func(ctx context.Context, hostFilter *chart.HostFilter) ([]uint, error) {
		assert.Equal(t, []uint{1, 2}, hostFilter.LabelIDs)
		assert.Equal(t, []string{"darwin"}, hostFilter.Platforms)
		return []uint{1, 3}, nil
	}

	resp, err := svc.GetChartData(t.Context(), "uptime", chart.RequestOpts{
		Days:      7,
		LabelIDs:  []uint{1, 2},
		Platforms: []string{"darwin"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, resp.TotalHosts) // len(filteredHostIDs)
	assert.Equal(t, []uint{1, 2}, resp.Filters.LabelIDs)
	assert.Equal(t, []string{"darwin"}, resp.Filters.Platforms)

	// Hour 10 should have value 2 (AND of {1,2,3} with {1,3} = {1,3}).
	var found bool
	for _, dp := range resp.Data {
		if dp.Timestamp.Hour() == 10 && dp.Timestamp.Day() == 7 {
			assert.Equal(t, 2, dp.Value)
			found = true
		}
	}
	assert.True(t, found, "expected filtered data point at hour 10")
}

func TestFillZeroValues(t *testing.T) {
	t.Run("hourly", func(t *testing.T) {
		// 5 hours apart → 5 buckets ending at hour 5
		start := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
		end := time.Date(2026, 4, 7, 5, 0, 0, 0, time.UTC)
		data := []chart.DataPoint{
			{Timestamp: time.Date(2026, 4, 7, 2, 0, 0, 0, time.UTC), Value: 42},
		}
		result := fillZeroValues(data, start, end, 0, 0)
		// 5 hours → 5 data points: hours 1,2,3,4,5
		assert.Len(t, result, 5)
		assert.Equal(t, 42, result[1].Value)
	})

	t.Run("downsample 2", func(t *testing.T) {
		start := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
		end := time.Date(2026, 4, 7, 6, 0, 0, 0, time.UTC)
		data := []chart.DataPoint{
			{Timestamp: time.Date(2026, 4, 7, 2, 0, 0, 0, time.UTC), Value: 42},
		}
		result := fillZeroValues(data, start, end, 2, 0)
		// 6 hours / 2 = 3 buckets ending at 6: hours 2, 4, 6
		assert.Len(t, result, 3)
		assert.Equal(t, 42, result[0].Value) // hour 2
		assert.Equal(t, 0, result[1].Value)  // hour 4
		assert.Equal(t, 0, result[2].Value)  // hour 6
	})

	t.Run("downsample 4", func(t *testing.T) {
		start := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
		end := time.Date(2026, 4, 7, 8, 0, 0, 0, time.UTC)
		data := []chart.DataPoint{
			{Timestamp: time.Date(2026, 4, 7, 4, 0, 0, 0, time.UTC), Value: 10},
		}
		result := fillZeroValues(data, start, end, 4, 0)
		// 8 hours / 4 = 2 buckets ending at 8: hours 4, 8
		assert.Len(t, result, 2)
		assert.Equal(t, 10, result[0].Value) // hour 4
		assert.Equal(t, 0, result[1].Value)  // hour 8
	})

	t.Run("24h gives exactly 24 points", func(t *testing.T) {
		// Simulates days=1: 24 hours apart, should give exactly 24 data points
		s := time.Date(2026, 4, 7, 16, 0, 0, 0, time.UTC)
		e := time.Date(2026, 4, 8, 16, 0, 0, 0, time.UTC)
		data := []chart.DataPoint{
			{Timestamp: time.Date(2026, 4, 7, 18, 0, 0, 0, time.UTC), Value: 50},
			{Timestamp: time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC), Value: 75},
		}
		result := fillZeroValues(data, s, e, 0, 0)
		require.Len(t, result, 24)
		// First point: hour 17 yesterday (end - 23h)
		assert.Equal(t, time.Date(2026, 4, 7, 17, 0, 0, 0, time.UTC), result[0].Timestamp)
		// Last point: hour 16 today (the current hour)
		assert.Equal(t, time.Date(2026, 4, 8, 16, 0, 0, 0, time.UTC), result[23].Timestamp)
		// Check values at known timestamps
		assert.Equal(t, 50, result[1].Value)  // hour 18 yesterday
		assert.Equal(t, 75, result[17].Value) // hour 10 today
	})

	t.Run("downsample aligns end hour to step", func(t *testing.T) {
		// End hour 17 with downsample=2 → end aligns to 16
		s := time.Date(2026, 4, 7, 11, 0, 0, 0, time.UTC)
		e := time.Date(2026, 4, 7, 17, 0, 0, 0, time.UTC)
		data := []chart.DataPoint{
			{Timestamp: time.Date(2026, 4, 7, 14, 0, 0, 0, time.UTC), Value: 30},
		}
		result := fillZeroValues(data, s, e, 2, 0)
		// 6 hours / 2 = 3 buckets ending at aligned hour 16: 12, 14, 16
		require.Len(t, result, 3)
		assert.Equal(t, time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC), result[0].Timestamp)
		assert.Equal(t, 30, result[1].Value) // hour 14
		assert.Equal(t, time.Date(2026, 4, 7, 16, 0, 0, 0, time.UTC), result[2].Timestamp)
	})

	t.Run("downsample 4 aligns end hour", func(t *testing.T) {
		// 14 hours apart, end hour 15 aligns to 12, 14/4=3 buckets
		s := time.Date(2026, 4, 7, 1, 0, 0, 0, time.UTC)
		e := time.Date(2026, 4, 7, 15, 0, 0, 0, time.UTC)
		data := []chart.DataPoint{
			{Timestamp: time.Date(2026, 4, 7, 8, 0, 0, 0, time.UTC), Value: 55},
		}
		result := fillZeroValues(data, s, e, 4, 0)
		// End aligns to 12, 14 hours / 4 = 3 buckets: 4, 8, 12
		require.Len(t, result, 3)
		assert.Equal(t, time.Date(2026, 4, 7, 4, 0, 0, 0, time.UTC), result[0].Timestamp)
		assert.Equal(t, 55, result[1].Value) // hour 8
		assert.Equal(t, time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC), result[2].Timestamp)
	})
}

func TestCollectDatasets(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	now := time.Date(2026, 4, 8, 14, 0, 0, 0, time.UTC)

	ds.collectUptimeFn = func(ctx context.Context, ts time.Time) error {
		assert.Equal(t, now, ts)
		return nil
	}

	err := svc.CollectDatasets(t.Context(), now)
	require.NoError(t, err)
	assert.True(t, ds.collectUptimeInvoked)
}

func TestUptimeDatasetMetadata(t *testing.T) {
	d := &chart.UptimeDataset{}
	assert.Equal(t, "uptime", d.Name())
	assert.Equal(t, chart.StorageTypeBlob, d.StorageType())
	assert.Equal(t, "checkerboard", d.DefaultVisualization())
	assert.False(t, d.HasEntityDimension())
	assert.Nil(t, d.SupportedFilters())

	entityIDs, err := d.ResolveFilters(t.Context(), nil, nil)
	require.NoError(t, err)
	assert.Nil(t, entityIDs)
}
