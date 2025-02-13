package mysql

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/VividCortex/mysqlerr"
	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatastoreReplica(t *testing.T) {
	// a bit unfortunate to create temp databases just for this - could be mixed
	// with other tests when/if we move to subtests to minimize the number of
	// databases created for tests (see #1805).

	ctx := context.Background()
	t.Run("noreplica", func(t *testing.T) {
		ds := CreateMySQLDSWithOptions(t, nil)
		defer ds.Close()
		require.Equal(t, ds.reader(ctx), ds.writer(ctx))
	})

	t.Run("replica", func(t *testing.T) {
		opts := &DatastoreTestOptions{DummyReplica: true}
		ds := CreateMySQLDSWithOptions(t, opts)
		defer ds.Close()
		require.NotEqual(t, ds.reader(ctx), ds.writer(ctx))

		// create a new host
		host, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         ptr.String("1"),
			UUID:            "1",
			Hostname:        "foo.local",
			PrimaryIP:       "192.168.1.1",
			PrimaryMac:      "30-65-EC-6F-C4-58",
		})
		require.NoError(t, err)
		require.NotNil(t, host)

		// trying to read it fails, not replicated yet
		_, err = ds.Host(ctx, host.ID)
		require.Error(t, err)
		require.True(t, errors.Is(err, sql.ErrNoRows), err)

		// force read from primary works
		ctx = ctxdb.RequirePrimary(ctx, true)
		got, err := ds.Host(ctx, host.ID)
		require.NoError(t, err)
		require.Equal(t, host.ID, got.ID)

		// but from replica still fails
		ctx = ctxdb.RequirePrimary(ctx, false)
		_, err = ds.Host(ctx, host.ID)
		require.Error(t, err)
		require.True(t, errors.Is(err, sql.ErrNoRows))

		opts.RunReplication()

		// now it can read it from replica
		got, err = ds.Host(ctx, host.ID)
		require.NoError(t, err)
		require.Equal(t, host.ID, got.ID)
	})
}

func TestSanitizeColumn(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input  string
		output string
	}{
		{"", ""},
		{"foobar-column", "`foobar-column`"},
		{"foobar_column", "`foobar_column`"},
		{"foobar;column", "`foobarcolumn`"},
		{"foobar#", "`foobar`"},
		{"foobar*baz", "`foobarbaz`"},
		{"....", ""},
		{"h.id", "`h`.`id`"},
		{"id;delete from hosts", "`iddeletefromhosts`"},
		{"select * from foo", "`selectfromfoo`"},
	}

	for _, tt := range testCases {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.output, sanitizeColumn(tt.input))
		})
	}
}

func TestSearchLike(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		inSQL     string
		inParams  []interface{}
		match     string
		columns   []string
		outSQL    string
		outParams []interface{}
	}{
		{
			inSQL:     "SELECT * FROM HOSTS WHERE TRUE",
			inParams:  []interface{}{},
			match:     "foobar",
			columns:   []string{"hostname"},
			outSQL:    "SELECT * FROM HOSTS WHERE TRUE AND (hostname LIKE ?)",
			outParams: []interface{}{"%foobar%"},
		},
		{
			inSQL:     "SELECT * FROM HOSTS WHERE TRUE",
			inParams:  []interface{}{3},
			match:     "foobar",
			columns:   []string{},
			outSQL:    "SELECT * FROM HOSTS WHERE TRUE",
			outParams: []interface{}{3},
		},
		{
			inSQL:     "SELECT * FROM HOSTS WHERE TRUE",
			inParams:  []interface{}{1},
			match:     "foobar",
			columns:   []string{"hostname"},
			outSQL:    "SELECT * FROM HOSTS WHERE TRUE AND (hostname LIKE ?)",
			outParams: []interface{}{1, "%foobar%"},
		},
		{
			inSQL:     "SELECT * FROM HOSTS WHERE TRUE",
			inParams:  []interface{}{1},
			match:     "foobar",
			columns:   []string{"hostname", "uuid"},
			outSQL:    "SELECT * FROM HOSTS WHERE TRUE AND (hostname LIKE ? OR uuid LIKE ?)",
			outParams: []interface{}{1, "%foobar%", "%foobar%"},
		},
		{
			inSQL:     "SELECT * FROM HOSTS WHERE TRUE",
			inParams:  []interface{}{1},
			match:     "foobar",
			columns:   []string{"hostname", "uuid"},
			outSQL:    "SELECT * FROM HOSTS WHERE TRUE AND (hostname LIKE ? OR uuid LIKE ?)",
			outParams: []interface{}{1, "%foobar%", "%foobar%"},
		},
		{
			inSQL:     "SELECT * FROM HOSTS WHERE 1=1",
			inParams:  []interface{}{1},
			match:     "forty_%",
			columns:   []string{"ipv4", "uuid"},
			outSQL:    "SELECT * FROM HOSTS WHERE 1=1 AND (ipv4 LIKE ? OR uuid LIKE ?)",
			outParams: []interface{}{1, "%forty\\_\\%%", "%forty\\_\\%%"},
		},
		{
			inSQL:     "SELECT * FROM HOSTS WHERE 1=1",
			inParams:  []interface{}{1},
			match:     "forty_%",
			columns:   []string{"ipv4", "uuid"},
			outSQL:    "SELECT * FROM HOSTS WHERE 1=1 AND (ipv4 LIKE ? OR uuid LIKE ?)",
			outParams: []interface{}{1, "%forty\\_\\%%", "%forty\\_\\%%"},
		},
		{
			inSQL:     "SELECT * FROM HOSTS WHERE 1=1",
			inParams:  []interface{}{1},
			match:     "a@b.c",
			columns:   []string{"ipv4", "uuid"},
			outSQL:    "SELECT * FROM HOSTS WHERE 1=1 AND (ipv4 LIKE ? OR uuid LIKE ?)",
			outParams: []interface{}{1, "%a@b.c%", "%a@b.c%"},
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			sql, params := searchLike(tt.inSQL, tt.inParams, tt.match, tt.columns...)
			assert.Equal(t, tt.outSQL, sql)
			assert.Equal(t, tt.outParams, params)
		})
	}
}

func TestHostSearchLike(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		inSQL     string
		inParams  []interface{}
		match     string
		columns   []string
		outSQL    string
		outParams []interface{}
	}{
		{
			inSQL:     "SELECT * FROM HOSTS h WHERE TRUE",
			inParams:  []interface{}{},
			match:     "foobar",
			columns:   []string{"hostname"},
			outSQL:    "SELECT * FROM HOSTS h WHERE TRUE AND (hostname LIKE ?)",
			outParams: []interface{}{"%foobar%"},
		},
		{
			inSQL:     "SELECT * FROM HOSTS h WHERE 1=1",
			inParams:  []interface{}{1},
			match:     "a@b.c",
			columns:   []string{"ipv4"},
			outSQL:    "SELECT * FROM HOSTS h WHERE 1=1 AND (ipv4 LIKE ? OR ( EXISTS (SELECT 1 FROM host_emails he WHERE he.host_id = h.id AND he.email LIKE ?)))",
			outParams: []interface{}{1, "%a@b.c%", "%a@b.c%"},
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			sql, params, _ := hostSearchLike(tt.inSQL, tt.inParams, tt.match, tt.columns...)
			assert.Equal(t, tt.outSQL, sql)
			assert.Equal(t, tt.outParams, params)
		})
	}
}

func mockDatastore(t *testing.T) (sqlmock.Sqlmock, *Datastore) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	dbmock := sqlx.NewDb(db, "sqlmock")
	ds := &Datastore{
		primary: dbmock,
		replica: dbmock,
		logger:  log.NewNopLogger(),
	}

	return mock, ds
}

func TestWithRetryTxxSuccess(t *testing.T) {
	mock, ds := mockDatastore(t)
	defer ds.Close()

	mock.ExpectBegin()
	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	require.NoError(t, ds.withRetryTxx(context.Background(), func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(context.Background(), "SELECT 1")
		return err
	}))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithRetryTxxRollbackSuccess(t *testing.T) {
	mock, ds := mockDatastore(t)
	defer ds.Close()

	mock.ExpectBegin()
	mock.ExpectExec("SELECT 1").WillReturnError(errors.New("fail"))
	mock.ExpectRollback()

	require.Error(t, ds.withRetryTxx(context.Background(), func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(context.Background(), "SELECT 1")
		return err
	}))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithRetryTxxRollbackError(t *testing.T) {
	mock, ds := mockDatastore(t)
	defer ds.Close()

	mock.ExpectBegin()
	mock.ExpectExec("SELECT 1").WillReturnError(errors.New("fail"))
	mock.ExpectRollback().WillReturnError(errors.New("rollback failed"))

	require.Error(t, ds.withRetryTxx(context.Background(), func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(context.Background(), "SELECT 1")
		return err
	}))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithRetryTxxRetrySuccess(t *testing.T) {
	mock, ds := mockDatastore(t)
	defer ds.Close()

	mock.ExpectBegin()
	// Return a retryable error
	mock.ExpectExec("SELECT 1").WillReturnError(&mysql.MySQLError{Number: mysqlerr.ER_LOCK_DEADLOCK})
	mock.ExpectRollback()
	mock.ExpectBegin()
	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	assert.NoError(t, ds.withRetryTxx(context.Background(), func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(context.Background(), "SELECT 1")
		return err
	}))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithRetryTxxCommitRetrySuccess(t *testing.T) {
	mock, ds := mockDatastore(t)
	defer ds.Close()

	mock.ExpectBegin()
	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(1, 1))
	// Return a retryable error
	mock.ExpectCommit().WillReturnError(&mysql.MySQLError{Number: mysqlerr.ER_LOCK_DEADLOCK})
	mock.ExpectBegin()
	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	assert.NoError(t, ds.withRetryTxx(context.Background(), func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(context.Background(), "SELECT 1")
		return err
	}))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithRetryTxxCommitError(t *testing.T) {
	mock, ds := mockDatastore(t)
	defer ds.Close()

	mock.ExpectBegin()
	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(1, 1))
	// Return a retryable error
	mock.ExpectCommit().WillReturnError(errors.New("fail"))

	assert.Error(t, ds.withRetryTxx(context.Background(), func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(context.Background(), "SELECT 1")
		return err
	}))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAppendListOptionsToSQL(t *testing.T) {
	sql := "SELECT * FROM my_table"
	opts := fleet.ListOptions{
		OrderKey: "***name***",
	}

	actual, _ := appendListOptionsToSQL(sql, &opts)
	expected := "SELECT * FROM my_table ORDER BY `name` ASC LIMIT 1000000"
	if actual != expected {
		t.Error("Expected", expected, "Actual", actual)
	}

	sql = "SELECT * FROM my_table"
	opts.OrderDirection = fleet.OrderDescending
	actual, _ = appendListOptionsToSQL(sql, &opts)
	expected = "SELECT * FROM my_table ORDER BY `name` DESC LIMIT 1000000"
	if actual != expected {
		t.Error("Expected", expected, "Actual", actual)
	}

	opts = fleet.ListOptions{
		PerPage: 10,
	}

	sql = "SELECT * FROM my_table"
	actual, _ = appendListOptionsToSQL(sql, &opts)
	expected = "SELECT * FROM my_table LIMIT 10"
	if actual != expected {
		t.Error("Expected", expected, "Actual", actual)
	}

	sql = "SELECT * FROM my_table"
	opts.Page = 2
	actual, _ = appendListOptionsToSQL(sql, &opts)
	expected = "SELECT * FROM my_table LIMIT 10 OFFSET 20"
	if actual != expected {
		t.Error("Expected", expected, "Actual", actual)
	}

	opts = fleet.ListOptions{}
	sql = "SELECT * FROM my_table"
	actual, _ = appendListOptionsToSQL(sql, &opts)
	expected = "SELECT * FROM my_table LIMIT 1000000"

	if actual != expected {
		t.Error("Expected", expected, "Actual", actual)
	}
}

func TestWhereFilterHostsByTeams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		filter   fleet.TeamFilter
		expected string
	}{
		// No teams or global role
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{},
			},
			expected: "FALSE",
		},
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{Teams: []fleet.UserTeam{}},
			},
			expected: "FALSE",
		},

		// Global role
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			},
			expected: "TRUE",
		},
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			},
			expected: "TRUE",
		},
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			},
			expected: "FALSE",
		},
		{
			filter: fleet.TeamFilter{
				User:            &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
				IncludeObserver: true,
			},
			expected: "TRUE",
		},

		// Team roles
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
					},
				},
			},
			expected: "FALSE",
		},
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
					},
				},
				IncludeObserver: true,
			},
			expected: "hosts.team_id IN (1)",
		},
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 2}},
					},
				},
			},
			expected: "FALSE",
		},
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
						{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 2}},
					},
				},
			},
			expected: "hosts.team_id IN (2)",
		},
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
						{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 2}},
					},
				},
				IncludeObserver: true,
			},
			expected: "hosts.team_id IN (1,2)",
		},
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
						{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 2}},
						// Invalid role should be ignored
						{Role: "bad", Team: fleet.Team{ID: 37}},
					},
				},
			},
			expected: "hosts.team_id IN (2)",
		},
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
						{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 2}},
						{Role: fleet.RoleAdmin, Team: fleet.Team{ID: 3}},
						// Invalid role should be ignored
					},
				},
			},
			expected: "hosts.team_id IN (2,3)",
		},
		{
			filter: fleet.TeamFilter{
				TeamID: ptr.Uint(1),
			},
			expected: "FALSE",
		},
		{
			filter: fleet.TeamFilter{
				User:            &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
				IncludeObserver: true,
				TeamID:          ptr.Uint(1),
			},
			expected: "hosts.team_id = 1",
		},
		{
			filter: fleet.TeamFilter{
				User:            &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
				IncludeObserver: false,
				TeamID:          ptr.Uint(1),
			},
			expected: "FALSE",
		},
		{
			filter: fleet.TeamFilter{
				User:            &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
				IncludeObserver: false,
				TeamID:          ptr.Uint(1),
			},
			expected: "hosts.team_id = 1",
		},
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
						{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 2}},
					},
				},
				TeamID: ptr.Uint(3),
			},
			expected: "FALSE",
		},
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
						{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 2}},
					},
				},
				TeamID: ptr.Uint(2),
			},
			expected: "hosts.team_id = 2",
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run("", func(t *testing.T) {
			ds := &Datastore{logger: log.NewNopLogger()}
			sql := ds.whereFilterHostsByTeams(tt.filter, "hosts")
			assert.Equal(t, tt.expected, sql)
		})
	}
}

func TestWhereOmitIDs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		omits    []uint
		expected string
	}{
		{
			omits:    nil,
			expected: "TRUE",
		},
		{
			omits:    []uint{},
			expected: "TRUE",
		},
		{
			omits:    []uint{1, 3, 4},
			expected: "id NOT IN (1,3,4)",
		},
		{
			omits:    []uint{42},
			expected: "id NOT IN (42)",
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run("", func(t *testing.T) {
			ds := &Datastore{logger: log.NewNopLogger()}
			sql := ds.whereOmitIDs("id", tt.omits)
			assert.Equal(t, tt.expected, sql)
		})
	}
}

func TestWithRetryTxWithRollback(t *testing.T) {
	mock, ds := mockDatastore(t)
	defer ds.Close()

	mock.ExpectBegin()
	mock.ExpectExec("SELECT 1").WillReturnError(errors.New("let's rollback!"))
	mock.ExpectRollback()

	assert.Error(t, ds.withRetryTxx(context.Background(), func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(context.Background(), "SELECT 1")
		return err
	}))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithRetryTxWillRollbackWhenPanic(t *testing.T) {
	mock, ds := mockDatastore(t)
	defer ds.Close()
	defer func() { recover() }() //nolint:errcheck

	mock.ExpectBegin()
	mock.ExpectExec("SELECT 1").WillReturnError(errors.New("let's rollback!"))
	mock.ExpectRollback()

	assert.Error(t, ds.withRetryTxx(context.Background(), func(tx sqlx.ExtContext) error {
		panic("ROLLBACK")
	}))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithTxWithRollback(t *testing.T) {
	mock, ds := mockDatastore(t)
	defer ds.Close()

	mock.ExpectBegin()
	mock.ExpectExec("SELECT 1").WillReturnError(errors.New("let's rollback!"))
	mock.ExpectRollback()

	assert.Error(t, ds.withTx(context.Background(), func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(context.Background(), "SELECT 1")
		return err
	}))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithTxWillRollbackWhenPanic(t *testing.T) {
	mock, ds := mockDatastore(t)
	defer ds.Close()
	defer func() { recover() }() //nolint:errcheck

	mock.ExpectBegin()
	mock.ExpectExec("SELECT 1").WillReturnError(errors.New("let's rollback!"))
	mock.ExpectRollback()

	assert.Error(t, ds.withTx(context.Background(), func(tx sqlx.ExtContext) error {
		panic("ROLLBACK")
	}))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNewReadsPasswordFromDisk(t *testing.T) {
	passwordFile, err := os.CreateTemp(t.TempDir(), "*.passwordtest")
	require.NoError(t, err)
	_, err = passwordFile.WriteString(testPassword)
	require.NoError(t, err)
	passwordPath := passwordFile.Name()
	require.NoError(t, passwordFile.Close())

	dbName := t.Name()

	// Create a datastore client in order to run migrations as usual
	mysqlConfig := config.MysqlConfig{
		Username:     testUsername,
		Password:     "",
		PasswordPath: passwordPath,
		Address:      testAddress,
		Database:     dbName,
	}
	ds, err := newDSWithConfig(t, dbName, mysqlConfig)
	require.NoError(t, err)
	defer ds.Close()
	require.NoError(t, ds.HealthCheck())
}

func newDSWithConfig(t *testing.T, dbName string, config config.MysqlConfig) (*Datastore, error) {
	db, err := sql.Open(
		"mysql",
		fmt.Sprintf("%s:%s@tcp(%s)/?multiStatements=true", testUsername, testPassword, testAddress),
	)
	require.NoError(t, err)
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s; CREATE DATABASE %s;", dbName, dbName))
	require.NoError(t, err)

	ds, err := New(config, clock.NewMockClock(), Logger(log.NewNopLogger()), LimitAttempts(1))
	return ds, err
}

func generateTestCert(t *testing.T) (string, string) {
	privateKeyCA, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	require.NoError(t, err)
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"aa"},
		},
		NotBefore:             time.Now().Add(-1 * time.Duration(24) * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKeyCA.PublicKey, privateKeyCA)
	require.NoError(t, err)

	publicPem, err := os.CreateTemp(t.TempDir(), "*-ca.pem")
	require.NoError(t, err)
	require.NoError(t, pem.Encode(publicPem, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}))
	require.NoError(t, publicPem.Close())

	keyPem, err := os.CreateTemp(t.TempDir(), "*-key.pem")
	require.NoError(t, err)
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKeyCA)
	require.NoError(t, pem.Encode(keyPem, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes}))
	require.NoError(t, keyPem.Close())

	return publicPem.Name(), keyPem.Name()
}

func TestNewUsesRegisterTLS(t *testing.T) {
	dbName := t.Name()

	ca, _ := generateTestCert(t)
	cert, key := generateTestCert(t)

	mysqlConfig := config.MysqlConfig{
		Username: testUsername,
		Password: testPassword,
		Address:  testAddress,
		Database: dbName,
		TLSCA:    ca,
		TLSCert:  cert,
		TLSKey:   key,
	}
	// This fails because the certificate mysql is using is different than the one generated here
	_, err := newDSWithConfig(t, dbName, mysqlConfig)
	require.Error(t, err)
	// TODO: we're using a Regexp because the message is different depending on the version of mysql,
	// we should refactor and use different error types instead.
	require.Regexp(t, "(x509|tls|EOF)", err.Error())
}

func TestWhereFilterTeams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		filter   fleet.TeamFilter
		expected string
	}{
		// No teams or global role
		{
			filter:   fleet.TeamFilter{User: nil},
			expected: "FALSE",
		},
		{
			filter: fleet.TeamFilter{
				User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			},
			expected: "TRUE",
		},
		{
			filter: fleet.TeamFilter{
				User:            &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
				IncludeObserver: false,
			},
			expected: "FALSE",
		},
		{
			filter: fleet.TeamFilter{
				User:            &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
				IncludeObserver: true,
			},
			expected: "TRUE",
		},
		{
			filter:   fleet.TeamFilter{User: &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}},
			expected: "t.id IN (1)",
		},
		{
			filter:   fleet.TeamFilter{User: &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}}},
			expected: "t.id IN (1)",
		},
		{
			filter:   fleet.TeamFilter{User: &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}}},
			expected: "FALSE",
		},
		{
			filter: fleet.TeamFilter{
				User:            &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
				IncludeObserver: true,
			},
			expected: "t.id IN (1)",
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run("", func(t *testing.T) {
			ds := &Datastore{logger: log.NewNopLogger()}
			sql := ds.whereFilterTeams(tt.filter, "t")
			assert.Equal(t, tt.expected, sql)
		})
	}
}

func TestCompareVersions(t *testing.T) {
	for _, tc := range []struct {
		name string

		v1            []int64
		v2            []int64
		knownUnknowns map[int64]struct{}

		expMissing []int64
		expUnknown []int64
		expEqual   bool
	}{
		{
			name:     "both-empty",
			v1:       nil,
			v2:       nil,
			expEqual: true,
		},
		{
			name:     "equal",
			v1:       []int64{1, 2, 3},
			v2:       []int64{1, 2, 3},
			expEqual: true,
		},
		{
			name:     "equal-out-of-order",
			v1:       []int64{1, 2, 3},
			v2:       []int64{1, 3, 2},
			expEqual: true,
		},
		{
			name:       "empty-with-unknown",
			v1:         nil,
			v2:         []int64{1},
			expEqual:   false,
			expUnknown: []int64{1},
		},
		{
			name:       "empty-with-missing",
			v1:         []int64{1},
			v2:         nil,
			expEqual:   false,
			expMissing: []int64{1},
		},
		{
			name:       "missing",
			v1:         []int64{1, 2, 3},
			v2:         []int64{1, 3},
			expMissing: []int64{2},
			expEqual:   false,
		},
		{
			name:       "unknown",
			v1:         []int64{1, 2, 3},
			v2:         []int64{1, 2, 3, 4},
			expUnknown: []int64{4},
			expEqual:   false,
		},
		{
			name: "known-unknown",
			v1:   []int64{1, 2, 3},
			v2:   []int64{1, 2, 3, 4},
			knownUnknowns: map[int64]struct{}{
				4: {},
			},
			expEqual: true,
		},
		{
			name:       "unknowns",
			v1:         []int64{1, 2, 3},
			v2:         []int64{1, 2, 3, 4, 5},
			expUnknown: []int64{5},
			knownUnknowns: map[int64]struct{}{
				4: {},
			},
			expEqual: false,
		},
		{
			name:       "missing-and-unknown",
			v1:         []int64{1, 2, 3},
			v2:         []int64{1, 2, 4},
			expMissing: []int64{3},
			expUnknown: []int64{4},
			expEqual:   false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			missing, unknown, equal := compareVersions(tc.v1, tc.v2, tc.knownUnknowns)
			require.Equal(t, tc.expMissing, missing)
			require.Equal(t, tc.expUnknown, unknown)
			require.Equal(t, tc.expEqual, equal)
		})
	}
}

func TestDebugs(t *testing.T) {
	ds := CreateMySQLDS(t)

	status, err := ds.InnoDBStatus(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, status)

	processList, err := ds.ProcessList(context.Background())
	require.NoError(t, err)
	require.Greater(t, len(processList), 0)
}

func TestWantedModesEnabled(t *testing.T) {
	ds := CreateMySQLDS(t)

	var sqlMode string
	err := ds.writer(context.Background()).GetContext(context.Background(), &sqlMode, `SELECT @@SQL_MODE`)
	require.NoError(t, err)
	require.Contains(t, sqlMode, "ANSI_QUOTES")
	require.Contains(t, sqlMode, "ONLY_FULL_GROUP_BY")
}

func Test_buildWildcardMatchPhrase(t *testing.T) {
	type args struct {
		matchQuery string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "",
			args: args{matchQuery: "test"},
			want: "%test%",
		},
		{
			name: "underscores are escaped",
			args: args{matchQuery: "Host_1"},
			want: "%Host\\_1%",
		},
		{
			name: "percent are escaped",
			args: args{matchQuery: "Host%1"},
			want: "%Host\\%1%",
		},
		{
			name: "percent & underscore are escaped",
			args: args{matchQuery: "Host_%1"},
			want: "%Host\\_\\%1%",
		},
		{
			name: "underscores added for wildcard search are not escaped",
			args: args{matchQuery: "Alice‘s MacbookPro"},
			want: "%Alice_s MacbookPro%",
		},
		{
			name: "underscores added for wildcard search are not escaped, but underscores in matchQuery are",
			args: args{matchQuery: "Alice‘s Macbook_Pro"},
			want: "%Alice_s Macbook\\_Pro%",
		},
		{
			name: "multiple occurances of wildcard are not escaped",
			args: args{matchQuery: "Alice‘‘s Macbook_Pro"},
			want: "%Alice__s Macbook\\_Pro%",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, buildWildcardMatchPhrase(tt.args.matchQuery), "buildWildcardMatchPhrase(%v)", tt.args.matchQuery)
		})
	}
}

func TestWhereFilterGlobalOrTeamIDByTeams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		filter   fleet.TeamFilter
		expected string
	}{
		// No teams or global role
		{
			name: "empty user",
			filter: fleet.TeamFilter{
				User: &fleet.User{},
			},
			expected: "FALSE",
		},
		{
			name: "empty user teams",
			filter: fleet.TeamFilter{
				User: &fleet.User{Teams: []fleet.UserTeam{}},
			},
			expected: "FALSE",
		},

		// Global role
		{
			name: "global admin",
			filter: fleet.TeamFilter{
				User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			},
			expected: "hosts.team_id = 0 AND hosts.global_stats = 1",
		},
		{
			name: "global maintainer",
			filter: fleet.TeamFilter{
				User: &fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			},
			expected: "hosts.team_id = 0 AND hosts.global_stats = 1",
		},
		{
			name: "global observer",
			filter: fleet.TeamFilter{
				User: &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			},
			expected: "FALSE",
		},
		{
			name: "global observer include",
			filter: fleet.TeamFilter{
				User:            &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
				IncludeObserver: true,
			},
			expected: "hosts.team_id = 0 AND hosts.global_stats = 1",
		},

		// Team roles
		{
			name: "team observer",
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
					},
				},
			},
			expected: "FALSE",
		},
		{
			name: "team observer include",
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
					},
				},
				IncludeObserver: true,
			},
			expected: "hosts.team_id IN (1)",
		},
		{
			name: "multi team observer",
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 2}},
					},
				},
			},
			expected: "FALSE",
		},
		{
			name: "multi team maintainer and observer",
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
						{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 2}},
					},
				},
			},
			expected: "hosts.team_id IN (2)",
		},
		{
			name: "multi team maintainer and observer include",
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
						{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 2}},
					},
				},
				IncludeObserver: true,
			},
			expected: "hosts.team_id IN (1,2)",
		},
		{
			name: "multi team maintainer and observer with invalid role",
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
						{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 2}},
						// Invalid role should be ignored
						{Role: "bad", Team: fleet.Team{ID: 37}},
					},
				},
			},
			expected: "hosts.team_id IN (2)",
		},
		{
			name: "multi team maintainer and observer and admin",
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
						{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 2}},
						{Role: fleet.RoleAdmin, Team: fleet.Team{ID: 3}},
						// Invalid role should be ignored
					},
				},
			},
			expected: "hosts.team_id IN (2,3)",
		},
		{
			name: "team id only",
			filter: fleet.TeamFilter{
				TeamID: ptr.Uint(1),
			},
			expected: "FALSE",
		},
		{
			name: "team id with observer include",
			filter: fleet.TeamFilter{
				User:            &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
				IncludeObserver: true,
				TeamID:          ptr.Uint(1),
			},
			expected: "hosts.team_id = 1",
		},
		{
			name: "team id with observer exclude",
			filter: fleet.TeamFilter{
				User:            &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
				IncludeObserver: false,
				TeamID:          ptr.Uint(1),
			},
			expected: "FALSE",
		},
		{
			name: "team id with admin exclude observer",
			filter: fleet.TeamFilter{
				User:            &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
				IncludeObserver: false,
				TeamID:          ptr.Uint(1),
			},
			expected: "hosts.team_id = 1",
		},
		{
			name: "team id not in multiple team roles",
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
						{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 2}},
					},
				},
				TeamID: ptr.Uint(3),
			},
			expected: "FALSE",
		},
		{
			name: "team id in multiple team roles",
			filter: fleet.TeamFilter{
				User: &fleet.User{
					Teams: []fleet.UserTeam{
						{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
						{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 2}},
					},
				},
				TeamID: ptr.Uint(2),
			},
			expected: "hosts.team_id = 2",
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ds := &Datastore{logger: log.NewNopLogger()}
			sql := ds.whereFilterGlobalOrTeamIDByTeams(tt.filter, "hosts")
			assert.Equal(t, tt.expected, sql)
		})
	}
}

func TestBatchProcessDB(t *testing.T) {
	type testData struct {
		id    int
		value string
	}

	payload := []interface{}{
		&testData{id: 1, value: "a"},
		&testData{id: 2, value: "b"},
		&testData{id: 3, value: "c"},
	}

	generateValueArgs := func(item interface{}) (string, []any) {
		p := item.(*testData)
		valuePart := "(?, ?),"
		args := []any{p.id, p.value}
		return valuePart, args
	}

	t.Run("TestEmptyPayload", func(t *testing.T) {
		executeBatch := func(valuePart string, args []any) error {
			return errors.New("execute shouldn't be called for an empty payload")
		}
		err := batchProcessDB([]interface{}{}, 1000, generateValueArgs, executeBatch)
		require.NoError(t, err)
	})

	t.Run("TestSingleBatch", func(t *testing.T) {
		callCount := 0
		executeBatch := func(valuePart string, args []any) error {
			callCount++
			require.Equal(t, 2, len(args)/2) // each item adds 2 args
			return nil
		}
		err := batchProcessDB(payload[:2], 2, generateValueArgs, executeBatch)
		require.NoError(t, err)
		require.Equal(t, 1, callCount)
	})

	t.Run("TestMultipleBatches", func(t *testing.T) {
		callCount := 0
		executeBatch := func(valuePart string, args []any) error {
			callCount++
			require.Equal(t, 2/callCount, len(args)/2) // each item adds 2 args
			return nil
		}
		err := batchProcessDB(payload, 2, generateValueArgs, executeBatch)
		require.NoError(t, err)
		require.Equal(t, 2, callCount)
	})
}

func TestGetContextTryStmt(t *testing.T) {
	ctx := context.Background()

	dbMock, ds := mockDatastore(t)
	ds.stmtCache = map[string]*sqlx.Stmt{}

	t.Run("get with unknown statement error", func(t *testing.T) {
		count := 0
		query := "SELECT 1"

		// first call to cache the statement
		dbMock.ExpectPrepare(query)
		mockResult := sqlmock.NewRows([]string{query})
		mockResult.AddRow("1")
		dbMock.ExpectQuery(query).WillReturnRows(mockResult)
		err := ds.getContextTryStmt(ctx, &count, query)
		require.NoError(t, err)
		require.NoError(t, dbMock.ExpectationsWereMet())

		// verify that the statement was cached
		stmt := ds.loadOrPrepareStmt(ctx, query)
		require.NotNil(t, stmt)

		// call again to trigger the unknown statement error and ensure it retries
		// first query, make it fail
		queryMock := dbMock.ExpectQuery(query)
		mySQLErr := &mysql.MySQLError{
			Number: mysqlerr.ER_UNKNOWN_STMT_HANDLER,
		}
		queryMock.WillReturnError(mySQLErr)

		// after the failure, a second call is made, this time without
		// the prepared statement
		mockResult = sqlmock.NewRows([]string{query})
		mockResult.AddRow("1")
		dbMock.ExpectQuery(query).WillReturnRows(mockResult)

		// make the call and verify we removed the prepared statement
		err = ds.getContextTryStmt(ctx, &count, query)
		require.NoError(t, err)
		require.NoError(t, dbMock.ExpectationsWereMet())
		stmt = ds.loadOrPrepareStmt(ctx, query)
		require.Nil(t, stmt)
	})

	t.Run("get with other error", func(t *testing.T) {
		dbMock, ds := mockDatastore(t)
		ds.stmtCache = map[string]*sqlx.Stmt{}
		count := 0
		query := "SELECT 1"

		// first call to cache the statement
		dbMock.ExpectPrepare(query)
		mockResult := sqlmock.NewRows([]string{query})
		mockResult.AddRow("1")
		dbMock.ExpectQuery(query).WillReturnRows(mockResult)
		err := ds.getContextTryStmt(ctx, &count, query)
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.NoError(t, dbMock.ExpectationsWereMet())

		// verify that the statement was cached
		stmt := ds.loadOrPrepareStmt(ctx, query)
		require.NotNil(t, stmt)

		// return a duplicate error
		queryMock := dbMock.ExpectQuery(query)
		mySQLErr := &mysql.MySQLError{
			Number: mysqlerr.ER_DUP_ENTRY,
		}
		queryMock.WillReturnError(mySQLErr)

		count = 0
		err = ds.getContextTryStmt(ctx, &count, query)
		require.ErrorIs(t, mySQLErr, err)
		require.NoError(t, dbMock.ExpectationsWereMet())
		stmt = ds.loadOrPrepareStmt(ctx, query)
		require.NotNil(t, stmt)
	})

}
