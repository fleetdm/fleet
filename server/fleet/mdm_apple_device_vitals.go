package fleet

import (
	"encoding/json"
	"fmt"
	"time"
)

// MDMAppleCellularTechnology is the cellular radio technology a device
// supports, as reported by the CellularTechnology key of a DeviceInformation
// command ack. Apple reports it as an integer, which is what Fleet persists;
// it's mapped to a display string only when serializing an API response, so
// the stored value stays exactly what Apple sent.
//
// Reference: https://developer.apple.com/documentation/devicemanagement/deviceinformationresponse/queryresponses-data.dictionary
type MDMAppleCellularTechnology int64

const (
	MDMAppleCellularTechnologyNone       MDMAppleCellularTechnology = 0
	MDMAppleCellularTechnologyGSM        MDMAppleCellularTechnology = 1
	MDMAppleCellularTechnologyCDMA       MDMAppleCellularTechnology = 2
	MDMAppleCellularTechnologyGSMAndCDMA MDMAppleCellularTechnology = 3
	// MDMAppleCellularTechnologyUnknown is not a value Apple reports. It's
	// what a value outside Apple's documented set maps to, so that Apple
	// extending the enum can't fail serialization of a whole host response
	// (nor deserialization of one, e.g. by fleetctl). The integer Apple
	// actually sent is preserved in host_mdm_apple_device_vitals either way.
	MDMAppleCellularTechnologyUnknown MDMAppleCellularTechnology = -1
)

func (c MDMAppleCellularTechnology) String() string {
	switch c {
	case MDMAppleCellularTechnologyNone:
		return "None"
	case MDMAppleCellularTechnologyGSM:
		return "GSM"
	case MDMAppleCellularTechnologyCDMA:
		return "CDMA"
	case MDMAppleCellularTechnologyGSMAndCDMA:
		return "GSM and CDMA"
	default:
		return "unknown"
	}
}

func (c MDMAppleCellularTechnology) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

func (c *MDMAppleCellularTechnology) UnmarshalJSON(b []byte) error {
	// Accept the raw integer too, so a value straight out of Apple's ack (or
	// the database) round-trips through this type.
	var i int64
	if err := json.Unmarshal(b, &i); err == nil {
		*c = MDMAppleCellularTechnology(i)
		return nil
	}

	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("invalid MDMAppleCellularTechnology: %s", string(b))
	}
	switch s {
	case "None":
		*c = MDMAppleCellularTechnologyNone
	case "GSM":
		*c = MDMAppleCellularTechnologyGSM
	case "CDMA":
		*c = MDMAppleCellularTechnologyCDMA
	case "GSM and CDMA":
		*c = MDMAppleCellularTechnologyGSMAndCDMA
	default:
		*c = MDMAppleCellularTechnologyUnknown
	}
	return nil
}

// MDMAppleAccessibilitySettings is the accessibility settings currently set
// on an iOS/iPadOS device, as reported by the AccessibilitySettings key of a
// DeviceInformation command ack.
// csv:"-" on every field below because gocsv flattens nested struct fields
// into the List Hosts CSV report regardless of the csv tag on the outer
// fleet.Host field that holds the pointer to this struct; the outer tag
// alone doesn't stop it from being flattened in CSV (only List Hosts, not
// GET a single host, exports CSV, and this data isn't loaded for that
// endpoint anyway).
type MDMAppleAccessibilitySettings struct {
	BoldTextEnabled            *bool  `json:"bold_text_enabled,omitempty" csv:"-"`
	GrayscaleEnabled           *bool  `json:"grayscale_enabled,omitempty" csv:"-"`
	IncreaseContrastEnabled    *bool  `json:"increase_contrast_enabled,omitempty" csv:"-"`
	ReduceMotionEnabled        *bool  `json:"reduce_motion_enabled,omitempty" csv:"-"`
	ReduceTransparencyEnabled  *bool  `json:"reduce_transparency_enabled,omitempty" csv:"-"`
	TextSize                   *int64 `json:"text_size,omitempty" csv:"-"`
	TouchAccommodationsEnabled *bool  `json:"touch_accommodations_enabled,omitempty" csv:"-"`
	VoiceOverEnabled           *bool  `json:"voice_over_enabled,omitempty" csv:"-"`
	ZoomEnabled                *bool  `json:"zoom_enabled,omitempty" csv:"-"`
}

// MDMAppleOrganizationInfo is the MDM server's organization info as reported
// back by the device via the OrganizationInfo key of a DeviceInformation
// command ack.
type MDMAppleOrganizationInfo struct {
	OrganizationName    *string `json:"organization_name,omitempty" csv:"-"`
	OrganizationAddress *string `json:"organization_address,omitempty" csv:"-"`
	OrganizationPhone   *string `json:"organization_phone,omitempty" csv:"-"`
	OrganizationEmail   *string `json:"organization_email,omitempty" csv:"-"`
	OrganizationMagic   *string `json:"organization_magic,omitempty" csv:"-"`
}

// MDMAppleDeviceVitalsMDMOptions is the device's view of MDM options
// currently in effect, as reported by the MDMOptions key of a
// DeviceInformation command ack.
type MDMAppleDeviceVitalsMDMOptions struct {
	ActivationLockAllowedWhileSupervised             *bool `json:"activation_lock_allowed_while_supervised,omitempty" csv:"-"`
	BootstrapTokenAllowed                            *bool `json:"bootstrap_token_allowed,omitempty" csv:"-"`
	PromptUserToAllowBootstrapTokenForAuthentication *bool `json:"prompt_user_to_allow_bootstrap_token_for_authentication,omitempty" csv:"-"`
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

// HostMDMAppleDeviceVitals is MDMAppleDeviceVitals reshaped for the GET host
// API response: json-tagged instead of db-tagged, and without HostUUID
// (redundant with Host.UUID). fleet.Host embeds this anonymously so its
// fields flatten into the top-level host JSON response.
//
// Every field is tagged db:"-" because these are loaded via a separate query
// for iOS/iPadOS hosts only (see loadHostMDMAppleDeviceVitalsDB), not the
// main hosts SELECT, and csv:"-" because gocsv flattens embedded struct
// fields into the List Hosts CSV export regardless of any tag on the
// embedding field itself — only a tag on each individual field here stops
// that (that endpoint doesn't load this data anyway, only GET a single host
// does).
type HostMDMAppleDeviceVitals struct {
	UDID                       *string `json:"udid,omitempty" db:"-" csv:"-"`
	ModelNumber                *string `json:"model_number,omitempty" db:"-" csv:"-"`
	ModemFirmwareVersion       *string `json:"modem_firmware_version,omitempty" db:"-" csv:"-"`
	SupplementalBuildVersion   *string `json:"supplemental_build_version,omitempty" db:"-" csv:"-"`
	SupplementalOSVersionExtra *string `json:"supplemental_os_version_extra,omitempty" db:"-" csv:"-"`
	BluetoothMAC               *string `json:"bluetooth_mac,omitempty" db:"-" csv:"-"`
	WiFiMAC                    *string `json:"wifi_mac,omitempty" db:"-" csv:"-"`
	EASDeviceIdentifier        *string `json:"eas_device_identifier,omitempty" db:"-" csv:"-"`
	ITunesStoreAccountHash     *string `json:"itunes_store_account_hash,omitempty" db:"-" csv:"-"`
	PushToken                  []byte  `json:"push_token,omitempty" db:"-" csv:"-"`

	BatteryLevel *float64 `json:"battery_level,omitempty" db:"-" csv:"-"`
	// CellularTechnology serializes as Apple's display label ("GSM", "CDMA",
	// ...), not the raw integer stored in the DB. See
	// MDMAppleCellularTechnology.
	CellularTechnology *MDMAppleCellularTechnology `json:"cellular_technology,omitempty" db:"-" csv:"-"`

	AppAnalyticsEnabled           *bool `json:"app_analytics_enabled,omitempty" db:"-" csv:"-"`
	AwaitingConfiguration         *bool `json:"awaiting_configuration,omitempty" db:"-" csv:"-"`
	DataRoamingEnabled            *bool `json:"data_roaming_enabled,omitempty" db:"-" csv:"-"`
	DiagnosticSubmissionEnabled   *bool `json:"diagnostic_submission_enabled,omitempty" db:"-" csv:"-"`
	IsCloudBackupEnabled          *bool `json:"is_cloud_backup_enabled,omitempty" db:"-" csv:"-"`
	IsDeviceLocatorServiceEnabled *bool `json:"is_device_locator_service_enabled,omitempty" db:"-" csv:"-"`
	IsDoNotDisturbInEffect        *bool `json:"is_do_not_disturb_in_effect,omitempty" db:"-" csv:"-"`
	IsMDMLostModeEnabled          *bool `json:"is_mdm_lost_mode_enabled,omitempty" db:"-" csv:"-"`
	IsNetworkTethered             *bool `json:"is_network_tethered,omitempty" db:"-" csv:"-"`
	ITunesStoreAccountIsActive    *bool `json:"itunes_store_account_is_active,omitempty" db:"-" csv:"-"`
	PersonalHotspotEnabled        *bool `json:"personal_hotspot_enabled,omitempty" db:"-" csv:"-"`

	LastCloudBackupDate *time.Time `json:"last_cloud_backup_date,omitempty" db:"-" csv:"-"`

	AccessibilitySettings *MDMAppleAccessibilitySettings  `json:"accessibility_settings,omitempty" db:"-" csv:"-"`
	OrganizationInfo      *MDMAppleOrganizationInfo       `json:"organization_info,omitempty" db:"-" csv:"-"`
	MDMOptions            *MDMAppleDeviceVitalsMDMOptions `json:"mdm_options,omitempty" db:"-" csv:"-"`
	// DevicePropertiesAttestation is the raw DER certificate chain Apple
	// returns (base64-encoded once marshaled to JSON), not a boolean/status
	// object. See MDMAppleDeviceVitals.DevicePropertiesAttestation.
	DevicePropertiesAttestation [][]byte `json:"device_properties_attestation,omitempty" db:"-" csv:"-"`

	ServiceSubscriptions []MDMAppleServiceSubscription `json:"service_subscriptions,omitempty" db:"-" csv:"-"`
}
