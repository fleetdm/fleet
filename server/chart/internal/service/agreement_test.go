package service_test

import (
	"context"
	"fmt"
	"math/rand/v2"
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

// agreementStore answers the host and CVE filters with fixed sets, so the chart reads run on the MySQL store
// without hosts or vulnerability data, and counts which read answered.
type agreementStore struct {
	types.Datastore
	hosts      []uint
	entities   []string
	fullCopies bool // report the change storage as not started
	replays    int
	reads      int
}

func (s *agreementStore) GetHostIDsForFilter(context.Context, *types.HostFilter) ([]uint, error) {
	return s.hosts, nil
}

func (s *agreementStore) ResolveCVEChartEntities(context.Context, types.CVEChartFilter) ([]string, error) {
	return s.entities, nil
}

func (s *agreementStore) SCDChangesCoveredFrom(ctx context.Context, dataset string) (time.Time, bool, error) {
	if s.fullCopies {
		return time.Time{}, false, nil
	}
	return s.Datastore.SCDChangesCoveredFrom(ctx, dataset)
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
}
