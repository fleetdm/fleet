package service

import (
	"context"
	"errors"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/chart/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
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

// stubLicense implements license.LicenseChecker. The chart context can't import
// server/fleet (arch_test forbids it, tests included), so tests can't build a
// real fleet.LicenseInfo and stub the interface instead.
type stubLicense struct{ premium bool }

func (s stubLicense) IsPremium() bool               { return s.premium }
func (s stubLicense) IsAllowDisableTelemetry() bool { return false }
func (s stubLicense) GetTier() string {
	if s.premium {
		return "premium"
	}
	return "free"
}
func (s stubLicense) GetOrganization() string { return "test" }
func (s stubLicense) GetDeviceCount() int     { return 1 }

func requirePremiumRequired(t *testing.T, err error) {
	t.Helper()
	var msgErr *platform_http.UserMessageError
	require.ErrorAs(t, err, &msgErr)
	require.Equal(t, http.StatusPaymentRequired, msgErr.StatusCode())
	require.Contains(t, err.Error(), "Fleet Premium")
}

func premiumCtx(t *testing.T) context.Context {
	t.Helper()
	return license.NewContext(t.Context(), stubLicense{premium: true})
}

func freeCtx(t *testing.T) context.Context {
	t.Helper()
	return license.NewContext(t.Context(), stubLicense{premium: false})
}

type mockDatastore struct {
	getSCDDataFunc          func(ctx context.Context, dataset string, startDate, endDate time.Time, bucketSize time.Duration, strategy api.SampleStrategy, filterMask *roaring.Bitmap, entityIDs []string) ([]api.DataPoint, error)
	getHostIDsForFilterFunc func(ctx context.Context, hostFilter *types.HostFilter) ([]uint, error)
	findOnlineHostIDsFn     func(ctx context.Context, now time.Time, disabledFleetIDs []uint) ([]uint, error)
	affectedHostIDsByCVEFn  func(ctx context.Context, disabledFleetIDs []uint, cves []string) (map[string]*roaring.Bitmap, error)
	collectibleCVEsFn       func(ctx context.Context) ([]string, error)
	resolveCVEEntitiesFn    func(ctx context.Context, filter types.CVEChartFilter) ([]string, error)
	recordBucketDataFn      func(ctx context.Context, dataset string, bucketStart time.Time, bucketSize time.Duration, strategy api.SampleStrategy, entityBitmaps map[string]*roaring.Bitmap) error
	recordBucketDataInvoked bool
	deleteAllForDatasetFn   func(ctx context.Context, dataset string, batchSize int) error
	hostIDsInFleetsFn       func(ctx context.Context, fleetIDs []uint) ([]uint, error)
	applyScrubMaskFn        func(ctx context.Context, dataset string, mask *roaring.Bitmap, batchSize int) error
}

func (m *mockDatastore) FindOnlineHostIDs(ctx context.Context, now time.Time, disabledFleetIDs []uint) ([]uint, error) {
	if m.findOnlineHostIDsFn != nil {
		return m.findOnlineHostIDsFn(ctx, now, disabledFleetIDs)
	}
	return nil, nil
}

func (m *mockDatastore) AffectedHostIDsByCVE(ctx context.Context, disabledFleetIDs []uint, cves []string) (map[string]*roaring.Bitmap, error) {
	if m.affectedHostIDsByCVEFn != nil {
		return m.affectedHostIDsByCVEFn(ctx, disabledFleetIDs, cves)
	}
	return nil, nil
}

func (m *mockDatastore) CollectibleCVEs(ctx context.Context) ([]string, error) {
	if m.collectibleCVEsFn != nil {
		return m.collectibleCVEsFn(ctx)
	}
	// Match the real contract: non-nil, empty when nothing matches.
	return []string{}, nil
}

func (m *mockDatastore) ResolveCVEChartEntities(ctx context.Context, filter types.CVEChartFilter) ([]string, error) {
	if m.resolveCVEEntitiesFn != nil {
		return m.resolveCVEEntitiesFn(ctx, filter)
	}
	// Match the real contract: non-nil, empty means "match nothing" (never nil,
	// which would be interpreted as "no entity filter").
	return []string{}, nil
}

func (m *mockDatastore) RecordBucketData(ctx context.Context, dataset string, bucketStart time.Time, bucketSize time.Duration, strategy api.SampleStrategy, entityBitmaps map[string]*roaring.Bitmap) error {
	m.recordBucketDataInvoked = true
	if m.recordBucketDataFn != nil {
		return m.recordBucketDataFn(ctx, dataset, bucketStart, bucketSize, strategy, entityBitmaps)
	}
	return nil
}

func (m *mockDatastore) GetSCDData(ctx context.Context, dataset string, startDate, endDate time.Time, bucketSize time.Duration, strategy api.SampleStrategy, filterMask *roaring.Bitmap, entityIDs []string) ([]api.DataPoint, error) {
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

func (m *mockDatastore) ApplyScrubMaskToDataset(ctx context.Context, dataset string, mask *roaring.Bitmap, batchSize int) error {
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

	_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 32})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid days value")

	_, err = svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 0})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid days value")

	_, err = svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: -1})
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
	var gotMask *roaring.Bitmap
	ds.getSCDDataFunc = func(_ context.Context, dataset string, start, end time.Time, bucketSize time.Duration, strategy api.SampleStrategy, mask *roaring.Bitmap, _ []string) ([]api.DataPoint, error) {
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
	assert.Equal(t, uint64(200), chart.BlobPopcount(gotMask), "filter mask should encode all 200 host IDs")
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
			ds.getSCDDataFunc = func(_ context.Context, _ string, _, _ time.Time, bucketSize time.Duration, _ api.SampleStrategy, _ *roaring.Bitmap, _ []string) ([]api.DataPoint, error) {
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
			ds.getSCDDataFunc = func(_ context.Context, _ string, _, _ time.Time, bucketSize time.Duration, strategy api.SampleStrategy, _ *roaring.Bitmap, _ []string) ([]api.DataPoint, error) {
				gotBucketSize = bucketSize
				gotStrategy = strategy
				return nil, nil
			}

			resp, err := svc.GetChartData(premiumCtx(t), "cve", api.RequestOpts{Days: 30, Resolution: tc.resolution})
			require.NoError(t, err)
			assert.Equal(t, tc.resolutionStr, resp.Resolution)
			assert.Equal(t, tc.bucketSize, gotBucketSize)
			assert.Equal(t, api.SampleStrategySnapshot, gotStrategy)
		})
	}
}

func TestGetChartDataUptimePassesNilEntityIDs(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
	svc.RegisterDataset(&chart.UptimeDataset{})

	// The uptime path must not resolve CVE entities — fail loudly if it does.
	ds.resolveCVEEntitiesFn = func(_ context.Context, _ types.CVEChartFilter) ([]string, error) {
		t.Fatal("uptime path must not call ResolveCVEChartEntities")
		return nil, nil
	}
	gotEntityIDsIsNil := false
	ds.getSCDDataFunc = func(_ context.Context, _ string, _, _ time.Time, _ time.Duration, _ api.SampleStrategy, _ *roaring.Bitmap, entityIDs []string) ([]api.DataPoint, error) {
		gotEntityIDsIsNil = entityIDs == nil
		return nil, nil
	}

	_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{Days: 7})
	require.NoError(t, err)
	assert.True(t, gotEntityIDsIsNil, "uptime must pass nil entityIDs — the CVE branch must not leak")
}

func newCVEService(ds *mockDatastore) *Service {
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
	svc.RegisterDataset(&chart.CVEDataset{})
	return svc
}

func captureCVEFilter(ds *mockDatastore) *types.CVEChartFilter {
	got := &types.CVEChartFilter{}
	ds.resolveCVEEntitiesFn = func(_ context.Context, filter types.CVEChartFilter) ([]string, error) {
		*got = filter
		return []string{}, nil
	}
	return got
}

func TestGetChartDataCVESeverityPassthrough(t *testing.T) {
	cases := []struct {
		name     string
		min, max *float64
	}{
		{name: "no bounds"},
		{name: "both bounds", min: new(1.0), max: new(5.0)},
		{name: "lower bound only", min: new(7.0)},
		{name: "upper bound only", max: new(3.9)},
		{name: "full range is not the same as no bounds", min: new(0.0), max: new(10.0)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ds := &mockDatastore{}
			got := captureCVEFilter(ds)

			resp, err := newCVEService(ds).GetChartData(premiumCtx(t), "cve", api.RequestOpts{
				Days:        7,
				SeverityMin: tc.min,
				SeverityMax: tc.max,
			})
			require.NoError(t, err)

			require.Equal(t, tc.min, got.CVSSMin, "severity_min must reach the resolver unchanged")
			require.Equal(t, tc.max, got.CVSSMax, "severity_max must reach the resolver unchanged")

			// What was applied is what gets echoed, on every case.
			require.Equal(t, tc.min, resp.Filters.SeverityMin)
			require.Equal(t, tc.max, resp.Filters.SeverityMax)
		})
	}
}

func TestGetChartDataCVEAlwaysResolvesEntities(t *testing.T) {
	t.Run("no filters still resolves a concrete set", func(t *testing.T) {
		ds := &mockDatastore{}
		resolveCalled := false
		ds.resolveCVEEntitiesFn = func(_ context.Context, _ types.CVEChartFilter) ([]string, error) {
			resolveCalled = true
			return []string{"CVE-2026-0001"}, nil
		}
		var gotEntityIDs []string
		ds.getSCDDataFunc = func(_ context.Context, _ string, _, _ time.Time, _ time.Duration, _ api.SampleStrategy, _ *roaring.Bitmap, entityIDs []string) ([]api.DataPoint, error) {
			gotEntityIDs = entityIDs
			return nil, nil
		}

		_, err := newCVEService(ds).GetChartData(premiumCtx(t), "cve", api.RequestOpts{Days: 7})
		require.NoError(t, err)
		require.True(t, resolveCalled, "the CVE metric must always resolve its entity set")
		require.Equal(t, []string{"CVE-2026-0001"}, gotEntityIDs, "resolved set must be forwarded to GetSCDData, never nil")
	})

	t.Run("entity filters are forwarded to the resolver", func(t *testing.T) {
		ds := &mockDatastore{}
		gotFilter := captureCVEFilter(ds)

		opts := api.RequestOpts{
			Days:            7,
			SoftwareFilters: []string{api.CVECategoryBrowsers, api.CVECategoryAdobe},
			KnownExploit:    true,
			EPSSMin:         new(0.5),
			EPSSMax:         new(1.0),
			ExcludeCVEs:     []string{"CVE-2026-9999"},
		}
		_, err := newCVEService(ds).GetChartData(premiumCtx(t), "cve", opts)
		require.NoError(t, err)
		require.Equal(t, []string{api.CVECategoryBrowsers, api.CVECategoryAdobe}, gotFilter.Categories)
		require.True(t, gotFilter.KnownExploit)
		require.Equal(t, new(0.5), gotFilter.EPSSMin)
		require.Equal(t, []string{"CVE-2026-9999"}, gotFilter.ExcludeCVEs)
	})
}

func TestGetChartDataCVERequiresPremium(t *testing.T) {
	// The uptime dataset is registered alongside CVE so the "other metrics"
	// case exercises a real free-tier chart rather than an unknown metric.
	newSvc := func(ds *mockDatastore) *Service {
		svc := newCVEService(ds)
		svc.RegisterDataset(&chart.UptimeDataset{})
		return svc
	}

	t.Run("free tier is refused before any query runs", func(t *testing.T) {
		ds := &mockDatastore{}
		resolveCalled := false
		ds.resolveCVEEntitiesFn = func(_ context.Context, _ types.CVEChartFilter) ([]string, error) {
			resolveCalled = true
			return []string{}, nil
		}

		_, err := newSvc(ds).GetChartData(freeCtx(t), "cve", api.RequestOpts{Days: 7})
		requirePremiumRequired(t, err)
		require.False(t, resolveCalled, "the gate must short-circuit before entity resolution")
	})

	t.Run("a missing license is treated as free", func(t *testing.T) {
		_, err := newSvc(&mockDatastore{}).GetChartData(t.Context(), "cve", api.RequestOpts{Days: 7})
		requirePremiumRequired(t, err)
	})

	t.Run("the license is checked before request validation", func(t *testing.T) {
		// Ordering matters: an unlicensed caller gets the license error, not a
		// validation error that would tell them how the filter behaves.
		_, err := newSvc(&mockDatastore{}).GetChartData(freeCtx(t), "cve", api.RequestOpts{
			Days:        7,
			SeverityMin: new(8.0),
			SeverityMax: new(2.0), // inverted, would otherwise be a 400
		})
		requirePremiumRequired(t, err)
	})

	t.Run("premium is allowed", func(t *testing.T) {
		ds := &mockDatastore{}
		captureCVEFilter(ds)
		_, err := newSvc(ds).GetChartData(premiumCtx(t), "cve", api.RequestOpts{Days: 7})
		require.NoError(t, err)
	})

	t.Run("the gate does not apply to other metrics", func(t *testing.T) {
		_, err := newSvc(&mockDatastore{}).GetChartData(freeCtx(t), "uptime", api.RequestOpts{Days: 7})
		require.NoError(t, err, "only the cve metric is premium-gated")
	})
}

func TestGetChartDataSeverityValidation(t *testing.T) {
	cases := []struct {
		name    string
		min     *float64
		max     *float64
		wantErr bool
	}{
		{name: "no bounds", wantErr: false},
		{name: "full range", min: new(0.0), max: new(10.0), wantErr: false},
		{name: "equal bounds", min: new(5.0), max: new(5.0), wantErr: false},
		{name: "min only", min: new(7.0), wantErr: false},
		{name: "max only", max: new(3.9), wantErr: false},
		{name: "min above range", min: new(10.1), wantErr: true},
		{name: "max above range", max: new(11.0), wantErr: true},
		{name: "min below range", min: new(-1.0), wantErr: true},
		{name: "max below range", max: new(-0.1), wantErr: true},
		{name: "min greater than max", min: new(8.0), max: new(2.0), wantErr: true},
		{name: "min is NaN", min: new(math.NaN()), wantErr: true},
		{name: "max is NaN", max: new(math.NaN()), wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ds := &mockDatastore{}
			resolveCalled := false
			ds.resolveCVEEntitiesFn = func(_ context.Context, _ types.CVEChartFilter) ([]string, error) {
				resolveCalled = true
				return []string{}, nil
			}

			_, err := newCVEService(ds).GetChartData(premiumCtx(t), "cve", api.RequestOpts{
				Days:        7,
				SeverityMin: tc.min,
				SeverityMax: tc.max,
			})
			if !tc.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			var badReq *platform_http.BadRequestError
			require.ErrorAs(t, err, &badReq)
			require.False(t, resolveCalled, "an invalid bound must be rejected before any query runs")
		})
	}

	t.Run("non-CVE metrics ignore invalid severity bounds", func(t *testing.T) {
		svc := NewService(&mockAuthorizer{}, &mockDatastore{}, globalViewer(), nil)
		svc.RegisterDataset(&chart.UptimeDataset{})

		_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{
			Days:        7,
			SeverityMin: new(99.0),
			SeverityMax: new(-1.0),
		})
		require.NoError(t, err)
	})
}

func TestGetChartDataEPSSValidation(t *testing.T) {
	cases := []struct {
		name    string
		min     *float64
		max     *float64
		wantErr bool
	}{
		{name: "no bounds", wantErr: false},
		{name: "full range", min: new(0.0), max: new(1.0), wantErr: false},
		{name: "equal bounds", min: new(0.5), max: new(0.5), wantErr: false},
		{name: "min only", min: new(0.85), wantErr: false},
		{name: "max only", max: new(0.1), wantErr: false},
		{name: "unconverted percentage", min: new(50.0), wantErr: true},
		{name: "min above range", min: new(1.1), wantErr: true},
		{name: "max above range", max: new(100.0), wantErr: true},
		{name: "min below range", min: new(-0.1), wantErr: true},
		{name: "max below range", max: new(-1.0), wantErr: true},
		{name: "min greater than max", min: new(0.9), max: new(0.2), wantErr: true},
		{name: "min is NaN", min: new(math.NaN()), wantErr: true},
		{name: "max is NaN", max: new(math.NaN()), wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ds := &mockDatastore{}
			resolveCalled := false
			ds.resolveCVEEntitiesFn = func(_ context.Context, _ types.CVEChartFilter) ([]string, error) {
				resolveCalled = true
				return []string{}, nil
			}
			svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
			svc.RegisterDataset(&chart.CVEDataset{})

			_, err := svc.GetChartData(premiumCtx(t), "cve", api.RequestOpts{
				Days:    7,
				EPSSMin: tc.min,
				EPSSMax: tc.max,
			})
			if !tc.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			var badReq *platform_http.BadRequestError
			require.ErrorAs(t, err, &badReq)
			require.False(t, resolveCalled, "an invalid bound must be rejected before any query runs")
		})
	}

	t.Run("non-CVE metrics ignore invalid EPSS bounds", func(t *testing.T) {
		svc := NewService(&mockAuthorizer{}, &mockDatastore{}, globalViewer(), nil)
		svc.RegisterDataset(&chart.UptimeDataset{})

		_, err := svc.GetChartData(t.Context(), "uptime", api.RequestOpts{
			Days:    7,
			EPSSMin: new(50.0),
			EPSSMax: new(-1.0),
		})
		require.NoError(t, err)
	})
}

func TestGetChartDataOmitsUnsetSeverityFromEcho(t *testing.T) {
	ds := &mockDatastore{}
	ds.resolveCVEEntitiesFn = func(_ context.Context, _ types.CVEChartFilter) ([]string, error) {
		return []string{}, nil
	}
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
	svc.RegisterDataset(&chart.CVEDataset{})

	resp, err := svc.GetChartData(premiumCtx(t), "cve", api.RequestOpts{Days: 7})
	require.NoError(t, err)
	require.Nil(t, resp.Filters.SeverityMin)
	require.Nil(t, resp.Filters.SeverityMax)

	// A one-sided request echoes only the side that was set.
	resp, err = svc.GetChartData(premiumCtx(t), "cve", api.RequestOpts{Days: 7, SeverityMax: new(3.9)})
	require.NoError(t, err)
	require.Nil(t, resp.Filters.SeverityMin)
	require.NotNil(t, resp.Filters.SeverityMax)
	require.InDelta(t, 3.9, *resp.Filters.SeverityMax, 0)
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
	ds.getSCDDataFunc = func(_ context.Context, _ string, _, _ time.Time, _ time.Duration, _ api.SampleStrategy, mask *roaring.Bitmap, _ []string) ([]api.DataPoint, error) {
		assert.Equal(t, uint64(2), chart.BlobPopcount(mask), "mask should encode the 2 host IDs returned")
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

	ds.findOnlineHostIDsFn = func(_ context.Context, gotNow time.Time, _ []uint) ([]uint, error) {
		assert.Equal(t, now, gotNow)
		return []uint{1, 2, 3}, nil
	}
	ds.recordBucketDataFn = func(_ context.Context, dataset string, bucketStart time.Time, bucketSize time.Duration, strategy api.SampleStrategy, entityBitmaps map[string]*roaring.Bitmap) error {
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

	wantTracked := []string{"CVE-2024-0001", "CVE-2024-0002"}
	ds.collectibleCVEsFn = func(_ context.Context) ([]string, error) {
		return wantTracked, nil
	}
	var gotCVEs []string
	ds.affectedHostIDsByCVEFn = func(_ context.Context, _ []uint, cves []string) (map[string]*roaring.Bitmap, error) {
		gotCVEs = cves
		return map[string]*roaring.Bitmap{
			"CVE-2024-0001": roaring.BitmapOf(1, 2, 3),
			"CVE-2024-0002": roaring.BitmapOf(2, 4),
		}, nil
	}
	ds.recordBucketDataFn = func(_ context.Context, dataset string, bucketStart time.Time, bucketSize time.Duration, strategy api.SampleStrategy, entityBitmaps map[string]*roaring.Bitmap) error {
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
	assert.Equal(t, wantTracked, gotCVEs, "CollectibleCVEs result must be forwarded as the cves filter")
}

// TestCollectDatasetsCVEEmptyTracked verifies that when CollectibleCVEs
// returns an empty set, the collector still calls RecordBucketData with empty
// bitmaps so recordSnapshot's "absent entities" branch can close any open
// rows from prior cron ticks. Without this, dropping a CVE from the tracked
// set would leave its open row hanging forever.
func TestCollectDatasetsCVEEmptyTracked(t *testing.T) {
	ds := &mockDatastore{}
	svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
	svc.RegisterDataset(&chart.CVEDataset{})

	ds.collectibleCVEsFn = func(_ context.Context) ([]string, error) {
		return []string{}, nil
	}
	ds.affectedHostIDsByCVEFn = func(_ context.Context, _ []uint, cves []string) (map[string]*roaring.Bitmap, error) {
		assert.Empty(t, cves, "empty tracked set must propagate as empty cves filter")
		return map[string]*roaring.Bitmap{}, nil
	}
	var gotBitmaps map[string]*roaring.Bitmap
	ds.recordBucketDataFn = func(_ context.Context, _ string, _ time.Time, _ time.Duration, _ api.SampleStrategy, entityBitmaps map[string]*roaring.Bitmap) error {
		gotBitmaps = entityBitmaps
		return nil
	}

	err := svc.CollectDatasets(t.Context(), time.Now(), nil)
	require.NoError(t, err)
	assert.True(t, ds.recordBucketDataInvoked, "RecordBucketData must run on empty tracked set to close stale rows")
	assert.Empty(t, gotBitmaps)
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
		ds.findOnlineHostIDsFn = func(_ context.Context, _ time.Time, _ []uint) ([]uint, error) {
			t.Fatal("FindOnlineHostIDs should not have been called when scope returned skip=true")
			return nil, nil
		}
		err := svc.CollectDatasets(t.Context(), now, func(name string) (bool, []uint) {
			assert.Equal(t, "uptime", name)
			return true, nil
		})
		require.NoError(t, err)
		assert.False(t, ds.recordBucketDataInvoked)
	})

	t.Run("disabledFleetIDs forwarded to FindOnlineHostIDs", func(t *testing.T) {
		ds := &mockDatastore{}
		svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
		svc.RegisterDataset(&chart.UptimeDataset{})

		var gotDisabled []uint
		ds.findOnlineHostIDsFn = func(_ context.Context, _ time.Time, disabled []uint) ([]uint, error) {
			gotDisabled = disabled
			return []uint{1}, nil
		}
		ds.recordBucketDataFn = func(_ context.Context, _ string, _ time.Time, _ time.Duration, _ api.SampleStrategy, _ map[string]*roaring.Bitmap) error {
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

		ds.collectibleCVEsFn = func(_ context.Context) ([]string, error) {
			return []string{"CVE-1"}, nil
		}
		var gotDisabled []uint
		ds.affectedHostIDsByCVEFn = func(_ context.Context, disabled []uint, _ []string) (map[string]*roaring.Bitmap, error) {
			gotDisabled = disabled
			return map[string]*roaring.Bitmap{"CVE-1": roaring.BitmapOf(1)}, nil
		}
		ds.recordBucketDataFn = func(_ context.Context, _ string, _ time.Time, _ time.Duration, _ api.SampleStrategy, _ map[string]*roaring.Bitmap) error {
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

		gotDisabled := []uint{0xDEADBEEF} // sentinel — should be replaced with nil
		ds.findOnlineHostIDsFn = func(_ context.Context, _ time.Time, disabled []uint) ([]uint, error) {
			gotDisabled = disabled
			return []uint{1}, nil
		}
		ds.recordBucketDataFn = func(_ context.Context, _ string, _ time.Time, _ time.Duration, _ api.SampleStrategy, _ map[string]*roaring.Bitmap) error {
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
		var gotMask *roaring.Bitmap
		var gotBatchSize int
		ds.applyScrubMaskFn = func(_ context.Context, dataset string, mask *roaring.Bitmap, batchSize int) error {
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
		assert.Equal(t, uint64(3), chart.BlobPopcount(gotMask))
	})

	t.Run("empty fleet IDs is no-op", func(t *testing.T) {
		ds := &mockDatastore{}
		svc := NewService(&mockAuthorizer{}, ds, globalViewer(), nil)
		ds.hostIDsInFleetsFn = func(_ context.Context, _ []uint) ([]uint, error) {
			t.Fatal("HostIDsInFleets should not have been called for empty input")
			return nil, nil
		}
		ds.applyScrubMaskFn = func(_ context.Context, _ string, _ *roaring.Bitmap, _ int) error {
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
		ds.applyScrubMaskFn = func(_ context.Context, _ string, _ *roaring.Bitmap, _ int) error {
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
