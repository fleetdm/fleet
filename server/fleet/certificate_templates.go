package fleet

type CertificateRequestSpec struct {
	Name                   string `json:"name"`
	Team                   string `json:"team,omitempty"`
	CertificateAuthorityId uint   `json:"certificate_authority_id"`
	SubjectName            string `json:"subject_name"`
}

type CertificateTemplate struct {
	Name                   string
	TeamID                 uint
	CertificateAuthorityID uint
	SubjectName            string
}

func (c *CertificateTemplate) AuthzType() string {
	return "certificate_template"
}

type CertificateTemplateResponseSummary struct {
	ID                       uint   `json:"id" db:"id"`
	Name                     string `json:"name" db:"name"`
	SubjectName              string `json:"subject_name" db:"subject_name"`
	CertificateAuthorityId   uint   `json:"certificate_authority_id" db:"certificate_authority_id"`
	CertificateAuthorityName string `json:"certificate_authority_name" db:"certificate_authority_name"`
	CreatedAt                string `json:"created_at" db:"created_at"`
}

// CertificateTemplateResponse contains certificate template details without host-specific data.
type CertificateTemplateResponse struct {
	CertificateTemplateResponseSummary
	CertificateAuthorityType string `json:"certificate_authority_type" db:"certificate_authority_type"`
	TeamID                   uint   `json:"-" db:"team_id"`
}

// CertificateTemplateResponseForHost contains certificate template details with host-specific data.
// Used when a host (Android agent) requests its certificate.
type CertificateTemplateResponseForHost struct {
	CertificateTemplateResponse
	Status                 CertificateTemplateStatus `json:"status" db:"status"`
	SCEPChallenge          *string                   `json:"scep_challenge" db:"scep_challenge"`
	FleetChallenge         *string                   `json:"fleet_challenge" db:"fleet_challenge"`
	SCEPChallengeEncrypted []byte                    `json:"-" db:"scep_challenge_encrypted"`
}

type CertificateTemplateStatus string

var (
	CertificateTemplatePending    CertificateTemplateStatus = "pending"
	CertificateTemplateDelivering CertificateTemplateStatus = "delivering"
	CertificateTemplateDelivered  CertificateTemplateStatus = "delivered"
	CertificateTemplateFailed     CertificateTemplateStatus = "failed"
	CertificateTemplateVerified   CertificateTemplateStatus = "verified"
)

// CertificateTemplateStatusToMDMDeliveryStatus converts a CertificateTemplateStatus to MDMDeliveryStatus.
// This is used when converting HostCertificateTemplate to HostMDMProfile for the GetHost endpoint.
func CertificateTemplateStatusToMDMDeliveryStatus(s CertificateTemplateStatus) MDMDeliveryStatus {
	switch s {
	case CertificateTemplateVerified:
		return MDMDeliveryVerified
	case CertificateTemplateFailed:
		return MDMDeliveryFailed
	default:
		// All in-progress states (pending, delivering, delivered) map to MDMDeliveryPending
		return MDMDeliveryPending
	}
}

// HostCertificateTemplatesForDelivery contains the result of preparing certificate templates
// for delivery to a host. It includes both the templates being transitioned to delivering
// status and the templates that are already installed (verified/delivered).
type HostCertificateTemplatesForDelivery struct {
	// DeliveringTemplateIDs are the certificate template IDs that were transitioned
	// from pending to delivering status in this operation.
	DeliveringTemplateIDs []uint
	// OtherTemplateIDs are other certificate template IDs.
	OtherTemplateIDs []uint
}
