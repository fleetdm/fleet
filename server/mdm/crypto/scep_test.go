package mdmcrypto

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestSCEPVerifierVerifyEmptyCerts(t *testing.T) {
	v := &SCEPVerifier{}
	err := v.Verify(context.Background(), nil)
	require.ErrorContains(t, err, "no certificate provided")
}

func TestVerify(t *testing.T) {
	ds := new(mock.Store)
	verifier := NewSCEPVerifier(ds)

	// generate a valid root certificate with ExtKeyUsageClientAuth
	validRootCertBytes, validRootCert, rootKey := generateRootCertificate(t, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth})
	_, validClientCert := generateClientCertificate(t, validRootCert, rootKey)

	// generate a root certificate with an unrelated ExtKeyUsage
	rootWithOtherUsagesBytes, rootWithOtherUsageCert, rootWithOtherUsageKey := generateRootCertificate(t, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth})
	_, validClientCertFromMultipleUsageRoot := generateClientCertificate(t, rootWithOtherUsageCert, rootWithOtherUsageKey)

	cases := []struct {
		name         string
		rootCert     []byte
		certToVerify *x509.Certificate
		wantErr      string
	}{
		{
			name:         "no certificate provided",
			rootCert:     nil,
			certToVerify: nil,
			wantErr:      "no certificate provided",
		},
		{
			name:         "error loading root cert from database",
			rootCert:     nil,
			certToVerify: validClientCert,
			wantErr:      "loading existing assets from the database",
		},
		{
			name:         "valid certificate verification succeeds",
			rootCert:     validRootCertBytes,
			certToVerify: validClientCert,
			wantErr:      "",
		},
		{
			name:         "valid certificate with unrelated key usage in root cert",
			rootCert:     rootWithOtherUsagesBytes,
			certToVerify: validClientCertFromMultipleUsageRoot,
			wantErr:      "",
		},
		{
			name:         "mismatched certificate presented",
			rootCert:     rootWithOtherUsagesBytes,
			certToVerify: validClientCert,
			wantErr:      "certificate signed by unknown authority",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
				_ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
				if tt.rootCert == nil {
					return nil, errors.New("test error")
				}

				return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
					fleet.MDMAssetCACert: {Value: tt.rootCert},
				}, nil
			}

			err := verifier.Verify(context.Background(), tt.certToVerify)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func generateRootCertificate(t *testing.T, extKeyUsages []x509.ExtKeyUsage) ([]byte, *x509.Certificate, *ecdsa.PrivateKey) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	rootCertTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Root CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           extKeyUsages,
		BasicConstraintsValid: true,
	}

	rootCertDER, err := x509.CreateCertificate(rand.Reader, rootCertTemplate, rootCertTemplate, &priv.PublicKey, priv)
	require.NoError(t, err)

	rootCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootCertDER})

	rootCert, err := x509.ParseCertificate(rootCertDER)
	require.NoError(t, err)

	return rootCertPEM, rootCert, priv
}

func generateClientCertificate(t *testing.T, rootCert *x509.Certificate, rootKey *ecdsa.PrivateKey) ([]byte, *x509.Certificate) {
	clientPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	clientCertTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Client"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(1 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientCertTemplate, rootCert, &clientPriv.PublicKey, rootKey)
	require.NoError(t, err)

	clientCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCertDER})

	clientCert, err := x509.ParseCertificate(clientCertDER)
	require.NoError(t, err)

	return clientCertPEM, clientCert
}

func TestVerifyIgnoreExpiry(t *testing.T) {
	now := time.Now()
	caValidFrom, caValidTo := now.Add(-48*time.Hour), now.Add(10*365*24*time.Hour)
	rootPEM, rootCert, rootKey := generateRSACA(t, caValidFrom, caValidTo)
	_, otherRootCert, otherRootKey := generateRSACA(t, caValidFrom, caValidTo)

	ds := new(mock.Store)
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Value: rootPEM},
		}, nil
	}

	strict := NewSCEPVerifier(ds)
	lenient := NewSCEPVerifier(ds, WithIgnoreExpiry())

	// SCEP enrollments use RSA 2048 device keys, ACME enrollments use EC P-384.
	leafKeys := map[string]func(t *testing.T) crypto.Signer{
		"rsa":  func(t *testing.T) crypto.Signer { return generateRSAKey(t) },
		"p384": func(t *testing.T) crypto.Signer { return generateP384Key(t) },
	}
	for name, newKey := range leafKeys {
		t.Run(name, func(t *testing.T) {
			issue := func(caCert *x509.Certificate, caKey crypto.Signer, notBefore, notAfter time.Time) *x509.Certificate {
				return generateClientCertWithValidity(t, caCert, caKey, newKey(t).Public(), notBefore, notAfter)
			}
			valid := issue(rootCert, rootKey, now.Add(-time.Hour), now.Add(24*time.Hour))
			expired := issue(rootCert, rootKey, now.Add(-24*time.Hour), now.Add(-time.Hour))
			notYetValid := issue(rootCert, rootKey, now.Add(time.Hour), now.Add(24*time.Hour))
			expiredFromOtherCA := issue(otherRootCert, otherRootKey, now.Add(-24*time.Hour), now.Add(-time.Hour))

			require.NoError(t, strict.Verify(t.Context(), valid))
			require.NoError(t, lenient.Verify(t.Context(), valid))

			require.ErrorContains(t, strict.Verify(t.Context(), expired), "certificate has expired or is not yet valid")
			require.NoError(t, lenient.Verify(t.Context(), expired))

			require.ErrorContains(t, lenient.Verify(t.Context(), notYetValid), "certificate has expired or is not yet valid")
			require.ErrorContains(t, lenient.Verify(t.Context(), expiredFromOtherCA), "certificate signed by unknown authority")
		})
	}
}

func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func generateP384Key(t *testing.T) *ecdsa.PrivateKey {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	return key
}

func generateRSACA(t *testing.T, notBefore, notAfter time.Time) ([]byte, *x509.Certificate, *rsa.PrivateKey) {
	key := generateRSAKey(t)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Test Root CA"}},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), cert, key
}

func generateClientCertWithValidity(t *testing.T, caCert *x509.Certificate, caKey crypto.Signer, pub crypto.PublicKey,
	notBefore, notAfter time.Time,
) *x509.Certificate {
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{Organization: []string{"Test Client"}},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, pub, caKey)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}
