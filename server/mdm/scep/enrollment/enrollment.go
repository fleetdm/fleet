// Package enrollment is the client side of a SCEP PKCSReq exchange, shared by the Fleet server,
// fleetd, and osquery-perf: fetch the CA certificates, send a CSR, and return the certificate the
// CA issued for it. Callers build the CSR, the signer, and the HTTP client (TLS options, timeout).
package enrollment

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"log/slog"
	"time"

	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/kitlogadapter"
	"github.com/smallstep/pkcs7"
	smallstepscep "github.com/smallstep/scep"
)

// Request is one PKCSReq.
type Request struct {
	// CSR is sent as-is. Its key does not have to be the signer's.
	CSR *x509.CertificateRequest
	// SignerKey and SignerCert sign the PKCSReq, and the CA encrypts the CertRep to SignerCert, so
	// the key must be RSA.
	SignerKey  *rsa.PrivateKey
	SignerCert *x509.Certificate
	// Logger receives the SCEP library's debug logs. Nil discards them.
	Logger *slog.Logger
}

// RequestError is a GetCACert or PKIOperation request that failed, either without a response or
// with an HTTP error status (a scepserver.ResponseStatusError in the chain).
type RequestError struct {
	Op  string
	Err error
}

func (e RequestError) Error() string { return e.Op + ": " + e.Err.Error() }

func (e RequestError) Unwrap() error { return e.Err }

// RejectedError is a CertRep whose status is not SUCCESS: the CA answered and declined, or
// deferred, the request.
type RejectedError struct {
	Status   smallstepscep.PKIStatus
	FailInfo smallstepscep.FailInfo
}

func (e RejectedError) Error() string {
	switch e.Status {
	case smallstepscep.FAILURE:
		// FailInfo.String panics on values outside the RFC 8894 set, so those are printed raw.
		failInfo := fmt.Sprintf("%q", string(e.FailInfo))
		switch e.FailInfo {
		case smallstepscep.BadAlg, smallstepscep.BadMessageCheck, smallstepscep.BadRequest, smallstepscep.BadTime, smallstepscep.BadCertID:
			failInfo = e.FailInfo.String()
		}
		return "status FAILURE with fail info " + failInfo
	case smallstepscep.PENDING:
		return "status PENDING; requests that need manual approval are not supported"
	default:
		return fmt.Sprintf("unknown status %q", string(e.Status))
	}
}

// FetchCACerts returns the SCEP server's GetCACert chain. A lone CA certificate is served as raw
// DER; a CA with an RA, the usual NDES setup, serves a PKCS7 chain. Each parser rejects the
// other format, so both are tried rather than trusting the Content-Type, which servers mislabel.
func FetchCACerts(ctx context.Context, client scepclient.Client) ([]*x509.Certificate, error) {
	data, _, err := client.GetCACert(ctx, "")
	if err != nil {
		return nil, RequestError{Op: "getting CA certificates from SCEP URL", Err: err}
	}
	caCerts, err := x509.ParseCertificates(data)
	if err != nil {
		if caCerts, err = smallstepscep.CACerts(data); err != nil {
			return nil, fmt.Errorf("parsing CA certificates from SCEP URL: %w", err)
		}
	}
	if len(caCerts) == 0 {
		return nil, errors.New("SCEP URL did not return a CA certificate")
	}
	return caCerts, nil
}

// Enroll sends req.CSR as a PKCSReq to the SCEP server whose GetCACert chain is caCerts and
// returns the certificate issued for the CSR's public key.
func Enroll(ctx context.Context, client scepclient.Client, caCerts []*x509.Certificate, req Request) (*x509.Certificate, error) {
	logger := req.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	scepLogger := kitlogadapter.NewLogger(logger)
	msg, err := smallstepscep.NewCSRRequest(req.CSR, &smallstepscep.PKIMessage{
		MessageType: smallstepscep.PKCSReq,
		Recipients:  caCerts,
		SignerKey:   req.SignerKey,
		SignerCert:  req.SignerCert,
	}, smallstepscep.WithLogger(scepLogger), smallstepscep.WithCertsSelector(RecipientCertsSelector()))
	if err != nil {
		return nil, fmt.Errorf("creating SCEP PKCSReq: %w", err)
	}

	respBytes, err := client.PKIOperation(ctx, msg.Raw)
	if err != nil {
		return nil, RequestError{Op: "sending SCEP PKCSReq", Err: err}
	}
	// Verified against the whole GetCACert chain, not just the recipients: NDES signs the CertRep
	// with its signing RA certificate, which the encipherment selector leaves out.
	resp, err := smallstepscep.ParsePKIMessage(respBytes, smallstepscep.WithLogger(scepLogger), smallstepscep.WithCACerts(caCerts))
	if err != nil {
		return nil, fmt.Errorf("parsing SCEP CertRep: %w", err)
	}
	// PKIStatus is promoted from CertRepMessage, which is nil for any other message type.
	if resp.CertRepMessage == nil {
		return nil, fmt.Errorf("SCEP server responded with message type %q instead of CertRep", string(resp.MessageType))
	}
	if resp.PKIStatus != smallstepscep.SUCCESS {
		return nil, fmt.Errorf("SCEP server rejected the request: %w", RejectedError{Status: resp.PKIStatus, FailInfo: resp.FailInfo})
	}
	certs, err := decryptCertRepCertificates(respBytes, req.SignerCert, req.SignerKey)
	if err != nil {
		return nil, fmt.Errorf("decrypting SCEP CertRep: %w", err)
	}
	return certificateForCSR(certs, req.CSR)
}

// decryptCertRepCertificates returns every certificate in a CertRep that ParsePKIMessage has already
// verified. PKIMessage.DecryptPKIEnvelope keeps only the first certificate, and panics when there
// is none, while RFC 8894 lets a CA send its own certificates alongside the issued one in any order.
func decryptCertRepCertificates(certRep []byte, signerCert *x509.Certificate, signerKey *rsa.PrivateKey) ([]*x509.Certificate, error) {
	signed, err := pkcs7.Parse(certRep)
	if err != nil {
		return nil, err
	}
	enveloped, err := pkcs7.Parse(signed.Content)
	if err != nil {
		return nil, err
	}
	degenerate, err := enveloped.Decrypt(signerCert, signerKey)
	if err != nil {
		return nil, err
	}
	return smallstepscep.CACerts(degenerate)
}

// certificateForCSR returns the certificate issued for csr's public key, so a CA certificate sent
// alongside it, or a certificate for another key, is never handed back as the caller's.
func certificateForCSR(certs []*x509.Certificate, csr *x509.CertificateRequest) (*x509.Certificate, error) {
	csrKey, err := x509.MarshalPKIXPublicKey(csr.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("encoding CSR public key: %w", err)
	}
	for _, cert := range certs {
		certKey, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
		if err == nil && bytes.Equal(certKey, csrKey) {
			return cert, nil
		}
	}
	return nil, errors.New("SCEP CertRep has no certificate for the CSR's public key")
}

// NewEphemeralSigner returns a fresh RSA key with a certificate from NewSignerCert, to sign one
// PKCSReq when the CSR's own key cannot: it is not RSA or not held by the caller.
func NewEphemeralSigner(subject pkix.Name) (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generating SCEP signer key: %w", err)
	}
	cert, err := NewSignerCert(key, subject)
	if err != nil {
		return nil, nil, err
	}
	return key, cert, nil
}

// NewSignerCert returns a short-lived self-signed certificate for key to sign one PKCSReq, which
// RFC 8894 section 2.3 allows. The certificate is never issued to anyone; the CA only uses it to
// verify the request and encrypt the CertRep.
func NewSignerCert(key *rsa.PrivateKey, subject pkix.Name) (*x509.Certificate, error) {
	now := time.Now()
	template := &x509.Certificate{
		Subject:               subject,
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.Add(10 * time.Minute),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("creating SCEP signer certificate: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parsing SCEP signer certificate: %w", err)
	}
	return cert, nil
}

// RecipientCertsSelector picks the RA encryption certificate from a GetCACert chain. A CA that
// serves no RA and decrypts with its own certificate often lacks the encipherment key usage, so
// the whole chain is used when nothing qualifies.
func RecipientCertsSelector() smallstepscep.CertsSelectorFunc {
	encipherment := smallstepscep.EnciphermentCertsSelector()
	return func(certs []*x509.Certificate) []*x509.Certificate {
		if selected := encipherment(certs); len(selected) > 0 {
			return selected
		}
		return certs
	}
}
