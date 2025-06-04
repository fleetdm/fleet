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

// VerifyMdmSignature verifies an Apple MDM "Mdm-Signature" header and returns the signing certificate.
// See https://developer.apple.com/documentation/devicemanagement/implementing_device_management/managing_certificates_for_mdm_servers_and_devices
// section "Pass an Identity Certificate Through a Proxy."
func VerifyMdmSignature(header string, body []byte) (*x509.Certificate, error) {
	sig, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return nil, err
	}
	p7, err := pkcs7.Parse(sig)
	if err != nil {
		return nil, err
	}
	p7.Content = body
	err = p7.Verify()
	if err != nil {
		return nil, err
	}
	cert := p7.GetOnlySigner()
	if cert == nil {
		return nil, errors.New("invalid or missing signer")
	}
	return cert, nil
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
