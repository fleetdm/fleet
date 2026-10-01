package fleet

// MDMAndroidDeviceVitalMaxLength is the width of the varchar columns in
// host_mdm_android_device_vitals. AMAPI values are device-reported, so they're
// truncated to this before being written.
const MDMAndroidDeviceVitalMaxLength = 255

// MDMAndroidTelephonyInfo is a single SIM card's telephony information as
// reported by the Android Management API in networkInfo.telephonyInfos. A
// device may report more than one (dual-SIM), and AMAPI only populates this
// at all for fully managed (company-owned) devices running Android 6+.
//
// Reference: https://developers.google.com/android/management/reference/rest/v1/enterprises.devices#TelephonyInfo
type MDMAndroidTelephonyInfo struct {
	PhoneNumber string `json:"phone_number,omitempty"`
	CarrierName string `json:"carrier_name,omitempty"`
	ICCID       string `json:"iccid,omitempty"`
	// ActivationState and ConfigMode are eSIM-only and are reported as
	// ACTIVATION_STATE_UNSPECIFIED / CONFIG_MODE_UNSPECIFIED for physical SIMs
	// and for anything below Android 15.
	ActivationState string `json:"activation_state,omitempty"`
	ConfigMode      string `json:"config_mode,omitempty"`
}

// MDMAndroidPostureDetail is one security risk contributing to a device's
// security posture, as reported by AMAPI in securityPosture.postureDetails.
//
// Reference: https://developers.google.com/android/management/reference/rest/v1/enterprises.devices#PostureDetail
type MDMAndroidPostureDetail struct {
	// SecurityRisk is AMAPI's raw enum value (UNKNOWN_OS, COMPROMISED_OS,
	// HARDWARE_BACKED_EVALUATION_FAILED, ...), stored as sent.
	SecurityRisk string `json:"security_risk,omitempty"`
	// Advice holds the default (non-localized) message of each piece of
	// admin-facing advice AMAPI attaches to the risk. The localized variants
	// are dropped: Fleet has no device locale to select one with.
	Advice []string `json:"advice,omitempty"`
}

// MDMAndroidDeviceVitals holds the Android host vitals Fleet collects from
// AMAPI status reports that don't live on the hosts table itself. Persisted
// to host_mdm_android_device_vitals.
//
// Every field is a pointer (or nil-able slice) because vitals arrive
// asynchronously via Pub/Sub and a device may not report a given section at
// all — depending on its Android version, ownership type, and whether the
// applied policy enables the relevant status reporting setting.
//
// Enum-valued fields hold AMAPI's raw enum string ("ACTIVE", "SECURE",
// "UP_TO_DATE", ...) so that what Fleet stores stays exactly what Google
// sent; mapping them to display strings is the frontend's job.
//
// Reference: https://developers.google.com/android/management/reference/rest/v1/enterprises.devices
type MDMAndroidDeviceVitals struct {
	HostUUID string `db:"host_uuid"`

	AdbEnabled         *bool `db:"adb_enabled"`
	PasscodeProtected  *bool `db:"passcode_protected"`
	PlayProtectEnabled *bool `db:"play_protect_enabled"`

	EncryptionType        *string `db:"encryption_type"`
	Manufacturer          *string `db:"manufacturer"`
	SecurityUpdateVersion *string `db:"security_update_version"`
	DeviceKernelVersion   *string `db:"device_kernel_version"`
	BootloaderVersion     *string `db:"bootloader_version"`
	SystemUpdateStatus    *string `db:"system_update_status"`
	SecurityPosture       *string `db:"security_posture"`

	// IMEI and MEID are the device's hardware radio identifier: IMEI on a GSM
	// device, MEID on a CDMA one, so a device reports at most one of them.
	// AMAPI only reports either for company-owned devices, and Fleet treats
	// them with the same sensitivity as a phone number (see TelephonyInfos).
	IMEI *string `db:"imei"`
	MEID *string `db:"meid"`

	APILevel *int64 `db:"api_level"`

	// SecurityPostureDetails and TelephonyInfos are stored as JSON columns,
	// each a variable-length array written and read as a single unit. They're
	// tagged db:"-": the element types don't implement sql.Scanner /
	// driver.Valuer, so sqlx can't bind or scan them the way it does the
	// scalar fields above — the mysql datastore package converts to and from
	// its own row type instead (see android_device_vitals.go).
	SecurityPostureDetails []MDMAndroidPostureDetail `db:"-"`
	TelephonyInfos         []MDMAndroidTelephonyInfo `db:"-"`
}

// HostMDMAndroidDeviceVitals is MDMAndroidDeviceVitals reshaped for the GET
// host API response: json-tagged instead of db-tagged, and without HostUUID
// (redundant with Host.UUID). fleet.Host embeds this anonymously so its
// fields flatten into the top-level host JSON response.
//
// Every field is tagged db:"-" because these are loaded via a separate query
// for Android hosts only (see LoadHostMDMAndroidDeviceVitals), not the main
// hosts SELECT, and csv:"-" because gocsv flattens embedded struct fields
// into the List Hosts CSV export regardless of any tag on the embedding
// field itself — only a tag on each individual field here stops that (that
// endpoint doesn't load this data anyway, only GET a single host does).
type HostMDMAndroidDeviceVitals struct {
	AdbEnabled         *bool `json:"adb_enabled,omitempty" db:"-" csv:"-"`
	PasscodeProtected  *bool `json:"passcode_protected,omitempty" db:"-" csv:"-"`
	PlayProtectEnabled *bool `json:"play_protect_enabled,omitempty" db:"-" csv:"-"`

	EncryptionType        *string `json:"encryption_type,omitempty" db:"-" csv:"-"`
	Manufacturer          *string `json:"manufacturer,omitempty" db:"-" csv:"-"`
	SecurityUpdateVersion *string `json:"security_update_version,omitempty" db:"-" csv:"-"`
	DeviceKernelVersion   *string `json:"device_kernel_version,omitempty" db:"-" csv:"-"`
	BootloaderVersion     *string `json:"bootloader_version,omitempty" db:"-" csv:"-"`
	SystemUpdateStatus    *string `json:"system_update_status,omitempty" db:"-" csv:"-"`
	SecurityPosture       *string `json:"security_posture,omitempty" db:"-" csv:"-"`
	IMEI                  *string `json:"imei,omitempty" db:"-" csv:"-"`
	MEID                  *string `json:"meid,omitempty" db:"-" csv:"-"`

	APILevel *int64 `json:"api_level,omitempty" db:"-" csv:"-"`

	SecurityPostureDetails []MDMAndroidPostureDetail `json:"security_posture_details,omitempty" db:"-" csv:"-"`
	TelephonyInfos         []MDMAndroidTelephonyInfo `json:"telephony_infos,omitempty" db:"-" csv:"-"`
}
