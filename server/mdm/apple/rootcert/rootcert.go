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

// appleRootCertG2 is https://www.apple.com/certificateauthority/AppleRootCA-G2.cer
// (SHA-256 fingerprint C2B9B042DD57830E7D117DAC55AC8AE19407D38E41D88F3215BC3A890444A050)
//
//go:embed AppleRootCA-G2.cer
var appleRootCertG2 []byte

// AppleRootCAG2 is Apple's Root CA - G2 (RSA-4096) parsed to an *x509.Certificate
var AppleRootCAG2 = NewAppleCert(appleRootCertG2)

// appleRootCertG3 is https://www.apple.com/certificateauthority/AppleRootCA-G3.cer
// (SHA-256 fingerprint 63343ABFB89A6A03EBB57E9B3F5FA7BE7C4F5C756F3017B3A8C488C3653E9179)
//
//go:embed AppleRootCA-G3.cer
var appleRootCertG3 []byte

// AppleRootCAG3 is Apple's Root CA - G3 (ECC) parsed to an *x509.Certificate
var AppleRootCAG3 = NewAppleCert(appleRootCertG3)

// AppleRootCAs returns all pinned Apple root CAs that Apple device-identity
// certificate chains may terminate at.
func AppleRootCAs() []*x509.Certificate {
	return []*x509.Certificate{AppleRootCA, AppleRootCAG2, AppleRootCAG3}
}

func NewAppleCert(crt []byte) *x509.Certificate {
	cert, err := x509.ParseCertificate(crt)
	if err != nil {
		panic(fmt.Errorf("could not parse cert: %w", err))
	}
	return cert
}
