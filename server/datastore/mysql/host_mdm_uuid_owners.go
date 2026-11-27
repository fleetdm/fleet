package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// Hosts can share a hardware UUID, but MDM state is keyed by UUID, so host_mdm_uuid_owners names the one host per MDM platform
// that owns it.

const (
	windowsMDMOwnerClaimedByEnrollment = "windows_enrollment"
	windowsMDMOwnerClaimedByLink       = "windows_link"
)

// hostIsWindowsMDMOwner is true when the hosts row aliased h owns its UUID's Windows MDM state.
const hostIsWindowsMDMOwner = `EXISTS (
	SELECT 1 FROM host_mdm_uuid_owners hmuo
	WHERE hmuo.host_id = h.id AND hmuo.host_uuid = h.uuid AND hmuo.mdm_platform = 'windows'
)`

// windowsHostIDOfUUIDOwnerFirst selects the host with the UUID bound to its placeholder: its Windows MDM owner, else the lowest id.
const windowsHostIDOfUUIDOwnerFirst = `SELECT h.id FROM hosts h WHERE h.uuid = ? ORDER BY ` + hostIsWindowsMDMOwner + ` DESC, h.id LIMIT 1`

// windowsMDMOwnerIsLive is true while the owner row aliased hmuo's host exists with that UUID. Losing its enrollment, as on
// re-enrollment, doesn't end ownership: only deleting the host hands the device to another.
const windowsMDMOwnerIsLive = `EXISTS (SELECT 1 FROM hosts ho WHERE ho.id = hmuo.host_id AND ho.uuid = hmuo.host_uuid)`

// claimWindowsMDMUUIDOwnerDB makes hostID the owner of hostUUID's Windows MDM state, or returns a
// *fleet.WindowsMDMUUIDOwnedByOtherHostError if another host is its live owner: what a host reports about its device can be faked,
// so ownership never moves on its own.
func claimWindowsMDMUUIDOwnerDB(ctx context.Context, tx sqlx.ExtContext, hostUUID string, hostID uint, claimedBy string) error {
	if hostUUID == "" || hostID == 0 {
		return nil
	}

	var owner struct {
		HostID uint `db:"host_id"`
		Live   bool `db:"live"`
	}
	err := sqlx.GetContext(ctx, tx, &owner, `
		SELECT hmuo.host_id, `+windowsMDMOwnerIsLive+` AS live
		FROM host_mdm_uuid_owners hmuo
		WHERE hmuo.host_uuid = ? AND hmuo.mdm_platform = 'windows'
		FOR UPDATE`, hostUUID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return ctxerr.Wrap(ctx, err, "load windows mdm owner of uuid")
	case owner.HostID == hostID:
		return nil
	case owner.Live:
		return ctxerr.Wrap(ctx, &fleet.WindowsMDMUUIDOwnedByOtherHostError{HostUUID: hostUUID, OwnerHostID: owner.HostID},
			"claim windows mdm owner of uuid")
	}

	// A host owns one UUID, so drop its claim on any other (its UUID changed before UUIDs became immutable).
	if _, err := tx.ExecContext(ctx, `DELETE FROM host_mdm_uuid_owners WHERE host_id = ?`, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "release previous windows mdm owner claim of host")
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO host_mdm_uuid_owners (host_uuid, mdm_platform, host_id, claimed_by) VALUES (?, 'windows', ?, ?)
		ON DUPLICATE KEY UPDATE host_id = VALUES(host_id), claimed_at = CURRENT_TIMESTAMP(6), claimed_by = VALUES(claimed_by)`,
		hostUUID, hostID, claimedBy); err != nil {
		return ctxerr.Wrap(ctx, err, "claim windows mdm owner of uuid")
	}
	return nil
}

func (ds *Datastore) GetWindowsMDMUUIDOwner(ctx context.Context, hostUUID string) (uint, error) {
	if hostUUID == "" {
		return 0, nil
	}
	var ownerID uint
	err := sqlx.GetContext(ctx, ds.reader(ctx), &ownerID, `
		SELECT hmuo.host_id FROM host_mdm_uuid_owners hmuo
		WHERE hmuo.host_uuid = ? AND hmuo.mdm_platform = 'windows' AND `+windowsMDMOwnerIsLive, hostUUID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return 0, nil
	case err != nil:
		return 0, ctxerr.Wrap(ctx, err, "get windows mdm owner of uuid")
	}
	return ownerID, nil
}
