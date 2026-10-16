package fleet

import (
	"context"
	"time"
)

// EJBCACA is what Fleet needs to enroll certificates from an EJBCA instance over its REST API with mutual TLS. Callers expand the Fleet
// variables in the common name, username, and UPNs before enrolling.
type EJBCACA struct {
	ID   uint   `json:"-"`
	Name string `json:"name"`
	URL  string `json:"url"`

	// MTLSCertPEM is the client certificate chain, leaf first, and MTLSKeyPEM its private key.
	MTLSCertPEM string `json:"-"`
	MTLSKeyPEM  string `json:"-"`
	// TrustCABundlePEM verifies EJBCA's HTTPS certificate. The system roots are used when it is empty.
	TrustCABundlePEM string `json:"trust_ca_bundle"`

	// The names of the EJBCA CA and profiles to enroll with.
	CertificateAuthorityName string `json:"certificate_authority_name"`
	CertificateProfileName   string `json:"certificate_profile_name"`
	EndEntityProfileName     string `json:"end_entity_profile_name"`

	CertificateCommonName         string   `json:"certificate_common_name"`
	UsernameTemplate              string   `json:"username_template"`
	CertificateUserPrincipalNames []string `json:"certificate_user_principal_names"`
}

type EJBCACertificate struct {
	PfxData        []byte
	Password       string
	NotValidBefore time.Time
	NotValidAfter  time.Time
	SerialNumber   string
}

type EJBCAService interface {
	// VerifyConnection checks that EJBCA accepts the client certificate, that the CA is one it may use, and, when EJBCA can look it up,
	// that the certificate profile exists.
	VerifyConnection(ctx context.Context, config EJBCACA) error
	// GetCertificate enrolls a certificate and returns it with a newly generated key in a PKCS#12 bundle.
	GetCertificate(ctx context.Context, config EJBCACA) (*EJBCACertificate, error)
}
