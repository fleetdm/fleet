package fleet

import (
	"crypto/sha1" // nolint:gosec // used for compatibility with existing osquery data
	"crypto/x509"
	"encoding/base64"
	"errors"
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

	NotValidAfter             time.Time `json:"-" db:"not_valid_after"`
	NotValidBefore            time.Time `json:"-" db:"not_valid_before"`
	CertificateAuthority      bool      `json:"-" db:"certificate_authority"`
	CommonName                string    `json:"-" db:"common_name"`
	KeyAlgorithm              string    `json:"-" db:"key_algorithm"`
	KeyStrength               int       `json:"-" db:"key_strength"`
	KeyUsage                  string    `json:"-" db:"key_usage"`
	Serial                    string    `json:"-" db:"serial"`
	SigningAlgorithm          string    `json:"-" db:"signing_algorithm"`
	SubjectCountry            string    `json:"-" db:"subject_country"`
	SubjectOrganization       string    `json:"-" db:"subject_org"`
	SubjectOrganizationalUnit string    `json:"-" db:"subject_org_unit"`
	SubjectCommonName         string    `json:"-" db:"subject_common_name"`
	IssuerCountry             string    `json:"-" db:"issuer_country"`
	IssuerOrganization        string    `json:"-" db:"issuer_org"`
	IssuerOrganizationalUnit  string    `json:"-" db:"issuer_org_unit"`
	IssuerCommonName          string    `json:"-" db:"issuer_common_name"`
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
	// Data is the DER encoded certificate.
	Data       b64Data `plist:"Data"`
	IsIdentity bool    `plist:"IsIdentity"`
}

func (c *MDMAppleCertificateListItem) Parse() (*HostCertificateRecord, error) {
	hash := sha1.Sum(c.Data) // nolint:gosec

	parsed, err := x509.ParseCertificate(c.Data)
	if err != nil {
		return nil, err
	}

	return &HostCertificateRecord{
		SHA1Sum:              hash[:],
		NotValidBefore:       parsed.NotBefore,
		NotValidAfter:        parsed.NotAfter,
		CertificateAuthority: parsed.IsCA,
		// TODO: we need to define methodology for determining common name analogous to osquery,
		// which seems to preferentially use Subject.CommonName for this value:
		// https://github.com/osquery/osquery/blob/16bb01508eeca6d663b6d4f7e15034306be0fc3d/osquery/tables/system/posix/openssl_utils.cpp#L253
		CommonName:   parsed.Subject.CommonName,
		KeyAlgorithm: parsed.PublicKeyAlgorithm.String(),
		// TODO: we need to define methodology for determining key strength analogous to osquery,
		// which describes this value as "Key size used for RSA/DSA, or curve name":
		// https://github.com/osquery/osquery/blob/16bb01508eeca6d663b6d4f7e15034306be0fc3d/osquery/tables/system/posix/openssl_utils.cpp#L337
		KeyStrength: 0,
		// TODO: we need to define methodology for determining key usage analogous to osquery, which
		// describes this as "Certificate key usage and extended key usage":
		// https://github.com/osquery/osquery/blob/16bb01508eeca6d663b6d4f7e15034306be0fc3d/osquery/tables/system/posix/openssl_utils.cpp#L166
		KeyUsage:                  "",
		Serial:                    parsed.SerialNumber.String(),
		SigningAlgorithm:          parsed.SignatureAlgorithm.String(),
		SubjectCommonName:         parsed.Subject.CommonName,
		SubjectCountry:            parsed.Subject.Country[0],            // TODO: confirm methodology
		SubjectOrganization:       parsed.Subject.Organization[0],       // TODO: confirm methodology
		SubjectOrganizationalUnit: parsed.Subject.OrganizationalUnit[0], // TODO: confirm methodology
		IssuerCommonName:          parsed.Issuer.CommonName,
		IssuerCountry:             parsed.Issuer.Country[0],            // TODO: confirm methodology
		IssuerOrganization:        parsed.Issuer.Organization[0],       // TODO: confirm methodology
		IssuerOrganizationalUnit:  parsed.Issuer.OrganizationalUnit[0], // TODO: confirm methodology
	}, nil
}

// MdmAppleErrorChainItem is the plist model for an error chain item.
// https://developer.apple.com/documentation/devicemanagement/certificatelistresponse/errorchainitem
type MDMAppleErrorChainItem struct {
	ErrorCode            int    `plist:"ErrorCode"`
	ErrorDomain          string `plist:"ErrorDomain"`
	LocalizedDescription string `plist:"LocalizedDescription"`
	USEnglishDescription string `plist:"USEnglishDescription"`
}

// b64Data is a byte slice that can be base64 encoded.
type b64Data []byte

// String returns the base64-encoded string form of b
func (b b64Data) String() string {
	return base64.StdEncoding.EncodeToString(b)
}

// ExtractDetailsFromOsqueryDistinguishedName parses a distinguished name and returns the country,
// organization, and organizational unit. It assumes provided string follows the formatting used by
// osquery `certificates` table[1], which appears to follow the style used by openSSL for `-subj`
// values). Key-value pairs are assumed to be separated by forward slashes, for example:
// "/C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM".
//
// See https://osquery.io/schema/5.15.0/#certificates
func ExtractDetailsFromOsqueryDistinguishedName(str string) (*HostCertificateNameDetails, error) {
	// TODO
	return nil, errors.New("not implemented")
}
