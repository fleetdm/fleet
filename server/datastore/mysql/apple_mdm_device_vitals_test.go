package mysql

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestHostMDMAppleDeviceVitals(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"InsertThenUpdate", testHostMDMAppleDeviceVitalsInsertThenUpdate},
		{"NullHandling", testHostMDMAppleDeviceVitalsNullHandling},
		{"ServiceSubscriptionsReplace", testHostMDMAppleDeviceVitalsServiceSubscriptionsReplace},
		{"ResubmitIdenticalPayload", testHostMDMAppleDeviceVitalsResubmitIdenticalPayload},
		{"Load", testLoadHostMDMAppleDeviceVitalsDB},
		{"LoadPartial", testLoadHostMDMAppleDeviceVitalsDBPartial},
		{"LoadNoRow", testLoadHostMDMAppleDeviceVitalsDBNoRow},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testHostMDMAppleDeviceVitalsInsertThenUpdate(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "vitals-host", "1.1.1.1", "vitals-host-key", "vitals-host-uuid", time.Now(), test.WithPlatform("ipados"))

	vitals := fleet.MDMAppleDeviceVitals{
		UDID:               new("00008030-AAA"),
		BatteryLevel:       new(0.75),
		CellularTechnology: new(int64(1)),
		AccessibilitySettings: &fleet.MDMAppleAccessibilitySettings{
			VoiceOverEnabled: new(true),
		},
		DevicePropertiesAttestation: [][]byte{[]byte("leaf-cert"), []byte("intermediate-cert")},
	}
	require.NoError(t, ds.SetOrUpdateHostMDMAppleDeviceVitals(ctx, host.UUID, vitals))

	var row struct {
		UDID                        *string  `db:"udid"`
		BatteryLevel                *float64 `db:"battery_level"`
		DevicePropertiesAttestation []byte   `db:"device_properties_attestation"`
	}
	require.NoError(t, ds.writer(ctx).Get(&row, `SELECT udid, battery_level, device_properties_attestation FROM host_mdm_apple_device_vitals WHERE host_uuid = ?`, host.UUID))
	require.Equal(t, "00008030-AAA", *row.UDID)
	require.InDelta(t, 0.75, *row.BatteryLevel, 0.001)

	// Round-trip the JSON column back into the same Go type used to write it.
	var gotAttestation [][]byte
	require.NoError(t, json.Unmarshal(row.DevicePropertiesAttestation, &gotAttestation))
	require.Equal(t, [][]byte{[]byte("leaf-cert"), []byte("intermediate-cert")}, gotAttestation)

	var count int
	require.NoError(t, ds.writer(ctx).Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_device_vitals WHERE host_uuid = ?`, host.UUID))
	require.Equal(t, 1, count)

	// A second call for the same host must UPDATE the existing row, not insert a
	// second one.
	vitals.UDID = new("00008030-BBB")
	require.NoError(t, ds.SetOrUpdateHostMDMAppleDeviceVitals(ctx, host.UUID, vitals))

	require.NoError(t, ds.writer(ctx).Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_device_vitals WHERE host_uuid = ?`, host.UUID))
	require.Equal(t, 1, count)
	require.NoError(t, ds.writer(ctx).Get(&row, `SELECT udid, battery_level FROM host_mdm_apple_device_vitals WHERE host_uuid = ?`, host.UUID))
	require.Equal(t, "00008030-BBB", *row.UDID)
}

func testHostMDMAppleDeviceVitalsNullHandling(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "vitals-null-host", "1.1.1.2", "vitals-null-host-key", "vitals-null-host-uuid", time.Now(), test.WithPlatform("ios"))

	// A field absent from the ack (nil in the Go struct) must persist as SQL
	// NULL, not error.
	require.NoError(t, ds.SetOrUpdateHostMDMAppleDeviceVitals(ctx, host.UUID, fleet.MDMAppleDeviceVitals{}))

	var row struct {
		UDID                   *string  `db:"udid"`
		BatteryLevel           *float64 `db:"battery_level"`
		AccessibilitySettings  []byte   `db:"accessibility_settings"`
		DevicePropertiesAttest []byte   `db:"device_properties_attestation"`
	}
	require.NoError(t, ds.writer(ctx).Get(&row, `
		SELECT udid, battery_level, accessibility_settings, device_properties_attestation
		FROM host_mdm_apple_device_vitals WHERE host_uuid = ?`, host.UUID))
	require.Nil(t, row.UDID)
	require.Nil(t, row.BatteryLevel)
	require.Nil(t, row.AccessibilitySettings)
	require.Nil(t, row.DevicePropertiesAttest)
}

func testHostMDMAppleDeviceVitalsServiceSubscriptionsReplace(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "vitals-subs-host", "1.1.1.3", "vitals-subs-host-key", "vitals-subs-host-uuid", time.Now(), test.WithPlatform("ios"))

	// Slot names and the sparse second row mirror a real physical+eSIM
	// dual-SIM device observed during manual testing: the active line reports
	// most fields, while an inactive/unprovisioned eSIM slot reports only
	// EID/IMEI (everything else NULL).
	vitals := fleet.MDMAppleDeviceVitals{
		ServiceSubscriptions: []fleet.MDMAppleServiceSubscription{
			{Slot: "CTSubscriptionSlotOne", ICCID: new("iccid-1"), IsDataPreferred: new(true)},
			{Slot: "CTSubscriptionSlotTwo", EID: new("eid-2")},
		},
	}
	require.NoError(t, ds.SetOrUpdateHostMDMAppleDeviceVitals(ctx, host.UUID, vitals))

	var slots []string
	require.NoError(t, ds.writer(ctx).Select(&slots, `SELECT slot FROM host_mdm_apple_service_subscriptions WHERE host_uuid = ? ORDER BY slot`, host.UUID))
	require.Equal(t, []string{"CTSubscriptionSlotOne", "CTSubscriptionSlotTwo"}, slots)

	var slotTwoICCID *string
	require.NoError(t, ds.writer(ctx).Get(&slotTwoICCID, `SELECT iccid FROM host_mdm_apple_service_subscriptions WHERE host_uuid = ? AND slot = ?`,
		host.UUID, "CTSubscriptionSlotTwo"))
	require.Nil(t, slotTwoICCID, "an unprovisioned eSIM slot's absent fields must persist as NULL")

	// A subsequent call with a different set of slots must drop the stale slot
	// (the eSIM got provisioned and dropped out of the response, say) and
	// update the surviving one in place (not duplicate-insert it).
	vitals.ServiceSubscriptions = []fleet.MDMAppleServiceSubscription{
		{Slot: "CTSubscriptionSlotOne", ICCID: new("iccid-1-updated"), IsDataPreferred: new(true)},
	}
	require.NoError(t, ds.SetOrUpdateHostMDMAppleDeviceVitals(ctx, host.UUID, vitals))

	require.NoError(t, ds.writer(ctx).Select(&slots, `SELECT slot FROM host_mdm_apple_service_subscriptions WHERE host_uuid = ? ORDER BY slot`, host.UUID))
	require.Equal(t, []string{"CTSubscriptionSlotOne"}, slots)

	var iccid string
	require.NoError(t, ds.writer(ctx).Get(&iccid, `SELECT iccid FROM host_mdm_apple_service_subscriptions WHERE host_uuid = ? AND slot = ?`,
		host.UUID, "CTSubscriptionSlotOne"))
	require.Equal(t, "iccid-1-updated", iccid)
}

func testHostMDMAppleDeviceVitalsResubmitIdenticalPayload(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "vitals-idempotent-host", "1.1.1.9", "vitals-idempotent-host-key", "vitals-idempotent-host-uuid", time.Now(), test.WithPlatform("ios"))

	vitals := fleet.MDMAppleDeviceVitals{
		UDID:         new("00008030-CCC"),
		BatteryLevel: new(0.5),
		ServiceSubscriptions: []fleet.MDMAppleServiceSubscription{
			{Slot: "CTSubscriptionSlotOne", ICCID: new("iccid-1")},
		},
	}
	require.NoError(t, ds.SetOrUpdateHostMDMAppleDeviceVitals(ctx, host.UUID, vitals))
	// Resubmitting the exact same values (main row and subscription slot) must
	// not error. Relies on clientFoundRows=true in the connection DSN so
	// RowsAffected reflects rows matched, not rows changed -- otherwise the
	// no-op UPDATE would report 0 affected and the fallback INSERT would hit a
	// duplicate key.
	require.NoError(t, ds.SetOrUpdateHostMDMAppleDeviceVitals(ctx, host.UUID, vitals))

	var count int
	require.NoError(t, ds.writer(ctx).Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_device_vitals WHERE host_uuid = ?`, host.UUID))
	require.Equal(t, 1, count)
	require.NoError(t, ds.writer(ctx).Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_service_subscriptions WHERE host_uuid = ?`, host.UUID))
	require.Equal(t, 1, count)
}

func testLoadHostMDMAppleDeviceVitalsDB(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "vitals-load-host", "1.1.1.4", "vitals-load-host-key", "vitals-load-host-uuid", time.Now(), test.WithPlatform("ios"))

	lastBackup := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	vitals := fleet.MDMAppleDeviceVitals{
		UDID:                          new("00008030-AAA"),
		ModelNumber:                   new("MNEP3LL/A"),
		ModemFirmwareVersion:          new("2.01.00"),
		SupplementalBuildVersion:      new("21E236"),
		SupplementalOSVersionExtra:    new("a"),
		BluetoothMAC:                  new("a4:83:e7:12:34:57"),
		WiFiMAC:                       new("a4:83:e7:12:34:58"),
		EASDeviceIdentifier:           new("3E2A1F9C"),
		ITunesStoreAccountHash:        new("a1b2c3"),
		PushToken:                     []byte("push-token-bytes"),
		BatteryLevel:                  new(0.87),
		CellularTechnology:            new(int64(1)),
		AppAnalyticsEnabled:           new(true),
		AwaitingConfiguration:         new(false),
		DataRoamingEnabled:            new(false),
		DiagnosticSubmissionEnabled:   new(true),
		IsCloudBackupEnabled:          new(true),
		IsDeviceLocatorServiceEnabled: new(true),
		IsDoNotDisturbInEffect:        new(false),
		IsMDMLostModeEnabled:          new(false),
		IsNetworkTethered:             new(false),
		ITunesStoreAccountIsActive:    new(true),
		PersonalHotspotEnabled:        new(false),
		LastCloudBackupDate:           &lastBackup,
		AccessibilitySettings: &fleet.MDMAppleAccessibilitySettings{
			VoiceOverEnabled: new(true),
			GrayscaleEnabled: new(false),
		},
		OrganizationInfo: &fleet.MDMAppleOrganizationInfo{
			OrganizationName: new("Acme Corp"),
		},
		MDMOptions: &fleet.MDMAppleDeviceVitalsMDMOptions{
			BootstrapTokenAllowed: new(true),
		},
		DevicePropertiesAttestation: [][]byte{[]byte("leaf-cert"), []byte("intermediate-cert")},
		ServiceSubscriptions: []fleet.MDMAppleServiceSubscription{
			{Slot: "CTSubscriptionSlotOne", ICCID: new("iccid-1")},
			{Slot: "CTSubscriptionSlotTwo", EID: new("eid-2")},
		},
	}
	require.NoError(t, ds.SetOrUpdateHostMDMAppleDeviceVitals(ctx, host.UUID, vitals))

	loaded := fleet.Host{UUID: host.UUID}
	require.NoError(t, ds.LoadHostMDMAppleDeviceVitals(ctx, &loaded))

	require.Equal(t, "00008030-AAA", *loaded.UDID)
	require.Equal(t, "MNEP3LL/A", *loaded.ModelNumber)
	require.Equal(t, "2.01.00", *loaded.ModemFirmwareVersion)
	require.Equal(t, "21E236", *loaded.SupplementalBuildVersion)
	require.Equal(t, "a", *loaded.SupplementalOSVersionExtra)
	require.Equal(t, "a4:83:e7:12:34:57", *loaded.BluetoothMAC)
	require.Equal(t, "a4:83:e7:12:34:58", *loaded.WiFiMAC)
	require.Equal(t, "3E2A1F9C", *loaded.EASDeviceIdentifier)
	require.Equal(t, "a1b2c3", *loaded.ITunesStoreAccountHash)
	require.Equal(t, []byte("push-token-bytes"), loaded.PushToken)
	require.InDelta(t, 0.87, *loaded.BatteryLevel, 0.001)
	require.EqualValues(t, 1, *loaded.CellularTechnology)
	require.True(t, *loaded.AppAnalyticsEnabled)
	require.False(t, *loaded.AwaitingConfiguration)
	require.False(t, *loaded.DataRoamingEnabled)
	require.True(t, *loaded.DiagnosticSubmissionEnabled)
	require.True(t, *loaded.IsCloudBackupEnabled)
	require.True(t, *loaded.IsDeviceLocatorServiceEnabled)
	require.False(t, *loaded.IsDoNotDisturbInEffect)
	require.False(t, *loaded.IsMDMLostModeEnabled)
	require.False(t, *loaded.IsNetworkTethered)
	require.True(t, *loaded.ITunesStoreAccountIsActive)
	require.False(t, *loaded.PersonalHotspotEnabled)
	require.WithinDuration(t, lastBackup, *loaded.LastCloudBackupDate, time.Second)

	require.NotNil(t, loaded.AccessibilitySettings)
	require.True(t, *loaded.AccessibilitySettings.VoiceOverEnabled)
	require.False(t, *loaded.AccessibilitySettings.GrayscaleEnabled)
	require.NotNil(t, loaded.OrganizationInfo)
	require.Equal(t, "Acme Corp", *loaded.OrganizationInfo.OrganizationName)
	require.NotNil(t, loaded.MDMOptions)
	require.True(t, *loaded.MDMOptions.BootstrapTokenAllowed)
	require.Equal(t, [][]byte{[]byte("leaf-cert"), []byte("intermediate-cert")}, loaded.DevicePropertiesAttestation)

	require.Len(t, loaded.ServiceSubscriptions, 2)
	require.Equal(t, "CTSubscriptionSlotOne", loaded.ServiceSubscriptions[0].Slot)
	require.Equal(t, "iccid-1", *loaded.ServiceSubscriptions[0].ICCID)
	require.Equal(t, "CTSubscriptionSlotTwo", loaded.ServiceSubscriptions[1].Slot)
	require.Equal(t, "eid-2", *loaded.ServiceSubscriptions[1].EID)
}

func testLoadHostMDMAppleDeviceVitalsDBPartial(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "vitals-partial-host", "1.1.1.5", "vitals-partial-host-key", "vitals-partial-host-uuid", time.Now(), test.WithPlatform("ipados"))

	// Simulates an enrollment method that doesn't support every key: only a
	// subset of fields present in the ack, the rest absent from the table row.
	vitals := fleet.MDMAppleDeviceVitals{
		UDID:         new("00008030-CCC"),
		BatteryLevel: new(0.5),
	}
	require.NoError(t, ds.SetOrUpdateHostMDMAppleDeviceVitals(ctx, host.UUID, vitals))

	loaded := fleet.Host{UUID: host.UUID}
	require.NoError(t, ds.LoadHostMDMAppleDeviceVitals(ctx, &loaded))

	require.Equal(t, "00008030-CCC", *loaded.UDID)
	require.InDelta(t, 0.5, *loaded.BatteryLevel, 0.001)
	require.Nil(t, loaded.ModelNumber)
	require.Nil(t, loaded.AccessibilitySettings)
	require.Nil(t, loaded.OrganizationInfo)
	require.Nil(t, loaded.MDMOptions)
	require.Nil(t, loaded.DevicePropertiesAttestation)
	require.Empty(t, loaded.ServiceSubscriptions)
}

func testLoadHostMDMAppleDeviceVitalsDBNoRow(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "vitals-no-row-host", "1.1.1.6", "vitals-no-row-host-key", "vitals-no-row-host-uuid", time.Now(), test.WithPlatform("ios"))

	// No refetch has happened yet since this shipped: no row at all in
	// host_mdm_apple_device_vitals for this host.
	loaded := fleet.Host{UUID: host.UUID}
	require.NoError(t, ds.LoadHostMDMAppleDeviceVitals(ctx, &loaded))

	require.Nil(t, loaded.UDID)
	require.Nil(t, loaded.BatteryLevel)
	require.Nil(t, loaded.AccessibilitySettings)
	require.Empty(t, loaded.ServiceSubscriptions)
}
