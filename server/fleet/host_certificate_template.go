package fleet

import "time"

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
	UUID                  string                    `db:"uuid"`
	CreatedAt             string                    `db:"created_at"`
	UpdatedAt             string                    `db:"updated_at"`
	NotValidBefore        *time.Time                `db:"not_valid_before"` // for future use
	NotValidAfter         *time.Time                `db:"not_valid_after"`
	Serial                *string                   `db:"serial"` // for future use
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
	UUID                  *string                    `db:"uuid"`
	CAType                CAConfigAssetType          `db:"ca_type"`
	CAName                string                     `db:"ca_name"`
}

// CertificateStatusUpdate holds all fields for updating a host's certificate status.
type CertificateStatusUpdate struct {
	HostUUID              string
	CertificateTemplateID uint
	Status                MDMDeliveryStatus
	Detail                *string
	OperationType         MDMOperationType
	NotValidBefore        *time.Time
	NotValidAfter         *time.Time
	Serial                *string
}

// HostCertificateTemplateForRenewal represents a certificate template that needs renewal.
type HostCertificateTemplateForRenewal struct {
	HostUUID              string    `db:"host_uuid"`
	CertificateTemplateID uint      `db:"certificate_template_id"`
	NotValidAfter         time.Time `db:"not_valid_after"`
}
