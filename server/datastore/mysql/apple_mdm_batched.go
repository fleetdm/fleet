package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

// GetAppleMDMHostForReconcile returns the Apple-MDM reconcile info for a
// single host UUID, or (nil, nil) if the host is not enrolled or not an
// Apple platform. Uses the same JOIN as ListAppleMDMHostsForReconcileBatch
// so per-host and per-batch reconcile paths see the same eligibility rules.
func (ds *Datastore) GetAppleMDMHostForReconcile(
	ctx context.Context,
	hostUUID string,
) (*fleet.AppleHostReconcileInfo, error) {
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
			AND h.uuid = ?
		LIMIT 1
	`

	var host fleet.AppleHostReconcileInfo
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &host, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "get apple mdm host for reconcile")
	}
	return &host, nil
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
	return ds.listAppleProfilesForReconcile(ctx, nil)
}

// ListAppleProfilesForReconcileByTeam is the per-host variant: it loads
// only profiles for the host's team. team_id=0 is its own team (the
// "no team" scope); a host with a real team does NOT inherit team_id=0
// profiles. Used by ReconcileProfilesForEnrollingHost so the worker
// doesn't scan every profile in the system on each enrollment — a real
// concern in suites that accumulate profile rows across many sub-tests
// without cleanup.
func (ds *Datastore) ListAppleProfilesForReconcileByTeam(ctx context.Context, teamID uint) ([]*fleet.AppleProfileForReconcile, error) {
	return ds.listAppleProfilesForReconcile(ctx, &teamID)
}

func (ds *Datastore) listAppleProfilesForReconcile(ctx context.Context, teamID *uint) ([]*fleet.AppleProfileForReconcile, error) {
	type profileRow struct {
		ProfileUUID       string             `db:"profile_uuid"`
		ProfileIdentifier string             `db:"identifier"`
		ProfileName       string             `db:"name"`
		TeamID            uint               `db:"team_id"`
		Checksum          []byte             `db:"checksum"`
		SecretsUpdatedAt  sql.NullTime       `db:"secrets_updated_at"`
		Scope             fleet.PayloadScope `db:"scope"`
	}

	profStmt := `
		SELECT profile_uuid, identifier, name, team_id, checksum, secrets_updated_at, scope
		FROM mdm_apple_configuration_profiles
	`
	var profArgs []any
	if teamID != nil {
		// team_id=0 is the "no team" / global team — its own scope.
		// A host with a real team only matches profiles for that team;
		// it does NOT also inherit team_id=0 profiles. EffectiveTeamID()
		// on the host already maps nil → 0, so equality is correct.
		profStmt += ` WHERE team_id = ?`
		profArgs = append(profArgs, *teamID)
	}

	var rows []profileRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, profStmt, profArgs...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list apple profiles for reconcile")
	}
	if len(rows) == 0 {
		return nil, nil
	}

	byUUID := make(map[string]*fleet.AppleProfileForReconcile, len(rows))
	out := make([]*fleet.AppleProfileForReconcile, 0, len(rows))
	profileUUIDs := make([]string, 0, len(rows))
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
		profileUUIDs = append(profileUUIDs, r.ProfileUUID)
	}

	// Load label assignments, joining labels to get membership type and
	// label creation time (needed by the exclude-any handler).
	//
	// Do not COALESCE label_created_at to a string literal — MySQL would
	// coerce the result column to VARCHAR and the driver returns []uint8,
	// which sql.NullTime cannot scan. The exclude-any handler already
	// treats a zero CreatedAt as "no timing check", which is the natural
	// outcome of a NULL → invalid NullTime → zero time.Time.
	//
	// When teamID is set we restrict the label rows to the profile UUIDs
	// we just loaded — the WHERE IN clause is on the same set so the
	// query never returns labels for profiles outside our team window.
	labelStmt := `
		SELECT
			mcpl.apple_profile_uuid AS profile_uuid,
			mcpl.label_id           AS label_id,
			mcpl.exclude            AS exclude,
			mcpl.require_all        AS require_all,
			lbl.created_at          AS label_created_at,
			COALESCE(lbl.label_membership_type, 0) AS label_membership_type
		FROM mdm_configuration_profile_labels mcpl
		LEFT JOIN labels lbl ON lbl.id = mcpl.label_id
		WHERE mcpl.apple_profile_uuid IS NOT NULL
	`
	var labelStmtArgs []any
	if teamID != nil {
		labelStmt += ` AND mcpl.apple_profile_uuid IN (?)`
		q, args, err := sqlx.In(labelStmt, profileUUIDs)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "build apple profile labels query")
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
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labelRows, labelStmt, labelStmtArgs...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list apple profile labels for reconcile")
	}

	// Per-profile include-mode discovery. Include labels for a single
	// profile must share a single require_all value; the first include
	// row sets the mode and later disagreements mark it mixed. Exclude
	// rows always go to ExcludeLabels and have a single "exclude any"
	// semantic (their require_all column is ignored). A profile may
	// carry both an include set and an exclude set.
	type includeAccum struct {
		set   bool
		mode  fleet.AppleProfileIncludeMode
		mixed bool
	}
	includeModes := make(map[string]*includeAccum, len(byUUID))

	for _, lr := range labelRows {
		p, ok := byUUID[lr.ProfileUUID]
		if !ok {
			continue
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

		if lr.Exclude {
			p.ExcludeLabels = append(p.ExcludeLabels, ref)
			continue
		}

		// Include row.
		p.IncludeLabels = append(p.IncludeLabels, ref)

		var rowMode fleet.AppleProfileIncludeMode
		if lr.RequireAll {
			rowMode = fleet.AppleProfileIncludeAll
		} else {
			rowMode = fleet.AppleProfileIncludeAny
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
		if ia.mixed {
			// Defensive: include rows disagreed on require_all (should
			// be impossible in production — the upsert path enforces a
			// single mode). Drop the include set so we don't guess at
			// intent; exclude labels (if any) are preserved.
			p.IncludeLabels = nil
			p.IncludeMode = fleet.AppleProfileIncludeNone
			ds.logger.WarnContext(ctx, "apple profile has mixed include label modes; ignoring include labels", "profile_uuid", uuid, "team_id", p.TeamID)
			continue
		}
		p.IncludeMode = ia.mode
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

	const stmt = `SELECT host_id, label_id FROM label_membership WHERE host_id IN (?) AND label_id IN (?)`

	type membershipRow struct {
		HostID  uint `db:"host_id"`
		LabelID uint `db:"label_id"`
	}

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

			var rows []membershipRow
			if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, q, args...); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "query host label memberships")
			}

			for _, r := range rows {
				set, ok := out[r.HostID]
				if !ok {
					set = make(map[uint]struct{})
					out[r.HostID] = set
				}
				set[r.LabelID] = struct{}{}
			}
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

// ListAppleDeclarationsForReconcile loads every Apple declaration in the
// system, paired with its label assignments. Mirrors
// ListAppleProfilesForReconcile so the batched DDM reconciler runs the
// same label-row processing logic.
func (ds *Datastore) ListAppleDeclarationsForReconcile(ctx context.Context) ([]*fleet.AppleDeclarationForReconcile, error) {
	type declRow struct {
		DeclarationUUID       string             `db:"declaration_uuid"`
		DeclarationIdentifier string             `db:"identifier"`
		DeclarationName       string             `db:"name"`
		TeamID                uint               `db:"team_id"`
		Token                 []byte             `db:"token"`
		SecretsUpdatedAt      sql.NullTime       `db:"secrets_updated_at"`
		Scope                 fleet.PayloadScope `db:"scope"`
	}

	const declStmt = `
		SELECT declaration_uuid, identifier, name, team_id, token, secrets_updated_at, scope
		FROM mdm_apple_declarations
	`

	var rows []declRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, declStmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list apple declarations for reconcile")
	}
	if len(rows) == 0 {
		return nil, nil
	}

	byUUID := make(map[string]*fleet.AppleDeclarationForReconcile, len(rows))
	out := make([]*fleet.AppleDeclarationForReconcile, 0, len(rows))
	for _, r := range rows {
		d := &fleet.AppleDeclarationForReconcile{
			DeclarationUUID:       r.DeclarationUUID,
			DeclarationIdentifier: r.DeclarationIdentifier,
			DeclarationName:       r.DeclarationName,
			TeamID:                r.TeamID,
			Token:                 r.Token,
			Scope:                 r.Scope,
		}
		if r.SecretsUpdatedAt.Valid {
			t := r.SecretsUpdatedAt.Time
			d.SecretsUpdatedAt = &t
		}
		byUUID[r.DeclarationUUID] = d
		out = append(out, d)
	}

	// Declaration label assignments live in their own table —
	// mdm_declaration_labels — NOT mdm_configuration_profile_labels.
	// The two tables have separate schemas and FK relationships:
	// declaration labels FK to mdm_apple_declarations.declaration_uuid,
	// profile labels FK to mdm_apple_configuration_profiles.profile_uuid.
	// Querying the wrong table returns zero rows for declarations.
	const labelStmt = `
		SELECT
			mdl.apple_declaration_uuid AS entity_uuid,
			mdl.label_id               AS label_id,
			mdl.exclude                AS exclude,
			mdl.require_all            AS require_all,
			lbl.created_at             AS label_created_at,
			COALESCE(lbl.label_membership_type, 0) AS label_membership_type
		FROM mdm_declaration_labels mdl
		LEFT JOIN labels lbl ON lbl.id = mdl.label_id
	`

	type labelRow struct {
		EntityUUID          string        `db:"entity_uuid"`
		LabelID             sql.NullInt64 `db:"label_id"`
		Exclude             bool          `db:"exclude"`
		RequireAll          bool          `db:"require_all"`
		LabelCreatedAt      sql.NullTime  `db:"label_created_at"`
		LabelMembershipType int           `db:"label_membership_type"`
	}

	var labelRows []labelRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labelRows, labelStmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list apple declaration labels for reconcile")
	}

	type includeAccum struct {
		set   bool
		mode  fleet.AppleProfileIncludeMode
		mixed bool
	}
	includeModes := make(map[string]*includeAccum, len(byUUID))

	for _, lr := range labelRows {
		d, ok := byUUID[lr.EntityUUID]
		if !ok {
			continue
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

		if lr.Exclude {
			d.ExcludeLabels = append(d.ExcludeLabels, ref)
			continue
		}

		d.IncludeLabels = append(d.IncludeLabels, ref)

		var rowMode fleet.AppleProfileIncludeMode
		if lr.RequireAll {
			rowMode = fleet.AppleProfileIncludeAll
		} else {
			rowMode = fleet.AppleProfileIncludeAny
		}

		ia := includeModes[lr.EntityUUID]
		if ia == nil {
			ia = &includeAccum{}
			includeModes[lr.EntityUUID] = ia
		}
		if !ia.set {
			ia.mode = rowMode
			ia.set = true
		} else if ia.mode != rowMode {
			ia.mixed = true
		}
	}

	for uuid, ia := range includeModes {
		d := byUUID[uuid]
		if ia.mixed {
			d.IncludeLabels = nil
			d.IncludeMode = fleet.AppleProfileIncludeNone

			ds.logger.WarnContext(ctx, "apple declaration has mixed include label modes; ignoring include labels", "declaration_uuid", uuid, "team_id", d.TeamID)
			continue
		}
		d.IncludeMode = ia.mode
	}

	// Mark declarations that reference Fleet variables. The batched DDM
	// reconciler uses this to set host_mdm_apple_declarations.variables_updated_at
	// on install rows, mirroring the legacy setVariablesUpdatedAtForDeclarations.
	declUUIDs := make([]string, 0, len(byUUID))
	for u := range byUUID {
		declUUIDs = append(declUUIDs, u)
	}
	if len(declUUIDs) > 0 {
		const varsStmt = `SELECT DISTINCT apple_declaration_uuid FROM mdm_configuration_profile_variables WHERE apple_declaration_uuid IN (?)`
		q, args, err := sqlx.In(varsStmt, declUUIDs)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "build apple declaration variables query")
		}
		var withVars []string
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &withVars, q, args...); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "select apple declarations with fleet variables")
		}
		for _, u := range withVars {
			if d, ok := byUUID[u]; ok {
				d.HasFleetVariables = true
			}
		}
	}

	return out, nil
}

// BulkGetHostMDMAppleDeclarationsByUUIDs returns the current
// host_mdm_apple_declarations rows for the given host UUIDs, grouped by
// host UUID. Mirrors BulkGetHostMDMAppleProfilesByUUIDs.
func (ds *Datastore) BulkGetHostMDMAppleDeclarationsByUUIDs(
	ctx context.Context,
	hostUUIDs []string,
) (map[string][]*fleet.MDMAppleHostDeclaration, error) {
	out := make(map[string][]*fleet.MDMAppleHostDeclaration, len(hostUUIDs))
	if len(hostUUIDs) == 0 {
		return out, nil
	}

	// host_mdm_apple_declarations.token is binary(16). Selecting it
	// directly lets the MySQL driver return raw bytes that sqlx scans
	// into MDMAppleHostDeclaration.Token (string holding raw bytes) —
	// matching how the legacy DDM code reads the same column. The diff
	// in computeAppleDeclarationDeltas compares this against
	// AppleDeclarationForReconcile.Token ([]byte, also raw bytes), so
	// both sides are the same 16-byte binary content.
	const stmt = `
		SELECT
			host_uuid,
			declaration_uuid,
			declaration_identifier,
			declaration_name,
			status,
			operation_type,
			COALESCE(detail, '') AS detail,
			token,
			secrets_updated_at,
			variables_updated_at
		FROM host_mdm_apple_declarations
		WHERE host_uuid IN (?)
	`

	const chunk = 5000

	for i := 0; i < len(hostUUIDs); i += chunk {
		end := min(i+chunk, len(hostUUIDs))
		batch := hostUUIDs[i:end]

		q, args, err := sqlx.In(stmt, batch)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "build host mdm apple declarations query")
		}

		var rows []*fleet.MDMAppleHostDeclaration
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, q, args...); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "select host mdm apple declarations")
		}

		for _, r := range rows {
			out[r.HostUUID] = append(out[r.HostUUID], r)
		}
	}

	return out, nil
}

// GetMDMAppleDeclarationReconcileCursor / SetMDMAppleDeclarationReconcileCursor
// mirror the profile cursor helpers — bare mysql.Datastore is a no-op, the
// mysqlredis wrapper backs both with Redis under a separate key so the
// profile and declaration cursors advance independently.
func (ds *Datastore) GetMDMAppleDeclarationReconcileCursor(_ context.Context) (string, error) {
	return "", nil
}

func (ds *Datastore) SetMDMAppleDeclarationReconcileCursor(_ context.Context, _ string) error {
	return nil
}

// BulkUpsertMDMAppleHostDeclarations writes the given host declaration
// rows, setting status / operation_type / token / secrets_updated_at.
// The Status, OperationType and Token fields on each row are used
// directly (per-row), unlike the legacy
// mdmAppleBatchSetPendingHostDeclarationsDB which forces a single
// status across all rows. Used by the batched DDM reconciler.
func (ds *Datastore) BulkUpsertMDMAppleHostDeclarations(
	ctx context.Context,
	rows []*fleet.MDMAppleHostDeclaration,
) error {
	if len(rows) == 0 {
		return nil
	}

	const baseStmt = `
		INSERT INTO host_mdm_apple_declarations
		  (host_uuid, declaration_uuid, declaration_identifier, declaration_name,
		   status, operation_type, token, secrets_updated_at, variables_updated_at)
		VALUES %s
		ON DUPLICATE KEY UPDATE
		  status = VALUES(status),
		  operation_type = VALUES(operation_type),
		  token = VALUES(token),
		  declaration_identifier = VALUES(declaration_identifier),
		  declaration_name = VALUES(declaration_name),
		  secrets_updated_at = VALUES(secrets_updated_at),
		  variables_updated_at = VALUES(variables_updated_at)
	`

	const batchSize = 1000
	for i := 0; i < len(rows); i += batchSize {
		end := min(i+batchSize, len(rows))
		batch := rows[i:end]

		valueParts := make([]string, 0, len(batch))
		args := make([]any, 0, len(batch)*9)
		batchByKey := make(map[string]*fleet.MDMAppleHostDeclaration, len(batch))
		for _, r := range batch {
			valueParts = append(valueParts, "(?, ?, ?, ?, ?, ?, ?, ?, ?)")
			args = append(args,
				r.HostUUID, r.DeclarationUUID, r.Identifier, r.Name,
				r.Status, r.OperationType, r.Token, r.SecretsUpdatedAt, r.VariablesUpdatedAt,
			)
			batchByKey[fmt.Sprintf("%s\n%s", r.HostUUID, r.DeclarationUUID)] = r
		}

		stmt := fmt.Sprintf(baseStmt, strings.Join(valueParts, ","))
		err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			if _, ierr := tx.ExecContext(ctx, stmt, args...); ierr != nil {
				return ierr
			}
			return cleanUpDuplicateRemoveInstall(ctx, tx, batchByKey)
		})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "bulk upsert mdm apple host declarations")
		}
	}
	return nil
}
