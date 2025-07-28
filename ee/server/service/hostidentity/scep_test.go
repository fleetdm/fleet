package hostidentity

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/smallstep/scep"
	"github.com/stretchr/testify/require"
)

// notFoundError implements fleet.NotFoundError for testing
type notFoundError struct{}

func (e *notFoundError) Error() string    { return "not found" }
func (e *notFoundError) IsNotFound() bool { return true }

func TestChallengeMiddleware_Renewal(t *testing.T) {
	ctx := t.Context()

	// Create a mock datastore
	ds := new(mock.Store)

	// Create a mock signer that just returns a test certificate
	mockSigner := &mockCSRSigner{
		signFunc: func(_ context.Context, m *scep.CSRReqMessage) (*x509.Certificate, error) {
			return &x509.Certificate{
				SerialNumber: big.NewInt(1),
				Subject:      m.CSR.Subject,
				NotBefore:    time.Now(),
				NotAfter:     time.Now().Add(365 * 24 * time.Hour),
			}, nil
		},
	}

	// Create the middleware
	middleware := challengeMiddleware(ds, mockSigner)

	t.Run("initial enrollment with valid challenge", func(t *testing.T) {
		// Create a test CSR
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		template := x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: "test-host",
			},
		}
		csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, priv)
		require.NoError(t, err)

		csr, err := x509.ParseCertificateRequest(csrDER)
		require.NoError(t, err)

		m := &scep.CSRReqMessage{
			CSR:               csr,
			ChallengePassword: "valid-secret",
		}

		// Set up mock to accept the enrollment secret
		ds.VerifyEnrollSecretFunc = func(_ context.Context, secret string) (*fleet.EnrollSecret, error) {
			if secret == "valid-secret" {
				return &fleet.EnrollSecret{Secret: secret}, nil
			}
			return nil, &notFoundError{}
		}

		cert, err := middleware(ctx, m)
		require.NoError(t, err)
		require.NotNil(t, cert)
	})

	t.Run("initial enrollment with invalid challenge", func(t *testing.T) {
		// Create a test CSR
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		template := x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: "test-host",
			},
		}
		csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, priv)
		require.NoError(t, err)

		csr, err := x509.ParseCertificateRequest(csrDER)
		require.NoError(t, err)

		m := &scep.CSRReqMessage{
			CSR:               csr,
			ChallengePassword: "invalid-secret",
		}

		// Set up mock to reject the enrollment secret
		ds.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
			return nil, &notFoundError{}
		}

		cert, err := middleware(ctx, m)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid challenge")
		require.Nil(t, cert)
	})

	t.Run("renewal with valid certificate", func(t *testing.T) {
		// Create a test key pair
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		// Create a valid certificate (not expired)
		existingCert := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName: "test-host",
			},
			NotBefore: time.Now().Add(-24 * time.Hour),
			NotAfter:  time.Now().Add(24 * time.Hour),
			PublicKey: priv.Public(),
		}

		// Create a CSR with the same public key
		template := x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: "test-host",
			},
		}
		csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, priv)
		require.NoError(t, err)

		csr, err := x509.ParseCertificateRequest(csrDER)
		require.NoError(t, err)

		m := &scep.CSRReqMessage{
			CSR: csr,
			// No challenge password for renewal
		}

		// Put the renewal cert in context
		ctx := context.WithValue(context.Background(), renewalCertKey, existingCert)

		cert, err := middleware(ctx, m)
		require.NoError(t, err)
		require.NotNil(t, cert)
	})

	t.Run("renewal with different public key", func(t *testing.T) {
		// Create two different key pairs
		priv1, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		priv2, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		// Create a certificate with the first key
		existingCert := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName: "test-host",
			},
			NotBefore: time.Now().Add(-24 * time.Hour),
			NotAfter:  time.Now().Add(24 * time.Hour),
			PublicKey: priv1.Public(),
		}

		// Create a CSR with the second key (different from cert)
		template := x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: "test-host",
			},
		}
		csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, priv2)
		require.NoError(t, err)

		csr, err := x509.ParseCertificateRequest(csrDER)
		require.NoError(t, err)

		m := &scep.CSRReqMessage{
			CSR: csr,
		}

		// Put the renewal cert in context
		ctx := context.WithValue(context.Background(), renewalCertKey, existingCert)

		cert, err := middleware(ctx, m)
		require.Error(t, err)
		require.Contains(t, err.Error(), "CSR public key does not match signer certificate")
		require.Nil(t, cert)
	})
}

// mockCSRSigner implements scepserver.CSRSignerContext for testing
type mockCSRSigner struct {
	signFunc func(ctx context.Context, m *scep.CSRReqMessage) (*x509.Certificate, error)
}

func (m *mockCSRSigner) SignCSRContext(ctx context.Context, csr *scep.CSRReqMessage) (*x509.Certificate, error) {
	if m.signFunc != nil {
		return m.signFunc(ctx, csr)
	}
	return nil, nil
}
