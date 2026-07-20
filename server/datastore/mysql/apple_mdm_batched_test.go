package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestGetAppleProfileReconcileSnapshotChecksMDMStatus(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	// Both hosts are darwin and fully nano-enrolled (Device enrollment +
	// nano_devices row), so the only thing that differs between them is their
	// host_mdm.enrolled flag. This isolates the EXISTS(host_mdm ... enrolled = 1)
	// filter in the reconcile query.
	newEnrolledHost := func(suffix string, enrolled bool) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   new("osquery-" + suffix),
			NodeKey:         new("nodekey-" + suffix),
			UUID:            "uuid-" + suffix,
			Hostname:        "hostname-" + suffix,
			HardwareSerial:  "serial-" + suffix,
			Platform:        "darwin",
		})
		require.NoError(t, err)

		nanoEnroll(t, ds, h, false)
		err = ds.SetOrUpdateMDMData(ctx, h.ID, false, enrolled, "https://example.com", true, fleet.WellKnownMDMFleet, "", false)
		require.NoError(t, err)

		return h
	}

	// host_mdm.enrolled = 1 -> should be returned.
	enrolledHost := newEnrolledHost("enrolled", true)
	// host_mdm.enrolled = 0 -> should NOT be returned.
	notEnrolledHost := newEnrolledHost("not-enrolled", false)

	hosts, _, _, _, err := ds.GetAppleProfileReconcileSnapshot(ctx, "", 100)
	require.NoError(t, err)

	gotUUIDs := make(map[string]struct{}, len(hosts))
	for _, h := range hosts {
		gotUUIDs[h.UUID] = struct{}{}
	}

	require.Contains(t, gotUUIDs, enrolledHost.UUID, "host with host_mdm.enrolled = 1 should be returned")
	require.NotContains(t, gotUUIDs, notEnrolledHost.UUID, "host with host_mdm.enrolled = 0 should not be returned")
}

func TestGetAppleMDMHostForReconcileIgnoresHostMDMStatus(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	// Both hosts are darwin and fully nano-enrolled (Device enrollment +
	// nano_devices row), so the only thing that differs between them is their
	// host_mdm.enrolled flag. This isolates the EXISTS(host_mdm ... enrolled = 1)
	// filter in the reconcile query.
	newEnrolledHost := func(suffix string, enrolled bool) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   new("osquery-" + suffix),
			NodeKey:         new("nodekey-" + suffix),
			UUID:            "uuid-" + suffix,
			Hostname:        "hostname-" + suffix,
			HardwareSerial:  "serial-" + suffix,
			Platform:        "darwin",
		})
		require.NoError(t, err)

		nanoEnroll(t, ds, h, false)
		err = ds.SetOrUpdateMDMData(ctx, h.ID, false, enrolled, "https://example.com", true, fleet.WellKnownMDMFleet, "", false)
		require.NoError(t, err)

		return h
	}

	// host_mdm.enrolled = 1 -> should be returned.
	enrolledHost := newEnrolledHost("enrolled", true)

	info, err := ds.GetAppleMDMHostForReconcile(ctx, enrolledHost.UUID)
	require.NoError(t, err)

	require.NotNil(t, info)
	require.Equal(t, enrolledHost.UUID, info.UUID)

	// host_mdm.enrolled = 0 -> should also be returned for enrolling hosts
	notEnrolledHost := newEnrolledHost("not-enrolled", false)
	info, err = ds.GetAppleMDMHostForReconcile(ctx, notEnrolledHost.UUID)
	require.NoError(t, err)

	require.NotNil(t, info)
	require.Equal(t, notEnrolledHost.UUID, info.UUID)
}
