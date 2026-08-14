package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/acl/notificationsacl"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	notifications_api "github.com/fleetdm/fleet/v4/server/notifications/api"
	notifications_bootstrap "github.com/fleetdm/fleet/v4/server/notifications/bootstrap"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
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
				notifications_api.EndUserNotificationPending,
				"patch",
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
			notifications_api.EndUserNotificationPending),
	}
	for _, stmt := range stmts {
		mysqltest.ExecAdhocSQL(tb, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), stmt)
			return err
		})
	}
}

// countDispatchableNotifications confirms the seed produced the expected
// amount of due work, without depending on the notifications bounded
// context's internals.
func countDispatchableNotifications(tb testing.TB, ds *mysql.Datastore) int {
	tb.Helper()
	var count int
	mysqltest.ExecAdhocSQL(tb, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &count,
			"SELECT COUNT(*) FROM end_user_notifications WHERE status = ?", notifications_api.EndUserNotificationPending)
	})
	return count
}

func newBenchNotificationsService(tb testing.TB, ds *mysql.Datastore, logger *slog.Logger) notifications_api.Service {
	tb.Helper()
	dbConns := &common_mysql.DBConnections{Primary: ds.TestPrimaryDB(), Replica: ds.TestPrimaryDB()}
	svc, _ := notifications_bootstrap.New(dbConns, notificationsacl.NewFleetServiceAdapter(ds), logger)
	return svc
}

func BenchmarkDispatchEndUserNotifications(b *testing.B) {
	ds := mysqltest.CreateMySQLDS(b)
	sz := pickCronBenchSize(b)
	seedDispatchableFleet(b, ds, sz)

	logger := slog.New(slog.DiscardHandler)
	notificationsSvc := newBenchNotificationsService(b, ds, logger)

	// confirm the seed really is all dispatchable, or the pass would measure
	// almost nothing
	if countDispatchableNotifications(b, ds) == 0 {
		b.Fatal("seed produced no dispatchable notifications")
	}

	passes := 0
	b.ReportAllocs()
	for b.Loop() {
		b.StopTimer()
		resetDispatchedFleet(b, ds)
		b.StartTimer()

		if err := notificationsSvc.Dispatch(b.Context()); err != nil {
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

	logger := slog.New(slog.DiscardHandler)
	notificationsSvc := newBenchNotificationsService(b, ds, logger)

	mysqltest.ExecAdhocSQL(b, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(), "UPDATE end_user_notifications SET status = ?",
			notifications_api.EndUserNotificationDispatched)
		return err
	})

	if countDispatchableNotifications(b, ds) != 0 {
		b.Fatal("notifications still dispatchable, want none")
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := notificationsSvc.Dispatch(b.Context()); err != nil {
			b.Fatal(err)
		}
	}
}
