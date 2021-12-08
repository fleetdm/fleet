package mysql

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/go-kit/kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const (
	testUsername              = "root"
	testPassword              = "toor"
	testAddress               = "localhost:3307"
	testReplicaDatabaseSuffix = "_replica"
)

func connectMySQL(t testing.TB, testName string, opts *DatastoreTestOptions) *Datastore {
	config := config.MysqlConfig{
		Username: testUsername,
		Password: testPassword,
		Database: testName,
		Address:  testAddress,
	}

	// Create datastore client
	var replicaOpt DBOption
	if opts.Replica {
		replicaConf := config
		replicaConf.Database += testReplicaDatabaseSuffix
		replicaOpt = Replica(&replicaConf)
	}
	ds, err := New(config, clock.NewMockClock(), Logger(log.NewNopLogger()), LimitAttempts(1), replicaOpt)
	require.Nil(t, err)

	if opts.Replica {
		setupReadReplica(t, testName, ds, opts)
	}

	return ds
}

func setupReadReplica(t testing.TB, testName string, ds *Datastore, opts *DatastoreTestOptions) {
	t.Helper()

	// create the context that will cancel the replication goroutine on test exit
	var cancel func()
	ctx := context.Background()
	if tt, ok := t.(*testing.T); ok {
		if dl, ok := tt.Deadline(); ok {
			ctx, cancel = context.WithDeadline(ctx, dl)
		} else {
			ctx, cancel = context.WithCancel(ctx)
		}
	}
	t.Cleanup(cancel)

	// start the replication goroutine that runs when signalled through a
	// channel, the replication runs in lock-step - the test is in control of
	// when the replication happens, by calling opts.RunReplication(), and when
	// that call returns, the replication is guaranteed to be done. This supports
	// simulating all kinds of replica lag.
	ch := make(chan chan struct{})
	go func() {
		// if it exits because of a panic/failed replication, cancel the context
		// immediately so that RunReplication is unblocked too.
		defer cancel()

		primary := ds.writer
		replica := ds.reader.(*sqlx.DB)
		replicaDB := testName + testReplicaDatabaseSuffix
		last := time.Now().Add(-time.Minute)

		// drop all foreign keys in the replica, as that causes issues even with
		// FOREIGN_KEY_CHECKS=0
		var fks []struct {
			TableName      string `db:"table_name"`
			ConstraintName string `db:"constraint_name"`
		}
		err := primary.SelectContext(ctx, &fks, `
          SELECT
            table_name, constraint_name
          FROM
            information_schema.key_column_usage
          WHERE
            table_schema = ? AND
            referenced_table_name IS NOT NULL`, testName)
		require.NoError(t, err)
		for _, fk := range fks {
			stmt := fmt.Sprintf(`ALTER TABLE %s.%s DROP FOREIGN KEY %s`, replicaDB, fk.TableName, fk.ConstraintName)
			_, err := replica.ExecContext(ctx, stmt)
			require.NoError(t, err)
		}

		for {
			select {
			case out := <-ch:
				// identify tables with changes since the last call
				var tables []string
				err := primary.SelectContext(ctx, &tables, `
          SELECT
            table_name
          FROM
            information_schema.tables
          WHERE
            table_schema = ? AND
            table_type = 'BASE TABLE' AND
            update_time >= ?`, testName, last)
				require.NoError(t, err)

				err = primary.GetContext(ctx, &last, `
          SELECT
            MAX(update_time)
          FROM
            information_schema.tables
          WHERE
            table_schema = ? AND
            table_type = 'BASE TABLE'`, testName)
				require.NoError(t, err)

				// replicate by dropping the existing table and re-creating it from
				// the primary.
				for _, tbl := range tables {
					stmt := fmt.Sprintf(`DROP TABLE IF EXISTS %s.%s`, replicaDB, tbl)
					t.Log(stmt)
					_, err = replica.ExecContext(ctx, stmt)
					require.NoError(t, err)
					stmt = fmt.Sprintf(`CREATE TABLE %s.%s LIKE %s.%s`, replicaDB, tbl, testName, tbl)
					t.Log(stmt)
					_, err = replica.ExecContext(ctx, stmt)
					require.NoError(t, err)
					stmt = fmt.Sprintf(`INSERT INTO %s.%s SELECT * FROM %s.%s`, replicaDB, tbl, testName, tbl)
					t.Log(stmt)
					_, err = replica.ExecContext(ctx, stmt)
					require.NoError(t, err)
				}

				out <- struct{}{}
				t.Logf("replication step executed, next will consider updates since %s", last)

			case <-ctx.Done():
				return
			}
		}
	}()

	// set RunReplication to a function that triggers the replication and waits
	// for it to complete.
	opts.RunReplication = func() {
		done := make(chan struct{})
		ch <- done
		select {
		case <-done:
		case <-ctx.Done():
		}
	}
}

// initializeDatabase loads the dumped schema into a newly created database in
// MySQL. This is much faster than running the full set of migrations on each
// test.
func initializeDatabase(t testing.TB, testName string, opts *DatastoreTestOptions) *Datastore {
	_, filename, _, _ := runtime.Caller(0)
	base := path.Dir(filename)
	schema, err := ioutil.ReadFile(path.Join(base, "schema.sql"))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// execute the schema for the test db, and once more for the replica db if
	// that option is set.
	dbs := []string{testName}
	if opts.Replica {
		dbs = append(dbs, testName+testReplicaDatabaseSuffix)
	}
	for _, dbName := range dbs {
		// Load schema from dumpfile
		if out, err := exec.Command(
			"docker-compose", "exec", "-T", "mysql_test",
			// Command run inside container
			"mysql",
			"-u"+testUsername, "-p"+testPassword,
			"-e",
			fmt.Sprintf(
				"DROP DATABASE IF EXISTS %s; CREATE DATABASE %s; USE %s; SET FOREIGN_KEY_CHECKS=0; %s;",
				dbName, dbName, dbName, schema,
			),
		).CombinedOutput(); err != nil {
			t.Error(err)
			t.Error(string(out))
			t.FailNow()
		}
	}
	return connectMySQL(t, testName, opts)
}

// DatastoreTestOptions configures how the test datastore is created
// by CreateMySQLDSWithOptions.
type DatastoreTestOptions struct {
	// Replica indicates that a read replica test database should be created.
	Replica bool

	// RunReplication is the function to call to execute the replication of all
	// missing changes from the primary to the replica. The function is created
	// and set automatically by CreateMySQLDSWithOptions. The test is in full
	// control of when the replication is executed.
	RunReplication func()
}

func createMySQLDSWithOptions(t testing.TB, opts *DatastoreTestOptions) *Datastore {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.Skip("MySQL tests are disabled")
	}

	if tt, ok := t.(*testing.T); ok {
		tt.Parallel()
	}

	if opts == nil {
		// so it is never nil in internal helper functions
		opts = new(DatastoreTestOptions)
	}

	pc, _, _, ok := runtime.Caller(2)
	details := runtime.FuncForPC(pc)
	if !ok || details == nil {
		t.FailNow()
	}

	cleanName := strings.ReplaceAll(
		strings.TrimPrefix(details.Name(), "github.com/fleetdm/fleet/v4/"), "/", "_",
	)
	cleanName = strings.ReplaceAll(cleanName, ".", "_")
	ds := initializeDatabase(t, cleanName, opts)
	t.Cleanup(func() { ds.Close() })
	return ds
}

func CreateMySQLDSWithOptions(t *testing.T, opts *DatastoreTestOptions) *Datastore {
	return createMySQLDSWithOptions(t, opts)
}

func CreateMySQLDS(t testing.TB) *Datastore {
	return createMySQLDSWithOptions(t, nil)
}

func CreateNamedMySQLDS(t *testing.T, name string) *Datastore {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.Skip("MySQL tests are disabled")
	}

	t.Parallel()
	ds := initializeDatabase(t, name, new(DatastoreTestOptions))
	t.Cleanup(func() { ds.Close() })
	return ds
}

func ExecAdhocSQL(tb testing.TB, ds *Datastore, fn func(q sqlx.ExtContext) error) {
	err := fn(ds.writer)
	require.NoError(tb, err)
}

// TruncateTables truncates the specified tables, in order, using ds.writer.
// Note that the order is typically not important because FK checks are
// disabled while truncating. If no table is provided, all tables (except
// those that are seeded by the SQL schema file) are truncated.
func TruncateTables(t testing.TB, ds *Datastore, tables ...string) {
	// By setting DISABLE_TRUNCATE_TABLES a developer can troubleshoot tests
	// by inspecting mysql tables.
	if os.Getenv("DISABLE_TRUNCATE_TABLES") != "" {
		return
	}

	// those tables are seeded with the schema.sql and as such must not
	// be truncated - a more precise approach must be used for those, e.g.
	// delete where id > max before test, or something like that.
	nonEmptyTables := map[string]bool{
		"app_config_json":         true,
		"migration_status_tables": true,
		"osquery_options":         true,
	}
	ctx := context.Background()

	require.NoError(t, ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		var skipSeeded bool

		if len(tables) == 0 {
			skipSeeded = true
			sql := `
      SELECT
        table_name
      FROM
        information_schema.tables
      WHERE
        table_schema = database() AND
        table_type = 'BASE TABLE'
    `
			if err := sqlx.SelectContext(ctx, tx, &tables, sql); err != nil {
				return err
			}
		}

		if _, err := tx.ExecContext(ctx, `SET FOREIGN_KEY_CHECKS=0`); err != nil {
			return err
		}
		for _, tbl := range tables {
			if nonEmptyTables[tbl] {
				if skipSeeded {
					continue
				}
				return fmt.Errorf("cannot truncate table %s, it contains seed data from schema.sql", tbl)
			}
			if _, err := tx.ExecContext(ctx, "TRUNCATE TABLE "+tbl); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `SET FOREIGN_KEY_CHECKS=1`); err != nil {
			return err
		}
		return nil
	}))
}
