package fleet

type CertificateRequestSpec struct {
	Name                   string `json:"name"`
	Team                   string `json:"team"`
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
	SubjectName              string             `json:"subject_name" db:"subject_name"`
	CertificateAuthorityType string             `json:"certificate_authority_type" db:"certificate_authority_type"`
	Status                   *MDMDeliveryStatus `json:"status" db:"status"`
	SCEPChallenge            *string            `json:"scep_challenge" db:"scep_challenge"`
	FleetChallenge           *string            `json:"fleet_challenge" db:"fleet_challenge"`
	SCEPChallengeEncrypted   []byte             `json:"-" db:"scep_challenge_encrypted"`
	TeamID                   uint               `json:"-" db:"team_id"`
}
