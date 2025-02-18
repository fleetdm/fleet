package microsoft_mdm

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	"github.com/golang-jwt/jwt/v4"
	"github.com/smallstep/pkcs7"
)

// CertManager is an interface for certificate management tasks associated with Microsoft MDM (e.g.,
// signing CSRs).
type CertManager interface {
	// IdentityFingerprint returns the hex-encoded, uppercased sha1 fingerprint of the identity certificate.
	IdentityFingerprint() string

	// SignClientCSR signs a client CSR and returns the signed, DER-encoded certificate bytes and
	// its uppercased, hex-endcoded sha1 fingerprint. The subject passed is set as the common name of
	// the signed certificate.
	SignClientCSR(ctx context.Context, subject string, clientCSR *x509.CertificateRequest) ([]byte, string, error)

	// IdentityCert returns the identity certificate of the depot.
	IdentityCert() x509.Certificate

	// NewSTSAuthToken returns an STS auth token for the given UPN claim.
	NewSTSAuthToken(upn string) (string, error)

	// GetSTSAuthTokenUPNClaim validates the given token and returns the UPN claim
	GetSTSAuthTokenUPNClaim(token string) (string, error)

	// TODO: implement other methods as needed:
	// - verify certificate-device association
	// - certificate lifecycle management (e.g., renewal, revocation)
}

// CertStore implements storage tasks associated with MS-WSTEP messages in the MS-MDE2
// protocol. It is implemented by fleet.Datastore.
type CertStore interface {
	WSTEPStoreCertificate(ctx context.Context, name string, crt *x509.Certificate) error
	WSTEPNewSerial(ctx context.Context) (*big.Int, error)
	WSTEPAssociateCertHash(ctx context.Context, deviceUUID string, hash string) error
}

type STSClaims struct {
	UPN string `json:"upn"`
	jwt.RegisteredClaims
}

type AzureData struct {
	UPN        string
	TenantID   string
	UniqueName string
	SCP        string
}

type manager struct {
	store CertStore

	// identityCert holds the identity certificate of the depot.
	identityCert *x509.Certificate
	// identityPrivateKey holds the private key of the depot.
	identityPrivateKey *rsa.PrivateKey
	// identityFingerprint holds the hex-encoded, sha1 fingerprint of the identity certificate.
	identityFingerprint string
	// maxSerialNumber holds the maximum serial number. The maximum value a serial number can have
	// is 2^160. However, this could be limited further if required.
	maxSerialNumber *big.Int
}

// NewCertManager returns a new CertManager instance.
func NewCertManager(store CertStore, certPEM []byte, privKeyPEM []byte) (CertManager, error) {
	return newManager(store, certPEM, privKeyPEM)
}

func newManager(store CertStore, certPEM []byte, privKeyPEM []byte) (*manager, error) {
	crt, err := cryptoutil.DecodePEMCertificate(certPEM)
	if err != nil {
		return nil, fmt.Errorf("decode certificate: %w", err)
	}
	key, err := server.DecodePrivateKeyPEM(privKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	fp := CertFingerprintHexStr(crt)

	return &manager{
		store:               store,
		identityCert:        crt,
		identityPrivateKey:  key,
		identityFingerprint: fp,
		maxSerialNumber:     new(big.Int).Lsh(big.NewInt(1), 128), // 2^12,
	}, nil
}

func (m *manager) IdentityFingerprint() string {
	if m == nil {
		return ""
	}

	return m.identityFingerprint
}

func (m *manager) IdentityCert() x509.Certificate {
	if m == nil {
		return x509.Certificate{}
	}

	return *m.identityCert
}

// SignClientCSR returns a signed certificate from the client certificate signing request and the certificate fingerprint
// subject is the DeviceID of the about to be MDM enrolled device, it will be used as the CommonName of the certificate
// clientCSR is the client certificate signing request
func (m *manager) SignClientCSR(ctx context.Context, subject string, clientCSR *x509.CertificateRequest) ([]byte, string, error) {
	if m == nil {
		return nil, "", errors.New("windows mdm identity keypair was not configured")
	}

	if m.identityCert == nil || m.identityPrivateKey == nil {
		return nil, "", errors.New("invalid identity certificate or private key")
	}

	// serial number is used to uniquely identify the certificate
	sn, err := m.store.WSTEPNewSerial(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	// populate the client certificate template
	tmpl, err := populateClientCert(sn, subject, m.identityCert, clientCSR)
	if err != nil {
		return nil, "", fmt.Errorf("failed to populate client certificate: %w", err)
	}

	rawSignedDER, err := x509.CreateCertificate(rand.Reader, tmpl, m.identityCert, clientCSR.PublicKey, m.identityPrivateKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign client certificate: %w", err)
	}

	signedCert, err := x509.ParseCertificate(rawSignedDER)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse client certificate: %w", err)
	}

	if err := m.store.WSTEPStoreCertificate(ctx, subject, signedCert); err != nil {
		return nil, "", fmt.Errorf("failed to store client certificate: %w", err)
	}

	return rawSignedDER, CertFingerprintHexStr(signedCert), nil
}

// NewSTSAuthToken returns an STS auth token for the given UPN claim.
func (m *manager) NewSTSAuthToken(upn string) (string, error) {
	if m == nil {
		return "", errors.New("windows mdm identity keypair was not configured")
	}

	if m.identityCert == nil || m.identityPrivateKey == nil {
		return "", errors.New("invalid identity certificate or private key")
	}

	if len(upn) == 0 {
		return "", errors.New("invalid upn field")
	}

	// Create claims with upn field populated
	claims := STSClaims{
		upn,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   "STSAuthToken",
		},
	}

	// Create a new token with the claims and sign it with the private key
	token := jwt.NewWithClaims(jwt.GetSigningMethod("RS256"), claims)
	signedToken, err := token.SignedString(m.identityPrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign STS token: %w", err)
	}

	return signedToken, nil
}

// GetSTSAuthToken validates the given token and returns the UPN claim
func (m *manager) GetSTSAuthTokenUPNClaim(tokenStr string) (string, error) {
	if m == nil {
		return "", errors.New("windows mdm identity keypair was not configured")
	}

	if m.identityCert == nil || m.identityPrivateKey == nil {
		return "", errors.New("invalid identity certificate or private key")
	}

	if len(tokenStr) == 0 {
		return "", errors.New("invalid STS token")
	}

	// Since we used the private key to sign the tokens, we use the public counterpart to verify the signature
	token, err := jwt.ParseWithClaims(tokenStr, &STSClaims{}, func(token *jwt.Token) (interface{}, error) {
		return m.identityCert.PublicKey, nil
	})
	if err != nil {
		return "", fmt.Errorf("there was an error parsing the STS token claims: %w", err)
	}

	if claims, ok := token.Claims.(*STSClaims); ok && token.Valid {
		if len(claims.UPN) == 0 {
			return "", errors.New("issue with UPN token claim")
		}

		return claims.UPN, nil
	}

	return "", errors.New("issue with STS token validation")
}

// GetAzureAuthTokenClaims validates the given Azure AD token and returns
// UPN, TenantID, UniqueName, DeviceID
func GetAzureAuthTokenClaims(tokenStr string) (AzureData, error) {
	if len(tokenStr) == 0 {
		return AzureData{}, errors.New("invalid STS token")
	}

	// Decode base64 token
	tokenBytes, err := base64.StdEncoding.DecodeString(tokenStr)
	if err != nil {
		return AzureData{}, errors.New("invalid Azure JWT token")
	}

	// Validate token format (header.payload.signature)
	parts := bytes.Split(tokenBytes, []byte("."))
	if len(parts) != 3 {
		return AzureData{}, errors.New("invalid Azure JWT format")
	}

	// Parse JWT token
	token, _, err := new(jwt.Parser).ParseUnverified(string(tokenBytes), jwt.MapClaims{})
	if err != nil {
		return AzureData{}, errors.New("parse error Azure JWT content")
	}

	// Parse JWT token
	claims := token.Claims.(jwt.MapClaims)

	// Get UPN claim
	upnClaim, ok := claims["upn"].(string)
	if !ok || len(upnClaim) == 0 {
		return AzureData{}, errors.New("invalid UPN claim")
	}

	// Get TenantID claim
	tenantIDClaim, ok := claims["tid"].(string)
	if !ok || len(tenantIDClaim) == 0 {
		return AzureData{}, errors.New("invalid TenantID claim")
	}

	// Get UniqueName claim
	uniqueNameClaim, ok := claims["unique_name"].(string)
	if !ok {
		return AzureData{}, errors.New("invalid UniqueName claim")
	}

	// Get SCP claim
	azureSCPClaim, ok := claims["scp"].(string)
	if !ok || azureSCPClaim != "mdm_delegation" {
		return AzureData{}, errors.New("invalid SCP claim")
	}

	return AzureData{
		UPN:        upnClaim,
		TenantID:   tenantIDClaim,
		UniqueName: uniqueNameClaim,
		SCP:        azureSCPClaim,
	}, nil
}

func populateClientCert(sn *big.Int, subject string, issuerCert *x509.Certificate, csr *x509.CertificateRequest) (*x509.Certificate, error) {
	certRenewalPeriodInSecsInt, err := strconv.Atoi(syncml.PolicyCertRenewalPeriodInSecs)
	if err != nil {
		return nil, fmt.Errorf("invalid renewal time: %w", err)
	}

	notBeforeDuration := time.Now().Add(time.Duration(certRenewalPeriodInSecsInt) * -time.Second)
	yearDuration := 365 * 24 * time.Hour

	certSubject := pkix.Name{
		OrganizationalUnit: []string{syncml.DocProvisioningAppProviderID},
		CommonName:         subject,
	}

	tmpl := &x509.Certificate{
		Subject:            certSubject,
		Issuer:             issuerCert.Issuer,
		Version:            csr.Version,
		PublicKey:          csr.PublicKey,
		PublicKeyAlgorithm: csr.PublicKeyAlgorithm,
		Signature:          csr.Signature,
		SignatureAlgorithm: x509.SHA256WithRSA,
		Extensions:         csr.Extensions,
		ExtraExtensions:    csr.ExtraExtensions,
		IPAddresses:        csr.IPAddresses,
		EmailAddresses:     csr.EmailAddresses,
		DNSNames:           csr.DNSNames,
		URIs:               csr.URIs,
		NotBefore:          notBeforeDuration,
		NotAfter:           notBeforeDuration.Add(yearDuration),
		SerialNumber:       sn,
		KeyUsage:           x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,

		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}
	return tmpl, nil
}

// GetClientCSR returns the client certificate signing request from the BinarySecurityToken
func GetClientCSR(binSecTokenData string, tokenType string) (*x509.CertificateRequest, error) {
	// Checking if this is a valid enroll security token (CSR)
	if (tokenType != syncml.EnrollReqTypePKCS10) && (tokenType != syncml.EnrollReqTypePKCS7) {
		return nil, fmt.Errorf("token type is not valid for MDM enrollment: %s", tokenType)
	}

	// Decoding the Base64 encoded binary security token to obtain the client CSR bytes
	rawCSR, err := base64.StdEncoding.DecodeString(binSecTokenData)
	if err != nil {
		return nil, fmt.Errorf("decoding the binary security token: %w", err)
	}

	// Sanity checks on binary signature token
	// Sanity checks are done on PKCS10 for the moment
	if tokenType == syncml.EnrollReqTypePKCS7 {
		// Parse the CSR in PKCS7 Syntax Standard
		pk7CSR, err := pkcs7.Parse(rawCSR)
		if err != nil {
			return nil, fmt.Errorf("parsing the binary security token: %v", err)
		}

		// Verify the signatures of the CSR PKCS7 object
		err = pk7CSR.Verify()
		if err != nil {
			return nil, fmt.Errorf("verifying CSR data: %v", err)
		}

		// Verify signing time
		currentTime := time.Now()
		if currentTime.Before(pk7CSR.GetOnlySigner().NotBefore) || currentTime.After(pk7CSR.GetOnlySigner().NotAfter) {
			return nil, fmt.Errorf("invalid CSR signing time: %v", err)
		}
	}

	// Decode and verify CSR
	certCSR, err := ParseCertificateRequestFromWindowsDevice(rawCSR)
	if err != nil {
		return nil, fmt.Errorf("parsing CSR data: %v", err)
	}

	err = certCSR.CheckSignature()
	if err != nil {
		return nil, fmt.Errorf("CSR signature: %v", err)
	}

	if certCSR.PublicKey == nil {
		return nil, fmt.Errorf("CSR public key: %v", err)
	}

	if len(certCSR.Subject.String()) == 0 {
		return nil, fmt.Errorf("CSR subject: %v", err)
	}

	return certCSR, nil
}

// CertFingerprintHexStr returns the hex-encoded, uppercased sha1 fingerprint of the certificate.
func CertFingerprintHexStr(cert *x509.Certificate) string {
	// Windows Certificate Store requires passing the certificate thumbprint, which is the same as
	// SHA1 fingerprint. See also:
	// https://security.stackexchange.com/questions/14330/what-is-the-actual-value-of-a-certificate-fingerprint
	// https://www.thesslstore.com/blog/ssl-certificate-still-sha-1-thumbprint/
	fingerprint := sha1.Sum(cert.Raw) //nolint:gosec
	return strings.ToUpper(hex.EncodeToString(fingerprint[:]))
}
