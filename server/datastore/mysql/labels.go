package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ApplyLabelSpecs(ctx context.Context, specs []*fleet.LabelSpec) (err error) {
	return ds.ApplyLabelSpecsWithAuthor(ctx, specs, nil)
}

func (ds *Datastore) SetAsideLabels(ctx context.Context, notOnTeamID *uint, names []string, user fleet.User) error {
	if len(names) == 0 {
		return nil
	}

	type existingLabel struct {
		ID       uint  `db:"id"`
		AuthorID *uint `db:"author_id"`
		TeamID   *uint `db:"team_id"`
	}

	stmt := `SELECT id, author_id, team_id FROM labels WHERE name IN (?) AND label_type != ?`
	stmt, args, err := sqlx.In(stmt, names, uint(fleet.LabelTypeBuiltIn))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build labels query")
	}

	var labels []existingLabel
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &labels, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "query existing labels")
	}

	errCannotSetAside := ctxerr.New(ctx, "one or more specified labels to set aside do not exist or cannot be set aside")
	errGlobal := ctxerr.New(ctx, "one or more specified labels to set aside is on the same team as you are trying to modify")

	if len(labels) != len(names) {
		return errCannotSetAside
	}

	// Helper function to check if user has a global write role (admin, maintainer, or gitops)
	hasGlobalWriteRole := func() bool {
		if user.GlobalRole == nil {
			return false
		}
		return *user.GlobalRole == fleet.RoleAdmin ||
			*user.GlobalRole == fleet.RoleMaintainer ||
			*user.GlobalRole == fleet.RoleGitOps
	}

	// Helper function to check if user has a write role on any team
	hasWriteRoleAnywhere := func() bool {
		for _, team := range user.Teams {
			if team.Role == fleet.RoleAdmin ||
				team.Role == fleet.RoleMaintainer ||
				team.Role == fleet.RoleGitOps {
				return true
			}
		}
		return false
	}

	// Helper function to check if user has a write role on a specific team
	hasWriteRoleOnTeam := func(teamID uint) bool {
		for _, team := range user.Teams {
			if team.ID == teamID &&
				(team.Role == fleet.RoleAdmin ||
					team.Role == fleet.RoleMaintainer ||
					team.Role == fleet.RoleGitOps) {
				return true
			}
		}
		return false
	}

	for _, label := range labels {
		if label.TeamID == nil { // Global label
			if notOnTeamID == nil { // Disallow moving aside since the label is on the same team
				return errGlobal
			}

			if hasGlobalWriteRole() {
				continue
			}

			if hasWriteRoleAnywhere() && label.AuthorID != nil && *label.AuthorID == user.ID {
				continue
			}

			// User doesn't have permission to set aside this global label
			return errCannotSetAside
		}

		// Team label
		if notOnTeamID != nil && *notOnTeamID == *label.TeamID { // label is on the same team we're applying specs for
			return errCannotSetAside // generic error here because label may not be visible to the user
		}

		if hasGlobalWriteRole() || hasWriteRoleOnTeam(*label.TeamID) {
			continue
		}

		// User doesn't have permission to set aside this team label
		return errCannotSetAside
	}

	// Bulk update to rename labels by appending __team_{team_id} (or __team_0 for global labels)
	updateStmt := `UPDATE labels SET name = CONCAT(name, '__team_', COALESCE(team_id, 0)) WHERE name IN (?)`
	updateStmt, updateArgs, err := sqlx.In(updateStmt, names)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build update labels query")
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, updateStmt, updateArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "rename labels to set aside")
	}

	return nil
}

func (ds *Datastore) ApplyLabelSpecsWithAuthor(ctx context.Context, specs []*fleet.LabelSpec, authorID *uint) (err error) {
	// First, get existing labels to detect platform changes
	labelNames := make([]string, 0, len(specs))
	for _, s := range specs {
		if s.Name != "" {
			labelNames = append(labelNames, s.Name)
		}
	}

	type existingLabel struct {
		ID       uint   `db:"id"`
		Name     string `db:"name"`
		Platform string `db:"platform"`
		TeamID   *uint  `db:"team_id"`
	}
	existingLabels := make(map[string]existingLabel, len(specs))

	// NOTE: Thie assumes the caller has verified that label specs are all writable by the user, either for authorship
	// or team affiliation. We'll catch cases where a user is attempting to move the label between teams (which
	// should've been cleaned up by SetAsideLabels).

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// TODO: do we want to allow on duplicate updating label_type or
		// label_membership_type or should those always be immutable?
		// are we ok depending solely on the caller to ensure that these fields
		// are not changed?

		if len(labelNames) > 0 {
			stmt := `SELECT id, name, platform, team_id FROM labels WHERE name IN (?)`
			stmt, args, err := sqlx.In(stmt, labelNames)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "build existing labels query")
			}

			var labels []existingLabel
			if err := sqlx.SelectContext(ctx, tx, &labels, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "query existing labels")
			}

			for _, label := range labels {
				existingLabels[strings.ToLower(label.Name)] = label
			}

			for _, spec := range specs {
				if existingLabel, ok := existingLabels[strings.ToLower(spec.Name)]; ok &&
					(existingLabel.TeamID != nil && spec.TeamID == nil ||
						existingLabel.TeamID == nil && spec.TeamID != nil ||
						(existingLabel.TeamID != nil && spec.TeamID != nil && *existingLabel.TeamID != *spec.TeamID)) {
					return ctxerr.Wrap(ctx, err, "one or more specified labels exists on another team")
				}
			}
		}

		sql := `
		INSERT INTO labels (
			name,
			description,
			query,
			platform,
			label_type,
			label_membership_type,
			criteria,
			author_id,
			team_id
		) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ? )
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			description = VALUES(description),
			query = VALUES(query),
			platform = VALUES(platform),
			label_type = VALUES(label_type),
			label_membership_type = VALUES(label_membership_type),
			criteria = VALUES(criteria)
		`

		prepTx, ok := tx.(sqlx.PreparerContext)
		if !ok {
			return ctxerr.New(ctx, "tx in ApplyLabelSpecs is not a sqlx.PreparerContext")
		}
		stmt, err := prepTx.PrepareContext(ctx, sql)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "prepare ApplyLabelSpecs insert")
		}
		defer stmt.Close()

		for _, s := range specs {
			if s.Name == "" {
				return ctxerr.New(ctx, "label name must not be empty")
			}
			insertLabelResult, err := stmt.ExecContext(ctx, s.Name, s.Description, s.Query, s.Platform, s.LabelType, s.LabelMembershipType, s.HostVitalsCriteria, authorID, s.TeamID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "exec ApplyLabelSpecs insert")
			}

			// Check if this is an existing label and platform changed -> clean up memberships if needed
			if existing, ok := existingLabels[strings.ToLower(s.Name)]; ok && existing.Platform != s.Platform {
				// When a label's platform changes, we delete all existing memberships.
				// This ensures a clean slate - the label's query will be re-evaluated
				// by Fleet's label execution system, and only hosts matching the new
				// platform will be added back. This is simpler than trying to selectively
				// remove hosts and handles all edge cases consistently.
				cleanupSQL := `DELETE FROM label_membership WHERE label_id = ?`
				_, err = tx.ExecContext(ctx, cleanupSQL, existing.ID)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "cleanup label membership for platform change")
				}
			}

			if s.LabelType == fleet.LabelTypeBuiltIn ||
				s.LabelMembershipType != fleet.LabelMembershipTypeManual {
				// No need to update membership
				continue
			}

			// For manual labels, we need the label ID to update membership
			var labelID uint
			if existing, ok := existingLabels[strings.ToLower(s.Name)]; ok {
				// Use the existing label ID
				labelID = existing.ID
			} else {
				// New label - fetch the ID we just created
				id, err := insertLabelResult.LastInsertId()
				if err != nil {
					return ctxerr.Wrap(ctx, err, "get new label ID for manual membership")
				}
				labelID = uint(id) //nolint:gosec
			}

			sql = `
DELETE FROM label_membership WHERE label_id = ?
`
			_, err = tx.ExecContext(ctx, sql, labelID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "clear membership for ID")
			}

			if len(s.Hosts) == 0 {
				continue
			}

			intRegex := regexp.MustCompile(`^[0-9]+$`)
			// Split hostnames into batches to avoid parameter limit in MySQL.
			for _, hostIdentifiers := range batchHostnames(s.Hosts) {
				var stringIdents []string
				// Start with 0 so id IN (?) always has at least one element.
				// id = 0 never matches any real host.
				intIdents := []uint64{0}

				for _, s := range hostIdentifiers {
					stringIdents = append(stringIdents, s)
					// Use strconv to check if it's a valid integer
					if intRegex.MatchString(s) {
						n, _ := strconv.ParseUint(s, 10, 64)
						intIdents = append(intIdents, n)
					}
				}

				// Use ignore because duplicate hostnames could appear in
				// different batches and would result in duplicate key errors.
				sql = `
INSERT IGNORE INTO label_membership (label_id, host_id) (SELECT DISTINCT ?, id FROM hosts where hostname IN (?) OR hardware_serial IN (?) OR uuid IN (?) OR id IN (?))`
				sql, args, err := sqlx.In(sql, labelID, stringIdents, stringIdents, stringIdents, intIdents)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "build membership IN statement")
				}
				_, err = tx.ExecContext(ctx, sql, args...)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "execute membership INSERT")
				}
			}
		}

		return nil
	})

	return ctxerr.Wrap(ctx, err, "ApplyLabelSpecs transaction")
}

func batchHostnames(hostnames []string) [][]string {
	// Split hostnames into batches so that they can all be inserted without
	// overflowing the MySQL max number of parameters (somewhere around 65,000
	// but not well documented). Algorithm from
	// https://github.com/golang/go/wiki/SliceTricks#batching-with-minimal-allocation
	//
	// WARNING: This is used in ApplyLabelSpecsWithAuthor and the batch sizes have to be small
	// enough to allow for three copies each hostname list in the query. The batch size is 15_000
	// because 60_001 binding arguments is less than the maximum of 65,535.

	const batchSize = 15_000 // Large, but well under the undocumented limit
	batches := make([][]string, 0, (len(hostnames)+batchSize-1)/batchSize)

	for batchSize < len(hostnames) {
		hostnames, batches = hostnames[batchSize:], append(batches, hostnames[0:batchSize:batchSize])
	}
	batches = append(batches, hostnames)
	return batches
}

func (ds *Datastore) UpdateLabelMembershipByHostIDs(ctx context.Context, label fleet.Label, hostIds []uint, teamFilter fleet.TeamFilter) (*fleet.Label, []uint, error) {
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// delete all label membership
		sql := `
	DELETE FROM label_membership WHERE label_id = ?
	`
		_, err := tx.ExecContext(ctx, sql, label.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "clear membership for ID")
		}

		if len(hostIds) == 0 {
			return nil
		}

		// Split hostIds into batches to avoid parameter limit in MySQL.
		for _, hostIds := range batchHostIds(hostIds) {
			if label.TeamID != nil { // team labels can only be applied to hosts on that team
				hostTeamCheckSql := `SELECT COUNT(id) FROM hosts WHERE (team_id != ? OR team_id IS NULL) AND id IN (?)`
				hostTeamCheckSql, args, err := sqlx.In(hostTeamCheckSql, label.TeamID, hostIds)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "build host team membership check IN statement")
				}

				var hostCountOnWrongTeam int
				if err := tx.QueryRowxContext(ctx, hostTeamCheckSql, args...).Scan(&hostCountOnWrongTeam); err != nil {
					return ctxerr.Wrap(ctx, err, "execute host team membership check query")
				}
				if hostCountOnWrongTeam > 0 {
					return ctxerr.Wrap(ctx, errors.New("supplied hosts are on a different team than the label"))
				}
			}

			// Use ignore because duplicate hostIds could appear in
			// different batches and would result in duplicate key errors.
			var values []any
			var placeholders []string

			for _, hostID := range hostIds {
				values = append(values, label.ID, hostID)
				placeholders = append(placeholders, "(?, ?)")
			}

			// Build the final SQL query with the dynamically generated placeholders
			sql := `
INSERT IGNORE INTO label_membership (label_id, host_id)
VALUES ` + strings.Join(placeholders, ", ")
			sql, args, err := sqlx.In(sql, values...)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "build membership IN statement")
			}
			_, err = tx.ExecContext(ctx, sql, args...)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "execute membership INSERT")
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "UpdateLabelMembershipByHostIDs transaction")
	}

	updatedLabel, hostIDs, err := ds.labelDB(ctx, label.ID, teamFilter, ds.writer(ctx))
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "UpdateLabelMembershipByHostIDs get label after update")
	}

	return updatedLabel.GetLabel(), hostIDs, err
}

// Update label membership for a host vitals label.
func (ds *Datastore) UpdateLabelMembershipByHostCriteria(ctx context.Context, hvl fleet.HostVitalsLabel) (*fleet.Label, error) {
	// Get the label data.
	label := hvl.GetLabel()

	// If the label isn't a host vitals label, bail out.
	if label.LabelMembershipType != fleet.LabelMembershipTypeHostVitals {
		return nil, ctxerr.New(ctx, "label is not a host vitals label")
	}

	// Get the query and value params for the host vitals label.
	query, queryVals, err := hvl.CalculateHostVitalsQuery()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "calculating host vitals query")
	}
	if query == "" {
		return nil, ctxerr.New(ctx, "label query is empty after calculating host vitals query")
	}

	labelSelect := fmt.Sprintf("%d as label_id, hosts.id as host_id", label.ID)
	labelQuery := fmt.Sprintf(query, labelSelect, "hosts")
	if label.TeamID != nil {
		labelQuery = fmt.Sprintf(query, labelSelect, fmt.Sprintf("hosts JOIN (SELECT %d team_id) label_team ON label_team.team_id = hosts.team_id", *label.TeamID))
	}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Insert new label membership based on the label query.
		sql := fmt.Sprintf(`INSERT INTO label_membership (label_id, host_id) SELECT candidate.label_id, candidate.host_id FROM (%s) as candidate ON DUPLICATE KEY UPDATE host_id = label_membership.host_id`, labelQuery)
		_, err := tx.ExecContext(ctx, sql, queryVals...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "execute membership INSERT")
		}

		// Remove any existing label membership for the label that is not in the new query.
		sql = fmt.Sprintf(`DELETE FROM label_membership WHERE label_id = %d AND NOT EXISTS (SELECT 1 FROM (%s) as candidate WHERE candidate.host_id = label_membership.host_id)`, label.ID, labelQuery)
		_, err = tx.ExecContext(ctx, sql, queryVals...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "execute membership DELETE")
		}

		// Get the new number of members.
		sql = `SELECT COUNT(*) FROM label_membership WHERE label_id = ?`
		var count int
		if err := sqlx.GetContext(ctx, tx, &count, sql, label.ID); err != nil {
			return ctxerr.Wrap(ctx, err, "get label membership count")
		}
		label.HostCount = count
		return nil
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "UpdateLabelMembershipByHostCriteria transaction")
	}

	return label, err
}

func batchHostIds(hostIds []uint) [][]uint {
	// same functionality as `batchHostnames`, but for host IDs
	const batchSize = 50000 // Large, but well under the undocumented limit
	batches := make([][]uint, 0, (len(hostIds)+batchSize-1)/batchSize)

	for batchSize < len(hostIds) {
		hostIds, batches = hostIds[batchSize:], append(batches, hostIds[0:batchSize:batchSize])
	}
	batches = append(batches, hostIds)
	return batches
}

func (ds *Datastore) GetLabelSpecs(ctx context.Context, filter fleet.TeamFilter) ([]*fleet.LabelSpec, error) {
	var specs []*fleet.LabelSpec
	// Get basic specs
	query, params, err := applyLabelTeamFilter(`SELECT id, name, description, query, platform,
       label_type, label_membership_type, criteria, team_id
		FROM labels l`, filter)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query for getting label specs")
	}
	// Normally, we want to show all available labels for e.g. applying to a resource, but for specs it's
	// better to show only the labels on a given team when a filter is applied. Doing this query hack rather
	// than editing the applyLabelTeamFilter implementation to avoid adding a flag that's only set here.
	if filter.TeamID != nil && *filter.TeamID > 0 {
		query += " AND l.team_id IS NOT NULL"
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &specs, query, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get labels")
	}

	for _, spec := range specs {
		if spec.LabelType != fleet.LabelTypeBuiltIn &&
			spec.LabelMembershipType == fleet.LabelMembershipTypeManual {
			if err := ds.getLabelHostIDs(ctx, spec); err != nil {
				return nil, err
			}
		}
	}

	return specs, nil
}

func (ds *Datastore) GetLabelSpec(ctx context.Context, filter fleet.TeamFilter, name string) (*fleet.LabelSpec, error) {
	var specs []*fleet.LabelSpec
	query, params, err := applyLabelTeamFilter(`
SELECT l.id, l.name, l.description, l.query, l.platform, l.label_type, l.label_membership_type
FROM labels l
WHERE l.name = ?`, filter, name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query for getting label spec")
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &specs, query, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get label")
	}
	if len(specs) == 0 {
		return nil, ctxerr.Wrap(ctx, notFound("Label").WithName(name))
	}
	if len(specs) > 1 {
		return nil, ctxerr.Errorf(ctx, "expected 1 label row, got %d", len(specs))
	}

	spec := specs[0]
	if spec.LabelType != fleet.LabelTypeBuiltIn &&
		spec.LabelMembershipType == fleet.LabelMembershipTypeManual {
		err := ds.getLabelHostIDs(ctx, spec)
		if err != nil {
			return nil, err
		}
	}

	return spec, nil
}

func (ds *Datastore) getLabelHostIDs(ctx context.Context, label *fleet.LabelSpec) error {
	sql := `
		SELECT id
		FROM hosts
		WHERE id IN
		(
			SELECT host_id
			FROM label_membership
			WHERE label_id = (SELECT id FROM labels WHERE name = ?)
		)
	`
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &label.Hosts, sql, label.Name)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get hostnames for label")
	}
	return nil
}

// NewLabel creates a new fleet.Label
func (ds *Datastore) NewLabel(ctx context.Context, label *fleet.Label, opts ...fleet.OptionalArg) (*fleet.Label, error) {
	query := `
	INSERT INTO labels (
		name,
		description,
		query,
		criteria,
		platform,
		label_type,
		label_membership_type,
		author_id,
		team_id
	) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ? )
	`
	result, err := ds.writer(ctx).ExecContext(
		ctx,
		query,
		label.Name,
		label.Description,
		label.Query,
		label.HostVitalsCriteria,
		label.Platform,
		label.LabelType,
		label.LabelMembershipType,
		label.AuthorID,
		label.TeamID,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting label")
	}

	id, _ := result.LastInsertId()
	label.ID = uint(id) //nolint:gosec // dismiss G115
	return label, nil
}

func (ds *Datastore) SaveLabel(ctx context.Context, label *fleet.Label, teamFilter fleet.TeamFilter) (*fleet.LabelWithTeamName, []uint, error) {
	query := `UPDATE labels SET name = ?, description = ? WHERE id = ?`
	_, err := ds.writer(ctx).ExecContext(ctx, query, label.Name, label.Description, label.ID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "saving label")
	}

	// Update the label name in mdm_configuration_profile_labels
	query = `UPDATE mdm_configuration_profile_labels SET label_name = ? WHERE label_id = ?`
	_, err = ds.writer(ctx).ExecContext(ctx, query, label.Name, label.ID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "updating mdm configuration profile label")
	}

	return ds.labelDB(ctx, label.ID, teamFilter, ds.writer(ctx))
}

// DeleteLabel deletes a fleet.Label
func (ds *Datastore) DeleteLabel(ctx context.Context, name string, filter fleet.TeamFilter) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var labelID uint

		query, params, err := applyLabelTeamFilter(`select l.id FROM labels l WHERE l.name = ?`, filter, name)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting label id to delete")
		}

		err = sqlx.GetContext(ctx, tx, &labelID, query, params...)
		if err != nil {
			if err == sql.ErrNoRows {
				return ctxerr.Wrap(ctx, notFound("Label").WithName(name))
			}
			return ctxerr.Wrap(ctx, err, "getting label id to delete")
		}
		if err := deleteLabelsInTx(ctx, tx, []uint{labelID}); err != nil {
			if isMySQLForeignKey(err) {
				return ctxerr.Wrap(ctx, foreignKey("labels", name), "delete label")
			}
			return ctxerr.Wrap(ctx, err, "delete labels in tx")
		}
		return nil
	})
}

func deleteLabelsInTx(ctx context.Context, tx sqlx.ExtContext, labelIDs []uint) error {
	query, args, err := sqlx.In(`DELETE FROM labels WHERE id IN (?)`, labelIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build sqlx.In statement for labels")
	}
	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "delete label")
	}

	query, args, err = sqlx.In(`DELETE FROM label_membership WHERE label_id IN (?)`, labelIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build sqlx.In statement for label_membership")
	}
	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "delete label_membership")
	}

	query, args, err = sqlx.In(`DELETE FROM pack_targets WHERE type = ? AND target_id IN (?)`, fleet.TargetLabel, labelIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build sqlx.In statement for pack_targets")
	}
	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting pack_targets")
	}

	return nil
}

// LabelByName returns a fleet.Label identified by name if one exists and is accessible to the specified user.
func (ds *Datastore) LabelByName(ctx context.Context, name string, teamFilter fleet.TeamFilter) (*fleet.Label, error) {
	stmt, params, err := applyLabelTeamFilter("SELECT l.* FROM labels l WHERE l.name = ?", teamFilter, name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building label select query")
	}

	var label fleet.Label
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &label, stmt, params...); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Label").WithName(name))
		}
		return nil, ctxerr.Wrap(ctx, err, "selecting label")
	}

	return &label, nil
}

// Label returns a fleet.LabelWithTeamName identified by lid if one exists and is accessible to the specified user.
func (ds *Datastore) Label(ctx context.Context, lid uint, teamFilter fleet.TeamFilter) (*fleet.LabelWithTeamName, []uint, error) {
	return ds.labelDB(ctx, lid, teamFilter, ds.reader(ctx))
}

func (ds *Datastore) labelDB(ctx context.Context, lid uint, teamFilter fleet.TeamFilter, q sqlx.QueryerContext) (*fleet.LabelWithTeamName, []uint, error) {
	stmt := fmt.Sprintf(`
		SELECT
		       l.*, teams.name team_name,
		       (SELECT COUNT(1) FROM label_membership lm JOIN hosts h ON (lm.host_id = h.id) WHERE lm.label_id = l.id AND %s) AS host_count
		FROM labels l LEFT JOIN teams ON teams.id = l.team_id
		WHERE l.id = ?
	`, ds.whereFilterHostsByTeams(teamFilter, "h"))

	stmt, params, err := applyLabelTeamFilter(stmt, teamFilter, lid)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "building label select query")
	}

	var label fleet.LabelWithTeamName
	if err := sqlx.GetContext(ctx, q, &label, stmt, params...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ctxerr.Wrap(ctx, notFound("Label").WithID(lid))
		}
		return nil, nil, ctxerr.Wrap(ctx, err, "selecting label")
	}

	var hostIDs []uint
	if label.LabelMembershipType == fleet.LabelMembershipTypeManual && label.HostCount > 0 {
		if err := sqlx.SelectContext(ctx, q, &hostIDs, `SELECT host_id FROM label_membership WHERE label_id = ?`, lid); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "selecting label host IDs")
		}
	}

	return &label, hostIDs, nil
}

// ListLabels returns all labels limited or sorted by fleet.ListOptions.
// MatchQuery not supported
func (ds *Datastore) ListLabels(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions, includeHostCounts bool) ([]*fleet.Label, error) {
	if opt.After != "" {
		return nil, &fleet.BadRequestError{Message: "parameter 'after' is not supported"}
	}
	if opt.MatchQuery != "" {
		return nil, &fleet.BadRequestError{Message: "parameter 'query' is not supported"}
	}

	query := "SELECT l.* FROM labels l "
	// When applicable, filter host membership by team and return counts with the labels.
	if filter.User != nil && includeHostCounts {
		query = fmt.Sprintf(`
				SELECT l.*,
					(SELECT COUNT(1)
					 FROM label_membership lm
					     JOIN hosts h ON (lm.host_id = h.id) WHERE label_id = l.id AND %s
					 ) AS host_count
				FROM labels l
			`, ds.whereFilterHostsByTeams(filter, "h"),
		)
	}

	query, params, err := applyLabelTeamFilter(query, filter)
	if err != nil {
		return nil, err
	}

	query, params = appendListOptionsWithCursorToSQL(query, params, &opt)
	var labels []*fleet.Label

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, query, params...); err != nil {
		// it's ok if no labels exist
		if err == sql.ErrNoRows {
			return labels, nil
		}

		return nil, ctxerr.Wrap(ctx, err, "selecting labels")
	}

	return labels, nil
}

var errInaccessibleTeam = errors.New("The team ID you provided refers to a team that either does not exist or you do not have permission to access.")

// applyLabelTeamFilter requires the labels table to be aliased as "l" to work
func applyLabelTeamFilter(query string, filter fleet.TeamFilter, initialParams ...any) (string, []any, error) {
	// using this rather than a "contains a WHERE" check because some queries have subqueries
	// but don't have any parameters for those subqueries
	whereOrAnd := " WHERE "
	if len(initialParams) > 0 {
		whereOrAnd = " AND "
	}

	// apply sqlx.In if we had initial params, as they may include slices for where-ins other than the team one
	maybeIn := func(query string) (string, []any, error) {
		if len(initialParams) > 0 {
			return sqlx.In(query, initialParams...)
		}
		return query, nil, nil
	}

	if filter.User == nil { // fall back to safe (global-only) filter if this happens (it shouldn't)
		return maybeIn(query + whereOrAnd + " l.team_id IS NULL")
	}

	if filter.TeamID != nil {
		if *filter.TeamID == 0 { // global labels only; any user can see them
			return maybeIn(query + whereOrAnd + "l.team_id IS NULL")
		} else if !filter.UserCanAccessSelectedTeam() {
			return "", nil, fleet.NewUserMessageError(errInaccessibleTeam, 403)
		} // else user can see the team labels they're asking for; return global labels plus that team's labels

		return sqlx.In(query+whereOrAnd+"(l.team_id IS NULL OR l.team_id = ?)", append(initialParams, *filter.TeamID)...)
	}

	if !filter.User.HasAnyGlobalRole() && filter.User.HasAnyTeamRole() { // filter to teams user can see
		return sqlx.In(query+whereOrAnd+"(l.team_id IS NULL OR l.team_id IN (?))", append(initialParams, filter.User.TeamIDsWithAnyRole())...)
	} // else user exists and has a global role, so we don't need to filter out any team labels

	return maybeIn(query)
}

func platformForHost(host *fleet.Host) string {
	if host.Platform != "rhel" {
		return host.Platform
	}
	if strings.Contains(strings.ToLower(host.OSVersion), "centos") {
		return "centos"
	}
	return host.Platform
}

func (ds *Datastore) LabelQueriesForHost(ctx context.Context, host *fleet.Host) (map[string]string, error) {
	var rows *sql.Rows
	var err error
	platform := platformForHost(host)
	query := `SELECT id, query FROM labels WHERE
		(platform = ? OR platform = '') AND
		label_membership_type = ? AND
		(team_id IS NULL OR team_id = ?)`
	rows, err = ds.reader(ctx).QueryContext(ctx, query, platform, fleet.LabelMembershipTypeDynamic, host.TeamID)
	if err != nil && err != sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, err, "selecting label queries for host")
	}
	defer rows.Close()

	results := map[string]string{}
	for rows.Next() {
		var id, query string
		if err = rows.Scan(&id, &query); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "scanning label queries for host")
		}
		results[id] = query
	}
	if err := rows.Err(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "iterating over returned rows")
	}

	return results, nil
}

func (ds *Datastore) RecordLabelQueryExecutions(ctx context.Context, host *fleet.Host, results map[uint]*bool, updated time.Time, deferredSaveHost bool) error {
	// Sort the results to have generated SQL queries ordered to minimize
	// deadlocks. See https://github.com/fleetdm/fleet/issues/1146.
	orderedIDs := make([]uint, 0, len(results))
	for labelID := range results {
		orderedIDs = append(orderedIDs, labelID)
	}
	sort.Slice(orderedIDs, func(i, j int) bool { return orderedIDs[i] < orderedIDs[j] })

	// Loop through results, collecting which labels we need to insert/update,
	// and which we need to delete
	vals := []interface{}{}
	bindvars := []string{}
	removes := []uint{}
	for _, labelID := range orderedIDs {
		matches := results[labelID]
		if matches != nil && *matches {
			// Add/update row
			bindvars = append(bindvars, "(?,?,?)")
			vals = append(vals, updated, labelID, host.ID)
		} else {
			// Delete row
			removes = append(removes, labelID)
		}
	}

	// NOTE: the insert/delete of label membership that follows must be kept in
	// sync with the async implementations in
	// AsyncBatch{Insert,Delete}LabelMembership, and the update of the
	// label_updated_at timestamp in sync with the
	// AsyncBatchUpdateLabelTimestamp method (that is, their processing must be
	// semantically equivalent, even though here it processes a single host and
	// in async mode it processes a batch of hosts).

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Complete inserts if necessary
		if len(vals) > 0 {
			sql := `INSERT INTO label_membership (updated_at, label_id, host_id) VALUES `
			sql += strings.Join(bindvars, ",") + ` ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at)`

			_, err := tx.ExecContext(ctx, sql, vals...)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "insert label query executions (%v)", vals)
			}
		}

		// Complete deletions if necessary
		if len(removes) > 0 {
			sql := `DELETE FROM label_membership WHERE host_id = ? AND label_id IN (?)`
			query, args, err := sqlx.In(sql, host.ID, removes)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "IN for DELETE FROM label_membership")
			}
			query = tx.Rebind(query)
			_, err = tx.ExecContext(ctx, query, args...)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "delete label query executions")
			}
		}

		// if we are deferring host updates, we return at this point and do the change outside of the tx
		if deferredSaveHost {
			return nil
		}

		_, err := tx.ExecContext(ctx, `UPDATE hosts SET label_updated_at = ? WHERE id=?`, host.LabelUpdatedAt, host.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "updating hosts label updated at")
		}

		return nil
	})
	if err != nil {
		return err
	}

	if deferredSaveHost {
		errCh := make(chan error, 1)
		defer close(errCh)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ds.writeCh <- itemToWrite{
			ctx:   ctx,
			errCh: errCh,
			item: hostXUpdatedAt{
				hostID:    host.ID,
				updatedAt: updated,
				what:      "label_updated_at",
			},
		}:
			return <-errCh
		}
	}
	return nil
}

// ListLabelsForHost returns a list of fleet.Label for a given host id.
func (ds *Datastore) ListLabelsForHost(ctx context.Context, hid uint) ([]*fleet.Label, error) {
	sqlStatement := `
		SELECT labels.* from labels JOIN label_membership lm
		WHERE lm.host_id = ?
		AND lm.label_id = labels.id
	`

	labels := []*fleet.Label{}
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, sqlStatement, hid)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting host labels")
	}

	return labels, nil
}

// ListHostsInLabel returns a list of fleet.Host that are associated
// with fleet.Label referenced by Label ID
func (ds *Datastore) ListHostsInLabel(ctx context.Context, filter fleet.TeamFilter, lid uint, opt fleet.HostListOptions) ([]*fleet.Host, error) {
	labelCheckSql, labelCheckParams, err := applyLabelTeamFilter(`SELECT l.id FROM labels l WHERE id = ?`, filter, lid)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query to confirm label existence")
	}

	var foundID uint
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &foundID, labelCheckSql, labelCheckParams...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // matches previous behavior (invalid labels return no hosts)
		}
		return nil, ctxerr.Wrap(ctx, err, "confirming label existence")
	}

	queryFmt := `
    SELECT
      h.id,
      h.osquery_host_id,
      h.created_at,
      h.updated_at,
      h.detail_updated_at,
      h.node_key,
      h.hostname,
      h.uuid,
      h.platform,
      h.osquery_version,
      h.os_version,
      h.build,
      h.platform_like,
      h.code_name,
      h.uptime,
      h.memory,
      h.cpu_type,
      h.cpu_subtype,
      h.cpu_brand,
      h.cpu_physical_cores,
      h.cpu_logical_cores,
      h.hardware_vendor,
      h.hardware_model,
      h.hardware_version,
      h.hardware_serial,
      h.computer_name,
      h.primary_ip_id,
      h.distributed_interval,
      h.logger_tls_period,
      h.config_tls_refresh,
      h.primary_ip,
      h.primary_mac,
      h.label_updated_at,
      h.last_enrolled_at,
      h.refetch_requested,
      h.refetch_critical_queries_until,
      h.team_id,
      h.policy_updated_at,
      h.public_ip,
      COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
      COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available,
      COALESCE(hd.gigs_total_disk_space, 0) as gigs_total_disk_space,
      COALESCE(hst.seen_time, h.created_at) as seen_time,
      COALESCE(hu.software_updated_at, h.created_at) AS software_updated_at,
      h.last_restarted_at,
      (SELECT name FROM teams t WHERE t.id = h.team_id) AS team_name
      %s
      %s
			%s
    FROM label_membership lm
    JOIN hosts h ON (lm.host_id = h.id)
    LEFT JOIN host_seen_times hst ON (h.id=hst.host_id)
    LEFT JOIN host_updates hu ON (h.id = hu.host_id)
    LEFT JOIN host_disks hd ON (h.id=hd.host_id)
    %s
    %s
		%s
`
	failingIssuesSelect := `,
		COALESCE(host_issues.failing_policies_count, 0) AS failing_policies_count,
		COALESCE(host_issues.critical_vulnerabilities_count, 0) AS critical_vulnerabilities_count,
		COALESCE(host_issues.total_issues_count, 0) AS total_issues_count
	`
	failingIssuesJoin := `LEFT JOIN host_issues ON h.id = host_issues.host_id`

	if opt.DisableIssues {
		failingIssuesSelect = ""
		failingIssuesJoin = ""
	}

	deviceMappingJoin := fmt.Sprintf(`LEFT JOIN (
	SELECT
		host_id,
		CONCAT('[', GROUP_CONCAT(JSON_OBJECT('email', email, 'source', %s)), ']') AS device_mapping
	FROM
		host_emails
	GROUP BY
		host_id) dm ON dm.host_id = h.id`, deviceMappingTranslateSourceColumn(""))
	if !opt.DeviceMapping {
		deviceMappingJoin = ""
	}

	var deviceMappingSelect string
	if opt.DeviceMapping {
		deviceMappingSelect = `,
	COALESCE(dm.device_mapping, 'null') as device_mapping`
	}

	query := fmt.Sprintf(
		queryFmt, hostMDMSelect, failingIssuesSelect, deviceMappingSelect, hostMDMJoin, failingIssuesJoin, deviceMappingJoin,
	)

	query, params, err := ds.applyHostLabelFilters(ctx, filter, lid, query, opt)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "applying label query filters")
	}

	hosts := []*fleet.Host{}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, query, params...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting label query executions")
	}
	return hosts, nil
}

// NOTE: the hosts table must be aliased to `h` in the query passed to this function.
func (ds *Datastore) applyHostLabelFilters(ctx context.Context, filter fleet.TeamFilter, lid uint, query string, opt fleet.HostListOptions) (string, []interface{}, error) {
	// prior to returning, params will be appended in the following order: joinParams, whereParams
	var whereParams, joinParams []interface{}

	if opt.ListOptions.OrderKey == "display_name" {
		query += ` JOIN host_display_names hdn ON h.id = hdn.host_id `
	}

	softwareStatusJoin := ""
	// if opt.SoftwareVersionIDFilter != nil {
	// 	// TODO: Do we currently support filtering by software version ID and label?
	// } else if opt.SoftwareIDFilter != nil {
	// 	// TODO: Do we currently support filtering by software version ID and label?
	// }
	if opt.SoftwareTitleIDFilter != nil && opt.SoftwareStatusFilter != nil {
		installerID, vppID, inHouseID, err := ds.installerAvailableForInstallForTeamAndTitleID(ctx, opt.TeamFilter, *opt.SoftwareTitleIDFilter)
		switch {
		case err != nil:
			// it does not return an error for not found, only for actual db error
			return "", nil, ctxerr.Wrap(ctx, err, "get available installer by team and title id")

		case installerID > 0:
			// found a software installer package
			installerJoin, installerParams, err := ds.softwareInstallerJoin(*opt.SoftwareTitleIDFilter, *opt.SoftwareStatusFilter)
			if err != nil {
				return "", nil, ctxerr.Wrap(ctx, err, "software installer join")
			}
			softwareStatusJoin = installerJoin
			joinParams = append(joinParams, installerParams...)

		case vppID != nil:
			// found a VPP app
			vppAppJoin, vppAppParams, err := ds.vppAppJoin(*vppID, *opt.SoftwareStatusFilter)
			if err != nil {
				return "", nil, ctxerr.Wrap(ctx, err, "vpp app join")
			}
			softwareStatusJoin = vppAppJoin
			joinParams = append(joinParams, vppAppParams...)

		case inHouseID > 0:
			inHouseJoin, inHouseParams, err := ds.inHouseAppJoin(inHouseID, *opt.SoftwareStatusFilter)
			if err != nil {
				return "", nil, ctxerr.Wrap(ctx, err, "in-house app join")
			}
			softwareStatusJoin = inHouseJoin
			joinParams = append(joinParams, inHouseParams...)

		default:
			// no installer found, return as was done before (which was a not-found error, here, unlike in applyHostsFilter)
			return "", nil, ctxerr.Wrap(ctx, notFound("installerAvailableForInstall"), "get available software installer by team and title id")
		}
	}
	if softwareStatusJoin != "" {
		query += softwareStatusJoin
	}

	if opt.ConnectedToFleetFilter != nil && *opt.ConnectedToFleetFilter ||
		opt.OSSettingsFilter.IsValid() ||
		opt.MacOSSettingsFilter.IsValid() ||
		opt.MacOSSettingsDiskEncryptionFilter.IsValid() ||
		opt.OSSettingsDiskEncryptionFilter.IsValid() {
		query += `
		  LEFT JOIN nano_enrollments ne ON ne.id = h.uuid AND ne.enabled = 1 AND ne.type IN ('Device', 'User Enrollment (Device)')
		  LEFT JOIN mdm_windows_enrollments mwe ON mwe.host_uuid = h.uuid AND mwe.device_state = ?
		  LEFT JOIN android_devices ad ON ad.host_id = h.id`
		joinParams = append(joinParams, microsoft_mdm.MDMDeviceStateEnrolled)
	}

	if opt.OSSettingsFilter.IsValid() ||
		opt.MacOSSettingsFilter.IsValid() {
		query += sqlJoinMDMAppleProfilesStatus()
		query += sqlJoinMDMAppleDeclarationsStatus()
	}

	if opt.OSSettingsFilter.IsValid() {
		query += sqlJoinMDMAndroidProfilesStatus()
	}

	query += fmt.Sprintf(` WHERE lm.label_id = ? AND %s `, ds.whereFilterHostsByTeams(filter, "h"))
	whereParams = append(whereParams, lid)

	if opt.LowDiskSpaceFilter != nil {
		query += ` AND hd.gigs_disk_space_available < ? `
		whereParams = append(whereParams, *opt.LowDiskSpaceFilter)
	}

	var err error
	query, whereParams = filterHostsByStatus(ds.clock.Now(), query, opt, whereParams)
	query, whereParams = filterHostsByTeam(query, opt, whereParams)
	query, whereParams = filterHostsByMDM(query, opt, whereParams)
	query, whereParams, err = filterHostsByMacOSSettingsStatus(query, opt, whereParams)
	if err != nil {
		return "", nil, ctxerr.Wrap(ctx, err, "building macOS settings status filter")
	}
	query, whereParams = filterHostsByMacOSDiskEncryptionStatus(query, opt, whereParams)
	query, whereParams = filterHostsByMDMBootstrapPackageStatus(query, opt, whereParams)
	if diskEncryptionConfig, err := ds.GetConfigEnableDiskEncryption(ctx, opt.TeamFilter); err != nil {
		return "", nil, err
	} else if opt.OSSettingsFilter.IsValid() {
		query, whereParams, err = ds.filterHostsByOSSettingsStatus(query, opt, whereParams, diskEncryptionConfig)
		if err != nil {
			return "", nil, err
		}
	} else if opt.OSSettingsDiskEncryptionFilter.IsValid() {
		query, whereParams = ds.filterHostsByOSSettingsDiskEncryptionStatus(query, opt, whereParams, diskEncryptionConfig)
	}
	// TODO: should search columns include display_name (requires join to host_display_names)?
	query, whereParams, _ = hostSearchLike(query, whereParams, opt.MatchQuery, hostSearchColumns...)

	if opt.ListOptions.OrderKey == "issues" {
		opt.ListOptions.OrderKey = "host_issues.total_issues_count"
	}
	query, whereParams = appendListOptionsWithCursorToSQL(query, whereParams, &opt.ListOptions)
	return query, append(joinParams, whereParams...), nil
}

func (ds *Datastore) CountHostsInLabel(ctx context.Context, filter fleet.TeamFilter, lid uint, opt fleet.HostListOptions) (int, error) {
	query := `SELECT count(*) FROM label_membership lm
    JOIN hosts h ON (lm.host_id = h.id)
	LEFT JOIN host_seen_times hst ON (h.id=hst.host_id)
	LEFT JOIN host_disks hd ON (h.id=hd.host_id)
 	`

	query += hostMDMJoin

	query, params, err := ds.applyHostLabelFilters(ctx, filter, lid, query, opt)
	if err != nil {
		return 0, err
	}

	var count int
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &count, query, params...); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count hosts")
	}

	return count, nil
}

func (ds *Datastore) searchLabelsWithOmits(ctx context.Context, filter fleet.TeamFilter, query string, omit ...uint) ([]*fleet.Label, error) {
	sqlStatement := fmt.Sprintf(`
			SELECT l.*,
				(SELECT COUNT(1)
					FROM label_membership lm JOIN hosts h ON (lm.host_id = h.id)
					WHERE label_id = l.id AND %s
				) AS host_count
			FROM labels l
			WHERE (
				MATCH(l.name) AGAINST(? IN BOOLEAN MODE)
			)
			AND l.id NOT IN (?)
		`, ds.whereFilterHostsByTeams(filter, "h"),
	)

	sql, args, err := applyLabelTeamFilter(sqlStatement, filter, transformQuery(query), omit)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query for labels with omits")
	}

	sql += ` ORDER BY label_type DESC, id ASC`

	sql = ds.reader(ctx).Rebind(sql)

	matches := []*fleet.Label{}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &matches, sql, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting labels with omits")
	}

	matches, err = ds.addAllHostsLabelToList(ctx, filter, matches, omit...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "adding all hosts label to matches")
	}

	return matches, nil
}

// When we search labels, we always want to make sure that the All Hosts label
// is included in the results set. Sometimes it already is and we don't need to
// add it, sometimes it's not so we explicitly add it.
func (ds *Datastore) addAllHostsLabelToList(ctx context.Context, filter fleet.TeamFilter, labels []*fleet.Label, omit ...uint) ([]*fleet.Label, error) {
	sql := fmt.Sprintf(`
			SELECT *,
				(SELECT COUNT(1)
					FROM label_membership lm JOIN hosts h ON (lm.host_id = h.id)
					WHERE label_id = l.id AND %s
				) AS host_count
			FROM labels l
			WHERE
			  label_type=?
				AND name = 'All Hosts'
			LIMIT 1
		`, ds.whereFilterHostsByTeams(filter, "h"),
	)

	var allHosts fleet.Label
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &allHosts, sql, fleet.LabelTypeBuiltIn); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get all hosts label")
	}

	for _, omission := range omit {
		if omission == allHosts.ID {
			return labels, nil
		}
	}

	for _, label := range labels {
		if label.ID == allHosts.ID {
			return labels, nil
		}
	}

	return append(labels, &allHosts), nil
}

func (ds *Datastore) searchLabelsDefault(ctx context.Context, filter fleet.TeamFilter, omit ...uint) ([]*fleet.Label, error) {
	sql := fmt.Sprintf(`
			SELECT l.*,
				(SELECT COUNT(1)
					FROM label_membership lm JOIN hosts h ON (lm.host_id = h.id)
					WHERE label_id = l.id AND %s
				) AS host_count
			FROM labels l
			WHERE l.id NOT IN (?)
		`, ds.whereFilterHostsByTeams(filter, "h"),
	)

	var in interface{}
	{
		// use -1 if there are no values to omit.
		// Avoids empty args error for `sqlx.In`
		in = omit
		if len(omit) == 0 {
			in = -1
		}
	}

	var labels []*fleet.Label
	sql, args, err := applyLabelTeamFilter(sql, filter, in)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "searching default labels")
	}
	sql += ` GROUP BY id ORDER BY label_type DESC, id ASC`

	sql = ds.reader(ctx).Rebind(sql)
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "searching default labels rebound")
	}

	labels, err = ds.addAllHostsLabelToList(ctx, filter, labels, omit...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting all host label")
	}

	return labels, nil
}

// SearchLabels performs wildcard searches on fleet.Label name
func (ds *Datastore) SearchLabels(ctx context.Context, filter fleet.TeamFilter, query string, omit ...uint) ([]*fleet.Label, error) {
	transformedQuery := transformQuery(query)
	if !queryMinLength(transformedQuery) {
		return ds.searchLabelsDefault(ctx, filter, omit...)
	}
	if len(omit) > 0 {
		return ds.searchLabelsWithOmits(ctx, filter, query, omit...)
	}

	// Ordering first by label_type ensures that built-in labels come
	// first. We will probably need to make a custom ordering function here
	// if additional label types are added. Ordering next by ID ensures
	// that the order is always consistent.
	sql := fmt.Sprintf(`
			SELECT l.*,
				(SELECT COUNT(1)
						FROM label_membership lm JOIN hosts h ON (lm.host_id = h.id)
						WHERE label_id = l.id AND %s
					) AS host_count
				FROM labels l
			WHERE (
				MATCH(name) AGAINST(? IN BOOLEAN MODE)
			)
		`, ds.whereFilterHostsByTeams(filter, "h"),
	)

	sql, args, err := applyLabelTeamFilter(sql, filter, transformQuery(query))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query for searching labels")
	}

	sql += ` ORDER BY label_type DESC, id ASC`

	matches := []*fleet.Label{}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &matches, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting labels for search")
	}

	matches, err = ds.addAllHostsLabelToList(ctx, filter, matches, omit...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "adding all hosts label to matches")
	}

	return matches, nil
}

func (ds *Datastore) LabelIDsByName(ctx context.Context, names []string, filter fleet.TeamFilter) (map[string]uint, error) {
	if len(names) == 0 {
		return map[string]uint{}, nil
	}

	sql, args, err := applyLabelTeamFilter(`SELECT l.id, l.name FROM labels l WHERE l.name IN (?)`, filter, names)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query to get label ids by name")
	}

	var labels []fleet.Label
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get label ids by name")
	}

	result := make(map[string]uint, len(labels))
	for _, label := range labels {
		result[label.Name] = label.ID
	}

	return result, nil
}

func (ds *Datastore) LabelsByName(ctx context.Context, names []string, filter fleet.TeamFilter) (map[string]*fleet.Label, error) {
	if len(names) == 0 {
		return map[string]*fleet.Label{}, nil
	}

	sqlStatement, args, err := applyLabelTeamFilter(`SELECT l.* FROM labels l WHERE l.name IN (?)`, filter, names)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query to get label ids by name")
	}

	var labels []*fleet.Label
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, sqlStatement, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get labels by name")
	}

	result := make(map[string]*fleet.Label, len(labels))
	for _, label := range labels {
		result[label.Name] = label
	}

	return result, nil
}

// AsyncBatchInsertLabelMembership inserts into the label_membership table the
// batch of label_id + host_id tuples represented by the [2]uint array.
func (ds *Datastore) AsyncBatchInsertLabelMembership(ctx context.Context, batch [][2]uint) error {
	// NOTE: this is tested via the server/service/async package tests.

	sql := `INSERT INTO label_membership (label_id, host_id) VALUES `
	sql += strings.Repeat(`(?, ?),`, len(batch))
	sql = strings.TrimSuffix(sql, ",")
	sql += ` ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at)`

	vals := make([]interface{}, 0, len(batch)*2)
	for _, tup := range batch {
		vals = append(vals, tup[0], tup[1])
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, sql, vals...)
		return ctxerr.Wrap(ctx, err, "insert into label_membership")
	})
}

// AsyncBatchDeleteLabelMembership deletes from the label_membership table the
// batch of label_id + host_id tuples represented by the [2]uint array.
func (ds *Datastore) AsyncBatchDeleteLabelMembership(ctx context.Context, batch [][2]uint) error {
	// NOTE: this is tested via the server/service/async package tests.

	rest := strings.Repeat(`UNION ALL SELECT ?, ? `, len(batch)-1)
	sql := fmt.Sprintf(`
    DELETE
      lm
    FROM
      label_membership lm
    JOIN
      (SELECT ? label_id, ? host_id %s) del_list
    ON
      lm.label_id = del_list.label_id AND
      lm.host_id = del_list.host_id`, rest)

	vals := make([]interface{}, 0, len(batch)*2)
	for _, tup := range batch {
		vals = append(vals, tup[0], tup[1])
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, sql, vals...)
		return ctxerr.Wrap(ctx, err, "delete from label_membership")
	})
}

// AsyncBatchUpdateLabelTimestamp updates the hosts' label_updated_at timestamp
// for the batch of host ids provided.
func (ds *Datastore) AsyncBatchUpdateLabelTimestamp(ctx context.Context, ids []uint, ts time.Time) error {
	// NOTE: this is tested via the server/service/async package tests.
	sql := `
      UPDATE
        hosts
      SET
        label_updated_at = ?
      WHERE
        id IN (?)`
	query, args, err := sqlx.In(sql, ts, ids)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building query to update hosts.label_updated_at")
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return ctxerr.Wrap(ctx, err, "update hosts.label_updated_at")
	})
}

func amountLabelsDB(ctx context.Context, db sqlx.QueryerContext) (int, error) {
	var amount int
	err := sqlx.GetContext(ctx, db, &amount, `SELECT count(*) FROM labels`)
	if err != nil {
		return 0, err
	}
	return amount, nil
}

func (ds *Datastore) LabelsSummary(ctx context.Context, filter fleet.TeamFilter) ([]*fleet.LabelSummary, error) {
	var labelsSummary []*fleet.LabelSummary

	query := "SELECT id, name, description, label_type, team_id FROM labels l"
	query, params, err := applyLabelTeamFilter(query, filter)
	if err != nil {
		return nil, err
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labelsSummary, query, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "labels summary")
	}
	return labelsSummary, nil
}

// HostMemberOfAllLabels returns whether the given host is a member of all the provided labels.
// If the labels do not exist, then the host is considered not a member of the provided labels.
// A host will always be a member of an empty label set, so this method returns (true, nil)
// if labelNames is empty.
func (ds *Datastore) HostMemberOfAllLabels(ctx context.Context, hostID uint, labelNames []string) (bool, error) {
	if len(labelNames) == 0 {
		return true, nil
	}

	sqlStatement := `
		SELECT COUNT(*) = ? FROM labels l
		LEFT JOIN (SELECT label_id FROM label_membership WHERE host_id = ?) lm
		ON l.id = lm.label_id
		WHERE l.name IN (?) AND lm.label_id IS NOT NULL;
	`
	sql, args, err := sqlx.In(sqlStatement, len(labelNames), hostID, labelNames)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "building query to get label IDs")
	}

	var ok bool
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &ok, sql, args...); err != nil {
		return false, ctxerr.Wrap(ctx, err, "get label IDs")
	}

	return ok, nil
}

// AddLabelsToHost skips auth as it's only used in tests, and where label teams have already been validated.
func (ds *Datastore) AddLabelsToHost(ctx context.Context, hostID uint, labelIDs []uint) error {
	if len(labelIDs) == 0 {
		return nil
	}
	sql := `INSERT INTO label_membership (host_id, label_id) VALUES `
	sql += strings.Repeat(`(?, ?),`, len(labelIDs))
	sql = strings.TrimSuffix(sql, ",")
	sql += ` ON DUPLICATE KEY UPDATE updated_at = NOW()`
	args := make([]interface{}, 0, len(labelIDs)*2)
	for _, labelID := range labelIDs {
		args = append(args, hostID, labelID)
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, sql, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert into label_membership")
	}
	return nil
}

func (ds *Datastore) RemoveLabelsFromHost(ctx context.Context, hostID uint, labelIDs []uint) error {
	// We *don't* check label team here because a wrong-team label won't be on the host in the first place
	if len(labelIDs) == 0 {
		return nil
	}
	sql := `DELETE FROM label_membership WHERE host_id = ? AND label_id IN (?)`
	sql, args, err := sqlx.In(sql, hostID, labelIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build label_membership IN query")
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, sql, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "delete from label_membership")
	}
	return nil
}
