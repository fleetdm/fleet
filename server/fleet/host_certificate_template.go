package fleet

import "time"

// AndroidCertificateTemplateProfileID Used by the front-end for determining the displaying logic.
const AndroidCertificateTemplateProfileID = "fleet-host-certificate-template"

// ONCProfileWithheldDetailPrefix is the prefix used in the detail field of withheld Android
// profiles that are waiting for a certificate to be installed before they can be applied.
const ONCProfileWithheldDetailPrefix = "Waiting for certificate"

// MaxCertificateInstallRetries is the maximum number of automatic retries after the initial attempt
// when the Android agent reports a certificate install failure. Manual resend via the UI sets
// retry_count to this value so the resend gets exactly one attempt with no automatic retry.
const MaxCertificateInstallRetries uint = 3

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
	NotValidBefore        *time.Time                `db:"not_valid_before"`
	NotValidAfter         *time.Time                `db:"not_valid_after"`
	Serial                *string                   `db:"serial"` // for future use
	RetryCount            uint                      `db:"retry_count"`
}

// ToHostMDMProfile maps a HostCertificateTemplate to a HostMDMProfile, suitable for use in the MDM API
func (p *HostCertificateTemplate) ToHostMDMProfile() HostMDMProfile {
	if p == nil {
		return HostMDMProfile{}
	}

	status := string(p.Status)
	certTemplateID := p.CertificateTemplateID
	profile := HostMDMProfile{
		HostUUID:              p.HostUUID,
		Name:                  p.Name,
		Platform:              "android",
		Status:                &status,
		OperationType:         p.OperationType,
		ProfileUUID:           AndroidCertificateTemplateProfileID,
		CertificateTemplateID: &certTemplateID,
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
	HostUUID              string            `db:"host_uuid"`
	CertificateTemplateID uint              `db:"certificate_template_id"`
	Status                MDMDeliveryStatus `db:"status"`
	Detail                *string           `db:"detail"`
	OperationType         MDMOperationType  `db:"operation_type"`
	NotValidBefore        *time.Time        `db:"not_valid_before"`
	NotValidAfter         *time.Time        `db:"not_valid_after"`
	Serial                *string           `db:"serial"`
}

// HostCertificateTemplateForRenewal represents a certificate template that needs renewal.
type HostCertificateTemplateForRenewal struct {
	HostUUID              string    `db:"host_uuid"`
	CertificateTemplateID uint      `db:"certificate_template_id"`
	NotValidAfter         time.Time `db:"not_valid_after"`
}
