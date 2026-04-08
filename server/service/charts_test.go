package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChartDataUnknownMetric(t *testing.T) {
	ds := new(mock.DataStore)
	cs := NewChartService(ds)

	_, err := cs.GetChartData(t.Context(), "nonexistent", fleet.ChartRequestOpts{Days: 7})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown chart metric")
}

func TestGetChartDataInvalidDays(t *testing.T) {
	ds := new(mock.DataStore)
	cs := NewChartService(ds)
	cs.RegisterDataset(&UptimeDataset{})

	_, err := cs.GetChartData(t.Context(), "uptime", fleet.ChartRequestOpts{Days: 5})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid days value")
}

func TestGetChartDataInvalidDownsample(t *testing.T) {
	ds := new(mock.DataStore)
	cs := NewChartService(ds)
	cs.RegisterDataset(&UptimeDataset{})

	_, err := cs.GetChartData(t.Context(), "uptime", fleet.ChartRequestOpts{Days: 7, Downsample: 3})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid downsample value")
}

func TestGetChartDataHourly(t *testing.T) {
	ds := new(mock.DataStore)
	cs := NewChartService(ds)
	cs.RegisterDataset(&UptimeDataset{})

	ds.GetChartDataFunc = func(ctx context.Context, dataset string, startDate, endDate time.Time, hostFilter *fleet.ChartHostFilter, entityIDs []uint, hasEntityDimension bool, downsample int) ([]fleet.ChartDataPoint, error) {
		assert.Equal(t, "uptime", dataset)
		assert.Equal(t, 0, downsample)
		assert.False(t, hasEntityDimension)
		assert.Nil(t, hostFilter)
		return []fleet.ChartDataPoint{
			{Timestamp: time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC), Value: 100},
		}, nil
	}
	ds.CountHostsForChartFilterFunc = func(ctx context.Context, hostFilter *fleet.ChartHostFilter) (int, error) {
		return 200, nil
	}

	resp, err := cs.GetChartData(t.Context(), "uptime", fleet.ChartRequestOpts{Days: 7})
	require.NoError(t, err)
	assert.Equal(t, "uptime", resp.Metric)
	assert.Equal(t, "line", resp.Visualization)
	assert.Equal(t, "hourly", resp.Resolution)
	assert.Equal(t, 200, resp.TotalHosts)
	assert.Equal(t, 7, resp.Days)
	assert.True(t, ds.GetChartDataFuncInvoked)
}

func TestGetChartDataDownsample(t *testing.T) {
	ds := new(mock.DataStore)
	cs := NewChartService(ds)
	cs.RegisterDataset(&UptimeDataset{})

	for _, tc := range []struct {
		downsample int
		resolution string
	}{
		{2, "2-hour"},
		{4, "4-hour"},
		{8, "8-hour"},
	} {
		t.Run(tc.resolution, func(t *testing.T) {
			ds.GetChartDataFunc = func(ctx context.Context, dataset string, startDate, endDate time.Time, hostFilter *fleet.ChartHostFilter, entityIDs []uint, hasEntityDimension bool, downsample int) ([]fleet.ChartDataPoint, error) {
				assert.Equal(t, tc.downsample, downsample)
				return nil, nil
			}
			ds.CountHostsForChartFilterFunc = func(ctx context.Context, hostFilter *fleet.ChartHostFilter) (int, error) {
				return 100, nil
			}

			resp, err := cs.GetChartData(t.Context(), "uptime", fleet.ChartRequestOpts{Days: 30, Downsample: tc.downsample})
			require.NoError(t, err)
			assert.Equal(t, tc.resolution, resp.Resolution)
		})
	}
}

func TestGetChartDataWithHostFilters(t *testing.T) {
	ds := new(mock.DataStore)
	cs := NewChartService(ds)
	cs.RegisterDataset(&UptimeDataset{})

	ds.GetChartDataFunc = func(ctx context.Context, dataset string, startDate, endDate time.Time, hostFilter *fleet.ChartHostFilter, entityIDs []uint, hasEntityDimension bool, downsample int) ([]fleet.ChartDataPoint, error) {
		require.NotNil(t, hostFilter)
		assert.Equal(t, []uint{1, 2}, hostFilter.LabelIDs)
		assert.Equal(t, []string{"darwin"}, hostFilter.Platforms)
		assert.Equal(t, []uint{99}, hostFilter.ExcludeHostIDs)
		return nil, nil
	}
	ds.CountHostsForChartFilterFunc = func(ctx context.Context, hostFilter *fleet.ChartHostFilter) (int, error) {
		require.NotNil(t, hostFilter)
		return 50, nil
	}

	resp, err := cs.GetChartData(t.Context(), "uptime", fleet.ChartRequestOpts{
		Days:           7,
		LabelIDs:       []uint{1, 2},
		Platforms:      []string{"darwin"},
		ExcludeHostIDs: []uint{99},
	})
	require.NoError(t, err)
	assert.Equal(t, 50, resp.TotalHosts)
	assert.Equal(t, []uint{1, 2}, resp.Filters.LabelIDs)
	assert.Equal(t, []string{"darwin"}, resp.Filters.Platforms)
}

func TestFillZeroValues(t *testing.T) {
	start := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 7, 5, 0, 0, 0, time.UTC)

	t.Run("hourly", func(t *testing.T) {
		data := []fleet.ChartDataPoint{
			{Timestamp: time.Date(2026, 4, 7, 2, 0, 0, 0, time.UTC), Value: 42},
		}
		result := fillZeroValues(data, start, end, 0)
		// Hours 0-5 = 6 data points
		assert.Len(t, result, 6)
		assert.Equal(t, 0, result[0].Value)
		assert.Equal(t, 0, result[1].Value)
		assert.Equal(t, 42, result[2].Value)
		assert.Equal(t, 0, result[3].Value)
	})

	t.Run("downsample 2", func(t *testing.T) {
		data := []fleet.ChartDataPoint{
			{Timestamp: time.Date(2026, 4, 7, 2, 0, 0, 0, time.UTC), Value: 42},
		}
		result := fillZeroValues(data, start, end, 2)
		// Hours 0,2,4 = 3 data points
		assert.Len(t, result, 3)
		assert.Equal(t, 0, result[0].Value)  // hour 0
		assert.Equal(t, 42, result[1].Value) // hour 2
		assert.Equal(t, 0, result[2].Value)  // hour 4
	})

	t.Run("downsample 4", func(t *testing.T) {
		data := []fleet.ChartDataPoint{
			{Timestamp: time.Date(2026, 4, 7, 4, 0, 0, 0, time.UTC), Value: 10},
		}
		result := fillZeroValues(data, start, end, 4)
		// Hours 0,4 = 2 data points
		assert.Len(t, result, 2)
		assert.Equal(t, 0, result[0].Value)  // hour 0
		assert.Equal(t, 10, result[1].Value) // hour 4
	})
}

func TestCollectDatasets(t *testing.T) {
	ds := new(mock.DataStore)
	cs := NewChartService(ds)
	cs.RegisterDataset(&UptimeDataset{})

	now := time.Date(2026, 4, 8, 14, 0, 0, 0, time.UTC)

	ds.CollectUptimeChartDataFunc = func(ctx context.Context, ts time.Time) error {
		assert.Equal(t, now, ts)
		return nil
	}

	err := cs.CollectDatasets(t.Context(), now)
	require.NoError(t, err)
	assert.True(t, ds.CollectUptimeChartDataFuncInvoked)
}

func TestUptimeDatasetMetadata(t *testing.T) {
	d := &UptimeDataset{}
	assert.Equal(t, "uptime", d.Name())
	assert.Equal(t, "line", d.DefaultVisualization())
	assert.False(t, d.HasEntityDimension())
	assert.Nil(t, d.SupportedFilters())

	entityIDs, err := d.ResolveFilters(t.Context(), nil, nil)
	require.NoError(t, err)
	assert.Nil(t, entityIDs)
}
