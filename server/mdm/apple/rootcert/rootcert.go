package rootcert

import (
	"crypto/x509"
	_ "embed"
	"fmt"
)

// appleRootCert is https://www.apple.com/appleca/AppleIncRootCertificate.cer
//
//go:embed AppleIncRootCertificate.cer
var appleRootCert []byte

// AppleRootCA is Apple's Root CA parsed to an *x509.Certificate
var AppleRootCA = NewAppleCert(appleRootCert)

func NewAppleCert(crt []byte) *x509.Certificate {
	cert, err := x509.ParseCertificate(crt)
	if err != nil {
		panic(fmt.Errorf("could not parse cert: %w", err))
	}
	return cert
}
