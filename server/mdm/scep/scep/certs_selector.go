package scep

import (
	"bytes"
	"crypto"
	"crypto/x509"
)

// A CertsSelector filters certificates.
type CertsSelector interface {
	SelectCerts([]*x509.Certificate) []*x509.Certificate
}

// CertsSelectorFunc is a type of function that filters certificates.
type CertsSelectorFunc func([]*x509.Certificate) []*x509.Certificate

func (f CertsSelectorFunc) SelectCerts(certs []*x509.Certificate) []*x509.Certificate {
	return f(certs)
}

// NopCertsSelector returns a CertsSelectorFunc that does not do anything.
func NopCertsSelector() CertsSelectorFunc {
	return func(certs []*x509.Certificate) []*x509.Certificate {
		return certs
	}
}

// A EnciphermentCertsSelector returns a CertsSelectorFunc that selects
// certificates eligible for key encipherment. This certsSelector can be used
// to filter PKCSReq recipients.
func EnciphermentCertsSelector() CertsSelectorFunc {
	return func(certs []*x509.Certificate) (selected []*x509.Certificate) {
		enciphermentKeyUsages := x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment
		for _, cert := range certs {
			if cert.KeyUsage&enciphermentKeyUsages != 0 {
				selected = append(selected, cert)
			}
		}
		return selected
	}
}

// FingerprintCertsSelector selects a certificate that matches hash using
// hashType against the digest of the raw certificate DER bytes
func FingerprintCertsSelector(hashType crypto.Hash, hash []byte) CertsSelectorFunc {
	return func(certs []*x509.Certificate) (selected []*x509.Certificate) {
		for _, cert := range certs {
			h := hashType.New()
			h.Write(cert.Raw)
			if bytes.Compare(hash, h.Sum(nil)) == 0 {
				selected = append(selected, cert)
				return
			}
		}
		return
	}
}
