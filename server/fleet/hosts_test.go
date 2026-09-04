package fleet

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ensure all Linuxes are in one of the three package support groups
func TestHostLinuxPlatformPackageCompatibility(t *testing.T) {
	for _, os := range HostLinuxOSs {
		h := &Host{Platform: os}
		_, isNeitherDebNorRpm := HostNeitherDebNorRpmPackageOSs[h.Platform]
		if isNeitherDebNorRpm {
			continue
		}

		require.True(t, h.PlatformSupportsDebPackages() || h.PlatformSupportsRpmPackages())
	}
}

func TestIsLUKSSupported(t *testing.T) {
	for _, tc := range []struct {
		platform  string
		osVersion string
		expected  bool
	}{
		{platform: "ubuntu", expected: true},
		{platform: "zorin", expected: true},
		// Fedora hosts report their platform as "rhel", so they are identified by OS version.
		{platform: "rhel", osVersion: "Fedora Linux 41", expected: true},
		{platform: "rhel", osVersion: "CentOS Linux 7.9.2009", expected: false},
		// Arch and its derivatives.
		{platform: "arch", expected: true},
		{platform: "archarm", expected: true},
		{platform: "manjaro", expected: true},
		{platform: "manjaro-arm", expected: true},
		{platform: "cachyos", expected: true},
		{platform: "omarchy", expected: true},
		// Linux platforms without LUKS support, and non-Linux platforms.
		{platform: "debian", expected: false},
		{platform: "darwin", expected: false},
		{platform: "windows", expected: false},
	} {
		t.Run(tc.platform+" "+tc.osVersion, func(t *testing.T) {
			h := &Host{Platform: tc.platform, OSVersion: tc.osVersion}
			require.Equal(t, tc.expected, h.IsLUKSSupported())
		})
	}
}

func TestHostStatus(t *testing.T) {
	mockClock := clock.NewMockClock()

	testCases := []struct {
		seenTime            time.Time
		distributedInterval uint
		configTLSRefresh    uint
		status              HostStatus
	}{
		{mockClock.Now().Add(-30 * time.Second), 10, 3600, StatusOnline},
		{mockClock.Now().Add(-75 * time.Second), 10, 3600, StatusOffline},
		{mockClock.Now().Add(-30 * time.Second), 3600, 10, StatusOnline},
		{mockClock.Now().Add(-75 * time.Second), 3600, 10, StatusOffline},

		{mockClock.Now().Add(-60 * time.Second), 60, 60, StatusOnline},
		{mockClock.Now().Add(-121 * time.Second), 60, 60, StatusOffline},

		{mockClock.Now().Add(-1 * time.Second), 10, 10, StatusOnline},
		{mockClock.Now().Add(-2 * time.Minute), 10, 10, StatusOffline},
		{mockClock.Now().Add(-31 * 24 * time.Hour), 10, 10, StatusOffline}, // As of Fleet 4.15, StatusMIA is deprecated in favor of StatusOffline

		// Ensure behavior is reasonable if we don't have the values
		{mockClock.Now().Add(-1 * time.Second), 0, 0, StatusOnline},
		{mockClock.Now().Add(-2 * time.Minute), 0, 0, StatusOffline},
		{mockClock.Now().Add(-31 * 24 * time.Hour), 0, 0, StatusOffline}, // As of Fleet 4.15, StatusMIA is deprecated in favor of StatusOffline
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			// Save interval values
			h := Host{
				DistributedInterval: tt.distributedInterval,
				ConfigTLSRefresh:    tt.configTLSRefresh,
				SeenTime:            tt.seenTime,
			}

			assert.Equal(t, tt.status, h.Status(mockClock.Now()))
		})
	}
}

func TestHostStatusIsValid(t *testing.T) {
	for _, tt := range []struct {
		name     string
		status   HostStatus
		expected bool
	}{
		{"online", StatusOnline, true},
		{"offline", StatusOffline, true},
		{"new", StatusNew, true},
		{"missing", StatusMissing, true},
		{"mia", StatusMIA, true}, // As of Fleet 4.15, StatusMIA is deprecated in favor of StatusOffline
		{"empty", HostStatus(""), false},
		{"invalid", HostStatus("invalid"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsValid())
		})
	}
}

func TestHostIsNew(t *testing.T) {
	mockClock := clock.NewMockClock()

	host := Host{}

	host.CreatedAt = mockClock.Now().AddDate(0, 0, -1)
	assert.True(t, host.IsNew(mockClock.Now()))

	host.CreatedAt = mockClock.Now().AddDate(0, 0, -2)
	assert.False(t, host.IsNew(mockClock.Now()))
}

func TestPlatformFromHost(t *testing.T) {
	for _, tc := range []struct {
		host        string
		expPlatform string
	}{
		{
			host:        "unknown",
			expPlatform: "",
		},
		{
			host:        "",
			expPlatform: "",
		},
		{
			host:        "linux",
			expPlatform: "linux",
		},
		{
			host:        "ubuntu",
			expPlatform: "linux",
		},
		{
			host:        "debian",
			expPlatform: "linux",
		},
		{
			host:        "rhel",
			expPlatform: "linux",
		},
		{
			host:        "centos",
			expPlatform: "linux",
		},
		{
			host:        "sles",
			expPlatform: "linux",
		},
		{
			host:        "kali",
			expPlatform: "linux",
		},
		{
			host:        "gentoo",
			expPlatform: "linux",
		},
		{
			host:        "tuxedo",
			expPlatform: "linux",
		},
		{
			host:        "flatcar",
			expPlatform: "linux",
		},
		{
			host:        "coreos",
			expPlatform: "linux",
		},
		{
			host:        "omarchy",
			expPlatform: "linux",
		},
		{
			host:        "darwin",
			expPlatform: "darwin",
		},
		{
			host:        "windows",
			expPlatform: "windows",
		},
	} {
		fleetPlatform := PlatformFromHost(tc.host)
		require.Equal(t, tc.expPlatform, fleetPlatform)

	}
}

func TestHostDisplayName(t *testing.T) {
	const (
		computerName   = "K0mpu73rN4M3"
		hostname       = "h0s7N4ME"
		hardwareModel  = "M0D3l"
		hardwareSerial = "53r14l"
	)
	for _, tc := range []struct {
		host     Host
		expected string
	}{
		{
			host:     Host{ComputerName: computerName, Hostname: hostname, HardwareModel: hardwareModel, HardwareSerial: hardwareSerial},
			expected: computerName, // If ComputerName is present, DisplayName is ComputerName
		},
		{
			host:     Host{ComputerName: "", Hostname: "h0s7N4ME", HardwareModel: "M0D3l", HardwareSerial: "53r14l"},
			expected: hostname, // If ComputerName is empty, DisplayName is Hostname (if present)
		},
		{
			host:     Host{ComputerName: "", Hostname: "", HardwareModel: "M0D3l", HardwareSerial: "53r14l"},
			expected: fmt.Sprintf("%s (%s)", hardwareModel, hardwareSerial), // If ComputerName and Hostname are empty, DisplayName is composite of HardwareModel and HardwareSerial (if both are present)
		},
		{
			host:     Host{ComputerName: "", Hostname: "", HardwareModel: "", HardwareSerial: hardwareSerial},
			expected: "", // If HarwareModel and/or HardwareSerial are empty, DisplayName is also empty
		},
		{
			host:     Host{ComputerName: "", Hostname: "", HardwareModel: hardwareModel, HardwareSerial: ""},
			expected: "", // If HarwareModel and/or HardwareSerial are empty, DisplayName is also empty
		},
		{
			host:     Host{ComputerName: "", Hostname: "", HardwareModel: "", HardwareSerial: ""},
			expected: "", // If HarwareModel and/or HardwareSerial are empty, DisplayName is also empty
		},
	} {
		require.Equal(t, tc.expected, tc.host.DisplayName())
	}
}

func TestMDMEnrollmentStatus(t *testing.T) {
	for _, tc := range []struct {
		hostMDM  HostMDM
		expected string
	}{
		{
			hostMDM:  HostMDM{Enrolled: true, InstalledFromDep: true, IsPersonalEnrollment: false},
			expected: "On (automatic)",
		},
		{
			hostMDM:  HostMDM{Enrolled: true, InstalledFromDep: false, IsPersonalEnrollment: false},
			expected: "On (manual)",
		},
		{
			hostMDM:  HostMDM{Enrolled: true, InstalledFromDep: false, IsPersonalEnrollment: true},
			expected: "On (manual - personal)",
		},
		{
			hostMDM:  HostMDM{Enrolled: false, InstalledFromDep: true},
			expected: "Pending",
		},
		{
			hostMDM:  HostMDM{Enrolled: false, InstalledFromDep: false},
			expected: "Off",
		},
	} {
		require.Equal(t, tc.expected, tc.hostMDM.EnrollmentStatus())
	}
}

func TestIsEligibleForDEPMigration(t *testing.T) {
	testCases := []struct {
		name                    string
		osqueryHostID           *string
		depAssignedToFleet      *bool
		depProfileResponse      DEPAssignProfileResponseStatus
		enrolledInThirdPartyMDM bool
		expected                bool
		expectedManual          bool
		hostOS                  string
	}{
		{
			name:                    "Eligible for DEP migration",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(true),
			depProfileResponse:      DEPAssignProfileResponseSuccess,
			enrolledInThirdPartyMDM: true,
			expected:                true,
			expectedManual:          false,
		},
		{
			name:                    "Not eligible - osqueryHostID nil",
			osqueryHostID:           nil,
			depAssignedToFleet:      ptr.Bool(true),
			depProfileResponse:      DEPAssignProfileResponseSuccess,
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          false,
		},
		{
			name:                    "Not eligible - not DEP assigned to Fleet",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(false),
			depProfileResponse:      DEPAssignProfileResponseSuccess,
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          false,
		},
		{
			name:                    "Not eligible - not enrolled in third-party MDM",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(true),
			depProfileResponse:      DEPAssignProfileResponseSuccess,
			enrolledInThirdPartyMDM: false,
			expected:                false,
			expectedManual:          false,
		},
		{
			name:                    "Not eligible - not DEP assigned and DEP profile failed",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(false),
			depProfileResponse:      DEPAssignProfileResponseNotAccessible,
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          true,
			hostOS:                  "macOS 14.5",
		},
		{
			name:                    "Not eligible - DEP assigned and DEP profile failed",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(true),
			depProfileResponse:      DEPAssignProfileResponseFailed,
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          false,
		},
		{
			name:                    "Not eligible - DEP assigned and DEP profile throttled",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(true),
			depProfileResponse:      DEPAssignProfileResponseThrottled,
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          false,
		},
		{
			name:                    "Not eligible - DEP assigned but not response yet",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(true),
			depProfileResponse:      "",
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          false,
		},
		{
			name:                    "Not eligible - DEP assigned but not accessible",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(true),
			depProfileResponse:      DEPAssignProfileResponseNotAccessible,
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          false,
		},
		{
			name:                    "Manual migration eligible - enrolled in 3rd party, but not DEP",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(false),
			depProfileResponse:      "",
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          true,
			hostOS:                  "macOS 14.5",
		},
		{
			name:                    "Manual migration ineligible - enrolled in 3rd party, not DEP, but OS version too low",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(false),
			depProfileResponse:      "",
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          false,
			hostOS:                  "macOS 13.9",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			host := &Host{
				OsqueryHostID:      tc.osqueryHostID,
				DEPAssignedToFleet: tc.depAssignedToFleet,
				OSVersion:          tc.hostOS,
			}

			mdmInfo := &HostMDM{
				Enrolled:               tc.enrolledInThirdPartyMDM,
				Name:                   "Some MDM",
				DEPProfileAssignStatus: ptr.String(string(tc.depProfileResponse)),
			}

			require.Equal(t, tc.expected, IsEligibleForDEPMigration(host, mdmInfo, false))
			manual, err := IsEligibleForManualMigration(host, mdmInfo, false)
			require.NoError(t, err)
			require.Equal(t, tc.expectedManual, manual)
		})
	}
}

func TestHasJSONProfileAssigned(t *testing.T) {
	testCases := []struct {
		name     string
		hostMDM  *HostMDM
		expected bool
	}{
		{
			name:     "nil HostMDM",
			hostMDM:  nil,
			expected: false,
		},
		{
			name: "nil DEPProfileAssignStatus",
			hostMDM: &HostMDM{
				DEPProfileAssignStatus: nil,
			},
			expected: false,
		},
		{
			name: "DEPProfileAssignStatus not successful",
			hostMDM: &HostMDM{
				DEPProfileAssignStatus: new(string),
			},
			expected: false,
		},
		{
			name: "DEPProfileAssignStatus successful",
			hostMDM: &HostMDM{
				DEPProfileAssignStatus: ptr.String(string(DEPAssignProfileResponseSuccess)),
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.hostMDM.HasJSONProfileAssigned()
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestMDMNameFromServerURL(t *testing.T) {
	testCases := []struct {
		name      string
		serverURL string
		expected  string
	}{
		{"empty", "", UnknownMDMName},
		{"unknown", "https://example.com/mdm", UnknownMDMName},
		{"kandji", "https://example.kandji.io", WellKnownMDMIru},
		{"iru", "https://mdm.iru.com", WellKnownMDMIru},
		{"jamf", "https://example.jamfcloud.com", WellKnownMDMJamf},
		{"jumpcloud", "https://example.jumpcloud.com", WellKnownMDMJumpCloud},
		{"airwatch", "https://example.airwatch.com", WellKnownMDMVMWare},
		{"awmdm", "https://example.awmdm.com", WellKnownMDMVMWare},
		{"microsoft intune", "https://manage.microsoft.com", WellKnownMDMIntune},
		{"simplemdm", "https://example.simplemdm.com", WellKnownMDMSimpleMDM},
		{"fleetdm", "https://example.fleetdm.com", WellKnownMDMFleet},
		{"mosyle", "https://example.mosyle.com", WellKnownMDMMosyle},
		{"mixed case is normalized", "https://Example.JumpCloud.com", WellKnownMDMJumpCloud},
		// Ambiguous URLs must resolve deterministically. JumpCloud's MDM is hosted on
		// AirWatch/awmdm.com infrastructure, so jumpcloud.awmdm.com must resolve to
		// JumpCloud rather than VMware Workspace ONE.
		{"jumpcloud on awmdm infrastructure", "https://jumpcloud.awmdm.com", WellKnownMDMJumpCloud},
		{"zentral cloud", "https://mdm.example.zentral.io/public/mdm/connect/", WellKnownMDMZentral},
		{"zentral self-hosted", "https://zentral.company.com/public/mdm/connect/", WellKnownMDMZentral},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, MDMNameFromServerURL(tc.serverURL))
		})
	}
}

func TestIsPlaceholderHardwareSerial(t *testing.T) {
	for _, tc := range []struct {
		name   string
		serial string
		want   bool
	}{
		// Empty / whitespace.
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"tabs and newline", "\t \n", true},

		// Known OEM/BIOS placeholders, matched case-insensitively and trimmed.
		{"to be filled exact", "To Be Filled By O.E.M.", true},
		{"to be filled lower", "to be filled by o.e.m.", true},
		{"to be filled upper", "TO BE FILLED BY O.E.M.", true},
		{"to be filled padded", "  To Be Filled By O.E.M.  ", true},
		{"default string", "Default string", true},
		{"system serial number", "System Serial Number", true},
		{"not specified", "Not Specified", true},
		{"not applicable", "Not Applicable", true},
		{"none", "None", true},
		{"oem", "OEM", true},
		{"o.e.m.", "O.E.M.", true},
		{"default", "Default", true},
		{"unknown", "Unknown", true},
		{"chassis serial number", "Chassis Serial Number", true},
		{"base board serial number", "Base Board Serial Number", true},
		{"baseboard serial number", "Baseboard Serial Number", true},
		{"sequential 123456789", "123456789", true},
		{"sequential 0123456789", "0123456789", true},
		{"sequential 1234567890", "1234567890", true},
		{"sequential 1234567", "1234567", true},
		{"n/a", "N/A", true},
		{"na", "na", true},
		{"invalid", "INVALID", true},

		// Repeated-character heuristic (cannot be enumerated).
		{"single zero", "0", true},
		{"all zeros", "00000000", true},
		{"all x", "xxxxxxx", true},
		{"all dashes", "-------", true},
		{"all dots", "........", true},

		// Real, unique serials must NOT be flagged.
		{"dell service tag", "7XQ2W13", false},
		{"lenovo serial", "PF0ABCDE", false},
		{"apple-style serial", "C02ABCDEFGHJ", false},
		{"vmware unique", "VMware-56 4d 1a 2b 3c", false},
		{"none as substring", "NONE123", false},
		{"default as substring", "DEFAULT-7H2K", false},
		{"leading zeros but real", "00000001", false},
		{"long alphanumeric", "ABC123XYZ789", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsPlaceholderHardwareSerial(tc.serial))
		})
	}
}

// The frontend keys the host page's Enrollment ID row off this exact field name, so pin
// the wire format: MDMHostData is scanned from a JSON_OBJECT built in SQL, which makes a
// silent rename easy to miss.
func TestMDMHostDataIsPersonalEnrollmentJSON(t *testing.T) {
	for _, want := range []bool{true, false} {
		b, err := json.Marshal(MDMHostData{IsPersonalEnrollment: want})
		require.NoError(t, err)
		require.Contains(t, string(b), fmt.Sprintf(`"is_personal_enrollment":%t`, want))
	}

	// It is also the key the datastore's JSON_OBJECT emits, so round-tripping through
	// Scan has to land on the same field.
	var data MDMHostData
	require.NoError(t, data.Scan([]byte(`{"is_personal_enrollment": true}`)))
	require.True(t, data.IsPersonalEnrollment)
}

func TestHostMDMHostNameSettingJSON(t *testing.T) {
	// Omitted entirely when there is no enforcement (host_name is a nil pointer
	// with omitempty), matching the recovery-lock treatment for ineligible hosts.
	b, err := json.Marshal(HostMDMOSSettings{})
	require.NoError(t, err)
	require.NotContains(t, string(b), "host_name")

	// Present with the fleets-forward status/detail contract the frontend consumes.
	b, err = json.Marshal(HostMDMOSSettings{
		HostName: &HostMDMHostNameSetting{Status: HostNameSettingFailed, Detail: "boom"},
	})
	require.NoError(t, err)
	require.Contains(t, string(b), `"host_name":{"status":"failed","detail":"boom"}`)
}

func TestPopulateOSSettingsAndMacOSSettingsMatrix(t *testing.T) {
	const fvIdent = "com.fleetdm.fleet.mdm.filevault"

	bothOn := DiskEncryptionConfig{MacOSEnabled: true, MacOSEscrowEnabled: true}
	enforceOnly := DiskEncryptionConfig{MacOSEnabled: true}
	escrowOnly := DiskEncryptionConfig{MacOSEscrowEnabled: true}
	offOff := DiskEncryptionConfig{}

	type want struct {
		status DiskEncryptionStatus // "" means no status
		action ActionRequiredState  // "" means no action
	}
	fvProf := func(op MDMOperationType, status *MDMDeliveryStatus) *HostMDMAppleProfile {
		return &HostMDMAppleProfile{HostUUID: "abc", Identifier: fvIdent, OperationType: op, Status: status}
	}
	w := func(s DiskEncryptionStatus, a ActionRequiredState) want { return want{s, a} }

	// key signals: "none" (no key row), "undecryptable", "unknown" (row, not yet checked), "decryptable"
	keySignals := map[string]*int{"none": new(-1), "undecryptable": new(0), "unknown": nil, "decryptable": new(1)}
	// disk signals: "unknown" (not reported), "unencrypted", "encrypted"
	diskSignals := map[string]*bool{"unknown": nil, "unencrypted": new(false), "encrypted": new(true)}

	keyBasedVerifying := map[string]want{
		"none": w(DiskEncryptionActionRequired, ActionRequiredRotateKey), "undecryptable": w(DiskEncryptionActionRequired, ActionRequiredRotateKey),
		"unknown": w(DiskEncryptionVerifying, ""), "decryptable": w(DiskEncryptionVerifying, ""),
	}
	keyBasedVerified := map[string]want{
		"none": w(DiskEncryptionActionRequired, ActionRequiredRotateKey), "undecryptable": w(DiskEncryptionActionRequired, ActionRequiredRotateKey),
		"unknown": w(DiskEncryptionVerifying, ""), "decryptable": w(DiskEncryptionVerified, ""),
	}
	diskBasedVerifying := map[string]want{
		"unknown": w(DiskEncryptionVerifying, ""), "unencrypted": w(DiskEncryptionActionRequired, ActionRequiredLogOut), "encrypted": w(DiskEncryptionVerifying, ""),
	}
	diskBasedVerified := map[string]want{
		"unknown": w(DiskEncryptionVerifying, ""), "unencrypted": w(DiskEncryptionActionRequired, ActionRequiredLogOut), "encrypted": w(DiskEncryptionVerified, ""),
	}

	fixedCases := []struct {
		name string
		prof *HostMDMAppleProfile
		want want
	}{
		{"no profile", nil, want{}},
		{"pending install", fvProf(MDMOperationTypeInstall, &MDMDeliveryPending), w(DiskEncryptionEnforcing, "")},
		{"null status install", fvProf(MDMOperationTypeInstall, nil), w(DiskEncryptionEnforcing, "")},
		{"failed install", fvProf(MDMOperationTypeInstall, &MDMDeliveryFailed), w(DiskEncryptionFailed, "")},
		{"pending remove", fvProf(MDMOperationTypeRemove, &MDMDeliveryPending), w(DiskEncryptionRemovingEnforcement, "")},
		{"failed remove", fvProf(MDMOperationTypeRemove, &MDMDeliveryFailed), w(DiskEncryptionFailed, "")},
		{"removed", fvProf(MDMOperationTypeRemove, &MDMDeliveryVerifying), want{}},
	}

	check := func(t *testing.T, cfg DiskEncryptionConfig, prof *HostMDMAppleProfile, rawDecryptable *int, disk *bool, exp want) {
		var d MDMHostData
		raw := "null"
		if rawDecryptable != nil {
			raw = fmt.Sprintf("%d", *rawDecryptable)
		}
		require.NoError(t, d.Scan(fmt.Appendf(nil, `{"raw_decryptable": %s}`, raw)))
		var profs []HostMDMAppleProfile
		if prof != nil {
			profs = []HostMDMAppleProfile{*prof}
		}
		d.PopulateOSSettingsAndMacOSSettings(profs, fvIdent, cfg, disk)

		require.NotNil(t, d.MacOSSettings)
		require.NotNil(t, d.OSSettings)
		if exp.status == "" {
			require.Nil(t, d.MacOSSettings.DiskEncryption)
			require.Nil(t, d.OSSettings.DiskEncryption.Status)
		} else {
			require.NotNil(t, d.MacOSSettings.DiskEncryption)
			require.Equal(t, exp.status, *d.MacOSSettings.DiskEncryption)
			require.NotNil(t, d.OSSettings.DiskEncryption.Status)
			require.Equal(t, exp.status, *d.OSSettings.DiskEncryption.Status)
		}
		if exp.action == "" {
			require.Nil(t, d.MacOSSettings.ActionRequired)
		} else {
			require.NotNil(t, d.MacOSSettings.ActionRequired)
			require.Equal(t, exp.action, *d.MacOSSettings.ActionRequired)
		}
	}

	for _, combo := range []struct {
		name     string
		cfg      DiskEncryptionConfig
		keyBased bool
	}{
		{"enforce on, escrow on", bothOn, true},
		{"enforce off, escrow on", escrowOnly, true},
		{"enforce off, escrow off", offOff, true},
		{"enforce on, escrow off", enforceOnly, false},
	} {
		t.Run(combo.name, func(t *testing.T) {
			for _, c := range fixedCases {
				for keyName, key := range keySignals {
					for diskName, disk := range diskSignals {
						t.Run(fmt.Sprintf("%s/key=%s/disk=%s", c.name, keyName, diskName), func(t *testing.T) {
							check(t, combo.cfg, c.prof, key, disk, c.want)
						})
					}
				}
			}

			for _, delivered := range []struct {
				status                      MDMDeliveryStatus
				keyBasedWant, diskBasedWant map[string]want
			}{
				{MDMDeliveryVerifying, keyBasedVerifying, diskBasedVerifying},
				{MDMDeliveryVerified, keyBasedVerified, diskBasedVerified},
			} {
				for keyName, key := range keySignals {
					for diskName, disk := range diskSignals {
						exp := delivered.keyBasedWant[keyName]
						if !combo.keyBased {
							exp = delivered.diskBasedWant[diskName]
						}
						t.Run(fmt.Sprintf("%s install/key=%s/disk=%s", delivered.status, keyName, diskName), func(t *testing.T) {
							status := delivered.status
							check(t, combo.cfg, fvProf(MDMOperationTypeInstall, &status), key, disk, exp)
						})
					}
				}
			}
		})
	}
}
