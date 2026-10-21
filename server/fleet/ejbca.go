package fleet

import (
	"context"
)

// EJBCA is what Fleet needs to enroll certificates from an EJBCA instance over its REST API with mutual TLS. Callers expand the Fleet
// variables in the common name, username, and UPNs before enrolling.
type EJBCA struct {
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

	// ClientP12 is the base64 PKCS#12 with the client certificate and key, and Password opens it.
	ClientP12 string `json:"client_p12"`
	Password  string `json:"password"`
}

func (e *EJBCA) Preprocess() {
	e.Name = Preprocess(e.Name)
	e.URL = Preprocess(e.URL)
	e.CertificateAuthorityName = Preprocess(e.CertificateAuthorityName)
	e.CertificateProfileName = Preprocess(e.CertificateProfileName)
	e.EndEntityProfileName = Preprocess(e.EndEntityProfileName)
}

// EJBCAUpdatePayload changes the fields it sets and leaves the others as they are.
type EJBCAUpdatePayload struct {
	Name                          *string   `json:"name"`
	URL                           *string   `json:"url"`
	ClientP12                     *string   `json:"client_p12"`
	Password                      *string   `json:"password"`
	TrustCABundle                 *string   `json:"trust_ca_bundle"`
	CertificateAuthorityName      *string   `json:"certificate_authority_name"`
	CertificateProfileName        *string   `json:"certificate_profile_name"`
	EndEntityProfileName          *string   `json:"end_entity_profile_name"`
	CertificateCommonName         *string   `json:"certificate_common_name"`
	UsernameTemplate              *string   `json:"username_template"`
	CertificateUserPrincipalNames *[]string `json:"certificate_user_principal_names"`
}

// IsEmpty checks if the struct only has all empty values
func (p EJBCAUpdatePayload) IsEmpty() bool {
	return p == EJBCAUpdatePayload{}
}

func (p *EJBCAUpdatePayload) Preprocess() {
	for _, field := range []*string{p.Name, p.URL, p.CertificateAuthorityName, p.CertificateProfileName, p.EndEntityProfileName} {
		if field != nil {
			*field = Preprocess(*field)
		}
	}
}

// EJBCACertificate is the certificate EJBCA issued for a host.
type EJBCACertificate = IssuedCertificate

type EJBCAService interface {
	// VerifyConnection checks that EJBCA accepts the client certificate, that the CA is one it may use, and, when EJBCA can look it up,
	// that the certificate profile exists.
	VerifyConnection(ctx context.Context, config EJBCA) error
	// GetCertificate enrolls a certificate and returns it with a newly generated key in a PKCS#12 bundle.
	GetCertificate(ctx context.Context, config EJBCA) (*EJBCACertificate, error)
}
