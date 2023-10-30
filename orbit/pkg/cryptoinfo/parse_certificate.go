package cryptoinfo

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"net"
	"net/url"
	"time"
)

type certExtract struct {
	CRLDistributionPoints       []string
	DNSNames                    []string
	EmailAddresses              []string
	ExcludedDNSDomains          []string
	ExcludedEmailAddresses      []string
	ExcludedIPRanges            []*net.IPNet
	ExcludedURIDomains          []string
	IPAddresses                 []net.IP
	Issuer                      pkix.Name
	IssuerParsed                string
	IssuingCertificateURL       []string
	KeyUsage                    int
	KeyUsageParsed              []string
	NotBefore, NotAfter         time.Time
	OCSPServer                  []string
	PermittedDNSDomains         []string
	PermittedDNSDomainsCritical bool
	PermittedEmailAddresses     []string
	PermittedIPRanges           []*net.IPNet
	PermittedURIDomains         []string
	PublicKeyAlgorithm          string
	SerialNumber                string
	SignatureAlgorithm          string
	Subject                     pkix.Name
	SubjectParsed               string
	URIs                        []*url.URL
	Version                     int
}

// parseCertificate parses a certificate from a stream of bytes. We use this, instead of a bare x509.ParseCertificate, to handle some
// string conversions, and bitfield enumerations.
func parseCertificate(certBytes []byte) (interface{}, error) {
	c, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing certificate: %w", err)
	}

	return extractCert(c)
}

func extractCert(c *x509.Certificate) (interface{}, error) {
	return &certExtract{
		CRLDistributionPoints: c.CRLDistributionPoints,
		DNSNames:              c.DNSNames,
		EmailAddresses:        c.EmailAddresses,
		IPAddresses:           c.IPAddresses,
		Issuer:                c.Issuer,
		IssuerParsed:          c.Issuer.String(),
		IssuingCertificateURL: c.IssuingCertificateURL,
		KeyUsage:              int(c.KeyUsage),
		KeyUsageParsed:        keyUsageToStrings(c.KeyUsage),
		NotAfter:              c.NotAfter,
		NotBefore:             c.NotBefore,
		OCSPServer:            c.OCSPServer,
		PublicKeyAlgorithm:    c.PublicKeyAlgorithm.String(),
		SerialNumber:          c.SerialNumber.String(),
		SignatureAlgorithm:    c.SignatureAlgorithm.String(),
		Subject:               c.Subject,
		SubjectParsed:         c.Subject.String(),
		URIs:                  c.URIs,
		Version:               c.Version,
	}, nil
}

var keyUsageBits = map[x509.KeyUsage]string{
	x509.KeyUsageDigitalSignature:  "Digital Signature",
	x509.KeyUsageContentCommitment: "Content Commitment",
	x509.KeyUsageKeyEncipherment:   "Key Encipherment",
	x509.KeyUsageDataEncipherment:  "Data Encipherment",
	x509.KeyUsageKeyAgreement:      "Key Agreement",
	x509.KeyUsageCertSign:          "Certificate Sign",
	x509.KeyUsageCRLSign:           "CRL Sign",
	x509.KeyUsageEncipherOnly:      "Encipher Only",
	x509.KeyUsageDecipherOnly:      "Decipher Only",
}

func keyUsageToStrings(k x509.KeyUsage) []string {
	var usage []string

	for usageBit, usageMeaning := range keyUsageBits {
		if k&usageBit != 0 {
			usage = append(usage, usageMeaning)
		}
	}

	return usage
}
