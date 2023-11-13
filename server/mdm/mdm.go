package mdm

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"unicode"

	"go.mozilla.org/pkcs7"
)

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

func GetRawProfilePlatform(profile []byte) string {
	// Function to compare byte slices case-insensitively for a specified length
	compareBytesCaseInsensitive := func(slice1, slice2 []byte, length int) bool {
		for i := 0; i < length && i < len(slice1) && i < len(slice2); i++ {
			if unicode.ToLower(rune(slice1[i])) != unicode.ToLower(rune(slice2[i])) {
				return false
			}
		}
		return true
	}

	// Trimming leading whitespaces
	trimmedProfile := bytes.TrimSpace(profile)

	// Checking for darwin platform with case-insensitive comparison
	darwinPrefix := []byte("<?xml")
	if len(trimmedProfile) >= len(darwinPrefix) && compareBytesCaseInsensitive(trimmedProfile, darwinPrefix, len(darwinPrefix)) {
		return "darwin"
	}

	// Checking for windows platform with case-insensitive comparison
	windowsPrefix := []byte("<replace")
	if len(trimmedProfile) >= len(windowsPrefix) && compareBytesCaseInsensitive(trimmedProfile, windowsPrefix, len(windowsPrefix)) {
		return "windows"
	}

	return ""
}
