package apple_mdm

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math"
	"net/url"
	"path"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"golang.org/x/crypto/pbkdf2"
	"howett.net/plist"
)

// Note Apple rejects CSRs if the key size is not 2048.
const rsaKeySize = 2048

// newPrivateKey creates an RSA private key
func newPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, rsaKeySize)
}

func EncodeCertRequestPEM(cert *x509.CertificateRequest) []byte {
	pemBlock := &pem.Block{
		Type:    "CERTIFICATE REQUEST",
		Headers: nil,
		Bytes:   cert.Raw,
	}

	return pem.EncodeToMemory(pemBlock)
}

// GenerateRandomPin generates a `length`-digit random PIN number
//
// The implementation details for converting the randomness to a PIN
// have been mostly taken from https://github.com/pquerna/otp
func GenerateRandomPin(length int) (string, error) {
	buf := make([]byte, 16)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
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
	return fmt.Sprintf(f, v), nil
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

// IsRecoveryLockPasswordMismatchError checks if the error chain indicates that the
// recovery lock password provided does not match the one on the device. This is a
// terminal error that should not be retried automatically.
//
// Known error signatures:
// - MDMClientError (70): "Existing recovery lock password not provided"
// - ROSLockoutServiceDaemonErrorDomain (8): "The provided recovery password failed to validate."
func IsRecoveryLockPasswordMismatchError(chain []mdm.ErrorChain) bool {
	for _, e := range chain {
		// MDMClientError 70: "Existing recovery lock password not provided"
		if e.ErrorDomain == "MDMClientError" && e.ErrorCode == 70 {
			return true
		}
		// ROSLockoutServiceDaemonErrorDomain 8: "The provided recovery password failed to validate"
		if e.ErrorDomain == "ROSLockoutServiceDaemonErrorDomain" && e.ErrorCode == 8 {
			return true
		}
	}
	return false
}

// IsProfileNotFoundError checks if the error chain indicates that a profile
// was not found on the device. When this error occurs during a RemoveProfile
// command, it means the profile is already absent — the desired outcome.
//
// Known error signatures:
// - MDMClientError (89): "Profile with identifier '...' not found."
// - MDMErrorDomain (12075): "The profile '...' is not installed."
func IsProfileNotFoundError(chain []mdm.ErrorChain) bool {
	for _, e := range chain {
		if e.ErrorDomain == "MDMClientError" && e.ErrorCode == 89 {
			return true
		}

		if e.ErrorDomain == "MDMErrorDomain" && e.ErrorCode == 12075 {
			return true
		}
	}
	return false
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

const (
	// ManagedAccountPasswordGroupCount is the number of character groups in a managed account password.
	ManagedAccountPasswordGroupCount = 6
	// ManagedAccountPasswordGroupLen is the number of characters per group.
	ManagedAccountPasswordGroupLen = 4
	// pbkdf2Iterations is the number of PBKDF2 iterations for the managed account password hash.
	pbkdf2Iterations = 40000
	// pbkdf2KeyLen is the derived key length in bytes (128 bytes as required by Apple).
	pbkdf2KeyLen = 128
	// pbkdf2SaltLen is the salt length in bytes.
	pbkdf2SaltLen = 32
)

// GenerateManagedAccountPassword generates a cryptographically random password
// in the same format as recovery lock passwords (e.g., "5ADZ-HTZ8-LJJ4-B2F8-JWH3-YPBT").
func GenerateManagedAccountPassword() string {
	groups := make([]string, ManagedAccountPasswordGroupCount)
	charsetLen := len(RecoveryLockPasswordCharset)

	for i := range ManagedAccountPasswordGroupCount {
		randBytes := make([]byte, ManagedAccountPasswordGroupLen)
		_, _ = rand.Read(randBytes) // rand.Read never returns an error; it panics on failure

		group := make([]byte, ManagedAccountPasswordGroupLen)
		for j := range ManagedAccountPasswordGroupLen {
			group[j] = RecoveryLockPasswordCharset[int(randBytes[j])%charsetLen]
		}
		groups[i] = string(group)
	}

	return strings.Join(groups, "-")
}

// saltedSHA512PBKDF2 is the plist structure expected by Apple's AutoSetupAdminAccountItem.passwordHash.
type saltedSHA512PBKDF2 struct {
	PBKDF2 pbkdf2Dict `plist:"SALTED-SHA512-PBKDF2"`
}

type pbkdf2Dict struct {
	Entropy    []byte `plist:"entropy"`
	Salt       []byte `plist:"salt"`
	Iterations int    `plist:"iterations"`
}

// GenerateSaltedSHA512PBKDF2Hash generates the password hash structure required by
// Apple's [AutoSetupAdminAccountItem.passwordHash][1] field.
//
// Returns a plist-encoded byte slice containing a SALTED-SHA512-PBKDF2 dictionary
// with 32-byte salt, 128-byte derived key (entropy), and 40,000 iterations.
// The caller should base64-encode this into the <data> field of the AccountConfiguration plist.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/passwordhash/salted-sha512-pbkdf2-data.dictionary
func GenerateSaltedSHA512PBKDF2Hash(password string) ([]byte, error) {
	salt := make([]byte, pbkdf2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}

	entropy := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, pbkdf2KeyLen, sha512.New)

	hashPlist := saltedSHA512PBKDF2{
		PBKDF2: pbkdf2Dict{
			Entropy:    entropy,
			Salt:       salt,
			Iterations: pbkdf2Iterations,
		},
	}

	data, err := plist.Marshal(hashPlist, plist.XMLFormat)
	if err != nil {
		return nil, fmt.Errorf("marshaling PBKDF2 hash plist: %w", err)
	}
	return data, nil
}
