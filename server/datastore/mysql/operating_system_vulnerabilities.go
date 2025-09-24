package mysql

import (
	"context"
	"fmt"
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

func (ds *Datastore) ListVulnsByOsNameAndVersion(ctx context.Context, name, version string, includeCVSS bool, teamID *uint) (fleet.Vulnerabilities, error) {
	r := fleet.Vulnerabilities{}

	stmt := `
			SELECT
				osv.cve,
				MIN(osv.created_at) created_at
			FROM operating_system_vulnerabilities osv
			JOIN operating_systems os ON os.id = osv.operating_system_id
				AND os.name = ? AND os.version = ?
			GROUP BY osv.cve

			UNION

			SELECT DISTINCT
				software_cve.cve,
				MIN(software_cve.created_at) created_at
			FROM
				software_cve
				JOIN kernel_host_counts ON kernel_host_counts.software_id = software_cve.software_id
				JOIN operating_systems ON operating_systems.os_version_id = kernel_host_counts.os_version_id
			WHERE
				operating_systems.name = ?
				AND operating_systems.version = ?
				AND kernel_host_counts.hosts_count > 0
				%s
			GROUP BY software_cve.cve

			`

	if includeCVSS {
		// The group_concat below ensures we only one item is returned per CVE, and the assumption that we have a
		// consistent resolved-in version across architectures/build numbers currently holds, so we shouldn't see
		// distinct resolved-in versions under normal operation. We *could* see a different created_at if we see
		// a new architecture for the same OS version after a vulnerability has been reported for that OS version.
		stmt = `
			SELECT
				osv.cve,
				cm.cvss_score,
				cm.epss_probability,
				cm.cisa_known_exploit,
				cm.published as cve_published,
				cm.description,
				osv.resolved_in_version,
				osv.created_at
			FROM (
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
					software_cve.cve,
					MIN(software_cve.created_at) created_at,
					GROUP_CONCAT(DISTINCT software_cve.resolved_in_version SEPARATOR ',') resolved_in_version
				FROM
					software_cve
					JOIN kernel_host_counts ON kernel_host_counts.software_id = software_cve.software_id
					JOIN operating_systems ON operating_systems.os_version_id = kernel_host_counts.os_version_id
				WHERE
					operating_systems.name = ?
					AND operating_systems.version = ?
					AND kernel_host_counts.hosts_count > 0
					%s
				GROUP BY software_cve.cve
			) osv
			LEFT JOIN cve_meta cm ON cm.cve = osv.cve
			`
	}

	var tmID uint
	var teamFilter string
	args := []any{name, version, name, version}
	if teamID != nil {
		tmID = *teamID
		teamFilter = "AND kernel_host_counts.team_id = ?"
		args = append(args, tmID)
	}

	stmt = fmt.Sprintf(stmt, teamFilter)

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &r, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error executing SQL statement")
	}

	return r, nil
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
) (map[string]fleet.Vulnerabilities, error) {
	result := make(map[string]fleet.Vulnerabilities)
	if len(osVersions) == 0 {
		return result, nil
	}

	// Step 1: Batch fetch OS IDs by name and version
	// The OSVersions from ds.OSVersions() don't include the ID field,
	// only OSVersionID, so we need to look them up
	osIDMap := make(map[uint]string, len(osVersions))
	osIDs := make([]uint, 0, len(osVersions))
	linuxOSIDs := make([]uint, 0)          // Track Linux OS IDs separately for kernel vulnerabilities
	platformMap := make(map[string]string) // Map "name-version" to platform

	// Build query using IN with tuples - more efficient with the index on (name, version)
	tuples := make([]string, 0, len(osVersions))
	args := make([]any, 0, len(osVersions)*2)

	for _, os := range osVersions {
		tuples = append(tuples, "(?, ?)")
		args = append(args, os.NameOnly, os.Version)
		// Store the platform for each OS version
		platformMap[fmt.Sprintf("%s-%s", os.NameOnly, os.Version)] = os.Platform
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
		return nil, ctxerr.Wrap(ctx, err, "batch fetch OS IDs")
	}

	for _, r := range osResults {
		key := fmt.Sprintf("%s-%s", r.Name, r.Version)
		osIDs = append(osIDs, r.ID)
		osIDMap[r.ID] = key

		// Check if this OS is Linux and add to Linux-specific list
		if platform, ok := platformMap[key]; ok && fleet.IsLinux(platform) {
			linuxOSIDs = append(linuxOSIDs, r.ID)
		}
	}

	if len(osIDs) == 0 {
		return result, nil
	}

	// Initialize result map
	for _, key := range osIDMap {
		result[key] = make(fleet.Vulnerabilities, 0)
	}

	// Step 2: Execute queries
	vulnsByKey := make(map[string][]fleet.CVE)
	cveSet := make(map[string]struct{})

	// Query 1: OS Vulnerabilities
	osVulnsQuery := `
		SELECT
			osv.operating_system_id,
			osv.cve,
			osv.resolved_in_version,
			osv.created_at
		FROM operating_system_vulnerabilities osv
		WHERE osv.operating_system_id IN (` + strings.TrimSuffix(strings.Repeat("?,", len(osIDs)), ",") + `)
	`

	osArgs := make([]any, len(osIDs))
	for i, id := range osIDs {
		osArgs[i] = id
	}

	var osVulnResults []struct {
		OSID              uint      `db:"operating_system_id"`
		CVE               string    `db:"cve"`
		ResolvedInVersion *string   `db:"resolved_in_version"`
		CreatedAt         time.Time `db:"created_at"`
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &osVulnResults, osVulnsQuery, osArgs...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "batch query OS vulnerabilities")
	}

	for _, r := range osVulnResults {
		key := osIDMap[r.OSID]
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

	// Query 2: Kernel Vulnerabilities (for Linux only)
	if len(linuxOSIDs) > 0 {
		kernelQuery := `
			SELECT
				os.id as os_id,
				sc.cve,
				sc.resolved_in_version,
				MIN(sc.created_at) as created_at
			FROM software_cve sc
			JOIN kernel_host_counts khc ON khc.software_id = sc.software_id
			JOIN operating_systems os ON os.os_version_id = khc.os_version_id
			WHERE os.id IN (` + strings.TrimSuffix(strings.Repeat("?,", len(linuxOSIDs)), ",") + `)
			AND khc.hosts_count > 0
		`

		kargs := make([]any, len(linuxOSIDs))
		for i, id := range linuxOSIDs {
			kargs[i] = id
		}

		if teamID != nil {
			kernelQuery += ` AND khc.team_id = ?`
			kargs = append(kargs, *teamID)
		}

		kernelQuery += ` GROUP BY os.id, sc.cve, sc.resolved_in_version`

		var kernelVulnResults []struct {
			OSID              uint      `db:"os_id"`
			CVE               string    `db:"cve"`
			ResolvedInVersion *string   `db:"resolved_in_version"`
			CreatedAt         time.Time `db:"created_at"`
		}

		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &kernelVulnResults, kernelQuery, kargs...); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "batch query kernel vulnerabilities")
		}
		for _, r := range kernelVulnResults {
			key := osIDMap[r.OSID]
			vuln := fleet.CVE{
				CVE:       r.CVE,
				CreatedAt: r.CreatedAt,
			}

			if r.ResolvedInVersion != nil {
				resolvedVersion := r.ResolvedInVersion // avoid address of range var field
				vuln.ResolvedInVersion = &resolvedVersion
			}

			// Check if we already have this CVE
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
	}

	// Step 3: Fetch CVE metadata (for Linux kernels only)
	if includeCVSS && len(cveSet) > 0 {
		cveList := make([]string, 0, len(cveSet))
		for cve := range cveSet {
			cveList = append(cveList, cve)
		}

		// Fetch metadata in batches using the common batch processing utility
		batchSize := 500
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

	// Step 4: Assign vulnerabilities to result
	for key, vulns := range vulnsByKey {
		result[key] = vulns
	}

	return result, nil
}
