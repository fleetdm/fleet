package mysql

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// Test-only Windows profile eager-reconciliation helpers. Production
// reconciles via the mdm_windows_profile_manager cron; tests that assert
// on host_mdm_windows_profiles immediately after BulkSetPendingMDMHostProfiles
// must opt in by calling ds.EnableTestWindowsEagerHook(t).

// EnableTestWindowsEagerHook makes BulkSetPendingMDMHostProfiles reconcile
// Windows rows synchronously for the rest of the test. The prior hook is
// restored via t.Cleanup so sibling tests on a shared datastore (e.g.
// TestMDMShared) aren't affected.
func (ds *Datastore) EnableTestWindowsEagerHook(tb testing.TB) {
	tb.Helper()
	prev := ds.testWindowsEagerHook
	ds.testWindowsEagerHook = ds.bulkSetPendingMDMWindowsHostProfilesForTests
	tb.Cleanup(func() { ds.testWindowsEagerHook = prev })
}

// bulkSetPendingMDMWindowsHostProfilesForTests reconciles Windows profile
// state for the given hosts by marking obsolete profiles for removal and
// upserting desired profiles. Each batch is committed in its own
// transaction so InnoDB row locks on host_mdm_windows_profiles are held
// only for the duration of one batch.
func (ds *Datastore) bulkSetPendingMDMWindowsHostProfilesForTests(
	ctx context.Context,
	hostUUIDs []string,
	onlyProfileUUIDs []string,
) (updatedDB bool, err error) {
	if len(hostUUIDs) == 0 {
		return false, nil
	}

	// The pre-scan listings are read-only and run outside any transaction;
	// they use the writer (same node that handled the prior tx) so they
	// observe their own committed writes without replica lag.
	profilesToInstall, err := ds.listMDMWindowsProfilesToInstallDB(ctx, ds.writer(ctx), hostUUIDs, onlyProfileUUIDs)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "list profiles to install")
	}

	profilesToRemove, err := ds.listMDMWindowsProfilesToRemoveDB(ctx, ds.writer(ctx), hostUUIDs, onlyProfileUUIDs)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "list profiles to remove")
	}

	if len(profilesToInstall) == 0 && len(profilesToRemove) == 0 {
		return false, nil
	}

	// Sort by (HostUUID, ProfileUUID) to match the host_mdm_windows_profiles
	// PRIMARY KEY (host_uuid, profile_uuid) so concurrent callers acquire
	// InnoDB row locks in a consistent order, reducing deadlock risk.
	cmpByHostThenProfile := func(a, b *fleet.MDMWindowsProfilePayload) int {
		if c := cmp.Compare(a.HostUUID, b.HostUUID); c != 0 {
			return c
		}
		return cmp.Compare(a.ProfileUUID, b.ProfileUUID)
	}
	slices.SortFunc(profilesToRemove, cmpByHostThenProfile)
	slices.SortFunc(profilesToInstall, cmpByHostThenProfile)

	const defaultBatchSize = 2000
	removeBatchSize := defaultBatchSize
	if ds.testDeleteMDMProfilesBatchSize > 0 {
		removeBatchSize = ds.testDeleteMDMProfilesBatchSize
	}
	installBatchSize := defaultBatchSize
	if ds.testUpsertMDMDesiredProfilesBatchSize > 0 {
		installBatchSize = ds.testUpsertMDMDesiredProfilesBatchSize
	}

	if len(profilesToRemove) > 0 {
		// Mark profiles for removal instead of deleting them. The reconciler
		// will pick these up (status=NULL, operation_type='remove') and
		// generate <Delete> SyncML commands.
		//
		// Tuple order `(host_uuid, profile_uuid)` matches the PK so MySQL can
		// perform direct PK point lookups for each pair.
		err := common_mysql.BatchProcessSimple(profilesToRemove, removeBatchSize, func(batch []*fleet.MDMWindowsProfilePayload) error {
			return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
				var sb strings.Builder
				sb.WriteString(`UPDATE host_mdm_windows_profiles
					SET operation_type = ?, status = NULL, command_uuid = '', detail = ''
					WHERE (host_uuid, profile_uuid) IN (`)
				args := make([]any, 0, 1+len(batch)*2)
				args = append(args, fleet.MDMOperationTypeRemove)
				for j, p := range batch {
					if j > 0 {
						sb.WriteByte(',')
					}
					sb.WriteString("(?,?)")
					args = append(args, p.HostUUID, p.ProfileUUID)
				}
				sb.WriteByte(')')
				// Use RowsAffected to keep Apple-parity: idempotent re-applies
				// (rows already at op=remove,status=NULL) flip nothing, so
				// updatedDB must stay false in that case. MySQL's
				// CLIENT_FOUND_ROWS is not enabled for our connections, so
				// RowsAffected reflects rows actually changed.
				res, err := tx.ExecContext(ctx, sb.String(), args...)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "marking profiles for removal")
				}
				if rows, _ := res.RowsAffected(); rows > 0 {
					updatedDB = true
				}
				return nil
			})
		})
		if err != nil {
			return updatedDB, err
		}
	}

	if len(profilesToInstall) == 0 {
		return updatedDB, nil
	}

	err = common_mysql.BatchProcessSimple(profilesToInstall, installBatchSize, func(batch []*fleet.MDMWindowsProfilePayload) error {
		return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			didUpdate, execErr := executeWindowsProfileUpsertBatchForTests(ctx, tx, batch)
			if execErr != nil {
				return execErr
			}
			if didUpdate {
				updatedDB = true
			}
			return nil
		})
	})
	if err != nil {
		return updatedDB, err
	}

	return updatedDB, nil
}

// executeWindowsProfileUpsertBatchForTests performs the pre-read
// skip-if-unchanged check and the ON DUPLICATE KEY UPDATE upsert for a
// single batch of Windows profile installs. Returns true if the batch
// resulted in a DB write.
func executeWindowsProfileUpsertBatchForTests(
	ctx context.Context,
	tx sqlx.ExtContext,
	batch []*fleet.MDMWindowsProfilePayload,
) (bool, error) {
	profilesToInsert := make(map[string]*fleet.MDMWindowsProfilePayload, len(batch))
	var psb strings.Builder
	pargs := make([]any, 0, len(batch)*6)
	for _, p := range batch {
		// Build the desired post-upsert payload used for the skip-if-unchanged
		// comparison below. The payload returned by
		// listMDMWindowsProfilesToInstallDB only populates a subset of fields,
		// so we construct a full payload here matching the values the ON
		// DUPLICATE KEY UPDATE clause will set.
		desired := &fleet.MDMWindowsProfilePayload{
			ProfileUUID:      p.ProfileUUID,
			ProfileName:      p.ProfileName,
			HostUUID:         p.HostUUID,
			Status:           nil,
			OperationType:    fleet.MDMOperationTypeInstall,
			Detail:           p.Detail,
			CommandUUID:      p.CommandUUID,
			Retries:          p.Retries,
			Checksum:         p.Checksum,
			SecretsUpdatedAt: p.SecretsUpdatedAt,
		}
		profilesToInsert[fmt.Sprintf("%s\n%s", p.ProfileUUID, p.HostUUID)] = desired
		pargs = append(pargs, p.ProfileUUID, p.HostUUID, p.ProfileName,
			fleet.MDMOperationTypeInstall, p.Checksum, p.SecretsUpdatedAt)
		psb.WriteString("(?, ?, ?, ?, NULL, '', ?, ?),")
	}

	// Tuple order `(host_uuid, profile_uuid)` matches the PK for direct point
	// lookups.
	selectStmt := fmt.Sprintf(`
		SELECT
			profile_uuid,
			host_uuid,
			status,
			checksum,
			secrets_updated_at,
			COALESCE(operation_type, '') AS operation_type,
			COALESCE(detail, '') AS detail,
			COALESCE(command_uuid, '') AS command_uuid,
			COALESCE(profile_name, '') AS profile_name
		FROM host_mdm_windows_profiles WHERE (host_uuid, profile_uuid) IN (%s)`,
		strings.TrimSuffix(strings.Repeat("(?,?),", len(batch)), ","))
	selectArgs := make([]any, 0, 2*len(batch))
	for _, p := range batch {
		selectArgs = append(selectArgs, p.HostUUID, p.ProfileUUID)
	}
	var existingProfiles []fleet.MDMWindowsProfilePayload
	if err := sqlx.SelectContext(ctx, tx, &existingProfiles, selectStmt, selectArgs...); err != nil {
		return false, ctxerr.Wrap(ctx, err, "bulk set pending profile status select existing")
	}
	var updateNeeded bool
	if len(existingProfiles) == len(profilesToInsert) {
		for _, exist := range existingProfiles {
			insert, ok := profilesToInsert[fmt.Sprintf("%s\n%s", exist.ProfileUUID, exist.HostUUID)]
			if !ok || !exist.Equal(*insert) {
				updateNeeded = true
				break
			}
		}
	} else {
		updateNeeded = true
	}
	if !updateNeeded {
		return false, nil
	}

	baseStmt := fmt.Sprintf(`
			INSERT INTO host_mdm_windows_profiles (
				profile_uuid,
				host_uuid,
				profile_name,
				operation_type,
				status,
				command_uuid,
				checksum,
				secrets_updated_at
			)
			VALUES %s
			ON DUPLICATE KEY UPDATE
				profile_name = VALUES(profile_name),
				operation_type = VALUES(operation_type),
				status = NULL,
				command_uuid = VALUES(command_uuid),
				detail = '',
				checksum = VALUES(checksum),
				secrets_updated_at = VALUES(secrets_updated_at)
		`, strings.TrimSuffix(psb.String(), ","))

	if _, err := tx.ExecContext(ctx, baseStmt, pargs...); err != nil {
		return false, ctxerr.Wrap(ctx, err, "bulk set pending profile status execute batch")
	}
	return true, nil
}
