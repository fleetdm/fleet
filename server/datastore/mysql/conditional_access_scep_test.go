package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConditionalAccessSCEP(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"GetCertBySerialAndCreatedAt", testGetConditionalAccessCertBySerialAndCreatedAt},
		{"RevokedCertsNotReturned", testRevokedConditionalAccessCertsNotReturned},
		{"ExpiredCertsNotReturned", testExpiredConditionalAccessCertsNotReturned},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testGetConditionalAccessCertBySerialAndCreatedAt(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create a test host
	host, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("test-host-1"),
		NodeKey:         ptr.String("test-node-key-1"),
		UUID:            "test-uuid-1",
		Hostname:        "test-hostname-1",
		Platform:        "darwin",
		DetailUpdatedAt: time.Now(),
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	// Insert a valid test certificate
	now := time.Now()
	serialNumber := insertConditionalAccessCert(t, ds, ctx, host.ID, "test-cn", now.Add(-24*time.Hour), now.Add(365*24*time.Hour), false)

	// Test retrieval by serial number
	hostID, err := ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, serialNumber)
	require.NoError(t, err)
	assert.Equal(t, host.ID, hostID)

	// Test retrieval of created_at by host ID
	createdAt, err := ds.GetConditionalAccessCertCreatedAtByHostID(ctx, host.ID)
	require.NoError(t, err)
	require.NotNil(t, createdAt)
	// Verify timestamp is reasonable (created in the past, within last 24 hours)
	assert.True(t, createdAt.Before(time.Now()))
	assert.True(t, createdAt.After(time.Now().Add(-24*time.Hour)))

	// Test non-existent serial
	_, err = ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, 999)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	// Test non-existent host
	_, err = ds.GetConditionalAccessCertCreatedAtByHostID(ctx, 999999)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func testRevokedConditionalAccessCertsNotReturned(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create a test host
	host, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("test-host-6"),
		NodeKey:         ptr.String("test-node-key-6"),
		UUID:            "test-uuid-6",
		Hostname:        "test-hostname-6",
		Platform:        "darwin",
		DetailUpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	// Insert a revoked certificate
	now := time.Now()
	serialNumber := insertConditionalAccessCert(t, ds, ctx, host.ID, "revoked-cert", now.Add(-24*time.Hour), now.Add(365*24*time.Hour), true)

	// Revoked certs should not be returned by serial number lookup
	_, err = ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, serialNumber)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	// Note: GetConditionalAccessCertCreatedAtByHostID doesn't filter by revoked status
	// since it's used for rate limiting checks, not authentication
}

func testExpiredConditionalAccessCertsNotReturned(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create a test host
	host, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("test-host-7"),
		NodeKey:         ptr.String("test-node-key-7"),
		UUID:            "test-uuid-7",
		Hostname:        "test-hostname-7",
		Platform:        "darwin",
		DetailUpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	// Insert an expired certificate
	now := time.Now()
	serialNumber := insertConditionalAccessCert(t, ds, ctx, host.ID, "expired-cert", now.Add(-400*24*time.Hour), now.Add(-24*time.Hour), false)

	// Expired certs should not be returned by serial number lookup
	_, err = ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, serialNumber)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	// Note: GetConditionalAccessCertCreatedAtByHostID doesn't filter by expiration status
	// since it's used for rate limiting checks, not authentication
}

// insertConditionalAccessCert inserts a conditional access SCEP certificate for testing.
// Returns the serial number of the inserted certificate.
func insertConditionalAccessCert(t *testing.T, ds *Datastore, ctx context.Context, hostID uint, name string, notValidBefore, notValidAfter time.Time, revoked bool) uint64 {
	t.Helper()

	certPEM := `-----BEGIN CERTIFICATE-----
MIICEjCCAXsCAg36MA0GCSqGSIb3DQEBBQUAMIGbMQswCQYDVQQGEwJKUDEOMAwG
-----END CERTIFICATE-----`

	var serialNumber uint64
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		result, err := q.ExecContext(ctx, `INSERT INTO conditional_access_scep_serials () VALUES ()`)
		require.NoError(t, err)

		lastID, err := result.LastInsertId()
		require.NoError(t, err)
		serialNumber = uint64(lastID) // nolint:gosec,G115

		_, err = q.ExecContext(ctx, `
			INSERT INTO conditional_access_scep_certificates
				(serial, host_id, name, not_valid_before, not_valid_after, certificate_pem, revoked)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, serialNumber, hostID, name, notValidBefore, notValidAfter, certPEM, revoked)
		return err
	})

	return serialNumber
}
