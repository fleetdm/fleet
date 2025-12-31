package fleet

// AndroidCertificateTemplateProfileID Used by the front-end for determining the displaying logic.
const AndroidCertificateTemplateProfileID = "fleet-host-certificate-template"

type HostCertificateTemplate struct {
	ID                    uint                      `db:"id"`
	Name                  string                    `db:"name"`
	HostUUID              string                    `db:"host_uuid"`
	CertificateTemplateID uint                      `db:"certificate_template_id"`
	FleetChallenge        *string                   `db:"fleet_challenge"`
	Status                CertificateTemplateStatus `db:"status"`
	OperationType         MDMOperationType          `db:"operation_type"`
	Detail                *string                   `db:"detail" json:"-"`
	CreatedAt             string                    `db:"created_at"`
	UpdatedAt             string                    `db:"updated_at"`
}

// ToHostMDMProfile maps a HostCertificateTemplate to a HostMDMProfile, suitable for use in the MDM API
func (p *HostCertificateTemplate) ToHostMDMProfile() HostMDMProfile {
	if p == nil {
		return HostMDMProfile{}
	}

	status := string(p.Status)
	profile := HostMDMProfile{
		HostUUID:      p.HostUUID,
		Name:          p.Name,
		Platform:      "android",
		Status:        &status,
		OperationType: p.OperationType,
		ProfileUUID:   AndroidCertificateTemplateProfileID,
	}
	if p.Detail != nil {
		profile.Detail = *p.Detail
	}
	return profile
}

type CertificateTemplateForHost struct {
	HostUUID              string                     `db:"host_uuid"`
	CertificateTemplateID uint                       `db:"certificate_template_id"`
	FleetChallenge        *string                    `db:"fleet_challenge"`
	Status                *CertificateTemplateStatus `db:"status"`
	OperationType         *MDMOperationType          `db:"operation_type"`
	CAType                CAConfigAssetType          `db:"ca_type"`
	CAName                string                     `db:"ca_name"`
}
