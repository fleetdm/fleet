package fleet

import (
	"crypto/sha1" // nolint:gosec // used for compatibility with existing osquery certificates table schema
	"crypto/x509"
	"fmt"
	"strings"
	"time"
)

// HostCertificateRecord is the database model for a host certificate.
type HostCertificateRecord struct {
	ID     uint `json:"-" db:"id"`
	HostID uint `json:"-" db:"host_id"`

	// SHA1Sum is a SHA-1 hash of the DER encoded certificate.
	SHA1Sum []byte `json:"-" db:"sha1_sum"`

	// CreatedAt is the time the certificate was recorded by Fleet (i.e. certificate initially
	// reported to Fleet).
	CreatedAt time.Time `json:"-" db:"created_at"`
	// DeletedAt is the time the certificate was soft deleted by Fleet (i.e. previously reported to
	// Fleet certificate is subsequently not reported).
	DeletedAt *time.Time `json:"-" db:"deleted_at"`

	// The following fields are extracted from the certificate.
	NotValidAfter        time.Time `json:"-" db:"not_valid_after"`
	NotValidBefore       time.Time `json:"-" db:"not_valid_before"`
	CertificateAuthority bool      `json:"-" db:"certificate_authority"`
	CommonName           string    `json:"-" db:"common_name"`
	KeyAlgorithm         string    `json:"-" db:"key_algorithm"`
	KeyStrength          int       `json:"-" db:"key_strength"`
	KeyUsage             string    `json:"-" db:"key_usage"`
	Serial               string    `json:"-" db:"serial"`
	SigningAlgorithm     string    `json:"-" db:"signing_algorithm"`

	SubjectCountry            string `json:"-" db:"subject_country"`
	SubjectOrganization       string `json:"-" db:"subject_org"`
	SubjectOrganizationalUnit string `json:"-" db:"subject_org_unit"`
	SubjectCommonName         string `json:"-" db:"subject_common_name"`
	IssuerCountry             string `json:"-" db:"issuer_country"`
	IssuerOrganization        string `json:"-" db:"issuer_org"`
	IssuerOrganizationalUnit  string `json:"-" db:"issuer_org_unit"`
	IssuerCommonName          string `json:"-" db:"issuer_common_name"`
}

func NewHostCertificateRecord(
	hostID uint,
	cert *x509.Certificate,
) *HostCertificateRecord {
	hash := sha1.Sum(cert.Raw) // nolint:gosec

	return &HostCertificateRecord{
		HostID:               hostID,
		SHA1Sum:              hash[:], // nolint:gosec
		NotValidAfter:        cert.NotAfter,
		NotValidBefore:       cert.NotBefore,
		CertificateAuthority: cert.IsCA,
		// TODO: we need to define methodology for determining common name analogous to osquery,
		// which seems to preferentially use Subject.CommonName for this value:
		// https://github.com/osquery/osquery/blob/16bb01508eeca6d663b6d4f7e15034306be0fc3d/osquery/tables/system/posix/openssl_utils.cpp#L253
		CommonName:   cert.Subject.CommonName,
		KeyAlgorithm: cert.PublicKeyAlgorithm.String(),
		// TODO: we need to define methodology for determining key strength analogous to osquery,
		// which describes this value as "Key size used for RSA/DSA, or curve name":
		// https://github.com/osquery/osquery/blob/16bb01508eeca6d663b6d4f7e15034306be0fc3d/osquery/tables/system/posix/openssl_utils.cpp#L337
		KeyStrength: 0, // TODO: add key strength here
		// TODO: we need to define methodology for determining key usage analogous to osquery, which
		// describes this as "Certificate key usage and extended key usage":
		// https://github.com/osquery/osquery/blob/16bb01508eeca6d663b6d4f7e15034306be0fc3d/osquery/tables/system/posix/openssl_utils.cpp#L166
		KeyUsage:                  "",
		Serial:                    cert.SerialNumber.String(),
		SigningAlgorithm:          cert.SignatureAlgorithm.String(),
		SubjectCommonName:         cert.Subject.CommonName,
		SubjectCountry:            firstOrEmpty(cert.Subject.Country),            // TODO: confirm methodology
		SubjectOrganization:       firstOrEmpty(cert.Subject.Organization),       // TODO: confirm methodology
		SubjectOrganizationalUnit: firstOrEmpty(cert.Subject.OrganizationalUnit), // TODO: confirm methodology
		IssuerCommonName:          cert.Issuer.CommonName,
		IssuerCountry:             firstOrEmpty(cert.Issuer.Country),            // TODO: confirm methodology
		IssuerOrganization:        firstOrEmpty(cert.Issuer.Organization),       // TODO: confirm methodology
		IssuerOrganizationalUnit:  firstOrEmpty(cert.Issuer.OrganizationalUnit), // TODO: confirm methodology
	}
}

// ToPayload fills a HostCertificatePayload with the fields of a
// HostCertificateRecord. The HostCertificatePayload is used in API responses.
func (r *HostCertificateRecord) ToPayload() *HostCertificatePayload {
	subject := &HostCertificateNameDetails{
		CommonName:         r.SubjectCommonName,
		Country:            r.SubjectCountry,
		Organization:       r.SubjectOrganization,
		OrganizationalUnit: r.SubjectOrganizationalUnit,
	}
	issuer := &HostCertificateNameDetails{
		CommonName:         r.IssuerCommonName,
		Country:            r.IssuerCountry,
		Organization:       r.IssuerOrganization,
		OrganizationalUnit: r.IssuerOrganizationalUnit,
	}
	return &HostCertificatePayload{
		ID:                   r.ID,
		NotValidAfter:        r.NotValidAfter,
		NotValidBefore:       r.NotValidBefore,
		CertificateAuthority: r.CertificateAuthority,
		CommonName:           r.CommonName,
		KeyAlgorithm:         r.KeyAlgorithm,
		KeyStrength:          r.KeyStrength,
		KeyUsage:             r.KeyUsage,
		Serial:               r.Serial,
		SigningAlgorithm:     r.SigningAlgorithm,
		Subject:              subject,
		Issuer:               issuer,
	}
}

// HostCertificatePayload is the JSON model for API endpoints that return host certificates.
type HostCertificatePayload struct {
	ID                   uint      `json:"id"`
	NotValidAfter        time.Time `json:"not_valid_after"`
	NotValidBefore       time.Time `json:"not_valid_before"`
	CertificateAuthority bool      `json:"certificate_authority"`
	CommonName           string    `json:"common_name"`
	KeyAlgorithm         string    `json:"key_algorithm"`
	KeyStrength          int       `json:"key_strength"`
	KeyUsage             string    `json:"key_usage"`
	Serial               string    `json:"serial"`
	SigningAlgorithm     string    `json:"signing_algorithm"`

	Subject *HostCertificateNameDetails `json:"subject,omitempty"`
	Issuer  *HostCertificateNameDetails `json:"issuer,omitempty"`
}

type HostCertificateNameDetails struct {
	CommonName         string `json:"common_name"`
	Country            string `json:"country"`
	Organization       string `json:"organization"`
	OrganizationalUnit string `json:"organizational_unit"`
}

// MDMAppleCertificateListResponse is the plist model for a certificate list response.
// https://developer.apple.com/documentation/devicemanagement/certificatelistresponse
type MDMAppleCertificateListResponse struct {
	CertificateList  []MDMAppleCertificateListItem `plist:"CertificateList"`
	CommandUUID      string                        `plist:"CommandUUID"`
	EnrollmentID     string                        `plist:"EnrollmentID"`
	EnrollmentUserID string                        `plist:"EnrollmentUserID"`
	ErrorChain       []MDMAppleErrorChainItem      `plist:"ErrorChain"`
	NotOnConsole     bool                          `plist:"NotOnConsole"`
	Status           string                        `plist:"Status"`
	UDID             string                        `plist:"UDID"`
	UserID           string                        `plist:"UserID"`
	UserLongName     string                        `plist:"UserLongName"`
	UserShortName    string                        `plist:"UserShortName"`
}

// MDMAppleCertificateListItem is the plist model for a certificate.
// https://developer.apple.com/documentation/devicemanagement/certificatelistresponse/certificatelistitem
type MDMAppleCertificateListItem struct {
	CommonName string `plist:"CommonName"`
	// Data is the DER-encoded certificate.
	Data       []byte `plist:"Data"`
	IsIdentity bool   `plist:"IsIdentity"`
}

func (c *MDMAppleCertificateListItem) Parse(hostID uint) (*HostCertificateRecord, error) {
	cert, err := x509.ParseCertificate(c.Data)
	if err != nil {
		return nil, err
	}
	return NewHostCertificateRecord(hostID, cert), nil
}

// MdmAppleErrorChainItem is the plist model for an error chain item.
// https://developer.apple.com/documentation/devicemanagement/certificatelistresponse/errorchainitem
type MDMAppleErrorChainItem struct {
	ErrorCode            int    `plist:"ErrorCode"`
	ErrorDomain          string `plist:"ErrorDomain"`
	LocalizedDescription string `plist:"LocalizedDescription"`
	USEnglishDescription string `plist:"USEnglishDescription"`
}

// ExtractDetailsFromOsqueryDistinguishedName parses a distinguished name and returns the country,
// organization, and organizational unit. It assumes provided string follows the formatting used by
// osquery `certificates` table[1], which appears to follow the style used by openSSL for `-subj`
// values). Key-value pairs are assumed to be separated by forward slashes, for example:
// "/C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM".
//
// See https://osquery.io/schema/5.15.0/#certificates
func ExtractDetailsFromOsqueryDistinguishedName(str string) (*HostCertificateNameDetails, error) {
	str = strings.TrimSpace(str)
	str = strings.Trim(str, "/")
	parts := strings.Split(str, "/")

	var details HostCertificateNameDetails
	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid distinguished name, wrong key value pair format: %s", str)
		}

		switch strings.ToUpper(kv[0]) {
		case "C":
			details.Country = strings.Trim(kv[1], " ")
		case "O":
			details.Organization = strings.Trim(kv[1], " ")
		case "OU":
			details.OrganizationalUnit = strings.Trim(kv[1], " ")
		case "CN":
			details.CommonName = strings.Trim(kv[1], " ")
		}
	}

	return &details, nil
}

func firstOrEmpty(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}
