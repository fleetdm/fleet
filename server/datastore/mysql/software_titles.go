package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) SoftwareTitleByID(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
	var teamFilter string
	if teamID != nil {
		teamFilter = fmt.Sprintf("sthc.team_id = %d", *teamID)
	} else {
		teamFilter = ds.whereFilterGlobalOrTeamIDByTeams(tmFilter, "sthc")
	}

	selectSoftwareTitleStmt := fmt.Sprintf(`
SELECT
	st.id,
	st.name,
	st.source,
	st.browser,
	SUM(sthc.hosts_count) as hosts_count,
	MAX(sthc.updated_at)  as counts_updated_at
FROM software_titles st
JOIN software_titles_host_counts sthc ON sthc.software_title_id = st.id
WHERE st.id = ? AND %s
AND sthc.hosts_count > 0
GROUP BY
	st.id,
	st.name,
	st.source,
	st.browser
	`, teamFilter,
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
) ([]fleet.SoftwareTitle, int, *fleet.PaginationMetadata, error) {
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

	dbReader := ds.reader(ctx)
	getTitlesStmt, args := selectSoftwareTitlesSQL(opt)
	// build the count statement before adding the pagination constraints to `getTitlesStmt`
	getTitlesCountStmt := fmt.Sprintf(`SELECT COUNT(DISTINCT s.id) FROM (%s) AS s`, getTitlesStmt)

	// grab titles that match the list options
	var titles []fleet.SoftwareTitle
	getTitlesStmt, args = appendListOptionsWithCursorToSQL(getTitlesStmt, args, &opt.ListOptions)
	// appendListOptionsWithCursorToSQL doesn't support multicolumn sort, so
	// we need to add it here
	getTitlesStmt = spliceSecondaryOrderBySoftwareTitlesSQL(getTitlesStmt, opt.ListOptions)
	if err := sqlx.SelectContext(ctx, dbReader, &titles, getTitlesStmt, args...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "select software titles")
	}

	// perform a second query to grab the counts
	var counts int
	if err := sqlx.GetContext(ctx, dbReader, &counts, getTitlesCountStmt, args...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "get software titles count")
	}

	// if we don't have any matching titles, there's no point trying to
	// find matching versions. Early return
	if len(titles) == 0 {
		return titles, counts, &fleet.PaginationMetadata{}, nil
	}

	// grab all the IDs to find matching versions below
	titleIDs := make([]uint, len(titles))
	// build an index to quickly access a title by it's ID
	titleIndex := make(map[uint]int, len(titles))
	for i, title := range titles {
		titleIDs[i] = title.ID
		titleIndex[title.ID] = i
	}

	// we grab matching versions separately and build the desired object in
	// the application logic. This is because we need to support MySQL 5.7
	// and there's no good way to do an aggregation that builds a structure
	// (like a JSON) object for nested arrays.
	getVersionsStmt, args, err := ds.selectSoftwareVersionsSQL(
		titleIDs,
		nil,
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
			titles[i].VersionsCount++
			titles[i].Versions = append(titles[i].Versions, version)
		}
	}

	var metaData *fleet.PaginationMetadata
	if opt.ListOptions.IncludeMetadata {
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opt.ListOptions.Page > 0}
		if len(titles) > int(opt.ListOptions.PerPage) {
			metaData.HasNextResults = true
			titles = titles[:len(titles)-1]
		}
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
	MAX(sthc.hosts_count) as hosts_count,
	MAX(sthc.updated_at) as counts_updated_at
FROM software_titles st
JOIN software_titles_host_counts sthc ON sthc.software_title_id = st.id
-- placeholder for JOIN on software/software_cve
%s
-- placeholder for optional extra WHERE filter
WHERE sthc.team_id = ? %s
AND sthc.hosts_count > 0
GROUP BY st.id`

	cveJoinType := "LEFT"
	if opt.VulnerableOnly {
		cveJoinType = "INNER"
	}

	args := []any{0}
	if opt.TeamID != nil {
		args[0] = *opt.TeamID
	}

	additionalWhere := ""
	match := opt.ListOptions.MatchQuery
	softwareJoin := ""
	if match != "" || opt.VulnerableOnly {
		softwareJoin = fmt.Sprintf(`
			JOIN software s ON s.title_id = st.id
			-- placeholder for changing the JOIN type to filter vulnerable software
			%s JOIN software_cve scve ON s.id = scve.software_id
		`, cveJoinType)
	}

	if match != "" {
		additionalWhere += " AND (st.name LIKE ? OR scve.cve LIKE ?)"
		match = likePattern(match)
		args = append(args, match, match)
	}

	stmt = fmt.Sprintf(stmt, softwareJoin, additionalWhere)
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
LEFT JOIN software_host_counts shc ON shc.software_id = s.id
LEFT JOIN software_cve scve ON shc.software_id = scve.software_id
WHERE s.title_id IN (?)
AND %s
AND shc.hosts_count > 0
GROUP BY s.id`

	extraSelect := ""
	if withCounts {
		extraSelect = "MAX(shc.hosts_count) AS hosts_count,"
	}

	selectVersionsStmt = fmt.Sprintf(selectVersionsStmt, extraSelect, teamFilter)

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
                st.id as software_title_id
            FROM software_titles st
            JOIN software s ON s.title_id = st.id
            JOIN host_software hs ON hs.software_id = s.id
            GROUP BY st.id`

		teamCountsStmt = `
            SELECT
                COUNT(DISTINCT hs.host_id),
                h.team_id,
                st.id as software_title_id
            FROM software_titles st
            JOIN software s ON s.title_id = st.id
            JOIN host_software hs ON hs.software_id = s.id
            INNER JOIN hosts h ON hs.host_id = h.id
            WHERE h.team_id IS NOT NULL AND hs.software_id > 0
            GROUP BY st.id, h.team_id`

		insertStmt = `
            INSERT INTO software_titles_host_counts
                (software_title_id, hosts_count, team_id, updated_at)
            VALUES
                %s
            ON DUPLICATE KEY UPDATE
                hosts_count = VALUES(hosts_count),
                updated_at = VALUES(updated_at)`

		valuesPart = `(?, ?, ?, ?),`

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
	stmtLabel := []string{"global", "team"}
	for i, countStmt := range []string{globalCountsStmt, teamCountsStmt} {
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
				sid    uint
			)

			if err := rows.Scan(&count, &teamID, &sid); err != nil {
				return ctxerr.Wrapf(ctx, err, "scan %s row into variables", stmtLabel[i])
			}

			args = append(args, sid, count, teamID, updatedAt)
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
