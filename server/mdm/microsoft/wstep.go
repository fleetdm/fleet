package microsoft_mdm

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/micromdm/nanomdm/cryptoutil"
	"go.mozilla.org/pkcs7"
)

// TODO: Replace with imports from Marcos' PR
const (
	DocProvisioningAppProviderID = "FleetDM"
	CertRenewalPeriodInSecs      = "15552000"
)

// TODO: Replace with imports from Marcos' PR
type BinSecTokenType int

// TODO: Replace with imports from Marcos' PR
const (
	MDETokenPKCS7 BinSecTokenType = iota
	MDETokenPKCS10
	MDETokenPKCSInvalid
)

// WSTEPDepot implements certificate management associated with MS-WSTEP messages in the MS-MDE2 protocol.
type WSTEPDepot struct {
	// identityCert holds the identity certificate of the depot.
	identityCert *x509.Certificate
	// identityPrivateKey holds the private key of the depot.
	identityPrivateKey *rsa.PrivateKey
	// identityFingerprint holds the hex-encoded, sha1 fingerprint of the identity certificate.
	IdentityFingerprint *string
	// maxSerialNumber holds the maximum serial number. The maximum value a serial number can have
	// is 2^160. However, this could be limited further if required.
	maxSerialNumber *big.Int
}

type CertDepot interface {
	SignClientCSR(subject string, clientCSR *x509.CertificateRequest) ([]byte, string, error)
}

// newWSTEPDepot creates and returns a *WSTEPDepot.
func NewWSTEPDepot(certPEM []byte, privKeyPEM []byte) (*WSTEPDepot, error) {
	crt, err := cryptoutil.DecodePEMCertificate(certPEM)
	if err != nil {
		return nil, err
	}
	key, err := decodeRSAKeyFromPEM(privKeyPEM)
	if err != nil {
		return nil, err
	}
	fp := CertFingerprintHexStr(crt)

	return &WSTEPDepot{
		identityCert:        crt,
		identityPrivateKey:  key,
		IdentityFingerprint: &fp,
		maxSerialNumber:     new(big.Int).Lsh(big.NewInt(1), 128), // 2^12,
	}, nil
}

func decodeRSAKeyFromPEM(key []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(key)
	if block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("PEM type is not RSA PRIVATE KEY")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func CertFingerprintHexStr(cert *x509.Certificate) string {
	fingerprint := sha1.Sum(cert.Raw)
	return strings.ToUpper(hex.EncodeToString(fingerprint[:]))
}

// GetClientCSR returns the client certificate signing request from the BinarySecurityToken
func GetClientCSR(binSecTokenData string, tokenType BinSecTokenType) (*x509.CertificateRequest, error) {
	// Verify the token padding type
	if (tokenType != MDETokenPKCS7) && (tokenType != MDETokenPKCS10) {
		return nil, fmt.Errorf("provided binary security token type is invalid: %d", tokenType)
	}

	// Decoding the Base64 encoded binary security token to obtain the client CSR bytes
	rawCSR, err := base64.StdEncoding.DecodeString(binSecTokenData)
	if err != nil {
		return nil, fmt.Errorf("problem decoding the binary security token: %v", err)
	}

	// Sanity checks on binary signature token
	// Sanity checks are done on PKCS10 for the moment
	if tokenType == MDETokenPKCS7 {
		// Parse the CSR in PKCS7 Syntax Standard
		pk7CSR, err := pkcs7.Parse(rawCSR)
		if err != nil {
			return nil, fmt.Errorf("problem parsing the binary security token: %v", err)
		}

		// Verify the signatures of the CSR PKCS7 object
		err = pk7CSR.Verify()
		if err != nil {
			return nil, fmt.Errorf("problem verifying CSR data: %v", err)
		}

		// Verify signing time
		currentTime := time.Now()
		if currentTime.Before(pk7CSR.GetOnlySigner().NotBefore) || currentTime.After(pk7CSR.GetOnlySigner().NotAfter) {
			return nil, fmt.Errorf("invalid CSR signing time: %v", err)
		}
	}

	// Decode and verify CSR
	certCSR, err := x509.ParseCertificateRequest(rawCSR)
	if err != nil {
		return nil, fmt.Errorf("problem parsing CSR data: %v", err)
	}

	err = certCSR.CheckSignature()
	if err != nil {
		return nil, fmt.Errorf("invalid CSR signature: %v", err)
	}

	if certCSR.PublicKey == nil {
		return nil, fmt.Errorf("invalid CSR public key: %v", err)
	}

	if len(certCSR.Subject.String()) == 0 {
		return nil, fmt.Errorf("invalid CSR subject: %v", err)
	}

	return certCSR, nil
}

// SignClientCSR returns a signed certificate from the client certificate signing request and the
// certificate fingerprint. The certificate common name should be passed in the subject parameter.
func (d WSTEPDepot) SignClientCSR(subject string, clientCSR *x509.CertificateRequest) ([]byte, string, error) {
	if d.identityCert == nil || d.identityPrivateKey == nil {
		return nil, "", errors.New("invalid identity certificate or private key")
	}

	certRenewalPeriodInSecsInt, err := strconv.Atoi(CertRenewalPeriodInSecs)
	if err != nil {
		return nil, "", fmt.Errorf("invalid renewal time: %v", err)
	}

	// Time durations
	notBeforeDuration := time.Now().Add(time.Duration(certRenewalPeriodInSecsInt) * -time.Second)
	yearDuration := time.Duration(365 * 24 * time.Hour)

	certSubject := pkix.Name{
		OrganizationalUnit: []string{DocProvisioningAppProviderID},
		CommonName:         subject,
	}

	// Generate cryptographically strong pseudo-random number.
	// The serial number is used to uniquely identify the certificate
	serialNumber, err := rand.Int(rand.Reader, d.maxSerialNumber)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate serial number: %v", err.Error())
	}

	// Create the client certificate
	clientCertificate := &x509.Certificate{
		Subject:            certSubject,
		Issuer:             d.identityCert.Issuer,
		Version:            clientCSR.Version,
		PublicKey:          clientCSR.PublicKey,
		PublicKeyAlgorithm: clientCSR.PublicKeyAlgorithm,
		Signature:          clientCSR.Signature,
		SignatureAlgorithm: clientCSR.SignatureAlgorithm,
		Extensions:         clientCSR.Extensions,
		ExtraExtensions:    clientCSR.ExtraExtensions,
		IPAddresses:        clientCSR.IPAddresses,
		EmailAddresses:     clientCSR.EmailAddresses,
		DNSNames:           clientCSR.DNSNames,
		URIs:               clientCSR.URIs,
		NotBefore:          notBeforeDuration,
		NotAfter:           notBeforeDuration.Add(yearDuration),
		SerialNumber:       serialNumber,
		KeyUsage:           x509.KeyUsageDigitalSignature,

		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	rawSignedCertDER, err := x509.CreateCertificate(rand.Reader, clientCertificate, d.identityCert, clientCSR.PublicKey, d.identityPrivateKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign client certificate: %v", err.Error())
	}

	// Generate signed cert fingerprint
	fingerprint := sha1.Sum(rawSignedCertDER)
	fingerprintHex := strings.ToUpper(hex.EncodeToString(fingerprint[:]))

	return rawSignedCertDER, fingerprintHex, nil
}
