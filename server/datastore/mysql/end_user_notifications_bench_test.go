package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Phase timings for one cron pass of the end user notification dispatcher.
// Each sub-benchmark isolates one phase of dispatchEndUserNotifications.
//
// To run:
//
//	MYSQL_TEST=1 go test -bench=BenchmarkEndUserNotificationDispatch -benchtime=10x \
//	    -run='^$' ./server/datastore/mysql/
//
// To profile:
//
//	MYSQL_TEST=1 go test -bench=BenchmarkEndUserNotificationDispatch -benchtime=20x \
//	    -run='^$' -cpuprofile=/tmp/cpu.out -memprofile=/tmp/mem.out \
//	    ./server/datastore/mysql/
//	go tool pprof -http=: /tmp/cpu.out
//
// A CPU profile only says something useful about build_in_args, which is the one
// phase that is pure Go. The other phases are almost entirely waiting on MySQL,
// so their ns/op is the number that matters and a CPU profile of them shows
// runtime.netpoll rather than anything actionable. Use -blockprofile to see the
// waiting itself.
//
// Tune the dataset with FLEET_BENCH_SIZE: "smoke" (default), "realistic", or
// "large". Absolute numbers from a local MySQL container are not prod numbers;
// compare phases against each other and compare runs against each other.

// dispatchBenchHostBatchSize matches the batch size the cron uses, so a
// benchmark iteration is one real pass.
const dispatchBenchHostBatchSize = 500

type notificationBenchSize struct {
	numHosts int
	// notifications per host, so the GROUP BY in ListEndUserNotificationsToDispatch
	// has more than one row per group to collapse
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
func seedNotificationDispatchData(tb testing.TB, ds *Datastore, sz notificationBenchSize) {
	tb.Helper()
	ctx := context.Background()
	w := ds.writer(ctx)

	if _, err := w.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS=0"); err != nil {
		tb.Fatalf("disable FK: %v", err)
	}
	defer func() {
		if _, err := w.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS=1"); err != nil {
			tb.Logf("re-enable FK: %v", err)
		}
	}()

	batchInsert(tb, ds,
		"INSERT INTO hosts (id, osquery_host_id, node_key, hostname, uuid, platform) VALUES ",
		sz.numHosts, 6, func(i int) []any {
			return []any{
				i + 1,
				fmt.Sprintf("osquery-%d", i+1),
				fmt.Sprintf("nodekey-%d", i+1),
				fmt.Sprintf("host-%d", i+1),
				fmt.Sprintf("uuid-%d", i+1),
				"darwin",
			}
		})

	// desktop_version has to be set for a host to be considered able to display
	// a notification
	batchInsert(tb, ds,
		"INSERT INTO host_orbit_info (host_id, version, desktop_version, scripts_enabled) VALUES ",
		sz.numHosts, 4, func(i int) []any {
			return []any{i + 1, "1.40.0", "1.5.0", true}
		})

	payload, err := json.Marshal(map[string]any{"apps": []string{"1Password", "Slack"}})
	if err != nil {
		tb.Fatal(err)
	}

	total := sz.numHosts * sz.notificationsPerHost
	batchInsert(tb, ds,
		"INSERT INTO end_user_notifications (uuid, host_id, status, kind, payload, next_attempt_at) VALUES ",
		total, 6, func(i int) []any {
			return []any{
				fmt.Sprintf("notification-%d", i+1),
				(i % sz.numHosts) + 1,
				fleet.EndUserNotificationPending,
				"notify_before_patching",
				payload,
				nil,
			}
		})
}

// resetNotificationsToPending puts the notifications back the way the dispatch
// query wants to find them, so an iteration measures the same work as the last.
func resetNotificationsToPending(tb testing.TB, ds *Datastore) {
	tb.Helper()
	ctx := context.Background()
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE end_user_notifications SET status = ?, execution_id = NULL, attempt_count = 0`,
		fleet.EndUserNotificationPending,
	); err != nil {
		tb.Fatalf("reset notifications: %v", err)
	}
}

func clearQueuedActivities(tb testing.TB, ds *Datastore) {
	tb.Helper()
	ctx := context.Background()
	for _, stmt := range []string{
		"DELETE FROM script_upcoming_activities",
		"DELETE FROM upcoming_activities",
		"DELETE FROM host_script_results",
	} {
		if _, err := ds.writer(ctx).ExecContext(ctx, stmt); err != nil {
			tb.Fatalf("clear queued activities (%s): %v", stmt, err)
		}
	}
}

// seedUnactivatedScripts queues scripts the way the dispatcher does but stops
// short of activating them, which is the state the activate phase starts from.
func seedUnactivatedScripts(tb testing.TB, ds *Datastore, hostIDs []uint) {
	tb.Helper()
	ctx := context.Background()

	contentsRes, err := insertScriptContents(ctx, ds.writer(ctx), fleet.EndUserNotificationScript)
	if err != nil {
		tb.Fatalf("insert script contents: %v", err)
	}
	scriptContentID, _ := contentsRes.LastInsertId()

	executionIDs := make([]string, 0, len(hostIDs))
	for range hostIDs {
		executionIDs = append(executionIDs, uuid.New().String())
	}

	batchInsert(tb, ds,
		`INSERT INTO upcoming_activities (host_id, priority, activity_type, execution_id, payload) VALUES `,
		len(hostIDs), 5, func(i int) []any {
			return []any{hostIDs[i], 0, "script", executionIDs[i], `{"sync_request": 0, "is_internal": 1}`}
		})

	stmt, args, err := sqlx.In(`
INSERT INTO script_upcoming_activities (upcoming_activity_id, script_content_id)
SELECT ua.id, ? FROM upcoming_activities ua WHERE ua.execution_id IN (?)`, scriptContentID, executionIDs)
	if err != nil {
		tb.Fatalf("build child insert: %v", err)
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		tb.Fatalf("insert script upcoming activities: %v", err)
	}
}

func BenchmarkEndUserNotificationDispatch(b *testing.B) {
	ds := CreateMySQLDS(b)
	sz := pickNotificationBenchSize(b)
	seedNotificationDispatchData(b, ds, sz)
	ctx := context.Background()

	// every phase after the first works on one batch, so take one now
	batch, err := ds.ListEndUserNotificationsToDispatch(ctx, dispatchBenchHostBatchSize)
	if err != nil {
		b.Fatal(err)
	}
	// fewer than the limit when a host has several due at once, since only the
	// first of each host's is returned
	if len(batch) == 0 {
		b.Fatal("seed produced no dispatchable notifications")
	}

	batchHostIDs := make([]uint, 0, len(batch))
	batchExecutionIDs := make([]string, 0, len(batch))
	for _, notification := range batch {
		batchHostIDs = append(batchHostIDs, notification.HostID)
		batchExecutionIDs = append(batchExecutionIDs, uuid.New().String())
	}

	// the dispatch query, including the GROUP BY that collapses to one
	// notification per host
	b.Run("list", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := ds.ListEndUserNotificationsToDispatch(ctx, dispatchBenchHostBatchSize); err != nil {
				b.Fatal(err)
			}
		}
	})

	// sqlx.In expanding the execution IDs for the child insert. No database, so
	// this is the one phase a CPU profile explains.
	const childInsertStmt = `
INSERT INTO script_upcoming_activities (upcoming_activity_id, script_content_id)
SELECT ua.id, ? FROM upcoming_activities ua WHERE ua.execution_id IN (?)`

	b.Run("build_in_args", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, _, err := sqlx.In(childInsertStmt, 1, batchExecutionIDs); err != nil {
				b.Fatal(err)
			}
		}
	})

	// the two inserts plus activation, which is what the cron actually calls
	b.Run("queue_and_activate", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			b.StopTimer()
			clearQueuedActivities(b, ds)
			b.StartTimer()

			if _, err := ds.BatchNewInternalHostScriptExecutionRequests(ctx, batchHostIDs, fleet.EndUserNotificationScript); err != nil {
				b.Fatal(err)
			}
		}
		clearQueuedActivities(b, ds)
	})

	// activation on its own, so the insert half of the phase above is the
	// difference between the two
	b.Run("activate_only", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			b.StopTimer()
			clearQueuedActivities(b, ds)
			seedUnactivatedScripts(b, ds, batchHostIDs)
			b.StartTimer()

			if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
				return ds.activateNextScriptActivitiesForHosts(ctx, tx, batchHostIDs)
			}); err != nil {
				b.Fatal(err)
			}
		}
		clearQueuedActivities(b, ds)
	})

	// the general activation path, which every other activity type still uses, on
	// the same rows. This is what the script-only path above is measured against.
	b.Run("activate_only_per_host_path", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			b.StopTimer()
			clearQueuedActivities(b, ds)
			seedUnactivatedScripts(b, ds, batchHostIDs)
			b.StartTimer()

			if err := ds.activateNextUpcomingActivityForBatchOfHosts(ctx, batchHostIDs); err != nil {
				b.Fatal(err)
			}
		}
		clearQueuedActivities(b, ds)
	})

	// the multi-row upsert that records which execution each notification was
	// queued as
	b.Run("set_dispatched", func(b *testing.B) {
		b.ReportAllocs()
		for i, notification := range batch {
			notification.ExecutionID = &batchExecutionIDs[i]
		}
		for b.Loop() {
			b.StopTimer()
			resetNotificationsToPending(b, ds)
			b.StartTimer()

			if err := ds.SetEndUserNotificationsDispatched(ctx, batch); err != nil {
				b.Fatal(err)
			}
		}
		resetNotificationsToPending(b, ds)
	})
}
