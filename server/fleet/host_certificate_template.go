package fleet

type HostCertificateTemplate struct {
	ID                    uint              `db:"id"`
	HostUUID              string            `db:"host_uuid"`
	CertificateTemplateID uint              `db:"certificate_template_id"`
	FleetChallenge        string            `db:"fleet_challenge"`
	Status                MDMDeliveryStatus `db:"status"`
	CreatedAt             string            `db:"created_at"`
	UpdatedAt             string            `db:"updated_at"`
}

type CertificateTemplateForHost struct {
	HostUUID              string             `db:"host_uuid"`
	CertificateTemplateID uint               `db:"certificate_template_id"`
	FleetChallenge        *string            `db:"fleet_challenge"`
	Status                *MDMDeliveryStatus `db:"status"`
}
