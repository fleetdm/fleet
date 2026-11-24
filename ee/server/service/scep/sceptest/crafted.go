package sceptest

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"math/big"
	"testing"
	"time"

	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/smallstep/pkcs7"
	smallstepscep "github.com/smallstep/scep"
	"github.com/stretchr/testify/require"
)

// CraftedServer is a SCEP server backed by a self-signed CA whose PKIOperation answers each test
// crafts, for CertReps a real SCEP server never sends.
type CraftedServer struct {
	URL    string
	CACert *x509.Certificate
	CAKey  *rsa.PrivateKey

	caCertResponse func(s *CraftedServer) ([]byte, int)
	respond        func(s *CraftedServer, req *smallstepscep.PKIMessage) ([]byte, error)
}

// CraftedOption configures NewCraftedServer.
type CraftedOption func(*CraftedServer)

// WithCACertResponse overrides GetCACert; a count above 1 is served as a PKCS7 chain.
func WithCACertResponse(f func(s *CraftedServer) ([]byte, int)) CraftedOption {
	return func(s *CraftedServer) { s.caCertResponse = f }
}

// NewCraftedServer starts a SCEP server that serves its CA certificate from GetCACert and answers
// each decrypted PKIOperation with respond. A nil respond issues the certificate (see Succeed).
func NewCraftedServer(t *testing.T, respond func(s *CraftedServer, req *smallstepscep.PKIMessage) ([]byte, error), opts ...CraftedOption) *CraftedServer {
	t.Helper()
	s := &CraftedServer{respond: respond}
	for _, opt := range opts {
		opt(s)
	}
	s.CACert, s.CAKey = NewSelfSignedCert(t, "Crafted SCEP CA",
		x509.KeyUsageCertSign|x509.KeyUsageKeyEncipherment|x509.KeyUsageDigitalSignature)
	if s.respond == nil {
		s.respond = func(s *CraftedServer, req *smallstepscep.PKIMessage) ([]byte, error) { return s.Succeed(t, req) }
	}
	s.URL = newSCEPHTTPServer(t, &craftedService{s}).URL + "/scep"
	return s
}

// Issue returns a certificate from the server's CA for req's CSR.
func (s *CraftedServer) Issue(t *testing.T, req *smallstepscep.PKIMessage) *x509.Certificate {
	t.Helper()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      req.CSRReqMessage.CSR.Subject,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, s.CACert, req.CSRReqMessage.CSR.PublicKey, s.CAKey)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}

// Succeed answers req with a SUCCESS CertRep carrying the certificate issued for its CSR.
func (s *CraftedServer) Succeed(t *testing.T, req *smallstepscep.PKIMessage) ([]byte, error) {
	rep, err := req.Success(s.CACert, s.CAKey, s.Issue(t, req))
	if err != nil {
		return nil, err
	}
	return rep.Raw, nil
}

// Fail answers req with a FAILURE CertRep.
func (s *CraftedServer) Fail(req *smallstepscep.PKIMessage, failInfo smallstepscep.FailInfo) ([]byte, error) {
	rep, err := req.Fail(s.CACert, s.CAKey, failInfo)
	if err != nil {
		return nil, err
	}
	return rep.Raw, nil
}

// CertRep builds a CertRep signed by the server's CA with status, which PKIMessage.Success and Fail
// cannot produce: a PENDING one, an unknown status, or a SUCCESS one carrying any set of certs.
func (s *CraftedServer) CertRep(t *testing.T, req *smallstepscep.PKIMessage, status smallstepscep.PKIStatus, certs []*x509.Certificate) []byte {
	t.Helper()
	var content []byte
	if status == smallstepscep.SUCCESS {
		var chain []byte
		for _, c := range certs {
			chain = append(chain, c.Raw...)
		}
		degenerate, err := pkcs7.DegenerateCertificate(chain)
		require.NoError(t, err)
		reqP7, err := pkcs7.Parse(req.Raw)
		require.NoError(t, err)
		content, err = pkcs7.Encrypt(degenerate, reqP7.Certificates)
		require.NoError(t, err)
	}

	scepOID := func(n int) asn1.ObjectIdentifier { return asn1.ObjectIdentifier{2, 16, 840, 1, 113733, 1, 9, n} }
	sd, err := pkcs7.NewSignedData(content)
	require.NoError(t, err)
	require.NoError(t, sd.AddSigner(s.CACert, s.CAKey, pkcs7.SignerInfoConfig{ExtraSignedAttributes: []pkcs7.Attribute{
		{Type: scepOID(7), Value: req.TransactionID},
		{Type: scepOID(3), Value: status},
		{Type: scepOID(2), Value: smallstepscep.CertRep},
		{Type: scepOID(5), Value: req.SenderNonce},
		{Type: scepOID(6), Value: req.SenderNonce},
	}}))
	raw, err := sd.Finish()
	require.NoError(t, err)
	return raw
}

// NewSelfSignedCert returns a self-signed CA certificate and its RSA key.
func NewSelfSignedCert(t *testing.T, cn string, keyUsage x509.KeyUsage) (*x509.Certificate, *rsa.PrivateKey) {
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

type craftedService struct{ s *CraftedServer }

func (c *craftedService) GetCACaps(context.Context) ([]byte, error) {
	return []byte(scepserver.DefaultCACaps), nil
}

func (c *craftedService) GetCACert(context.Context, string) ([]byte, int, error) {
	if c.s.caCertResponse != nil {
		data, n := c.s.caCertResponse(c.s)
		return data, n, nil
	}
	return c.s.CACert.Raw, 1, nil
}

func (c *craftedService) PKIOperation(_ context.Context, data []byte) ([]byte, error) {
	req, err := smallstepscep.ParsePKIMessage(data)
	if err != nil {
		return nil, err
	}
	if err := req.DecryptPKIEnvelope(c.s.CACert, c.s.CAKey); err != nil {
		return nil, err
	}
	return c.s.respond(c.s, req)
}

func (c *craftedService) GetNextCACert(context.Context) ([]byte, error) {
	return nil, errors.New("not implemented")
}
