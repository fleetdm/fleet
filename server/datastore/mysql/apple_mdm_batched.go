package mysql

import (
	"context"
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// ListAppleMDMHostsForReconcileBatch returns up to batchSize Apple-MDM-
// enrolled hosts with host_uuid > afterHostUUID, ordered ascending by uuid,
// along with the fields the batched reconciler needs to compute desired
// state in memory.
//
// Selection criteria mirror the host-side filters in the legacy desired-
// state query (generateDesiredStateQuery): platform in (darwin, ios, ipados),
// an enabled nano_enrollment of type Device or "User Enrollment (Device)",
// and an existing nano_devices row supplying authenticate_at.
func (ds *Datastore) ListAppleMDMHostsForReconcileBatch(
	ctx context.Context,
	afterHostUUID string,
	batchSize int,
) ([]*fleet.AppleHostReconcileInfo, error) {
	const stmt = `
		SELECT
			h.id              AS id,
			h.uuid            AS uuid,
			h.team_id         AS team_id,
			h.platform        AS platform,
			h.label_updated_at AS label_updated_at,
			nd.authenticate_at AS device_enrolled_at
		FROM hosts h
		JOIN nano_enrollments ne
			ON ne.device_id = h.uuid
			AND ne.enabled = 1
			AND ne.type IN ('Device', 'User Enrollment (Device)')
		JOIN nano_devices nd
			ON nd.id = ne.device_id
		WHERE
			(h.platform = 'darwin' OR h.platform = 'ios' OR h.platform = 'ipados')
			AND h.uuid > ?
		ORDER BY h.uuid
		LIMIT ?
	`

	var hosts []*fleet.AppleHostReconcileInfo
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, stmt, afterHostUUID, batchSize); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list apple mdm hosts for reconcile batch")
	}
	return hosts, nil
}

// ListAppleProfilesForReconcile loads every Apple configuration profile in
// the system, paired with its label assignments. The result is intended to
// be loaded once per reconciliation tick and used to evaluate desired state
// per host in memory.
//
// Label assignments include broken labels (label_id IS NULL) so the
// in-memory handlers can apply the same "broken-label" semantics as the
// legacy SQL: broken include-* profiles do not apply, and broken profiles
// are exempted from removal.
func (ds *Datastore) ListAppleProfilesForReconcile(ctx context.Context) ([]*fleet.AppleProfileForReconcile, error) {
	type profileRow struct {
		ProfileUUID       string             `db:"profile_uuid"`
		ProfileIdentifier string             `db:"identifier"`
		ProfileName       string             `db:"name"`
		TeamID            uint               `db:"team_id"`
		Checksum          []byte             `db:"checksum"`
		SecretsUpdatedAt  sql.NullTime       `db:"secrets_updated_at"`
		Scope             fleet.PayloadScope `db:"scope"`
	}

	const profStmt = `
		SELECT profile_uuid, identifier, name, team_id, checksum, secrets_updated_at, scope
		FROM mdm_apple_configuration_profiles
	`

	var rows []profileRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, profStmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list apple profiles for reconcile")
	}
	if len(rows) == 0 {
		return nil, nil
	}

	byUUID := make(map[string]*fleet.AppleProfileForReconcile, len(rows))
	out := make([]*fleet.AppleProfileForReconcile, 0, len(rows))
	for _, r := range rows {
		p := &fleet.AppleProfileForReconcile{
			ProfileUUID:       r.ProfileUUID,
			ProfileIdentifier: r.ProfileIdentifier,
			ProfileName:       r.ProfileName,
			TeamID:            r.TeamID,
			Checksum:          r.Checksum,
			Scope:             r.Scope,
		}
		if r.SecretsUpdatedAt.Valid {
			t := r.SecretsUpdatedAt.Time
			p.SecretsUpdatedAt = &t
		}
		byUUID[r.ProfileUUID] = p
		out = append(out, p)
	}

	// Load label assignments, joining labels to get membership type and
	// label creation time (needed by the exclude-any handler).
	const labelStmt = `
		SELECT
			mcpl.apple_profile_uuid AS profile_uuid,
			mcpl.label_id           AS label_id,
			mcpl.exclude            AS exclude,
			mcpl.require_all        AS require_all,
			COALESCE(lbl.created_at, '2000-01-01 00:00:00') AS label_created_at,
			COALESCE(lbl.label_membership_type, 0) AS label_membership_type
		FROM mdm_configuration_profile_labels mcpl
		LEFT JOIN labels lbl ON lbl.id = mcpl.label_id
		WHERE mcpl.apple_profile_uuid IS NOT NULL
	`

	type labelRow struct {
		ProfileUUID         string        `db:"profile_uuid"`
		LabelID             sql.NullInt64 `db:"label_id"`
		Exclude             bool          `db:"exclude"`
		RequireAll          bool          `db:"require_all"`
		LabelCreatedAt      sql.NullTime  `db:"label_created_at"`
		LabelMembershipType int           `db:"label_membership_type"`
	}

	var labelRows []labelRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labelRows, labelStmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list apple profile labels for reconcile")
	}

	// Per-profile label mode discovery. A profile's mode is set by the first
	// label row seen; if later rows disagree we mark it mixed and treat the
	// profile as no-label (defensive — upsert enforces consistency).
	type modeMarker struct {
		set   bool
		mode  fleet.AppleProfileLabelMode
		mixed bool
	}
	modes := make(map[string]*modeMarker, len(byUUID))

	for _, lr := range labelRows {
		p, ok := byUUID[lr.ProfileUUID]
		if !ok {
			continue
		}

		mm := modes[lr.ProfileUUID]
		if mm == nil {
			mm = &modeMarker{}
			modes[lr.ProfileUUID] = mm
		}

		var rowMode fleet.AppleProfileLabelMode
		switch {
		case lr.Exclude:
			rowMode = fleet.AppleProfileLabelModeExcludeAny
		case lr.RequireAll:
			rowMode = fleet.AppleProfileLabelModeIncludeAll
		default:
			rowMode = fleet.AppleProfileLabelModeIncludeAny
		}

		if !mm.set {
			mm.mode = rowMode
			mm.set = true
		} else if mm.mode != rowMode {
			mm.mixed = true
		}

		ref := fleet.AppleProfileLabelRef{
			LabelMembershipType: lr.LabelMembershipType,
		}
		if lr.LabelID.Valid {
			id := uint(lr.LabelID.Int64) //nolint:gosec // dismiss G115: labels.id is int unsigned in MySQL
			ref.LabelID = &id
		}
		if lr.LabelCreatedAt.Valid {
			ref.CreatedAt = lr.LabelCreatedAt.Time
		}
		p.Labels = append(p.Labels, ref)
	}

	for uuid, mm := range modes {
		p := byUUID[uuid]
		if mm.mixed {
			p.LabelMode = fleet.AppleProfileLabelModeNone
			p.Labels = nil
			continue
		}
		p.LabelMode = mm.mode
	}

	return out, nil
}

// BulkGetHostLabelMemberships returns, for each given host ID, the set of
// label IDs (from the provided labelIDs) the host is a member of.
//
// Both lists may be empty; in either case the result is an empty (non-nil)
// map. The IN clauses are chunked to keep total placeholders well under
// MySQL's prepared-statement parameter limit.
func (ds *Datastore) BulkGetHostLabelMemberships(
	ctx context.Context,
	hostIDs []uint,
	labelIDs []uint,
) (map[uint]map[uint]struct{}, error) {
	out := make(map[uint]map[uint]struct{}, len(hostIDs))
	if len(hostIDs) == 0 || len(labelIDs) == 0 {
		return out, nil
	}

	const (
		hostChunk  = 5000
		labelChunk = 1000
	)

	stmt := `SELECT host_id, label_id FROM label_membership WHERE host_id IN (?) AND label_id IN (?)`

	for hi := 0; hi < len(hostIDs); hi += hostChunk {
		hEnd := min(hi+hostChunk, len(hostIDs))
		hostBatch := hostIDs[hi:hEnd]

		for li := 0; li < len(labelIDs); li += labelChunk {
			lEnd := min(li+labelChunk, len(labelIDs))
			labelBatch := labelIDs[li:lEnd]

			q, args, err := sqlx.In(stmt, hostBatch, labelBatch)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "build label membership query")
			}

			rows, err := ds.reader(ctx).QueryxContext(ctx, q, args...)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "query host label memberships")
			}

			for rows.Next() {
				var hostID, labelID uint
				if err := rows.Scan(&hostID, &labelID); err != nil {
					rows.Close()
					return nil, ctxerr.Wrap(ctx, err, "scan label membership row")
				}
				set, ok := out[hostID]
				if !ok {
					set = make(map[uint]struct{})
					out[hostID] = set
				}
				set[labelID] = struct{}{}
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				return nil, ctxerr.Wrap(ctx, err, "iterate label membership rows")
			}
			rows.Close()
		}
	}

	return out, nil
}

// BulkGetHostMDMAppleProfilesByUUIDs returns the current host_mdm_apple_profiles
// rows for the given host UUIDs, grouped by host UUID.
//
// The returned MDMAppleProfilePayload fields mirror what the legacy
// listMDMAppleProfilesToRemoveTransaction returns. HostPlatform and
// DeviceEnrolledAt are left zero because they come from joined tables the
// in-memory reconciler already has from ListAppleMDMHostsForReconcileBatch.
func (ds *Datastore) BulkGetHostMDMAppleProfilesByUUIDs(
	ctx context.Context,
	hostUUIDs []string,
) (map[string][]*fleet.MDMAppleProfilePayload, error) {
	out := make(map[string][]*fleet.MDMAppleProfilePayload, len(hostUUIDs))
	if len(hostUUIDs) == 0 {
		return out, nil
	}

	const stmt = `
		SELECT
			profile_uuid,
			profile_identifier,
			profile_name,
			host_uuid,
			checksum,
			secrets_updated_at,
			status,
			operation_type,
			COALESCE(detail, '') AS detail,
			command_uuid,
			ignore_error,
			scope
		FROM host_mdm_apple_profiles
		WHERE host_uuid IN (?)
	`

	const chunk = 5000

	for i := 0; i < len(hostUUIDs); i += chunk {
		end := min(i+chunk, len(hostUUIDs))
		batch := hostUUIDs[i:end]

		q, args, err := sqlx.In(stmt, batch)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "build host mdm apple profiles query")
		}

		var rows []*fleet.MDMAppleProfilePayload
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, q, args...); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "select host mdm apple profiles")
		}

		for _, r := range rows {
			out[r.HostUUID] = append(out[r.HostUUID], r)
		}
	}

	return out, nil
}

// GetMDMAppleReconcileCursor returns the persisted host_uuid cursor used by
// the batched Apple MDM reconciliation cron. The bare mysql.Datastore has no
// place to persist it, so this returns "" (fresh start). The mysqlredis
// wrapper overrides this to back it with Redis.
func (ds *Datastore) GetMDMAppleReconcileCursor(_ context.Context) (string, error) {
	return "", nil
}

// SetMDMAppleReconcileCursor persists the host_uuid cursor used by the
// batched Apple MDM reconciliation cron. The bare mysql.Datastore is a
// no-op; the mysqlredis wrapper backs it with Redis.
func (ds *Datastore) SetMDMAppleReconcileCursor(_ context.Context, _ string) error {
	return nil
}
