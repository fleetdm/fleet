package service

import (
	"context"
	"errors"
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

// recordingAuthorizer captures the subject and action handed to Authorize so
// tests can assert against the authz input. allow controls the return value.
type recordingAuthorizer struct {
	gotSubject platform_authz.AuthzTyper
	gotAction  platform_authz.Action
	allow      bool
}

func (r *recordingAuthorizer) Authorize(_ context.Context, subject platform_authz.AuthzTyper, action platform_authz.Action) error {
	r.gotSubject = subject
	r.gotAction = action
	if r.allow {
		return nil
	}
	return errors.New("forbidden")
}

// mockViewerProvider returns pre-programmed viewer scope. Default (zero
// value) represents a global user — convenient for the many tests that don't
// care about team scoping.
type mockViewerProvider struct {
	isGlobal bool
	teamIDs  []uint
	err      error
}

func (m *mockViewerProvider) ViewerScope(_ context.Context) (bool, []uint, error) {
	return m.isGlobal, m.teamIDs, m.err
}

// globalViewer returns a viewer provider for a global user (sees everything).
func globalViewer() *mockViewerProvider { return &mockViewerProvider{isGlobal: true} }

// mockDatastore implements types.Datastore for unit tests.
type mockDatastore struct {
	getSCDDataFunc             func(ctx context.Context, dataset string, startDate, endDate time.Time, bucketSize time.Duration, strategy api.SampleStrategy, filterMask []byte, entityIDs []string) ([]api.DataPoint, error)
	getHostIDsForFilterFunc    func(ctx context.Context, hostFilter *types.HostFilter) ([]uint, error)
	findRecentlySeenHostIDsFn  func(ctx context.Context, since time.Time, disabledFleetIDs []uint) ([]uint, error)
	affectedHostIDsByCVEFn     func(ctx context.Context, disabledFleetIDs []uint) (map[string][]uint, error)
	trackedCriticalCVEsFn      func(ctx context.Context) ([]string, error)
	recordBucketDataFn         func(ctx context.Context, dataset string, bucketStart time.Time, bucketSize time.Duration, strategy api.SampleStrategy, entityBitmaps map[string][]byte) error
	recordBucketDataInvoked    bool
	deleteAllForDatasetFn      func(ctx context.Context, dataset string, batchSize int) error
	hostIDsInFleetsFn          func(ctx context.Context, fleetIDs []uint) ([]uint, error)
	applyScrubMaskFn           func(ctx context.Context, dataset string, mask []byte, batchSize int) error
}

func (m *mockDatastore) FindRecentlySeenHostIDs(ctx context.Context, since time.Time, disabledFleetIDs []uint) ([]uint, error) {
	if m.findRecentlySeenHostIDsFn != nil {
		return m.findRecentlySeenHostIDsFn(ctx, since, disabledFleetIDs)
	}
	return nil, nil
}

func (m *mockDatastore) AffectedHostIDsByCVE(ctx context.Context, disabledFleetIDs []uint) (map[string][]uint, error) {
	if m.affectedHostIDsByCVEFn != nil {
		return m.affectedHostIDsByCVEFn(ctx, disabledFleetIDs)
	}
	return nil, nil
}

func (m *mockDatastore) TrackedCriticalCVEs(ctx context.Context) ([]string, error) {
	if m.trackedCriticalCVEsFn != nil {
		return m.trackedCriticalCVEsFn(ctx)
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

func (m *mockDatastore) GetSCDData(ctx context.Context, dataset string, startDate, endDate time.Time, bucketSize time.Duration, strategy api.SampleStrategy, filterMask []byte, entityIDs []string) ([]api.DataPoint, error) {
	if m.getSCDDataFunc != nil {
		return m.getSCDDataFunc(ctx, dataset, startDate, endDate, bucketSize, strategy, filterMask, entityIDs)
	}
	return nil, nil
}

func (m *mockDatastore) GetHostIDsForFilter(ctx context.Context, hostFilter *types.HostFilter) ([]uint, error) {
	if m.getHostIDsForFilterFunc != nil {
		return m.getHostIDsForFilterFunc(ctx, hostFilter)
	}
	return nil, nil
}

func (m *mockDatastore) CleanupSCDData(_ context.Context, _ int) error {
	return nil
}

func (m *mockDatastore) DeleteAllForDataset(ctx context.Context, dataset string, batchSize int) error {
	if m.deleteAllForDatasetFn != nil {
		return m.deleteAllForDatasetFn(ctx, dataset, batchSize)
	}
	return nil
}

func (m *mockDatastore) HostIDsInFleets(ctx context.Context, fleetIDs []uint) ([]uint, error) {
	if m.hostIDsInFleetsFn != nil {
		return m.hostIDsInFleetsFn(ctx, fleetIDs)
	}
	return nil, nil
}

func (m *mockDatastore) ApplyScrubMaskToDataset(ctx context.Context, dataset string, mask []byte, batchSize int) error {
	if m.applyScrubMaskFn != nil {
		return m.applyScrubMaskFn(ctx, dataset, mask, batchSize)
	}
	return nil
}

func TestGetChartDataUnknownMetric(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)

	_, err := svc.GetChartData(t.Context(), "nonexistent", api.RequestOpts{Days: 7})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown chart metric")
}

func TestGetChartDataInvalidDays(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 5})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid days value")
}

func TestGetChartDataInvalidResolution(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	cases := []struct {
		name       string
		resolution int
	}{
		{"not a divisor of 24", 5},
		{"negative divisor of 24", -2},
		{"negative non-divisor", -5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 7, Resolution: tc.resolution})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid resolution value")
		})
	}
}

func TestGetChartDataUptimeDefault(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	// Drive TotalHosts via the host-ID list: bitmap popcount = 200.
	ds.getHostIDsForFilterFunc = func(_ context.Context, _ *types.HostFilter) ([]uint, error) {
		ids := make([]uint, 200)
		for i := range ids {
			ids[i] = uint(i + 1)
		}
		return ids, nil
	}

	var gotBucketSize time.Duration
	var gotStart, gotEnd time.Time
	var gotStrategy api.SampleStrategy
	var gotMask []byte
	ds.getSCDDataFunc = func(_ context.Context, dataset string, start, end time.Time, bucketSize time.Duration, strategy api.SampleStrategy, mask []byte, _ []string) ([]api.DataPoint, error) {
		assert.Equal(t, "uptime", dataset)
		gotBucketSize = bucketSize
		gotStart = start
		gotEnd = end
		gotStrategy = strategy
		gotMask = mask
		return []api.DataPoint{{Timestamp: start, Value: 42}}, nil
	}

	resp, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 7})
	require.NoError(t, err)
	assert.Equal(t, "uptime", resp.Metric)
	assert.Equal(t, "checkerboard", resp.Visualization)
	assert.Equal(t, "3-hour", resp.Resolution)
	assert.Equal(t, 200, resp.TotalHosts)
	assert.Equal(t, 7, resp.Days)
	assert.Equal(t, 3*time.Hour, gotBucketSize)
	assert.Equal(t, api.SampleStrategyAccumulate, gotStrategy)
	assert.Equal(t, 200, chart.BlobPopcount(gotMask), "filter mask should encode all 200 host IDs")
	// Span must be exactly 7 days.
	assert.Equal(t, 7*24*time.Hour, gotEnd.Sub(gotStart))
}

func TestGetChartDataUptimeResolution(t *testing.T) {
	for _, tc := range []struct {
		resolution    int
		resolutionStr string
		bucketSize    time.Duration
	}{
		{0, "3-hour", 3 * time.Hour},
		{1, "hourly", time.Hour},
		{2, "2-hour", 2 * time.Hour},
		{4, "4-hour", 4 * time.Hour},
		{8, "8-hour", 8 * time.Hour},
	} {
		t.Run(tc.resolutionStr, func(t *testing.T) {
			ds := &mockDatastore{}
			svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
			svc.RegisterDataset(&chart.UptimeDataset{})

			var gotBucketSize time.Duration
			ds.getSCDDataFunc = func(_ context.Context, _ string, _, _ time.Time, bucketSize time.Duration, _ api.SampleStrategy, _ []byte, _ []string) ([]api.DataPoint, error) {
				gotBucketSize = bucketSize
				return nil, nil
			}

			resp, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 30, Resolution: tc.resolution})
			require.NoError(t, err)
			assert.Equal(t, tc.resolutionStr, resp.Resolution)
			assert.Equal(t, tc.bucketSize, gotBucketSize)
		})
	}
}

func TestGetChartDataCVEResolution(t *testing.T) {
	// Resolution applies uniformly regardless of the dataset's default:
	// omitted → dataset default (24h for CVE), specified → that value in hours.
	for _, tc := range []struct {
		name          string
		resolution    int
		resolutionStr string
		bucketSize    time.Duration
	}{
		{"default", 0, "3-hour", 3 * time.Hour},
		{"hourly override", 1, "hourly", time.Hour},
		{"4-hour override", 4, "4-hour", 4 * time.Hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds := &mockDatastore{}
			svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
			svc.RegisterDataset(&chart.CVEDataset{})

			var gotBucketSize time.Duration
			var gotStrategy api.SampleStrategy
			ds.getSCDDataFunc = func(_ context.Context, _ string, _, _ time.Time, bucketSize time.Duration, strategy api.SampleStrategy, _ []byte, _ []string) ([]api.DataPoint, error) {
				gotBucketSize = bucketSize
				gotStrategy = strategy
				return nil, nil
			}

			resp, err := svc.GetChartData(t.Context(), "cve", api.RequestOpts{Days: 30, Resolution: tc.resolution})
			require.NoError(t, err)
			assert.Equal(t, tc.resolutionStr, resp.Resolution)
			assert.Equal(t, tc.bucketSize, gotBucketSize)
			assert.Equal(t, api.SampleStrategySnapshot, gotStrategy)
		})
	}
}

func TestGetChartDataCVEUsesCuratedFilter(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
	svc.RegisterDataset(&chart.CVEDataset{})

	ds.trackedCriticalCVEsFn = func(_ context.Context) ([]string, error) {
		return []string{"CVE-A", "CVE-B"}, nil
	}
	var gotEntityIDs []string
	ds.getSCDDataFunc = func(_ context.Context, _ string, _, _ time.Time, _ time.Duration, _ api.SampleStrategy, _ []byte, entityIDs []string) ([]api.DataPoint, error) {
		gotEntityIDs = entityIDs
		return nil, nil
	}

	_, err := svc.GetChartData(t.Context(), "cve", api.RequestOpts{Days: 7})
	require.NoError(t, err)
	assert.Equal(t, []string{"CVE-A", "CVE-B"}, gotEntityIDs)
}

func TestGetChartDataCVEEmptySetReturnsZeros(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
	svc.RegisterDataset(&chart.CVEDataset{})

	// Non-nil empty slice — the resolver produced no matches but a filter
	// was requested. The service MUST pass this through verbatim so the
	// storage layer's "AND 1=0" path fires.
	ds.trackedCriticalCVEsFn = func(_ context.Context) ([]string, error) {
		return []string{}, nil
	}
	var gotEntityIDs []string
	gotEntityIDsIsNil := true
	ds.getSCDDataFunc = func(_ context.Context, _ string, startDate, endDate time.Time, bucketSize time.Duration, _ api.SampleStrategy, _ []byte, entityIDs []string) ([]api.DataPoint, error) {
		gotEntityIDs = entityIDs
		gotEntityIDsIsNil = entityIDs == nil
		numBuckets := int(endDate.Sub(startDate) / bucketSize)
		points := make([]api.DataPoint, numBuckets)
		for i := range points {
			points[i] = api.DataPoint{Timestamp: startDate.Add(time.Duration(i+1) * bucketSize), Value: 0}
		}
		return points, nil
	}

	resp, err := svc.GetChartData(t.Context(), "cve", api.RequestOpts{Days: 7})
	require.NoError(t, err)
	assert.False(t, gotEntityIDsIsNil, "service must pass non-nil empty slice so storage layer emits AND 1=0")
	assert.Empty(t, gotEntityIDs)
	require.NotEmpty(t, resp.Data)
	for _, dp := range resp.Data {
		assert.Zero(t, dp.Value)
	}
}

func TestGetChartDataUptimePassesNilEntityIDs(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	// Stub TrackedCriticalCVEs so an accidental call would fail loudly.
	ds.trackedCriticalCVEsFn = func(_ context.Context) ([]string, error) {
		t.Fatal("uptime path must not call TrackedCriticalCVEs")
		return nil, nil
	}
	gotEntityIDsIsNil := false
	ds.getSCDDataFunc = func(_ context.Context, _ string, _, _ time.Time, _ time.Duration, _ api.SampleStrategy, _ []byte, entityIDs []string) ([]api.DataPoint, error) {
		gotEntityIDsIsNil = entityIDs == nil
		return nil, nil
	}

	_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 7})
	require.NoError(t, err)
	assert.True(t, gotEntityIDsIsNil, "uptime must pass nil entityIDs — the CVE branch must not leak")
}

func TestGetChartDataWithHostFilters(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	var gotFilter *types.HostFilter
	ds.getHostIDsForFilterFunc = func(_ context.Context, hostFilter *types.HostFilter) ([]uint, error) {
		gotFilter = hostFilter
		return []uint{10, 20}, nil
	}
	ds.getSCDDataFunc = func(_ context.Context, _ string, _, _ time.Time, _ time.Duration, _ api.SampleStrategy, mask []byte, _ []string) ([]api.DataPoint, error) {
		assert.Equal(t, 2, chart.BlobPopcount(mask), "mask should encode the 2 host IDs returned")
		return []api.DataPoint{{Value: 2}}, nil
	}

	teamID := uint(5)
	resp, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{
		Days:      7,
		TeamID:    &teamID,
		LabelIDs:  []uint{1, 2},
		Platforms: []string{"darwin"},
	})
	require.NoError(t, err)

	require.NotNil(t, gotFilter)
	assert.Equal(t, []uint{5}, gotFilter.TeamIDs, "explicit team_id becomes a single-element scope")
	assert.Equal(t, []uint{1, 2}, gotFilter.LabelIDs)
	assert.Equal(t, []string{"darwin"}, gotFilter.Platforms)

	assert.Equal(t, 2, resp.TotalHosts, "TotalHosts is now popcount of filter mask")
	require.NotNil(t, resp.Filters.TeamID)
	assert.Equal(t, uint(5), *resp.Filters.TeamID, "response echoes what the caller asked for")
	assert.Equal(t, []uint{1, 2}, resp.Filters.LabelIDs)
	assert.Equal(t, []string{"darwin"}, resp.Filters.Platforms)
}

func TestGetChartDataAuthzScope(t *testing.T) {
	t.Run("no fleet_id → ActionList with Host{} (rego allows team users)", func(t *testing.T) {
		auth := &recordingAuthorizer{allow: true}
		svc := NewService(auth, &mockDatastore{}, globalViewer(), nil)
		svc.RegisterDataset(&chart.UptimeDataset{})

		_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 7})
		require.NoError(t, err)

		host, ok := auth.gotSubject.(*api.Host)
		require.True(t, ok, "authz subject should be *api.Host")
		assert.Nil(t, host.TeamID, "without an explicit fleet_id, the subject's TeamID stays nil")
		assert.Equal(t, platform_authz.ActionList, auth.gotAction,
			"no fleet_id uses ActionList so rego's team-list rule can pass team users")
	})

	t.Run("explicit fleet_id=5 → ActionRead with Host{TeamID:5} (rego enforces exact team)", func(t *testing.T) {
		auth := &recordingAuthorizer{allow: true}
		svc := NewService(auth, &mockDatastore{}, globalViewer(), nil)
		svc.RegisterDataset(&chart.UptimeDataset{})

		teamID := uint(5)
		_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 7, TeamID: &teamID})
		require.NoError(t, err)

		host, ok := auth.gotSubject.(*api.Host)
		require.True(t, ok)
		require.NotNil(t, host.TeamID)
		assert.Equal(t, uint(5), *host.TeamID)
		assert.Equal(t, platform_authz.ActionRead, auth.gotAction,
			"explicit fleet_id uses ActionRead so rego's team-read rule can enforce exact-team access")
	})

	t.Run("authz denial propagates", func(t *testing.T) {
		auth := &recordingAuthorizer{allow: false}
		svc := NewService(auth, &mockDatastore{}, globalViewer(), nil)
		svc.RegisterDataset(&chart.UptimeDataset{})

		_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 7})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden")
	})

	t.Run("viewer provider error propagates before authz", func(t *testing.T) {
		auth := &recordingAuthorizer{allow: true}
		viewer := &mockViewerProvider{err: errors.New("no viewer in context")}
		svc := NewService(auth, &mockDatastore{}, viewer, nil)
		svc.RegisterDataset(&chart.UptimeDataset{})

		_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 7})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no viewer")
		assert.Nil(t, auth.gotSubject, "authz must not run when viewer resolution failed")
	})
}

func TestGetChartDataScopesDataByViewer(t *testing.T) {
	t.Run("global user, no fleet_id → nil TeamIDs (no team filter)", func(t *testing.T) {
		ds := &mockDatastore{}
		var gotFilter *types.HostFilter
		ds.getHostIDsForFilterFunc = func(_ context.Context, f *types.HostFilter) ([]uint, error) {
			gotFilter = f
			return []uint{1, 2, 3}, nil
		}
		svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
		svc.RegisterDataset(&chart.UptimeDataset{})

		_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 7})
		require.NoError(t, err)
		require.NotNil(t, gotFilter)
		assert.Nil(t, gotFilter.TeamIDs, "global user with no fleet_id gets no team filter")
	})

	t.Run("team user, no fleet_id → their accessible teams", func(t *testing.T) {
		ds := &mockDatastore{}
		var gotFilter *types.HostFilter
		ds.getHostIDsForFilterFunc = func(_ context.Context, f *types.HostFilter) ([]uint, error) {
			gotFilter = f
			return []uint{10, 11}, nil
		}
		viewer := &mockViewerProvider{isGlobal: false, teamIDs: []uint{3, 7}}
		svc := NewService(&mockAuthorizer{}, ds, viewer, nil)
		svc.RegisterDataset(&chart.UptimeDataset{})

		_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 7})
		require.NoError(t, err)
		require.NotNil(t, gotFilter)
		assert.Equal(t, []uint{3, 7}, gotFilter.TeamIDs,
			"team user without explicit fleet_id is scoped to the union of their teams")
	})

	t.Run("team user with zero accessible teams → empty non-nil TeamIDs (SQL no-match)", func(t *testing.T) {
		ds := &mockDatastore{}
		var gotFilter *types.HostFilter
		ds.getHostIDsForFilterFunc = func(_ context.Context, f *types.HostFilter) ([]uint, error) {
			gotFilter = f
			return nil, nil
		}
		viewer := &mockViewerProvider{isGlobal: false, teamIDs: nil}
		svc := NewService(&mockAuthorizer{}, ds, viewer, nil)
		svc.RegisterDataset(&chart.UptimeDataset{})

		resp, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 7})
		require.NoError(t, err)
		require.NotNil(t, gotFilter)
		require.NotNil(t, gotFilter.TeamIDs, "empty-not-nil signals 'team-scoped with no teams'")
		assert.Empty(t, gotFilter.TeamIDs)
		assert.Equal(t, 0, resp.TotalHosts, "no accessible teams means no hosts and no data")
	})

	t.Run("explicit fleet_id overrides viewer scope", func(t *testing.T) {
		ds := &mockDatastore{}
		var gotFilter *types.HostFilter
		ds.getHostIDsForFilterFunc = func(_ context.Context, f *types.HostFilter) ([]uint, error) {
			gotFilter = f
			return []uint{1}, nil
		}
		// Viewer sees teams 3, 7 — but caller explicitly asks for team 3.
		viewer := &mockViewerProvider{isGlobal: false, teamIDs: []uint{3, 7}}
		svc := NewService(&mockAuthorizer{}, ds, viewer, nil)
		svc.RegisterDataset(&chart.UptimeDataset{})

		teamID := uint(3)
		_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 7, TeamID: &teamID})
		require.NoError(t, err)
		require.NotNil(t, gotFilter)
		assert.Equal(t, []uint{3}, gotFilter.TeamIDs,
			"explicit fleet_id narrows to that team; authz (not the filter) enforced access above")
	})
}

func TestComputeBucketRange(t *testing.T) {
	t.Run("hourly UTC", func(t *testing.T) {
		now := time.Date(2026, 4, 8, 14, 37, 12, 0, time.UTC)
		start, end := computeBucketRange(now, time.Hour, 1, 0)
		assert.Equal(t, time.Date(2026, 4, 8, 14, 0, 0, 0, time.UTC), end)
		assert.Equal(t, time.Date(2026, 4, 7, 14, 0, 0, 0, time.UTC), start)
	})

	t.Run("sub-daily resolution aligns to step", func(t *testing.T) {
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
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	now := time.Date(2026, 4, 8, 14, 37, 0, 0, time.UTC)
	wantBucketStart := time.Date(2026, 4, 8, 14, 0, 0, 0, time.UTC)

	ds.findRecentlySeenHostIDsFn = func(_ context.Context, since time.Time, _ []uint) ([]uint, error) {
		assert.Equal(t, now.Add(-10*time.Minute), since)
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

	err := svc.CollectDatasets(t.Context(), now, nil)
	require.NoError(t, err)
	assert.True(t, ds.recordBucketDataInvoked)
}

func TestCollectDatasetsCVE(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
	svc.RegisterDataset(&chart.CVEDataset{})

	now := time.Date(2026, 4, 8, 14, 37, 0, 0, time.UTC)
	wantBucketStart := time.Date(2026, 4, 8, 14, 0, 0, 0, time.UTC)

	ds.affectedHostIDsByCVEFn = func(_ context.Context, _ []uint) (map[string][]uint, error) {
		return map[string][]uint{
			"CVE-2024-0001": {1, 2, 3},
			"CVE-2024-0002": {2, 4},
		}, nil
	}
	ds.recordBucketDataFn = func(_ context.Context, dataset string, bucketStart time.Time, bucketSize time.Duration, strategy api.SampleStrategy, entityBitmaps map[string][]byte) error {
		assert.Equal(t, "cve", dataset)
		assert.Equal(t, wantBucketStart, bucketStart)
		assert.Equal(t, time.Hour, bucketSize)
		assert.Equal(t, api.SampleStrategySnapshot, strategy)
		require.Len(t, entityBitmaps, 2)
		assert.NotEmpty(t, entityBitmaps["CVE-2024-0001"])
		assert.NotEmpty(t, entityBitmaps["CVE-2024-0002"])
		return nil
	}

	err := svc.CollectDatasets(t.Context(), now, nil)
	require.NoError(t, err)
	assert.True(t, ds.recordBucketDataInvoked)
}

// TestCollectDatasetsForwardsScope verifies the scope resolver wiring:
//   - skip=true → Collect not invoked
//   - skip=false → disabledFleetIDs forwarded to the store query
//   - nil scope → equivalent to (false, nil) for every dataset
func TestCollectDatasetsForwardsScope(t *testing.T) {
	now := time.Date(2026, 4, 8, 14, 37, 0, 0, time.UTC)

	t.Run("skip prevents Collect call", func(t *testing.T) {
		ds := &mockDatastore{}
		svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
		svc.RegisterDataset(&chart.UptimeDataset{})
		ds.findRecentlySeenHostIDsFn = func(_ context.Context, _ time.Time, _ []uint) ([]uint, error) {
			t.Fatal("FindRecentlySeenHostIDs should not have been called when scope returned skip=true")
			return nil, nil
		}
		err := svc.CollectDatasets(t.Context(), now, func(name string) (bool, []uint) {
			assert.Equal(t, "uptime", name)
			return true, nil
		})
		require.NoError(t, err)
		assert.False(t, ds.recordBucketDataInvoked)
	})

	t.Run("disabledFleetIDs forwarded to FindRecentlySeenHostIDs", func(t *testing.T) {
		ds := &mockDatastore{}
		svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
		svc.RegisterDataset(&chart.UptimeDataset{})

		var gotDisabled []uint
		ds.findRecentlySeenHostIDsFn = func(_ context.Context, _ time.Time, disabled []uint) ([]uint, error) {
			gotDisabled = disabled
			return []uint{1}, nil
		}
		ds.recordBucketDataFn = func(_ context.Context, _ string, _ time.Time, _ time.Duration, _ api.SampleStrategy, _ map[string][]byte) error {
			return nil
		}
		err := svc.CollectDatasets(t.Context(), now, func(_ string) (bool, []uint) {
			return false, []uint{5, 7}
		})
		require.NoError(t, err)
		assert.Equal(t, []uint{5, 7}, gotDisabled)
	})

	t.Run("disabledFleetIDs forwarded to AffectedHostIDsByCVE", func(t *testing.T) {
		ds := &mockDatastore{}
		svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
		svc.RegisterDataset(&chart.CVEDataset{})

		var gotDisabled []uint
		ds.affectedHostIDsByCVEFn = func(_ context.Context, disabled []uint) (map[string][]uint, error) {
			gotDisabled = disabled
			return map[string][]uint{"CVE-1": {1}}, nil
		}
		ds.recordBucketDataFn = func(_ context.Context, _ string, _ time.Time, _ time.Duration, _ api.SampleStrategy, _ map[string][]byte) error {
			return nil
		}
		err := svc.CollectDatasets(t.Context(), now, func(_ string) (bool, []uint) {
			return false, []uint{5, 7}
		})
		require.NoError(t, err)
		assert.Equal(t, []uint{5, 7}, gotDisabled)
	})

	t.Run("nil scope behaves as (false, nil) for every dataset", func(t *testing.T) {
		ds := &mockDatastore{}
		svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
		svc.RegisterDataset(&chart.UptimeDataset{})

		var gotDisabled []uint = []uint{0xDEADBEEF} // sentinel — should be replaced with nil
		ds.findRecentlySeenHostIDsFn = func(_ context.Context, _ time.Time, disabled []uint) ([]uint, error) {
			gotDisabled = disabled
			return []uint{1}, nil
		}
		ds.recordBucketDataFn = func(_ context.Context, _ string, _ time.Time, _ time.Duration, _ api.SampleStrategy, _ map[string][]byte) error {
			return nil
		}
		err := svc.CollectDatasets(t.Context(), now, nil)
		require.NoError(t, err)
		assert.Nil(t, gotDisabled)
	})
}

func TestScrubDatasetGlobal(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)

	var gotDataset string
	var gotBatchSize int
	ds.deleteAllForDatasetFn = func(_ context.Context, dataset string, batchSize int) error {
		gotDataset = dataset
		gotBatchSize = batchSize
		return nil
	}

	require.NoError(t, svc.ScrubDatasetGlobal(t.Context(), "uptime"))
	assert.Equal(t, "uptime", gotDataset)
	assert.Equal(t, scrubBatchSize, gotBatchSize)
}

func TestScrubDatasetFleet(t *testing.T) {
	t.Run("forwards mask built from fleet hosts", func(t *testing.T) {
		ds := &mockDatastore{}
		svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)

		var gotFleets []uint
		ds.hostIDsInFleetsFn = func(_ context.Context, fleetIDs []uint) ([]uint, error) {
			gotFleets = fleetIDs
			return []uint{3, 7, 12}, nil
		}

		var gotDataset string
		var gotMask []byte
		var gotBatchSize int
		ds.applyScrubMaskFn = func(_ context.Context, dataset string, mask []byte, batchSize int) error {
			gotDataset = dataset
			gotMask = mask
			gotBatchSize = batchSize
			return nil
		}

		require.NoError(t, svc.ScrubDatasetFleet(t.Context(), "cve", []uint{5, 7}))
		assert.Equal(t, []uint{5, 7}, gotFleets)
		assert.Equal(t, "cve", gotDataset)
		assert.Equal(t, scrubBatchSize, gotBatchSize)
		// Mask must have bits set at positions 3, 7, 12.
		assert.Equal(t, 3, chart.BlobPopcount(gotMask))
	})

	t.Run("empty fleet IDs is no-op", func(t *testing.T) {
		ds := &mockDatastore{}
		svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
		ds.hostIDsInFleetsFn = func(_ context.Context, _ []uint) ([]uint, error) {
			t.Fatal("HostIDsInFleets should not have been called for empty input")
			return nil, nil
		}
		ds.applyScrubMaskFn = func(_ context.Context, _ string, _ []byte, _ int) error {
			t.Fatal("ApplyScrubMaskToDataset should not have been called for empty input")
			return nil
		}
		require.NoError(t, svc.ScrubDatasetFleet(t.Context(), "uptime", nil))
		require.NoError(t, svc.ScrubDatasetFleet(t.Context(), "uptime", []uint{}))
	})

	t.Run("no hosts resolved is no-op", func(t *testing.T) {
		ds := &mockDatastore{}
		svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
		ds.hostIDsInFleetsFn = func(_ context.Context, _ []uint) ([]uint, error) {
			return nil, nil
		}
		ds.applyScrubMaskFn = func(_ context.Context, _ string, _ []byte, _ int) error {
			t.Fatal("ApplyScrubMaskToDataset should not be called when no hosts resolved")
			return nil
		}
		require.NoError(t, svc.ScrubDatasetFleet(t.Context(), "cve", []uint{5}))
	})
}

func TestUptimeDatasetMetadata(t *testing.T) {
	d := &chart.UptimeDataset{}
	assert.Equal(t, "uptime", d.Name())
	assert.Equal(t, 3, d.DefaultResolutionHours())
	assert.Equal(t, api.SampleStrategyAccumulate, d.SampleStrategy())
	assert.Equal(t, "checkerboard", d.DefaultVisualization())
}

func TestCVEDatasetMetadata(t *testing.T) {
	d := &chart.CVEDataset{}
	assert.Equal(t, "cve", d.Name())
	assert.Equal(t, 3, d.DefaultResolutionHours())
	assert.Equal(t, api.SampleStrategySnapshot, d.SampleStrategy())
	assert.Equal(t, "line", d.DefaultVisualization())
}
