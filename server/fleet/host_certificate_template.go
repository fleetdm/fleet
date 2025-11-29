package fleet

type HostCertificateTemplate struct {
	ID                    uint              `db:"id"`
	Name                  string            `db:"name"`
	HostUUID              string            `db:"host_uuid"`
	CertificateTemplateID uint              `db:"certificate_template_id"`
	FleetChallenge        string            `db:"fleet_challenge"`
	Status                MDMDeliveryStatus `db:"status"`
	CreatedAt             string            `db:"created_at"`
	UpdatedAt             string            `db:"updated_at"`
}

// ToHostMDMProfile maps a HostCertificateTemplate to a HostMDMProfile, suitable for use in the MDM API
func (p *HostCertificateTemplate) ToHostMDMProfile() HostMDMProfile {
	if p == nil {
		return HostMDMProfile{}
	}

	return HostMDMProfile{
		HostUUID: p.HostUUID,
		Name:     p.Name,
		Platform: "android",
		Status:   &p.Status,
	}
}

type CertificateTemplateForHost struct {
	HostUUID              string             `db:"host_uuid"`
	CertificateTemplateID uint               `db:"certificate_template_id"`
	FleetChallenge        *string            `db:"fleet_challenge"`
	Status                *MDMDeliveryStatus `db:"status"`
}
