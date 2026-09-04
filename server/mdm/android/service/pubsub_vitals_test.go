package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

func TestAndroidDeviceVitals(t *testing.T) {
	fullyReportingDevice := &androidmanagement.Device{
		ApiLevel:  36,
		Ownership: DeviceOwnershipCompanyOwned,
		DeviceSettings: &androidmanagement.DeviceSettings{
			AdbEnabled:        true,
			IsDeviceSecure:    true,
			VerifyAppsEnabled: false,
			EncryptionStatus:  "ACTIVE",
		},
		HardwareInfo: &androidmanagement.HardwareInfo{
			Manufacturer: "Google",
		},
		SoftwareInfo: &androidmanagement.SoftwareInfo{
			SecurityPatchLevel:  "2026-05-01",
			DeviceKernelVersion: "6.1.75-android14",
			BootloaderVersion:   "slider-1.4-12345678",
			SystemUpdateInfo: &androidmanagement.SystemUpdateInfo{
				UpdateStatus: "SECURITY_UPDATE_AVAILABLE",
			},
		},
		SecurityPosture: &androidmanagement.SecurityPosture{
			DevicePosture: "POTENTIALLY_COMPROMISED",
			PostureDetails: []*androidmanagement.PostureDetail{
				{
					SecurityRisk: "COMPROMISED_OS",
					Advice: []*androidmanagement.UserFacingMessage{
						{DefaultMessage: "Factory reset the device", LocalizedMessages: map[string]string{"fr": "ignored"}},
					},
				},
			},
		},
		NetworkInfo: &androidmanagement.NetworkInfo{
			Imei: "A1000031212",
			TelephonyInfos: []*androidmanagement.TelephonyInfo{
				{PhoneNumber: "+15555550100", CarrierName: "Acme Mobile", IccId: "8901410321111111111"},
				{PhoneNumber: "+15555550101", CarrierName: "Acme Mobile", ActivationState: "ACTIVATED", ConfigMode: "USER_CONFIGURED"},
			},
		},
	}

	cases := []struct {
		name   string
		device *androidmanagement.Device
		want   fleet.MDMAndroidDeviceVitals
	}{
		{
			name:   "fully reporting device",
			device: fullyReportingDevice,
			want: fleet.MDMAndroidDeviceVitals{
				APILevel:              new(int64(36)),
				AdbEnabled:            new(true),
				PasscodeProtected:     new(true),
				PlayProtectEnabled:    new(false),
				EncryptionType:        new("ACTIVE"),
				Manufacturer:          new("Google"),
				SecurityUpdateVersion: new("2026-05-01"),
				DeviceKernelVersion:   new("6.1.75-android14"),
				BootloaderVersion:     new("slider-1.4-12345678"),
				SystemUpdateStatus:    new("SECURITY_UPDATE_AVAILABLE"),
				SecurityPosture:       new("POTENTIALLY_COMPROMISED"),
				IMEI:                  new("A1000031212"),
				SecurityPostureDetails: []fleet.MDMAndroidPostureDetail{
					{SecurityRisk: "COMPROMISED_OS", Advice: []string{"Factory reset the device"}},
				},
				TelephonyInfos: []fleet.MDMAndroidTelephonyInfo{
					{PhoneNumber: "+15555550100", CarrierName: "Acme Mobile", ICCID: "8901410321111111111"},
					{PhoneNumber: "+15555550101", CarrierName: "Acme Mobile", ActivationState: "ACTIVATED", ConfigMode: "USER_CONFIGURED"},
				},
			},
		},
		{
			// Every section AMAPI can omit is omitted; nothing should be
			// inferred as false or "".
			name:   "device reporting nothing",
			device: &androidmanagement.Device{},
			want:   fleet.MDMAndroidDeviceVitals{},
		},
		{
			// deviceSettingsEnabled off in the applied policy: the settings
			// booleans must stay nil rather than collapse to false.
			name: "device settings not reported",
			device: &androidmanagement.Device{
				HardwareInfo: &androidmanagement.HardwareInfo{Manufacturer: "Motorola"},
			},
			want: fleet.MDMAndroidDeviceVitals{Manufacturer: new("Motorola")},
		},
		{
			// BYOD: AMAPI reports telephonyInfos for fully managed devices only.
			name: "personally owned device reports no telephony info",
			device: &androidmanagement.Device{
				NetworkInfo: &androidmanagement.NetworkInfo{WifiMacAddress: "7c:11:11:11:11:11"},
			},
			want: fleet.MDMAndroidDeviceVitals{},
		},
		{
			// Belt and braces: even if AMAPI did hand us network info for a
			// personally-owned device, neither the phone number nor the
			// hardware radio identifiers must be stored.
			name: "personally owned device with network info drops it",
			device: &androidmanagement.Device{
				Ownership: DeviceOwnershipPersonallyOwned,
				NetworkInfo: &androidmanagement.NetworkInfo{
					Imei:           "A1000031212",
					Meid:           "A00000292788E1",
					TelephonyInfos: []*androidmanagement.TelephonyInfo{{PhoneNumber: "+15555550100"}},
				},
			},
			want: fleet.MDMAndroidDeviceVitals{},
		},
		{
			// AMAPI omits ownership on some status reports. Treating that as
			// personally owned would erase, on the next report, the numbers
			// captured at enrollment -- so unspecified defers to what AMAPI
			// itself chose to send.
			name: "unspecified ownership keeps the telephony AMAPI sent",
			device: &androidmanagement.Device{
				NetworkInfo: &androidmanagement.NetworkInfo{
					TelephonyInfos: []*androidmanagement.TelephonyInfo{{PhoneNumber: "+15555550100"}},
				},
			},
			want: fleet.MDMAndroidDeviceVitals{
				TelephonyInfos: []fleet.MDMAndroidTelephonyInfo{{PhoneNumber: "+15555550100"}},
			},
		},
		{
			name: "explicitly unspecified ownership keeps the network info AMAPI sent",
			device: &androidmanagement.Device{
				Ownership: "OWNERSHIP_UNSPECIFIED",
				NetworkInfo: &androidmanagement.NetworkInfo{
					Imei:           "A1000031212",
					TelephonyInfos: []*androidmanagement.TelephonyInfo{{PhoneNumber: "+15555550100"}},
				},
			},
			want: fleet.MDMAndroidDeviceVitals{
				IMEI:           new("A1000031212"),
				TelephonyInfos: []fleet.MDMAndroidTelephonyInfo{{PhoneNumber: "+15555550100"}},
			},
		},
		{
			// A CDMA device reports meid in place of imei, with no telephony
			// info of its own; the other identifier stays nil rather than "".
			name: "cdma device reports meid only",
			device: &androidmanagement.Device{
				Ownership:   DeviceOwnershipCompanyOwned,
				NetworkInfo: &androidmanagement.NetworkInfo{Meid: "A00000292788E1"},
			},
			want: fleet.MDMAndroidDeviceVitals{MEID: new("A00000292788E1")},
		},
		{
			// Device-reported, so an oversized identifier must be truncated to
			// the column width like every other string vital.
			name: "oversized radio identifiers are truncated to the column width",
			device: &androidmanagement.Device{
				Ownership: DeviceOwnershipCompanyOwned,
				NetworkInfo: &androidmanagement.NetworkInfo{
					Imei: strings.Repeat("1", 300),
					Meid: strings.Repeat("2", 300),
				},
			},
			want: fleet.MDMAndroidDeviceVitals{
				IMEI: new(strings.Repeat("1", fleet.MDMAndroidDeviceVitalMaxLength)),
				MEID: new(strings.Repeat("2", fleet.MDMAndroidDeviceVitalMaxLength)),
			},
		},
		{
			// UPDATE_STATUS_UNKNOWN is the system-update enum's "no data"
			// member; it carries no _UNSPECIFIED suffix.
			name: "unknown system update status is dropped",
			device: &androidmanagement.Device{
				SoftwareInfo: &androidmanagement.SoftwareInfo{
					SystemUpdateInfo: &androidmanagement.SystemUpdateInfo{UpdateStatus: "UPDATE_STATUS_UNKNOWN"},
				},
			},
			want: fleet.MDMAndroidDeviceVitals{},
		},
		{
			// AMAPI's *_UNSPECIFIED sentinels mean "no data", not a state, and
			// physical SIMs report them on every device below Android 15.
			name: "unspecified enum sentinels are dropped",
			device: &androidmanagement.Device{
				DeviceSettings:  &androidmanagement.DeviceSettings{EncryptionStatus: "ENCRYPTION_STATUS_UNSPECIFIED"},
				SecurityPosture: &androidmanagement.SecurityPosture{DevicePosture: "POSTURE_UNSPECIFIED"},
				NetworkInfo: &androidmanagement.NetworkInfo{
					TelephonyInfos: []*androidmanagement.TelephonyInfo{{
						PhoneNumber:     "+15555550100",
						ActivationState: "ACTIVATION_STATE_UNSPECIFIED",
						ConfigMode:      "CONFIG_MODE_UNSPECIFIED",
					}},
				},
			},
			want: fleet.MDMAndroidDeviceVitals{
				AdbEnabled:         new(false),
				PasscodeProtected:  new(false),
				PlayProtectEnabled: new(false),
				TelephonyInfos:     []fleet.MDMAndroidTelephonyInfo{{PhoneNumber: "+15555550100"}},
			},
		},
		{
			// A posture detail carrying neither a risk nor advice is nothing
			// worth storing.
			name: "posture detail with only an unspecified risk is dropped",
			device: &androidmanagement.Device{
				SecurityPosture: &androidmanagement.SecurityPosture{
					DevicePosture: "SECURE",
					PostureDetails: []*androidmanagement.PostureDetail{
						{SecurityRisk: "SECURITY_RISK_UNSPECIFIED"},
					},
				},
			},
			want: fleet.MDMAndroidDeviceVitals{SecurityPosture: new("SECURE")},
		},
		{
			// Device-reported strings must not blow past the column width and
			// fail the whole status report under MySQL strict mode.
			name: "oversized values are truncated to the column width",
			device: &androidmanagement.Device{
				HardwareInfo: &androidmanagement.HardwareInfo{Manufacturer: strings.Repeat("a", 300)},
			},
			want: fleet.MDMAndroidDeviceVitals{
				Manufacturer: new(strings.Repeat("a", fleet.MDMAndroidDeviceVitalMaxLength)),
			},
		},
		{
			// Truncation must be by rune: slicing a multi-byte string by bytes
			// can cut mid-rune and produce invalid UTF-8 that the utf8mb4
			// column rejects.
			name: "oversized multi-byte values are truncated on a rune boundary",
			device: &androidmanagement.Device{
				HardwareInfo: &androidmanagement.HardwareInfo{Manufacturer: strings.Repeat("\u00e9", 300)},
			},
			want: fleet.MDMAndroidDeviceVitals{
				Manufacturer: new(strings.Repeat("\u00e9", fleet.MDMAndroidDeviceVitalMaxLength)),
			},
		},
		{
			// api_level is a bigint matching AMAPI's int64, so a large value
			// stores rather than failing the write.
			name:   "large api level is stored",
			device: &androidmanagement.Device{ApiLevel: math.MaxInt32 + 1},
			want:   fleet.MDMAndroidDeviceVitals{APILevel: new(int64(math.MaxInt32 + 1))},
		},
		{
			// 0 is AMAPI's "not reported" for this field.
			name:   "zero api level is dropped",
			device: &androidmanagement.Device{ApiLevel: 0},
			want:   fleet.MDMAndroidDeviceVitals{},
		},
		{
			// softwareInfo present but the device is too old to report a
			// pending system update.
			name: "software info without system update info",
			device: &androidmanagement.Device{
				SoftwareInfo: &androidmanagement.SoftwareInfo{SecurityPatchLevel: "2025-01-01"},
			},
			want: fleet.MDMAndroidDeviceVitals{SecurityUpdateVersion: new("2025-01-01")},
		},
		{
			// A risk with no advice attached, and an advice entry with no
			// default message, must not produce empty strings.
			name: "security posture without advice",
			device: &androidmanagement.Device{
				SecurityPosture: &androidmanagement.SecurityPosture{
					DevicePosture: "AT_RISK",
					PostureDetails: []*androidmanagement.PostureDetail{
						{SecurityRisk: "UNKNOWN_OS"},
						{SecurityRisk: "HARDWARE_BACKED_EVALUATION_FAILED", Advice: []*androidmanagement.UserFacingMessage{{}}},
					},
				},
			},
			want: fleet.MDMAndroidDeviceVitals{
				SecurityPosture: new("AT_RISK"),
				SecurityPostureDetails: []fleet.MDMAndroidPostureDetail{
					{SecurityRisk: "UNKNOWN_OS"},
					{SecurityRisk: "HARDWARE_BACKED_EVALUATION_FAILED"},
				},
			},
		},
		{
			// AMAPI sends empty strings for values it has no data for; those
			// must be stored as NULL, not "".
			name: "empty strings become nil",
			device: &androidmanagement.Device{
				DeviceSettings:  &androidmanagement.DeviceSettings{EncryptionStatus: ""},
				HardwareInfo:    &androidmanagement.HardwareInfo{Manufacturer: ""},
				SecurityPosture: &androidmanagement.SecurityPosture{DevicePosture: ""},
			},
			want: fleet.MDMAndroidDeviceVitals{
				AdbEnabled:         new(false),
				PasscodeProtected:  new(false),
				PlayProtectEnabled: new(false),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, androidDeviceVitals(c.device))
		})
	}
}

// TestUpdateHostPersistsVitals covers the ingestion wiring: a status report
// must upsert the extracted vitals against the host's UUID.
func TestUpdateHostPersistsVitals(t *testing.T) {
	svc, mockDS := createAndroidService(t)

	enterpriseSpecificID := "VITALS-UUID-12345"
	existingHost := wireDedupHost(t, mockDS, 1, enterpriseSpecificID)

	var (
		gotHostUUID string
		gotVitals   fleet.MDMAndroidDeviceVitals
	)
	mockDS.SetOrUpdateHostMDMAndroidDeviceVitalsFunc = func(ctx context.Context, hostUUID string, vitals fleet.MDMAndroidDeviceVitals) error {
		gotHostUUID = hostUUID
		gotVitals = vitals
		return nil
	}

	device := &androidmanagement.Device{
		Name:     createAndroidDeviceId("vitals-host"),
		ApiLevel: 36,
		HardwareInfo: &androidmanagement.HardwareInfo{
			EnterpriseSpecificId: enterpriseSpecificID,
			Brand:                "Google",
			Manufacturer:         "Google",
			Model:                "Pixel 9",
			SerialNumber:         "vitals-serial",
			Hardware:             "tokay",
		},
		SoftwareInfo: &androidmanagement.SoftwareInfo{
			AndroidBuildNumber: "vitals-build",
			AndroidVersion:     "16",
			SecurityPatchLevel: "2026-05-01",
		},
		DeviceSettings: &androidmanagement.DeviceSettings{
			AdbEnabled:       true,
			IsDeviceSecure:   true,
			EncryptionStatus: "ACTIVE",
		},
		Ownership: DeviceOwnershipCompanyOwned,
		NetworkInfo: &androidmanagement.NetworkInfo{
			Imei: "A1000031212",
			TelephonyInfos: []*androidmanagement.TelephonyInfo{
				{PhoneNumber: "+15555550100", CarrierName: "Acme Mobile"},
			},
		},
		MemoryInfo: &androidmanagement.MemoryInfo{
			TotalRam:             int64(8 * 1024 * 1024 * 1024),
			TotalInternalStorage: int64(128 * 1024 * 1024 * 1024),
		},
		LastStatusReportTime: "2026-05-02T12:00:00Z",
	}

	deviceBytes, err := json.Marshal(device)
	require.NoError(t, err)
	require.NoError(t, svc.ProcessPubSubPush(t.Context(), "value", &android.PubSubMessage{
		Attributes: map[string]string{"notificationType": string(android.PubSubStatusReport)},
		Data:       base64.StdEncoding.EncodeToString(deviceBytes),
	}))

	require.True(t, mockDS.SetOrUpdateHostMDMAndroidDeviceVitalsFuncInvoked)
	require.Equal(t, existingHost.Host.UUID, gotHostUUID)
	require.Equal(t, new(int64(36)), gotVitals.APILevel)
	require.Equal(t, new(true), gotVitals.AdbEnabled)
	require.Equal(t, new("ACTIVE"), gotVitals.EncryptionType)
	require.Equal(t, new("Google"), gotVitals.Manufacturer)
	require.Equal(t, new("2026-05-01"), gotVitals.SecurityUpdateVersion)
	// A company-owned device's telephony info and radio identifier must
	// survive the status-report path, not just enrollment.
	require.Equal(t,
		[]fleet.MDMAndroidTelephonyInfo{{PhoneNumber: "+15555550100", CarrierName: "Acme Mobile"}},
		gotVitals.TelephonyInfos)
	require.Equal(t, new("A1000031212"), gotVitals.IMEI)
}

// TestEnrollmentPersistsVitals covers the enrollment wiring: the ENROLLMENT
// payload already carries the vitals, so a freshly enrolled host must not have
// to wait for its first status report to show them.
func TestEnrollmentPersistsVitals(t *testing.T) {
	const enrollSecret = "global"

	cases := []struct {
		name              string
		ownership         string
		wantPhoneNumbers  []fleet.MDMAndroidTelephonyInfo
		wantIMEI          *string
		wantAdbEnabled    *bool
		wantAPILevelIsSet bool
	}{
		{
			name:              "company owned",
			ownership:         DeviceOwnershipCompanyOwned,
			wantPhoneNumbers:  []fleet.MDMAndroidTelephonyInfo{{PhoneNumber: "+15555550100", CarrierName: "Acme Mobile"}},
			wantIMEI:          new("A1000031212"),
			wantAdbEnabled:    new(true),
			wantAPILevelIsSet: true,
		},
		{
			// A personally-owned host must never carry a phone number or a
			// hardware radio identifier.
			name:              "personally owned",
			ownership:         DeviceOwnershipPersonallyOwned,
			wantPhoneNumbers:  nil,
			wantIMEI:          nil,
			wantAdbEnabled:    new(true),
			wantAPILevelIsSet: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, mockDS := createAndroidService(t)
			mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
			}
			mockDS.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
				return &fleet.EnrollSecret{Secret: enrollSecret}, nil
			}
			mockDS.AndroidHostLiteFunc = func(ctx context.Context, enterpriseSpecificID string) (*fleet.AndroidHost, error) {
				return nil, common_mysql.NotFound("android host lite")
			}
			mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, companyOwned bool) (*fleet.AndroidHost, error) {
				host.Host.ID = 7
				return host, nil
			}

			var (
				gotHostUUID string
				gotVitals   fleet.MDMAndroidDeviceVitals
			)
			mockDS.SetOrUpdateHostMDMAndroidDeviceVitalsFunc = func(ctx context.Context, hostUUID string, vitals fleet.MDMAndroidDeviceVitals) error {
				gotHostUUID = hostUUID
				gotVitals = vitals
				return nil
			}

			enrollTokenData, err := json.Marshal(enrollmentTokenRequest{EnrollSecret: enrollSecret})
			require.NoError(t, err)
			message := createEnrollmentMessage(t, androidmanagement.Device{
				Name:                createAndroidDeviceId("vitals-enroll"),
				EnrollmentTokenData: string(enrollTokenData),
				Ownership:           tc.ownership,
				ApiLevel:            36,
				DeviceSettings: &androidmanagement.DeviceSettings{
					AdbEnabled:       true,
					EncryptionStatus: "ACTIVE",
				},
				NetworkInfo: &androidmanagement.NetworkInfo{
					Imei: "A1000031212",
					TelephonyInfos: []*androidmanagement.TelephonyInfo{
						{PhoneNumber: "+15555550100", CarrierName: "Acme Mobile"},
					},
				},
			})
			require.NoError(t, svc.ProcessPubSubPush(t.Context(), "value", message))

			require.True(t, mockDS.SetOrUpdateHostMDMAndroidDeviceVitalsFuncInvoked)
			require.NotEmpty(t, gotHostUUID)
			require.Equal(t, tc.wantAdbEnabled, gotVitals.AdbEnabled)
			require.Equal(t, new("ACTIVE"), gotVitals.EncryptionType)
			require.Equal(t, tc.wantPhoneNumbers, gotVitals.TelephonyInfos)
			require.Equal(t, tc.wantIMEI, gotVitals.IMEI)
			if tc.wantAPILevelIsSet {
				require.Equal(t, new(int64(36)), gotVitals.APILevel)
			}
		})
	}
}
