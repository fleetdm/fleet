package mdm

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/base64"

	"go.mozilla.org/pkcs7"
)

// MaxProfileRetries is the maximum times an install profile command may be
// retried, after which marked as failed and no further attempts will be made
// to install the profile.
const MaxProfileRetries = 1

// DecryptBase64CMS decrypts a base64 encoded pkcs7-encrypted value using the
// provided certificate and private key.
func DecryptBase64CMS(p7Base64 string, cert *x509.Certificate, key crypto.PrivateKey) ([]byte, error) {
	p7Bytes, err := base64.StdEncoding.DecodeString(p7Base64)
	if err != nil {
		return nil, err
	}

	p7, err := pkcs7.Parse(p7Bytes)
	if err != nil {
		return nil, err
	}

	return p7.Decrypt(cert, key)
}

// GetRawProfilePlatform identifies the platform type of a profile bytes by
// examining its initial content:
//
//   - Returns "darwin" if the profile starts with "<?xml", typical of Darwin
//     platform profiles.
//   - Returns "windows" if the profile begins with "<replace" or "<add",
//   - Returns an empty string for profiles that are either unrecognized or
//     empty.
func GetRawProfilePlatform(profile []byte) string {
	trimmedProfile := bytes.TrimSpace(profile)

	if len(trimmedProfile) == 0 {
		return ""
	}

	prefixMatches := func(prefix []byte) bool {
		return len(trimmedProfile) >= len(prefix) &&
			bytes.EqualFold(prefix, trimmedProfile[:len(prefix)])
	}

	if prefixMatches([]byte("<?xml")) {
		return "darwin"
	}

	if prefixMatches([]byte("<replace")) || prefixMatches([]byte("<add")) {
		return "windows"
	}

	return ""
}

const (

	// FleetdConfigProfileName is the value for the PayloadDisplayName used by
	// fleetd to read configuration values from the system.
	FleetdConfigProfileName = "Fleetd configuration"

	// FleetdFileVaultProfileName is the value for the PayloadDisplayName used
	// by Fleet to configure FileVault and FileVault Escrow.
	FleetFileVaultProfileName        = "Disk encryption"
	FleetWindowsOSUpdatesProfileName = "Windows OS Updates"
)

// FleetReservedProfileNames returns a map of PayloadDisplayName strings
// that are reserved by Fleet.
func FleetReservedProfileNames() map[string]struct{} {
	return map[string]struct{}{
		FleetdConfigProfileName:          {},
		FleetFileVaultProfileName:        {},
		FleetWindowsOSUpdatesProfileName: {},
	}
}

// ListFleetReservedWindowsProfileNames returns a list of PayloadDisplayName strings
// that are reserved by Fleet for Windows.
func ListFleetReservedWindowsProfileNames() []string {
	return []string{FleetWindowsOSUpdatesProfileName}
}
