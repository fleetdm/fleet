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

	hosts, _, _, _, _, err := ds.GetAppleProfileReconcileSnapshot(ctx, "", 100)
	require.NoError(t, err)

	gotUUIDs := make(map[string]struct{}, len(hosts))
	for _, h := range hosts {
		gotUUIDs[h.UUID] = struct{}{}
	}

	require.Contains(t, gotUUIDs, enrolledHost.UUID, "host with host_mdm.enrolled = 1 should be returned")
	require.NotContains(t, gotUUIDs, notEnrolledHost.UUID, "host with host_mdm.enrolled = 0 should not be returned")
}

func TestGetAppleProfileReconcileSnapshotPageFullWithDuplicateUUIDs(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	newHost := func(suffix, uuid string) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   new("osquery-" + suffix),
			NodeKey:         new("nodekey-" + suffix),
			UUID:            uuid,
			Hostname:        "hostname-" + suffix,
			HardwareSerial:  "serial-" + suffix,
			Platform:        "darwin",
		})
		require.NoError(t, err)
		err = ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://example.com", true, fleet.WellKnownMDMFleet, "", false)
		require.NoError(t, err)
		return h
	}

	// Three UUIDs in cursor order; the middle one has two hosts rows — the
	// duplicate-enrollment state DEP re-enrollment can leave behind. The single
	// nano enrollment for the shared UUID joins both rows, so the window query
	// yields four raw rows: uuid-a, uuid-b (x2), uuid-c.
	hA := newHost("a", "uuid-a")
	nanoEnroll(t, ds, hA, false)
	newHost("b1", "uuid-b")
	hB2 := newHost("b2", "uuid-b")
	nanoEnroll(t, ds, hB2, false)
	hC := newHost("c", "uuid-c")
	nanoEnroll(t, ds, hC, false)

	// batchSize 3 fills the raw page with [uuid-a, uuid-b, uuid-b], which
	// dedupes down to two hosts. pageFull must still report true: deciding
	// end-of-universe from the deduped length wraps the cursor early and
	// permanently starves every host past this page (here, uuid-c).
	hosts, _, _, _, pageFull, err := ds.GetAppleProfileReconcileSnapshot(ctx, "", 3)
	require.NoError(t, err)
	require.Len(t, hosts, 2)
	require.Equal(t, "uuid-a", hosts[0].UUID)
	require.Equal(t, "uuid-b", hosts[1].UUID)
	require.Equal(t, hB2.ID, hosts[1].HostID, "dedupe keeps the highest host id")
	require.True(t, pageFull, "raw page hit the SQL limit, more hosts may remain")

	// Resuming from the last deduped host reaches the previously-starved host.
	hosts, _, _, _, pageFull, err = ds.GetAppleProfileReconcileSnapshot(ctx, hosts[len(hosts)-1].UUID, 3)
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	require.Equal(t, hC.UUID, hosts[0].UUID)
	require.False(t, pageFull)

	// The DDM snapshot shares the same host window; verify the same semantics.
	dHosts, _, _, _, dPageFull, err := ds.GetAppleDeclarationReconcileSnapshot(ctx, "", 3)
	require.NoError(t, err)
	require.Len(t, dHosts, 2)
	require.True(t, dPageFull, "raw page hit the SQL limit, more hosts may remain")

	// A raw page made up entirely of one duplicated UUID still makes progress:
	// pageFull is true, and the strict uuid > cursor pagination moves past
	// every row sharing that UUID on the next call.
	hosts, _, _, _, pageFull, err = ds.GetAppleProfileReconcileSnapshot(ctx, "uuid-a", 2)
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	require.Equal(t, "uuid-b", hosts[0].UUID)
	require.True(t, pageFull, "raw page hit the SQL limit, more hosts may remain")
	hosts, _, _, _, _, err = ds.GetAppleProfileReconcileSnapshot(ctx, hosts[0].UUID, 2)
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	require.Equal(t, hC.UUID, hosts[0].UUID)
}

func TestGetAppleMDMHostForReconcileIgnoresHostMDMStatus(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	// Both hosts are darwin and fully nano-enrolled (Device enrollment +
	// nano_devices row), so the only thing that differs between them is their
	// host_mdm.enrolled flag.
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
