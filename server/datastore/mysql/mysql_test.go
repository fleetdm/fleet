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
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/kit/log"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatastoreReplica(t *testing.T) {
	// a bit unfortunate to create temp databases just for this - could be mixed
	// with other tests when/if we move to subtests to minimize the number of
	// databases created for tests (see #1805).

	t.Run("noreplica", func(t *testing.T) {
		ds := CreateMySQLDSWithOptions(t, nil)
		defer ds.Close()
		require.Equal(t, ds.reader, ds.writer)
	})

	t.Run("replica", func(t *testing.T) {
		opts := &DatastoreTestOptions{Replica: true}
		ds := CreateMySQLDSWithOptions(t, opts)
		defer ds.Close()
		require.NotEqual(t, ds.reader, ds.writer)

		// create a new host
		host, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         "1",
			UUID:            "1",
			Hostname:        "foo.local",
			PrimaryIP:       "192.168.1.1",
			PrimaryMac:      "30-65-EC-6F-C4-58",
		})
		require.NoError(t, err)
		require.NotNil(t, host)

		// trying to read it fails, not replicated yet
		_, err = ds.Host(context.Background(), host.ID, false)
		require.Error(t, err)
		require.True(t, errors.Is(err, sql.ErrNoRows))

		opts.RunReplication()

		// now it can read it
		host2, err := ds.Host(context.Background(), host.ID, false)
		require.NoError(t, err)
		require.Equal(t, host.ID, host2.ID)
	})
}

func TestSanitizeColumn(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input  string
		output string
	}{
		{"foobar-column", "foobar-column"},
		{"foobar_column", "foobar_column"},
		{"foobar;column", "foobarcolumn"},
		{"foobar#", "foobar"},
		{"foobar*baz", "foobarbaz"},
	}

	for _, tt := range testCases {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.output, sanitizeColumn(tt.input))
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
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			sql, params := searchLike(tt.inSQL, tt.inParams, tt.match, tt.columns...)
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
		writer: dbmock,
		reader: dbmock,
		logger: log.NewNopLogger(),
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
		OrderKey: "name",
	}

	actual := appendListOptionsToSQL(sql, opts)
	expected := "SELECT * FROM my_table ORDER BY name ASC LIMIT 1000000"
	if actual != expected {
		t.Error("Expected", expected, "Actual", actual)
	}

	sql = "SELECT * FROM my_table"
	opts.OrderDirection = fleet.OrderDescending
	actual = appendListOptionsToSQL(sql, opts)
	expected = "SELECT * FROM my_table ORDER BY name DESC LIMIT 1000000"
	if actual != expected {
		t.Error("Expected", expected, "Actual", actual)
	}

	opts = fleet.ListOptions{
		PerPage: 10,
	}

	sql = "SELECT * FROM my_table"
	actual = appendListOptionsToSQL(sql, opts)
	expected = "SELECT * FROM my_table LIMIT 10"
	if actual != expected {
		t.Error("Expected", expected, "Actual", actual)
	}

	sql = "SELECT * FROM my_table"
	opts.Page = 2
	actual = appendListOptionsToSQL(sql, opts)
	expected = "SELECT * FROM my_table LIMIT 10 OFFSET 20"
	if actual != expected {
		t.Error("Expected", expected, "Actual", actual)
	}

	opts = fleet.ListOptions{}
	sql = "SELECT * FROM my_table"
	actual = appendListOptionsToSQL(sql, opts)
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
			t.Parallel()
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
			t.Parallel()
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
	defer func() { recover() }()

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
	defer func() { recover() }()

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
	require.Equal(t, "x509: certificate is not valid for any names, but wanted to match localhost", err.Error())
}

func TestWhereFilterTeas(t *testing.T) {
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
			t.Parallel()
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
