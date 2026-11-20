package mysql

import (
	"context"
	"strings"
	"sync"
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
		{"CertificateLifecycle", testConditionalAccessCertificateLifecycle},
		{"OneTimeChallenges", testConditionalAccessOneTimeChallenges},
		{"OneTimeChallengeAppleExpansion", testConditionalAccessOneTimeChallengeAppleExpansion},
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
		serialNumber = uint64(lastID) //nolint:gosec // G115

		_, err = q.ExecContext(ctx, `
			INSERT INTO conditional_access_scep_certificates
				(serial, host_id, name, not_valid_before, not_valid_after, certificate_pem, revoked)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, serialNumber, hostID, name, notValidBefore, notValidAfter, certPEM, revoked)
		return err
	})

	return serialNumber
}

func testConditionalAccessCertificateLifecycle(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	now := time.Now()

	// Create two test hosts
	host1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("test-host-lifecycle-1"), NodeKey: ptr.String("test-node-key-1"),
		UUID: "test-uuid-1", Hostname: "test-hostname-1", Platform: "darwin", DetailUpdatedAt: now,
	})
	require.NoError(t, err)
	host2, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("test-host-lifecycle-2"), NodeKey: ptr.String("test-node-key-2"),
		UUID: "test-uuid-2", Hostname: "test-hostname-2", Platform: "darwin", DetailUpdatedAt: now,
	})
	require.NoError(t, err)

	// Host 1: Multiple valid certs (simulating rotation)
	s1_old := insertConditionalAccessCert(t, ds, ctx, host1.ID, "h1-old", now.Add(-3*time.Hour), now.Add(24*time.Hour), false)
	s1_new := insertConditionalAccessCert(t, ds, ctx, host1.ID, "h1-new", now.Add(-2*time.Hour), now.Add(24*time.Hour), false)

	// Host 2: New cert within grace period + 1 revoked cert
	s2_old := insertConditionalAccessCert(t, ds, ctx, host2.ID, "h2-old", now.Add(-3*time.Hour), now.Add(24*time.Hour), false)
	s2_new := insertConditionalAccessCert(t, ds, ctx, host2.ID, "h2-new", now.Add(-30*time.Minute), now.Add(24*time.Hour), false)
	s2_revoked := insertConditionalAccessCert(t, ds, ctx, host2.ID, "h2-revoked", now.Add(-4*time.Hour), now.Add(24*time.Hour), true)

	// All valid certs authenticate
	for _, s := range []uint64{s1_old, s1_new, s2_old, s2_new} {
		_, err := ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, s)
		require.NoError(t, err)
	}

	// Revoked cert does not authenticate
	_, err = ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, s2_revoked)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	// Cleanup with 1-hour grace: only host1's old cert eligible (host1's new cert is 2h old)
	count, err := ds.RevokeOldConditionalAccessCerts(ctx, 1*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Host1's old cert now revoked, new cert still works
	_, err = ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, s1_old)
	assert.True(t, fleet.IsNotFound(err))
	_, err = ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, s1_new)
	require.NoError(t, err)

	// Host2's certs still work (new cert within grace period)
	_, err = ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, s2_old)
	require.NoError(t, err)
	_, err = ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, s2_new)
	require.NoError(t, err)

	// Second cleanup revokes nothing
	count, err = ds.RevokeOldConditionalAccessCerts(ctx, 1*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// GetConditionalAccessCertCreatedAtByHostID returns most recent cert
	createdAt, err := ds.GetConditionalAccessCertCreatedAtByHostID(ctx, host2.ID)
	require.NoError(t, err)
	assert.True(t, createdAt.After(now.Add(-1*time.Hour)) && createdAt.Before(now.Add(1*time.Minute)))
}

func testConditionalAccessOneTimeChallenges(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	enrollmentID := insertWindowsEnrollment(t, ds, "ca-hw-1", "ca-host-uuid").ID

	challenge, err := ds.FindOrCreateWindowsConditionalAccessSCEPChallenge(ctx, enrollmentID)
	require.NoError(t, err)
	// Windows sends the challenge as a PrintableString, which has no '_'.
	require.Regexp(t, `^[0-9a-f]{64}$`, challenge)

	var row struct {
		fleet.AppleSCEPChallengeInfo
		SecondsLeft int64 `db:"seconds_left"`
	}
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &row, `SELECT purpose, host_uuid, hardware_serial, idp_account_uuid,
			TIMESTAMPDIFF(SECOND, NOW(6), expires_at) AS seconds_left FROM mdm_apple_scep_challenges WHERE challenge = ?`, challenge)
	})
	require.Equal(t, fleet.AppleSCEPChallengeInfo{Purpose: fleet.ConditionalAccessSCEPChallengePurpose, UUID: new("ca-host-uuid")}, row.AppleSCEPChallengeInfo)
	require.InDelta(t, fleet.ConditionalAccessSCEPChallengeTTL.Seconds(), row.SecondsLeft, 5)

	// every send until the host uses it carries the same challenge
	again, err := ds.FindOrCreateWindowsConditionalAccessSCEPChallenge(ctx, enrollmentID)
	require.NoError(t, err)
	require.Equal(t, challenge, again)

	// Fleet's Apple MDM SCEP server neither accepts nor uses it up
	_, err = ds.ConsumeAppleSCEPChallenge(ctx, challenge)
	require.True(t, fleet.IsNotFound(err), err)

	// another host's attempt fails without using it up
	err = ds.ConsumeConditionalAccessSCEPChallenge(ctx, challenge, "other-host-uuid")
	require.True(t, fleet.IsNotFound(err), err)
	require.NoError(t, ds.ConsumeConditionalAccessSCEPChallenge(ctx, challenge, "ca-host-uuid"))

	// single use
	err = ds.ConsumeConditionalAccessSCEPChallenge(ctx, challenge, "ca-host-uuid")
	require.True(t, fleet.IsNotFound(err), err)

	// once used, the next send gets a new one
	next, err := ds.FindOrCreateWindowsConditionalAccessSCEPChallenge(ctx, enrollmentID)
	require.NoError(t, err)
	require.NotEqual(t, challenge, next)

	// an expired challenge is neither accepted nor reused
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE mdm_apple_scep_challenges SET expires_at = NOW(6) - INTERVAL 1 SECOND WHERE challenge = ?`, next)
		return err
	})
	err = ds.ConsumeConditionalAccessSCEPChallenge(ctx, next, "ca-host-uuid")
	require.True(t, fleet.IsNotFound(err), err)
	afterExpiry, err := ds.FindOrCreateWindowsConditionalAccessSCEPChallenge(ctx, enrollmentID)
	require.NoError(t, err)
	require.NotEqual(t, next, afterExpiry)

	require.Error(t, ds.ConsumeConditionalAccessSCEPChallenge(ctx, "", "ca-host-uuid"))
	require.Error(t, ds.ConsumeConditionalAccessSCEPChallenge(ctx, afterExpiry, ""))
	err = ds.ConsumeConditionalAccessSCEPChallenge(ctx, "does-not-exist", "ca-host-uuid")
	require.True(t, fleet.IsNotFound(err), err)

	// a challenge must be bound to a host, so an enrollment not linked to one gets none
	_, err = ds.FindOrCreateWindowsConditionalAccessSCEPChallenge(ctx, insertWindowsEnrollment(t, ds, "ca-hw-unlinked", "").ID)
	require.Error(t, err)
	_, err = ds.FindOrCreateWindowsConditionalAccessSCEPChallenge(ctx, 999999)
	require.True(t, fleet.IsNotFound(err), err)

	// the conditional access SCEP server doesn't accept an Apple MDM challenge, and leaves it usable
	insertAppleSCEPChallenge(t, ds, "apple-renewal", fleet.AppleMDMCertPurposeSCEPRenewal, time.Hour, nil)
	err = ds.ConsumeConditionalAccessSCEPChallenge(ctx, "apple-renewal", "uuid-apple-renewal")
	require.True(t, fleet.IsNotFound(err), err)
	_, err = ds.ConsumeAppleSCEPChallenge(ctx, "apple-renewal")
	require.NoError(t, err)

	// concurrent sends to the same enrollment create a single challenge
	concurrentID := insertWindowsEnrollment(t, ds, "ca-hw-concurrent", "ca-concurrent-uuid").ID
	const workers = 10
	results := make([]string, workers)
	var wg sync.WaitGroup
	for i := range workers {
		wg.Go(func() {
			c, err := ds.FindOrCreateWindowsConditionalAccessSCEPChallenge(ctx, concurrentID)
			assert.NoError(t, err)
			results[i] = c
		})
	}
	wg.Wait()
	for _, r := range results {
		require.Equal(t, results[0], r)
	}
}

func testConditionalAccessOneTimeChallengeAppleExpansion(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	doc := `<string>` + fleet.HostSecretPlaceholder(fleet.HostSecretConditionalAccessSCEPChallenge) + `</string>`
	expand := func(t *testing.T, enrollmentID string) string {
		t.Helper()
		expanded, err := ds.ExpandHostSecrets(ctx, doc, enrollmentID)
		require.NoError(t, err)
		challenge := strings.TrimSuffix(strings.TrimPrefix(expanded, "<string>"), "</string>")
		require.Regexp(t, `^[0-9a-f]{64}$`, challenge)
		return challenge
	}

	nanoEnroll(t, ds, &fleet.Host{UUID: "ca-mac-udid", Platform: "darwin"}, false)
	challenge := expand(t, "ca-mac-udid")
	require.Equal(t, challenge, expand(t, "ca-mac-udid"))

	// bound to the host the device channel names, which is the UUID in the profile's SAN
	err := ds.ConsumeConditionalAccessSCEPChallenge(ctx, challenge, "other-udid")
	require.True(t, fleet.IsNotFound(err), err)
	require.NoError(t, ds.ConsumeConditionalAccessSCEPChallenge(ctx, challenge, "ca-mac-udid"))
	require.NotEqual(t, challenge, expand(t, "ca-mac-udid"))

	// separate from the renewal challenge for the same enrollment
	renewal, err := ds.ExpandHostSecrets(ctx, `<string>`+fleet.HostSecretPlaceholder(fleet.HostSecretSCEPChallenge)+`</string>`, "ca-mac-udid")
	require.NoError(t, err)
	require.NotContains(t, renewal, expand(t, "ca-mac-udid"))

	_, err = ds.ExpandHostSecrets(ctx, doc, "unknown-enrollment")
	require.Error(t, err)
}
