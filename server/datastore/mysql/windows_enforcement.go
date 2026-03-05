package mysql

import (
	"context"
	"crypto/md5" //nolint:gosec // MD5 used only for checksum comparison, not security
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) BatchSetWindowsEnforcementProfiles(ctx context.Context, teamID *uint, profiles []*fleet.WindowsEnforcementProfile) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		return ds.batchSetWindowsEnforcementProfilesDB(ctx, tx, teamID, profiles)
	})
}

func (ds *Datastore) batchSetWindowsEnforcementProfilesDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	tmID *uint,
	profiles []*fleet.WindowsEnforcementProfile,
) error {
	const loadExistingProfiles = `
SELECT
  name,
  profile_uuid,
  raw_policy
FROM
  windows_enforcement_profiles
WHERE
  team_id = ? AND
  name IN (?)
`

	const deleteProfilesNotInList = `
DELETE FROM
  windows_enforcement_profiles
WHERE
  team_id = ? AND
  name NOT IN (?)
`

	const deleteAllProfilesForTeam = `
DELETE FROM
  windows_enforcement_profiles
WHERE
  team_id = ?
`

	const insertNewOrEditedProfile = `
INSERT INTO
  windows_enforcement_profiles (
    profile_uuid, team_id, name, raw_policy, checksum
  )
VALUES
  ( CONCAT('` + fleet.WindowsEnforcementUUIDPrefix + `', CONVERT(UUID() USING utf8mb4)), ?, ?, ?, UNHEX(MD5(raw_policy)) )
ON DUPLICATE KEY UPDATE
  updated_at = IF(raw_policy = VALUES(raw_policy) AND name = VALUES(name), updated_at, CURRENT_TIMESTAMP()),
  name = VALUES(name),
  raw_policy = VALUES(raw_policy),
  checksum = UNHEX(MD5(VALUES(raw_policy)))
`

	// use a profile team id of 0 if no-team
	var profTeamID uint
	if tmID != nil {
		profTeamID = *tmID
	}

	// build a list of names for the incoming profiles
	incomingNames := make([]string, len(profiles))
	incomingProfs := make(map[string]*fleet.WindowsEnforcementProfile, len(profiles))
	for i, p := range profiles {
		incomingNames[i] = p.Name
		incomingProfs[p.Name] = p
	}

	// delete profiles that are not in the incoming set
	if len(incomingNames) == 0 {
		if _, err := tx.ExecContext(ctx, deleteAllProfilesForTeam, profTeamID); err != nil {
			return ctxerr.Wrap(ctx, err, "delete all windows enforcement profiles for team")
		}
	} else {
		delStmt, delArgs, err := sqlx.In(deleteProfilesNotInList, profTeamID, incomingNames)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build delete windows enforcement profiles NOT IN")
		}
		if _, err := tx.ExecContext(ctx, delStmt, delArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete windows enforcement profiles NOT IN")
		}
	}

	// load existing profiles that match incoming names (to detect changes)
	if len(incomingNames) > 0 {
		var existingProfiles []fleet.WindowsEnforcementProfile
		loadStmt, loadArgs, err := sqlx.In(loadExistingProfiles, profTeamID, incomingNames)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build select existing windows enforcement profiles")
		}
		if err := sqlx.SelectContext(ctx, tx, &existingProfiles, loadStmt, loadArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "select existing windows enforcement profiles")
		}

		// index existing by name to compare
		existingByName := make(map[string]fleet.WindowsEnforcementProfile, len(existingProfiles))
		for _, ep := range existingProfiles {
			existingByName[ep.Name] = ep
		}

		// insert/update each incoming profile
		for _, p := range profiles {
			existing, ok := existingByName[p.Name]
			if ok {
				// check if the content actually changed
				newChecksum := md5.Sum(p.RawPolicy)        //nolint:gosec
				oldChecksum := md5.Sum(existing.RawPolicy) //nolint:gosec
				if newChecksum == oldChecksum {
					continue // no change, skip
				}
			}
			if _, err := tx.ExecContext(ctx, insertNewOrEditedProfile,
				profTeamID, p.Name, p.RawPolicy,
			); err != nil {
				return ctxerr.Wrap(ctx, err, "insert or update windows enforcement profile")
			}
		}
	}

	return nil
}

func (ds *Datastore) ListWindowsEnforcementProfiles(ctx context.Context, teamID *uint) ([]*fleet.WindowsEnforcementProfile, error) {
	var profTeamID uint
	if teamID != nil {
		profTeamID = *teamID
	}

	var profiles []*fleet.WindowsEnforcementProfile
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &profiles, `
SELECT
  profile_uuid,
  team_id,
  name,
  raw_policy,
  checksum,
  created_at,
  updated_at
FROM
  windows_enforcement_profiles
WHERE
  team_id = ?
ORDER BY name
`, profTeamID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list windows enforcement profiles")
	}
	return profiles, nil
}

func (ds *Datastore) GetWindowsEnforcementProfile(ctx context.Context, profileUUID string) (*fleet.WindowsEnforcementProfile, error) {
	var profile fleet.WindowsEnforcementProfile
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &profile, `
SELECT
  profile_uuid,
  team_id,
  name,
  raw_policy,
  checksum,
  created_at,
  updated_at
FROM
  windows_enforcement_profiles
WHERE
  profile_uuid = ?
`, profileUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get windows enforcement profile")
	}
	return &profile, nil
}

func (ds *Datastore) DeleteWindowsEnforcementProfile(ctx context.Context, profileUUID string) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
DELETE FROM windows_enforcement_profiles WHERE profile_uuid = ?
`, profileUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete windows enforcement profile")
	}
	return nil
}

// enforcementDesiredStateQuery returns the desired set of enforcement profiles
// for all Windows hosts, based on the host's team membership. This is
// analogous to the MDM desired state queries in microsoft_mdm.go.
const enforcementDesiredStateQuery = `
SELECT
  wep.profile_uuid,
  h.uuid as host_uuid,
  wep.name,
  wep.checksum
FROM
  windows_enforcement_profiles wep
  JOIN hosts h
    ON h.team_id = wep.team_id OR (h.team_id IS NULL AND wep.team_id = 0)
WHERE
  h.platform = 'windows'
`

func (ds *Datastore) ListWindowsEnforcementToInstall(ctx context.Context) ([]*fleet.HostWindowsEnforcement, error) {
	// Desired state LEFT JOIN current state: profiles in desired but not in
	// current, or in both but with NULL status (pending retry).
	const query = `
SELECT
  ds.profile_uuid,
  ds.host_uuid,
  ds.name
FROM
  (` + enforcementDesiredStateQuery + `) as ds
  LEFT JOIN host_windows_enforcement hwe
    ON hwe.profile_uuid = ds.profile_uuid AND hwe.host_uuid = ds.host_uuid
WHERE
  -- profiles in desired but not in current
  ( hwe.profile_uuid IS NULL AND hwe.host_uuid IS NULL ) OR
  -- profiles in both with operation_type=install and NULL status
  ( hwe.host_uuid IS NOT NULL AND hwe.operation_type = ? AND hwe.status IS NULL )
`

	var payloads []*fleet.HostWindowsEnforcement
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &payloads, query, fleet.MDMOperationTypeInstall); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list windows enforcement to install")
	}
	return payloads, nil
}

func (ds *Datastore) ListWindowsEnforcementToRemove(ctx context.Context) ([]*fleet.HostWindowsEnforcement, error) {
	// Current state RIGHT JOIN desired state: profiles in current but not in
	// desired.
	const query = `
SELECT
  hwe.profile_uuid,
  hwe.host_uuid,
  hwe.status,
  hwe.operation_type,
  COALESCE(hwe.detail, '') as detail
FROM
  (` + enforcementDesiredStateQuery + `) as ds
  RIGHT JOIN host_windows_enforcement hwe
    ON hwe.profile_uuid = ds.profile_uuid AND hwe.host_uuid = ds.host_uuid
WHERE
  ds.profile_uuid IS NULL AND ds.host_uuid IS NULL
`

	var payloads []*fleet.HostWindowsEnforcement
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &payloads, query); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list windows enforcement to remove")
	}
	return payloads, nil
}

func (ds *Datastore) BulkUpsertHostWindowsEnforcement(ctx context.Context, payload []*fleet.HostWindowsEnforcement) error {
	if len(payload) == 0 {
		return nil
	}

	const batchSize = 1000
	const baseStmt = `
INSERT INTO host_windows_enforcement (
  host_uuid,
  profile_uuid,
  status,
  operation_type,
  detail,
  retries
)
VALUES
`
	const onDup = `
ON DUPLICATE KEY UPDATE
  status = VALUES(status),
  operation_type = VALUES(operation_type),
  detail = VALUES(detail),
  retries = VALUES(retries)
`

	batchCount := 0
	args := make([]any, 0, batchSize*6)
	sb := strings.Builder{}
	sb.WriteString(baseStmt)

	execBatch := func() error {
		stmt := strings.TrimSuffix(sb.String(), ",") + onDup
		if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "bulk upsert host windows enforcement")
		}
		// reset
		sb.Reset()
		sb.WriteString(baseStmt)
		args = args[:0]
		batchCount = 0
		return nil
	}

	for _, p := range payload {
		args = append(args, p.HostUUID, p.ProfileUUID, p.Status, p.OperationType, p.Detail, p.Retries)
		sb.WriteString("(?, ?, ?, ?, ?, ?),")
		batchCount++

		if batchCount >= batchSize {
			if err := execBatch(); err != nil {
				return err
			}
		}
	}

	if batchCount > 0 {
		if err := execBatch(); err != nil {
			return err
		}
	}

	return nil
}

func (ds *Datastore) GetHostWindowsEnforcement(ctx context.Context, hostUUID string) ([]fleet.HostWindowsEnforcement, error) {
	var results []fleet.HostWindowsEnforcement
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, `
SELECT
  hwe.host_uuid,
  hwe.profile_uuid,
  COALESCE(wep.name, '') as name,
  hwe.status,
  hwe.operation_type,
  COALESCE(hwe.detail, '') as detail,
  hwe.retries
FROM
  host_windows_enforcement hwe
  LEFT JOIN windows_enforcement_profiles wep
    ON wep.profile_uuid = hwe.profile_uuid
WHERE
  hwe.host_uuid = ?
`, hostUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host windows enforcement")
	}
	return results, nil
}

func (ds *Datastore) BulkSetPendingWindowsEnforcementForHosts(ctx context.Context, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	// Get the host UUIDs for the given host IDs
	query := `
SELECT uuid FROM hosts WHERE id IN (?)
`
	stmt, args, err := sqlx.In(query, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build select host uuids for enforcement")
	}

	var hostUUIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostUUIDs, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "select host uuids for enforcement")
	}

	if len(hostUUIDs) == 0 {
		return nil
	}

	// For each host, get the desired profiles and upsert them as pending
	const desiredQuery = `
SELECT
  wep.profile_uuid,
  h.uuid as host_uuid,
  wep.name
FROM
  windows_enforcement_profiles wep
  JOIN hosts h
    ON h.team_id = wep.team_id OR (h.team_id IS NULL AND wep.team_id = 0)
WHERE
  h.uuid IN (?) AND h.platform = 'windows'
`

	stmt, args, err = sqlx.In(desiredQuery, hostUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build desired enforcement query")
	}

	type desiredRow struct {
		ProfileUUID string `db:"profile_uuid"`
		HostUUID    string `db:"host_uuid"`
		Name        string `db:"name"`
	}
	var desired []desiredRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &desired, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "select desired enforcement profiles for hosts")
	}

	if len(desired) == 0 {
		return nil
	}

	// Build the bulk upsert payload
	pending := fleet.MDMDeliveryPending
	payload := make([]*fleet.HostWindowsEnforcement, 0, len(desired))
	for _, d := range desired {
		payload = append(payload, &fleet.HostWindowsEnforcement{
			HostUUID:      d.HostUUID,
			ProfileUUID:   d.ProfileUUID,
			Name:          d.Name,
			Status:        &pending,
			OperationType: fleet.MDMOperationTypeInstall,
		})
	}

	return ds.BulkUpsertHostWindowsEnforcement(ctx, payload)
}

// GetHostWindowsEnforcementHash returns a hash representing the current
// enforcement profile set for a host. Used to detect changes.
func (ds *Datastore) GetHostWindowsEnforcementHash(ctx context.Context, hostUUID string) (string, error) {
	var hash string
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &hash, `
SELECT
  COALESCE(
    MD5(GROUP_CONCAT(wep.profile_uuid ORDER BY wep.profile_uuid SEPARATOR ',')),
    ''
  )
FROM
  windows_enforcement_profiles wep
  JOIN hosts h
    ON h.team_id = wep.team_id OR (h.team_id IS NULL AND wep.team_id = 0)
WHERE
  h.uuid = ? AND h.platform = 'windows'
`, hostUUID); err != nil {
		return "", ctxerr.Wrap(ctx, err, "get host windows enforcement hash")
	}
	return hash, nil
}

// GetPendingWindowsEnforcementForHost returns the enforcement policies that
// should be delivered to the host via OrbitConfig.
func (ds *Datastore) GetPendingWindowsEnforcementForHost(ctx context.Context, hostUUID string) ([]fleet.OrbitEnforcementPolicy, error) {
	var policies []fleet.OrbitEnforcementPolicy
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &policies, `
SELECT
  wep.profile_uuid,
  wep.name,
  wep.raw_policy
FROM
  windows_enforcement_profiles wep
  JOIN hosts h
    ON h.team_id = wep.team_id OR (h.team_id IS NULL AND wep.team_id = 0)
WHERE
  h.uuid = ? AND h.platform = 'windows'
ORDER BY wep.name
`, hostUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get pending windows enforcement for host")
	}
	return policies, nil
}
