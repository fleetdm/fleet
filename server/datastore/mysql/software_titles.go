package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) SoftwareTitleByID(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
	var (
		teamFilter                            string // used to filter software titles host counts by team
		softwareInstallerGlobalOrTeamIDFilter string
		vppAppsTeamsGlobalOrTeamIDFilter      string
	)

	if teamID != nil {
		teamFilter = fmt.Sprintf("sthc.team_id = %d AND sthc.global_stats = 0", *teamID)
		softwareInstallerGlobalOrTeamIDFilter = fmt.Sprintf("si.global_or_team_id = %d", *teamID)
		vppAppsTeamsGlobalOrTeamIDFilter = fmt.Sprintf("vat.global_or_team_id = %d", *teamID)
	} else {
		teamFilter = ds.whereFilterGlobalOrTeamIDByTeams(tmFilter, "sthc")
		softwareInstallerGlobalOrTeamIDFilter = "TRUE"
		vppAppsTeamsGlobalOrTeamIDFilter = "TRUE"
	}

	// Select software title but filter out if the software has zero host counts
	// and it's not an installer or VPP app.
	selectSoftwareTitleStmt := fmt.Sprintf(`
SELECT
	st.id,
	st.name,
	st.source,
	st.browser,
	st.bundle_identifier,
	COALESCE(SUM(sthc.hosts_count), 0) AS hosts_count,
	MAX(sthc.updated_at) AS counts_updated_at,
	COUNT(si.id) as software_installers_count,
	COUNT(vat.adam_id) AS vpp_apps_count
FROM software_titles st
LEFT JOIN software_titles_host_counts sthc ON sthc.software_title_id = st.id AND sthc.hosts_count > 0 AND (%s)
LEFT JOIN software_installers si ON si.title_id = st.id AND %s
LEFT JOIN vpp_apps vap ON vap.title_id = st.id
LEFT JOIN vpp_apps_teams vat ON vat.adam_id = vap.adam_id AND vat.platform = vap.platform AND %s
WHERE st.id = ? AND
	(sthc.hosts_count > 0 OR vat.adam_id IS NOT NULL OR si.id IS NOT NULL)
GROUP BY
	st.id,
	st.name,
	st.source,
	st.browser,
	st.bundle_identifier
	`, teamFilter, softwareInstallerGlobalOrTeamIDFilter, vppAppsTeamsGlobalOrTeamIDFilter,
	)
	var title fleet.SoftwareTitle
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &title, selectSoftwareTitleStmt, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("SoftwareTitle").WithID(id)
		}
		return nil, ctxerr.Wrap(ctx, err, "get software title")
	}

	selectSoftwareVersionsStmt, args, err := ds.selectSoftwareVersionsSQL([]uint{id}, teamID, tmFilter, true)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building versions statement")
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &title.Versions, selectSoftwareVersionsStmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software title version")
	}

	title.VersionsCount = uint(len(title.Versions))
	return &title, nil
}

func (ds *Datastore) ListSoftwareTitles(
	ctx context.Context,
	opt fleet.SoftwareTitleListOptions,
	tmFilter fleet.TeamFilter,
) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
	if opt.ListOptions.After != "" {
		return nil, 0, nil, fleet.NewInvalidArgumentError("after", "not supported for software titles")
	}

	if len(strings.Split(opt.ListOptions.OrderKey, ",")) > 1 {
		return nil, 0, nil, fleet.NewInvalidArgumentError("order_key", "multicolumn order key not supported for software titles")
	}

	if opt.ListOptions.OrderKey == "" {
		opt.ListOptions.OrderKey = "hosts_count"
		opt.ListOptions.OrderDirection = fleet.OrderDescending
	}

	if (opt.MinimumCVSS > 0 || opt.MaximumCVSS > 0 || opt.KnownExploit) && !opt.VulnerableOnly {
		return nil, 0, nil, fleet.NewInvalidArgumentError("query", "min_cvss_score, max_cvss_score, and exploit can only be provided with vulnerable=true")
	}

	dbReader := ds.reader(ctx)
	getTitlesStmt, args := selectSoftwareTitlesSQL(opt)
	// build the count statement before adding the pagination constraints to `getTitlesStmt`
	getTitlesCountStmt := fmt.Sprintf(`SELECT COUNT(DISTINCT s.id) FROM (%s) AS s`, getTitlesStmt)

	// grab titles that match the list options
	type softwareTitle struct {
		fleet.SoftwareTitleListResult
		PackageSelfService        *bool   `db:"package_self_service"`
		PackageName               *string `db:"package_name"`
		PackageVersion            *string `db:"package_version"`
		PackageURL                *string `db:"package_url"`
		PackageInstallDuringSetup *bool   `db:"package_install_during_setup"`
		VPPAppSelfService         *bool   `db:"vpp_app_self_service"`
		VPPAppAdamID              *string `db:"vpp_app_adam_id"`
		VPPAppVersion             *string `db:"vpp_app_version"`
		VPPAppIconURL             *string `db:"vpp_app_icon_url"`
		VPPInstallDuringSetup     *bool   `db:"vpp_install_during_setup"`
	}
	var softwareList []*softwareTitle
	getTitlesStmt, args = appendListOptionsWithCursorToSQL(getTitlesStmt, args, &opt.ListOptions)
	// appendListOptionsWithCursorToSQL doesn't support multicolumn sort, so
	// we need to add it here
	getTitlesStmt = spliceSecondaryOrderBySoftwareTitlesSQL(getTitlesStmt, opt.ListOptions)
	if err := sqlx.SelectContext(ctx, dbReader, &softwareList, getTitlesStmt, args...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "select software titles")
	}

	// perform a second query to grab the counts
	var counts int
	if err := sqlx.GetContext(ctx, dbReader, &counts, getTitlesCountStmt, args...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "get software titles count")
	}

	// if we don't have any matching titles, there's no point trying to
	// find matching versions. Early return
	if len(softwareList) == 0 {
		return nil, counts, &fleet.PaginationMetadata{}, nil
	}

	// grab all the IDs to find matching versions below
	titleIDs := make([]uint, len(softwareList))
	// build an index to quickly access a title by its ID
	titleIndex := make(map[uint]int, len(softwareList))
	for i, title := range softwareList {
		// promote the package name and version to the proper destination fields
		if title.PackageName != nil {
			var version string
			if title.PackageVersion != nil {
				version = *title.PackageVersion
			}

			title.SoftwarePackage = &fleet.SoftwarePackageOrApp{
				Name:               *title.PackageName,
				Version:            version,
				SelfService:        title.PackageSelfService,
				PackageURL:         title.PackageURL,
				InstallDuringSetup: title.PackageInstallDuringSetup,
			}
		}

		// promote the VPP app id and version to the proper destination fields
		if title.VPPAppAdamID != nil {
			var version string
			if title.VPPAppVersion != nil {
				version = *title.VPPAppVersion
			}
			title.AppStoreApp = &fleet.SoftwarePackageOrApp{
				AppStoreID:         *title.VPPAppAdamID,
				Version:            version,
				SelfService:        title.VPPAppSelfService,
				IconURL:            title.VPPAppIconURL,
				InstallDuringSetup: title.VPPInstallDuringSetup,
			}
		}

		titleIDs[i] = title.ID
		titleIndex[title.ID] = i
	}

	// Grab the automatic install policies, if any exist
	policies, err := ds.getPoliciesBySoftwareTitleIDs(ctx, titleIDs, opt.TeamID)
	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "batch getting policies by software title IDs")
	}

	for _, p := range policies {
		if i, ok := titleIndex[p.TitleID]; ok {
			if softwareList[i].AppStoreApp != nil {
				softwareList[i].AppStoreApp.AutomaticInstallPolicies = append(softwareList[i].AppStoreApp.AutomaticInstallPolicies, p)
			} else {
				softwareList[i].SoftwarePackage.AutomaticInstallPolicies = append(softwareList[i].SoftwarePackage.AutomaticInstallPolicies, p)
			}
		}
	}

	// we grab matching versions separately and build the desired object in
	// the application logic. This is because we need to support MySQL 5.7
	// and there's no good way to do an aggregation that builds a structure
	// (like a JSON) object for nested arrays.
	getVersionsStmt, args, err := ds.selectSoftwareVersionsSQL(
		titleIDs,
		opt.TeamID,
		tmFilter,
		false,
	)
	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "build get versions stmt")
	}
	var versions []fleet.SoftwareVersion
	if err := sqlx.SelectContext(ctx, dbReader, &versions, getVersionsStmt, args...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "get software versions")
	}

	// append matching versions to titles
	for _, version := range versions {
		if i, ok := titleIndex[version.TitleID]; ok {
			softwareList[i].VersionsCount++
			softwareList[i].Versions = append(softwareList[i].Versions, version)
		}
	}

	var metaData *fleet.PaginationMetadata
	if opt.ListOptions.IncludeMetadata {
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opt.ListOptions.Page > 0}
		if len(softwareList) > int(opt.ListOptions.PerPage) { //nolint:gosec // dismiss G115
			metaData.HasNextResults = true
			softwareList = softwareList[:len(softwareList)-1]
		}
	}

	titles := make([]fleet.SoftwareTitleListResult, 0, len(softwareList))
	for _, st := range softwareList {
		st := st
		titles = append(titles, st.SoftwareTitleListResult)
	}

	return titles, counts, metaData, nil
}

// spliceSecondaryOrderBySoftwareTitlesSQL adds a secondary order by clause, splicing it into the
// existing order by clause. This is necessary because multicolumn sort is not
// supported by appendListOptionsWithCursorToSQL.
func spliceSecondaryOrderBySoftwareTitlesSQL(stmt string, opts fleet.ListOptions) string {
	if opts.OrderKey == "" {
		return stmt
	}
	k := strings.ToLower(opts.OrderKey)

	targetSubstr := "ASC"
	if opts.OrderDirection == fleet.OrderDescending {
		targetSubstr = "DESC"
	}

	var secondaryOrderBy string
	switch k {
	case "name":
		secondaryOrderBy = ", hosts_count DESC"
	default:
		secondaryOrderBy = ", name ASC"
	}

	if k != "source" {
		secondaryOrderBy += ", source ASC"
	}
	if k != "browser" {
		secondaryOrderBy += ", browser ASC"
	}

	return strings.Replace(stmt, targetSubstr, targetSubstr+secondaryOrderBy, 1)
}

func selectSoftwareTitlesSQL(opt fleet.SoftwareTitleListOptions) (string, []any) {
	stmt := `
SELECT
	st.id,
	st.name,
	st.source,
	st.browser,
	st.bundle_identifier,
	MAX(COALESCE(sthc.hosts_count, 0)) as hosts_count,
	MAX(COALESCE(sthc.updated_at, date('0001-01-01 00:00:00'))) as counts_updated_at,
	si.self_service as package_self_service,
	si.filename as package_name,
	si.version as package_version,
	si.url AS package_url,
	si.install_during_setup as package_install_during_setup,
	vat.self_service as vpp_app_self_service,
	vat.adam_id as vpp_app_adam_id,
	vat.install_during_setup as vpp_install_during_setup,
	vap.latest_version as vpp_app_version,
	vap.icon_url as vpp_app_icon_url
FROM software_titles st
LEFT JOIN software_installers si ON si.title_id = st.id AND %s
LEFT JOIN vpp_apps vap ON vap.title_id = st.id AND %s
LEFT JOIN vpp_apps_teams vat ON vat.adam_id = vap.adam_id AND vat.platform = vap.platform AND %s
LEFT JOIN software_titles_host_counts sthc ON sthc.software_title_id = st.id AND (%s)
-- placeholder for JOIN on software/software_cve
%s
-- placeholder for optional extra WHERE filter
WHERE %s
-- placeholder for filter based on software installed on hosts + software installers
AND (%s)
GROUP BY st.id, package_self_service, package_name, package_version, package_url, package_install_during_setup, vpp_app_self_service, vpp_app_adam_id, vpp_app_version, vpp_app_icon_url, vpp_install_during_setup`

	cveJoinType := "LEFT"
	if opt.VulnerableOnly {
		cveJoinType = "INNER"
	}

	countsJoin := "TRUE"
	softwareInstallersJoinCond := "TRUE"
	vppAppsJoinCond := "TRUE"
	vppAppsTeamsJoinCond := "TRUE"
	includeVPPAppsAndSoftwareInstallers := "TRUE"
	switch {
	case opt.TeamID == nil:
		countsJoin = "sthc.team_id = 0 AND sthc.global_stats = 1"
		// When opt.TeamID is nil (aka "All teams") we do not include VPP-apps/installers
		// that are not installed on any host.
		includeVPPAppsAndSoftwareInstallers = "FALSE"
	case *opt.TeamID == 0:
		countsJoin = "sthc.team_id = 0 AND sthc.global_stats = 0"
		softwareInstallersJoinCond = fmt.Sprintf("si.global_or_team_id = %d", *opt.TeamID)
		vppAppsTeamsJoinCond = fmt.Sprintf("vat.global_or_team_id = %d", *opt.TeamID)
	case *opt.TeamID > 0:
		countsJoin = fmt.Sprintf("sthc.team_id = %d AND sthc.global_stats = 0", *opt.TeamID)
		softwareInstallersJoinCond = fmt.Sprintf("si.global_or_team_id = %d", *opt.TeamID)
		vppAppsTeamsJoinCond = fmt.Sprintf("vat.global_or_team_id = %d", *opt.TeamID)
	}

	if opt.PackagesOnly {
		vppAppsJoinCond = "FALSE"
		vppAppsTeamsJoinCond = "FALSE"
	}

	additionalWhere := "TRUE"

	match := opt.ListOptions.MatchQuery
	softwareJoin := ""
	if match != "" || opt.VulnerableOnly {
		// if we do a match but not vulnerable only, we want a LEFT JOIN on
		// software because software installers may not have entries in software
		// for their software title. If we do want vulnerable only, then we have to
		// INNER JOIN because a CVE implies a specific software version.
		softwareJoin = fmt.Sprintf(`
			%s JOIN software s ON s.title_id = st.id
			-- placeholder for changing the JOIN type to filter vulnerable software
			%[1]s JOIN software_cve scve ON s.id = scve.software_id
		`, cveJoinType)
	}

	var args []any
	if opt.VulnerableOnly && (opt.KnownExploit || opt.MinimumCVSS > 0 || opt.MaximumCVSS > 0) {
		softwareJoin += `
			INNER JOIN cve_meta cm ON scve.cve = cm.cve
		`
		if opt.KnownExploit {
			softwareJoin += `
				AND cm.cisa_known_exploit = 1
			`
		}
		if opt.MinimumCVSS > 0 {
			softwareJoin += `
				AND cm.cvss_score >= ?
			`
			args = append(args, opt.MinimumCVSS)
		}

		if opt.MaximumCVSS > 0 {
			softwareJoin += `
				AND cm.cvss_score <= ?
			`
			args = append(args, opt.MaximumCVSS)
		}
	}

	if match != "" {
		additionalWhere = " (st.name LIKE ? OR scve.cve LIKE ?)"
		match = likePattern(match)
		args = append(args, match, match)
	}

	if opt.Platform != "" {
		platforms := strings.Split(strings.ReplaceAll(opt.Platform, "macos", "darwin"), ",")
		platformPlaceholders := strings.TrimSuffix(strings.Repeat("?,", len(platforms)), ",")

		additionalWhere += fmt.Sprintf(` AND (si.platform IN (%s) OR vap.platform IN (%s))`, platformPlaceholders, platformPlaceholders)
		args = slices.Grow(args, len(platformPlaceholders)*2)
		for _, platform := range platforms { // for software installers
			args = append(args, platform)
		}
		for _, platform := range platforms { // for VPP apps; could micro-optimize later by dropping non-Apple platforms
			args = append(args, platform)
		}
	}

	// default to "a software installer or VPP app exists", and see next condition.
	defaultFilter := fmt.Sprintf(`
		((si.id IS NOT NULL OR vat.adam_id IS NOT NULL) AND %s)
	`, includeVPPAppsAndSoftwareInstallers)

	// add software installed for hosts if we're not filtering for "available for install" only
	if !opt.AvailableForInstall {
		defaultFilter = ` ( ` + defaultFilter + ` OR sthc.hosts_count > 0 ) `
	}
	if opt.SelfServiceOnly {
		defaultFilter += ` AND ( si.self_service = 1 OR vat.self_service = 1 ) `
	}

	// if excluding fleet maintained apps, join on the fleet_library_apps table by bundle ID
	// and filter out any row from software_titles that has a matching row in fleet_library_apps.
	if opt.ExcludeFleetMaintainedApps {
		additionalWhere += " AND NOT EXISTS ( SELECT FALSE FROM fleet_library_apps AS fla WHERE fla.bundle_identifier = st.bundle_identifier )"
	}

	stmt = fmt.Sprintf(stmt, softwareInstallersJoinCond, vppAppsJoinCond, vppAppsTeamsJoinCond, countsJoin, softwareJoin, additionalWhere, defaultFilter)
	return stmt, args
}

func (ds *Datastore) selectSoftwareVersionsSQL(titleIDs []uint, teamID *uint, tmFilter fleet.TeamFilter, withCounts bool) (
	string, []any, error,
) {
	var teamFilter string
	if teamID != nil {
		teamFilter = fmt.Sprintf("shc.team_id = %d", *teamID)
	} else {
		teamFilter = ds.whereFilterGlobalOrTeamIDByTeams(tmFilter, "shc")
	}

	selectVersionsStmt := `
SELECT
	s.title_id,
	s.id, s.version,
	%s -- placeholder for optional host_counts
	CONCAT('[', GROUP_CONCAT(JSON_QUOTE(scve.cve) SEPARATOR ','), ']') as vulnerabilities
FROM software s
LEFT JOIN software_host_counts shc ON shc.software_id = s.id AND %s
LEFT JOIN software_cve scve ON shc.software_id = scve.software_id
WHERE s.title_id IN (?)
AND %s
AND shc.hosts_count > 0
GROUP BY s.id`

	extraSelect := ""
	if withCounts {
		extraSelect = "MAX(shc.hosts_count) AS hosts_count,"
	}

	countsJoin := "TRUE"
	switch {
	case teamID == nil:
		countsJoin = "shc.team_id = 0 AND shc.global_stats = 1"
	case *teamID == 0:
		countsJoin = "shc.team_id = 0 AND shc.global_stats = 0"
	case *teamID > 0:
		countsJoin = fmt.Sprintf("shc.team_id = %d AND shc.global_stats = 0", *teamID)
	}

	selectVersionsStmt = fmt.Sprintf(selectVersionsStmt, extraSelect, countsJoin, teamFilter)

	selectVersionsStmt, args, err := sqlx.In(selectVersionsStmt, titleIDs)
	if err != nil {
		return "", nil, fmt.Errorf("building sqlx.In query: %w", err)
	}
	return selectVersionsStmt, args, nil
}

// SyncHostsSoftwareTitles calculates the number of hosts having each
// software installed and stores that information in the software_titles_host_counts
// table.
func (ds *Datastore) SyncHostsSoftwareTitles(ctx context.Context, updatedAt time.Time) error {
	const (
		resetStmt = `
            UPDATE software_titles_host_counts
            SET hosts_count = 0, updated_at = ?`

		globalCountsStmt = `
            SELECT
                COUNT(DISTINCT hs.host_id),
                0 as team_id,
				1 as global_stats,
                st.id as software_title_id
            FROM software_titles st
            JOIN software s ON s.title_id = st.id
            JOIN host_software hs ON hs.software_id = s.id
            GROUP BY st.id`

		teamCountsStmt = `
            SELECT
                COUNT(DISTINCT hs.host_id),
                h.team_id,
				0 as global_stats,
                st.id as software_title_id
            FROM software_titles st
            JOIN software s ON s.title_id = st.id
            JOIN host_software hs ON hs.software_id = s.id
            INNER JOIN hosts h ON hs.host_id = h.id
            WHERE h.team_id IS NOT NULL AND hs.software_id > 0
            GROUP BY st.id, h.team_id`

		noTeamCountsStmt = `
			SELECT
				COUNT(DISTINCT hs.host_id),
				0 as team_id,
				0 as global_stats,
				st.id as software_title_id
			FROM software_titles st
			JOIN software s ON s.title_id = st.id
			JOIN host_software hs ON hs.software_id = s.id
			INNER JOIN hosts h ON hs.host_id = h.id
			WHERE h.team_id IS NULL AND hs.software_id > 0
			GROUP BY st.id`

		insertStmt = `
            INSERT INTO software_titles_host_counts
                (software_title_id, hosts_count, team_id, global_stats, updated_at)
            VALUES
                %s
            ON DUPLICATE KEY UPDATE
                hosts_count = VALUES(hosts_count),
                updated_at = VALUES(updated_at)`

		valuesPart = `(?, ?, ?, ?, ?),`

		cleanupOrphanedStmt = `
            DELETE sthc
            FROM
                software_titles_host_counts sthc
                LEFT JOIN software_titles st ON st.id = sthc.software_title_id
            WHERE
                st.id IS NULL`

		cleanupTeamStmt = `
            DELETE sthc
            FROM software_titles_host_counts sthc
            LEFT JOIN teams t ON t.id = sthc.team_id
            WHERE
                sthc.team_id > 0 AND
                t.id IS NULL`
	)

	// first, reset all counts to 0
	if _, err := ds.writer(ctx).ExecContext(ctx, resetStmt, updatedAt); err != nil {
		return ctxerr.Wrap(ctx, err, "reset all software_titles_host_counts to 0")
	}

	// next get a cursor for the global and team counts for each software
	stmtLabel := []string{"global", "team", "no_team"}
	for i, countStmt := range []string{globalCountsStmt, teamCountsStmt, noTeamCountsStmt} {
		rows, err := ds.reader(ctx).QueryContext(ctx, countStmt)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "read %s counts from host_software", stmtLabel[i])
		}
		defer rows.Close()

		// use a loop to iterate to prevent loading all in one go in memory, as it
		// could get pretty big at >100K hosts with 1000+ software each. Use a write
		// batch to prevent making too many single-row inserts.
		const batchSize = 100
		var batchCount int
		args := make([]interface{}, 0, batchSize*4)
		for rows.Next() {
			var (
				count  int
				teamID uint
				gstats bool
				sid    uint
			)

			if err := rows.Scan(&count, &teamID, &gstats, &sid); err != nil {
				return ctxerr.Wrapf(ctx, err, "scan %s row into variables", stmtLabel[i])
			}

			args = append(args, sid, count, teamID, gstats, updatedAt)
			batchCount++

			if batchCount == batchSize {
				values := strings.TrimSuffix(strings.Repeat(valuesPart, batchCount), ",")
				if _, err := ds.writer(ctx).ExecContext(ctx, fmt.Sprintf(insertStmt, values), args...); err != nil {
					return ctxerr.Wrapf(ctx, err, "insert %s batch into software_titles_host_counts", stmtLabel[i])
				}

				args = args[:0]
				batchCount = 0
			}
		}
		if batchCount > 0 {
			values := strings.TrimSuffix(strings.Repeat(valuesPart, batchCount), ",")
			if _, err := ds.writer(ctx).ExecContext(ctx, fmt.Sprintf(insertStmt, values), args...); err != nil {
				return ctxerr.Wrapf(ctx, err, "insert last %s batch into software_titles_host_counts", stmtLabel[i])
			}
		}
		if err := rows.Err(); err != nil {
			return ctxerr.Wrapf(ctx, err, "iterate over %s host_software counts", stmtLabel[i])
		}
		rows.Close()
	}

	// remove any software count row for software that don't exist anymore
	if _, err := ds.writer(ctx).ExecContext(ctx, cleanupOrphanedStmt); err != nil {
		return ctxerr.Wrap(ctx, err, "delete software_titles_host_counts for non-existing software")
	}

	// remove any software count row for teams that don't exist anymore
	if _, err := ds.writer(ctx).ExecContext(ctx, cleanupTeamStmt); err != nil {
		return ctxerr.Wrap(ctx, err, "delete software_titles_host_counts for non-existing teams")
	}
	return nil
}

func (ds *Datastore) UploadedSoftwareExists(ctx context.Context, bundleIdentifier string, teamID *uint) (bool, error) {
	stmt := `
SELECT
	1
FROM
	software_titles st JOIN software_installers si ON si.title_id = st.id
WHERE
	st.bundle_identifier = ? AND si.global_or_team_id = ?
	`
	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	var titleExists bool
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &titleExists, stmt, bundleIdentifier, tmID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}

		return false, ctxerr.Wrap(ctx, err, "checking if software installer exists")
	}

	return titleExists, nil
}
