// Package mysqltest provides helpers (test datastore setup, cleanup, ABM /
// MDM fixtures, activity-service helpers) for tests that exercise the
// fleetdm/fleet mysql datastore.
//
// It imports the "testing" package and is therefore only imported from test
// code; importing it from production code would pull "testing" into the
// resulting binary.
package mysqltest

import (
	"bytes"
	"context" // nolint:gosec // this is a test package
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/acl/activityacl"
	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	activity_bootstrap "github.com/fleetdm/fleet/v4/server/activity/bootstrap"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	mdmtesting "github.com/fleetdm/fleet/v4/server/mdm/testing_utils"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	"github.com/fleetdm/fleet/v4/server/platform/logging"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/olekukonko/tablewriter"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultMinLastOpenedAtDiff mirrors the constant in package mysql.
const defaultMinLastOpenedAtDiff = time.Hour

// fromCommonMysqlConfig mirrors the same-named helper in package mysql.
func fromCommonMysqlConfig(conf *common_mysql.MysqlConfig) *config.MysqlConfig {
	if conf == nil {
		return nil
	}
	return &config.MysqlConfig{
		Protocol:        conf.Protocol,
		Address:         conf.Address,
		Username:        conf.Username,
		Password:        conf.Password,
		PasswordPath:    conf.PasswordPath,
		Database:        conf.Database,
		TLSCert:         conf.TLSCert,
		TLSKey:          conf.TLSKey,
		TLSCA:           conf.TLSCA,
		TLSServerName:   conf.TLSServerName,
		TLSConfig:       conf.TLSConfig,
		MaxOpenConns:    conf.MaxOpenConns,
		MaxIdleConns:    conf.MaxIdleConns,
		ConnMaxLifetime: conf.ConnMaxLifetime,
		SQLMode:         conf.SQLMode,
		Region:          conf.Region,
	}
}

// toCommonMysqlConfig mirrors the same-named helper in package mysql.
func toCommonMysqlConfig(conf *config.MysqlConfig) *common_mysql.MysqlConfig {
	return &common_mysql.MysqlConfig{
		Protocol:        conf.Protocol,
		Address:         conf.Address,
		Username:        conf.Username,
		Password:        conf.Password,
		PasswordPath:    conf.PasswordPath,
		Database:        conf.Database,
		TLSCert:         conf.TLSCert,
		TLSKey:          conf.TLSKey,
		TLSCA:           conf.TLSCA,
		TLSServerName:   conf.TLSServerName,
		TLSConfig:       conf.TLSConfig,
		MaxOpenConns:    conf.MaxOpenConns,
		MaxIdleConns:    conf.MaxIdleConns,
		ConnMaxLifetime: conf.ConnMaxLifetime,
		SQLMode:         conf.SQLMode,
		Region:          conf.Region,
	}
}

func connectMySQL(t testing.TB, testName string, opts *testing_utils.DatastoreTestOptions) *mysql.Datastore {
	commonCfg := testing_utils.MysqlTestConfig(testName)
	cfg := fromCommonMysqlConfig(commonCfg)

	// Create datastore client
	var replicaOpt mysql.DBOption
	if opts.DummyReplica {
		replicaConf := *cfg
		replicaConf.Database += testing_utils.TestReplicaDatabaseSuffix
		replicaOpt = mysql.Replica(&replicaConf)
	}

	// For use with WithFleetConfig. Note that since we're setting up the DB in a different way
	// than in production, we have to reset the MinSoftwareLastOpenedAtDiff field to its default so
	// it's not overwritten here.
	tc := config.TestConfig()
	tc.Osquery.MinSoftwareLastOpenedAtDiff = defaultMinLastOpenedAtDiff

	var dslogger *slog.Logger
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		dslogger = slog.New(slog.DiscardHandler)
	} else {
		dslogger = logging.NewSlogLogger(logging.Options{Output: os.Stdout, Debug: true})
	}

	// Use TestSQLMode which combines ANSI mode components with MySQL 8 strict modes
	// This ensures we catch data truncation errors and other strict behaviors during testing
	// Reference: https://dev.mysql.com/doc/refman/8.0/en/sql-mode.html
	ds, err := mysql.New(*cfg, clock.NewMockClock(), mysql.Logger(dslogger), mysql.LimitAttempts(1), replicaOpt, mysql.SQLMode(common_mysql.TestSQLMode), mysql.WithFleetConfig(&tc))
	require.NoError(t, err)

	if opts.DummyReplica {
		setupDummyReplica(t, testName, ds, opts)
	}
	if opts.RealReplica {
		replicaOpts := &common_mysql.DBOptions{
			MinLastOpenedAtDiff: defaultMinLastOpenedAtDiff,
			MaxAttempts:         1,
			Logger:              slog.New(slog.DiscardHandler),
			SqlMode:             common_mysql.TestSQLMode,
		}
		setupRealReplica(t, testName, ds, replicaOpts)
	}

	return ds
}

func setupDummyReplica(t testing.TB, testName string, ds *mysql.Datastore, opts *testing_utils.DatastoreTestOptions) {
	t.Helper()

	// create the context that will cancel the replication goroutine on test exit
	ctx, cancel := context.WithCancel(context.Background())
	if tt, ok := t.(*testing.T); ok {
		if dl, ok := tt.Deadline(); ok {
			ctx, cancel = context.WithDeadline(ctx, dl)
		}
	}
	t.Cleanup(cancel)

	type replicationRun struct {
		forceTables     []string
		replicationDone chan struct{}
	}

	// start the replication goroutine that runs when signalled through a
	// channel, the replication runs in lock-step - the test is in control of
	// when the replication happens, by calling opts.RunReplication(), and when
	// that call returns, the replication is guaranteed to be done.
	ch := make(chan replicationRun)
	go func() {
		defer cancel()

		primary := ds.TestPrimaryDB()
		replica := ds.TestReplica().(*sqlx.DB)
		replicaDB := testName + testing_utils.TestReplicaDatabaseSuffix
		last := time.Now().Add(-time.Minute)

		// drop all foreign keys in the replica
		var fks []struct {
			TableName      string `db:"TABLE_NAME"`
			ConstraintName string `db:"CONSTRAINT_NAME"`
		}
		err := primary.SelectContext(ctx, &fks, `
          SELECT
            TABLE_NAME, CONSTRAINT_NAME
          FROM
            INFORMATION_SCHEMA.KEY_COLUMN_USAGE
          WHERE
            TABLE_SCHEMA = ? AND
            REFERENCED_TABLE_NAME IS NOT NULL`, testName)
		if !assert.NoError(t, err) {
			return
		}
		for _, fk := range fks {
			stmt := fmt.Sprintf(`ALTER TABLE %s.%s DROP FOREIGN KEY %s`, replicaDB, fk.TableName, fk.ConstraintName)
			_, err := replica.ExecContext(ctx, stmt)
			if err != nil && strings.Contains(err.Error(), "check that column/key exists") {
				continue
			}

			if !assert.NoError(t, err) {
				return
			}
		}

		for {
			select {
			case out := <-ch:
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
				if !assert.NoError(t, err) {
					return
				}

				tableSet := make(map[string]struct{}, len(tables)+len(out.forceTables))
				for _, tbl := range tables {
					tableSet[tbl] = struct{}{}
				}
				for _, tbl := range out.forceTables {
					tableSet[tbl] = struct{}{}
				}
				tables = tables[:0]
				for tbl := range tableSet {
					tables = append(tables, tbl)
				}
				t.Logf("changed tables since %v: %v", last, tables)

				err = primary.GetContext(ctx, &last, `
          SELECT
            MAX(update_time)
          FROM
            information_schema.tables
          WHERE
            table_schema = ? AND
            table_type = 'BASE TABLE'`, testName)
				if !assert.NoError(t, err) {
					return
				}
				t.Logf("last update time of primary is now %v", last)

				for _, tbl := range tables {
					stmt := fmt.Sprintf(`DROP TABLE IF EXISTS %s.%s`, replicaDB, tbl)
					t.Log(stmt)
					_, err = replica.ExecContext(ctx, stmt)
					if !assert.NoError(t, err) {
						return
					}

					stmt = fmt.Sprintf(`CREATE TABLE %s.%s LIKE %s.%s`, replicaDB, tbl, testName, tbl)
					t.Log(stmt)
					_, err = replica.ExecContext(ctx, stmt)
					if !assert.NoError(t, err) {
						return
					}

					var columns string
					columnsStmt := fmt.Sprintf(`SELECT
                                                  GROUP_CONCAT(column_name ORDER BY ordinal_position)
                                                FROM information_schema.columns
                                                WHERE table_schema = '%s' AND table_name = '%s'
												  AND NOT (EXTRA LIKE '%%GENERATED%%' AND EXTRA NOT LIKE '%%DEFAULT_GENERATED%%');`, replicaDB, tbl)
					err = replica.GetContext(ctx, &columns, columnsStmt)
					if !assert.NoError(t, err) {
						return
					}

					stmt = fmt.Sprintf(`INSERT INTO %s.%s (%s)
                                        SELECT %s
                                        FROM %s.%s;`, replicaDB, tbl, columns, columns, testName, tbl)
					t.Log(stmt)
					_, err = replica.ExecContext(ctx, stmt)
					if !assert.NoError(t, err) {
						return
					}
				}

				out.replicationDone <- struct{}{}
				t.Logf("replication step executed, next will consider updates since %s", last)

			case <-ctx.Done():
				return
			}
		}
	}()

	opts.RunReplication = func(forceTables ...string) {
		done := make(chan struct{})
		ch <- replicationRun{forceTables, done}
		select {
		case <-done:
		case <-ctx.Done():
		}
	}
}

// we need to keep track of the databases that need replication in order to
// configure the replica to only track those.
var (
	mu                   sync.Mutex
	databasesToReplicate string
)

func setupRealReplica(t testing.TB, testName string, ds *mysql.Datastore, options *common_mysql.DBOptions) {
	t.Helper()
	const replicaUser = "replicator"
	const replicaPassword = "rotacilper"

	t.Cleanup(
		func() {
			if out, err := exec.Command(
				"docker", "compose", "exec", "-T", "mysql_replica_test",
				"mysql",
				"-u"+testing_utils.TestUsername, "-p"+testing_utils.TestPassword,
				"-e",
				"STOP REPLICA; RESET REPLICA ALL;",
			).CombinedOutput(); err != nil {
				t.Log(err)
				t.Log(string(out))
			}
		},
	)

	ctx := context.Background()

	primary := ds.TestPrimaryDB()
	_, err := primary.ExecContext(ctx, fmt.Sprintf("DROP USER IF EXISTS '%s'", replicaUser))
	require.NoError(t, err)
	_, err = primary.ExecContext(ctx, fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", replicaUser, replicaPassword))
	require.NoError(t, err)
	_, err = primary.ExecContext(ctx, fmt.Sprintf("GRANT REPLICATION SLAVE ON *.* TO '%s'@'%%'", replicaUser))
	require.NoError(t, err)
	_, err = primary.ExecContext(ctx, "FLUSH PRIVILEGES")
	require.NoError(t, err)

	var version string
	err = primary.GetContext(ctx, &version, "SELECT VERSION()")
	require.NoError(t, err)

	ms, err := ds.MasterStatus(ctx, version)
	require.NoError(t, err)

	mu.Lock()
	databasesToReplicate = strings.TrimPrefix(databasesToReplicate+fmt.Sprintf(", `%s`", testName), ",")
	mu.Unlock()

	setSourceStmt := fmt.Sprintf(`
			CHANGE REPLICATION SOURCE TO
				GET_SOURCE_PUBLIC_KEY=1,
				SOURCE_HOST='mysql_test',
				SOURCE_USER='%s',
				SOURCE_PASSWORD='%s',
				SOURCE_LOG_FILE='%s',
				SOURCE_LOG_POS=%d
		`, replicaUser, replicaPassword, ms.File, ms.Position)
	if strings.HasPrefix(version, "8.0") {
		setSourceStmt = fmt.Sprintf(`
			CHANGE MASTER TO
				GET_MASTER_PUBLIC_KEY=1,
				MASTER_HOST='mysql_test',
				MASTER_USER='%s',
				MASTER_PASSWORD='%s',
				MASTER_LOG_FILE='%s',
				MASTER_LOG_POS=%d
		`, replicaUser, replicaPassword, ms.File, ms.Position)
	}

	if out, err := exec.Command(
		"docker", "compose", "exec", "-T", "mysql_replica_test",
		"mysql",
		"-u"+testing_utils.TestUsername, "-p"+testing_utils.TestPassword,
		"-e",
		fmt.Sprintf(
			`
			STOP REPLICA;
			RESET REPLICA ALL;
			CHANGE REPLICATION FILTER REPLICATE_DO_DB = ( %s );
			%s;
			START REPLICA;
			`, databasesToReplicate, setSourceStmt,
		),
	).CombinedOutput(); err != nil {
		t.Error(err)
		t.Error(string(out))
		t.FailNow()
	}

	replicaConfig := config.MysqlConfig{
		Username: testing_utils.TestUsername,
		Password: testing_utils.TestPassword,
		Database: testName,
		Address:  testing_utils.TestReplicaAddress,
	}
	require.NoError(t, mysql.CheckAndModifyMysqlConfig(&replicaConfig))
	replica, err := mysql.NewDB(&replicaConfig, options)
	require.NoError(t, err)
	ds.TestSetReplica(replica)
	ds.TestSetReadReplicaConfig(toCommonMysqlConfig(&replicaConfig))
}

func initializeDatabase(t testing.TB, testName string, opts *testing_utils.DatastoreTestOptions) *mysql.Datastore {
	testing_utils.LoadDefaultSchema(t, testName, opts)
	return connectMySQL(t, testName, opts)
}

func createMySQLDSWithOptions(t testing.TB, opts *testing_utils.DatastoreTestOptions) *mysql.Datastore {
	cleanTestName, opts := testing_utils.ProcessOptions(t, opts)
	ds := initializeDatabase(t, cleanTestName, opts)
	t.Cleanup(func() { ds.Close() })
	return ds
}

// CreateMySQLDSWithReplica creates a *mysql.Datastore with a real MySQL
// replica, retrying until the replica is running.
func CreateMySQLDSWithReplica(t *testing.T, opts *testing_utils.DatastoreTestOptions) *mysql.Datastore {
	if opts == nil {
		opts = new(testing_utils.DatastoreTestOptions)
	}
	opts.RealReplica = true
	const numberOfAttempts = 10
	var ds *mysql.Datastore
	for attempt := 0; attempt < numberOfAttempts; {
		attempt++
		ds = createMySQLDSWithOptions(t, opts)
		status, err := ds.ReplicaStatus(context.Background())
		require.NoError(t, err)
		if status["Replica_SQL_Running"] != "Yes" {
			t.Logf("create replica attempt: %d replica status: %+v", attempt, status)
			if lastErr, ok := status["Last_Error"]; ok && lastErr != "" {
				t.Logf("replica not running after attempt %d; Last_Error: %s", attempt, lastErr)
			}
			ds.Close()
			continue
		}
		break
	}
	require.NotNil(t, ds)
	return ds
}

// CreateMySQLDSWithOptions creates a *mysql.Datastore using the provided
// DatastoreTestOptions.
func CreateMySQLDSWithOptions(t *testing.T, opts *testing_utils.DatastoreTestOptions) *mysql.Datastore {
	return createMySQLDSWithOptions(t, opts)
}

// CreateMySQLDS creates a *mysql.Datastore using default test options.
func CreateMySQLDS(t testing.TB) *mysql.Datastore {
	return createMySQLDSWithOptions(t, nil)
}

// CreateNamedMySQLDS creates a *mysql.Datastore with the given database name.
func CreateNamedMySQLDS(t *testing.T, name string) *mysql.Datastore {
	ds, _ := CreateNamedMySQLDSWithConns(t, name)
	return ds
}

// CreateNamedMySQLDSWithConns creates a *mysql.Datastore and returns both
// the datastore and the underlying database connections. This matches the
// production flow where DBConnections are created first and shared across
// datastores.
func CreateNamedMySQLDSWithConns(t *testing.T, name string) (*mysql.Datastore, *common_mysql.DBConnections) {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.Skip("MySQL tests are disabled")
	}

	ds := initializeDatabase(t, name, new(testing_utils.DatastoreTestOptions))
	t.Cleanup(func() { ds.Close() })

	return ds, TestDBConnections(t, ds)
}

// ExecAdhocSQL runs the given function with the primary *sqlx.DB and fails
// the test on error.
func ExecAdhocSQL(tb testing.TB, ds *mysql.Datastore, fn func(q sqlx.ExtContext) error) {
	tb.Helper()
	err := fn(ds.TestPrimaryDB())
	require.NoError(tb, err)
}

// ExecAdhocSQLWithError runs the given function with the primary *sqlx.DB
// and returns the error.
func ExecAdhocSQLWithError(ds *mysql.Datastore, fn func(q sqlx.ExtContext) error) error {
	return fn(ds.TestPrimaryDB())
}

// EncryptWithPrivateKey encrypts data with the server private key associated
// with the Datastore.
func EncryptWithPrivateKey(tb testing.TB, ds *mysql.Datastore, data []byte) ([]byte, error) {
	return ds.TestEncrypt(data)
}

// TruncateTables truncates the named tables, skipping a small allowlist of
// seed tables that must keep their schema-loaded rows.
func TruncateTables(t testing.TB, ds *mysql.Datastore, tables ...string) {
	nonEmptyTables := map[string]bool{
		"app_config_json":                  true,
		"fleet_variables":                  true,
		"mdm_apple_declaration_categories": true,
		"mdm_delivery_status":              true,
		"mdm_operation_types":              true,
		"migration_status_tables":          true,
		"osquery_options":                  true,
		"software_categories":              true,
	}
	testing_utils.TruncateTables(t, ds.TestWriter(context.Background()), ds.TestLogger(), nonEmptyTables, tables...)
}

// DumpTable prints all rows in the given table for debugging.
func DumpTable(t *testing.T, q sqlx.QueryerContext, tableName string, cols ...string) { //nolint: unused
	colList := "*"
	if len(cols) > 0 {
		colList = strings.Join(cols, ", ")
	}
	rows, err := q.QueryContext(context.Background(), fmt.Sprintf(`SELECT %s FROM %s`, colList, tableName))
	require.NoError(t, err)
	defer rows.Close()

	t.Logf(">> dumping table %s:", tableName)

	data := [][]string{}
	columns, err := rows.Columns()
	require.NoError(t, err)

	var anyDst []any
	var strDst []sql.NullString
	for rows.Next() {
		if anyDst == nil {
			anyDst = make([]any, len(columns))
			strDst = make([]sql.NullString, len(columns))
			for i := range columns {
				anyDst[i] = &strDst[i]
			}
		}
		require.NoError(t, rows.Scan(anyDst...))

		row := []string{}
		for _, v := range strDst {
			if v.Valid {
				row = append(row, v.String)
			} else {
				row = append(row, "NULL")
			}
		}
		data = append(data, row)
	}
	require.NoError(t, rows.Err())

	printDumpTable(t, columns, data)
	t.Logf("<< dumping table %s completed", tableName)
}

func printDumpTable(t *testing.T, cols []string, rows [][]string) {
	writer := bytes.NewBufferString("")
	table := tablewriter.NewWriter(writer)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderLine(true)
	table.SetRowLine(false)

	table.SetHeader(cols)
	table.AppendBulk(rows)
	table.Render()

	t.Logf("\n%s", writer.String())
}

func generateDummyWindowsProfile(uuidStr string) []byte {
	return fmt.Appendf([]byte{}, `<Atomic><Replace><Item><Target><LocURI>./Device/Foo/%s</LocURI></Target></Item></Replace></Atomic>`, uuidStr)
}

// InsertWindowsProfileForTest inserts a dummy windows profile for the given
// teamID and returns its UUID.
func InsertWindowsProfileForTest(t *testing.T, ds *mysql.Datastore, teamID uint) string {
	profUUID := "w" + uuid.NewString()
	prof := generateDummyWindowsProfile(profUUID)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		stmt := `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml, uploaded_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP);`
		_, err := q.ExecContext(context.Background(), stmt, profUUID, teamID, fmt.Sprintf("name-%s", profUUID), prof)
		return err
	})
	return profUUID
}

// GetAggregatedStats retrieves aggregated stats for the given query.
func GetAggregatedStats(ctx context.Context, ds *mysql.Datastore, aggregate fleet.AggregatedStatsType, id uint) (fleet.AggregatedStats, error) {
	result := fleet.AggregatedStats{}
	stmt := `
	SELECT
		   JSON_EXTRACT(json_value, '$.user_time_p50') as user_time_p50,
		   JSON_EXTRACT(json_value, '$.user_time_p95') as user_time_p95,
		   JSON_EXTRACT(json_value, '$.system_time_p50') as system_time_p50,
		   JSON_EXTRACT(json_value, '$.system_time_p95') as system_time_p95,
		   JSON_EXTRACT(json_value, '$.total_executions') as total_executions
	FROM aggregated_stats WHERE id=? AND type=?
	`
	err := sqlx.GetContext(ctx, ds.TestReader(ctx), &result, stmt, id, aggregate)
	return result, err
}

// SetOrderedCreatedAtTimestamps enforces an ordered sequence of created_at
// timestamps in a database table. The first timestamp will be after
// afterTime, and each provided key will have a timestamp incremented by 1s.
func SetOrderedCreatedAtTimestamps(t testing.TB, ds *mysql.Datastore, afterTime time.Time, table, keyCol string, keys ...any) time.Time {
	now := afterTime
	for i := range keys {
		now = now.Add(time.Second)
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(),
				fmt.Sprintf(`UPDATE %s SET created_at=? WHERE %s=?`, table, keyCol), now, keys[i])
			return err
		})
	}
	return now
}

// CreateABMKeyCertIfNotExists ensures the ABM key/cert mdm_config_assets rows
// exist (generating them if missing).
func CreateABMKeyCertIfNotExists(t testing.TB, ds *mysql.Datastore) {
	certPEM, keyPEM, _, err := GenerateTestABMAssets(t)
	require.NoError(t, err)
	var assets []fleet.MDMConfigAsset
	_, err = ds.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
		fleet.MDMAssetABMKey,
	}, nil)
	if err != nil {
		var nfe fleet.NotFoundError
		require.ErrorAs(t, err, &nfe)
		assets = append(assets, fleet.MDMConfigAsset{Name: fleet.MDMAssetABMKey, Value: keyPEM})
	}

	_, err = ds.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
		fleet.MDMAssetABMCert,
	}, nil)
	if err != nil {
		var nfe fleet.NotFoundError
		require.ErrorAs(t, err, &nfe)
		assets = append(assets, fleet.MDMConfigAsset{Name: fleet.MDMAssetABMCert, Value: certPEM})
	}

	if len(assets) != 0 {
		err = ds.InsertMDMConfigAssets(context.Background(), assets, ds.TestWriter(context.Background()))
		require.NoError(t, err)
	}
}

// CreateAndSetABMToken creates a new ABM token (using an existing ABM
// key/cert) and stores it in the DB.
func CreateAndSetABMToken(t testing.TB, ds *mysql.Datastore, orgName string) *fleet.ABMToken {
	assets, err := ds.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
		fleet.MDMAssetABMKey,
		fleet.MDMAssetABMCert,
	}, nil)
	require.NoError(t, err)

	certPEM := assets[fleet.MDMAssetABMCert].Value

	testBMToken := &nanodep_client.OAuth1Tokens{
		ConsumerKey:       "test_consumer",
		ConsumerSecret:    "test_secret",
		AccessToken:       "test_access_token",
		AccessSecret:      "test_access_secret",
		AccessTokenExpiry: time.Date(2037, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	rawToken, err := json.Marshal(testBMToken) //nolint:gosec // test helper, fake credentials
	require.NoError(t, err)

	smimeToken := fmt.Sprintf(
		"Content-Type: text/plain;charset=UTF-8\r\n"+
			"Content-Transfer-Encoding: 7bit\r\n"+
			"\r\n%s", rawToken,
	)

	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE", block.Type)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	encryptedToken, err := pkcs7.Encrypt([]byte(smimeToken), []*x509.Certificate{cert})
	require.NoError(t, err)

	tokenBytes := fmt.Sprintf(
		"Content-Type: application/pkcs7-mime; name=\"smime.p7m\"; smime-type=enveloped-data\r\n"+
			"Content-Transfer-Encoding: base64\r\n"+
			"Content-Disposition: attachment; filename=\"smime.p7m\"\r\n"+
			"Content-Description: S/MIME Encrypted Message\r\n"+
			"\r\n%s", base64.StdEncoding.EncodeToString(encryptedToken))

	tok, err := ds.InsertABMToken(context.Background(), &fleet.ABMToken{
		EncryptedToken:   []byte(tokenBytes),
		OrganizationName: orgName,
		RenewAt:          time.Now().Add(30 * 24 * time.Hour),
	})
	require.NoError(t, err)
	return tok
}

// SetTestABMAssets seeds APNS / ABM / CA assets and ABM token, marks MDM
// enabled in app config, and returns the ABM token.
func SetTestABMAssets(t testing.TB, ds *mysql.Datastore, orgName string) *fleet.ABMToken {
	apnsCert, apnsKey, err := GenerateTestCertBytes(mdmtesting.NewTestMDMAppleCertTemplate())
	require.NoError(t, err)

	certPEM, keyPEM, tokenBytes, err := GenerateTestABMAssets(t)
	require.NoError(t, err)
	assets := []fleet.MDMConfigAsset{
		{Name: fleet.MDMAssetABMCert, Value: certPEM},
		{Name: fleet.MDMAssetABMKey, Value: keyPEM},
		{Name: fleet.MDMAssetAPNSCert, Value: apnsCert},
		{Name: fleet.MDMAssetAPNSKey, Value: apnsKey},
		{Name: fleet.MDMAssetCACert, Value: certPEM},
		{Name: fleet.MDMAssetCAKey, Value: keyPEM},
	}

	err = ds.InsertMDMConfigAssets(context.Background(), assets, nil)
	require.NoError(t, err)

	tok, err := ds.InsertABMToken(context.Background(), &fleet.ABMToken{
		EncryptedToken:   tokenBytes,
		OrganizationName: orgName,
		RenewAt:          time.Now().Add(30 * 24 * time.Hour),
	})
	require.NoError(t, err)

	appCfg, err := ds.AppConfig(context.Background())
	require.NoError(t, err)
	appCfg.MDM.EnabledAndConfigured = true
	appCfg.MDM.AppleBMEnabledAndConfigured = true
	err = ds.SaveAppConfig(context.Background(), appCfg)
	require.NoError(t, err)

	return tok
}

// GenerateTestABMAssets returns a freshly generated ABM cert PEM, key PEM,
// and encrypted token bytes.
func GenerateTestABMAssets(t testing.TB) ([]byte, []byte, []byte, error) {
	certPEM, keyPEM, err := GenerateTestCertBytes(mdmtesting.NewTestMDMAppleCertTemplate())
	require.NoError(t, err)

	testBMToken := &nanodep_client.OAuth1Tokens{
		ConsumerKey:       "test_consumer",
		ConsumerSecret:    "test_secret",
		AccessToken:       "test_access_token",
		AccessSecret:      "test_access_secret",
		AccessTokenExpiry: time.Date(2037, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	rawToken, err := json.Marshal(testBMToken) //nolint:gosec // test helper, fake credentials
	require.NoError(t, err)

	smimeToken := fmt.Sprintf(
		"Content-Type: text/plain;charset=UTF-8\r\n"+
			"Content-Transfer-Encoding: 7bit\r\n"+
			"\r\n%s", rawToken,
	)

	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE", block.Type)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	encryptedToken, err := pkcs7.Encrypt([]byte(smimeToken), []*x509.Certificate{cert})
	require.NoError(t, err)

	tokenBytes := fmt.Sprintf(
		"Content-Type: application/pkcs7-mime; name=\"smime.p7m\"; smime-type=enveloped-data\r\n"+
			"Content-Transfer-Encoding: base64\r\n"+
			"Content-Disposition: attachment; filename=\"smime.p7m\"\r\n"+
			"Content-Description: S/MIME Encrypted Message\r\n"+
			"\r\n%s", base64.StdEncoding.EncodeToString(encryptedToken))

	return certPEM, keyPEM, []byte(tokenBytes), nil
}

// GenerateTestCertBytes generates an RSA key and a self-signed certificate
// from the given template, returning PEM-encoded cert and key.
func GenerateTestCertBytes(template *x509.Certificate) ([]byte, []byte, error) {
	if template == nil {
		return nil, nil, errors.New("template is nil")
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return certPEM, keyPEM, nil
}

// NormalizeSQL normalizes the SQL statement by removing extra spaces and
// new lines, etc.
func NormalizeSQL(query string) string {
	query = strings.ToUpper(query)
	query = strings.TrimSpace(query)

	transformations := []struct {
		pattern     *regexp.Regexp
		replacement string
	}{
		{
			regexp.MustCompile(`(?m)--.*$|/\*(?s).*?\*/`),
			"",
		},
		{
			regexp.MustCompile(`\s+`),
			" ",
		},
		{
			regexp.MustCompile(`\s*,\s*`),
			",",
		},
		{
			regexp.MustCompile(`\s*\(\s*`),
			" (",
		},
		{
			regexp.MustCompile(`\s*\)\s*`),
			") ",
		},
	}
	for _, tx := range transformations {
		query = tx.pattern.ReplaceAllString(query, tx.replacement)
	}
	return query
}

// testingAuthorizer is a mock authorizer that allows all requests.
type testingAuthorizer struct{}

func (t *testingAuthorizer) Authorize(_ context.Context, _ platform_authz.AuthzTyper, _ platform_authz.Action) error {
	return nil
}

// testingLookupService adapts mysql.Datastore to fleet.ActivityLookupService
// interface for tests.
type testingLookupService struct {
	ds *mysql.Datastore
}

func (t *testingLookupService) ListUsers(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
	return t.ds.ListUsers(ctx, opt)
}

func (t *testingLookupService) UsersByIDs(ctx context.Context, ids []uint) ([]*fleet.UserSummary, error) {
	return t.ds.UsersByIDs(ctx, ids)
}

func (t *testingLookupService) GetHostLite(ctx context.Context, id uint) (*fleet.Host, error) {
	return t.ds.HostLite(ctx, id)
}

func (t *testingLookupService) GetActivitiesWebhookSettings(ctx context.Context) (fleet.ActivitiesWebhookSettings, error) {
	appConfig, err := t.ds.AppConfig(ctx)
	if err != nil {
		return fleet.ActivitiesWebhookSettings{}, err
	}
	return appConfig.WebhookSettings.ActivitiesWebhook, nil
}

func (t *testingLookupService) ActivateNextUpcomingActivityForHost(ctx context.Context, hostID uint, fromCompletedExecID string) error {
	return t.ds.ActivateNextUpcomingActivityForHost(ctx, hostID, fromCompletedExecID)
}

// TestDBConnections extracts the underlying DB connections from a test
// Datastore.
func TestDBConnections(t testing.TB, ds *mysql.Datastore) *common_mysql.DBConnections {
	t.Helper()
	replica, ok := ds.TestReplica().(*sqlx.DB)
	require.True(t, ok, "ds.replica should be *sqlx.DB in tests")
	return &common_mysql.DBConnections{Primary: ds.TestPrimaryDB(), Replica: replica}
}

// NewTestActivityService creates an activity service backed by the given
// datastore. Tests use this to call the activity bounded context API. User
// data is fetched from the same database to support tests that verify user
// info in activities.
func NewTestActivityService(t testing.TB, ds *mysql.Datastore) activity_api.Service {
	t.Helper()

	dbConns := TestDBConnections(t, ds)

	lookupSvc := &testingLookupService{ds: ds}
	aclAdapter := activityacl.NewFleetServiceAdapter(lookupSvc)

	discardLogger := slog.New(slog.DiscardHandler)
	svc, _ := activity_bootstrap.New(dbConns, &testingAuthorizer{}, aclAdapter, discardLogger)
	return svc
}

// ListActivitiesAPI calls the activity bounded context's ListActivities API.
func ListActivitiesAPI(t testing.TB, ctx context.Context, svc activity_api.Service, opts activity_api.ListOptions) []*activity_api.Activity {
	t.Helper()

	if opts.OrderKey == "" {
		opts.OrderKey = "id"
		opts.OrderDirection = activity_api.OrderAscending
	}
	if opts.PerPage == 0 {
		opts.PerPage = fleet.DefaultPerPage
	}

	activities, _, err := svc.ListActivities(ctx, opts)
	require.NoError(t, err)
	return activities
}
