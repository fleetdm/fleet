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
	CertificateAuthorityId   uint   `json:"certificate_authority_id" db:"certificate_authority_id"`
	CertificateAuthorityName string `json:"certificate_authority_name" db:"certificate_authority_name"`
	CreatedAt                string `json:"created_at" db:"created_at"`
}

type CertificateTemplateResponseFull struct {
	CertificateTemplateResponseSummary
	SubjectName              string                     `json:"subject_name" db:"subject_name"`
	CertificateAuthorityType string                     `json:"certificate_authority_type" db:"certificate_authority_type"`
	Status                   *CertificateTemplateStatus `json:"status" db:"status"`
	SCEPChallenge            *string                    `json:"scep_challenge" db:"scep_challenge"`
	FleetChallenge           *string                    `json:"fleet_challenge" db:"fleet_challenge"`
	SCEPChallengeEncrypted   []byte                     `json:"-" db:"scep_challenge_encrypted"`
	TeamID                   uint                       `json:"-" db:"team_id"`
}

type CertificateTemplateStatus string

var (
	CertificateTemplateDelivered CertificateTemplateStatus = "delivered"
	CertificateTemplateFailed    CertificateTemplateStatus = "failed"
	CertificateTemplateVerified  CertificateTemplateStatus = "verified"
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
		// All other states (delivered, etc.) map to pending as in-progress states
		return MDMDeliveryPending
	}
}
