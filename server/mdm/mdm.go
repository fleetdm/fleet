package mdm

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"unicode"

	"github.com/smallstep/pkcs7"
)

// MaxAppleProfileRetries is the maximum number of times an Apple install profile command may be retried after the
// initial attempt, after which it is marked as failed and no further attempts will be made to install the profile.
const MaxAppleProfileRetries = 3

// MaxWindowsProfileRetries is the maximum number of times a Windows install profile command may be retried after the
// initial attempt, after which it is marked as failed and no further attempts will be made to install the profile.
const MaxWindowsProfileRetries = 1

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

// DecryptBase64CMSWithCerts tries each candidate certificate in order and
// returns the first successful decryption.
//
// CMS matches the recipient by issuer + serial number, not by key (see
// pkcs7.selectRecipientForCertificate). After a CA certificate rollover the
// renewed cert keeps the same private key but is issued a new serial, so a
// payload escrowed against a previous CA cert no longer matches the current
// cert even though the key can still decrypt it. Passing the current and any
// previous CA certs (all sharing the one private key) lets decryption succeed
// regardless of which cert was current when the payload was escrowed.
func DecryptBase64CMSWithCerts(p7Base64 string, key crypto.PrivateKey, certs []*x509.Certificate) ([]byte, error) {
	if len(certs) == 0 {
		return nil, errors.New("no certificates provided for decryption")
	}

	p7Bytes, err := base64.StdEncoding.DecodeString(p7Base64)
	if err != nil {
		return nil, err
	}
	p7, err := pkcs7.Parse(p7Bytes)
	if err != nil {
		return nil, err
	}

	var errs []error
	for _, cert := range certs {
		decrypted, err := p7.Decrypt(cert, key)
		if err == nil {
			return decrypted, nil
		}
		errs = append(errs, err)
	}
	return nil, errors.Join(errs...)
}

func prefixMatches(val []byte, prefix string) bool {
	return len(val) >= len(prefix) &&
		bytes.EqualFold([]byte(prefix), val[:len(prefix)])
}

// GetRawProfilePlatform identifies the platform type of a profile bytes by
// examining its initial content:
//
//   - Returns "darwin" if the profile starts with "<?xml", typical of Apple
//     platform profiles.
//   - Returns "windows" if the profile begins with "<replace" or "<add",
//   - Returns an empty string for profiles that are either unrecognized or
//     empty.
func GetRawProfilePlatform(profile []byte) string {
	trimmedProfile := bytes.TrimSpace(profile)

	if len(trimmedProfile) == 0 {
		return ""
	}

	if prefixMatches(trimmedProfile, "{") {
		isAppleJson, isAndroidJson, err := DetermineJSONConfigType(trimmedProfile)
		if err != nil {
			return ""
		}

		if isAppleJson {
			return "darwin"
		}
		if isAndroidJson {
			return "android"
		}

		return ""
	}

	if prefixMatches(trimmedProfile, "<?xml") {
		return "darwin"
	}

	if prefixMatches(trimmedProfile, "<replace") || prefixMatches(trimmedProfile, "<add") || prefixMatches(trimmedProfile, "<atomic") || prefixMatches(trimmedProfile, "<!--") {
		return "windows"
	}

	return ""
}

// DetermineJSONConfigType checks the JSON data to determine if it is an Apple or Android profile.
// Returns isApple, isAndroid, error.
func DetermineJSONConfigType(data []byte) (bool, bool, error) {
	type jsonObj map[string]interface{}
	var profileKeyMap jsonObj
	err := json.Unmarshal(data, &profileKeyMap)
	if err != nil {
		return false, false, fmt.Errorf("Couldn't add. The file should include valid JSON: %s", err.Error())
	}
	if len(profileKeyMap) == 0 {
		return false, false, errors.New("Couldn't add. JSON is empty")
	}
	hasTypeKey := false
	hasPayloadKey := false
	hasKeysStartingInUpper := false
	hasKeysStartingInLower := false
	hasKeysContainingNonLetters := false
	for k := range profileKeyMap {
		if k == "" {
			return false, false, errors.New("empty string is not a valid JSON configuration key")
		}
		if k == "Type" {
			hasTypeKey = true
			hasKeysStartingInUpper = true
			continue
		}
		if k == "Payload" {
			hasPayloadKey = true
			hasKeysStartingInUpper = true
			continue
		}

		for i, r := range k {
			if i == 0 {
				if unicode.IsUpper(r) {
					hasKeysStartingInUpper = true
				} else if unicode.IsLower(r) {
					hasKeysStartingInLower = true
				}
			}
			if !unicode.IsLetter(r) {
				hasKeysContainingNonLetters = true
			}
		}
	}

	// It's an Apple declaration or at least looks like one
	if hasKeysStartingInUpper && !hasKeysStartingInLower {
		if hasTypeKey && hasPayloadKey {
			return true, false, nil
		}
		if !hasTypeKey {
			return false, false, errors.New("apple declaration missing Type")
		}
		return false, false, errors.New("apple declaration missing Payload")
	}
	// Android declaration
	if !hasKeysStartingInUpper && hasKeysStartingInLower {
		if !hasKeysContainingNonLetters {
			return false, true, nil
		}
		return false, false, errors.New("android configuration profile contains invalid keys")
	}
	// Didn't match either one
	return false, false, errors.New("Couldn't add. Keys in declaration (DDM) profile must contain only letters and start with a uppercase letter. Keys in Android profile must contain only letters and start with a lowercase letter.")
}

// GuessProfileExtension determines the likely file extension of a profile
// based on its content.
//
// It returns a string representing the determined file extension ("xml",
// "json", or "") based on the profile's content.
func GuessProfileExtension(profile []byte) string {
	trimmedProfile := bytes.TrimSpace(profile)

	switch {
	case prefixMatches(trimmedProfile, "<?xml"),
		prefixMatches(trimmedProfile, "<replace"),
		prefixMatches(trimmedProfile, "<add"):
		return "xml"
	case prefixMatches(trimmedProfile, "{"):
		return "json"
	default:
		return ""
	}
}

func EncryptAndEncode(plainText string, symmetricKey string) (string, error) {
	block, err := aes.NewCipher([]byte(symmetricKey))
	if err != nil {
		return "", fmt.Errorf("create new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create new gcm: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	return base64.StdEncoding.EncodeToString(aesGCM.Seal(nonce, nonce, []byte(plainText), nil)), nil
}

func DecodeAndDecrypt(base64CipherText string, symmetricKey string) (string, error) {
	encrypted, err := base64.StdEncoding.DecodeString(base64CipherText)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}

	block, err := aes.NewCipher([]byte(symmetricKey))
	if err != nil {
		return "", fmt.Errorf("create new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create new gcm: %w", err)
	}

	// Get the nonce size
	nonceSize := aesGCM.NonceSize()

	// Extract the nonce from the encrypted data
	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]

	decrypted, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypting: %w", err)
	}

	return string(decrypted), nil
}

const (
	// FleetdConfigProfileName is the value for the PayloadDisplayName used by
	// fleetd to read configuration values from the system.
	FleetdConfigProfileName = "Fleetd configuration"

	// FleetCAConfigProfileName is the value for the PayloadDisplayName used by
	// fleetd to read configuration values from the system.
	FleetCAConfigProfileName = "Fleet root certificate authority (CA)"

	// FleetdFileVaultProfileName is the value for the PayloadDisplayName used
	// by Fleet to configure FileVault and FileVault Escrow.
	FleetFileVaultProfileName = "Disk encryption"

	// FleetWindowsOSUpdatesProfileName is the name of the profile used by Fleet
	// to configure Windows OS updates.
	FleetWindowsOSUpdatesProfileName = "Windows OS Updates"

	// FleetMacOSUpdatesProfileName is the name of the DDM profile used by Fleet
	// to configure macOS OS updates.
	FleetMacOSUpdatesProfileName = "Fleet macOS OS Updates"

	// FleetIOSUpdatesProfileName is the name of the DDM profile used by Fleet
	// to configure iOS OS updates.
	FleetIOSUpdatesProfileName = "Fleet iOS OS Updates"

	// FleetIPadOSUpdatesProfileName is the name of the DDM profile used by Fleet
	// to configure iPadOS OS updates.
	FleetIPadOSUpdatesProfileName = "Fleet iPadOS OS Updates"
)

// FleetReservedProfileNames returns a map of PayloadDisplayName or profile
// name strings that are reserved by Fleet.
func FleetReservedProfileNames() map[string]struct{} {
	return map[string]struct{}{
		FleetdConfigProfileName:          {},
		FleetFileVaultProfileName:        {},
		FleetWindowsOSUpdatesProfileName: {},
		FleetMacOSUpdatesProfileName:     {},
		FleetIOSUpdatesProfileName:       {},
		FleetIPadOSUpdatesProfileName:    {},
		FleetCAConfigProfileName:         {},
	}
}

// ListFleetReservedWindowsProfileNames returns a list of PayloadDisplayName strings
// that are reserved by Fleet for Windows.
func ListFleetReservedWindowsProfileNames() []string {
	return []string{FleetWindowsOSUpdatesProfileName}
}

// ListFleetReservedMacOSProfileNames returns a list of PayloadDisplayName strings
// that are reserved by Fleet for macOS.
func ListFleetReservedMacOSProfileNames() []string {
	return []string{FleetFileVaultProfileName, FleetdConfigProfileName, FleetCAConfigProfileName}
}

// ListFleetReservedMacOSDeclarationNames returns a list of declaration names
// that are reserved by Fleet for Apple DDM declarations.
func ListFleetReservedMacOSDeclarationNames() []string {
	return []string{
		FleetMacOSUpdatesProfileName,
		FleetIOSUpdatesProfileName,
		FleetIPadOSUpdatesProfileName,
	}
}

// softwareUpdateProfileError is an error type indicating OS updates is already configured
// and a new profile cannot be added until the existing OS updates configuration is removed.
// This signals to the caller that they should rollback/delete the recently saved profile.
type softwareUpdateProfileError struct {
	// OSUpdatesAlreadyConfigured sets the error message returned,
	// if true, then we report back OS updates are configured
	// if false, an existing custom OS updates profile exists.
	OSUpdatesAlreadyConfigured bool
	profileType                string

	// When set means something else went wrong, but we still want to rollback the upserted profile.
	internalErr error
}

// Use platform specific if possible, and only this for generic errors that requires a deletion of the upserted profile.
func NewSoftwareUpdateProfileError(internalErr error) *softwareUpdateProfileError {
	return &softwareUpdateProfileError{
		internalErr: internalErr,
	}
}

func NewAppleSoftwareUpdateProfileError(osUpdatesAlreadyConfigured bool) *softwareUpdateProfileError {
	return &softwareUpdateProfileError{
		OSUpdatesAlreadyConfigured: osUpdatesAlreadyConfigured,
		profileType:                "declaration",
	}
}

func NewWindowsSoftwareUpdateProfileError(osUpdatesAlreadyConfigured bool) *softwareUpdateProfileError {
	return &softwareUpdateProfileError{
		OSUpdatesAlreadyConfigured: osUpdatesAlreadyConfigured,
		profileType:                "", // Empty to leave out the declaration text
	}
}

func IsSoftwareUpdateProfileError(err error) bool {
	var target *softwareUpdateProfileError
	return errors.As(err, &target)
}

func (e *softwareUpdateProfileError) Error() string {
	if e.internalErr != nil {
		return e.internalErr.Error()
	}

	if !e.OSUpdatesAlreadyConfigured {
		extraText := ""
		if e.profileType == "declaration" {
			extraText = " declaration"
		}
		return fmt.Sprintf("Couldn't add profile. A custom OS updates%s profile already exists.", extraText)
	}

	return "Couldn't add profile. OS updates are already configured. Remove the OS updates settings first."
}

func (e *softwareUpdateProfileError) IsClientError() bool {
	return e.internalErr == nil
}

func (e *softwareUpdateProfileError) StatusCode() int {
	if e.internalErr != nil {
		return http.StatusInternalServerError
	}

	return http.StatusBadRequest
}
