package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

// ReconcileWindowsMaintainedAppSoftwareTitles collapses versioned Windows program
// titles onto the canonical FMA title (Windows has no bundle identifier to join on,
// so the program name is the only key). Unlike the darwin passes above, which are a
// rename, this is a merge: the versioned title and the installer's title are separate
// rows, so software is re-pointed from one to the other.
//
// References to the merged-away title are moved onto the destination and the title is
// then deleted, which collapses duplicate Windows program titles.
func (ds *Datastore) ReconcileWindowsMaintainedAppSoftwareTitles(ctx context.Context) error {
	fmaMatches, err := ds.GetWindowsFMAMatches(ctxdb.RequirePrimary(ctx, true))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get windows FMA matches for reconcile")
	}

	// Find the work for every app in one scan, then write only where there is any. Doing
	// this per app instead would open a transaction for each one just to discover there
	// is nothing to merge, which is the steady state: ingestion already files new
	// software under the right title, so this pass usually finds nothing.
	allPrefixes := windowsFMAPrefixes(fmaMatches)
	staleByDestination, err := ds.staleWindowsTitlesByDestination(ctx, fmaMatches, allPrefixes)
	if err != nil {
		return err
	}

	for destinationID, staleIDs := range staleByDestination {
		if err := ds.mergeWindowsFMATitle(ctx, destinationID, staleIDs); err != nil {
			return ctxerr.Wrapf(ctx, err, "merge onto windows software title %d", destinationID)
		}
	}

	return nil
}

// staleWindowsTitlesByDestination returns, per destination software title, the
// inventory-only titles whose reported names belong to it. One scan covers every app:
// the SQL narrows to names that could plausibly match any of them, and
// matchWindowsFMATitle then makes the per-name decision so prefix precedence and the
// cross-app ambiguity rule agree with the ingestion path exactly.
func (ds *Datastore) staleWindowsTitlesByDestination(
	ctx context.Context,
	fmaMatches []fleet.MaintainedApp,
	allPrefixes []windowsFMAPrefix,
) (map[uint][]uint, error) {
	var nameConds []string
	var args []any
	for i := range fmaMatches {
		for _, prefix := range fmaMatches[i].WinMatchPrefixes() {
			// Escape LIKE wildcards so a name containing % or _ can't widen the match. The
			// ESCAPE clause is stated explicitly rather than relying on the default.
			escaped := prefix
			for _, c := range []string{`\`, `%`, `_`} {
				escaped = strings.ReplaceAll(escaped, c, `\`+c)
			}
			nameConds = append(nameConds, `st.name = ? OR st.name LIKE ? ESCAPE '\\'`)
			args = append(args, prefix, escaped+" %")
		}
	}
	if len(nameConds) == 0 {
		// No app contributes a usable name, so nothing can match.
		return nil, nil
	}

	// Inventory-only versioned titles. Exclude titles with an upgrade code (those join
	// through it instead) and titles owned by an installer/VPP/in-house app, whose links
	// are the authoritative mapping. That last exclusion also removes every destination,
	// since a destination is by definition installer-owned.
	staleStmt := `
		SELECT st.id, st.name
		FROM software_titles st
		WHERE st.source = 'programs' AND st.extension_for = ''
			AND (` + strings.Join(nameConds, " OR ") + `)
			AND (st.upgrade_code IS NULL OR st.upgrade_code = '')
			AND NOT EXISTS (SELECT 1 FROM software_installers si WHERE si.title_id = st.id)
			AND NOT EXISTS (SELECT 1 FROM vpp_apps va WHERE va.title_id = st.id)
			AND NOT EXISTS (SELECT 1 FROM in_house_apps iha WHERE iha.title_id = st.id)`

	var candidates []struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctxdb.RequirePrimary(ctx, true)), &candidates, staleStmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select stale windows titles")
	}

	staleByDestination := make(map[uint][]uint)
	var unmatched []string
	for _, c := range candidates {
		match, ok := matchWindowsFMATitle(c.Name, allPrefixes)
		if !ok || match.titleID == c.ID {
			if !ok {
				unmatched = append(unmatched, c.Name)
			}
			continue
		}
		staleByDestination[match.titleID] = append(staleByDestination[match.titleID], c.ID)
	}

	if len(unmatched) > 0 && ds.logger != nil {
		ds.logger.DebugContext(ctx, "windows software titles matched a maintained app by name but were not merged",
			"names", unmatched,
		)
	}

	return staleByDestination, nil
}

// mergeWindowsFMATitle moves every reference to staleIDs onto destinationID and deletes
// the emptied titles, in one transaction so a title is never deleted before its
// references have moved. One transaction per destination keeps the locks short; the
// destinations are independent of each other.
func (ds *Datastore) mergeWindowsFMATitle(ctx context.Context, destinationID uint, staleIDs []uint) error {
	if destinationID == 0 || len(staleIDs) == 0 {
		return nil
	}

	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Re-point everything that references the stale titles onto the destination, then
		// delete them. This mirrors the DedupeWindowsProgramTitlesFromUpgradeCode
		// migration, which performs the same merge.
		//
		// software keeps its own name, so hosts still report the version they actually
		// have installed. The tables with a unique key on (team, title) use UPDATE IGNORE
		// because the destination may already have a row for that team; the delete below
		// then cascades away whatever was skipped.
		//
		// software_installers, vpp_apps and in_house_apps are absent by construction: the
		// scan above excludes any title they reference.
		repoint := []struct {
			label string
			stmt  string
		}{
			{"software", `UPDATE software SET title_id = ? WHERE title_id IN (?)`},
			{"host software installs", `UPDATE host_software_installs SET software_title_id = ? WHERE software_title_id IN (?)`},
			{"upcoming install activities", `UPDATE software_install_upcoming_activities SET software_title_id = ? WHERE software_title_id IN (?)`},
			{"patch policies", `UPDATE IGNORE policies SET patch_software_title_id = ? WHERE patch_software_title_id IN (?)`},
			{"update schedules", `UPDATE IGNORE software_update_schedules SET title_id = ? WHERE title_id IN (?)`},
			{"display names", `UPDATE IGNORE software_title_display_names SET software_title_id = ? WHERE software_title_id IN (?)`},
			{"icons", `UPDATE IGNORE software_title_icons SET software_title_id = ? WHERE software_title_id IN (?)`},
			{"team pins", `UPDATE IGNORE software_title_team_pins SET title_id = ? WHERE title_id IN (?)`},
		}
		for _, r := range repoint {
			stmt, repointArgs, err := sqlx.In(r.stmt, destinationID, staleIDs)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "build re-point statement for %s", r.label)
			}
			if _, err := tx.ExecContext(ctx, stmt, repointArgs...); err != nil {
				return ctxerr.Wrapf(ctx, err, "re-point %s to canonical windows title", r.label)
			}
		}

		// software_titles_host_counts has no foreign key, so it would be left behind by
		// the delete. The counts cron recomputes it.
		countsStmt, countsArgs, err := sqlx.In(`DELETE FROM software_titles_host_counts WHERE software_title_id IN (?)`, staleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build delete stale host counts statement")
		}
		if _, err := tx.ExecContext(ctx, countsStmt, countsArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete stale windows title host counts")
		}

		// Re-assert that nothing owns these titles. The scan above is a non-locking read,
		// so an installer, VPP app or in-house app can be attached to a candidate between
		// choosing it and deleting it. Those links are ON DELETE SET NULL, so deleting
		// such a title would leave the new owner pointing at nothing. Repeating the
		// checks here keeps that invariant on the statement that would do the damage.
		titlesStmt, titlesArgs, err := sqlx.In(`
			DELETE FROM software_titles
			WHERE id IN (?)
				AND NOT EXISTS (SELECT 1 FROM software_installers si WHERE si.title_id = software_titles.id)
				AND NOT EXISTS (SELECT 1 FROM vpp_apps va WHERE va.title_id = software_titles.id)
				AND NOT EXISTS (SELECT 1 FROM in_house_apps iha WHERE iha.title_id = software_titles.id)`,
			staleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build delete stale titles statement")
		}
		if _, err := tx.ExecContext(ctx, titlesStmt, titlesArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete stale windows titles")
		}

		return nil
	}); err != nil {
		return err
	}

	if ds.logger != nil {
		ds.logger.InfoContext(ctx, "merged Windows software titles into the title owned by a Fleet-maintained app installer",
			"destination_title_id", destinationID,
			"merged_title_ids", staleIDs,
			"merged_count", len(staleIDs),
		)
	}

	return nil
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

// GetWindowsFMAMatches returns the Windows FMAs that name matching should consider,
// populated with only the fields MaintainedApp.WinMatchPrefixes needs.
func (ds *Datastore) GetWindowsFMAMatches(ctx context.Context) ([]fleet.MaintainedApp, error) {
	// Restricted to FMAs that are actually added somewhere, via their installer link.
	// Prefix matching on a program name is only as precise as the name allows: the FMA
	// manifests pair it with a publisher check, which fleet_maintained_apps does not
	// carry, so requiring a deliberate install is what bounds the blast radius.
	//
	// The installer link also supplies the title to merge onto. An app's per-team
	// installer rows normally share one title, so they collapse here; an app somehow
	// spanning several is ambiguous and excluded rather than guessed at, matching how
	// the darwin passes above handle a bundle identifier shared by multiple FMAs.
	//
	// platform is selected even though it is also filtered on, because WinMatchPrefixes
	// checks it and would otherwise return nothing for every row.
	query := `
		SELECT fma.name, fma.unique_identifier, fma.platform,
			MIN(si.title_id) AS software_title_id, MIN(st.name) AS title_name
		FROM fleet_maintained_apps fma
		JOIN software_installers si
			ON si.fleet_maintained_app_id = fma.id AND si.title_id IS NOT NULL
		JOIN software_titles st ON st.id = si.title_id
		WHERE fma.platform = 'windows' AND fma.name != ''
		GROUP BY fma.id, fma.name, fma.unique_identifier, fma.platform
		HAVING COUNT(DISTINCT si.title_id) = 1`

	var apps []fleet.MaintainedApp
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &apps, query); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "query Windows FMA matches")
	}

	return apps, nil
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
