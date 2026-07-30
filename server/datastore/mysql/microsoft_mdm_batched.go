package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// listWindowsMDMHostsForReconcileBatchTransaction returns up to batchSize Windows-MDM-enrolled hosts with uuid > afterHostUUID,
// ordered ascending by uuid, along with the fields the batched reconciler needs to compute the desired state in memory.
//
// platform 'windows', an mdm_windows_enrollments row, and a host_mdm row with enrolled = 1. The two enrollment relationships are
// expressed as EXISTS subqueries rather than JOINs so a host with more than one mdm_windows_enrollments row (the table has no
// uniqueness on host_uuid) yields exactly one host record here.
func (ds *Datastore) listWindowsMDMHostsForReconcileBatchTransaction(
	ctx context.Context,
	tx common_mysql.DBReadTx,
	afterHostUUID string,
	batchSize int,
) ([]*fleet.WindowsHostReconcileInfo, error) {
	const stmt = `
		SELECT
			h.id               AS id,
			h.uuid             AS uuid,
			h.team_id          AS team_id,
			h.label_updated_at AS label_updated_at
		FROM hosts h
		WHERE
			h.platform = 'windows'
			AND h.uuid > ?
			AND EXISTS (
				SELECT 1 FROM mdm_windows_enrollments mwe WHERE mwe.host_uuid = h.uuid
			)
			AND EXISTS (
				SELECT 1 FROM host_mdm hmdm WHERE hmdm.host_id = h.id AND hmdm.enrolled = 1
			)
		ORDER BY h.uuid
		LIMIT ?
	`

	var hosts []*fleet.WindowsHostReconcileInfo
	if err := sqlx.SelectContext(ctx, tx, &hosts, stmt, afterHostUUID, batchSize); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list windows mdm hosts for reconcile batch")
	}
	return hosts, nil
}

// GetWindowsMDMHostForReconcile returns reconcile info for an eligible MDM-enrolled Windows host, using the same eligibility
// rules as the batched reconciler's host listing (platform 'windows', an mdm_windows_enrollments row, host_mdm.enrolled = 1).
// Returns (nil, nil) when the host doesn't exist or isn't eligible.
func (ds *Datastore) GetWindowsMDMHostForReconcile(ctx context.Context, hostUUID string) (*fleet.WindowsHostReconcileInfo, error) {
	const stmt = `
		SELECT
			h.id               AS id,
			h.uuid             AS uuid,
			h.team_id          AS team_id,
			h.label_updated_at AS label_updated_at
		FROM hosts h
		WHERE
			h.platform = 'windows'
			AND h.uuid = ?
			AND EXISTS (
				SELECT 1 FROM mdm_windows_enrollments mwe WHERE mwe.host_uuid = h.uuid
			)
			AND EXISTS (
				SELECT 1 FROM host_mdm hmdm WHERE hmdm.host_id = h.id AND hmdm.enrolled = 1
			)
		LIMIT 1
	`

	var host fleet.WindowsHostReconcileInfo
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &host, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "get windows mdm host for reconcile")
	}
	return &host, nil
}

// ListWindowsProfilesForReconcileByTeam is the per-host variant of the snapshot's profile loader: it loads only profiles for the
// host's team. team_id=0 is its own team (the "no team" scope); a host with a real team does NOT inherit team_id=0 profiles.
func (ds *Datastore) ListWindowsProfilesForReconcileByTeam(ctx context.Context, teamID uint) ([]*fleet.WindowsProfileForReconcile, error) {
	return ds.listWindowsProfilesForReconcileTransaction(ctx, ds.reader(ctx), &teamID)
}

// BulkGetHostMDMWindowsProfilesByUUIDs returns the current host_mdm_windows_profiles rows for the given host UUIDs, grouped by
// host UUID.
func (ds *Datastore) BulkGetHostMDMWindowsProfilesByUUIDs(ctx context.Context, hostUUIDs []string) (map[string][]*fleet.MDMWindowsProfilePayload, error) {
	return ds.bulkGetHostMDMWindowsProfilesByUUIDsTransaction(ctx, ds.reader(ctx), hostUUIDs)
}

// listWindowsProfilesForReconcileTransaction loads Windows configuration profiles in the system, paired with their label
// assignments. When teamID is nil every profile is returned (used by the batched reconciler snapshot); when teamID is set only
// profiles for that team are returned (used by the per-host enrollment path). Mirrors the Apple
// listAppleProfilesForReconcileTransaction so the in-memory handlers apply the same "broken-label" semantics and broken profiles
// are exempted from removal.
func (ds *Datastore) listWindowsProfilesForReconcileTransaction(
	ctx context.Context,
	tx common_mysql.DBReadTx,
	teamID *uint,
) ([]*fleet.WindowsProfileForReconcile, error) {
	type profileRow struct {
		ProfileUUID      string       `db:"profile_uuid"`
		ProfileName      string       `db:"name"`
		TeamID           uint         `db:"team_id"`
		Checksum         []byte       `db:"checksum"`
		SecretsUpdatedAt sql.NullTime `db:"secrets_updated_at"`
	}

	profStmt := `
		SELECT profile_uuid, name, team_id, checksum, secrets_updated_at
		FROM mdm_windows_configuration_profiles
	`
	var profArgs []any
	if teamID != nil {
		// team_id=0 is the "no team" / global team
		profStmt += ` WHERE team_id = ?`
		profArgs = append(profArgs, *teamID)
	}

	var rows []profileRow
	if err := sqlx.SelectContext(ctx, tx, &rows, profStmt, profArgs...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list windows profiles for reconcile")
	}
	if len(rows) == 0 {
		return nil, nil
	}

	byUUID := make(map[string]*fleet.WindowsProfileForReconcile, len(rows))
	out := make([]*fleet.WindowsProfileForReconcile, 0, len(rows))
	profileUUIDs := make([]string, 0, len(rows))
	for _, r := range rows {
		p := &fleet.WindowsProfileForReconcile{
			ProfileUUID: r.ProfileUUID,
			ProfileName: r.ProfileName,
			TeamID:      r.TeamID,
			Checksum:    r.Checksum,
		}
		if r.SecretsUpdatedAt.Valid {
			t := r.SecretsUpdatedAt.Time
			p.SecretsUpdatedAt = &t
		}
		byUUID[r.ProfileUUID] = p
		out = append(out, p)
		profileUUIDs = append(profileUUIDs, r.ProfileUUID)
	}

	// Load label assignments, joining labels to get membership type and label creation time (needed by the exclude-any handler).
	// Broken labels (label_id IS NULL after the LEFT JOIN, i.e. the label was deleted) are retained so the handlers can
	// disqualify/exempt the profile.
	//
	// Leave created_at un-COALESCE'd. NULL here means a broken (deleted) label and is intentional.
	labelStmt := `
		SELECT
			mcpl.windows_profile_uuid AS profile_uuid,
			mcpl.label_id             AS label_id,
			mcpl.exclude              AS exclude,
			mcpl.require_all          AS require_all,
			lbl.created_at            AS label_created_at,
			COALESCE(lbl.label_membership_type, 0) AS label_membership_type
		FROM mdm_configuration_profile_labels mcpl
		LEFT JOIN labels lbl ON lbl.id = mcpl.label_id
		WHERE mcpl.windows_profile_uuid IS NOT NULL
	`
	var labelStmtArgs []any
	if teamID != nil {
		// Per-host path: restrict label rows to the team-scoped profiles loaded above
		labelStmt += ` AND mcpl.windows_profile_uuid IN (?)`
		q, args, err := sqlx.In(labelStmt, profileUUIDs)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "build windows profile labels query")
		}
		labelStmt = q
		labelStmtArgs = args
	}

	type labelRow struct {
		ProfileUUID         string        `db:"profile_uuid"`
		LabelID             sql.NullInt64 `db:"label_id"`
		Exclude             bool          `db:"exclude"`
		RequireAll          bool          `db:"require_all"`
		LabelCreatedAt      sql.NullTime  `db:"label_created_at"`
		LabelMembershipType int           `db:"label_membership_type"`
	}

	var labelRows []labelRow
	if err := sqlx.SelectContext(ctx, tx, &labelRows, labelStmt, labelStmtArgs...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list windows profile labels for reconcile")
	}

	// Per-profile include-mode discovery. Include labels for a single profile must share a single require_all value; the first
	// include row sets the mode and later disagreements mark it mixed. Exclude rows always go to ExcludeLabels and have a single
	// "exclude any" semantic. A profile may carry both an include set and an exclude set.
	type includeAccum struct {
		set   bool
		mode  fleet.MDMProfileIncludeMode
		mixed bool
	}
	includeModes := make(map[string]*includeAccum, len(byUUID))

	for _, lr := range labelRows {
		p, ok := byUUID[lr.ProfileUUID]
		if !ok {
			continue
		}

		ref := fleet.MDMProfileLabelRef{
			LabelMembershipType: lr.LabelMembershipType,
		}
		if lr.LabelID.Valid {
			id := uint(lr.LabelID.Int64) //nolint:gosec // dismiss G115: labels.id is int unsigned in MySQL
			ref.LabelID = &id
		}
		if lr.LabelCreatedAt.Valid {
			ref.CreatedAt = lr.LabelCreatedAt.Time
		}

		if lr.Exclude {
			p.ExcludeLabels = append(p.ExcludeLabels, ref)
			continue
		}

		// Include row.
		p.IncludeLabels = append(p.IncludeLabels, ref)

		rowMode := fleet.MDMProfileIncludeAny
		if lr.RequireAll {
			rowMode = fleet.MDMProfileIncludeAll
		}

		ia := includeModes[lr.ProfileUUID]
		if ia == nil {
			ia = &includeAccum{}
			includeModes[lr.ProfileUUID] = ia
		}
		if !ia.set {
			ia.mode = rowMode
			ia.set = true
		} else if ia.mode != rowMode {
			ia.mixed = true
		}
	}

	for uuid, ia := range includeModes {
		p := byUUID[uuid]
		if p == nil {
			// Unreachable: every includeModes key came from a label row whose profile UUID is in byUUID. Guard anyway to satisfy nil
			// analysis.
			continue
		}
		if ia.mixed {
			// Defensive: include rows disagreed on require_all (should be impossible in production since the upsert path enforces a single mode).
			// Drop the include set so we don't guess at intent; exclude labels (if any) are preserved.
			p.IncludeLabels = nil
			p.IncludeMode = fleet.MDMProfileIncludeNone
			errMsg := "windows profile has mixed include label modes; ignoring include labels"
			ds.logger.ErrorContext(ctx, errMsg, "profile_uuid", uuid, "team_id",
				p.TeamID)
			ctxerr.Handle(ctx, errors.New(errMsg))
			continue
		}
		p.IncludeMode = ia.mode
	}

	return out, nil
}

// bulkGetHostMDMWindowsProfilesByUUIDsTransaction returns the current host_mdm_windows_profiles rows for the given host UUIDs,
// grouped by host UUID.
//
// The caller (GetWindowsProfileReconcileSnapshot) always passes the reconcile host window, bounded by
// reconcileWindowsProfilesBatchSize (a per-tick read budget in the low thousands), which stays far under MySQL's ~65k
// prepared-statement placeholder limit. The IN clause therefore fits in a single query and is intentionally not batched.
func (ds *Datastore) bulkGetHostMDMWindowsProfilesByUUIDsTransaction(
	ctx context.Context,
	tx common_mysql.DBReadTx,
	hostUUIDs []string,
) (map[string][]*fleet.MDMWindowsProfilePayload, error) {
	out := make(map[string][]*fleet.MDMWindowsProfilePayload, len(hostUUIDs))
	if len(hostUUIDs) == 0 {
		return out, nil
	}

	const stmt = `
		SELECT
			profile_uuid,
			host_uuid,
			profile_name,
			status,
			operation_type,
			COALESCE(detail, '') AS detail,
			command_uuid,
			retries,
			checksum,
			secrets_updated_at
		FROM host_mdm_windows_profiles
		WHERE host_uuid IN (?)
	`

	q, args, err := sqlx.In(stmt, hostUUIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build host mdm windows profiles query")
	}

	var rows []*fleet.MDMWindowsProfilePayload
	if err := sqlx.SelectContext(ctx, tx, &rows, q, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select host mdm windows profiles")
	}

	for _, r := range rows {
		out[r.HostUUID] = append(out[r.HostUUID], r)
	}

	return out, nil
}

// GetWindowsProfileReconcileSnapshot loads the four pieces of state the batched Windows profile reconciler needs — the bounded
// host window, every profile (with label assignments), host↔label memberships restricted to labels referenced by those profiles,
// and current host_mdm_windows_profiles rows for the hosts in the window. All reads run inside a single read-only transaction so
// they observe one MySQL snapshot.
//
// The read-only REPEATABLE READ transaction is load-bearing, not incidental: it makes the desired-state inputs (profiles, labels,
// memberships) and the current state (host_mdm_windows_profiles) coherent at one instant, so a concurrent admin mutation (e.g.
// deleting a profile, which also deletes its host rows) cannot produce a torn diff with spurious install/remove targets. Do not
// pull these reads out of the transaction (e.g. to load profiles once per tick) without weighing that consistency loss.
//
// When the host window is empty the remaining queries are skipped — the caller short-circuits in that case anyway, and there's no
// point loading profiles or memberships we won't use. Mirrors GetAppleProfileReconcileSnapshot.
func (ds *Datastore) GetWindowsProfileReconcileSnapshot(ctx context.Context, afterHostUUID string, batchSize int) (
	hosts []*fleet.WindowsHostReconcileInfo,
	allProfiles []*fleet.WindowsProfileForReconcile,
	hostLabels map[uint]map[uint]struct{},
	currentByHost map[string][]*fleet.MDMWindowsProfilePayload,
	err error,
) {
	err = ds.withReadTx(ctx, func(tx common_mysql.DBReadTx) error {
		var inner error
		hosts, inner = ds.listWindowsMDMHostsForReconcileBatchTransaction(ctx, tx, afterHostUUID, batchSize)
		if inner != nil {
			return inner
		}
		if len(hosts) == 0 {
			return nil
		}

		allProfiles, inner = ds.listWindowsProfilesForReconcileTransaction(ctx, tx, nil)
		if inner != nil {
			return inner
		}

		hostIDs := make([]uint, 0, len(hosts))
		hostUUIDs := make([]string, 0, len(hosts))
		for _, h := range hosts {
			hostIDs = append(hostIDs, h.HostID)
			hostUUIDs = append(hostUUIDs, h.UUID)
		}

		labelIDSet := make(map[uint]struct{})
		for _, p := range allProfiles {
			for _, lr := range p.IncludeLabels {
				if lr.LabelID != nil {
					labelIDSet[*lr.LabelID] = struct{}{}
				}
			}
			for _, lr := range p.ExcludeLabels {
				if lr.LabelID != nil {
					labelIDSet[*lr.LabelID] = struct{}{}
				}
			}
		}
		labelIDs := make([]uint, 0, len(labelIDSet))
		for id := range labelIDSet {
			labelIDs = append(labelIDs, id)
		}

		hostLabels, inner = ds.bulkGetHostLabelMembershipsTransaction(ctx, tx, hostIDs, labelIDs)
		if inner != nil {
			return inner
		}

		currentByHost, inner = ds.bulkGetHostMDMWindowsProfilesByUUIDsTransaction(ctx, tx, hostUUIDs)
		return inner
	})
	if err != nil {
		return nil, nil, nil, nil, ctxerr.Wrap(ctx, err, "windows profile reconcile snapshot")
	}
	return hosts, allProfiles, hostLabels, currentByHost, nil
}
