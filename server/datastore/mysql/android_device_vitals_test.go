package mysql

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestHostMDMAndroidDeviceVitals(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"InsertThenUpdate", testHostMDMAndroidDeviceVitalsInsertThenUpdate},
		{"NullHandling", testHostMDMAndroidDeviceVitalsNullHandling},
		{"OverwriteClearsUnreportedVitals", testHostMDMAndroidDeviceVitalsOverwriteClears},
		{"Load", testLoadHostMDMAndroidDeviceVitalsDB},
		{"LoadPartial", testLoadHostMDMAndroidDeviceVitalsDBPartial},
		{"LoadNoRow", testLoadHostMDMAndroidDeviceVitalsDBNoRow},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func fullAndroidVitals() fleet.MDMAndroidDeviceVitals {
	return fleet.MDMAndroidDeviceVitals{
		AdbEnabled:            new(true),
		PasscodeProtected:     new(true),
		PlayProtectEnabled:    new(false),
		EncryptionType:        new("ACTIVE"),
		Manufacturer:          new("Google"),
		SecurityUpdateVersion: new("2026-05-01"),
		DeviceKernelVersion:   new("6.1.75-android14"),
		BootloaderVersion:     new("slider-1.4-12345678"),
		SystemUpdateStatus:    new("UP_TO_DATE"),
		SecurityPosture:       new("SECURE"),
		IMEI:                  new("A1000031212"),
		MEID:                  new("A00000292788E1"),
		APILevel:              new(int64(36)),
		SecurityPostureDetails: []fleet.MDMAndroidPostureDetail{
			{SecurityRisk: "UNKNOWN_OS", Advice: []string{"Update the OS"}},
		},
		TelephonyInfos: []fleet.MDMAndroidTelephonyInfo{
			{PhoneNumber: "+15555550100", CarrierName: "Acme Mobile", ICCID: "8901410321111111111"},
			{PhoneNumber: "+15555550101", CarrierName: "Acme Mobile", ActivationState: "ACTIVATED"},
		},
	}
}

func testHostMDMAndroidDeviceVitalsInsertThenUpdate(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "android-vitals-host", "1.1.1.1", "android-vitals-key", "android-vitals-uuid", time.Now(),
		test.WithPlatform("android"))

	vitals := fullAndroidVitals()
	require.NoError(t, ds.SetOrUpdateHostMDMAndroidDeviceVitals(ctx, host.UUID, vitals))

	var row struct {
		Manufacturer   *string `db:"manufacturer"`
		APILevel       *int64  `db:"api_level"`
		AdbEnabled     *bool   `db:"adb_enabled"`
		IMEI           *string `db:"imei"`
		MEID           *string `db:"meid"`
		TelephonyInfos []byte  `db:"telephony_infos"`
	}
	require.NoError(t, ds.writer(ctx).Get(&row, `
		SELECT manufacturer, api_level, adb_enabled, imei, meid, telephony_infos
		FROM host_mdm_android_device_vitals WHERE host_uuid = ?`, host.UUID))
	require.Equal(t, "Google", *row.Manufacturer)
	require.Equal(t, int64(36), *row.APILevel)
	require.True(t, *row.AdbEnabled)
	require.Equal(t, "A1000031212", *row.IMEI)
	require.Equal(t, "A00000292788E1", *row.MEID)

	// Round-trip the JSON column back into the same Go type used to write it.
	var gotTelephony []fleet.MDMAndroidTelephonyInfo
	require.NoError(t, json.Unmarshal(row.TelephonyInfos, &gotTelephony))
	require.Equal(t, vitals.TelephonyInfos, gotTelephony)

	var count int
	require.NoError(t, ds.writer(ctx).Get(&count, `SELECT COUNT(*) FROM host_mdm_android_device_vitals WHERE host_uuid = ?`, host.UUID))
	require.Equal(t, 1, count)

	// A second status report for the same host must UPDATE the existing row,
	// not insert a second one. The radio identifiers change to distinct values
	// so that the UPDATE's two adjacent, identically-typed named parameters
	// can't be swapped without failing.
	vitals.Manufacturer = new("Motorola")
	vitals.IMEI = new("B2000042323")
	vitals.MEID = new("B00000383899F2")
	require.NoError(t, ds.SetOrUpdateHostMDMAndroidDeviceVitals(ctx, host.UUID, vitals))

	require.NoError(t, ds.writer(ctx).Get(&count, `SELECT COUNT(*) FROM host_mdm_android_device_vitals WHERE host_uuid = ?`, host.UUID))
	require.Equal(t, 1, count)
	require.NoError(t, ds.writer(ctx).Get(&row, `
		SELECT manufacturer, api_level, adb_enabled, imei, meid, telephony_infos
		FROM host_mdm_android_device_vitals WHERE host_uuid = ?`, host.UUID))
	require.Equal(t, "Motorola", *row.Manufacturer)
	require.Equal(t, "B2000042323", *row.IMEI)
	require.Equal(t, "B00000383899F2", *row.MEID)
}

func testHostMDMAndroidDeviceVitalsNullHandling(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "android-null-host", "1.1.1.2", "android-null-key", "android-null-uuid", time.Now(),
		test.WithPlatform("android"))

	// A section the device didn't report (nil in the Go struct) must persist as
	// SQL NULL, not error and not the JSON literal "null".
	require.NoError(t, ds.SetOrUpdateHostMDMAndroidDeviceVitals(ctx, host.UUID, fleet.MDMAndroidDeviceVitals{}))

	var row struct {
		AdbEnabled             *bool   `db:"adb_enabled"`
		Manufacturer           *string `db:"manufacturer"`
		APILevel               *int64  `db:"api_level"`
		IMEI                   *string `db:"imei"`
		MEID                   *string `db:"meid"`
		SecurityPostureDetails []byte  `db:"security_posture_details"`
		TelephonyInfos         []byte  `db:"telephony_infos"`
	}
	require.NoError(t, ds.writer(ctx).Get(&row, `
		SELECT adb_enabled, manufacturer, api_level, imei, meid, security_posture_details, telephony_infos
		FROM host_mdm_android_device_vitals WHERE host_uuid = ?`, host.UUID))
	require.Nil(t, row.AdbEnabled)
	require.Nil(t, row.Manufacturer)
	require.Nil(t, row.APILevel)
	require.Nil(t, row.IMEI)
	require.Nil(t, row.MEID)
	require.Nil(t, row.SecurityPostureDetails)
	require.Nil(t, row.TelephonyInfos)
}

func testHostMDMAndroidDeviceVitalsOverwriteClears(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "android-clear-host", "1.1.1.3", "android-clear-key", "android-clear-uuid", time.Now(),
		test.WithPlatform("android"))

	require.NoError(t, ds.SetOrUpdateHostMDMAndroidDeviceVitals(ctx, host.UUID, fullAndroidVitals()))

	// A status report is a full snapshot, so a section that stops being
	// reported (the policy turning deviceSettingsEnabled off, say) must clear
	// the stale values rather than leave them on the host.
	require.NoError(t, ds.SetOrUpdateHostMDMAndroidDeviceVitals(ctx, host.UUID, fleet.MDMAndroidDeviceVitals{
		Manufacturer: new("Google"),
	}))

	loaded := &fleet.Host{UUID: host.UUID}
	require.NoError(t, ds.LoadHostMDMAndroidDeviceVitals(ctx, loaded))
	require.Equal(t, "Google", *loaded.Manufacturer)
	require.Nil(t, loaded.AdbEnabled)
	require.Nil(t, loaded.EncryptionType)
	require.Nil(t, loaded.SecurityPosture)
	require.Nil(t, loaded.SecurityPostureDetails)
	require.Nil(t, loaded.TelephonyInfos)
	require.Nil(t, loaded.IMEI)
	require.Nil(t, loaded.MEID)
}

func testLoadHostMDMAndroidDeviceVitalsDB(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "android-load-host", "1.1.1.4", "android-load-key", "android-load-uuid", time.Now(),
		test.WithPlatform("android"))

	vitals := fullAndroidVitals()
	require.NoError(t, ds.SetOrUpdateHostMDMAndroidDeviceVitals(ctx, host.UUID, vitals))

	require.NoError(t, ds.LoadHostMDMAndroidDeviceVitals(ctx, host))
	require.Equal(t, vitals.AdbEnabled, host.AdbEnabled)
	require.Equal(t, vitals.PasscodeProtected, host.PasscodeProtected)
	require.Equal(t, vitals.PlayProtectEnabled, host.PlayProtectEnabled)
	require.Equal(t, vitals.EncryptionType, host.EncryptionType)
	require.Equal(t, vitals.Manufacturer, host.Manufacturer)
	require.Equal(t, vitals.SecurityUpdateVersion, host.SecurityUpdateVersion)
	require.Equal(t, vitals.DeviceKernelVersion, host.DeviceKernelVersion)
	require.Equal(t, vitals.BootloaderVersion, host.BootloaderVersion)
	require.Equal(t, vitals.SystemUpdateStatus, host.SystemUpdateStatus)
	require.Equal(t, vitals.SecurityPosture, host.SecurityPosture)
	require.Equal(t, vitals.IMEI, host.IMEI)
	require.Equal(t, vitals.MEID, host.MEID)
	require.Equal(t, vitals.APILevel, host.APILevel)
	require.Equal(t, vitals.SecurityPostureDetails, host.SecurityPostureDetails)
	require.Equal(t, vitals.TelephonyInfos, host.TelephonyInfos)
}

func testLoadHostMDMAndroidDeviceVitalsDBPartial(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "android-partial-host", "1.1.1.5", "android-partial-key", "android-partial-uuid", time.Now(),
		test.WithPlatform("android"))

	require.NoError(t, ds.SetOrUpdateHostMDMAndroidDeviceVitals(ctx, host.UUID, fleet.MDMAndroidDeviceVitals{
		Manufacturer: new("Samsung"),
		APILevel:     new(int64(34)),
	}))

	require.NoError(t, ds.LoadHostMDMAndroidDeviceVitals(ctx, host))
	require.Equal(t, "Samsung", *host.Manufacturer)
	require.Equal(t, int64(34), *host.APILevel)
	require.Nil(t, host.AdbEnabled)
	require.Nil(t, host.SecurityPosture)
	require.Nil(t, host.SecurityPostureDetails)
	require.Nil(t, host.TelephonyInfos)
	require.Nil(t, host.IMEI)
	require.Nil(t, host.MEID)
}

func testLoadHostMDMAndroidDeviceVitalsDBNoRow(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "android-norow-host", "1.1.1.6", "android-norow-key", "android-norow-uuid", time.Now(),
		test.WithPlatform("android"))

	// A host that hasn't sent a status report yet has no row at all; that's not
	// an error, it just leaves every field nil.
	require.NoError(t, ds.LoadHostMDMAndroidDeviceVitals(ctx, host))
	require.Nil(t, host.Manufacturer)
	require.Nil(t, host.AdbEnabled)
	require.Nil(t, host.TelephonyInfos)
	require.Nil(t, host.IMEI)
	require.Nil(t, host.MEID)
}
