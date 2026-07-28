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
// titles onto the canonical FMA title (Windows has no bundle identifier to join on,
// so the program name is the only key). Unlike the darwin passes above, which are a
// rename, this is a merge: the versioned title and the installer's title are separate
// rows, so software is re-pointed from one to the other.
//
// The emptied titles are deliberately left in place. software_titles is referenced
// with ON DELETE CASCADE by policies.patch_software_title_id, software_title_team_pins,
// software_update_schedules, software_title_icons and software_title_display_names,
// so deleting a merged title would silently discard admin configuration attached to
// it. A title with no software rows drops out of the software list once host counts
// are recomputed, since software_titles_host_counts requires hosts_count > 0.
//
// Each FMA runs in its own transaction and the operation is idempotent.
func (ds *Datastore) reconcileWindowsMaintainedAppSoftwareTitles(ctx context.Context) error {
	fmaNames, err := ds.GetWindowsFMANames(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get windows FMA names for reconcile")
	}

	// Two FMA rows can resolve to the same destination title (an app listed under two
	// slugs, say), in which case merging it twice is harmless. Processing the same
	// destination more than once is still wasted work, so collapse them.
	allPrefixes := windowsFMAPrefixes(fmaNames)
	seen := make(map[uint]struct{}, len(fmaNames))
	for _, fma := range fmaNames {
		if _, ok := seen[fma.TitleID]; ok {
			continue
		}
		seen[fma.TitleID] = struct{}{}
		if err := ds.mergeWindowsFMATitle(ctx, fma, allPrefixes); err != nil {
			return ctxerr.Wrapf(ctx, err, "merge windows FMA title %q", fma.Name)
		}
	}

	return nil
}

func (ds *Datastore) mergeWindowsFMATitle(ctx context.Context, fma fleet.WindowsFMAName, allPrefixes []windowsFMAPrefix) error {
	prefixes := fma.MatchPrefixes()
	if len(prefixes) == 0 {
		return nil
	}

	// Narrow the scan to names that could plausibly belong to this FMA. The final
	// decision is made in Go by matchWindowsFMATitle so that prefix precedence and the
	// ambiguity rule match the ingestion path exactly.
	var nameConds []string
	args := []any{fma.TitleID}
	for _, prefix := range prefixes {
		// Escape LIKE wildcards so a name containing % or _ can't widen the match. The
		// ESCAPE clause below is stated explicitly rather than relying on the default.
		escaped := prefix
		for _, c := range []string{`\`, `%`, `_`} {
			escaped = strings.ReplaceAll(escaped, c, `\`+c)
		}
		nameConds = append(nameConds, `st.name = ? OR st.name LIKE ? ESCAPE '\\'`)
		args = append(args, prefix, escaped+" %")
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Inventory-only versioned titles. Exclude the destination itself, titles with
		// an upgrade code (those join through it instead), and titles owned by an
		// installer/VPP/in-house app, whose links are the authoritative mapping.
		staleStmt := `
			SELECT st.id, st.name
			FROM software_titles st
			WHERE st.source = 'programs' AND st.extension_for = ''
				AND st.id <> ?
				AND (` + strings.Join(nameConds, " OR ") + `)
				AND (st.upgrade_code IS NULL OR st.upgrade_code = '')
				AND NOT EXISTS (SELECT 1 FROM software_installers si WHERE si.title_id = st.id)
				AND NOT EXISTS (SELECT 1 FROM vpp_apps va WHERE va.title_id = st.id)
				AND NOT EXISTS (SELECT 1 FROM in_house_apps iha WHERE iha.title_id = st.id)`
		var candidates []struct {
			ID   uint   `db:"id"`
			Name string `db:"name"`
		}
		if err := sqlx.SelectContext(ctx, tx, &candidates, staleStmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "select stale windows titles")
		}

		var staleIDs []uint
		for _, c := range candidates {
			if match, ok := matchWindowsFMATitle(c.Name, allPrefixes); ok && match.titleID == fma.TitleID {
				staleIDs = append(staleIDs, c.ID)
			}
		}
		if len(staleIDs) == 0 {
			return nil
		}

		// The destination is the title the installer owns, carried on the FMA itself, so
		// there is nothing to create or look up by name here.
		canonicalID := fma.TitleID

		// Re-point software onto the canonical title, leaving software.name unchanged so
		// hosts still report the version they actually have installed.
		repointStmt, repointArgs, err := sqlx.In(`UPDATE software SET title_id = ? WHERE title_id IN (?)`, canonicalID, staleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build re-point software statement")
		}
		if _, err := tx.ExecContext(ctx, repointStmt, repointArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "re-point software to canonical windows title")
		}

		// Drop the now-meaningless host counts for the emptied titles; the counts cron
		// recomputes them. The titles themselves are left alone on purpose, see the
		// comment on reconcileWindowsMaintainedAppSoftwareTitles.
		countsStmt, countsArgs, err := sqlx.In(`DELETE FROM software_titles_host_counts WHERE software_title_id IN (?)`, staleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build delete stale host counts statement")
		}
		if _, err := tx.ExecContext(ctx, countsStmt, countsArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete stale windows title host counts")
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

// GetWindowsFMANames returns each Windows FMA's canonical name and unique_identifier
// unchanged. Both are returned because either can be the one that matches a reported
// program name; choosing between them is WindowsFMAName.MatchPrefixes's job.
func (ds *Datastore) GetWindowsFMANames(ctx context.Context) ([]fleet.WindowsFMAName, error) {
	// Restricted to FMAs that are actually added somewhere, via their installer link.
	// Prefix matching on a program name is only as precise as the name allows: the FMA
	// manifests pair it with a publisher check, which fleet_maintained_apps does not
	// carry, so requiring a deliberate install is what bounds the blast radius.
	//
	// The installer link also supplies the title to merge onto. An app's per-team
	// installer rows normally share one title, so they collapse here; an app somehow
	// spanning several is ambiguous and excluded rather than guessed at, matching how
	// the darwin passes above handle a bundle identifier shared by multiple FMAs.
	query := `
		SELECT fma.name, fma.unique_identifier, MIN(si.title_id) AS title_id, MIN(st.name) AS title_name
		FROM fleet_maintained_apps fma
		JOIN software_installers si
			ON si.fleet_maintained_app_id = fma.id AND si.title_id IS NOT NULL
		JOIN software_titles st ON st.id = si.title_id
		WHERE fma.platform = 'windows' AND fma.name != ''
		GROUP BY fma.id, fma.name, fma.unique_identifier
		HAVING COUNT(DISTINCT si.title_id) = 1`

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
