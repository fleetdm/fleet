package mdm

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"unicode"

	"github.com/smallstep/pkcs7"
)

// MaxAppleProfileRetries is the maximum number of times an Apple install profile command may be retried after the
// initial attempt, after which it is marked as failed and no further attempts will be made to install the profile.
const MaxAppleProfileRetries = 3

// MaxAppleDeviceNameRetries is the maximum number of times Fleet automatically re-sends the host name template
// to a host (after drift or a failed command) before marking it as failed until a manual resend.
const MaxAppleDeviceNameRetries = 3

// MaxWindowsProfileRetries is the maximum number of times a Windows install profile command may be retried after the
// initial attempt, after which it is marked as failed and no further attempts will be made to install the profile.
const MaxWindowsProfileRetries = 3

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

// cmsContentInfo, cmsEnvelopedData and the types below mirror the part of a CMS message that names its recipient. The
// pkcs7 package keeps its own types unexported, so they are duplicated to read the issuer and serial out of a message.
type cmsContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
}

type cmsEnvelopedData struct {
	Version        int
	RecipientInfos []cmsRecipientInfo `asn1:"set"`
}

type cmsRecipientInfo struct {
	Version               int
	IssuerAndSerialNumber cmsIssuerAndSerial
}

type cmsIssuerAndSerial struct {
	IssuerName   asn1.RawValue
	SerialNumber *big.Int
}

// DecryptBase64CMSWithKey decrypts a DER message, as pkcs7.Encrypt produces, encrypted to a single recipient using
// only that recipient's private key.
//
// CMS matches the recipient by issuer and serial number, so a renewed certificate with the same key and a new serial
// no longer matches what the message names. Fleet encrypts to exactly one certificate, so the recipient named in the
// message is used as the match and the private key does the rest.
func DecryptBase64CMSWithKey(p7Base64 string, key crypto.PrivateKey) (plain []byte, err error) {
	// The pkcs7 package panics on some malformed inputs instead of returning an error, and a stored blob is data a host
	// sent, so a bad one must not take down the caller.
	defer func() {
		if r := recover(); r != nil {
			plain, err = nil, fmt.Errorf("parse CMS message: %v", r)
		}
	}()

	p7Bytes, err := base64.StdEncoding.DecodeString(p7Base64)
	if err != nil {
		return nil, err
	}

	var info cmsContentInfo
	rest, err := asn1.Unmarshal(p7Bytes, &info)
	if err != nil {
		return nil, fmt.Errorf("parse CMS content info: %w", err)
	}
	if len(rest) != 0 {
		return nil, errors.New("parse CMS content info: trailing data")
	}
	if !info.ContentType.Equal(pkcs7.OIDEnvelopedData) {
		return nil, pkcs7.ErrNotEncryptedContent
	}
	// The struct omits the optional originatorInfo and only knows issuer-and-serial recipients, which is all
	// pkcs7.Encrypt produces.
	var env cmsEnvelopedData
	rest, err = asn1.Unmarshal(info.Content.Bytes, &env)
	if err != nil {
		return nil, fmt.Errorf("parse CMS enveloped data: %w", err)
	}
	if len(rest) != 0 {
		return nil, errors.New("parse CMS enveloped data: trailing data")
	}
	if len(env.RecipientInfos) != 1 {
		return nil, fmt.Errorf("CMS message has %d recipients, expected 1", len(env.RecipientInfos))
	}

	p7, err := pkcs7.Parse(p7Bytes)
	if err != nil {
		return nil, err
	}
	recipient := env.RecipientInfos[0].IssuerAndSerialNumber
	return p7.Decrypt(&x509.Certificate{
		RawIssuer:    recipient.IssuerName.FullBytes,
		SerialNumber: recipient.SerialNumber,
	}, key)
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
	if len(encrypted) < nonceSize {
		return "", errors.New("decrypting: ciphertext is shorter than the nonce")
	}
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

	// FleetWindowsEnrollSecretProfileName is the name of the profile used by Fleet to deliver a one-time enroll secret to a
	// Windows host's registry.
	FleetWindowsEnrollSecretProfileName = "Fleetd enroll secret" //nolint:gosec // G101 false positive, a profile name
)

// FleetReservedProfileNames returns a map of PayloadDisplayName or profile
// name strings that are reserved by Fleet.
func FleetReservedProfileNames() map[string]struct{} {
	return map[string]struct{}{
		FleetdConfigProfileName:             {},
		FleetFileVaultProfileName:           {},
		FleetWindowsOSUpdatesProfileName:    {},
		FleetMacOSUpdatesProfileName:        {},
		FleetIOSUpdatesProfileName:          {},
		FleetIPadOSUpdatesProfileName:       {},
		FleetCAConfigProfileName:            {},
		FleetWindowsEnrollSecretProfileName: {},
	}
}

// ListFleetReservedWindowsProfileNames returns a list of PayloadDisplayName strings
// that are reserved by Fleet for Windows.
func ListFleetReservedWindowsProfileNames() []string {
	return []string{FleetWindowsOSUpdatesProfileName, FleetWindowsEnrollSecretProfileName}
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
