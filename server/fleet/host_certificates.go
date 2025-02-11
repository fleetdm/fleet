package fleet

import (
	"encoding/base64"
	"time"
)

// HostCertificateDB is the database model for a host certificate.
type HostCertificateDB struct {
	HostID        uint `json:"-" db:"host_id"`
	CertificateID uint `json:"-" db:"cert_id"`

	// Checksum is a SHA-1 hash of the DER encoded certificate.
	Checksum []byte `json:"-" db:"checksum"`

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
