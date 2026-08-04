package fleet

import "time"

// MDMAppleAccessibilitySettings is the accessibility settings currently set
// on an iOS/iPadOS device, as reported by the AccessibilitySettings key of a
// DeviceInformation command ack.
type MDMAppleAccessibilitySettings struct {
	BoldTextEnabled            *bool  `json:"bold_text_enabled,omitempty"`
	GrayscaleEnabled           *bool  `json:"grayscale_enabled,omitempty"`
	IncreaseContrastEnabled    *bool  `json:"increase_contrast_enabled,omitempty"`
	ReduceMotionEnabled        *bool  `json:"reduce_motion_enabled,omitempty"`
	ReduceTransparencyEnabled  *bool  `json:"reduce_transparency_enabled,omitempty"`
	TextSize                   *int64 `json:"text_size,omitempty"`
	TouchAccommodationsEnabled *bool  `json:"touch_accommodations_enabled,omitempty"`
	VoiceOverEnabled           *bool  `json:"voice_over_enabled,omitempty"`
	ZoomEnabled                *bool  `json:"zoom_enabled,omitempty"`
}

// MDMAppleOrganizationInfo is the MDM server's organization info as reported
// back by the device via the OrganizationInfo key of a DeviceInformation
// command ack.
type MDMAppleOrganizationInfo struct {
	OrganizationName    *string `json:"organization_name,omitempty"`
	OrganizationAddress *string `json:"organization_address,omitempty"`
	OrganizationPhone   *string `json:"organization_phone,omitempty"`
	OrganizationEmail   *string `json:"organization_email,omitempty"`
	OrganizationMagic   *string `json:"organization_magic,omitempty"`
}

// MDMAppleDeviceVitalsMDMOptions is the device's view of MDM options
// currently in effect, as reported by the MDMOptions key of a
// DeviceInformation command ack.
type MDMAppleDeviceVitalsMDMOptions struct {
	ActivationLockAllowedWhileSupervised             *bool `json:"activation_lock_allowed_while_supervised,omitempty"`
	BootstrapTokenAllowed                            *bool `json:"bootstrap_token_allowed,omitempty"`
	PromptUserToAllowBootstrapTokenForAuthentication *bool `json:"prompt_user_to_allow_bootstrap_token_for_authentication,omitempty"`
}

// MDMAppleServiceSubscription is a single cellular service subscription
// reported by a device via the ServiceSubscriptions key of a
// DeviceInformation command ack. Devices with dual-SIM support may report
// more than one; per manual testing against a physical dual-SIM (physical +
// eSIM) iPhone, an inactive/unprovisioned eSIM slot reports a Slot with most
// other fields absent (only EID/IMEI present), so most fields here are
// commonly nil even on a fully-populated response.
type MDMAppleServiceSubscription struct {
	HostUUID                 string  `json:"-" db:"host_uuid"`
	Slot                     string  `json:"slot" db:"slot"`
	CarrierSettingsVersion   *string `json:"carrier_settings_version,omitempty" db:"carrier_settings_version"`
	CurrentCarrierNetwork    *string `json:"current_carrier_network,omitempty" db:"current_carrier_network"`
	CurrentMCC               *string `json:"current_mcc,omitempty" db:"current_mcc"`
	CurrentMNC               *string `json:"current_mnc,omitempty" db:"current_mnc"`
	EID                      *string `json:"eid,omitempty" db:"eid"`
	ICCID                    *string `json:"iccid,omitempty" db:"iccid"`
	IMEI                     *string `json:"imei,omitempty" db:"imei"`
	IsDataPreferred          *bool   `json:"is_data_preferred,omitempty" db:"is_data_preferred"`
	IsRoaming                *bool   `json:"is_roaming,omitempty" db:"is_roaming"`
	IsVoicePreferred         *bool   `json:"is_voice_preferred,omitempty" db:"is_voice_preferred"`
	Label                    *string `json:"label,omitempty" db:"label"`
	LabelID                  *string `json:"label_id,omitempty" db:"label_id"`
	MEID                     *string `json:"meid,omitempty" db:"meid"`
	PhoneNumber              *string `json:"phone_number,omitempty" db:"phone_number"`
	SubscriberCarrierNetwork *string `json:"subscriber_carrier_network,omitempty" db:"subscriber_carrier_network"`
}

// MDMAppleDeviceVitals holds the iOS/iPadOS host vitals Fleet collects via
// the DeviceInformation MDM command that don't live on the hosts table
// itself. Persisted to host_mdm_apple_device_vitals and
// host_mdm_apple_service_subscriptions.
//
// Reference: https://developer.apple.com/documentation/devicemanagement/deviceinformationcommand/command-data.dictionary/queries-data.dictionary
type MDMAppleDeviceVitals struct {
	HostUUID string `db:"host_uuid"`

	UDID                       *string `db:"udid"`
	ModelNumber                *string `db:"model_number"`
	ModemFirmwareVersion       *string `db:"modem_firmware_version"`
	SupplementalBuildVersion   *string `db:"supplemental_build_version"`
	SupplementalOSVersionExtra *string `db:"supplemental_os_version_extra"`
	BluetoothMAC               *string `db:"bluetooth_mac"`
	WiFiMAC                    *string `db:"wifi_mac"`
	EASDeviceIdentifier        *string `db:"eas_device_identifier"`
	ITunesStoreAccountHash     *string `db:"itunes_store_account_hash"`
	PushToken                  []byte  `db:"push_token"`

	BatteryLevel       *float64 `db:"battery_level"`
	CellularTechnology *int64   `db:"cellular_technology"`

	AppAnalyticsEnabled           *bool `db:"app_analytics_enabled"`
	AwaitingConfiguration         *bool `db:"awaiting_configuration"`
	DataRoamingEnabled            *bool `db:"data_roaming_enabled"`
	DiagnosticSubmissionEnabled   *bool `db:"diagnostic_submission_enabled"`
	IsCloudBackupEnabled          *bool `db:"is_cloud_backup_enabled"`
	IsDeviceLocatorServiceEnabled *bool `db:"is_device_locator_service_enabled"`
	IsDoNotDisturbInEffect        *bool `db:"is_do_not_disturb_in_effect"`
	IsMDMLostModeEnabled          *bool `db:"is_mdm_lost_mode_enabled"`
	IsNetworkTethered             *bool `db:"is_network_tethered"`
	ITunesStoreAccountIsActive    *bool `db:"itunes_store_account_is_active"`
	PersonalHotspotEnabled        *bool `db:"personal_hotspot_enabled"`

	LastCloudBackupDate *time.Time `db:"last_cloud_backup_date"`

	// AccessibilitySettings, OrganizationInfo, MDMOptions, and
	// DevicePropertiesAttestation are stored as JSON columns, each a
	// fixed-shape nested object written/read as a single unit. They're tagged
	// db:"-": neither this type nor the nested ones implement sql.Scanner /
	// driver.Valuer, so sqlx can't bind/scan them directly the way it does the
	// scalar fields above — the mysql datastore package converts to/from its
	// own row type instead (see apple_mdm_device_vitals.go).
	AccessibilitySettings *MDMAppleAccessibilitySettings  `db:"-"`
	OrganizationInfo      *MDMAppleOrganizationInfo       `db:"-"`
	MDMOptions            *MDMAppleDeviceVitalsMDMOptions `db:"-"`
	// DevicePropertiesAttestation is the raw DER certificate chain Apple
	// returns (rooted at the Apple Enterprise Attestation Root CA; per manual
	// testing against a physical iPhone, typically 2 certificates — the leaf
	// and the "Apple Enterprise Attestation Sub CA" intermediate). Fleet does
	// not currently parse the chain's custom OIDs into a derived signal. See
	// https://github.com/fleetdm/fleet/issues/49984.
	DevicePropertiesAttestation [][]byte `db:"-"`

	ServiceSubscriptions []MDMAppleServiceSubscription `db:"-"`
}
