package testing_utils

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"time"
)

func NewTestMDMAppleCertTemplate() *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			ExtraNames: []pkix.AttributeTypeAndValue{
				{
					Type:  asn1.ObjectIdentifier{0, 9, 2342, 19200300, 100, 1, 1},
					Value: "com.apple.mgmt.Example",
				},
			},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
}
