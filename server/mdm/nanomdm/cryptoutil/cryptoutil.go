// Package cryptoutil contains crypto-related helpers and utilities.
package cryptoutil

import (
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/smallstep/pkcs7"
)

// OID for UID (User ID) attribute
// See https://tools.ietf.org/html/rfc4519#section-2.39
var oidUID = asn1.ObjectIdentifier{0, 9, 2342, 19200300, 100, 1, 1}

// TopicFromCert extracts the APNs Topic (UserID OID) from cert.
func TopicFromCert(cert *x509.Certificate) (string, error) {
	for _, v := range cert.Subject.Names {
		if v.Type.Equal(oidUID) {
			userId, ok := v.Value.(string)
			if ok && strings.HasPrefix(userId, "com.apple.mgmt") {
				return userId, nil
			}
			return "", fmt.Errorf("invalid APNs Topic: %q", userId)
		}
	}
	return "", errors.New("no APNs Topic found")
}

// TopicFromPEMCert extracts the APNs Topic from a PEM-encoded cert.
func TopicFromPEMCert(pemCert []byte) (string, error) {
	cert, err := DecodePEMCertificate(pemCert)
	if err != nil {
		return "", err
	}
	return TopicFromCert(cert)
}

// maxMdmSignatureBytes caps the decoded length of the Mdm-Signature header.
// Real signing-cert + signed-attrs + signature blobs come in under 4 KB;
const maxMdmSignatureBytes = 10 * 1024

// VerifyMdmSignature verifies an Apple MDM "Mdm-Signature" header and returns the signing certificate.
// See https://developer.apple.com/documentation/devicemanagement/implementing_device_management/managing_certificates_for_mdm_servers_and_devices
// section "Pass an Identity Certificate Through a Proxy."
func VerifyMdmSignature(header string, body []byte) (*x509.Certificate, error) {
	return verifyMdmSignature(header, body, false)
}

// VerifyMdmSignatureIgnoringExpiry is like VerifyMdmSignature but accepts
// signatures whose signing time falls outside the signer certificate's
// validity period, so that devices with expired identity certificates can
// still check in and receive a renewal.
func VerifyMdmSignatureIgnoringExpiry(header string, body []byte) (*x509.Certificate, error) {
	return verifyMdmSignature(header, body, true)
}

func verifyMdmSignature(header string, body []byte, ignoreExpiry bool) (*x509.Certificate, error) {
	sig, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return nil, err
	}
	if len(sig) > maxMdmSignatureBytes {
		return nil, fmt.Errorf("Mdm-Signature header exceeds %d bytes", maxMdmSignatureBytes)
	}
	if err := ValidateBERDepth(sig, MaxBERDepth); err != nil {
		return nil, err
	}
	p7, err := pkcs7.Parse(sig)
	if err != nil {
		return nil, err
	}
	p7.Content = body
	err = p7.Verify()
	var signingTimeErr *pkcs7.SigningTimeNotValidError
	if ignoreExpiry && errors.As(err, &signingTimeErr) {
		err = verifyIgnoringValidityPeriod(p7)
	}
	if err != nil {
		return nil, err
	}
	cert := p7.GetOnlySigner()
	if cert == nil {
		return nil, errors.New("invalid or missing signer")
	}
	return cert, nil
}

// verifyIgnoringValidityPeriod verifies p7 against copies of its certificates
// with an unbounded validity period. pkcs7 offers no option to skip the
// signing-time check, and mutating the originals would leak the fake dates to
// callers of GetOnlySigner.
func verifyIgnoringValidityPeriod(p7 *pkcs7.PKCS7) error {
	originals := p7.Certificates
	defer func() { p7.Certificates = originals }()

	widened := make([]*x509.Certificate, len(originals))
	for i, c := range originals {
		cp := *c
		cp.NotBefore = time.Time{}
		cp.NotAfter = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
		widened[i] = &cp
	}
	p7.Certificates = widened
	return p7.Verify()
}

// PEMCertificate returns derBytes encoded as a PEM block
func PEMCertificate(derBytes []byte) []byte {
	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	}
	return pem.EncodeToMemory(block)
}

// DecodePEMCertificate returns an X509 certificate from a PEM-encoded
// certificate provided in pemData.
func DecodePEMCertificate(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("failed to decode PEM certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}
