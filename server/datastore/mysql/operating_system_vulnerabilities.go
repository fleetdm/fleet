package mysql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListOSVulnerabilities(ctx context.Context, hostIDs []uint) ([]fleet.OSVulnerability, error) {
	r := []fleet.OSVulnerability{}

	stmt := dialect.
		From(goqu.T("operating_system_vulnerabilities")).
		Select(
			goqu.I("host_id"),
			goqu.I("operating_system_id"),
			goqu.I("cve"),
			goqu.I("resolved_in_version"),
		).
		Where(goqu.C("host_id").In(hostIDs))

	sql, args, err := stmt.ToSQL()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error generating SQL statement")
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &r, sql, args...); err != nil {
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
	sql := fmt.Sprintf(`INSERT IGNORE INTO operating_system_vulnerabilities (host_id, operating_system_id, cve, source) VALUES %s`, values)

	for _, v := range vulnerabilities {
		args = append(args, v.HostID, v.OSID, v.CVE, source)
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
			host_id,
			operating_system_id,
			cve,
			source,
			resolved_in_version
		) VALUES (?,?,?,?,?)
		ON DUPLICATE KEY UPDATE
			operating_system_id = VALUES(operating_system_id),
			source = VALUES(source),
			resolved_in_version = VALUES(resolved_in_version),
			updated_at = ?
	`

	args = append(args, v.HostID, v.OSID, v.CVE, s, v.ResolvedInVersion, time.Now().UTC())

	res, err := ds.writer(ctx).ExecContext(ctx, sqlStmt, args...)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "insert operating system vulnerability")
	}

	return insertOnDuplicateDidInsert(res), nil
}

func (ds *Datastore) DeleteOSVulnerabilities(ctx context.Context, vulnerabilities []fleet.OSVulnerability) error {
	if len(vulnerabilities) == 0 {
		return nil
	}

	sql := fmt.Sprintf(
		`DELETE FROM operating_system_vulnerabilities WHERE (host_id, cve) IN (%s)`,
		strings.TrimSuffix(strings.Repeat("(?,?),", len(vulnerabilities)), ","),
	)

	var args []interface{}
	for _, v := range vulnerabilities {
		args = append(args, v.HostID, v.CVE)
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
