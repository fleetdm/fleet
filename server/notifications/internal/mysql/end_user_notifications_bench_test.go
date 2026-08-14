package mysql

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/fleetdm/fleet/v4/server/notifications/internal/testutils"
	"github.com/google/uuid"
)

// Phase timings for the notifications-owned part of one cron pass: listing
// what's due and recording what got dispatched. The script queueing and
// activation phases are benchmarked in server/datastore/mysql, since that
// table isn't owned here.
//
// To run:
//
//	MYSQL_TEST=1 go test -bench=BenchmarkEndUserNotificationDispatch -benchtime=10x \
//	    -run='^$' ./server/notifications/internal/mysql/
//
// Tune the dataset with FLEET_BENCH_SIZE: "smoke" (default), "realistic", or
// "large". Absolute numbers from a local MySQL container are not prod numbers;
// compare phases against each other and compare runs against each other.

const dispatchBenchHostBatchSize = 500

type notificationBenchSize struct {
	numHosts int
	// notifications per host, so the dedup in ListEndUserNotificationsToDispatch
	// has more than one row per host to collapse
	notificationsPerHost int
}

var notificationBenchSizes = map[string]notificationBenchSize{
	"smoke":     {numHosts: 2_000, notificationsPerHost: 1},
	"realistic": {numHosts: 50_000, notificationsPerHost: 2},
	"large":     {numHosts: 500_000, notificationsPerHost: 2},
}

func pickNotificationBenchSize(tb testing.TB) notificationBenchSize {
	tb.Helper()
	name := os.Getenv("FLEET_BENCH_SIZE")
	if name == "" {
		name = "smoke"
	}
	sz, ok := notificationBenchSizes[name]
	if !ok {
		tb.Fatalf("unknown FLEET_BENCH_SIZE=%q (want smoke|realistic|large)", name)
	}
	return sz
}

// seedNotificationDispatchData creates darwin hosts running Fleet Desktop, each
// with pending notifications due now, which is what the dispatch query looks for.
func seedNotificationDispatchData(tb testing.TB, env *testutils.TestDB, sz notificationBenchSize) []uint {
	tb.Helper()

	hostIDs := make([]uint, sz.numHosts)
	for i := range hostIDs {
		hostID := env.InsertHost(tb, fmt.Sprintf("host-%d", i+1), "darwin")
		env.InsertHostOrbitInfo(tb, hostID)
		hostIDs[i] = hostID
	}

	for i := 0; i < sz.numHosts*sz.notificationsPerHost; i++ {
		hostID := hostIDs[i%sz.numHosts]
		env.InsertNotification(tb, hostID, "patch", nil, nil)
	}
	return hostIDs
}

func resetNotificationsToPending(tb testing.TB, ds *Datastore, env *testutils.TestDB) {
	tb.Helper()
	if _, err := env.DB.ExecContext(context.Background(),
		`UPDATE end_user_notifications SET status = ?, execution_id = NULL, attempt_count = 0`,
		api.EndUserNotificationPending,
	); err != nil {
		tb.Fatalf("reset notifications: %v", err)
	}
}

func BenchmarkEndUserNotificationDispatch(b *testing.B) {
	tdb := testutils.SetupTestDB(b, "notifications_bench")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	sz := pickNotificationBenchSize(b)
	seedNotificationDispatchData(b, tdb, sz)
	ctx := context.Background()

	batch, err := ds.ListEndUserNotificationsToDispatch(ctx, dispatchBenchHostBatchSize)
	if err != nil {
		b.Fatal(err)
	}
	if len(batch) == 0 {
		b.Fatal("seed produced no dispatchable notifications")
	}

	b.Run("list", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := ds.ListEndUserNotificationsToDispatch(ctx, dispatchBenchHostBatchSize); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("set_dispatched", func(b *testing.B) {
		b.ReportAllocs()
		for _, notification := range batch {
			executionID := uuid.New().String()
			notification.ExecutionID = &executionID
		}
		for b.Loop() {
			b.StopTimer()
			resetNotificationsToPending(b, ds, tdb)
			b.StartTimer()

			if err := ds.SetEndUserNotificationsDispatched(ctx, batch); err != nil {
				b.Fatal(err)
			}
		}
		resetNotificationsToPending(b, ds, tdb)
	})
}
