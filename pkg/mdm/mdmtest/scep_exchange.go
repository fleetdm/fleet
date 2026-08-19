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
	"math/big"
	"time"

	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
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

	caResp, _, err := client.GetCACert(ctx, "")
	if err != nil {
		return nil, nil, fmt.Errorf("scep exchange: get ca cert: %w", err)
	}
	caCerts, err := x509.ParseCertificates(caResp)
	if err != nil {
		return nil, nil, fmt.Errorf("scep exchange: parse ca certs: %w", err)
	}
	if len(caCerts) == 0 {
		return nil, nil, errors.New("scep exchange: server returned no ca certificates")
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

	signerCert, err := selfSignedSignerCert(privKey, req.Subject)
	if err != nil {
		return nil, nil, fmt.Errorf("scep exchange: build signer cert: %w", err)
	}

	pkiReq := &smallstepscep.PKIMessage{
		MessageType: smallstepscep.PKCSReq,
		Recipients:  caCerts,
		SignerKey:   privKey,
		SignerCert:  signerCert,
	}
	msg, err := smallstepscep.NewCSRRequest(csr, pkiReq)
	if err != nil {
		return nil, nil, fmt.Errorf("scep exchange: build pkcsreq: %w", err)
	}
	respBytes, err := client.PKIOperation(ctx, msg.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf("scep exchange: pki operation: %w", err)
	}
	pkiResp, err := smallstepscep.ParsePKIMessage(respBytes, smallstepscep.WithCACerts(msg.Recipients))
	if err != nil {
		return nil, nil, fmt.Errorf("scep exchange: parse pki response: %w", err)
	}
	if pkiResp.PKIStatus != smallstepscep.SUCCESS {
		return nil, nil, fmt.Errorf("scep exchange: pki status %v (failInfo=%v)", pkiResp.PKIStatus, pkiResp.FailInfo)
	}
	if err := pkiResp.DecryptPKIEnvelope(signerCert, privKey); err != nil {
		return nil, nil, fmt.Errorf("scep exchange: decrypt pki envelope: %w", err)
	}
	if pkiResp.CertRepMessage == nil || pkiResp.CertRepMessage.Certificate == nil {
		return nil, nil, errors.New("scep exchange: response contained no certificate")
	}
	return pkiResp.CertRepMessage.Certificate, privKey, nil
}

// selfSignedSignerCert builds a short-lived self-signed certificate wrapping key, used as the
// outer-envelope signer cert for first-time SCEP enrollment per RFC 8894 §2.4.
func selfSignedSignerCert(key *rsa.PrivateKey, subject pkix.Name) (*x509.Certificate, error) {
	now := time.Now()
	tpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               subject,
		NotBefore:             now.Add(-1 * time.Minute),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(cryptorand.Reader, &tpl, &tpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(der)
}
