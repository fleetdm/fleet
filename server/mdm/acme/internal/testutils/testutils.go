// Package testutils provides shared test utilities for the ACME service module.
package testutils

import (
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	mysql_testing_utils "github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// TestDB holds the database connection for tests.
type TestDB struct {
	DB     *sqlx.DB
	Logger *slog.Logger
}

// SetupTestDB creates a test database with the Fleet schema loaded.
func SetupTestDB(t *testing.T, testNamePrefix string) *TestDB {
	t.Helper()

	testName, opts := mysql_testing_utils.ProcessOptions(t, &mysql_testing_utils.DatastoreTestOptions{
		UniqueTestName: testNamePrefix + "_" + t.Name(),
	})

	mysql_testing_utils.LoadDefaultSchema(t, testName, opts)
	config := mysql_testing_utils.MysqlTestConfig(testName)
	db, err := common_mysql.NewDB(config, &common_mysql.DBOptions{}, "")
	require.NoError(t, err)

	t.Cleanup(func() { db.Close() })

	return &TestDB{
		DB:     db,
		Logger: slog.New(slog.DiscardHandler),
	}
}

// Conns returns DBConnections for creating a datastore.
func (tdb *TestDB) Conns() *common_mysql.DBConnections {
	return &common_mysql.DBConnections{Primary: tdb.DB, Replica: tdb.DB}
}

// TruncateTables clears the tables used by acme bounded context.
func (tdb *TestDB) TruncateTables(t *testing.T) {
	t.Helper()
	mysql_testing_utils.TruncateTables(t, tdb.DB, tdb.Logger, nil, "acme_enrollments", "acme_accounts", "acme_orders", "acme_authorizations", "acme_challenges", "identity_certificates", "identity_serials")
}

// InsertACMEEnrollment creates an enrollment in the database and updates the enrollment struct
// with the generated identifiers (if they were empty) and unique id.
func (tdb *TestDB) InsertACMEEnrollment(t *testing.T, enrollment *types.Enrollment) {
	t.Helper()
	ctx := t.Context()

	if enrollment.PathIdentifier == "" {
		enrollment.PathIdentifier = uuid.NewString()
	}
	if enrollment.HostIdentifier == "" {
		enrollment.HostIdentifier = uuid.NewString()
	}

	result, err := tdb.DB.ExecContext(ctx, `
		INSERT INTO acme_enrollments (path_identifier, host_identifier, not_valid_after, revoked)
		VALUES (?, ?, ?, ?)
	`, enrollment.PathIdentifier, enrollment.HostIdentifier, enrollment.NotValidAfter, enrollment.Revoked)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)
	enrollment.ID = uint(id) //nolint:gosec // dismiss G115
}
