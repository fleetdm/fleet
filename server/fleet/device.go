package fleet

import "time"

// DesktopSummary is a summary of the status of a host that's used by Fleet
// Desktop to operate (show/hide menu items, etc)
type DesktopSummary struct {
	FailingPolicies *uint                `json:"failing_policies_count,omitempty"`
	SelfService     *bool                `json:"self_service"`
	Notifications   DesktopNotifications `json:"notifications,omitempty"`
	Config          DesktopConfig        `json:"config"`
}

// DesktopNotifications are notifications that the fleet server sends to
// Fleet Desktop so that it can run commands or more generally react to this
// information.
type DesktopNotifications struct {
	NeedsMDMMigration      bool `json:"needs_mdm_migration,omitempty"`
	RenewEnrollmentProfile bool `json:"renew_enrollment_profile,omitempty"`
}

// DesktopConfig is a subset of AppConfig with information relevant to Fleet
// Desktop to operate.
type DesktopConfig struct {
	OrgInfo DesktopOrgInfo   `json:"org_info,omitempty"`
	MDM     DesktopMDMConfig `json:"mdm"`
}

// DesktopMDMConfig is a subset of fleet.MDM with configuration that's relevant
// to Fleet Desktop to operate.
type DesktopMDMConfig struct {
	MacOSMigration struct {
		Mode MacOSMigrationMode `json:"mode"`
	} `json:"macos_migration"`
}

// DesktopMDMConfig is a subset of fleet.OrgInfo with configuration that's relevant
// to Fleet Desktop to operate.
type DesktopOrgInfo struct {
	OrgName                   string `json:"org_name"`
	OrgLogoURL                string `json:"org_logo_url"`
	OrgLogoURLLightBackground string `json:"org_logo_url_light_background"`
	ContactURL                string `json:"contact_url"`
}

type MigrateMDMDeviceWebhookPayload struct {
	Timestamp time.Time `json:"timestamp"`
	Host      struct {
		ID             uint   `json:"id"`
		UUID           string `json:"uuid"`
		HardwareSerial string `json:"hardware_serial"`
	} `json:"host"`
}
