package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// One full pass of the end user notification dispatcher cron, from expiring
// through to the last batch, on a fleet where every host has a notification due.
// Per-phase numbers live in the datastore package's benchmark of the same name.
//
// To run:
//
//	MYSQL_TEST=1 go test -bench=BenchmarkDispatchEndUserNotifications -benchtime=5x \
//	    -run='^$' ./cmd/fleet/
//
// To profile:
//
//	MYSQL_TEST=1 go test -bench=BenchmarkDispatchEndUserNotifications -benchtime=5x \
//	    -run='^$' -cpuprofile=/tmp/cron-cpu.out -blockprofile=/tmp/cron-block.out \
//	    ./cmd/fleet/
//	go tool pprof -http=: /tmp/cron-block.out
//
// A pass is mostly waiting on MySQL, so the block profile is the informative one;
// a CPU profile shows netpoll. The hosts/s metric is the number to watch, since it
// says how much of a fleet one minute of this cron can get through.
//
// Tune with FLEET_BENCH_SIZE: "tiny" (10 hosts), "smoke" (default, 2 batches), "realistic" (20
// batches), "large" (200 batches). Absolute numbers from a local MySQL container
// are not prod numbers.

type cronBenchSize struct {
	numHosts int
}

var cronBenchSizes = map[string]cronBenchSize{
	"tiny":      {numHosts: 10},
	"smoke":     {numHosts: 1_000},
	"realistic": {numHosts: 10_000},
	"large":     {numHosts: 100_000},
}

func pickCronBenchSize(tb testing.TB) cronBenchSize {
	tb.Helper()
	name := os.Getenv("FLEET_BENCH_SIZE")
	if name == "" {
		name = "smoke"
	}
	sz, ok := cronBenchSizes[name]
	if !ok {
		tb.Fatalf("unknown FLEET_BENCH_SIZE=%q (want smoke|realistic|large)", name)
	}
	return sz
}

func benchBatchInsert(tb testing.TB, ds *mysql.Datastore, prefix string, n, cols int, row func(i int) []any) {
	tb.Helper()
	const rowsPerStmt = 500
	placeholder := "(" + strings.Repeat("?,", cols-1) + "?)"
	for start := 0; start < n; start += rowsPerStmt {
		end := min(start+rowsPerStmt, n)
		parts := make([]string, 0, end-start)
		args := make([]any, 0, (end-start)*cols)
		for i := start; i < end; i++ {
			parts = append(parts, placeholder)
			args = append(args, row(i)...)
		}
		stmt := prefix + strings.Join(parts, ",")
		mysqltest.ExecAdhocSQL(tb, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), stmt, args...)
			return err
		})
	}
}

// seedDispatchableFleet creates darwin hosts running Fleet Desktop, each with one
// pending notification due now, which is the state a pass has the most work to do
// in.
func seedDispatchableFleet(tb testing.TB, ds *mysql.Datastore, sz cronBenchSize) {
	tb.Helper()

	benchBatchInsert(tb, ds,
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

	benchBatchInsert(tb, ds,
		"INSERT INTO host_orbit_info (host_id, version, desktop_version, scripts_enabled) VALUES ",
		sz.numHosts, 4, func(i int) []any {
			return []any{i + 1, "1.40.0", "1.5.0", true}
		})

	payload, err := json.Marshal(map[string]any{"apps": []string{"1Password", "Slack"}})
	if err != nil {
		tb.Fatal(err)
	}

	benchBatchInsert(tb, ds,
		"INSERT INTO end_user_notifications (uuid, host_id, status, kind, payload) VALUES ",
		sz.numHosts, 5, func(i int) []any {
			return []any{
				fmt.Sprintf("notification-%d", i+1),
				i + 1,
				fleet.EndUserNotificationPending,
				"notify_before_patching",
				payload,
			}
		})
}

// resetDispatchedFleet undoes a pass, so the next iteration starts with the same
// amount of work to do.
func resetDispatchedFleet(tb testing.TB, ds *mysql.Datastore) {
	tb.Helper()
	stmts := []string{
		"DELETE FROM script_upcoming_activities",
		"DELETE FROM upcoming_activities",
		"DELETE FROM host_script_results",
		fmt.Sprintf("UPDATE end_user_notifications SET status = '%s', execution_id = NULL, attempt_count = 0",
			fleet.EndUserNotificationPending),
	}
	for _, stmt := range stmts {
		mysqltest.ExecAdhocSQL(tb, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), stmt)
			return err
		})
	}
}

func BenchmarkDispatchEndUserNotifications(b *testing.B) {
	ds := mysqltest.CreateMySQLDS(b)
	sz := pickCronBenchSize(b)
	seedDispatchableFleet(b, ds, sz)

	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	// confirm the seed really is all dispatchable, or the pass would measure
	// almost nothing
	dispatchable, err := ds.ListEndUserNotificationsToDispatch(ctx, sz.numHosts)
	if err != nil {
		b.Fatal(err)
	}
	if len(dispatchable) == 0 {
		b.Fatal("seed produced no dispatchable notifications")
	}

	passes := 0
	b.ReportAllocs()
	for b.Loop() {
		b.StopTimer()
		resetDispatchedFleet(b, ds)
		b.StartTimer()

		if err := dispatchEndUserNotifications(ctx, ds, logger); err != nil {
			b.Fatal(err)
		}
		passes++
	}

	if elapsed := b.Elapsed().Seconds(); elapsed > 0 {
		b.ReportMetric(float64(sz.numHosts*passes)/elapsed, "hosts/s")
	}
}

// What the cron costs on a fleet with nothing due, which is what almost every
// run of it does. The hosts and their notifications exist, they are just all
// dispatched already, so a pass expires and then finds an empty batch.
func BenchmarkDispatchEndUserNotificationsIdle(b *testing.B) {
	ds := mysqltest.CreateMySQLDS(b)
	sz := pickCronBenchSize(b)
	seedDispatchableFleet(b, ds, sz)

	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	mysqltest.ExecAdhocSQL(b, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE end_user_notifications SET status = ?",
			fleet.EndUserNotificationDispatched)
		return err
	})

	nothingDue, err := ds.ListEndUserNotificationsToDispatch(ctx, sz.numHosts)
	if err != nil {
		b.Fatal(err)
	}
	if len(nothingDue) != 0 {
		b.Fatalf("%d notifications still dispatchable, want none", len(nothingDue))
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := dispatchEndUserNotifications(ctx, ds, logger); err != nil {
			b.Fatal(err)
		}
	}
}
