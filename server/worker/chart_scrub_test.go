package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	chart_api "github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeChartService implements chart_api.Service. Only the scrub methods are
// exercised here; the rest panic to surface accidental call paths during tests.
type fakeChartService struct {
	scrubGlobalFn func(ctx context.Context, dataset string) error
	scrubFleetFn  func(ctx context.Context, dataset string, fleetIDs []uint) error
}

func (f *fakeChartService) GetChartData(_ context.Context, _ string, _ chart_api.RequestOpts) (*chart_api.Response, error) {
	panic("not used")
}
func (f *fakeChartService) RegisterDataset(_ chart_api.Dataset) { panic("not used") }
func (f *fakeChartService) CollectDatasets(_ context.Context, _ time.Time, _ chart_api.CollectScopeFn) error {
	panic("not used")
}
func (f *fakeChartService) CleanupData(_ context.Context, _ int) error { panic("not used") }
func (f *fakeChartService) ScrubDatasetGlobal(ctx context.Context, dataset string) error {
	return f.scrubGlobalFn(ctx, dataset)
}
func (f *fakeChartService) ScrubDatasetFleet(ctx context.Context, dataset string, fleetIDs []uint) error {
	return f.scrubFleetFn(ctx, dataset, fleetIDs)
}

func TestChartScrubGlobalRun(t *testing.T) {
	t.Run("forwards dataset to service", func(t *testing.T) {
		var got string
		fake := &fakeChartService{scrubGlobalFn: func(_ context.Context, ds string) error {
			got = ds
			return nil
		}}
		j := &ChartScrubGlobal{ChartService: fake}
		args, _ := json.Marshal(ChartScrubGlobalArgs{Dataset: "uptime"})
		require.NoError(t, j.Run(t.Context(), args))
		assert.Equal(t, "uptime", got)
	})

	t.Run("rejects empty dataset", func(t *testing.T) {
		fake := &fakeChartService{scrubGlobalFn: func(_ context.Context, _ string) error {
			t.Fatal("service should not be invoked for empty dataset")
			return nil
		}}
		j := &ChartScrubGlobal{ChartService: fake}
		args, _ := json.Marshal(ChartScrubGlobalArgs{})
		require.Error(t, j.Run(t.Context(), args))
	})

	t.Run("propagates service error", func(t *testing.T) {
		boom := errors.New("boom")
		fake := &fakeChartService{scrubGlobalFn: func(_ context.Context, _ string) error {
			return boom
		}}
		j := &ChartScrubGlobal{ChartService: fake}
		args, _ := json.Marshal(ChartScrubGlobalArgs{Dataset: "uptime"})
		require.ErrorIs(t, j.Run(t.Context(), args), boom)
	})

	t.Run("rejects malformed args JSON", func(t *testing.T) {
		j := &ChartScrubGlobal{ChartService: &fakeChartService{}}
		require.Error(t, j.Run(t.Context(), json.RawMessage("not-json")))
	})
}

func TestChartScrubFleetRun(t *testing.T) {
	t.Run("forwards dataset and fleet IDs to service", func(t *testing.T) {
		var gotDS string
		var gotFleets []uint
		fake := &fakeChartService{scrubFleetFn: func(_ context.Context, ds string, fleets []uint) error {
			gotDS = ds
			gotFleets = fleets
			return nil
		}}
		j := &ChartScrubFleet{ChartService: fake}
		args, _ := json.Marshal(ChartScrubFleetArgs{Dataset: "cve", FleetIDs: []uint{5, 7, 11}})
		require.NoError(t, j.Run(t.Context(), args))
		assert.Equal(t, "cve", gotDS)
		assert.Equal(t, []uint{5, 7, 11}, gotFleets)
	})

	t.Run("empty fleet list is no-op", func(t *testing.T) {
		fake := &fakeChartService{scrubFleetFn: func(_ context.Context, _ string, _ []uint) error {
			t.Fatal("service should not be invoked for empty fleet list")
			return nil
		}}
		j := &ChartScrubFleet{ChartService: fake}
		args, _ := json.Marshal(ChartScrubFleetArgs{Dataset: "cve"})
		require.NoError(t, j.Run(t.Context(), args))
	})

	t.Run("rejects empty dataset", func(t *testing.T) {
		fake := &fakeChartService{scrubFleetFn: func(_ context.Context, _ string, _ []uint) error {
			t.Fatal("service should not be invoked for empty dataset")
			return nil
		}}
		j := &ChartScrubFleet{ChartService: fake}
		args, _ := json.Marshal(ChartScrubFleetArgs{FleetIDs: []uint{5}})
		require.Error(t, j.Run(t.Context(), args))
	})

	t.Run("propagates service error", func(t *testing.T) {
		boom := errors.New("boom")
		fake := &fakeChartService{scrubFleetFn: func(_ context.Context, _ string, _ []uint) error {
			return boom
		}}
		j := &ChartScrubFleet{ChartService: fake}
		args, _ := json.Marshal(ChartScrubFleetArgs{Dataset: "cve", FleetIDs: []uint{5}})
		require.ErrorIs(t, j.Run(t.Context(), args), boom)
	})
}

// TestChartScrubJobNamesMatchFleetMirror locks the wire contract between the
// fleet-package enqueue helper (which can't import the worker package due to
// an import cycle) and this package's job names + payload struct shapes.
// If this test fails, update the constants in server/fleet/historical_data.go
// to match — workers won't pick up jobs that get enqueued under stale names.
func TestChartScrubJobNamesMatchFleetMirror(t *testing.T) {
	require.Equal(t, "chart_scrub_dataset_global", ChartScrubDatasetGlobalJobName)
	require.Equal(t, "chart_scrub_dataset_fleet", ChartScrubDatasetFleetJobName)

	// Round-trip the args through JSON to verify field names match.
	gJSON, err := json.Marshal(ChartScrubGlobalArgs{Dataset: "uptime"})
	require.NoError(t, err)
	require.JSONEq(t, `{"dataset":"uptime"}`, string(gJSON))

	fJSON, err := json.Marshal(ChartScrubFleetArgs{Dataset: "cve", FleetIDs: []uint{5, 7}})
	require.NoError(t, err)
	require.JSONEq(t, `{"dataset":"cve","fleet_ids":[5,7]}`, string(fJSON))
}
