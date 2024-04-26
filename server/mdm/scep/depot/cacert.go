package depot

import (
	"crypto"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/cryptoutil"
)

// CACert represents a new self-signed CA certificate
type CACert struct {
	commonName         string
	country            string
	organization       string
	organizationalUnit string
	years              int
	keyUsage           x509.KeyUsage
}

// NewCACert creates a new CACert object with options
func NewCACert(opts ...CACertOption) *CACert {
	c := &CACert{
		organization:       "scep-ca",
		organizationalUnit: "SCEP CA",
		years:              10,
		keyUsage:           x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type CACertOption func(*CACert)

// WithOrganization specifies the Organization on the CA template.
func WithOrganization(o string) CACertOption {
	return func(c *CACert) {
		c.organization = o
	}
}

// WithOrganizationalUnit specifies the OrganizationalUnit on the CA template.
func WithOrganizationalUnit(ou string) CACertOption {
	return func(c *CACert) {
		c.organizationalUnit = ou
	}
}

// WithYears specifies the validity date of the CA.
func WithYears(y int) CACertOption {
	return func(c *CACert) {
		c.years = y
	}
}

// WithCountry specifies the Country on the CA template.
func WithCountry(country string) CACertOption {
	return func(c *CACert) {
		c.country = country
	}
}

// WithCommonName specifies the CommonName on the CA template.
func WithCommonName(name string) CACertOption {
	return func(c *CACert) {
		c.commonName = name
	}
}

// WithKeyUsage specifies the X.509 Key Usage on the CA template.
func WithKeyUsage(usage x509.KeyUsage) CACertOption {
	return func(c *CACert) {
		c.keyUsage = usage
	}
}

// newPkixName creates a new pkix.Name from c
func (c *CACert) newPkixName() *pkix.Name {
	return &pkix.Name{
		Country:            []string{c.country},
		Organization:       []string{c.organization},
		OrganizationalUnit: []string{c.organizationalUnit},
		CommonName:         c.commonName,
	}
}

// SelfSign creates an x509 template based off our settings and self-signs it using priv.
func (c *CACert) SelfSign(rand io.Reader, pub crypto.PublicKey, priv interface{}) ([]byte, error) {
	subjKeyId, err := cryptoutil.GenerateSubjectKeyID(pub)
	if err != nil {
		return nil, err
	}
	// Build CA based on RFC5280
	tmpl := x509.Certificate{
		Subject:      *c.newPkixName(),
		SerialNumber: big.NewInt(1),

		// NotBefore is set to be 10min earlier to fix gap on time difference in cluster
		NotBefore: time.Now().Add(-600).UTC(),
		NotAfter:  time.Now().AddDate(c.years, 0, 0).UTC(),

		// Used for certificate signing only
		KeyUsage: c.keyUsage,

		// activate CA
		BasicConstraintsValid: true,
		IsCA:                  true,

		// Not allow any non-self-issued intermediate CA
		MaxPathLen: 0,

		// 160-bit SHA-1 hash of the value of the BIT STRING subjectPublicKey
		// (excluding the tag, length, and number of unused bits)
		SubjectKeyId: subjKeyId,
	}

	return x509.CreateCertificate(rand, &tmpl, &tmpl, pub, priv)
}
