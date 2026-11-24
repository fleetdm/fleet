package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestCertBytes(t *testing.T, isCA bool, commonName string) []byte {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Fleet Test Org"},
		},
		Issuer: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Fleet Test Org"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour).Truncate(time.Second),
		NotAfter:              time.Now().Add(24 * time.Hour).Truncate(time.Second),
		IsCA:                  isCA,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	certDer, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDer,
	})
}

func TestCreateAndroidCACertificate(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}})
	adminCtx := viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	certBytes := generateTestCertBytes(t, true, "Corporate CA")

	ds.CreateAndroidCACertificateFunc = func(ctx context.Context, cert *fleet.AndroidCACertificate) (*fleet.AndroidCACertificate, error) {
		cert.ID = 1
		return cert, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	t.Run("Requires premium license", func(t *testing.T) {
		freeSvc, freeCtx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierFree}})
		freeAdminCtx := viewer.NewContext(freeCtx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
		_, err := freeSvc.CreateAndroidCACertificate(freeAdminCtx, &fleet.AndroidCACertificate{Name: "CA"}, certBytes)
		require.ErrorIs(t, err, fleet.ErrMissingLicense)
	})

	t.Run("Rejects empty certificate bytes", func(t *testing.T) {
		_, err := svc.CreateAndroidCACertificate(adminCtx, &fleet.AndroidCACertificate{Name: "CA"}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("Creates CA certificate successfully", func(t *testing.T) {
		created, err := svc.CreateAndroidCACertificate(adminCtx, &fleet.AndroidCACertificate{
			TeamID: 0,
			Name:   "Corp Root CA",
		}, certBytes)
		require.NoError(t, err)
		assert.Equal(t, uint(1), created.ID)
		assert.Equal(t, "Corp Root CA", created.Name)
		assert.NotEmpty(t, created.SHA256Fingerprint)
	})
}

func TestDeviceCACertificateEndpoints(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}})

	h := &fleet.Host{
		ID:       10,
		UUID:     "host-uuid-10",
		Platform: "android",
		TeamID:   ptr.Uint(1),
	}
	deviceCtx := host.NewContext(ctx, h)

	ds.GetAndroidCACertificateByIDFunc = func(ctx context.Context, id uint) (*fleet.AndroidCACertificate, error) {
		return &fleet.AndroidCACertificate{
			ID:                id,
			TeamID:            1,
			Name:              "Test CA",
			CertificatePEM:    "pem-content",
			SHA256Fingerprint: "sha256-hash",
		}, nil
	}

	ds.UpsertHostAndroidCACertificateStatusFunc = func(ctx context.Context, hostID uint, hostUUID string, caCertID uint, status fleet.MDMDeliveryStatus, detail string) error {
		assert.Equal(t, uint(10), hostID)
		assert.Equal(t, "host-uuid-10", hostUUID)
		assert.Equal(t, uint(5), caCertID)
		assert.Equal(t, fleet.MDMDeliveryVerified, status)
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	t.Run("Get device CA cert", func(t *testing.T) {
		cert, err := svc.GetDeviceCACertificate(deviceCtx, 5)
		require.NoError(t, err)
		assert.Equal(t, uint(5), cert.ID)
		assert.Equal(t, "Test CA", cert.Name)
		assert.Equal(t, "pem-content", cert.CertificatePEM)
	})

	t.Run("Update device CA cert status", func(t *testing.T) {
		err := svc.UpdateDeviceCACertificateStatus(deviceCtx, 5, fleet.MDMDeliveryVerified, nil)
		require.NoError(t, err)
	})
}
