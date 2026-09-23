package scep

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/x509util"
	"github.com/gorilla/mux"
	"github.com/smallstep/pkcs7"
	smallstepscep "github.com/smallstep/scep"
	"github.com/stretchr/testify/require"
)

func TestEnrollmentRejectedError(t *testing.T) {
	for name, tc := range map[string]struct {
		err  enrollmentRejectedError
		want string
	}{
		// FailInfo.String panics on values outside the RFC set.
		"failure with unknown fail info": {
			enrollmentRejectedError{Status: smallstepscep.FAILURE, FailInfo: "99"},
			`status FAILURE with fail info "99"; if this certificate authority requires a challenge, include it as the CSR's challengePassword attribute`,
		},
		"pending": {
			enrollmentRejectedError{Status: smallstepscep.PENDING},
			"status PENDING; requests that need manual approval are not supported",
		},
		"unknown status": {
			enrollmentRejectedError{Status: "7"},
			`unknown status "7"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.err.Error())
		})
	}
}

// craftedSCEPService answers PKIOperation with whatever respond returns.
type craftedSCEPService struct {
	caCert *x509.Certificate
	caKey  *rsa.PrivateKey
	// caCertResponse overrides GetCACert; a count above 1 is served as a chain.
	caCertResponse func() ([]byte, int)
	respond        func(req *smallstepscep.PKIMessage) ([]byte, error)
}

func (s *craftedSCEPService) GetCACaps(context.Context) ([]byte, error) {
	return []byte(scepserver.DefaultCACaps), nil
}

func (s *craftedSCEPService) GetCACert(context.Context, string) ([]byte, int, error) {
	if s.caCertResponse != nil {
		data, n := s.caCertResponse()
		return data, n, nil
	}
	return s.caCert.Raw, 1, nil
}

func (s *craftedSCEPService) PKIOperation(_ context.Context, data []byte) ([]byte, error) {
	req, err := smallstepscep.ParsePKIMessage(data)
	if err != nil {
		return nil, err
	}
	if err := req.DecryptPKIEnvelope(s.caCert, s.caKey); err != nil {
		return nil, err
	}
	return s.respond(req)
}

func (s *craftedSCEPService) GetNextCACert(context.Context) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func newSelfSignedTestCert(t *testing.T, cn string, keyUsage x509.KeyUsage) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              keyUsage,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert, key
}

// pendingCertRep builds a PENDING CertRep, which the SCEP library has no constructor for.
func pendingCertRep(t *testing.T, req *smallstepscep.PKIMessage, caCert *x509.Certificate, caKey *rsa.PrivateKey) []byte {
	t.Helper()
	scepOID := func(n int) asn1.ObjectIdentifier { return asn1.ObjectIdentifier{2, 16, 840, 1, 113733, 1, 9, n} }
	sd, err := pkcs7.NewSignedData(nil)
	require.NoError(t, err)
	require.NoError(t, sd.AddSigner(caCert, caKey, pkcs7.SignerInfoConfig{ExtraSignedAttributes: []pkcs7.Attribute{
		{Type: scepOID(7), Value: req.TransactionID},
		{Type: scepOID(3), Value: smallstepscep.PENDING},
		{Type: scepOID(2), Value: smallstepscep.CertRep},
		{Type: scepOID(5), Value: req.SenderNonce},
		{Type: scepOID(6), Value: req.SenderNonce},
	}}))
	raw, err := sd.Finish()
	require.NoError(t, err)
	return raw
}

// statusServer returns a SCEP URL whose server answers every request with status.
func statusServer(t *testing.T, status int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
	t.Cleanup(srv.Close)
	return srv.URL + "/scep"
}

// successCertRep builds a SUCCESS CertRep carrying certs; PKIMessage.Success sends only one.
func successCertRep(t *testing.T, req *smallstepscep.PKIMessage, caCert *x509.Certificate, caKey *rsa.PrivateKey, certs []*x509.Certificate) []byte {
	t.Helper()
	var chain []byte
	for _, c := range certs {
		chain = append(chain, c.Raw...)
	}
	degenerate, err := pkcs7.DegenerateCertificate(chain)
	require.NoError(t, err)
	reqP7, err := pkcs7.Parse(req.Raw)
	require.NoError(t, err)
	enveloped, err := pkcs7.Encrypt(degenerate, reqP7.Certificates)
	require.NoError(t, err)

	scepOID := func(n int) asn1.ObjectIdentifier { return asn1.ObjectIdentifier{2, 16, 840, 1, 113733, 1, 9, n} }
	sd, err := pkcs7.NewSignedData(enveloped)
	require.NoError(t, err)
	require.NoError(t, sd.AddSigner(caCert, caKey, pkcs7.SignerInfoConfig{ExtraSignedAttributes: []pkcs7.Attribute{
		{Type: scepOID(7), Value: req.TransactionID},
		{Type: scepOID(3), Value: smallstepscep.SUCCESS},
		{Type: scepOID(2), Value: smallstepscep.CertRep},
		{Type: scepOID(5), Value: req.SenderNonce},
		{Type: scepOID(6), Value: req.SenderNonce},
	}}))
	raw, err := sd.Finish()
	require.NoError(t, err)
	return raw
}

func TestEnrollmentClientGetCertificate(t *testing.T) {
	caCert, caKey := newSelfSignedTestCert(t, "Crafted SCEP CA",
		x509.KeyUsageCertSign|x509.KeyUsageKeyEncipherment|x509.KeyUsageDigitalSignature)

	issue := func(t *testing.T, req *smallstepscep.PKIMessage) *x509.Certificate {
		t.Helper()
		template := &x509.Certificate{
			SerialNumber: big.NewInt(2),
			Subject:      req.CSRReqMessage.CSR.Subject,
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
		}
		der, err := x509.CreateCertificate(rand.Reader, template, caCert, req.CSRReqMessage.CSR.PublicKey, caKey)
		require.NoError(t, err)
		cert, err := x509.ParseCertificate(der)
		require.NoError(t, err)
		return cert
	}

	newServerWithCACert := func(t *testing.T, caCertResponse func() ([]byte, int), respond func(req *smallstepscep.PKIMessage) ([]byte, error)) string {
		t.Helper()
		svc := &craftedSCEPService{caCert: caCert, caKey: caKey, caCertResponse: caCertResponse, respond: respond}
		r := mux.NewRouter()
		r.Handle("/scep", scepserver.MakeHTTPHandler(scepserver.MakeServerEndpoints(svc), svc, slog.New(slog.DiscardHandler)))
		server := httptest.NewServer(r)
		t.Cleanup(server.Close)
		return server.URL + "/scep"
	}
	newServer := func(t *testing.T, respond func(req *smallstepscep.PKIMessage) ([]byte, error)) string {
		t.Helper()
		return newServerWithCACert(t, nil, respond)
	}

	csrKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	csrDER, err := x509util.CreateCertificateRequest(rand.Reader, &x509util.CertificateRequest{
		Subject:           pkix.Name{CommonName: "test-host managementAttestation"},
		ChallengePassword: "challenge",
	}, csrKey)
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(csrDER)
	require.NoError(t, err)

	newClient := func() *EnrollmentClient {
		return NewEnrollmentClient(slog.New(slog.DiscardHandler))
	}

	succeed := func(req *smallstepscep.PKIMessage) ([]byte, error) {
		rep, err := req.Success(caCert, caKey, issue(t, req))
		if err != nil {
			return nil, err
		}
		return rep.Raw, nil
	}

	t.Run("request failures that clear on their own are CA transient, others are not", func(t *testing.T) {
		closed := httptest.NewServer(http.NotFoundHandler())
		closed.Close()
		for name, tc := range map[string]struct {
			url           string
			wantTransient bool
		}{
			"no response": {url: closed.URL + "/scep", wantTransient: true},
			"HTTP 503":    {url: statusServer(t, http.StatusServiceUnavailable), wantTransient: true},
			"HTTP 429":    {url: statusServer(t, http.StatusTooManyRequests), wantTransient: true},
			"HTTP 404":    {url: statusServer(t, http.StatusNotFound), wantTransient: false},
			"HTTP 401":    {url: statusServer(t, http.StatusUnauthorized), wantTransient: false},
		} {
			t.Run(name, func(t *testing.T) {
				_, err := newClient().GetCertificate(t.Context(), tc.url, csr)
				require.ErrorContains(t, err, "getting CA certificates from SCEP URL")
				transient, ok := errors.AsType[fleet.CertificateAuthorityTransientError](err)
				require.Equal(t, tc.wantTransient, ok)
				if ok {
					require.Equal(t, enrollmentRetryAfterSeconds, transient.RetryAfterSeconds)
				}
			})
		}
	})

	t.Run("GetCACert body that is neither is an error", func(t *testing.T) {
		url := newServerWithCACert(t, func() ([]byte, int) { return []byte("not a certificate"), 1 }, succeed)
		_, err := newClient().GetCertificate(t.Context(), url, csr)
		require.ErrorContains(t, err, "parsing CA certificates from SCEP URL")
	})

	t.Run("the certificate for the CSR's key is returned when the CA's certificate comes first", func(t *testing.T) {
		var issued *x509.Certificate
		url := newServer(t, func(req *smallstepscep.PKIMessage) ([]byte, error) {
			issued = issue(t, req)
			return successCertRep(t, req, caCert, caKey, []*x509.Certificate{caCert, issued}), nil
		})
		cert, err := newClient().GetCertificate(t.Context(), url, csr)
		require.NoError(t, err)
		require.Equal(t, issued.Raw, cert.Raw)
	})

	t.Run("a certificate for another key is rejected", func(t *testing.T) {
		other, _ := newSelfSignedTestCert(t, "someone else", x509.KeyUsageDigitalSignature)
		url := newServer(t, func(req *smallstepscep.PKIMessage) ([]byte, error) {
			return successCertRep(t, req, caCert, caKey, []*x509.Certificate{other}), nil
		})
		_, err := newClient().GetCertificate(t.Context(), url, csr)
		require.ErrorContains(t, err, "SCEP CertRep has no certificate for the CSR's public key")
	})

	t.Run("an empty certificate bundle is an error, not a panic", func(t *testing.T) {
		url := newServer(t, func(req *smallstepscep.PKIMessage) ([]byte, error) {
			return successCertRep(t, req, caCert, caKey, nil), nil
		})
		_, err := newClient().GetCertificate(t.Context(), url, csr)
		require.ErrorContains(t, err, "SCEP CertRep has no certificate for the CSR's public key")
	})

	t.Run("PENDING is a rejection", func(t *testing.T) {
		url := newServer(t, func(req *smallstepscep.PKIMessage) ([]byte, error) {
			return pendingCertRep(t, req, caCert, caKey), nil
		})
		_, err := newClient().GetCertificate(t.Context(), url, csr)
		rejected, ok := errors.AsType[enrollmentRejectedError](err)
		require.True(t, ok, "got %v", err)
		require.Equal(t, smallstepscep.PENDING, rejected.Status)
	})

	t.Run("a message other than CertRep is an error, not a panic", func(t *testing.T) {
		url := newServer(t, func(req *smallstepscep.PKIMessage) ([]byte, error) {
			msg, err := smallstepscep.NewCSRRequest(req.CSRReqMessage.CSR, &smallstepscep.PKIMessage{
				MessageType: smallstepscep.PKCSReq,
				Recipients:  []*x509.Certificate{caCert},
				SignerKey:   caKey,
				SignerCert:  caCert,
			})
			if err != nil {
				return nil, err
			}
			return msg.Raw, nil
		})
		_, err := newClient().GetCertificate(t.Context(), url, csr)
		require.ErrorContains(t, err, "instead of CertRep")
		_, rejected := errors.AsType[enrollmentRejectedError](err)
		require.False(t, rejected)
	})

	t.Run("a CertRep encrypted to another signer fails to decrypt", func(t *testing.T) {
		otherCert, otherKey := newSelfSignedTestCert(t, "other signer", x509.KeyUsageKeyEncipherment|x509.KeyUsageDigitalSignature)
		url := newServer(t, func(req *smallstepscep.PKIMessage) ([]byte, error) {
			other, err := smallstepscep.NewCSRRequest(req.CSRReqMessage.CSR, &smallstepscep.PKIMessage{
				MessageType: smallstepscep.PKCSReq,
				Recipients:  []*x509.Certificate{caCert},
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
			if err := parsed.DecryptPKIEnvelope(caCert, caKey); err != nil {
				return nil, err
			}
			rep, err := parsed.Success(caCert, caKey, issue(t, req))
			if err != nil {
				return nil, err
			}
			return rep.Raw, nil
		})
		_, err := newClient().GetCertificate(t.Context(), url, csr)
		require.ErrorContains(t, err, "decrypting SCEP CertRep")
	})
}
