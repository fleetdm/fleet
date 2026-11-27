package fleet

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

// AndroidCACertificateProfileID is used by the frontend to identify static Android CA profiles.
const AndroidCACertificateProfileID = "fleet-host-android-ca-certificate"

// MaxAndroidCACertificateSize is the max file size for an uploaded CA certificate (64 KB).
const MaxAndroidCACertificateSize = 64 * 1024

// AndroidCACertificate represents a static Root or Intermediate CA certificate distributed to Android hosts.
type AndroidCACertificate struct {
	ID                uint       `json:"id" db:"id"`
	TeamID            uint       `json:"team_id" db:"team_id" renameto:"fleet_id"`
	Name              string     `json:"name" db:"name"`
	CertificatePEM    string     `json:"certificate_pem,omitempty" db:"certificate_pem"`
	SHA256Fingerprint string     `json:"sha256_fingerprint" db:"sha256_fingerprint"`
	Subject           string     `json:"subject" db:"subject"`
	Issuer            string     `json:"issuer" db:"issuer"`
	NotValidBefore    *time.Time `json:"not_valid_before" db:"not_valid_before"`
	NotValidAfter     *time.Time `json:"not_valid_after" db:"not_valid_after"`
	LabelsIncludeAll  []string   `json:"labels_include_all,omitempty" db:"-"`
	LabelsIncludeAny  []string   `json:"labels_include_any,omitempty" db:"-"`
	LabelsExcludeAny  []string   `json:"labels_exclude_any,omitempty" db:"-"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

func (c *AndroidCACertificate) AuthzType() string {
	return "android_ca_certificate"
}

// AndroidCACertificateResponse is the summary returned for admin list/get responses.
type AndroidCACertificateResponse struct {
	ID                uint       `json:"id"`
	TeamID            uint       `json:"team_id" renameto:"fleet_id"`
	Name              string     `json:"name"`
	SHA256Fingerprint string     `json:"sha256_fingerprint"`
	Subject           string     `json:"subject"`
	Issuer            string     `json:"issuer"`
	NotValidBefore    *time.Time `json:"not_valid_before"`
	NotValidAfter     *time.Time `json:"not_valid_after"`
	LabelsIncludeAll  []string   `json:"labels_include_all,omitempty"`
	LabelsIncludeAny  []string   `json:"labels_include_any,omitempty"`
	LabelsExcludeAny  []string   `json:"labels_exclude_any,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// ToResponse converts an AndroidCACertificate to AndroidCACertificateResponse.
func (c *AndroidCACertificate) ToResponse() *AndroidCACertificateResponse {
	if c == nil {
		return nil
	}
	return &AndroidCACertificateResponse{
		ID:                c.ID,
		TeamID:            c.TeamID,
		Name:              c.Name,
		SHA256Fingerprint: c.SHA256Fingerprint,
		Subject:           c.Subject,
		Issuer:            c.Issuer,
		NotValidBefore:    c.NotValidBefore,
		NotValidAfter:     c.NotValidAfter,
		LabelsIncludeAll:  c.LabelsIncludeAll,
		LabelsIncludeAny:  c.LabelsIncludeAny,
		LabelsExcludeAny:  c.LabelsExcludeAny,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
}

// HostAndroidCACertificate represents an Android host's assignment and delivery status of a static CA.
type HostAndroidCACertificate struct {
	ID                uint              `json:"id" db:"id"`
	HostID            uint              `json:"host_id" db:"host_id"`
	HostUUID          string            `json:"host_uuid" db:"host_uuid"`
	CACertificateID   uint              `json:"ca_certificate_id" db:"ca_certificate_id"`
	CertificateName   string            `json:"certificate_name" db:"certificate_name"`
	Subject           string            `json:"subject" db:"subject"`
	Issuer            string            `json:"issuer" db:"issuer"`
	SHA256Fingerprint string            `json:"sha256_fingerprint" db:"sha256_fingerprint"`
	NotValidBefore    *time.Time        `json:"not_valid_before" db:"not_valid_before"`
	NotValidAfter     *time.Time        `json:"not_valid_after" db:"not_valid_after"`
	Status            MDMDeliveryStatus `json:"status" db:"status"`
	Detail            string            `json:"detail,omitempty" db:"detail"`
	CreatedAt         time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at" db:"updated_at"`
}

// ToHostMDMProfile converts a HostAndroidCACertificate to HostMDMProfile for OS Settings on host details.
func (p *HostAndroidCACertificate) ToHostMDMProfile() HostMDMProfile {
	if p == nil {
		return HostMDMProfile{}
	}
	status := string(p.Status)
	caCertID := p.CACertificateID
	return HostMDMProfile{
		HostUUID:              p.HostUUID,
		Name:                  p.CertificateName,
		Platform:              "android",
		Status:                &status,
		OperationType:         MDMOperationTypeInstall,
		ProfileUUID:           AndroidCACertificateProfileID,
		CertificateTemplateID: &caCertID,
		Detail:                p.Detail,
	}
}

// ParsedAndroidCACertificate holds the parsed and validated certificate information.
type ParsedAndroidCACertificate struct {
	CertificatePEM    string
	SHA256Fingerprint string
	Subject           string
	Issuer            string
	NotValidBefore    time.Time
	NotValidAfter     time.Time
	CommonName        string
	Certificates      []*x509.Certificate
}

// ValidateAndParseAndroidCACertificate validates and parses an uploaded PEM or DER certificate.
func ValidateAndParseAndroidCACertificate(certBytes []byte) (*ParsedAndroidCACertificate, error) {
	if len(certBytes) == 0 {
		return nil, &BadRequestError{Message: "Certificate file cannot be empty."}
	}
	if len(certBytes) > MaxAndroidCACertificateSize {
		return nil, &BadRequestError{Message: fmt.Sprintf("Certificate file exceeds maximum allowed size of %d KB.", MaxAndroidCACertificateSize/1024)}
	}

	rest := certBytes
	var certBlocks []*pem.Block
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		// Reject files containing any PRIVATE KEY blocks
		if strings.Contains(strings.ToUpper(block.Type), "PRIVATE KEY") {
			return nil, &BadRequestError{Message: "Certificate file must not contain private keys."}
		}
		if block.Type == "CERTIFICATE" {
			certBlocks = append(certBlocks, block)
		}
	}

	var derCerts [][]byte
	if len(certBlocks) > 0 {
		for _, b := range certBlocks {
			derCerts = append(derCerts, b.Bytes)
		}
	} else {
		// If no PEM blocks found, check if it's raw DER encoded certificate
		if _, err := x509.ParseCertificate(certBytes); err == nil {
			derCerts = append(derCerts, certBytes)
		} else {
			return nil, &BadRequestError{Message: "Invalid certificate format. Expected PEM or DER encoded X.509 certificate."}
		}
	}

	var x509Certs []*x509.Certificate
	for _, der := range derCerts {
		parsed, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, &BadRequestError{Message: fmt.Sprintf("Failed to parse X.509 certificate: %v", err)}
		}
		x509Certs = append(x509Certs, parsed)
	}

	if len(x509Certs) == 0 {
		return nil, &BadRequestError{Message: "No valid X.509 certificate found in file."}
	}

	primaryCert := x509Certs[0]
	fingerprint := sha256.Sum256(primaryCert.Raw)
	fingerprintHex := hex.EncodeToString(fingerprint[:])

	var pemBuf bytes.Buffer
	for _, cert := range x509Certs {
		if err := pem.Encode(&pemBuf, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		}); err != nil {
			return nil, fmt.Errorf("encoding pem: %w", err)
		}
	}

	return &ParsedAndroidCACertificate{
		CertificatePEM:    pemBuf.String(),
		SHA256Fingerprint: fingerprintHex,
		Subject:           primaryCert.Subject.String(),
		Issuer:            primaryCert.Issuer.String(),
		NotValidBefore:    primaryCert.NotBefore,
		NotValidAfter:     primaryCert.NotAfter,
		CommonName:        primaryCert.Subject.CommonName,
		Certificates:      x509Certs,
	}, nil
}
