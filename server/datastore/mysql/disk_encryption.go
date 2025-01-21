package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/jmoiron/sqlx"
)

func cleanupDiskEncryptionKeysOnTeamChangeDB(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint, newTeamID *uint) error {
	// We are using Apple's encryption profile to determine if any hosts, including Windows and Linux, are encrypted.
	// This is a safe assumption since encryption is enabled for the whole team.
	_, err := getMDMAppleConfigProfileByTeamAndIdentifierDB(ctx, tx, newTeamID, mobileconfig.FleetFileVaultPayloadIdentifier)
	if err != nil {
		if fleet.IsNotFound(err) {
			// the new team does not have a filevault profile so we need to delete the existing ones
			if err := bulkDeleteHostDiskEncryptionKeysDB(ctx, tx, hostIDs); err != nil {
				return ctxerr.Wrap(ctx, err, "reconcile filevault profiles on team change bulk delete host disk encryption keys")
			}
		} else {
			return ctxerr.Wrap(ctx, err, "reconcile filevault profiles on team change get profile")
		}
	}
	return nil
}

func bulkDeleteHostDiskEncryptionKeysDB(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	const moveKeysToArchiveQuery = `
INSERT INTO host_disk_encryption_keys_archive (host_id, base64_encrypted, base64_encrypted_salt, key_slot, decryptable, original_created_at, original_updated_at, reset_requested, client_error)
SELECT host_id, base64_encrypted, base64_encrypted_salt, key_slot, decryptable, created_at as original_created_at, updated_at as original_updated_at, reset_requested, client_error
FROM host_disk_encryption_keys
WHERE host_id IN (?)
`
	moveStmt, moveArgs, err := sqlx.In(moveKeysToArchiveQuery, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building move encryption keys query")
	}
	_, err = tx.ExecContext(ctx, moveStmt, moveArgs...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "moving encryption keys to archive table")
	}

	delteStmt, deleteArgs, err := sqlx.In(
		"DELETE FROM host_disk_encryption_keys WHERE host_id IN (?)",
		hostIDs,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building query")
	}

	_, err = tx.ExecContext(ctx, delteStmt, deleteArgs...)
	return err
}
