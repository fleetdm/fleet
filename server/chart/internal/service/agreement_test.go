package service_test

import (
	"context"
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/chart/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/chart/internal/service"
	"github.com/fleetdm/fleet/v4/server/chart/internal/testutils"
	"github.com/fleetdm/fleet/v4/server/chart/internal/types"
	"github.com/stretchr/testify/require"
)

// agreementStore answers the host and CVE filters with fixed sets, so the chart reads and the collection run on
// the MySQL store without hosts or vulnerability data, and counts which read answered.
type agreementStore struct {
	types.Datastore
	hosts      []uint
	entities   []string
	collected  map[string]*roaring.Bitmap // what the CVE collection finds
	fullCopies bool                       // report the change storage as not started
	replays    int
	reads      int
	// beforeRetire, when set, runs once before the first RetireSCDData call.
	beforeRetire func()
}

func (s *agreementStore) CollectibleCVEs(context.Context) ([]string, error) {
	return slices.Collect(maps.Keys(s.collected)), nil
}

func (s *agreementStore) AffectedHostIDsByCVE(context.Context, []uint, []string) (map[string]*roaring.Bitmap, error) {
	return s.collected, nil
}

func (s *agreementStore) RetireSCDData(ctx context.Context, dataset string) (bool, error) {
	if s.beforeRetire != nil {
		s.beforeRetire()
		s.beforeRetire = nil
	}
	return s.Datastore.RetireSCDData(ctx, dataset)
}

func (s *agreementStore) GetHostIDsForFilter(context.Context, *types.HostFilter) ([]uint, error) {
	return s.hosts, nil
}

func (s *agreementStore) ResolveCVEChartEntities(context.Context, types.CVEChartFilter) ([]string, error) {
	return s.entities, nil
}

func (s *agreementStore) SCDChangesCoverage(ctx context.Context, dataset string) (types.SCDChangeCoverage, error) {
	if s.fullCopies {
		return types.SCDChangeCoverage{}, nil
	}
	return s.Datastore.SCDChangesCoverage(ctx, dataset)
}

func (s *agreementStore) ReplaySCDChanges(ctx context.Context, dataset string, start, end time.Time, size time.Duration, mask *roaring.Bitmap, entityIDs []string) ([]api.DataPoint, error) {
	s.replays++
	return s.Datastore.ReplaySCDChanges(ctx, dataset, start, end, size, mask, entityIDs)
}

func (s *agreementStore) GetSCDData(ctx context.Context, dataset string, start, end time.Time, size time.Duration, strategy api.SampleStrategy, mask *roaring.Bitmap, entityIDs []string) ([]api.DataPoint, error) {
	s.reads++
	return s.Datastore.GetSCDData(ctx, dataset, start, end, size, strategy, mask, entityIDs)
}

type agreementRequest struct {
	hosts    []uint
	entities []string
	opts     api.RequestOpts
}

func (r agreementRequest) String() string {
	return fmt.Sprintf("hosts=%v entities=%v days=%d resolution=%d tz=%d", r.hosts, r.entities, r.opts.Days, r.opts.Resolution, r.opts.TZOffsetMinutes)
}

// TestGetChartDataCVEAgreement checks that a vulnerability exposure chart response is the same whichever read
// answers it: the full copies before the history is converted, replay after.
func TestGetChartDataCVEAgreement(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_service")
	ds := mysql.NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := service.PremiumCtx(t)

	// Cleanup computes its cutoff from the wall clock, so the history is written around it.
	now := time.Now().UTC()
	clock := now
	h := now.Truncate(time.Hour)
	changesFrom := h.Add(-5 * 24 * time.Hour)

	entities := []string{"CVE-0", "CVE-1", "CVE-2", "CVE-3", "CVE-4", "CVE-5"}
	rng := rand.New(rand.NewPCG(58928, 5)) //nolint:gosec // deterministic test data
	tick := func(hour time.Time) {
		sets := map[string]*roaring.Bitmap{}
		for _, e := range entities {
			if rng.IntN(5) == 0 {
				continue
			}
			ids := []uint{uint(1 + rng.IntN(12))}
			for host := uint(1); host <= 12; host++ {
				if rng.IntN(3) == 0 {
					ids = append(ids, host)
				}
			}
			if rng.IntN(4) == 0 {
				ids = append(ids, 70000+uint(rng.IntN(3)))
			}
			sets[e] = chart.NewBitmap(ids)
		}
		require.NoError(t, ds.RecordBucketData(t.Context(), api.MetricCVE, hour, time.Hour, api.SampleStrategySnapshot, sets))
		if !hour.Before(changesFrom) {
			require.NoError(t, ds.RecordSCDChanges(t.Context(), api.MetricCVE, hour, sets))
		}
	}
	// 40 days of history, the last 5 of them also collected into the change storage.
	for hour := h.Add(-40 * 24 * time.Hour); !hour.After(h); hour = hour.Add(5 * time.Hour) {
		tick(hour)
	}

	var requests []agreementRequest
	hostSets := [][]uint{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 70000, 70001, 70002}, {1, 3, 5, 7, 70001}, {}}
	entitySets := [][]string{entities, {"CVE-1", "CVE-4", "CVE-missing"}, {}}
	optsList := []api.RequestOpts{
		{Days: 31},
		{Days: 31, Resolution: 1, TZOffsetMinutes: -330},
		{Days: 31, Resolution: 24, TZOffsetMinutes: 300},
		{Days: 7, Resolution: 24, TZOffsetMinutes: -330},
		{Days: 1, Resolution: 1},
	}
	for _, hosts := range hostSets {
		for _, ents := range entitySets {
			for _, opts := range optsList {
				requests = append(requests, agreementRequest{hosts: hosts, entities: ents, opts: opts})
			}
		}
	}

	// get returns the response and how many times each read answered it.
	get := func(r agreementRequest, fullCopies bool) (resp *api.Response, replays, reads int) {
		store := &agreementStore{Datastore: ds, hosts: r.hosts, entities: r.entities, fullCopies: fullCopies}
		svc := service.NewService(&service.AllowAllAuthorizer{}, store, service.GlobalViewer(), nil, tdb.Logger)
		svc.RegisterDataset(&chart.CVEDataset{})
		service.SetNow(svc, func() time.Time { return clock })
		resp, err := svc.GetChartData(ctx, api.MetricCVE, r.opts)
		require.NoError(t, err, "%s", r)
		return resp, store.replays, store.reads
	}

	// Before conversion the changes cover only the last 5 days.
	before := make([]*api.Response, len(requests))
	for i, r := range requests {
		resp, replays, reads := get(r, false)
		if r.opts.Days == 31 {
			require.Equal(t, 1, reads, "%s", r)
			require.Zero(t, replays, "%s", r)
		}
		if r.opts.Days == 1 {
			require.Equal(t, 1, replays, "%s", r)
		}
		before[i] = resp
	}

	for more := true; more; {
		var err error
		more, err = ds.ConvertSCDHistory(t.Context(), api.MetricCVE)
		require.NoError(t, err)
	}

	for i, r := range requests {
		resp, replays, reads := get(r, false)
		require.Equal(t, 1, replays, "%s", r)
		require.Zero(t, reads, "%s", r)
		require.Equal(t, before[i], resp, "%s", r)
	}

	// requireAgreement compares replay with the full copies for the buckets that end at or after from.
	requireAgreement := func(from time.Time) {
		t.Helper()
		masked := func(resp *api.Response, size time.Duration) *api.Response {
			for i, p := range resp.Data {
				if p.Timestamp.Add(size).Before(from) {
					resp.Data[i].Value = -1
				}
			}
			return resp
		}
		for _, r := range requests {
			replayed, replays, _ := get(r, false)
			require.Equal(t, 1, replays, "%s", r)
			full, _, reads := get(r, true)
			require.Equal(t, 1, reads, "%s", r)
			size := 3 * time.Hour
			if r.opts.Resolution > 0 {
				size = time.Duration(r.opts.Resolution) * time.Hour
			}
			require.Equal(t, masked(full, size), masked(replayed, size), "%s", r)
		}
	}

	// Further collections, with the clock moving with them.
	for i := 1; i <= 7; i++ {
		clock = now.Add(time.Duration(i) * time.Hour)
		tick(h.Add(time.Duration(i) * time.Hour))
	}
	requireAgreement(time.Time{})

	// Cleanup removes part of the history of the oldest buckets of a 31-day request; the hour of margin covers
	// the wall clock moving past an hour boundary before cleanup reads it.
	cutoff := time.Now().UTC().AddDate(0, 0, -30).Add(time.Hour)
	require.NoError(t, ds.CleanupSCDData(t.Context(), 30))
	requireAgreement(cutoff)

	// Once the history is converted, responses do not depend on the full copies: deleting them changes none.
	beforeRetire := make([]*api.Response, len(requests))
	for i, r := range requests {
		beforeRetire[i], _, _ = get(r, false)
	}
	for more := true; more; {
		var err error
		more, err = ds.RetireSCDData(t.Context(), api.MetricCVE)
		require.NoError(t, err)
	}
	var cveRows int
	require.NoError(t, tdb.DB.GetContext(t.Context(), &cveRows, `SELECT COUNT(*) FROM host_scd_data WHERE dataset = ?`, api.MetricCVE))
	require.Zero(t, cveRows)
	for i, r := range requests {
		resp, replays, reads := get(r, false)
		require.Equal(t, 1, replays, "%s", r)
		require.Zero(t, reads, "%s", r)
		require.Equal(t, beforeRetire[i], resp, "%s", r)
	}
}

// TestCollectCVERetiresFullCopies runs the vulnerability exposure collection on the MySQL store across the end of
// the conversion: it writes the full copies until the history is converted, and then deletes them.
func TestCollectCVERetiresFullCopies(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_service")
	ds := mysql.NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := service.PremiumCtx(t)

	countRows := func(dataset string, validFrom *time.Time) int {
		t.Helper()
		stmt, args := `SELECT COUNT(*) FROM host_scd_data WHERE dataset = ?`, []any{dataset}
		if validFrom != nil {
			stmt, args = stmt+` AND valid_from = ?`, append(args, *validFrom)
		}
		var n int
		require.NoError(t, tdb.DB.GetContext(t.Context(), &n, stmt, args...))
		return n
	}
	sets := func(m map[string][]uint) map[string]*roaring.Bitmap {
		out := make(map[string]*roaring.Bitmap, len(m))
		for e, ids := range m {
			out[e] = chart.NewBitmap(ids)
		}
		return out
	}

	h := time.Now().UTC().Truncate(time.Hour)
	for i, hour := range []time.Time{h.Add(-72 * time.Hour), h.Add(-30 * time.Hour), h.Add(-5 * time.Hour)} {
		require.NoError(t, ds.RecordBucketData(t.Context(), api.MetricCVE, hour, time.Hour, api.SampleStrategySnapshot,
			sets(map[string][]uint{"CVE-A": {1, uint(2 + i)}, "CVE-B": {uint(5 + i)}})))
	}
	require.NoError(t, ds.RecordBucketData(t.Context(), "uptime", h, time.Hour, api.SampleStrategyAccumulate,
		sets(map[string][]uint{"": {1, 2, 3}})))

	store := &agreementStore{Datastore: ds, hosts: []uint{1, 2, 3, 4, 5, 6, 7, 8}, entities: []string{"CVE-A", "CVE-B", "CVE-C"}}
	svc := service.NewService(&service.AllowAllAuthorizer{}, store, service.GlobalViewer(), nil, tdb.Logger)
	svc.RegisterDataset(&chart.CVEDataset{})
	clock := h.Add(30 * time.Minute)
	service.SetNow(svc, func() time.Time { return clock })
	getChart := func() *api.Response {
		t.Helper()
		resp, err := svc.GetChartData(ctx, api.MetricCVE, api.RequestOpts{Days: 7, Resolution: 1})
		require.NoError(t, err)
		return resp
	}

	// The first collection writes both formats and converts the history.
	store.collected = sets(map[string][]uint{"CVE-A": {1, 4, 6}, "CVE-C": {8}})
	require.NoError(t, svc.CollectDatasets(t.Context(), h.Add(10*time.Minute), nil))
	require.Equal(t, 2, countRows(api.MetricCVE, &h), "the full copies are written until the history is converted")
	converted, err := ds.SCDHistoryConverted(t.Context(), api.MetricCVE)
	require.NoError(t, err)
	require.True(t, converted)

	// The next collection writes only the change storage, then deletes the full copies; the response does not
	// depend on them.
	next := h.Add(time.Hour)
	clock = next.Add(30 * time.Minute)
	store.collected = sets(map[string][]uint{"CVE-A": {1, 6}, "CVE-B": {2}})
	var beforeDelete *api.Response
	store.beforeRetire = func() {
		require.Zero(t, countRows(api.MetricCVE, &next), "no full copy is written once the history is converted")
		require.Positive(t, countRows(api.MetricCVE, nil))
		beforeDelete = getChart()
	}
	require.NoError(t, svc.CollectDatasets(t.Context(), next.Add(10*time.Minute), nil))
	require.NotNil(t, beforeDelete, "the full copies are deleted after the history is converted")
	require.Zero(t, countRows(api.MetricCVE, nil))
	require.Equal(t, 1, countRows("uptime", nil), "other datasets keep their rows")

	store.replays, store.reads = 0, 0
	resp := getChart()
	require.Equal(t, beforeDelete, resp)
	require.Equal(t, 1, store.replays)
	require.Zero(t, store.reads)
	last := resp.Data[len(resp.Data)-1]
	require.Equal(t, next, last.Timestamp)
	require.Equal(t, 3, last.Value, "hosts 1, 2 and 6 after the last collection")
}
