package mysql

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Batch script queueing and activation, the part of the end user notification
// dispatcher cron (and any other caller of BatchNewInternalHostScriptExecutionRequests)
// that lives in this package. The notifications-specific phases (listing what's
// due, recording what got dispatched) are benchmarked in
// server/notifications/internal/mysql.
//
// To run:
//
//	MYSQL_TEST=1 go test -bench=BenchmarkScriptActivationBatch -benchtime=10x \
//	    -run='^$' ./server/datastore/mysql/
//
// To profile:
//
//	MYSQL_TEST=1 go test -bench=BenchmarkScriptActivationBatch -benchtime=20x \
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

type scriptActivationBenchSize struct {
	numHosts int
}

var scriptActivationBenchSizes = map[string]scriptActivationBenchSize{
	"smoke":     {numHosts: 2_000},
	"realistic": {numHosts: 50_000},
	"large":     {numHosts: 500_000},
}

func pickScriptActivationBenchSize(tb testing.TB) scriptActivationBenchSize {
	tb.Helper()
	name := os.Getenv("FLEET_BENCH_SIZE")
	if name == "" {
		name = "smoke"
	}
	sz, ok := scriptActivationBenchSizes[name]
	if !ok {
		tb.Fatalf("unknown FLEET_BENCH_SIZE=%q (want smoke|realistic|large)", name)
	}
	return sz
}

// seedScriptActivationHosts creates darwin hosts and returns their IDs.
func seedScriptActivationHosts(tb testing.TB, ds *Datastore, sz scriptActivationBenchSize) []uint {
	tb.Helper()

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

	hostIDs := make([]uint, sz.numHosts)
	for i := range hostIDs {
		hostIDs[i] = uint(i + 1) //nolint:gosec // dismiss G115
	}
	return hostIDs
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

	contentsRes, err := insertScriptContents(ctx, ds.writer(ctx), "echo hi")
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

func BenchmarkScriptActivationBatch(b *testing.B) {
	ds := CreateMySQLDS(b)
	sz := pickScriptActivationBenchSize(b)
	hostIDs := seedScriptActivationHosts(b, ds, sz)
	ctx := context.Background()

	executionIDs := make([]string, 0, len(hostIDs))
	for range hostIDs {
		executionIDs = append(executionIDs, uuid.New().String())
	}

	// sqlx.In expanding the execution IDs for the child insert. No database, so
	// this is the one phase a CPU profile explains.
	const childInsertStmt = `
INSERT INTO script_upcoming_activities (upcoming_activity_id, script_content_id)
SELECT ua.id, ? FROM upcoming_activities ua WHERE ua.execution_id IN (?)`

	b.Run("build_in_args", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, _, err := sqlx.In(childInsertStmt, 1, executionIDs); err != nil {
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

			if _, err := ds.BatchNewInternalHostScriptExecutionRequests(ctx, hostIDs, "echo hi"); err != nil {
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
			seedUnactivatedScripts(b, ds, hostIDs)
			b.StartTimer()

			if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
				return ds.activateNextScriptActivitiesForHosts(ctx, tx, hostIDs)
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
			seedUnactivatedScripts(b, ds, hostIDs)
			b.StartTimer()

			if err := ds.activateNextUpcomingActivityForBatchOfHosts(ctx, hostIDs); err != nil {
				b.Fatal(err)
			}
		}
		clearQueuedActivities(b, ds)
	})
}
