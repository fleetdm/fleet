package fleet

type HostCertificateTemplate struct {
	ID                    uint              `db:"id"`
	Name                  string            `db:"name"`
	HostUUID              string            `db:"host_uuid"`
	CertificateTemplateID uint              `db:"certificate_template_id"`
	FleetChallenge        string            `db:"fleet_challenge"`
	Status                MDMDeliveryStatus `db:"status"`
	Detail                *string           `db:"detail" json:"-"`
	CreatedAt             string            `db:"created_at"`
	UpdatedAt             string            `db:"updated_at"`
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

type CertificateTemplateForHost struct {
	HostUUID              string             `db:"host_uuid"`
	CertificateTemplateID uint               `db:"certificate_template_id"`
	FleetChallenge        *string            `db:"fleet_challenge"`
	Status                *MDMDeliveryStatus `db:"status"`
}
