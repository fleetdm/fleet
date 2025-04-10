package apple_mdm

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"math"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

// Note Apple rejects CSRs if the key size is not 2048.
const rsaKeySize = 2048

// newPrivateKey creates an RSA private key
func newPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, rsaKeySize)
}

// EncodeCertPEM returns PEM-endcoded certificate data.
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

func EncodeCertRequestPEM(cert *x509.CertificateRequest) []byte {
	pemBlock := &pem.Block{
		Type:    "CERTIFICATE REQUEST",
		Headers: nil,
		Bytes:   cert.Raw,
	}

	return pem.EncodeToMemory(pemBlock)
}

// EncodePrivateKeyPEM returns PEM-encoded private key data
func EncodePrivateKeyPEM(key *rsa.PrivateKey) []byte {
	block := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.EncodeToMemory(&block)
}

// GenerateRandomPin generates a `lenght`-digit PIN number that takes into
// account the current time as described in rfc4226 (for one time passwords)
//
// The implementation details have been mostly taken from https://github.com/pquerna/otp
func GenerateRandomPin(length int) string {
	counter := uint64(time.Now().Unix()) //nolint:gosec // dismiss G115
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	m := sha256.New()
	m.Write(buf)
	sum := m.Sum(nil)
	offset := sum[len(sum)-1] & 0xf
	value := int64(((int(sum[offset]) & 0x7f) << 24) |
		((int(sum[offset+1] & 0xff)) << 16) |
		((int(sum[offset+2] & 0xff)) << 8) |
		(int(sum[offset+3]) & 0xff))
	v := int32(value % int64(math.Pow10(length))) //nolint:gosec // dismiss G115
	f := fmt.Sprintf("%%0%dd", length)
	return fmt.Sprintf(f, v)
}

// FmtErrorChain formats Command error message for macOS MDM v1
func FmtErrorChain(chain []mdm.ErrorChain) string {
	var sb strings.Builder
	for _, mdmErr := range chain {
		desc := mdmErr.USEnglishDescription
		if desc == "" {
			desc = mdmErr.LocalizedDescription
		}
		sb.WriteString(fmt.Sprintf("%s (%d): %s\n", mdmErr.ErrorDomain, mdmErr.ErrorCode, desc))
	}
	return sb.String()
}

// FmtDDMError formats a DDM error message
func FmtDDMError(reasons []fleet.MDMAppleDDMStatusErrorReason) string {
	var errMsg strings.Builder
	for _, r := range reasons {
		errMsg.WriteString(fmt.Sprintf("%s: %s %+v\n", r.Code, r.Description, r.Details))
	}
	return errMsg.String()
}

func EnrollURL(token string, appConfig *fleet.AppConfig) (string, error) {
	enrollURL, err := url.Parse(appConfig.MDMUrl())
	if err != nil {
		return "", err
	}
	enrollURL.Path = path.Join(enrollURL.Path, EnrollPath)
	q := enrollURL.Query()
	q.Set("token", token)
	enrollURL.RawQuery = q.Encode()
	return enrollURL.String(), nil
}

// IsLessThanVersion returns true if the current version is less than the target version.
// If either version is invalid, an error is returned.
func IsLessThanVersion(current string, target string) (bool, error) {
	cv, err := fleet.VersionToSemverVersion(current)
	if err != nil {
		return false, fmt.Errorf("invalid current version: %w", err)
	}
	tv, err := fleet.VersionToSemverVersion(target)
	if err != nil {
		return false, fmt.Errorf("invalid target version: %w", err)
	}

	return cv.LessThan(tv), nil
}
