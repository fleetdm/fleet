package mysql

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/types"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostIdentitySCEP(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"GetHostIdentityCert", testGetHostIdentityCert},
		{"UpdateHostIdentityCertHostIDBySerial", testUpdateHostIdentityCertHostIDBySerial},
		{"HostIdentityCertificateIntegration", testHostIdentityCertificateIntegration},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// insertTestCertificate is a shared helper function to insert a host identity certificate for testing
func insertTestCertificate(t *testing.T, ds *Datastore, serial uint64, hostID *uint, name string, notBefore, notAfter time.Time, revoked bool) {
	ctx := t.Context()
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)

	publicKeyRaw, err := types.CreateECDSAPublicKeyRaw(&privateKey.PublicKey)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(int64(serial)), // nolint:gosec // ignore integer overflow
		Subject: pkix.Name{
			CommonName: name,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Insert serial number
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_identity_scep_serials (serial) VALUES (?)`, serial)
	require.NoError(t, err)

	// Insert certificate
	_, err = ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO host_identity_scep_certificates 
		(serial, host_id, name, not_valid_before, not_valid_after, certificate_pem, public_key_raw, revoked)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, serial, hostID, name, notBefore, notAfter, string(certPEM), publicKeyRaw, revoked)
	require.NoError(t, err)
}

// insertSimpleTestCertificate is a convenience wrapper for insertTestCertificate with default times
func insertSimpleTestCertificate(t *testing.T, ds *Datastore, serial uint64, hostID *uint, name string) {
	now := time.Now()
	notBefore := now.Add(-1 * time.Hour)
	notAfter := now.Add(24 * time.Hour)
	insertTestCertificate(t, ds, serial, hostID, name, notBefore, notAfter, false)
}

func testGetHostIdentityCert(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create a test host
	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("test-node-key"),
		UUID:            "test-uuid",
		Hostname:        "test-hostname",
		Platform:        "linux",
	})
	require.NoError(t, err)

	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)
	futureTime := now.Add(24 * time.Hour)

	// Test cases that apply to both methods
	type testCase struct {
		name        string
		serial      uint64
		certName    string
		hostID      *uint
		notBefore   time.Time
		notAfter    time.Time
		revoked     bool
		shouldExist bool
	}

	testCases := []testCase{
		{
			name:        "valid certificate",
			serial:      1001,
			certName:    "test-host-valid",
			hostID:      &host.ID,
			notBefore:   pastTime,
			notAfter:    futureTime,
			revoked:     false,
			shouldExist: true,
		},
		{
			name:        "expired certificate",
			serial:      1002,
			certName:    "test-host-expired",
			hostID:      &host.ID,
			notBefore:   pastTime,
			notAfter:    now.Add(-1 * time.Hour),
			revoked:     false,
			shouldExist: false,
		},
		{
			name:        "revoked certificate",
			serial:      1003,
			certName:    "test-host-revoked",
			hostID:      &host.ID,
			notBefore:   pastTime,
			notAfter:    futureTime,
			revoked:     true,
			shouldExist: false,
		},
		{
			name:        "certificate with nil host_id",
			serial:      1004,
			certName:    "test-host-no-host-id",
			hostID:      nil,
			notBefore:   pastTime,
			notAfter:    futureTime,
			revoked:     false,
			shouldExist: true,
		},
	}

	// Helper function to assert certificate properties
	assertCertificate := func(t *testing.T, cert *types.HostIdentityCertificate, err error, tc testCase) {
		if tc.shouldExist {
			require.NoError(t, err)
			require.NotNil(t, cert)
			assert.Equal(t, tc.serial, cert.SerialNumber)
			assert.Equal(t, tc.certName, cert.CommonName)
			if tc.hostID != nil {
				assert.Equal(t, *tc.hostID, *cert.HostID)
			} else {
				assert.Nil(t, cert.HostID)
			}
			assert.WithinDuration(t, tc.notAfter, cert.NotValidAfter, 5*time.Second)
			assert.NotEmpty(t, cert.PublicKeyRaw)

			// Test that we can unmarshal the public key
			publicKey, err := cert.UnmarshalPublicKey()
			require.NoError(t, err)
			assert.Equal(t, elliptic.P384(), publicKey.Curve)
		} else {
			require.Error(t, err)
			assert.Nil(t, cert)
			assert.True(t, fleet.IsNotFound(err))
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Insert the test certificate
			insertTestCertificate(t, ds, tc.serial, tc.hostID, tc.certName, tc.notBefore, tc.notAfter, tc.revoked)

			// Test GetHostIdentityCertBySerialNumber
			certBySerial, err := ds.GetHostIdentityCertBySerialNumber(ctx, tc.serial)
			assertCertificate(t, certBySerial, err, tc)

			// Test GetHostIdentityCertByName
			certByName, err := ds.GetHostIdentityCertByName(ctx, tc.certName)
			assertCertificate(t, certByName, err, tc)
		})
	}

	// Test cases specific to serial number lookup
	t.Run("certificate not found by serial", func(t *testing.T) {
		serial := uint64(9999)

		cert, err := ds.GetHostIdentityCertBySerialNumber(ctx, serial)
		require.Error(t, err)
		assert.Nil(t, cert)
		assert.True(t, fleet.IsNotFound(err))
	})

	// Test cases specific to name lookup
	t.Run("certificate not found by name", func(t *testing.T) {
		name := "non-existent-cert"

		cert, err := ds.GetHostIdentityCertByName(ctx, name)
		require.Error(t, err)
		assert.Nil(t, cert)
		assert.True(t, fleet.IsNotFound(err))
	})

	t.Run("empty name parameter", func(t *testing.T) {
		cert, err := ds.GetHostIdentityCertByName(ctx, "")
		require.Error(t, err)
		assert.Nil(t, cert)
		assert.True(t, fleet.IsNotFound(err))
	})
}

func testUpdateHostIdentityCertHostIDBySerial(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create test hosts
	host1, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("test-node-key-1"),
		UUID:            "test-uuid-1",
		Hostname:        "test-hostname-1",
		Platform:        "linux",
	})
	require.NoError(t, err)

	host2, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("test-node-key-2"),
		UUID:            "test-uuid-2",
		Hostname:        "test-hostname-2",
		Platform:        "linux",
	})
	require.NoError(t, err)

	t.Run("update certificate with nil host_id", func(t *testing.T) {
		serial := uint64(2001)
		name := "test-cert-update-nil"

		insertSimpleTestCertificate(t, ds, serial, nil, name)

		err := ds.UpdateHostIdentityCertHostIDBySerial(ctx, serial, host1.ID)
		require.NoError(t, err)

		// Verify the update
		cert, err := ds.GetHostIdentityCertBySerialNumber(ctx, serial)
		require.NoError(t, err)
		require.NotNil(t, cert.HostID)
		assert.Equal(t, host1.ID, *cert.HostID)
	})

	t.Run("update certificate with existing host_id", func(t *testing.T) {
		serial := uint64(2002)
		name := "test-cert-update-existing"

		insertSimpleTestCertificate(t, ds, serial, &host1.ID, name)

		err := ds.UpdateHostIdentityCertHostIDBySerial(ctx, serial, host2.ID)
		require.NoError(t, err)

		// Verify the update
		cert, err := ds.GetHostIdentityCertBySerialNumber(ctx, serial)
		require.NoError(t, err)
		require.NotNil(t, cert.HostID)
		assert.Equal(t, host2.ID, *cert.HostID)
	})

	t.Run("update non-existent certificate", func(t *testing.T) {
		serial := uint64(9999)

		err := ds.UpdateHostIdentityCertHostIDBySerial(ctx, serial, host1.ID)
		require.NoError(t, err) // MySQL UPDATE with no matching rows doesn't return an error
	})
}

func testHostIdentityCertificateIntegration(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create a test host
	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("test-node-key"),
		UUID:            "test-uuid",
		Hostname:        "test-hostname",
		Platform:        "linux",
	})
	require.NoError(t, err)

	t.Run("complete workflow", func(t *testing.T) {
		serial := uint64(4001)
		name := "integration-test-cert"

		// 1. Insert certificate without host_id
		insertSimpleTestCertificate(t, ds, serial, nil, name)

		// 2. Verify we can get it by serial number
		cert, err := ds.GetHostIdentityCertBySerialNumber(ctx, serial)
		require.NoError(t, err)
		require.NotNil(t, cert)
		assert.Nil(t, cert.HostID)

		// 3. Verify we can get it by name
		cert, err = ds.GetHostIdentityCertByName(ctx, name)
		require.NoError(t, err)
		require.NotNil(t, cert)
		assert.Nil(t, cert.HostID)

		// 4. Update the host_id
		err = ds.UpdateHostIdentityCertHostIDBySerial(ctx, serial, host.ID)
		require.NoError(t, err)

		// 5. Verify the update worked for both methods
		cert, err = ds.GetHostIdentityCertBySerialNumber(ctx, serial)
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.NotNil(t, cert.HostID)
		assert.Equal(t, host.ID, *cert.HostID)

		cert, err = ds.GetHostIdentityCertByName(ctx, name)
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.NotNil(t, cert.HostID)
		assert.Equal(t, host.ID, *cert.HostID)
	})
}
