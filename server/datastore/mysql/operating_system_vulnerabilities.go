package mysql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
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
	kernel_host_counts.os_version_id = ? %s GROUP BY id, cve, version
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
