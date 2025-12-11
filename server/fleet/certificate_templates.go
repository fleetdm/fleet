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

type CertificateTemplateResponseFull struct {
	CertificateTemplateResponseSummary
	CertificateAuthorityType string             `json:"certificate_authority_type" db:"certificate_authority_type"`
	Status                   *MDMDeliveryStatus `json:"status" db:"status"`
	SCEPChallenge            *string            `json:"scep_challenge" db:"scep_challenge"`
	FleetChallenge           *string            `json:"fleet_challenge" db:"fleet_challenge"`
	SCEPChallengeEncrypted   []byte             `json:"-" db:"scep_challenge_encrypted"`
	TeamID                   uint               `json:"-" db:"team_id"`
}

// CertificateTemplateDeviceResponseFull should merge with CertificateTemplateResponseFull
// as part of https://github.com/fleetdm/fleet/issues/36684 work.
type CertificateTemplateDeviceResponseFull struct {
	CertificateTemplateResponseSummary
	SubjectName              string                     `json:"subject_name"`
	CertificateAuthorityType string                     `json:"certificate_authority_type"`
	Status                   *CertificateTemplateStatus `json:"status"`
	SCEPChallenge            *string                    `json:"scep_challenge"`
	FleetChallenge           *string                    `json:"fleet_challenge"`
}

type CertificateTemplateStatus string

var (
	CertificateTemplateDelivered CertificateTemplateStatus = "delivered"
	CertificateTemplateFailed    CertificateTemplateStatus = "failed"
	CertificateTemplateVerified  CertificateTemplateStatus = "verified"
)

// ToDeviceResponse converts a CertificateTemplateResponseFull to CertificateTemplateDeviceResponseFull.
// It maps the MDMDeliveryStatus to CertificateTemplateStatus.
func (c *CertificateTemplateResponseFull) ToDeviceResponse() *CertificateTemplateDeviceResponseFull {
	var status *CertificateTemplateStatus
	if c.Status != nil {
		var s CertificateTemplateStatus
		switch *c.Status {
		case MDMDeliveryVerified:
			s = CertificateTemplateVerified
		case MDMDeliveryFailed:
			s = CertificateTemplateFailed
		default:
			// The only other expected status is MDMDeliveryPending.
			// If it's anything else, we assume it's delivered so that Android agent will fetch the certificate.
			s = CertificateTemplateDelivered
		}
		status = &s
	}

	return &CertificateTemplateDeviceResponseFull{
		CertificateTemplateResponseSummary: c.CertificateTemplateResponseSummary,
		SubjectName:                        c.SubjectName,
		CertificateAuthorityType:           c.CertificateAuthorityType,
		Status:                             status,
		SCEPChallenge:                      c.SCEPChallenge,
		FleetChallenge:                     c.FleetChallenge,
	}
}
