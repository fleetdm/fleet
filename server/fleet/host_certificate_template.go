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
	retryCount := p.RetryCount
	maxRetries := MaxCertificateInstallRetries
	retrying := p.IsRetrying()
	profile := HostMDMProfile{
		HostUUID:              p.HostUUID,
		Name:                  p.Name,
		Platform:              "android",
		Status:                &status,
		OperationType:         p.OperationType,
		ProfileUUID:           AndroidCertificateTemplateProfileID,
		CertificateTemplateID: &certTemplateID,
		Retrying:              &retrying,
		RetryCount:            &retryCount,
		MaxRetries:            &maxRetries,
	}
	if p.Detail != nil {
		profile.Detail = *p.Detail
	}
	return profile
}

// IsRetrying reports whether Fleet is in the middle of automatically retrying this certificate
// after a failed install. A retry is put back into an in-progress status and delivered again, so
// nothing in the status itself distinguishes it from a first delivery.
//
// A manual resend also leaves a retry count behind, but it clears the detail to NULL, whereas a
// retry always writes one — even an empty string, since the host is not required to report a
// message with a failure. That NULL-versus-set distinction is only available here: the detail is
// flattened to a plain string by the time it reaches an API consumer.
func (p *HostCertificateTemplate) IsRetrying() bool {
	if p == nil || p.OperationType != MDMOperationTypeInstall || p.RetryCount == 0 || p.Detail == nil {
		// Removals are never retried, and a certificate that has not failed yet has no retries.
		return false
	}

	switch p.Status {
	case CertificateTemplatePending, CertificateTemplateDelivering, CertificateTemplateDelivered:
		return true
	default:
		// Terminally failed or verified: whatever happened, Fleet is no longer retrying.
		return false
	}
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
