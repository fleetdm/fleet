// Package fleetcrt contains utilities to load and process TLS certificate files.
package fleetcrt

import (
	"crypto/tls"
	"errors"
	"fmt"
	"os"
)

// Certificate holds a loaded certificate and its raw parts.
type Certificate struct {
	Crt    tls.Certificate
	RawCrt []byte
	RawKey []byte
}

// LoadCertificateFromFiles loads a TLS certificate from PEM cert and key file paths.
//
// Returns (nil, nil) if both files do not exist.
func LoadCertificateFromFiles(crtPath, keyPath string) (*Certificate, error) {
	checkFileExists := func(filePath string) (bool, error) {
		switch s, err := os.Stat(filePath); {
		case err == nil:
			return !s.IsDir(), nil
		case errors.Is(err, os.ErrNotExist):
			return false, nil
		default:
			return false, err
		}
	}

	crtExists, err := checkFileExists(crtPath)
	if err != nil {
		return nil, err
	}
	keyExists, err := checkFileExists(keyPath)
	if err != nil {
		return nil, err
	}

	if crtExists != keyExists {
		return nil, fmt.Errorf(
			"both crt and key files must exist: %s: %t, %s: %t",
			crtPath, crtExists, keyPath, keyExists,
		)
	}
	if !crtExists {
		return nil, nil
	}

	crtBytes, err := os.ReadFile(crtPath)
	if err != nil {
		return nil, err
	}
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	crt, err := tls.X509KeyPair(crtBytes, keyBytes)
	if err != nil {
		return nil, err
	}

	return &Certificate{
		Crt:    crt,
		RawCrt: crtBytes,
		RawKey: keyBytes,
	}, nil
}

// LoadCertificate loads a certificate from the given PEM cert and key strings.
//
// Returns (nil, nil) if both values are empty.
func LoadCertificate(crt, key string) (*tls.Certificate, error) {
	if (crt != "") != (key != "") {
		return nil, fmt.Errorf(
			"both crt and key must be set: crt=%t, key=%t", crt != "", key != "",
		)
	}
	if crt == "" {
		return nil, nil
	}

	cert, err := tls.X509KeyPair([]byte(crt), []byte(key))
	if err != nil {
		return nil, err
	}

	return &cert, nil
}
