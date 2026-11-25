package mysql

import (
	"testing"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/chart/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestRetireSCDData(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	defer func(n, step int) { scdRetireBatch, scdConvertStepHours = n, step }(scdRetireBatch, scdConvertStepHours)
	scdRetireBatch = 2
	scdConvertStepHours = 1

	converted := func() bool {
		t.Helper()
		ok, err := ds.SCDHistoryConverted(t.Context(), "cve")
		require.NoError(t, err)
		return ok
	}
	// retireAll runs RetireSCDData until none remains and returns how many calls ran.
	retireAll := func() int {
		t.Helper()
		for n := 1; ; n++ {
			more, err := ds.RetireSCDData(t.Context(), "cve")
			require.NoError(t, err)
			if !more {
				return n
			}
			require.Less(t, n, 1000, "retiring does not finish")
		}
	}
	cveRows := func() int { return countDatasetRows(t, tdb, "cve")["host_scd_data"] }

	h := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	oldTick(t, ds, h.Add(-3*time.Hour), hostSets{"CVE-A": {1}, "CVE-B": {2}, "CVE-C": {3}})
	oldTick(t, ds, h.Add(-2*time.Hour), hostSets{"CVE-A": {1, 2}, "CVE-B": {2}, "CVE-C": {3, 4}})
	require.NoError(t, ds.RecordBucketData(t.Context(), "uptime", h, time.Hour, api.SampleStrategyAccumulate,
		map[string]*roaring.Bitmap{"": chart.NewBitmap([]uint{1, 2})}))

	require.False(t, converted(), "the change storage has not started")
	require.Equal(t, 1, retireAll())
	require.Equal(t, 5, cveRows(), "nothing is deleted before the change storage starts")

	collectTick(t, ds, "cve", h, hostSets{"CVE-A": {1, 2}, "CVE-B": {2, 5}})
	require.False(t, converted(), "the history has not been converted")
	require.Equal(t, 1, retireAll())
	require.Equal(t, 6, cveRows(), "nothing is deleted before the conversion starts")

	more, err := ds.ConvertSCDHistory(t.Context(), "cve")
	require.NoError(t, err)
	require.True(t, more)
	require.False(t, converted(), "the conversion is part way")
	require.Equal(t, 1, retireAll())
	require.Equal(t, 6, cveRows(), "nothing is deleted while the conversion is part way")

	convertAll(t, ds, "cve")
	require.True(t, converted())

	q := replayQuery{start: h.Add(-6 * time.Hour), end: h.Add(3 * time.Hour), size: time.Hour}
	replay := func() []api.DataPoint {
		t.Helper()
		got, err := ds.ReplaySCDChanges(t.Context(), "cve", q.start, q.end, q.size, nil, nil)
		require.NoError(t, err)
		return got
	}
	wantReplay := replay()
	wantChanges, wantCurrent := loadChanges(t, tdb, "cve"), loadCurrent(t, tdb, "cve")
	wantChangesFrom, wantCoveredFrom, _ := loadChangeState(t, tdb, "cve")

	require.Equal(t, 4, retireAll(), "6 rows in batches of 2, then an empty batch")
	require.Zero(t, cveRows())
	require.Equal(t, 1, countDatasetRows(t, tdb, "uptime")["host_scd_data"], "other datasets keep their rows")
	require.Equal(t, wantChanges, loadChanges(t, tdb, "cve"))
	require.Equal(t, wantCurrent, loadCurrent(t, tdb, "cve"))
	changesFrom, coveredFrom, found := loadChangeState(t, tdb, "cve")
	require.True(t, found)
	require.Equal(t, wantChangesFrom, changesFrom)
	require.Equal(t, wantCoveredFrom, coveredFrom)
	require.True(t, converted())
	require.Equal(t, wantReplay, replay())

	// A global scrub deletes the change storage. Collection after the dataset is enabled again writes the full
	// copies until its history is converted again, and they are then deleted.
	require.NoError(t, ds.DeleteAllForDataset(t.Context(), "cve", 0))
	require.False(t, converted())
	collectTick(t, ds, "cve", h.Add(5*time.Hour), hostSets{"CVE-A": {7}})
	require.False(t, converted())
	require.Equal(t, 1, retireAll())
	require.Equal(t, 1, cveRows())
	convertAll(t, ds, "cve")
	require.True(t, converted())
	require.Equal(t, 1, retireAll())
	require.Zero(t, cveRows())
	require.Equal(t, hostSets{"CVE-A": {7}}, loadCurrent(t, tdb, "cve"))
}
