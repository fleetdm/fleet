package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// maintainedAppsAllowedOrderKeys allowlists order keys for listing
// Fleet-maintained apps. The list is a combined-by-name view, so name is the
// only meaningful key; it's validation-only, since ORDER BY is hard-coded below.
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
		// upsert the maintained app
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

// ReconcileMaintainedAppSoftwareNames normalizes the names of macOS
// software_titles (and the matching software inventory rows) so they use the
// canonical Fleet-maintained app name instead of the osquery- or
// installer-reported name (e.g. "Code" -> "Microsoft Visual Studio Code").
//
// It is called once per maintained-apps sync (hourly), is set-based, and is
// idempotent, so it self-heals: a row that already matches is left untouched.
//
// Because an FMA's unique_identifier (the macOS bundle identifier) is NOT unique
// across the FMA list — Firefox and Firefox ESR both report org.mozilla.firefox,
// LibreOffice and a future LibreOffice-still would share org.libreoffice.script —
// renaming purely by bundle identifier is ambiguous and would clobber titles with
// whichever sibling FMA was synced last. To stay correct it renames in two passes:
//
//  1. Precise: a title that was added via a specific FMA is linked through
//     software_installers.fleet_maintained_app_id. When that link resolves to a
//     single FMA name, use it — this disambiguates shared identifiers because we
//     know exactly which FMA the user added, and it repairs titles previously
//     mislabeled by the old blind-rename behavior.
//
//  2. Heuristic: for titles not added via an FMA (osquery-detected, custom
//     installer, VPP), fall back to matching by bundle identifier, but ONLY when
//     that identifier maps to exactly one FMA name. Shared identifiers are skipped
//     entirely, since there is no single correct canonical name.
func (ds *Datastore) ReconcileMaintainedAppSoftwareNames(ctx context.Context) error {
	// titleNameByFMA maps each software_title to the canonical name of the single
	// darwin FMA it was added with, resolved through its installer link
	// (software_installers.fleet_maintained_app_id). A title linked to
	// differently-named FMAs (e.g. Firefox on one team, Firefox ESR on another,
	// sharing one global title) is excluded by the HAVING clause, since we cannot
	// pick a single name for it. Grouping to one row per title also keeps the
	// UPDATEs below from fanning out over a title's per-team installer rows.
	const titleNameByFMA = `
		SELECT si.title_id, MIN(fma.name) AS name
		FROM software_installers si
		JOIN fleet_maintained_apps fma
			ON fma.id = si.fleet_maintained_app_id AND fma.platform = 'darwin'
		GROUP BY si.title_id
		HAVING COUNT(DISTINCT fma.name) = 1`

	// unambiguousByIdentifier is the set of darwin bundle identifiers that map to
	// exactly one FMA name. Identifiers shared by differently-named FMAs (Firefox /
	// Firefox ESR) are excluded by the HAVING clause and never auto-renamed.
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
		// Pass 1 (precise): rename titles linked to a single FMA via their installer.
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

		// Pass 2 (heuristic): rename rows matched by an unambiguous bundle identifier.
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

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		for _, u := range updates {
			if _, err := tx.ExecContext(ctx, u.stmt); err != nil {
				return ctxerr.Wrapf(ctx, err, "reconcile maintained app names: %s", u.label)
			}
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
				SELECT DISTINCT st.id, st.unique_identifier, st.name, si.platform, si.fleet_maintained_app_id
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
				-- When the title was added via a specific FMA, match that exact app by
				-- its installer link. This disambiguates FMAs that share a bundle
				-- identifier (e.g. Firefox vs Firefox ESR): adding Firefox must not mark
				-- Firefox ESR as added.
				ON team_titles.fleet_maintained_app_id = fma.id
				-- Otherwise (detected-only, custom installer, or VPP), the title is not
				-- tied to a specific FMA, so fall back to matching by bundle identifier.
				OR (
					team_titles.fleet_maintained_app_id IS NULL
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

	// We paginate by distinct app NAME, because the UI combines an app's macOS
	// and Windows entries into a single row and an app must not be split across a
	// page boundary. The count, by contrast, is the total number of apps (each
	// platform entry is its own installable app). The team join lets us tell
	// whether each app has already been added, which the "available only" filter
	// needs.
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

	// Total count of matching apps. We count distinct rows (by primary key), not
	// distinct names: an app's macOS and Windows entries are separate installable
	// apps and are each counted, even though the UI combines them into one row.
	// DISTINCT fma.id also collapses any duplicate rows from the team join's
	// fan-out.
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

	// Select the page of app names, fetching one extra to detect a next page.
	perPage := opt.GetPerPage()
	pageNamesStmt := fmt.Sprintf(
		`SELECT DISTINCT fma.name %s%s ORDER BY fma.name %s LIMIT %d OFFSET %d`,
		fromClause, where, direction, perPage+1, perPage*opt.Page,
	)
	pageNamesArgs := append(append([]any{}, fromArgs...), whereArgs...)
	var pageNames []string
	if err := sqlx.SelectContext(ctx, dbReader, &pageNames, pageNamesStmt, pageNamesArgs...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "selecting fleet maintained app page names")
	}

	meta := &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0, TotalResults: uint(filteredCount)} //nolint:gosec // dismiss G115
	if uint(len(pageNames)) > perPage {                                                                    //nolint:gosec // dismiss G115
		meta.HasNextResults = true
		pageNames = pageNames[:perPage]
	}
	if len(pageNames) == 0 {
		// Page is past the last result.
		return []fleet.MaintainedApp{}, meta, nil
	}

	// Fetch every platform row for the apps on this page so the UI can combine
	// an app's macOS and Windows entries into a single row.
	selectStmt := `SELECT fma.id, fma.name, fma.platform, fma.slug, `
	var rowsArgs []any
	if teamID != nil {
		selectStmt += teamFMATitlesJoin + ` WHERE fma.name IN (?)`
		rowsArgs = []any{teamID, teamID, pageNames}
	} else {
		selectStmt += `NULL software_title_id FROM fleet_maintained_apps fma WHERE fma.name IN (?)`
		rowsArgs = []any{pageNames}
	}
	selectStmt += fmt.Sprintf(` ORDER BY fma.name %s, fma.platform ASC`, direction)

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
	// Only return identifiers that map to exactly one FMA name. A bundle
	// identifier shared by differently-named FMAs (e.g. org.mozilla.firefox for
	// both Firefox and Firefox ESR) has no single canonical name, so it is
	// omitted here and callers fall back to the osquery-reported name.
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
