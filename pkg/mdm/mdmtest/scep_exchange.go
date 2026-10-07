package mdmtest

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"log/slog"
	"time"

	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/enrollment"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/x509util"
	smallstepscep "github.com/smallstep/scep"
)

// scepExchangeRequest describes a single SCEP enrollment exchange driven by a test MDM client.
// Used internally by performSCEPExchange to unify the Apple and Windows test paths.
type scepExchangeRequest struct {
	URL       string
	Subject   pkix.Name
	Challenge string
	KeyBits   int // defaults to 2048 if 0
}

// performSCEPExchange runs the full SCEP CSR-then-cert exchange against the SCEP server at
// req.URL and returns the issued device certificate together with the RSA private key the cert
// was bound to. It is shared between the Apple and Windows test MDM clients.
//
// This is test-only: TLS verification on the SCEP server is skipped, and the CSR is always
// signed with SHA-256 RSA. The signer envelope cert is a short-lived self-signed cert wrapping
// the same RSA key, per RFC 8894 §2.4 for first-time SCEP enrollment.
func performSCEPExchange(
	ctx context.Context,
	req scepExchangeRequest,
	logger *slog.Logger,
) (*x509.Certificate, *rsa.PrivateKey, error) {
	if req.URL == "" {
		return nil, nil, errors.New("scep exchange: missing server URL")
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	keyBits := req.KeyBits
	if keyBits <= 0 {
		keyBits = 2048
	}

	timeout := 30 * time.Second
	client, err := scepclient.New(req.URL, logger,
		scepclient.WithTimeout(&timeout),
		scepclient.Insecure(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("scep exchange: create client: %w", err)
	}

	caCerts, err := enrollment.FetchCACerts(ctx, client)
	if err != nil {
		return nil, nil, fmt.Errorf("scep exchange: %w", err)
	}

	privKey, err := rsa.GenerateKey(cryptorand.Reader, keyBits)
	if err != nil {
		return nil, nil, fmt.Errorf("scep exchange: generate rsa key: %w", err)
	}

	csrTpl := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject:            req.Subject,
			SignatureAlgorithm: x509.SHA256WithRSA,
		},
		ChallengePassword: req.Challenge,
	}
	csrDER, err := x509util.CreateCertificateRequest(cryptorand.Reader, &csrTpl, privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("scep exchange: create csr: %w", err)
	}
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return nil, nil, fmt.Errorf("scep exchange: parse csr: %w", err)
	}

	signerCert, err := enrollment.NewSignerCert(privKey, req.Subject)
	if err != nil {
		return nil, nil, fmt.Errorf("scep exchange: %w", err)
	}

	cert, err := enrollment.Enroll(ctx, client, caCerts, enrollment.Request{
		CSR:        csr,
		SignerKey:  privKey,
		SignerCert: signerCert,
		Logger:     logger,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("scep exchange: %w", err)
	}
	return cert, privKey, nil
}

// NewPKCSReqUndecryptableBy builds a signed PKCSReq whose envelope is addressed
// to caCert (same issuer and serial) but encrypted under a freshly generated RSA
// key, so a SCEP server holding caCert's private key selects the recipient and
// then fails to decrypt. The returned message carries the transaction ID so
// callers can match it against the server's CertRep.
func NewPKCSReqUndecryptableBy(caCert *x509.Certificate) (*smallstepscep.PKIMessage, error) {
	requesterKey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("undecryptable pkcsreq: generate requester key: %w", err)
	}
	subject := pkix.Name{CommonName: "undecryptable-pkcsreq"}
	csrDER, err := x509.CreateCertificateRequest(cryptorand.Reader, &x509.CertificateRequest{
		Subject:            subject,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}, requesterKey)
	if err != nil {
		return nil, fmt.Errorf("undecryptable pkcsreq: create csr: %w", err)
	}
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return nil, fmt.Errorf("undecryptable pkcsreq: parse csr: %w", err)
	}
	signerCert, err := enrollment.NewSignerCert(requesterKey, subject)
	if err != nil {
		return nil, fmt.Errorf("undecryptable pkcsreq: %w", err)
	}
	recipient, err := certWithSameIssuerAndSerialAs(caCert)
	if err != nil {
		return nil, fmt.Errorf("undecryptable pkcsreq: build recipient cert: %w", err)
	}
	return smallstepscep.NewCSRRequest(csr, &smallstepscep.PKIMessage{
		MessageType: smallstepscep.PKCSReq,
		Recipients:  []*x509.Certificate{recipient},
		SignerKey:   requesterKey,
		SignerCert:  signerCert,
	})
}

// certWithSameIssuerAndSerialAs returns a certificate that PKCS#7 recipient
// matching cannot tell apart from caCert, but whose RSA public key belongs to a
// throwaway private key that is discarded.
func certWithSameIssuerAndSerialAs(caCert *x509.Certificate) (*x509.Certificate, error) {
	keyBits := 2048
	if pub, ok := caCert.PublicKey.(*rsa.PublicKey); ok {
		keyBits = pub.N.BitLen()
	}
	key, err := rsa.GenerateKey(cryptorand.Reader, keyBits)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tpl := &x509.Certificate{
		SerialNumber: caCert.SerialNumber,
		Subject:      caCert.Subject,
		NotBefore:    now.Add(-1 * time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment,
	}
	// x509.CreateCertificate copies the issuer from parent.RawSubject verbatim,
	// which keeps the encoded issuer byte-identical to caCert's.
	parent := &x509.Certificate{RawSubject: caCert.RawIssuer}
	der, err := x509.CreateCertificate(cryptorand.Reader, tpl, parent, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(der)
}
