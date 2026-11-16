package enrollment_test

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/ee/server/service/scep/sceptest"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/enrollment"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/x509util"
	smallstepscep "github.com/smallstep/scep"
	"github.com/stretchr/testify/require"
)

func newClient(t *testing.T, url string) scepclient.Client {
	t.Helper()
	client, err := scepclient.New(url, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	return client
}

func newCSR(t *testing.T, key crypto.Signer, challenge string) *x509.CertificateRequest {
	t.Helper()
	der, err := x509util.CreateCertificateRequest(rand.Reader, &x509util.CertificateRequest{
		Subject:           pkix.Name{CommonName: "test-host"},
		ChallengePassword: challenge,
	}, key)
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(der)
	require.NoError(t, err)
	return csr
}

// enroll runs the whole exchange for csr against url with an ephemeral signer.
func enroll(t *testing.T, url string, csr *x509.CertificateRequest) (*x509.Certificate, error) {
	t.Helper()
	client := newClient(t, url)
	caCerts, err := enrollment.FetchCACerts(t.Context(), client)
	if err != nil {
		return nil, err
	}
	signerKey, signerCert, err := enrollment.NewEphemeralSigner(csr.Subject)
	require.NoError(t, err)
	return enrollment.Enroll(t.Context(), client, caCerts, enrollment.Request{CSR: csr, SignerKey: signerKey, SignerCert: signerCert})
}

func TestEnrollAgainstSCEPServer(t *testing.T) {
	const challenge = "8CE317021F690069"
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		opts []sceptest.TestSCEPServerOption
		key  crypto.Signer
	}{
		"single CA certificate, ECDSA CSR": {key: ecKey},
		"RA chain, ECDSA CSR":              {opts: []sceptest.TestSCEPServerOption{sceptest.WithRAChain()}, key: ecKey},
		"RA chain, RSA CSR":                {opts: []sceptest.TestSCEPServerOption{sceptest.WithRAChain()}, key: rsaKey},
	} {
		t.Run(name, func(t *testing.T) {
			opts := append([]sceptest.TestSCEPServerOption{sceptest.WithIssuance(), sceptest.WithChallenge(challenge)}, tc.opts...)
			srv := sceptest.NewTestSCEPServer(t, opts...)

			cert, err := enroll(t, srv.URL+"/scep", newCSR(t, tc.key, challenge))
			require.NoError(t, err)
			require.Equal(t, "test-host", cert.Subject.CommonName)
			require.NoError(t, cert.CheckSignatureFrom(sceptest.CACertificate(t)))
			require.True(t, tc.key.Public().(interface{ Equal(crypto.PublicKey) bool }).Equal(cert.PublicKey))
		})
	}

	t.Run("wrong challenge is a FAILURE", func(t *testing.T) {
		srv := sceptest.NewTestSCEPServer(t, sceptest.WithIssuance(), sceptest.WithChallenge(challenge))
		_, err := enroll(t, srv.URL+"/scep", newCSR(t, ecKey, "wrong"))
		rejected, ok := errors.AsType[enrollment.RejectedError](err)
		require.True(t, ok, "got %v", err)
		require.Equal(t, smallstepscep.FAILURE, rejected.Status)
		require.EqualError(t, err, "SCEP server rejected the request: status FAILURE with fail info badRequest (2)")
	})
}

func TestEnrollCraftedResponses(t *testing.T) {
	csrKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	csr := newCSR(t, csrKey, "")

	t.Run("the certificate for the CSR's key is returned when the CA's certificate comes first", func(t *testing.T) {
		var issued *x509.Certificate
		srv := sceptest.NewCraftedServer(t, func(s *sceptest.CraftedServer, req *smallstepscep.PKIMessage) ([]byte, error) {
			issued = s.Issue(t, req)
			return s.CertRep(t, req, smallstepscep.SUCCESS, []*x509.Certificate{s.CACert, issued}), nil
		})
		cert, err := enroll(t, srv.URL, csr)
		require.NoError(t, err)
		require.Equal(t, issued.Raw, cert.Raw)
	})

	t.Run("a certificate for another key is rejected", func(t *testing.T) {
		other, _ := sceptest.NewSelfSignedCert(t, "someone else", x509.KeyUsageDigitalSignature)
		srv := sceptest.NewCraftedServer(t, func(s *sceptest.CraftedServer, req *smallstepscep.PKIMessage) ([]byte, error) {
			return s.CertRep(t, req, smallstepscep.SUCCESS, []*x509.Certificate{other}), nil
		})
		_, err := enroll(t, srv.URL, csr)
		require.EqualError(t, err, "SCEP CertRep has no certificate for the CSR's public key")
	})

	t.Run("an empty certificate bundle is an error, not a panic", func(t *testing.T) {
		srv := sceptest.NewCraftedServer(t, func(s *sceptest.CraftedServer, req *smallstepscep.PKIMessage) ([]byte, error) {
			return s.CertRep(t, req, smallstepscep.SUCCESS, nil), nil
		})
		_, err := enroll(t, srv.URL, csr)
		require.EqualError(t, err, "SCEP CertRep has no certificate for the CSR's public key")
	})

	t.Run("PENDING is a rejection", func(t *testing.T) {
		srv := sceptest.NewCraftedServer(t, func(s *sceptest.CraftedServer, req *smallstepscep.PKIMessage) ([]byte, error) {
			return s.CertRep(t, req, smallstepscep.PENDING, nil), nil
		})
		_, err := enroll(t, srv.URL, csr)
		rejected, ok := errors.AsType[enrollment.RejectedError](err)
		require.True(t, ok, "got %v", err)
		require.Equal(t, smallstepscep.PENDING, rejected.Status)
	})

	t.Run("an unknown fail info is an error, not a panic", func(t *testing.T) {
		srv := sceptest.NewCraftedServer(t, func(s *sceptest.CraftedServer, req *smallstepscep.PKIMessage) ([]byte, error) {
			return s.Fail(req, "99")
		})
		_, err := enroll(t, srv.URL, csr)
		require.EqualError(t, err, `SCEP server rejected the request: status FAILURE with fail info "99"`)
	})

	t.Run("a message other than CertRep is an error, not a panic", func(t *testing.T) {
		srv := sceptest.NewCraftedServer(t, func(s *sceptest.CraftedServer, req *smallstepscep.PKIMessage) ([]byte, error) {
			msg, err := smallstepscep.NewCSRRequest(req.CSRReqMessage.CSR, &smallstepscep.PKIMessage{
				MessageType: smallstepscep.PKCSReq,
				Recipients:  []*x509.Certificate{s.CACert},
				SignerKey:   s.CAKey,
				SignerCert:  s.CACert,
			})
			if err != nil {
				return nil, err
			}
			return msg.Raw, nil
		})
		_, err := enroll(t, srv.URL, csr)
		require.ErrorContains(t, err, "instead of CertRep")
		_, rejected := errors.AsType[enrollment.RejectedError](err)
		require.False(t, rejected)
	})

	t.Run("a CertRep encrypted to another signer fails to decrypt", func(t *testing.T) {
		otherCert, otherKey := sceptest.NewSelfSignedCert(t, "other signer", x509.KeyUsageKeyEncipherment|x509.KeyUsageDigitalSignature)
		srv := sceptest.NewCraftedServer(t, func(s *sceptest.CraftedServer, req *smallstepscep.PKIMessage) ([]byte, error) {
			other, err := smallstepscep.NewCSRRequest(req.CSRReqMessage.CSR, &smallstepscep.PKIMessage{
				MessageType: smallstepscep.PKCSReq,
				Recipients:  []*x509.Certificate{s.CACert},
				SignerKey:   otherKey,
				SignerCert:  otherCert,
			})
			if err != nil {
				return nil, err
			}
			parsed, err := smallstepscep.ParsePKIMessage(other.Raw)
			if err != nil {
				return nil, err
			}
			if err := parsed.DecryptPKIEnvelope(s.CACert, s.CAKey); err != nil {
				return nil, err
			}
			return s.Succeed(t, parsed)
		})
		_, err := enroll(t, srv.URL, csr)
		require.ErrorContains(t, err, "decrypting SCEP CertRep")
	})

	t.Run("a CertRep signed by a certificate outside the CA chain is rejected", func(t *testing.T) {
		otherCA, otherKey := sceptest.NewSelfSignedCert(t, "other CA", x509.KeyUsageCertSign|x509.KeyUsageDigitalSignature)
		srv := sceptest.NewCraftedServer(t, func(s *sceptest.CraftedServer, req *smallstepscep.PKIMessage) ([]byte, error) {
			rep, err := req.Success(otherCA, otherKey, s.Issue(t, req))
			if err != nil {
				return nil, err
			}
			return rep.Raw, nil
		})
		_, err := enroll(t, srv.URL, csr)
		require.ErrorContains(t, err, "parsing SCEP CertRep")
	})
}

// The FAILURE and PENDING messages are asserted end to end above and in the server client's tests.
func TestRejectedErrorUnknownStatus(t *testing.T) {
	require.EqualError(t, enrollment.RejectedError{Status: "7"}, `unknown status "7"`)
}

func TestFetchCACerts(t *testing.T) {
	t.Run("a body that is neither DER nor PKCS7 is an error", func(t *testing.T) {
		srv := sceptest.NewCraftedServer(t, nil, sceptest.WithCACertResponse(func(*sceptest.CraftedServer) ([]byte, int) {
			return []byte("not a certificate"), 1
		}))
		_, err := enrollment.FetchCACerts(t.Context(), newClient(t, srv.URL))
		require.ErrorContains(t, err, "parsing CA certificates from SCEP URL")
	})

	t.Run("a PKCS7 chain is returned in order", func(t *testing.T) {
		srv := sceptest.NewTestSCEPServer(t, sceptest.WithRAChain())
		certs, err := enrollment.FetchCACerts(t.Context(), newClient(t, srv.URL+"/scep"))
		require.NoError(t, err)
		require.Len(t, certs, 2)
		require.Equal(t, "Test NDES RA", certs[0].Subject.CommonName)
		require.Equal(t, sceptest.CACertificate(t).Raw, certs[1].Raw)
	})

	t.Run("request failures are RequestErrors carrying the HTTP status", func(t *testing.T) {
		closed := httptest.NewServer(http.NotFoundHandler())
		closed.Close()
		_, err := enrollment.FetchCACerts(t.Context(), newClient(t, closed.URL+"/scep"))
		reqErr, ok := errors.AsType[enrollment.RequestError](err)
		require.True(t, ok, "got %v", err)
		require.Equal(t, "getting CA certificates from SCEP URL", reqErr.Op)
		_, hasStatus := errors.AsType[scepserver.ResponseStatusError](err)
		require.False(t, hasStatus)

		unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		t.Cleanup(unavailable.Close)
		_, err = enrollment.FetchCACerts(t.Context(), newClient(t, unavailable.URL+"/scep"))
		_, ok = errors.AsType[enrollment.RequestError](err)
		require.True(t, ok, "got %v", err)
		statusErr, ok := errors.AsType[scepserver.ResponseStatusError](err)
		require.True(t, ok, "got %v", err)
		require.Equal(t, http.StatusServiceUnavailable, statusErr.Code)
	})
}

func TestRecipientCertsSelector(t *testing.T) {
	signOnly, _ := sceptest.NewSelfSignedCert(t, "sign only", x509.KeyUsageCertSign|x509.KeyUsageDigitalSignature)
	encipher, _ := sceptest.NewSelfSignedCert(t, "encipher", x509.KeyUsageKeyEncipherment)

	selector := enrollment.RecipientCertsSelector()
	require.Equal(t, []*x509.Certificate{encipher}, selector.SelectCerts([]*x509.Certificate{signOnly, encipher}))
	require.Equal(t, []*x509.Certificate{signOnly}, selector.SelectCerts([]*x509.Certificate{signOnly}))
}
