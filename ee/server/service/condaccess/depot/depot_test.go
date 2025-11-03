package depot

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConditionalAccessSCEPDepot(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *mysql.Datastore)
	}{
		{"CARetrievalWithoutAssets", testCARetrievalWithoutAssets},
		{"SerialAllocation", testSerialAllocation},
		{"PutWithValidHost", testPutWithValidHost},
		{"PutWithoutHost", testPutWithoutHost},
		{"PutWithoutSANURI", testPutWithoutSANURI},
		{"PutMultipleCerts", testPutMultipleCerts},
		{"PutRateLimiting", testPutRateLimiting},
		{"ExtractUUID", testExtractUUID},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testCARetrievalWithoutAssets(t *testing.T, ds *mysql.Datastore) {
	cfg := config.TestConfig()
	d, err := createTestDepot(t, ds, cfg)
	require.NoError(t, err)

	// Without assets initialized, CA() should fail
	_, _, err = d.CA(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting assets")
}

func testSerialAllocation(t *testing.T, ds *mysql.Datastore) {
	cfg := config.TestConfig()
	d, err := createTestDepot(t, ds, cfg)
	require.NoError(t, err)

	// Allocate multiple serial numbers and verify they're unique and increasing
	serials := make([]*big.Int, 5)
	for i := 0; i < 5; i++ {
		serial, err := d.Serial()
		require.NoError(t, err)
		require.NotNil(t, serial)
		serials[i] = serial

		// Verify serial is positive
		assert.True(t, serial.Sign() > 0)

		// Verify serial is unique
		for j := 0; j < i; j++ {
			assert.NotEqual(t, serials[j].Int64(), serial.Int64())
		}
	}

	// Verify serials are increasing
	for i := 1; i < len(serials); i++ {
		assert.True(t, serials[i].Cmp(serials[i-1]) > 0)
	}
}

func testPutWithValidHost(t *testing.T, ds *mysql.Datastore) {
	ctx := t.Context()
	cfg := config.TestConfig()
	d, err := createTestDepot(t, ds, cfg)
	require.NoError(t, err)

	// Create a test host
	host, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("test-host-put-1"),
		NodeKey:         ptr.String("test-node-key-put-1"),
		UUID:            "test-uuid-put-1",
		Hostname:        "test-hostname-put-1",
		Platform:        "darwin",
		DetailUpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	// Create a certificate with SAN URI containing the host UUID
	cert := createTestCert(t, d, host.UUID)

	// Store the certificate
	err = d.Put("test-cn", cert)
	require.NoError(t, err)

	// Verify certificate was stored
	hostID, err := ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, uint64(cert.SerialNumber.Int64()))
	require.NoError(t, err)
	assert.Equal(t, host.ID, hostID)
}

func testPutWithoutHost(t *testing.T, ds *mysql.Datastore) {
	cfg := config.TestConfig()
	d, err := createTestDepot(t, ds, cfg)
	require.NoError(t, err)

	// Create a certificate with SAN URI for non-existent host
	cert := createTestCert(t, d, "non-existent-uuid")

	// Storing the certificate should fail
	err = d.Put("test-cn", cert)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "host not found")
}

func testPutWithoutSANURI(t *testing.T, ds *mysql.Datastore) {
	cfg := config.TestConfig()
	d, err := createTestDepot(t, ds, cfg)
	require.NoError(t, err)

	// Create a certificate without SAN URI
	cert := createTestCertWithOptions(t, d, nil)

	// Storing the certificate should fail
	err = d.Put("test-cn", cert)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no device UUID found")
}

func testPutMultipleCerts(t *testing.T, ds *mysql.Datastore) {
	ctx := t.Context()
	cfg := config.TestConfig()
	// Disable rate limiting for this test
	cfg.Osquery.EnrollCooldown = 0
	d, err := createTestDepot(t, ds, cfg)
	require.NoError(t, err)

	// Create a test host
	host, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("test-host-multi-1"),
		NodeKey:         ptr.String("test-node-key-multi-1"),
		UUID:            "test-uuid-multi-1",
		Hostname:        "test-hostname-multi-1",
		Platform:        "darwin",
		DetailUpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	// Store first certificate
	cert1 := createTestCert(t, d, host.UUID)
	err = d.Put("test-cn-1", cert1)
	require.NoError(t, err)

	// Verify first cert is stored
	hostID, err := ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, uint64(cert1.SerialNumber.Int64()))
	require.NoError(t, err)
	assert.Equal(t, host.ID, hostID)

	// Store second certificate for same host
	cert2 := createTestCert(t, d, host.UUID)
	err = d.Put("test-cn-2", cert2)
	require.NoError(t, err)

	// Verify second cert is stored
	hostID, err = ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, uint64(cert2.SerialNumber.Int64()))
	require.NoError(t, err)
	assert.Equal(t, host.ID, hostID)

	// Verify first cert is STILL VALID (not revoked)
	// This follows industry best practice: keep old certs valid during grace period
	hostID, err = ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, uint64(cert1.SerialNumber.Int64()))
	require.NoError(t, err)
	assert.Equal(t, host.ID, hostID, "old certificate should remain valid for grace period")
}

func testPutRateLimiting(t *testing.T, ds *mysql.Datastore) {
	ctx := t.Context()
	cfg := config.TestConfig()
	// Set a 5-second cooldown
	cfg.Osquery.EnrollCooldown = 5 * time.Second
	d, err := createTestDepot(t, ds, cfg)
	require.NoError(t, err)

	// Create a test host
	host, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("test-host-rate-1"),
		NodeKey:         ptr.String("test-node-key-rate-1"),
		UUID:            "test-uuid-rate-1",
		Hostname:        "test-hostname-rate-1",
		Platform:        "darwin",
		DetailUpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	// Store first certificate
	cert1 := createTestCert(t, d, host.UUID)
	err = d.Put("test-cn-1", cert1)
	require.NoError(t, err)

	// Try to store second certificate immediately (should fail)
	cert2 := createTestCert(t, d, host.UUID)
	err = d.Put("test-cn-2", cert2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requesting certificates too often")

	// Manually update the timestamp to simulate cooldown period passing
	// This is faster and more deterministic than time.Sleep
	mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			UPDATE conditional_access_scep_certificates
			SET created_at = DATE_SUB(NOW(), INTERVAL 10 SECOND)
			WHERE serial = ?
		`, cert1.SerialNumber.Int64())
		return err
	})

	// Now it should succeed
	cert3 := createTestCert(t, d, host.UUID)
	err = d.Put("test-cn-3", cert3)
	require.NoError(t, err)
}

func testExtractUUID(t *testing.T, ds *mysql.Datastore) {
	cfg := config.TestConfig()
	d, err := createTestDepot(t, ds, cfg)
	require.NoError(t, err)

	tests := []struct {
		name     string
		uuid     string
		expected string
	}{
		{
			name:     "valid UUID",
			uuid:     "550e8400-e29b-41d4-a716-446655440000",
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "another valid UUID",
			uuid:     "test-uuid-123",
			expected: "test-uuid-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a certificate with the UUID in SAN URI
			cert := createTestCert(t, d, tt.uuid)

			// Extract UUID
			uuid := extractUUIDFromCert(cert)
			assert.Equal(t, tt.expected, uuid)
		})
	}

	// Test with no URIs
	t.Run("no URIs", func(t *testing.T) {
		cert := createTestCertWithOptions(t, d, nil)

		uuid := extractUUIDFromCert(cert)
		assert.Equal(t, "", uuid)
	})
}

// Helper functions

func createTestDepot(t *testing.T, ds *mysql.Datastore, cfg config.FleetConfig) (*ConditionalAccessSCEPDepot, error) {
	t.Helper()

	// Access the underlying *sqlx.DB using the test helper.
	// Note: ds.primary is unexported, so tests in other packages need GetUnderlyingDB()
	// Tests in the mysql package can use ds.primary directly.
	db := mysql.GetUnderlyingDB(ds)
	return NewConditionalAccessSCEPDepot(db, ds, log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout)), &cfg)
}

// createTestCertWithOptions creates a test certificate with optional SAN URIs.
// If uris is nil, no URIs are added to the certificate.
func createTestCertWithOptions(t *testing.T, d *ConditionalAccessSCEPDepot, uris []*url.URL) *x509.Certificate {
	t.Helper()

	serial, err := d.Serial()
	require.NoError(t, err)

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "test-cn",
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		URIs:      uris,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return cert
}

// createTestCert creates a test certificate with SAN URI containing the given UUID
func createTestCert(t *testing.T, d *ConditionalAccessSCEPDepot, uuid string) *x509.Certificate {
	t.Helper()

	deviceURI, err := url.Parse("urn:device:apple:uuid:" + uuid)
	require.NoError(t, err)

	return createTestCertWithOptions(t, d, []*url.URL{deviceURI})
}
