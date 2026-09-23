package scep

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/kitlogadapter"
	"github.com/smallstep/pkcs7"
	smallstepscep "github.com/smallstep/scep"
)

// EnrollmentClient implements fleet.SCEPEnrollmentClient.
type EnrollmentClient struct {
	logger  *slog.Logger
	timeout *time.Duration
}

var _ fleet.SCEPEnrollmentClient = (*EnrollmentClient)(nil)

// NewEnrollmentClient returns an EnrollmentClient whose SCEP requests time out after 30 seconds.
func NewEnrollmentClient(logger *slog.Logger) *EnrollmentClient {
	return &EnrollmentClient{logger: logger, timeout: new(30 * time.Second)}
}

// GetCertificate enrolls csr against the SCEP server at url. The caller holds the CSR's private
// key, so the SCEP envelope is signed with an ephemeral key and self-signed certificate instead,
// which RFC 8894 section 2.3 allows. The CertRep is encrypted to that certificate.
func (c *EnrollmentClient) GetCertificate(ctx context.Context, url string, csr *x509.CertificateRequest) (*x509.Certificate, error) {
	client, err := scepclient.New(url, c.logger, scepclient.WithTimeout(c.timeout))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating SCEP client")
	}
	caCerts, err := fetchCACerts(ctx, client)
	if err != nil {
		return nil, err
	}
	signerKey, signerCert, err := newEphemeralSigner(ctx, csr.Subject)
	if err != nil {
		return nil, err
	}

	scepLogger := kitlogadapter.NewLogger(c.logger)
	msg, err := smallstepscep.NewCSRRequest(csr, &smallstepscep.PKIMessage{
		MessageType: smallstepscep.PKCSReq,
		Recipients:  caCerts,
		SignerKey:   signerKey,
		SignerCert:  signerCert,
	}, smallstepscep.WithLogger(scepLogger), smallstepscep.WithCertsSelector(recipientCertsSelector()))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating SCEP PKCSReq")
	}

	respBytes, err := client.PKIOperation(ctx, msg.Raw)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sending SCEP PKCSReq")
	}
	// Verified against the whole GetCACert chain, not just the recipients: NDES signs the CertRep
	// with its signing RA certificate, which the encipherment selector leaves out.
	resp, err := smallstepscep.ParsePKIMessage(respBytes, smallstepscep.WithLogger(scepLogger), smallstepscep.WithCACerts(caCerts))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parsing SCEP CertRep")
	}
	// PKIStatus is promoted from CertRepMessage, which is nil for any other message type.
	if resp.CertRepMessage == nil {
		return nil, ctxerr.Errorf(ctx, "SCEP server responded with message type %q instead of CertRep", string(resp.MessageType))
	}
	if resp.PKIStatus != smallstepscep.SUCCESS {
		return nil, ctxerr.Wrap(ctx, enrollmentRejectedError{Status: resp.PKIStatus, FailInfo: resp.FailInfo}, "SCEP server rejected the request")
	}
	certs, err := decryptCertRepCertificates(respBytes, signerCert, signerKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decrypting SCEP CertRep")
	}
	return certificateForCSR(ctx, certs, csr)
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
func certificateForCSR(ctx context.Context, certs []*x509.Certificate, csr *x509.CertificateRequest) (*x509.Certificate, error) {
	csrKey, err := x509.MarshalPKIXPublicKey(csr.PublicKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "encoding CSR public key")
	}
	for _, cert := range certs {
		certKey, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
		if err == nil && bytes.Equal(certKey, csrKey) {
			return cert, nil
		}
	}
	return nil, ctxerr.New(ctx, "SCEP CertRep has no certificate for the CSR's public key")
}

// fetchCACerts returns the SCEP server's GetCACert chain. A lone CA certificate is served as raw
// DER; a CA with an RA, the usual NDES setup, serves a PKCS7 chain. Each parser rejects the
// other format, so both are tried rather than trusting the Content-Type, which servers mislabel.
func fetchCACerts(ctx context.Context, client scepclient.Client) ([]*x509.Certificate, error) {
	data, _, err := client.GetCACert(ctx, "")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting CA certificates from SCEP URL")
	}
	caCerts, err := x509.ParseCertificates(data)
	if err != nil {
		if caCerts, err = smallstepscep.CACerts(data); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "parsing CA certificates from SCEP URL")
		}
	}
	if len(caCerts) == 0 {
		return nil, ctxerr.New(ctx, "SCEP URL did not return a CA certificate")
	}
	return caCerts, nil
}

// newEphemeralSigner returns a fresh RSA key with a short-lived self-signed certificate for it, to
// sign one SCEP request for a CSR whose private key Fleet does not hold. The certificate is never
// issued to anyone; the CA only uses it to verify the request and encrypt the CertRep.
func newEphemeralSigner(ctx context.Context, subject pkix.Name) (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "generating SCEP signer key")
	}

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
		return nil, nil, ctxerr.Wrap(ctx, err, "creating SCEP signer certificate")
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "parsing SCEP signer certificate")
	}
	return key, cert, nil
}

// recipientCertsSelector picks the RA encryption certificate from a GetCACert chain. A CA that
// serves no RA and decrypts with its own certificate often lacks the encipherment key usage, so
// the whole chain is used when nothing qualifies.
func recipientCertsSelector() smallstepscep.CertsSelectorFunc {
	encipherment := smallstepscep.EnciphermentCertsSelector()
	return func(certs []*x509.Certificate) []*x509.Certificate {
		if selected := encipherment(certs); len(selected) > 0 {
			return selected
		}
		return certs
	}
}

// enrollmentRejectedError is a CertRep whose status is not SUCCESS: the CA answered and declined,
// or deferred, the request.
type enrollmentRejectedError struct {
	Status   smallstepscep.PKIStatus
	FailInfo smallstepscep.FailInfo
}

func (e enrollmentRejectedError) Error() string {
	switch e.Status {
	case smallstepscep.FAILURE:
		// FailInfo.String panics on values outside the RFC 8894 set, so those are printed raw.
		failInfo := fmt.Sprintf("%q", string(e.FailInfo))
		switch e.FailInfo {
		case smallstepscep.BadAlg, smallstepscep.BadMessageCheck, smallstepscep.BadRequest, smallstepscep.BadTime, smallstepscep.BadCertID:
			failInfo = e.FailInfo.String()
		}
		// Not every SCEP CA requires a challenge, so Fleet cannot check for one up front, but a
		// missing or wrong one is the likeliest cause of a rejection.
		return fmt.Sprintf("status FAILURE with fail info %s; if this certificate authority requires a challenge, "+
			"include it as the CSR's challengePassword attribute", failInfo)
	case smallstepscep.PENDING:
		return "status PENDING; requests that need manual approval are not supported"
	default:
		return fmt.Sprintf("unknown status %q", string(e.Status))
	}
}
