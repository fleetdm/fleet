package mysql

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListOSVulnerabilitiesByOS(ctx context.Context, osID uint) ([]fleet.OSVulnerability, error) {
	r := []fleet.OSVulnerability{}

	stmt := `
		SELECT
			operating_system_id,
			cve,
			resolved_in_version,
			source
		FROM operating_system_vulnerabilities
		WHERE operating_system_id = ?
	`

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &r, stmt, osID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error executing SQL statement")
	}

	return r, nil
}

func (ds *Datastore) ListVulnsByOsNameAndVersion(ctx context.Context, name, version string, includeCVSS bool, teamID *uint, maxVulnerabilities *int) (fleet.OSVulnerabilitiesWithCount, error) {
	// Validate maxVulnerabilities parameter
	if maxVulnerabilities != nil && *maxVulnerabilities < 0 {
		return fleet.OSVulnerabilitiesWithCount{}, fleet.NewInvalidArgumentError("max_vulnerabilities", "max_vulnerabilities must be >= 0")
	}

	var tmID uint
	var linuxTeamFilter string
	baseArgs := []any{name, version, name, version}
	if teamID != nil {
		tmID = *teamID
		linuxTeamFilter = "AND osvv.team_id = ?"
		baseArgs = append(baseArgs, tmID)
	} else {
		// When no teamID is specified, query the "all teams" aggregated data (team_id = NULL)
		linuxTeamFilter = "AND osvv.team_id IS NULL"
	}

	if !includeCVSS {
		// Simple query without CVSS metadata
		baseCTE := `
		WITH all_vulns AS (
			SELECT
				osv.cve,
				MIN(osv.created_at) created_at
			FROM operating_system_vulnerabilities osv
			JOIN operating_systems os ON os.id = osv.operating_system_id
				AND os.name = ? AND os.version = ?
			GROUP BY osv.cve

			UNION

			SELECT DISTINCT
				osvv.cve,
				MIN(osvv.created_at) created_at
			FROM
				operating_system_version_vulnerabilities osvv
				JOIN operating_systems os ON os.os_version_id = osvv.os_version_id
			WHERE
				os.name = ?
				AND os.version = ?
				` + linuxTeamFilter + `
			GROUP BY osvv.cve
		)
		`

		var stmt string
		args := make([]any, len(baseArgs))
		copy(args, baseArgs)

		switch {
		case maxVulnerabilities != nil && *maxVulnerabilities == 0:
			// Return only count
			stmt = baseCTE + `
			SELECT
				'' as cve,
				NOW() as created_at,
				COUNT(*) as total_count
			FROM all_vulns`

		case maxVulnerabilities != nil:
			// Limit with ROW_NUMBER()
			stmt = baseCTE + `
			SELECT
				cve,
				created_at,
				total_count
			FROM (
				SELECT
					cve,
					created_at,
					ROW_NUMBER() OVER (ORDER BY cve) as rn,
					COUNT(*) OVER () as total_count
				FROM all_vulns
			) ranked
			WHERE rn <= ?`
			args = append(args, *maxVulnerabilities)

		default:
			// Return all with count
			stmt = baseCTE + `
			SELECT
				cve,
				created_at,
				COUNT(*) OVER () as total_count
			FROM all_vulns`
		}

		type simpleResult struct {
			CVE        string    `db:"cve"`
			CreatedAt  time.Time `db:"created_at"`
			TotalCount int       `db:"total_count"`
		}

		var results []simpleResult
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, stmt, args...); err != nil {
			return fleet.OSVulnerabilitiesWithCount{}, ctxerr.Wrap(ctx, err, "error executing SQL statement")
		}

		totalCount := 0
		vulns := make(fleet.Vulnerabilities, 0)
		for _, r := range results {
			totalCount = r.TotalCount
			if r.CVE != "" { // Skip the count-only row when max=0
				vulns = append(vulns, fleet.CVE{
					CVE:       r.CVE,
					CreatedAt: r.CreatedAt,
				})
			}
		}

		return fleet.OSVulnerabilitiesWithCount{
			Vulnerabilities: vulns,
			Count:           totalCount,
		}, nil
	}

	// Query with CVSS metadata
	baseCTE := `
	WITH all_vulns AS (
		SELECT
			v.cve,
			MIN(v.created_at) created_at,
			GROUP_CONCAT(DISTINCT v.resolved_in_version SEPARATOR ',') resolved_in_version
		FROM operating_system_vulnerabilities v
		JOIN operating_systems os ON os.id = v.operating_system_id
			AND os.name = ? AND os.version = ?
		GROUP BY v.cve

		UNION

		SELECT DISTINCT
			osvv.cve,
			MIN(osvv.created_at) created_at,
			GROUP_CONCAT(DISTINCT osvv.resolved_in_version SEPARATOR ',') resolved_in_version
		FROM
			operating_system_version_vulnerabilities osvv
			JOIN operating_systems os ON os.os_version_id = osvv.os_version_id
		WHERE
			os.name = ?
			AND os.version = ?
			` + linuxTeamFilter + `
		GROUP BY osvv.cve
	)
	`

	var stmt string
	args := make([]any, len(baseArgs))
	copy(args, baseArgs)

	switch {
	case maxVulnerabilities != nil && *maxVulnerabilities == 0:
		// Return only count
		stmt = baseCTE + `
		SELECT
			'' as cve,
			NULL as cvss_score,
			NULL as epss_probability,
			NULL as cisa_known_exploit,
			NULL as cve_published,
			NULL as description,
			NULL as resolved_in_version,
			NOW() as created_at,
			COUNT(*) as total_count
		FROM all_vulns`

	case maxVulnerabilities != nil:
		// Limit with ROW_NUMBER()
		stmt = baseCTE + `
		SELECT
			osv.cve,
			cm.cvss_score,
			cm.epss_probability,
			cm.cisa_known_exploit,
			cm.published as cve_published,
			cm.description,
			osv.resolved_in_version,
			osv.created_at,
			total_count
		FROM (
			SELECT
				cve,
				created_at,
				resolved_in_version,
				ROW_NUMBER() OVER (ORDER BY cve) as rn,
				COUNT(*) OVER () as total_count
			FROM all_vulns
		) osv
		LEFT JOIN cve_meta cm ON cm.cve = osv.cve
		WHERE rn <= ?`
		args = append(args, *maxVulnerabilities)

	default:
		// Return all with count
		stmt = baseCTE + `
		SELECT
			osv.cve,
			cm.cvss_score,
			cm.epss_probability,
			cm.cisa_known_exploit,
			cm.published as cve_published,
			cm.description,
			osv.resolved_in_version,
			osv.created_at,
			COUNT(*) OVER () as total_count
		FROM all_vulns osv
		LEFT JOIN cve_meta cm ON cm.cve = osv.cve`
	}

	type cvssResult struct {
		CVE               string     `db:"cve"`
		CVSSScore         *float64   `db:"cvss_score"`
		EPSSProbability   *float64   `db:"epss_probability"`
		CISAKnownExploit  *bool      `db:"cisa_known_exploit"`
		CVEPublished      *time.Time `db:"cve_published"`
		Description       *string    `db:"description"`
		ResolvedInVersion *string    `db:"resolved_in_version"`
		CreatedAt         time.Time  `db:"created_at"`
		TotalCount        int        `db:"total_count"`
	}

	var results []cvssResult
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, stmt, args...); err != nil {
		return fleet.OSVulnerabilitiesWithCount{}, ctxerr.Wrap(ctx, err, "error executing SQL statement")
	}

	totalCount := 0
	vulns := make(fleet.Vulnerabilities, 0)
	for _, r := range results {
		totalCount = r.TotalCount
		if r.CVE != "" { // Skip the count-only row when max=0
			vulns = append(vulns, fleet.CVE{
				CVE:               r.CVE,
				CreatedAt:         r.CreatedAt,
				CVSSScore:         &r.CVSSScore,
				EPSSProbability:   &r.EPSSProbability,
				CISAKnownExploit:  &r.CISAKnownExploit,
				CVEPublished:      &r.CVEPublished,
				Description:       &r.Description,
				ResolvedInVersion: &r.ResolvedInVersion,
			})
		}
	}

	return fleet.OSVulnerabilitiesWithCount{
		Vulnerabilities: vulns,
		Count:           totalCount,
	}, nil
}

func (ds *Datastore) InsertOSVulnerabilities(ctx context.Context, vulnerabilities []fleet.OSVulnerability, source fleet.VulnerabilitySource) (int64, error) {
	var args []interface{}

	if len(vulnerabilities) == 0 {
		return 0, nil
	}

	values := strings.TrimSuffix(strings.Repeat("(?,?,?,?),", len(vulnerabilities)), ",")
	sql := fmt.Sprintf(`INSERT IGNORE INTO operating_system_vulnerabilities (operating_system_id, cve, source, resolved_in_version) VALUES %s`, values)

	for _, v := range vulnerabilities {
		args = append(args, v.OSID, v.CVE, source, v.ResolvedInVersion)
	}
	res, err := ds.writer(ctx).ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insert operating system vulnerabilities")
	}
	count, _ := res.RowsAffected()

	return count, nil
}

func (ds *Datastore) InsertOSVulnerability(ctx context.Context, v fleet.OSVulnerability, s fleet.VulnerabilitySource) (bool, error) {
	if v.CVE == "" {
		return false, fmt.Errorf("inserting operating system vulnerability: CVE cannot be empty %#v", v)
	}

	var args []interface{}

	// statement assumes a unique index on (host_id, cve)
	sqlStmt := `
		INSERT INTO operating_system_vulnerabilities (
			operating_system_id,
			cve,
			source,
			resolved_in_version
		) VALUES (?,?,?,?)
		ON DUPLICATE KEY UPDATE
			operating_system_id = VALUES(operating_system_id),
			source = VALUES(source),
			resolved_in_version = VALUES(resolved_in_version),
			updated_at = NOW()
	`

	args = append(args, v.OSID, v.CVE, s, v.ResolvedInVersion)

	res, err := ds.writer(ctx).ExecContext(ctx, sqlStmt, args...)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "insert operating system vulnerability")
	}

	// inserts affect one row, updates affect 0 or 2; we don't care which because timestamp may not change if we
	// recently inserted the vuln and changed nothing else; see insertOnDuplicateDidInsertOrUpdate for context
	affected, _ := res.RowsAffected()
	lastID, _ := res.LastInsertId()
	return lastID != 0 && affected == 1, nil
}

func (ds *Datastore) DeleteOSVulnerabilities(ctx context.Context, vulnerabilities []fleet.OSVulnerability) error {
	if len(vulnerabilities) == 0 {
		return nil
	}

	sql := fmt.Sprintf(
		`DELETE FROM operating_system_vulnerabilities WHERE (operating_system_id, cve) IN (%s)`,
		strings.TrimSuffix(strings.Repeat("(?,?),", len(vulnerabilities)), ","),
	)

	var args []interface{}
	for _, v := range vulnerabilities {
		args = append(args, v.OSID, v.CVE)
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, sql, args...); err != nil {
		return ctxerr.Wrapf(ctx, err, "deleting operating system vulnerabilities")
	}
	return nil
}

func (ds *Datastore) DeleteOutOfDateOSVulnerabilities(ctx context.Context, src fleet.VulnerabilitySource, olderThan time.Time) error {
	// Note: operating_system_version_vulnerabilities cleanup is handled automatically
	// by RefreshOSVersionVulnerabilities, which removes stale entries during its refresh
	deleteStmt := `
		DELETE FROM operating_system_vulnerabilities
		WHERE source = ? AND updated_at < ?
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, deleteStmt, src, olderThan); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting out of date operating system vulnerabilities")
	}
	return nil
}

func (ds *Datastore) ListKernelsByOS(ctx context.Context, osVersionID uint, teamID *uint) ([]*fleet.Kernel, error) {
	var kernels []*fleet.Kernel

	stmt := `
SELECT DISTINCT
	software.id AS id,
	software_cve.cve AS cve,
	software.version AS version,
    SUM(kernel_host_counts.hosts_count) AS hosts_count
FROM
	software
	LEFT JOIN software_cve ON software.id = software_cve.software_id
	JOIN kernel_host_counts ON kernel_host_counts.software_id = software.id
WHERE
	kernel_host_counts.os_version_id = ?
	AND kernel_host_counts.hosts_count > 0
    %s
GROUP BY id, cve, version
`

	var tmID uint
	var teamFilter string
	args := []any{osVersionID}
	if teamID != nil {
		tmID = *teamID
		teamFilter = "AND kernel_host_counts.team_id = ?"
		args = append(args, tmID)
	}

	stmt = fmt.Sprintf(stmt, teamFilter)

	var results []struct {
		ID         uint    `db:"id"`
		CVE        *string `db:"cve"`
		Version    string  `db:"version"`
		HostsCount uint    `db:"hosts_count"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing kernels by OS name")
	}

	kernelSet := make(map[uint]*fleet.Kernel)

	for _, result := range results {
		k, ok := kernelSet[result.ID]
		if !ok {
			kernel := &fleet.Kernel{
				ID:         result.ID,
				Version:    result.Version,
				HostsCount: result.HostsCount,
			}

			kernelSet[kernel.ID] = kernel
			k = kernel
		}

		if result.CVE != nil {
			k.Vulnerabilities = append(k.Vulnerabilities, *result.CVE)
		}

	}
	for _, kernel := range kernelSet {
		kernels = append(kernels, kernel)
	}
	return kernels, nil
}

func (ds *Datastore) InsertKernelSoftwareMapping(ctx context.Context) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE kernel_host_counts SET hosts_count = 0`)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "zero out existing kernel hosts counts")
	}

	statsStmt := `
INSERT INTO kernel_host_counts (software_title_id, software_id, os_version_id, hosts_count, team_id)
	SELECT
		software_titles.id AS software_title_id,
		software.id AS software_id,
		operating_systems.os_version_id AS os_version_id,
		COUNT(host_operating_system.host_id) AS hosts_count,
		COALESCE(hosts.team_id, 0) AS team_id
	FROM
		software_titles
		JOIN software ON software.title_id = software_titles.id
		JOIN host_software ON host_software.software_id = software.id
		JOIN host_operating_system ON host_operating_system.host_id = host_software.host_id
		JOIN operating_systems ON operating_systems.id = host_operating_system.os_id
		JOIN hosts ON hosts.id = host_software.host_id
	WHERE
		software_titles.is_kernel = TRUE
	GROUP BY
		software_title_id,
		software_id,
		os_version_id,
		team_id
ON DUPLICATE KEY UPDATE
	hosts_count=VALUES(hosts_count)
	`

	_, err = ds.writer(ctx).ExecContext(ctx, statsStmt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "insert kernel software mapping")
	}

	_, err = ds.writer(ctx).ExecContext(ctx, `DELETE k FROM kernel_host_counts k LEFT JOIN software ON k.software_id = software.id WHERE software.id IS NULL`)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "clean up orphan kernels by software id")
	}

	_, err = ds.writer(ctx).ExecContext(ctx, `DELETE k FROM kernel_host_counts k LEFT JOIN operating_systems ON k.os_version_id = operating_systems.os_version_id WHERE operating_systems.id IS NULL`)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "clean up orphan kernels by os version id")
	}

	// Refresh the pre-aggregated OS version vulnerabilities table
	if err := ds.RefreshOSVersionVulnerabilities(ctx); err != nil {
		return ctxerr.Wrap(ctx, err, "refresh os version vulnerabilities after kernel mapping update")
	}

	return nil
}

// RefreshOSVersionVulnerabilities refreshes the pre-aggregated operating_system_version_vulnerabilities table
// with current data from kernel_host_counts and software_cve.
// This function completely refreshes the table and removes any stale entries.
// This function should be called after:
//   - InsertKernelSoftwareMapping (when kernel_host_counts is updated)
func (ds *Datastore) RefreshOSVersionVulnerabilities(ctx context.Context) error {
	// Capture timestamp at start - we'll use this to mark all refreshed rows
	// and clean up any rows that weren't touched (stale data)
	updatedAt := time.Now()

	// Refresh per-team Linux kernel vulnerabilities
	_, err := ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO operating_system_version_vulnerabilities
			(os_version_id, cve, team_id, source, resolved_in_version, created_at)
		SELECT
			khc.os_version_id,
			sc.cve,
			khc.team_id,
			sc.source,
			sc.resolved_in_version,
			MIN(sc.created_at) as created_at
		FROM kernel_host_counts khc
		JOIN software_cve sc ON sc.software_id = khc.software_id
		WHERE khc.hosts_count > 0
		GROUP BY khc.os_version_id, sc.cve, khc.team_id, sc.source, sc.resolved_in_version
		ON DUPLICATE KEY UPDATE
			source = VALUES(source),
			resolved_in_version = VALUES(resolved_in_version),
			created_at = VALUES(created_at),
			updated_at = ?
	`, updatedAt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "refresh per-team OS version vulnerabilities")
	}

	// Refresh "all teams" aggregated Linux kernel vulnerabilities (team_id = NULL)
	_, err = ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO operating_system_version_vulnerabilities
			(os_version_id, cve, team_id, source, resolved_in_version, created_at)
		SELECT
			khc.os_version_id,
			sc.cve,
			NULL as team_id,
			sc.source,
			sc.resolved_in_version,
			MIN(sc.created_at) as created_at
		FROM kernel_host_counts khc
		JOIN software_cve sc ON sc.software_id = khc.software_id
		WHERE khc.hosts_count > 0
		GROUP BY khc.os_version_id, sc.cve, sc.source, sc.resolved_in_version
		ON DUPLICATE KEY UPDATE
			source = VALUES(source),
			resolved_in_version = VALUES(resolved_in_version),
			created_at = VALUES(created_at),
			updated_at = ?
	`, updatedAt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "refresh all-teams OS version vulnerabilities")
	}

	// Clean up stale entries - any rows not touched by this refresh are outdated
	_, err = ds.writer(ctx).ExecContext(ctx, `
		DELETE FROM operating_system_version_vulnerabilities
		WHERE updated_at < ?
	`, updatedAt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "clean up stale OS version vulnerabilities")
	}

	return nil
}

// ListVulnsByMultipleOSVersions - Optimized batch query to fetch vulnerabilities for multiple OS versions
// This replaces the previous N+1 pattern with efficient batch queries, providing performance improvement
// for large datasets.
func (ds *Datastore) ListVulnsByMultipleOSVersions(
	ctx context.Context,
	osVersions []fleet.OSVersion,
	includeCVSS bool,
	teamID *uint,
	maxVulnerabilities *int,
) (map[string]fleet.OSVulnerabilitiesWithCount, error) {
	result := make(map[string]fleet.OSVulnerabilitiesWithCount)
	if len(osVersions) == 0 {
		return result, nil
	}

	// Validate maxVulnerabilities parameter
	if maxVulnerabilities != nil && *maxVulnerabilities < 0 {
		return nil, fleet.NewInvalidArgumentError("max_vulnerabilities", "max_vulnerabilities must be >= 0")
	}

	// Step 1: Separate Linux from non-Linux OS versions
	// For Linux: we'll query by os_version_id directly (no need to expand to os.id)
	// For non-Linux: we need to fetch os.id values to query operating_system_vulnerabilities table

	// Track unique Linux os_version_ids and their keys
	linuxOSVersionMap := make(map[uint]string) // os_version_id -> "name-version" key
	linuxOSVersionIDs := make([]uint, 0)       // unique os_version_id values for Linux

	// Track non-Linux OS info for database lookup
	nonLinuxOSVersions := make([]fleet.OSVersion, 0)
	nonLinuxOSIDMap := make(map[uint]string) // os.id -> "name-version" key
	nonLinuxOSIDs := make([]uint, 0)         // os.id values for non-Linux

	// Separate Linux from non-Linux and track unique os_version_ids for Linux
	for _, os := range osVersions {
		key := fmt.Sprintf("%s-%s", os.NameOnly, os.Version)

		if fleet.IsLinux(os.Platform) {
			// For Linux, track by os_version_id (no need to fetch os.id)
			if _, exists := linuxOSVersionMap[os.OSVersionID]; !exists {
				linuxOSVersionMap[os.OSVersionID] = key
				linuxOSVersionIDs = append(linuxOSVersionIDs, os.OSVersionID)
				// Initialize result map entry for Linux
				if _, exists := result[key]; !exists {
					result[key] = fleet.OSVulnerabilitiesWithCount{
						Vulnerabilities: make([]fleet.CVE, 0),
						Count:           0,
					}
				}
			}
		} else {
			// For non-Linux, we'll need to fetch os.id values
			nonLinuxOSVersions = append(nonLinuxOSVersions, os)
		}
	}

	// Fetch OS IDs for non-Linux platforms
	if len(nonLinuxOSVersions) > 0 {
		tuples := make([]string, 0, len(nonLinuxOSVersions))
		args := make([]any, 0, len(nonLinuxOSVersions)*2)

		for _, os := range nonLinuxOSVersions {
			tuples = append(tuples, "(?, ?)")
			args = append(args, os.NameOnly, os.Version)
		}

		stmt := `
			SELECT id, name, version
			FROM operating_systems
			WHERE (name, version) IN (` + strings.Join(tuples, ", ") + `)`

		var osResults []struct {
			ID      uint   `db:"id"`
			Name    string `db:"name"`
			Version string `db:"version"`
		}

		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &osResults, stmt, args...); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "batch fetch OS IDs for non-Linux")
		}

		for _, r := range osResults {
			key := fmt.Sprintf("%s-%s", r.Name, r.Version)
			nonLinuxOSIDs = append(nonLinuxOSIDs, r.ID)
			nonLinuxOSIDMap[r.ID] = key
			// Initialize result map entry for non-Linux
			if _, exists := result[key]; !exists {
				result[key] = fleet.OSVulnerabilitiesWithCount{
					Vulnerabilities: make([]fleet.CVE, 0),
					Count:           0,
				}
			}
		}
	}

	if len(linuxOSVersionIDs) == 0 && len(nonLinuxOSIDs) == 0 {
		return result, nil
	}

	// Step 2: Execute queries in parallel
	vulnsByKey := make(map[string][]fleet.CVE)
	cveSetByKey := make(map[string]map[string]struct{}) // Track unique CVEs per key for accurate counting
	cveSet := make(map[string]struct{})
	// Track total counts per key when using LIMIT - this is the count BEFORE limiting
	// We need to track by OSID/OSVersionID and then aggregate by key
	totalCountByOSID := make(map[uint]uint)        // For non-Linux: operating_system_id -> total count
	totalCountByOSVersionID := make(map[uint]uint) // For Linux: os_version_id -> total count

	type vulnResult struct {
		OSID              uint // For non-Linux: os.id
		OSVersionID       uint // For Linux: os_version_id
		CVE               string
		ResolvedInVersion *string
		CreatedAt         time.Time
		IsLinux           bool  // Flag to distinguish between Linux and non-Linux results
		TotalCount        *uint // Total count per operating_system_id/os_version_id (only set when using ROW_NUMBER)
	}

	var osVulnResults, kernelVulnResults []vulnResult

	// Launch goroutines for parallel query execution
	errChan := make(chan error, 2)
	osVulnsChan := make(chan []vulnResult, 1)
	kernelVulnsChan := make(chan []vulnResult, 1)

	// Query 1: OS Vulnerabilities (non-Linux only, as Linux uses kernel vulnerabilities)
	// The operating_system_vulnerabilities table does not contain Linux vulnerabilities
	go func() {
		if len(nonLinuxOSIDs) == 0 {
			osVulnsChan <- nil
			errChan <- nil
			return
		}

		// Build query based on maxVulnerabilities parameter
		var osVulnsQuery string
		osArgs := make([]any, len(nonLinuxOSIDs))
		for i, id := range nonLinuxOSIDs {
			osArgs[i] = id
		}

		switch {
		case maxVulnerabilities != nil && *maxVulnerabilities == 0:
			// For max=0, only fetch minimal data needed for counting
			osVulnsQuery = `
			SELECT
				osv.operating_system_id,
				osv.cve
			FROM operating_system_vulnerabilities osv
			WHERE osv.operating_system_id IN (` + strings.TrimSuffix(strings.Repeat("?,", len(nonLinuxOSIDs)), ",") + `)`

		case maxVulnerabilities != nil && *maxVulnerabilities > 0:
			// Use ROW_NUMBER() to limit at database level per operating_system_id
			// Include total count per operating_system_id for accurate counting after deduplication
			osVulnsQuery = `
			SELECT operating_system_id, cve, resolved_in_version, created_at, total_count
			FROM (
				SELECT
					operating_system_id,
					cve,
					resolved_in_version,
					created_at,
					ROW_NUMBER() OVER (PARTITION BY operating_system_id ORDER BY cve) as rn,
					COUNT(*) OVER (PARTITION BY operating_system_id) as total_count
				FROM operating_system_vulnerabilities
				WHERE operating_system_id IN (` + strings.TrimSuffix(strings.Repeat("?,", len(nonLinuxOSIDs)), ",") + `)
			) sub
			WHERE rn <= ?`
			osArgs = append(osArgs, *maxVulnerabilities)

		default:
			// Fetch all CVEs with full details
			osVulnsQuery = `
			SELECT
				osv.operating_system_id,
				osv.cve,
				osv.resolved_in_version,
				osv.created_at
			FROM operating_system_vulnerabilities osv
			WHERE osv.operating_system_id IN (` + strings.TrimSuffix(strings.Repeat("?,", len(nonLinuxOSIDs)), ",") + `)`
		}

		var osVulnDBResults []struct {
			OSID              uint       `db:"operating_system_id"`
			CVE               string     `db:"cve"`
			ResolvedInVersion *string    `db:"resolved_in_version"`
			CreatedAt         *time.Time `db:"created_at"`
			TotalCount        *uint      `db:"total_count"`
		}

		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &osVulnDBResults, osVulnsQuery, osArgs...); err != nil {
			osVulnsChan <- nil
			errChan <- ctxerr.Wrap(ctx, err, "batch query OS vulnerabilities")
			return
		}

		// Convert to generic vulnResult format
		results := make([]vulnResult, len(osVulnDBResults))
		for i, r := range osVulnDBResults {
			createdAt := time.Time{}
			if r.CreatedAt != nil {
				createdAt = *r.CreatedAt
			}
			results[i] = vulnResult{
				OSID:              r.OSID,
				CVE:               r.CVE,
				ResolvedInVersion: r.ResolvedInVersion,
				CreatedAt:         createdAt,
				IsLinux:           false,
				TotalCount:        r.TotalCount,
			}
		}

		osVulnsChan <- results
		errChan <- nil
	}()

	// Query 2: Kernel Vulnerabilities (for Linux only) from pre-aggregated table
	go func() {
		if len(linuxOSVersionIDs) == 0 {
			kernelVulnsChan <- nil
			errChan <- nil
			return
		}

		// Query the pre-aggregated table directly by os_version_id
		kargs := make([]any, 0, len(linuxOSVersionIDs)+1)
		for _, id := range linuxOSVersionIDs {
			kargs = append(kargs, id)
		}

		// Build team filter
		teamFilter := ""
		if teamID != nil {
			teamFilter = ` AND team_id = ?`
			kargs = append(kargs, *teamID)
		} else {
			teamFilter = ` AND team_id IS NULL`
		}

		// Build query based on maxVulnerabilities parameter
		var kernelQuery string
		switch {
		case maxVulnerabilities != nil && *maxVulnerabilities == 0:
			// For max=0, only return counts per os_version_id (no CVEs)
			// Go code handles deduplication across os_version_ids that share the same key (name-version)
			kernelQuery = `
			SELECT
				os_version_id,
				COUNT(*) as total_count
			FROM operating_system_version_vulnerabilities
			WHERE os_version_id IN (` + strings.TrimSuffix(strings.Repeat("?,", len(linuxOSVersionIDs)), ",") + `)` + teamFilter + `
			GROUP BY os_version_id`

		case maxVulnerabilities != nil && *maxVulnerabilities > 0:
			// Use LATERAL JOIN + CTE for optimal performance:
			// 1. Computing counts via GROUP BY (fast)
			// 2. Fetching only N CVEs per os_version_id via LATERAL LIMIT (fast)
			kernelQuery = `
			WITH counts AS (
				SELECT os_version_id, COUNT(*) as total_count
				FROM operating_system_version_vulnerabilities
				WHERE os_version_id IN (` + strings.TrimSuffix(strings.Repeat("?,", len(linuxOSVersionIDs)), ",") + `)` + teamFilter + `
				GROUP BY os_version_id
			)
			SELECT counts.os_version_id, v.cve, v.resolved_in_version, v.created_at, counts.total_count
			FROM counts
			CROSS JOIN LATERAL (
				SELECT cve, resolved_in_version, created_at
				FROM operating_system_version_vulnerabilities
				WHERE os_version_id = counts.os_version_id` + teamFilter + `
				ORDER BY cve
				LIMIT ?
			) v`
			kargs = append(kargs, *maxVulnerabilities)

		default:
			// Fetch all CVEs with full details
			kernelQuery = `
			SELECT
				os_version_id,
				cve,
				resolved_in_version,
				created_at
			FROM operating_system_version_vulnerabilities
			WHERE os_version_id IN (` + strings.TrimSuffix(strings.Repeat("?,", len(linuxOSVersionIDs)), ",") + `)` + teamFilter
		}

		var kernelVulnDBResults []struct {
			OSVersionID       uint       `db:"os_version_id"`
			CVE               string     `db:"cve"`
			ResolvedInVersion *string    `db:"resolved_in_version"`
			CreatedAt         *time.Time `db:"created_at"`
			TotalCount        *uint      `db:"total_count"`
		}

		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &kernelVulnDBResults, kernelQuery, kargs...); err != nil {
			kernelVulnsChan <- nil
			errChan <- ctxerr.Wrap(ctx, err, "batch query kernel vulnerabilities from pre-aggregated table")
			return
		}

		// Convert to generic vulnResult format
		results := make([]vulnResult, len(kernelVulnDBResults))
		for i, r := range kernelVulnDBResults {
			createdAt := time.Time{}
			if r.CreatedAt != nil {
				createdAt = *r.CreatedAt
			}
			results[i] = vulnResult{
				OSVersionID:       r.OSVersionID,
				CVE:               r.CVE,
				ResolvedInVersion: r.ResolvedInVersion,
				CreatedAt:         createdAt,
				IsLinux:           true,
				TotalCount:        r.TotalCount,
			}
		}

		kernelVulnsChan <- results
		errChan <- nil
	}()

	// Wait for both goroutines to complete
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			return nil, err
		}
	}
	osVulnResults = <-osVulnsChan
	kernelVulnResults = <-kernelVulnsChan

	// Process OS vulnerability results (non-Linux)
	for _, r := range osVulnResults {
		key := nonLinuxOSIDMap[r.OSID]

		// Track total count per OSID when LIMIT is used
		if r.TotalCount != nil {
			totalCountByOSID[r.OSID] = *r.TotalCount
		}

		// Track unique CVEs per key for accurate counting (handles overlapping CVEs across OSIDs)
		if cveSetByKey[key] == nil {
			cveSetByKey[key] = make(map[string]struct{})
		}
		cveSetByKey[key][r.CVE] = struct{}{}

		vuln := fleet.CVE{
			CVE:       r.CVE,
			CreatedAt: r.CreatedAt,
		}

		if r.ResolvedInVersion != nil {
			resolvedVersion := r.ResolvedInVersion // avoid address of range var field
			vuln.ResolvedInVersion = &resolvedVersion
		}

		// Check if we already have this CVE for this key (deduplication across architectures)
		found := false
		for i, existing := range vulnsByKey[key] {
			if existing.CVE == r.CVE {
				found = true
				// Keep the earliest CreatedAt time
				if r.CreatedAt.Before(existing.CreatedAt) {
					vulnsByKey[key][i].CreatedAt = r.CreatedAt
				}
				break
			}
		}
		if !found {
			vulnsByKey[key] = append(vulnsByKey[key], vuln)
		}
		cveSet[r.CVE] = struct{}{}
	}

	// Process kernel vulnerability results (Linux)
	for _, r := range kernelVulnResults {
		key := linuxOSVersionMap[r.OSVersionID]

		// Track total count per OSVersionID when LIMIT is used
		if r.TotalCount != nil {
			totalCountByOSVersionID[r.OSVersionID] = *r.TotalCount
		}

		// Track unique CVEs per key for accurate counting (handles overlapping CVEs across OSVersionIDs)
		if cveSetByKey[key] == nil {
			cveSetByKey[key] = make(map[string]struct{})
		}
		cveSetByKey[key][r.CVE] = struct{}{}

		vuln := fleet.CVE{
			CVE:       r.CVE,
			CreatedAt: r.CreatedAt,
		}

		if r.ResolvedInVersion != nil {
			resolvedVersion := r.ResolvedInVersion // avoid address of range var field
			vuln.ResolvedInVersion = &resolvedVersion
		}

		// Check if we already have this CVE (shouldn't happen as we're grouping in the query)
		found := false
		for _, existing := range vulnsByKey[key] {
			if existing.CVE == r.CVE {
				found = true
				break
			}
		}
		if !found {
			vulnsByKey[key] = append(vulnsByKey[key], vuln)
		}

		cveSet[r.CVE] = struct{}{}
	}

	// Step 3: Fetch CVE metadata for all CVEs
	if includeCVSS && len(cveSet) > 0 {
		cveList := make([]string, 0, len(cveSet))
		for cve := range cveSet {
			cveList = append(cveList, cve)
		}

		// Fetch metadata in batches using the common batch processing utility
		batchSize := 2000
		metadataMap := make(map[string]struct {
			CVSSScore        *float64
			EPSSProbability  *float64
			CISAKnownExploit *bool
			CVEPublished     *time.Time
			Description      *string
		})

		err := common_mysql.BatchProcessSimple(cveList, batchSize, func(batch []string) error {
			placeholders := strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")

			metaQuery := `
				SELECT
					cve,
					cvss_score,
					epss_probability,
					cisa_known_exploit,
					published,
					description
				FROM cve_meta
				WHERE cve IN (` + placeholders + `)`

			metaArgs := make([]any, len(batch))
			for j, cve := range batch {
				metaArgs[j] = cve
			}

			var metaResults []struct {
				CVE              string     `db:"cve"`
				CVSSScore        *float64   `db:"cvss_score"`
				EPSSProbability  *float64   `db:"epss_probability"`
				CISAKnownExploit *bool      `db:"cisa_known_exploit"`
				Published        *time.Time `db:"published"`
				Description      *string    `db:"description"`
			}

			if err := sqlx.SelectContext(ctx, ds.reader(ctx), &metaResults, metaQuery, metaArgs...); err != nil {
				return ctxerr.Wrap(ctx, err, "batch query CVE metadata")
			}

			for _, r := range metaResults {
				metadataMap[r.CVE] = struct {
					CVSSScore        *float64
					EPSSProbability  *float64
					CISAKnownExploit *bool
					CVEPublished     *time.Time
					Description      *string
				}{r.CVSSScore, r.EPSSProbability, r.CISAKnownExploit, r.Published, r.Description}
			}
			return nil
		})

		if err != nil {
			// BatchProcessSimple only returns error if batch processing fails
			return nil, ctxerr.Wrap(ctx, err, "batch processing CVE metadata")
		}

		// Apply metadata to vulnerabilities
		for _, vulns := range vulnsByKey {
			for i := range vulns {
				if meta, ok := metadataMap[vulns[i].CVE]; ok {
					if meta.CVSSScore != nil {
						vulns[i].CVSSScore = &meta.CVSSScore
					}
					if meta.EPSSProbability != nil {
						vulns[i].EPSSProbability = &meta.EPSSProbability
					}
					if meta.CISAKnownExploit != nil {
						vulns[i].CISAKnownExploit = &meta.CISAKnownExploit
					}
					if meta.CVEPublished != nil {
						vulns[i].CVEPublished = &meta.CVEPublished
					}
					if meta.Description != nil {
						vulns[i].Description = &meta.Description
					}
				}
			}
		}
	}

	// Step 4: Assign vulnerabilities and counts to result
	// We iterate over the result map keys (not vulnsByKey) to ensure we capture
	// all OS versions, including those with maxVulnerabilities=0 where vulnsByKey is empty
	for key := range result {
		vulns := vulnsByKey[key]

		// Calculate the actual count
		var count int
		if maxVulnerabilities != nil && *maxVulnerabilities > 0 {
			// When LIMIT was used, we need to calculate the total from the per-OSID/OSVersionID counts
			// stored in totalCountByOSID and totalCountByOSVersionID
			// We need to find all OSIDs/OSVersionIDs for this key and get their total counts

			// For non-Linux OSs, find all OSIDs with this key
			for osID, keyForOSID := range nonLinuxOSIDMap {
				if keyForOSID == key {
					if totalCount, exists := totalCountByOSID[osID]; exists {
						// Use the total count from the database (before limiting)
						if int(totalCount) > count {
							count = int(totalCount)
						}
					}
				}
			}

			// For Linux OSs, find all OSVersionIDs with this key
			for osVersionID, keyForOSVersionID := range linuxOSVersionMap {
				if keyForOSVersionID == key {
					if totalCount, exists := totalCountByOSVersionID[osVersionID]; exists {
						// Use the total count from the database (before limiting)
						if int(totalCount) > count {
							count = int(totalCount)
						}
					}
				}
			}
		} else {
			// When LIMIT was not used, count deduplicated CVEs
			if cveSetByKey[key] != nil {
				count = len(cveSetByKey[key])
			}
		}

		// Apply per-key limit after deduplication if maxVulnerabilities is specified.
		// The SQL queries limit per OSID/OSVersionID, but when multiple share the same name+version
		// (e.g., different architectures or kernel versions), we need to enforce the limit after merging.
		if maxVulnerabilities != nil && *maxVulnerabilities == 0 {
			// For max=0, return empty array but include the count
			vulns = make([]fleet.CVE, 0)
		} else if maxVulnerabilities != nil && *maxVulnerabilities > 0 && len(vulns) > *maxVulnerabilities {
			// Sort by CVE alphabetically for deterministic results (matches SQL ORDER BY cve)
			sort.Slice(vulns, func(i, j int) bool {
				return vulns[i].CVE < vulns[j].CVE
			})
			vulns = vulns[:*maxVulnerabilities]
		}

		if vulns == nil {
			vulns = make([]fleet.CVE, 0)
		}
		result[key] = fleet.OSVulnerabilitiesWithCount{
			Vulnerabilities: vulns,
			Count:           count,
		}
	}

	return result, nil
}
