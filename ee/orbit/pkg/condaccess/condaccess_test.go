package condaccess

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeSelfSignedCert creates a minimal self-signed cert for testing, expiring at notAfter.
func makeSelfSignedCert(t *testing.T, notAfter time.Time) *x509.Certificate {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}

// writeCertToDisk writes a DER-encoded cert as PEM to the given directory.
func writeCertToDisk(t *testing.T, dir string, cert *x509.Certificate) {
	t.Helper()
	certPath := filepath.Join(dir, constant.ConditionalAccessCertFileName)
	f, err := os.OpenFile(certPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	require.NoError(t, err)
	defer f.Close()
	require.NoError(t, pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
}

func TestEnroll_ValidCertExists(t *testing.T) {
	// A valid cert that is not near expiry — SCEP should never be called.
	dir := t.TempDir()
	cert := makeSelfSignedCert(t, time.Now().Add(90*24*time.Hour))
	writeCertToDisk(t, dir, cert)

	scepCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scepCalled = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	got, err := Enroll(context.Background(), dir, srv.URL, "challenge", "uuid-1234", "", true, zerolog.Nop())
	require.NoError(t, err)
	assert.Equal(t, cert.SerialNumber, got.SerialNumber)
	assert.False(t, scepCalled, "SCEP server must not be contacted for a valid cert")
}

func TestEnroll_ExpiringCert(t *testing.T) {
	// Cert expiring in 10 days — within the 30-day renewal threshold.
	dir := t.TempDir()
	oldCert := makeSelfSignedCert(t, time.Now().Add(10*24*time.Hour))
	writeCertToDisk(t, dir, oldCert)

	// We just verify that SCEP is contacted (it will fail with a non-SCEP response,
	// and that error surfaces). The important assertion is that a new enrollment was attempted.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	_, err := Enroll(context.Background(), dir, srv.URL, "challenge", "uuid-1234", "", true, zerolog.Nop())
	require.Error(t, err, "expected enrollment to fail when SCEP stub returns error")
	assert.Contains(t, err.Error(), "SCEP enrollment")
}

func TestEnroll_SCEPError(t *testing.T) {
	// No existing cert, SCEP fails — no partial cert file should be created.
	dir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	_, err := Enroll(context.Background(), dir, srv.URL, "challenge", "uuid-1234", "", true, zerolog.Nop())
	require.Error(t, err)

	// cert file must not exist (key file may exist since it's written before SCEP call)
	_, statErr := os.Stat(filepath.Join(dir, constant.ConditionalAccessCertFileName))
	assert.True(t, os.IsNotExist(statErr), "cert file must not be written on SCEP failure")
}

func TestEnroll_NoExistingCert_SavesKeyAndCert(t *testing.T) {
	// Without an httptest SCEP stub that speaks full SCEP protocol we can only verify
	// that Enroll attempts enrollment and that the key file is created before the SCEP call.
	dir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	_, err := Enroll(context.Background(), dir, srv.URL, "challenge", "uuid-1234", "", true, zerolog.Nop())
	require.Error(t, err, "expected error from stub SCEP server")

	// The key must have been written before the SCEP call.
	keyPath := filepath.Join(dir, constant.ConditionalAccessKeyFileName)
	keyInfo, statErr := os.Stat(keyPath)
	require.NoError(t, statErr, "key file must be created before SCEP call")
	assert.Equal(t, os.FileMode(constant.DefaultFileMode), keyInfo.Mode().Perm(), "key file must be mode 0600")
}

func TestCertNeedsRenewal(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	makeExpiry := func(d time.Duration) *x509.Certificate {
		template := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(d),
		}
		der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
		require.NoError(t, err)
		cert, err := x509.ParseCertificate(der)
		require.NoError(t, err)
		return cert
	}

	assert.True(t, certNeedsRenewal(makeExpiry(10*24*time.Hour), renewalThreshold), "10 days should need renewal")
	assert.False(t, certNeedsRenewal(makeExpiry(90*24*time.Hour), renewalThreshold), "90 days should not need renewal")
}
