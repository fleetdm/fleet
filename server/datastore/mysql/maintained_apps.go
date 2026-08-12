package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
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

// maintainedAppNameReconcileBatchSize caps how many rows a single reconcile UPDATE
// touches. Each batch commits its own transaction, so this bounds how long InnoDB
// holds row locks on software / software_titles. A var so tests can lower it.
var maintainedAppNameReconcileBatchSize = 500

// maintainedAppNameReconcileDiscoveryLimit caps how many mismatched rows one discovery
// SELECT returns, bounding the pass's memory no matter how many rows need renaming;
// the windowed loop in ReconcileMaintainedAppSoftwareNames drains rows past it. A var
// so tests can lower it.
var maintainedAppNameReconcileDiscoveryLimit = 10_000

// ReconcileMaintainedAppSoftwareNames renames macOS software_titles and software rows
// to the canonical Fleet-maintained app name (e.g. "Code" -> "Microsoft Visual Studio
// Code"). Inventory and the installer already share a title via bundle_identifier, so
// only the name needs correcting. Batched and idempotent.
//
// A bundle identifier is not unique across apps (Firefox and Firefox ESR both use
// org.mozilla.firefox), so renaming by identifier alone is ambiguous: it renames first
// by the precise installer link, then by bundle identifier but only where it maps to a
// single app name.
//
// Discovery is a different matter and does join. That hazard is specific to UPDATE: a SELECT
// here is a non-locking consistent read. So each pass finds its mismatched rows in
// LIMIT-bounded windows and renames them by primary key in small batches, each batch its own
// transaction: locks stay on a handful of rows and never span batches, and memory stays
// bounded no matter how many rows are mismatched. Every UPDATE re-checks `name <> ?`, which
// keeps the pass idempotent.
func (ds *Datastore) ReconcileMaintainedAppSoftwareNames(ctx context.Context) error {
	primaryCtx := ctxdb.RequirePrimary(ctx, true)

	var renamedTitles, renamedSoftware int64

	// Pass 1 renames via the precise installer link, pass 2 by bundle identifier where it maps
	// to a single app. They run in order, each discovering only after the previous has applied,
	// so where the two disagree the identifier wins and pass 2 never re-attempts what pass 1
	// already fixed.
	//
	// Within a pass the title is renamed before its software rows and each write commits on its
	// own, so a failure part-way can leave a title carrying the canonical name while its
	// software rows still carry the reported one. That window is preferred over the
	// alternative: one transaction spanning every write is what caused the lock contention this
	// pass was rewritten to avoid. The next run repairs it.
	steps := []struct {
		label      string
		selectStmt string
		updateStmt string
		renamed    *int64
	}{
		{"software_titles by installer link", mismatchedTitlesByInstallerLink, updateSoftwareTitleNames, &renamedTitles},
		{"software by installer link", mismatchedSoftwareByInstallerLink, updateSoftwareNames, &renamedSoftware},
		{"software_titles by bundle identifier", mismatchedTitlesByIdentifier, updateSoftwareTitleNames, &renamedTitles},
		{"software by bundle identifier", mismatchedSoftwareByIdentifier, updateSoftwareNames, &renamedSoftware},
	}

	for _, step := range steps {
		// Discovery is windowed by LIMIT so the pass never holds every mismatched row in
		// memory. Renaming a window removes its rows from the next SELECT's result -- every
		// UPDATE re-checks `name <> ?` -- so re-running the same query walks the remainder
		// without an offset, and a window that comes back short means nothing is left. A
		// rename that fails returns its error, so the loop cannot spin on rows it cannot fix.
		for {
			var rows []struct {
				ID   uint   `db:"id"`
				Name string `db:"name"`
			}
			if err := sqlx.SelectContext(primaryCtx, ds.reader(primaryCtx), &rows, step.selectStmt, maintainedAppNameReconcileDiscoveryLimit); err != nil {
				return ctxerr.Wrapf(ctx, err, "reconcile maintained app names: find %s", step.label)
			}
			if len(rows) == 0 {
				break
			}

			// One UPDATE carries one name, so group the ids by the name they should get.
			idsByName := make(map[string][]uint, len(rows))
			for _, r := range rows {
				idsByName[r.Name] = append(idsByName[r.Name], r.ID)
			}

			for name, ids := range idsByName {
				if err := common_mysql.BatchProcessSimple(ids, maintainedAppNameReconcileBatchSize, func(batch []uint) error {
					stmt, args, err := sqlx.In(step.updateStmt, name, batch, name)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "build rename statement")
					}
					n, err := ds.renameWithRetry(ctx, stmt, args...)
					if err != nil {
						return err
					}
					*step.renamed += n
					return nil
				}); err != nil {
					return ctxerr.Wrapf(ctx, err, "reconcile maintained app names: rename %s", step.label)
				}
			}

			if len(rows) < maintainedAppNameReconcileDiscoveryLimit {
				break
			}
		}
	}

	if (renamedTitles > 0 || renamedSoftware > 0) && ds.logger != nil {
		ds.logger.InfoContext(ctx, "reconciled Fleet-maintained app software names",
			"software_titles_renamed", renamedTitles, "software_renamed", renamedSoftware)
	}

	return nil
}

// Statements for ReconcileMaintainedAppSoftwareNames.
//
// The join order is pinned with STRAIGHT_JOIN so the materialized catalog is always the outer
// table and every software row is reached by index. Left to itself the optimizer can flip
// that around and scan the target table once per catalog row. The name comparison stays in
// SQL so it uses the columns' utf8mb4_unicode_ci collation; comparing in Go would be
// byte-exact, disagree with the UPDATE's own predicate, and re-select case-only differences
// on every run.
//
// additional_identifier is 1 for ios_apps and 2 for ipados_apps, so requiring 0 keeps a macOS
// app's canonical name off its iOS and iPadOS sibling titles, which are distinct products
// that happen to share a bundle identifier. `software` has no such column, so it excludes
// those sources directly.
const (
	// title_id -> name, for titles linked to a single app via their installer. GROUP BY also
	// collapses a title's per-team installer rows to avoid fan-out.
	catalogNamesByTitle = `
		SELECT si.title_id, MIN(fma.name) AS name
		FROM software_installers si
		JOIN fleet_maintained_apps fma
			ON fma.id = si.fleet_maintained_app_id AND fma.platform = 'darwin'
		WHERE si.title_id IS NOT NULL
		GROUP BY si.title_id
		HAVING COUNT(DISTINCT fma.name) = 1`

	// darwin bundle identifiers mapping to exactly one app name; shared ones are excluded.
	catalogNamesByIdentifier = `
		SELECT unique_identifier, MIN(name) AS name
		FROM fleet_maintained_apps
		WHERE platform = 'darwin'
		GROUP BY unique_identifier
		HAVING COUNT(DISTINCT name) = 1`

	mismatchedTitlesByInstallerLink = `
		SELECT st.id, fma.name
		FROM (` + catalogNamesByTitle + `) fma
		STRAIGHT_JOIN software_titles st ON st.id = fma.title_id
		WHERE st.name <> fma.name
		LIMIT ?`

	mismatchedSoftwareByInstallerLink = `
		SELECT s.id, fma.name
		FROM (` + catalogNamesByTitle + `) fma
		STRAIGHT_JOIN software s ON s.title_id = fma.title_id
		WHERE s.name <> fma.name
		LIMIT ?`

	mismatchedTitlesByIdentifier = `
		SELECT st.id, fma.name
		FROM (` + catalogNamesByIdentifier + `) fma
		STRAIGHT_JOIN software_titles st
			ON st.bundle_identifier = fma.unique_identifier AND st.additional_identifier = 0
		WHERE st.name <> fma.name
		LIMIT ?`

	mismatchedSoftwareByIdentifier = `
		SELECT s.id, fma.name
		FROM (` + catalogNamesByIdentifier + `) fma
		STRAIGHT_JOIN software s
			ON s.bundle_identifier = fma.unique_identifier
			AND s.source NOT IN ('ios_apps', 'ipados_apps')
		WHERE s.name <> fma.name
		LIMIT ?`

	updateSoftwareTitleNames = `UPDATE software_titles SET name = ? WHERE id IN (?) AND name <> ?`

	updateSoftwareNames = `UPDATE software SET name = ? WHERE id IN (?) AND name <> ?`
)

// renameWithRetry runs one rename statement, retrying the transient lock errors that
// concurrent software ingestion can produce. The batches are small and each runs in its own
// transaction, so the retry never widens the lock scope.
func (ds *Datastore) renameWithRetry(ctx context.Context, stmt string, args ...any) (int64, error) {
	var renamed int64
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, stmt, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "rename rows")
		}
		n, err := res.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "count renamed rows")
		}
		renamed = n
		return nil
	})
	return renamed, err
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
