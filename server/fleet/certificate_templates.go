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
	SubjectName string `json:"subject_name" db:"subject_name"`
	TeamID      uint   `json:"-" db:"team_id"`
}

// HostCertificateTemplate represents a certificate template associated with a particular host
type HostCertificateTemplate struct {
	HostUUID string            `db:"host_uuid" json:"-"`
	Name     string            `db:"name" json:"-"`
	Status   MDMDeliveryStatus `db:"status" json:"-"`
	Detail   *string           `db:"detail" json:"-"`
}

// ToHostMDMProfile maps a HostCertificateTemplate to a HostMDMProfile, suitable for use in the MDM API
func (p *HostCertificateTemplate) ToHostMDMProfile() HostMDMProfile {
	if p == nil {
		return HostMDMProfile{}
	}

	profile := HostMDMProfile{
		HostUUID: p.HostUUID,
		Name:     p.Name,
		Platform: "android",
		Status:   &p.Status,
	}
	if p.Detail != nil {
		profile.Detail = *p.Detail
	}
	return profile
}
