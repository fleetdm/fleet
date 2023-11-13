package mdm

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/base64"

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
	// trim leading whitespaces
	trimmedProfile := bytes.TrimSpace(profile)

	if len(trimmedProfile) == 0 {
		return ""
	}

	darwinPrefix := []byte("<?xml")
	if bytes.EqualFold(darwinPrefix, trimmedProfile[:len(darwinPrefix)]) {
		return "darwin"
	}

	windowsPrefix := []byte("<replace")
	if bytes.EqualFold(windowsPrefix, trimmedProfile[:len(windowsPrefix)]) {
		return "windows"
	}

	return ""
}
