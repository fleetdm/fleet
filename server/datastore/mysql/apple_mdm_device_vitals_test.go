package mysql

import (
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
	}
	require.NoError(t, ds.SetOrUpdateHostMDMAppleDeviceVitals(ctx, host.UUID, vitals))

	var row struct {
		UDID         *string  `db:"udid"`
		BatteryLevel *float64 `db:"battery_level"`
	}
	require.NoError(t, ds.writer(ctx).Get(&row, `SELECT udid, battery_level FROM host_mdm_apple_device_vitals WHERE host_uuid = ?`, host.UUID))
	require.Equal(t, "00008030-AAA", *row.UDID)
	require.InDelta(t, 0.75, *row.BatteryLevel, 0.001)

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
