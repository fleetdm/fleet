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

func (ds *Datastore) ListVulnsByOsNameAndVersion(ctx context.Context, name, version string, includeCVSS bool) (fleet.Vulnerabilities, error) {
	r := fleet.Vulnerabilities{}

	var sqlstmt string

	if includeCVSS {
		sqlstmt = `
			SELECT DISTINCT
				osv.cve,
				cm.cvss_score,
				cm.epss_probability,
				cm.cisa_known_exploit,
				cm.published as cve_published,
				cm.description,
				osv.resolved_in_version,
				osv.created_at
			FROM operating_system_vulnerabilities osv
			LEFT JOIN cve_meta cm ON cm.cve = osv.cve
			WHERE osv.operating_system_id IN (
				SELECT id FROM operating_systems WHERE name = ? AND version = ?
			)
			`
	} else {
		sqlstmt = `
			SELECT DISTINCT
				osv.cve,
				osv.created_at
			FROM operating_system_vulnerabilities osv
			WHERE osv.operating_system_id IN (
				SELECT id FROM operating_systems WHERE name = ? AND version = ?
			)
			`
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &r, sqlstmt, name, version); err != nil {
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
			updated_at = ?
	`

	args = append(args, v.OSID, v.CVE, s, v.ResolvedInVersion, time.Now().UTC())

	res, err := ds.writer(ctx).ExecContext(ctx, sqlStmt, args...)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "insert operating system vulnerability")
	}

	return insertOnDuplicateDidInsertOrUpdate(res), nil
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

func (ds *Datastore) DeleteOutOfDateOSVulnerabilities(ctx context.Context, src fleet.VulnerabilitySource, d time.Duration) error {
	deleteStmt := `
		DELETE FROM operating_system_vulnerabilities
		WHERE source = ? AND updated_at < ?
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, deleteStmt, src, time.Now().UTC().Add(-d)); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting out of date operating system vulnerabilities")
	}
	return nil
}
