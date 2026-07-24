package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// maintainedAppsAllowedOrderKeys allowlists order keys for listing
// Fleet-maintained apps. The list is a combined-by-app view (see
// ListAvailableFleetMaintainedApps), so name is the only meaningful key; it's
// validation-only, since ORDER BY is hard-coded below.
var maintainedAppsAllowedOrderKeys = common_mysql.OrderKeyAllowlist{
	"name": "fma.name",
}

func (ds *Datastore) UpsertMaintainedApp(ctx context.Context, app *fleet.MaintainedApp) (*fleet.MaintainedApp, error) {
	const upsertStmt = `
INSERT INTO
	fleet_maintained_apps (name, slug, platform, unique_identifier)
VALUES
	(?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
	name = VALUES(name),
	platform = VALUES(platform),
	unique_identifier = VALUES(unique_identifier)
`

	var appID uint
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, upsertStmt, app.Name, app.Slug, app.Platform, app.UniqueIdentifier)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "upsert maintained app")
		}
		id, _ := res.LastInsertId()
		appID = uint(id) //nolint:gosec // dismiss G115

		return nil
	})
	if err != nil {
		return nil, err
	}

	app.ID = appID
	return app, nil
}

// ReconcileMaintainedAppSoftwareNames renames macOS software_titles and software
// rows to the canonical FMA name (e.g. "Code" -> "Microsoft Visual Studio Code").
// Called once per sync; set-based and idempotent.
//
// A bundle identifier is not unique across FMAs (Firefox and Firefox ESR both use
// org.mozilla.firefox), so renaming by identifier alone is ambiguous. It renames in
// two passes: first by the precise installer link, then by bundle identifier but
// only when it maps to a single FMA name.
func (ds *Datastore) ReconcileMaintainedAppSoftwareNames(ctx context.Context) error {
	// title_id -> name, for titles linked to a single FMA via their installer.
	// GROUP BY also collapses a title's per-team installer rows to avoid fan-out.
	const titleNameByFMA = `
		SELECT si.title_id, MIN(fma.name) AS name
		FROM software_installers si
		JOIN fleet_maintained_apps fma
			ON fma.id = si.fleet_maintained_app_id AND fma.platform = 'darwin'
		GROUP BY si.title_id
		HAVING COUNT(DISTINCT fma.name) = 1`

	// darwin bundle identifiers mapping to exactly one FMA name; shared ones are excluded.
	const unambiguousByIdentifier = `
		SELECT unique_identifier, MIN(name) AS name
		FROM fleet_maintained_apps
		WHERE platform = 'darwin'
		GROUP BY unique_identifier
		HAVING COUNT(DISTINCT name) = 1`

	updates := []struct {
		label string
		stmt  string
	}{
		// Pass 1: precise, via installer link.
		{"software_titles by installer link", `
			UPDATE software_titles st
				JOIN (` + titleNameByFMA + `) fma ON fma.title_id = st.id
			SET st.name = fma.name
			WHERE st.name <> fma.name`},
		{"software by installer link", `
			UPDATE software s
				JOIN (` + titleNameByFMA + `) fma ON fma.title_id = s.title_id
			SET s.name = fma.name
			WHERE s.name <> fma.name`},

		// Pass 2: by bundle identifier, unambiguous only.
		{"software_titles by bundle identifier", `
			UPDATE software_titles st
				JOIN (` + unambiguousByIdentifier + `) fma ON fma.unique_identifier = st.bundle_identifier
			SET st.name = fma.name
			WHERE st.name <> fma.name`},
		{"software by bundle identifier", `
			UPDATE software s
				JOIN (` + unambiguousByIdentifier + `) fma ON fma.unique_identifier = s.bundle_identifier
			SET s.name = fma.name
			WHERE s.name <> fma.name`},
	}

	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		for _, u := range updates {
			if _, err := tx.ExecContext(ctx, u.stmt); err != nil {
				return ctxerr.Wrapf(ctx, err, "reconcile maintained app names: %s", u.label)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return ds.reconcileWindowsMaintainedAppSoftwareTitles(ctx)
}

// reconcileWindowsMaintainedAppSoftwareTitles collapses versioned Windows program
// titles onto the canonical FMA title by name prefix (Windows has no bundle
// identifier to join on). Unlike the macOS rename this is a merge: software is
// re-pointed at the canonical title and the orphaned versioned titles deleted.
// Each FMA runs in its own transaction and the operation is idempotent.
func (ds *Datastore) reconcileWindowsMaintainedAppSoftwareTitles(ctx context.Context) error {
	fmaNames, err := ds.GetWindowsFMANames(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get windows FMA names for reconcile")
	}

	for _, fma := range fmaNames {
		if err := ds.mergeWindowsFMATitle(ctx, fma); err != nil {
			return ctxerr.Wrapf(ctx, err, "merge windows FMA title %q", fma.Name)
		}
	}

	return nil
}

func (ds *Datastore) mergeWindowsFMATitle(ctx context.Context, fma fleet.WindowsFMAName) error {
	// Escape LIKE wildcards so an FMA name with % or _ can't widen the match.
	escaped := fma.Prefix
	for _, c := range []string{`\`, `%`, `_`} {
		escaped = strings.ReplaceAll(escaped, c, `\`+c)
	}
	likePrefix := escaped + " %"

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Inventory-only versioned titles matching the prefix. Exclude the canonical
		// name, MSI titles (upgrade_code collapses those), and titles owned by an
		// installer/VPP/in-house app (deleting them would null those links).
		const staleStmt = `
			SELECT st.id
			FROM software_titles st
			WHERE st.source = 'programs' AND st.extension_for = ''
				AND st.name <> ?
				AND (st.name = ? OR st.name LIKE ?)
				AND (st.upgrade_code IS NULL OR st.upgrade_code = '')
				AND NOT EXISTS (SELECT 1 FROM software_installers si WHERE si.title_id = st.id)
				AND NOT EXISTS (SELECT 1 FROM vpp_apps va WHERE va.title_id = st.id)
				AND NOT EXISTS (SELECT 1 FROM in_house_apps iha WHERE iha.title_id = st.id)`
		var staleIDs []uint
		if err := sqlx.SelectContext(ctx, tx, &staleIDs, staleStmt, fma.Name, fma.Prefix, likePrefix); err != nil {
			return ctxerr.Wrap(ctx, err, "select stale windows titles")
		}
		if len(staleIDs) == 0 {
			return nil
		}

		// Ensure the canonical title exists (the installer normally created it).
		if _, err := tx.ExecContext(ctx,
			`INSERT IGNORE INTO software_titles (name, source, extension_for, upgrade_code) VALUES (?, 'programs', '', '')`,
			fma.Name,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "ensure canonical windows title")
		}
		var canonicalID uint
		if err := sqlx.GetContext(ctx, tx, &canonicalID,
			`SELECT id FROM software_titles WHERE name = ? AND source = 'programs' AND extension_for = ''`,
			fma.Name,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "select canonical windows title")
		}

		// Re-point software onto the canonical title, leaving software.name unchanged.
		repointStmt, repointArgs, err := sqlx.In(`UPDATE software SET title_id = ? WHERE title_id IN (?)`, canonicalID, staleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build re-point software statement")
		}
		if _, err := tx.ExecContext(ctx, repointStmt, repointArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "re-point software to canonical windows title")
		}

		// Delete the orphaned host counts and titles (counts are recomputed by cron).
		countsStmt, countsArgs, err := sqlx.In(`DELETE FROM software_titles_host_counts WHERE software_title_id IN (?)`, staleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build delete stale host counts statement")
		}
		if _, err := tx.ExecContext(ctx, countsStmt, countsArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete stale windows title host counts")
		}

		titlesStmt, titlesArgs, err := sqlx.In(`DELETE FROM software_titles WHERE id IN (?)`, staleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build delete stale titles statement")
		}
		if _, err := tx.ExecContext(ctx, titlesStmt, titlesArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete stale windows titles")
		}

		return nil
	})
}

// fleetMaintainedAppsTeamJoin is the FROM clause plus the LEFT JOIN that
// determines, for a given team, whether each Fleet-maintained app has already
// been added (via a software installer or VPP app). team_titles.id is non-NULL
// when the app is already added to the team. It expects two `?` args, both the
// team's global_or_team_id.
const fleetMaintainedAppsTeamJoin = `
			FROM fleet_maintained_apps fma
			LEFT JOIN (
				-- COALESCE the platform so VPP-added titles (no installer row) still
				-- carry a platform for the platform-scoped identifier fallback below.
				SELECT DISTINCT st.id, st.unique_identifier, st.name, COALESCE(si.platform, va.platform) AS platform, si.fleet_maintained_app_id
				FROM software_titles st
				LEFT JOIN
					software_installers si
					ON si.title_id = st.id AND si.global_or_team_id = ?
					AND si.platform IN ('darwin','windows')
				LEFT JOIN
					vpp_apps va
					ON va.title_id = st.id
					AND va.platform = 'darwin'
				LEFT JOIN
					vpp_apps_teams vat
					ON vat.adam_id = va.adam_id
					AND vat.platform = va.platform
					AND vat.global_or_team_id = ?
				WHERE si.id IS NOT NULL OR vat.id IS NOT NULL
			) team_titles
				-- Match the exact FMA the title was added with, so a shared bundle
				-- identifier (Firefox vs Firefox ESR) doesn't mark the sibling added.
				ON team_titles.fleet_maintained_app_id = fma.id
				-- Not added via an FMA: fall back to the bundle identifier, scoped to
				-- the same platform so a darwin title can't match a windows FMA (or
				-- vice versa) when their identifiers happen to collide.
				OR (
					team_titles.fleet_maintained_app_id IS NULL
					AND team_titles.platform = fma.platform
					AND team_titles.unique_identifier = fma.unique_identifier
				)
				-- pattern match fma name to a similar title name, since upgrade_code is not surfaced in fma table
				OR (
					team_titles.fleet_maintained_app_id IS NULL
					AND team_titles.platform = fma.platform
					AND fma.platform = 'windows'
					-- Box Drive is the only FMA at the point of writing this where unique_identifier is shorter than name
					AND team_titles.name LIKE CONCAT(LEAST(fma.name, fma.unique_identifier), '%')
				)
`

// teamFMATitlesJoin selects software_title_id alongside the team join, for use
// directly after `SELECT fma.id, fma.name, ..., `.
const teamFMATitlesJoin = `team_titles.id software_title_id ` + fleetMaintainedAppsTeamJoin

func (ds *Datastore) GetMaintainedAppByID(ctx context.Context, appID uint, teamID *uint) (*fleet.MaintainedApp, error) {
	stmt := `SELECT fma.id, fma.name, fma.platform, fma.unique_identifier, fma.slug, `
	var args []any

	if teamID != nil {
		stmt += teamFMATitlesJoin
		args = []any{teamID, teamID}
	} else {
		stmt += `NULL software_title_id FROM fleet_maintained_apps fma`
	}

	stmt += ` WHERE fma.id = ?`
	args = append(args, appID)

	var app fleet.MaintainedApp
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &app, stmt, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("MaintainedApp"), "no matching maintained app found")
		}

		return nil, ctxerr.Wrap(ctx, err, "getting maintained app by id")
	}

	return &app, nil
}

func (ds *Datastore) GetMaintainedAppBySlug(ctx context.Context, slug string, teamID *uint) (*fleet.MaintainedApp, error) {
	stmt := `SELECT fma.id, fma.name, fma.platform, fma.unique_identifier, fma.slug, `
	var args []any

	if teamID != nil {
		stmt += teamFMATitlesJoin
		args = []any{teamID, teamID}
	} else {
		stmt += `NULL software_title_id FROM fleet_maintained_apps fma`
	}

	stmt += ` WHERE fma.slug = ?`
	args = append(args, slug)

	var app fleet.MaintainedApp
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &app, stmt, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("MaintainedApp"), "no matching maintained app found")
		}

		return nil, ctxerr.Wrap(ctx, err, "getting maintained app by slug")
	}

	return &app, nil
}

func (ds *Datastore) ListAvailableFleetMaintainedApps(ctx context.Context, teamID *uint, opt fleet.MaintainedAppListOptions) ([]fleet.MaintainedApp, *fleet.PaginationMetadata, error) {
	dbReader := ds.reader(ctx)

	// We paginate by distinct app token (the slug prefix, e.g. "figma" in
	// "figma/darwin"), which identifies an app across its platform entries: the UI
	// combines an app's macOS and Windows entries into one row, so an app must not
	// be split across a page boundary. Keying on the token rather than the name
	// keeps two distinct apps that share a name (e.g. gemini/darwin and
	// google-gemini/darwin) as separate rows. The count, by contrast, is the
	// number of installable platform entries: each is separately installable (its
	// own Add button), so an app shipped on both platforms counts twice. The team
	// join tells us whether each app is already added, for the "available only"
	// filter.
	fromClause := `FROM fleet_maintained_apps fma`
	var fromArgs []any
	if teamID != nil {
		fromClause = fleetMaintainedAppsTeamJoin
		fromArgs = []any{teamID, teamID}
	}

	// Build the filter conditions shared by the count and page-name queries.
	where := ` WHERE TRUE`
	var whereArgs []any
	if match := opt.MatchQuery; match != "" {
		where += ` AND fma.name LIKE ?`
		whereArgs = append(whereArgs, likePattern(match))
	}
	if opt.Platform == "darwin" || opt.Platform == "windows" {
		where += ` AND fma.platform = ?`
		whereArgs = append(whereArgs, opt.Platform)
	}
	if opt.AvailableOnly && teamID != nil {
		// "Hide added apps": keep only entries not yet added to this team.
		where += ` AND team_titles.id IS NULL`
	}

	// Count the installable platform entries (each Add button); DISTINCT id also
	// collapses the team join's fan-out.
	countArgs := append(append([]any{}, fromArgs...), whereArgs...)
	var filteredCount int
	if err := sqlx.GetContext(ctx, dbReader, &filteredCount, `SELECT COUNT(DISTINCT fma.id) `+fromClause+where, countArgs...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get fleet maintained apps count")
	}

	if filteredCount == 0 {
		// Distinguish an empty library (an error) from filters matching nothing
		// (an empty, non-error result).
		var totalCount int
		if err := sqlx.GetContext(ctx, dbReader, &totalCount, `SELECT COUNT(id) FROM fleet_maintained_apps`); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "get fleet maintained apps total count")
		}
		if totalCount == 0 {
			return nil, nil, &fleet.NoMaintainedAppsInDatabaseError{}
		}
		return []fleet.MaintainedApp{}, &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0}, nil
	}

	// Validate the requested order key against the allowlist, which permits only
	// "name" (the apps are always ordered by name below; see the allowlist
	// declaration). Any other key, including an empty one, is handled here: an
	// empty key skips validation and falls through to the default name ordering.
	if key := opt.OrderKey; key != "" {
		if _, ok := maintainedAppsAllowedOrderKeys[key]; !ok {
			return nil, nil, ctxerr.Wrap(ctx, common_mysql.InvalidOrderKeyError{Key: key, Allowed: maintainedAppsAllowedOrderKeys.AllowedKeys()}, "list fleet maintained apps")
		}
	}
	direction := "ASC"
	if opt.IsDescending() {
		direction = "DESC"
	}

	// Select the page of app tokens, fetching one extra to detect a next page.
	// Group by the token and order by the app's name (the token maps to a single
	// name), with the token as a deterministic tiebreaker for same-named apps.
	perPage := opt.GetPerPage()
	pageTokensStmt := fmt.Sprintf(
		`SELECT SUBSTRING_INDEX(fma.slug, '/', 1) AS app_token %s%s GROUP BY app_token ORDER BY MIN(fma.name) %s, app_token %s LIMIT %d OFFSET %d`,
		fromClause, where, direction, direction, perPage+1, perPage*opt.Page,
	)
	pageTokensArgs := append(append([]any{}, fromArgs...), whereArgs...)
	var pageTokens []string
	if err := sqlx.SelectContext(ctx, dbReader, &pageTokens, pageTokensStmt, pageTokensArgs...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "selecting fleet maintained app page tokens")
	}

	meta := &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0, TotalResults: uint(filteredCount)} //nolint:gosec // dismiss G115
	if uint(len(pageTokens)) > perPage {                                                                   //nolint:gosec // dismiss G115
		meta.HasNextResults = true
		pageTokens = pageTokens[:perPage]
	}
	if len(pageTokens) == 0 {
		// Page is past the last result.
		return []fleet.MaintainedApp{}, meta, nil
	}

	// Fetch every platform row for the apps on this page so the UI can combine
	// an app's macOS and Windows entries into a single row.
	selectStmt := `SELECT fma.id, fma.name, fma.platform, fma.slug, `
	var rowsArgs []any
	if teamID != nil {
		selectStmt += teamFMATitlesJoin + ` WHERE SUBSTRING_INDEX(fma.slug, '/', 1) IN (?)`
		rowsArgs = []any{teamID, teamID, pageTokens}
	} else {
		selectStmt += `NULL software_title_id FROM fleet_maintained_apps fma WHERE SUBSTRING_INDEX(fma.slug, '/', 1) IN (?)`
		rowsArgs = []any{pageTokens}
	}
	selectStmt += fmt.Sprintf(` ORDER BY fma.name %s, fma.slug ASC`, direction)

	selectStmt, rowsArgs, err := sqlx.In(selectStmt, rowsArgs...)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "building list fleet maintained apps query")
	}
	selectStmt = dbReader.Rebind(selectStmt)

	var avail []fleet.MaintainedApp
	if err := sqlx.SelectContext(ctx, dbReader, &avail, selectStmt, rowsArgs...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "selecting available fleet maintained apps")
	}

	return avail, meta, nil
}

func (ds *Datastore) GetFMANamesByIdentifier(ctx context.Context) (map[string]string, error) {
	// Only identifiers mapping to one FMA name; shared ones (Firefox/ESR) have no
	// single canonical name, so callers fall back to the osquery-reported name.
	query := `
		SELECT unique_identifier, MIN(name) AS name
		FROM fleet_maintained_apps
		WHERE platform = 'darwin'
		GROUP BY unique_identifier
		HAVING COUNT(DISTINCT name) = 1`

	rows, err := ds.reader(ctx).QueryContext(ctx, query)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "query FMA names by identifier")
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var identifier, name string
		if err := rows.Scan(&identifier, &name); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "scan FMA name row")
		}
		result[identifier] = name
	}
	if err := rows.Err(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "iterate FMA name rows")
	}

	return result, nil
}

// GetWindowsFMANames returns each Windows FMA's canonical name and match prefix.
// The prefix is the shorter of name and unique_identifier (LEAST), matching the
// fleetMaintainedAppsTeamJoin convention.
func (ds *Datastore) GetWindowsFMANames(ctx context.Context) ([]fleet.WindowsFMAName, error) {
	query := `
		SELECT DISTINCT LEAST(name, unique_identifier) AS prefix, name
		FROM fleet_maintained_apps
		WHERE platform = 'windows' AND name != '' AND unique_identifier != ''`

	var names []fleet.WindowsFMAName
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &names, query); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "query Windows FMA names")
	}

	return names, nil
}

func (ds *Datastore) ClearRemovedFleetMaintainedApps(ctx context.Context, slugsToKeep []string) error {
	stmt := `DELETE FROM fleet_maintained_apps WHERE slug NOT IN (?)`

	var err error
	var args []any
	switch len(slugsToKeep) {
	case 0:
		stmt = `DELETE FROM fleet_maintained_apps`
	default:
		stmt, args, err = sqlx.In(stmt, slugsToKeep)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building sqlx.In statement for clearing removed maintained apps")
		}
	}

	_, err = ds.writer(ctx).ExecContext(ctx, stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "clearing removed maintained apps")
	}

	return nil
}
