package mysql

import (
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testWindowsMDMUUIDOwners(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	newHost := func(hostUUID string, opts ...test.NewHostOption) *fleet.Host {
		name := "uuid-owner-" + uuid.NewString()
		return test.NewHost(t, ds, name, "10.0.0.1", name, hostUUID, time.Now(), append(opts, test.WithPlatform("windows"))...)
	}
	ownerOf := func(t *testing.T, hostUUID string) uint {
		ownerID, err := ds.GetWindowsMDMUUIDOwner(ctx, hostUUID)
		require.NoError(t, err)
		return ownerID
	}
	link := func(h *fleet.Host, device *fleet.MDMWindowsEnrolledDevice) (bool, error) {
		return ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, h.ID, h.UUID, device.MDMDeviceID)
	}
	requireOwnedByOther := func(t *testing.T, err error, owner *fleet.Host) {
		ownedErr, ok := errors.AsType[*fleet.WindowsMDMUUIDOwnedByOtherHostError](err)
		require.True(t, ok, "got %v", err)
		assert.Equal(t, owner.ID, ownedErr.OwnerHostID)
	}
	loadDevice := func(t *testing.T, device *fleet.MDMWindowsEnrolledDevice) *fleet.MDMWindowsEnrolledDevice {
		loaded, err := ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, device.MDMDeviceID)
		require.NoError(t, err)
		return loaded
	}
	setMDMEnrolled := func(t *testing.T, h *fleet.Host) {
		require.NoError(t, ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://example.com", false, fleet.WellKnownMDMFleet, "",
			fleet.PersonalEnrollmentTypeNone))
	}

	t.Run("a link claims an unowned UUID and the owner keeps it", func(t *testing.T) {
		sharedUUID := uuid.NewString()
		first, second := newHost(sharedUUID), newHost(sharedUUID)
		device := insertWindowsEnrollment(t, ds, uuid.NewString(), "")

		updated, err := link(first, device)
		require.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, first.ID, ownerOf(t, sharedUUID))

		_, err = link(second, device)
		requireOwnedByOther(t, err, first)
		assert.Equal(t, first.ID, ownerOf(t, sharedUUID))

		updated, err = link(first, device)
		require.NoError(t, err)
		assert.False(t, updated)

		// Other hardware reporting the same UUID, such as a cloned VM, is not linked.
		clone := insertWindowsEnrollment(t, ds, uuid.NewString(), "")
		_, err = link(second, clone)
		requireOwnedByOther(t, err, first)
		assert.Empty(t, loadDevice(t, clone).HostUUID)
	})

	t.Run("a fleetd enrollment claims for the enrolling host", func(t *testing.T) {
		sharedUUID := uuid.NewString()
		first, second := newHost(sharedUUID), newHost(sharedUUID)
		enrollment := func(h *fleet.Host) *fleet.MDMWindowsEnrolledDevice {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:            uuid.NewString(),
				MDMHardwareID:          uuid.NewString(),
				MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
				MDMDeviceType:          "CIMClient_Windows",
				MDMDeviceName:          "DESKTOP",
				MDMEnrollType:          "ProgrammaticEnrollment",
				MDMEnrollProtoVersion:  "5.0",
				MDMEnrollClientVersion: "10.0",
				HostUUID:               h.UUID,
				LinkedHostID:           &h.ID,
			}
		}

		require.NoError(t, ds.MDMWindowsInsertEnrolledDevice(ctx, enrollment(first)))
		assert.Equal(t, first.ID, ownerOf(t, sharedUUID))

		refused := enrollment(second)
		requireOwnedByOther(t, ds.MDMWindowsInsertEnrolledDevice(ctx, refused), first)
		_, err := ds.MDMWindowsGetEnrolledDeviceWithHardwareID(ctx, refused.MDMHardwareID)
		require.True(t, fleet.IsNotFound(err), "the refused enrollment is not stored")
	})

	t.Run("only an owner whose host is gone is replaced", func(t *testing.T) {
		// Losing its enrollment, as between the two steps of a re-enrollment, doesn't end ownership.
		unenrolledUUID := uuid.NewString()
		unenrolled, next := newHost(unenrolledUUID), newHost(unenrolledUUID)
		device := insertWindowsEnrollment(t, ds, uuid.NewString(), "")
		_, err := link(unenrolled, device)
		require.NoError(t, err)
		require.NoError(t, ds.MDMWindowsDeleteEnrolledDeviceWithDeviceID(ctx, device.MDMDeviceID))
		assert.Equal(t, unenrolled.ID, ownerOf(t, unenrolledUUID))
		_, err = link(next, insertWindowsEnrollment(t, ds, uuid.NewString(), ""))
		requireOwnedByOther(t, err, unenrolled)
		reenrolled := insertWindowsEnrollment(t, ds, uuid.NewString(), "")
		updated, err := link(unenrolled, reenrolled)
		require.NoError(t, err)
		assert.True(t, updated)

		// A device ID that matches no enrollment claims nothing.
		unclaimedUUID := uuid.NewString()
		unclaimed := newHost(unclaimedUUID)
		updated, err = ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, unclaimed.ID, unclaimedUUID, uuid.NewString())
		require.NoError(t, err)
		assert.False(t, updated)
		assert.Zero(t, ownerOf(t, unclaimedUUID))

		// Its host row is gone without deleteHosts.
		goneUUID := uuid.NewString()
		gone, replacement := newHost(goneUUID), newHost(goneUUID)
		device = insertWindowsEnrollment(t, ds, uuid.NewString(), "")
		_, err = link(gone, device)
		require.NoError(t, err)
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM hosts WHERE id = ?`, gone.ID)
			return err
		})
		assert.Zero(t, ownerOf(t, goneUUID))
		_, err = link(replacement, device)
		require.NoError(t, err)
		assert.Equal(t, replacement.ID, ownerOf(t, goneUUID))
	})

	t.Run("a host releases the UUID it no longer has", func(t *testing.T) {
		oldUUID, newUUID := uuid.NewString(), uuid.NewString()
		h := newHost(oldUUID)
		_, err := link(h, insertWindowsEnrollment(t, ds, uuid.NewString(), ""))
		require.NoError(t, err)
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE hosts SET uuid = ? WHERE id = ?`, newUUID, h.ID)
			return err
		})
		h.UUID = newUUID
		_, err = link(h, insertWindowsEnrollment(t, ds, uuid.NewString(), ""))
		require.NoError(t, err)
		assert.Equal(t, h.ID, ownerOf(t, newUUID))
		var oldOwners int
		require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &oldOwners,
			`SELECT COUNT(*) FROM host_mdm_uuid_owners WHERE host_uuid = ?`, oldUUID))
		assert.Zero(t, oldOwners)
	})

	t.Run("only the owner is the device's MDM host", func(t *testing.T) {
		teamA, err := ds.NewTeam(ctx, &fleet.Team{Name: "uuid-owner-a-" + uuid.NewString()})
		require.NoError(t, err)
		teamB, err := ds.NewTeam(ctx, &fleet.Team{Name: "uuid-owner-b-" + uuid.NewString()})
		require.NoError(t, err)
		sharedUUID := uuid.NewString()
		// The stale host has the lower id, which every lookup preferred before.
		stale := newHost(sharedUUID, test.WithTeamID(teamB.ID))
		setMDMEnrolled(t, stale)
		owner := newHost(sharedUUID, test.WithTeamID(teamA.ID))
		deviceID := windowsEnroll(t, ds, owner)

		for _, tc := range []struct {
			host      *fleet.Host
			connected bool
		}{{owner, true}, {stale, false}} {
			connected, err := ds.IsHostConnectedToFleetMDM(ctx, tc.host)
			require.NoError(t, err)
			assert.Equal(t, tc.connected, connected, "host %d", tc.host.ID)
			hostMDM, err := ds.GetHostMDM(ctx, tc.host.ID)
			require.NoError(t, err)
			assert.Equal(t, tc.connected, hostMDM.ConnectedToFleet, "host %d", tc.host.ID)
		}
		connected, err := ds.AreHostsConnectedToFleetMDM(ctx, []*fleet.Host{stale})
		require.NoError(t, err)
		assert.True(t, connected[sharedUUID], "the UUID's owner is connected")

		listed, err := ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{ConnectedToFleetFilter: new(true)})
		require.NoError(t, err)
		listedIDs := make([]uint, 0, len(listed))
		for _, h := range listed {
			listedIDs = append(listedIDs, h.ID)
		}
		assert.Contains(t, listedIDs, owner.ID)
		assert.NotContains(t, listedIDs, stale.ID)

		profileUUID := InsertWindowsProfileForTest(t, ds, 0)
		installWindowsProfilesAsVerified(t, ds, []string{sharedUUID}, []string{profileUUID})
		listed, err = ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{
			ProfileUUIDFilter: &profileUUID, ProfileStatusFilter: new(fleet.OSSettingsVerified),
		})
		require.NoError(t, err)
		require.Len(t, listed, 1)
		assert.Equal(t, owner.ID, listed[0].ID)

		lite, err := ds.WindowsHostLiteByUUID(ctx, sharedUUID)
		require.NoError(t, err)
		assert.Equal(t, owner.ID, lite.ID)
		device, err := ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
		require.NoError(t, err)
		require.NotNil(t, device.LinkedHostID)
		assert.Equal(t, owner.ID, *device.LinkedHostID)
	})

	t.Run("deleting a host that isn't the owner keeps the device's MDM state", func(t *testing.T) {
		profileUUID := InsertWindowsProfileForTest(t, ds, 0)
		sharedUUID := uuid.NewString()
		stale := newHost(sharedUUID)
		owner := newHost(sharedUUID)
		deviceID := windowsEnroll(t, ds, owner)
		installWindowsProfilesAsVerified(t, ds, []string{sharedUUID}, []string{profileUUID})
		profileRows := func(t *testing.T) int {
			profiles, err := ds.GetHostMDMWindowsProfiles(ctx, sharedUUID)
			require.NoError(t, err)
			return len(profiles)
		}

		require.NoError(t, ds.DeleteHost(ctx, stale.ID))
		assert.Equal(t, 1, profileRows(t))
		device, err := ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
		require.NoError(t, err)
		assert.Nil(t, device.DeletedHostTeamID)
		assert.Equal(t, owner.ID, ownerOf(t, sharedUUID))

		// Nor does deleting it along with a host of another platform.
		stale = newHost(sharedUUID)
		mac := test.NewHost(t, ds, "uuid-owner-mac-"+uuid.NewString(), "10.0.0.2", "mac-"+uuid.NewString(), sharedUUID, time.Now(),
			test.WithPlatform("darwin"))
		require.NoError(t, ds.DeleteHosts(ctx, []uint{stale.ID, mac.ID}))
		device, err = ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
		require.NoError(t, err)
		assert.Nil(t, device.DeletedHostTeamID)
		assert.Equal(t, owner.ID, ownerOf(t, sharedUUID))

		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "uuid-owner-later-" + uuid.NewString()})
		require.NoError(t, err)
		next := newHost(sharedUUID)
		later := newHost(sharedUUID, test.WithTeamID(team.ID))
		require.NoError(t, ds.DeleteHost(ctx, owner.ID))
		assert.Zero(t, profileRows(t))
		device, err = ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
		require.NoError(t, err)
		require.NotNil(t, device.DeletedHostTeamID)
		assert.Zero(t, *device.DeletedHostTeamID)
		assert.Zero(t, ownerOf(t, sharedUUID))

		// A host deleted after the owner keeps the fleet the owner's deletion recorded.
		require.NoError(t, ds.DeleteHost(ctx, later.ID))
		device, err = ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
		require.NoError(t, err)
		require.NotNil(t, device.DeletedHostTeamID)
		assert.Zero(t, *device.DeletedHostTeamID)

		// Deleting the owner is how an admin hands the device to the host that remains.
		updated, err := ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, next.ID, next.UUID, deviceID)
		require.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, next.ID, ownerOf(t, sharedUUID))
	})
}
