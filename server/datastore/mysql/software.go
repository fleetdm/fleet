package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

const (
	maxSoftwareNameLen             = 255
	maxSoftwareVersionLen          = 255
	maxSoftwareSourceLen           = 64
	maxSoftwareBundleIdentifierLen = 255
)

func truncateString(str string, length int) string {
	if len(str) > length {
		return str[:length]
	}
	return str
}

func softwareToUniqueString(s fleet.Software) string {
	return strings.Join([]string{s.Name, s.Version, s.Source, s.BundleIdentifier}, "\u0000")
}

func uniqueStringToSoftware(s string) fleet.Software {
	parts := strings.Split(s, "\u0000")
	return fleet.Software{
		Name:             truncateString(parts[0], maxSoftwareNameLen),
		Version:          truncateString(parts[1], maxSoftwareVersionLen),
		Source:           truncateString(parts[2], maxSoftwareSourceLen),
		BundleIdentifier: truncateString(parts[3], maxSoftwareBundleIdentifierLen),
	}
}

func softwareSliceToSet(softwares []fleet.Software) map[string]bool {
	result := make(map[string]bool)
	for _, s := range softwares {
		result[softwareToUniqueString(s)] = true
	}
	return result
}

func softwareSliceToIdMap(softwareSlice []fleet.Software) map[string]uint {
	result := make(map[string]uint)
	for _, s := range softwareSlice {
		result[softwareToUniqueString(s)] = s.ID
	}
	return result
}

func (d *Datastore) SaveHostSoftware(ctx context.Context, host *fleet.Host) error {
	if !host.HostSoftware.Modified {
		return nil
	}

	return d.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return saveHostSoftwareDB(ctx, tx, host)
	})
}

func saveHostSoftwareDB(ctx context.Context, tx sqlx.ExtContext, host *fleet.Host) error {
	if err := applyChangesForNewSoftwareDB(ctx, tx, host); err != nil {
		return err
	}

	host.HostSoftware.Modified = false
	return nil
}

func nothingChanged(current []fleet.Software, incoming []fleet.Software) bool {
	if len(current) != len(incoming) {
		return false
	}

	currentBitmap := make(map[string]bool)
	for _, s := range current {
		currentBitmap[softwareToUniqueString(s)] = true
	}
	for _, s := range incoming {
		if _, ok := currentBitmap[softwareToUniqueString(s)]; !ok {
			return false
		}
	}

	return true
}

func applyChangesForNewSoftwareDB(ctx context.Context, tx sqlx.ExtContext, host *fleet.Host) error {
	storedCurrentSoftware, err := listSoftwareDB(ctx, tx, &host.ID, fleet.SoftwareListOptions{SkipLoadingCVEs: true})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "loading current software for host")
	}

	if nothingChanged(storedCurrentSoftware, host.Software) {
		return nil
	}

	current := softwareSliceToIdMap(storedCurrentSoftware)
	incoming := softwareSliceToSet(host.Software)

	if err = deleteUninstalledHostSoftwareDB(ctx, tx, host.ID, current, incoming); err != nil {
		return err
	}

	if err = insertNewInstalledHostSoftwareDB(ctx, tx, host.ID, current, incoming); err != nil {
		return err
	}

	return nil
}

func deleteUninstalledHostSoftwareDB(
	ctx context.Context,
	tx sqlx.ExecerContext,
	hostID uint,
	currentIdmap map[string]uint,
	incomingBitmap map[string]bool,
) error {
	var deletesHostSoftware []interface{}
	deletesHostSoftware = append(deletesHostSoftware, hostID)

	for currentKey := range currentIdmap {
		if _, ok := incomingBitmap[currentKey]; !ok {
			deletesHostSoftware = append(deletesHostSoftware, currentIdmap[currentKey])
			// TODO: delete from software if no host has it
		}
	}
	if len(deletesHostSoftware) <= 1 {
		return nil
	}
	sql := fmt.Sprintf(
		`DELETE FROM host_software WHERE host_id = ? AND software_id IN (%s)`,
		strings.TrimSuffix(strings.Repeat("?,", len(deletesHostSoftware)-1), ","),
	)
	if _, err := tx.ExecContext(ctx, sql, deletesHostSoftware...); err != nil {
		return ctxerr.Wrap(ctx, err, "delete host software")
	}

	return nil
}

func getOrGenerateSoftwareIdDB(ctx context.Context, tx sqlx.ExtContext, s fleet.Software) (uint, error) {
	var existingId []int64
	if err := sqlx.SelectContext(ctx, tx,
		&existingId,
		`SELECT id FROM software WHERE name = ? AND version = ? AND source = ? AND bundle_identifier = ?`,
		s.Name, s.Version, s.Source, s.BundleIdentifier,
	); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "get software")
	}
	if len(existingId) > 0 {
		return uint(existingId[0]), nil
	}

	result, err := tx.ExecContext(ctx,
		`REPLACE INTO software (name, version, source, bundle_identifier) VALUES (?, ?, ?, ?)`,
		s.Name, s.Version, s.Source, s.BundleIdentifier,
	)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insert software")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "last id from software")
	}
	return uint(id), nil
}

func insertNewInstalledHostSoftwareDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostID uint,
	currentIdmap map[string]uint,
	incomingBitmap map[string]bool,
) error {
	var insertsHostSoftware []interface{}
	for s := range incomingBitmap {
		if _, ok := currentIdmap[s]; !ok {
			id, err := getOrGenerateSoftwareIdDB(ctx, tx, uniqueStringToSoftware(s))
			if err != nil {
				return err
			}
			insertsHostSoftware = append(insertsHostSoftware, hostID, id)
		}
	}
	if len(insertsHostSoftware) > 0 {
		values := strings.TrimSuffix(strings.Repeat("(?,?),", len(insertsHostSoftware)/2), ",")
		sql := fmt.Sprintf(`INSERT IGNORE INTO host_software (host_id, software_id) VALUES %s`, values)
		if _, err := tx.ExecContext(ctx, sql, insertsHostSoftware...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert host software")
		}
	}

	return nil
}

func listSoftwareDB(ctx context.Context, q sqlx.QueryerContext, hostID *uint, opt fleet.SoftwareListOptions) ([]fleet.Software, error) {
	hostWhere := `hs.host_id=?`
	if hostID == nil {
		hostWhere = "TRUE"
	}
	teamWhere := `h.team_id=?`
	if opt.TeamID == nil {
		teamWhere = "TRUE"
	}
	vulnerableJoin := "LEFT JOIN software_cpe scp ON (s.id=scp.software_id)"
	if opt.VulnerableOnly {
		vulnerableJoin = `JOIN software_cpe scp ON (s.id=scp.software_id)
		JOIN software_cve scv ON (scp.id=scv.cpe_id)`
	}
	sql := fmt.Sprintf(`
		SELECT DISTINCT s.*, coalesce(scp.cpe, "") as generated_cpe
		FROM host_software hs
		JOIN hosts h ON (hs.host_id=h.id)
		JOIN software s ON (hs.software_id=s.id)
		%s
		WHERE %s AND %s
	`, vulnerableJoin, hostWhere, teamWhere)

	var result []fleet.Software
	vars := []interface{}{}
	if hostID != nil {
		vars = append(vars, hostID)
	}
	if opt.TeamID != nil {
		vars = append(vars, opt.TeamID)
	}
	sql, listVars := searchLike(sql, vars, opt.MatchQuery, "s.name", "s.version")
	sql += ` GROUP BY s.id, s.name, s.version, s.source, generated_cpe `
	sql = appendListOptionsToSQL(sql, opt.ListOptions)
	if err := sqlx.SelectContext(ctx, q, &result, sql, listVars...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load host software")
	}

	if opt.SkipLoadingCVEs {
		return result, nil
	}

	sql = fmt.Sprintf(`
		SELECT DISTINCT hs.software_id, scv.cve
		FROM host_software hs
		JOIN hosts h ON (hs.host_id=h.id)
		JOIN software_cpe scp ON (hs.software_id=scp.software_id)
		JOIN software_cve scv ON (scp.id=scv.cpe_id)
		WHERE %s AND %s
	`, hostWhere, teamWhere)

	rows, err := q.QueryxContext(ctx, sql, vars...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load host software")
	}
	defer rows.Close()

	cvesBySoftware := make(map[uint]fleet.VulnerabilitiesSlice)
	for rows.Next() {
		var id uint
		var cve string
		if err := rows.Scan(&id, &cve); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "scanning cve")
		}
		cvesBySoftware[id] = append(cvesBySoftware[id], fleet.SoftwareCVE{
			CVE:         cve,
			DetailsLink: fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cve),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error iterating through cve rows")
	}

	var resultWithCVEs []fleet.Software
	for _, software := range result {
		software.Vulnerabilities = cvesBySoftware[software.ID]
		resultWithCVEs = append(resultWithCVEs, software)
	}

	return resultWithCVEs, nil
}

func (d *Datastore) LoadHostSoftware(ctx context.Context, host *fleet.Host) error {
	host.HostSoftware = fleet.HostSoftware{Modified: false}
	software, err := listSoftwareDB(ctx, d.reader, &host.ID, fleet.SoftwareListOptions{})
	if err != nil {
		return err
	}
	host.Software = software
	return nil
}

type softwareIterator struct {
	rows *sqlx.Rows
}

func (si *softwareIterator) Value() (*fleet.Software, error) {
	dest := fleet.Software{}
	err := si.rows.StructScan(&dest)
	if err != nil {
		return nil, err
	}
	return &dest, nil
}

func (si *softwareIterator) Err() error {
	return si.rows.Err()
}

func (si *softwareIterator) Close() error {
	return si.rows.Close()
}

func (si *softwareIterator) Next() bool {
	return si.rows.Next()
}

func (d *Datastore) AllSoftwareWithoutCPEIterator(ctx context.Context) (fleet.SoftwareIterator, error) {
	sql := `SELECT s.* FROM software s LEFT JOIN software_cpe sc on (s.id=sc.software_id) WHERE sc.id is null`
	// The rows.Close call is done by the caller once iteration using the
	// returned fleet.SoftwareIterator is done.
	rows, err := d.reader.QueryxContext(ctx, sql) //nolint:sqlclosecheck
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load host software")
	}
	return &softwareIterator{rows: rows}, nil
}

func (d *Datastore) AddCPEForSoftware(ctx context.Context, software fleet.Software, cpe string) error {
	_, err := addCPEForSoftwareDB(ctx, d.writer, software, cpe)
	return err
}

func addCPEForSoftwareDB(ctx context.Context, exec sqlx.ExecerContext, software fleet.Software, cpe string) (uint, error) {
	sql := `INSERT INTO software_cpe (software_id, cpe) VALUES (?, ?)`
	res, err := exec.ExecContext(ctx, sql, software.ID, cpe)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insert software cpe")
	}
	id, _ := res.LastInsertId() // cannot fail with the mysql driver
	return uint(id), nil
}

func (d *Datastore) AllCPEs(ctx context.Context) ([]string, error) {
	sql := `SELECT cpe FROM software_cpe`
	var cpes []string
	err := sqlx.SelectContext(ctx, d.reader, &cpes, sql)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loads cpes")
	}
	return cpes, nil
}

func (d *Datastore) InsertCVEForCPE(ctx context.Context, cve string, cpes []string) error {
	values := strings.TrimSuffix(strings.Repeat("((SELECT id FROM software_cpe WHERE cpe=?),?),", len(cpes)), ",")
	sql := fmt.Sprintf(`INSERT IGNORE INTO software_cve (cpe_id, cve) VALUES %s`, values)
	var args []interface{}
	for _, cpe := range cpes {
		args = append(args, cpe, cve)
	}
	_, err := d.writer.ExecContext(ctx, sql, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "insert software cve")
	}
	return nil
}

func (d *Datastore) ListSoftware(ctx context.Context, opt fleet.SoftwareListOptions) ([]fleet.Software, error) {
	return listSoftwareDB(ctx, d.reader, nil, opt)
}

func (d *Datastore) SoftwareByID(ctx context.Context, id uint) (*fleet.Software, error) {
	software := fleet.Software{}
	err := sqlx.GetContext(ctx, d.reader, &software, `SELECT * FROM software WHERE id=?`, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "software by id")
	}

	query := `
		SELECT DISTINCT scv.cve
		FROM software s
		JOIN software_cpe scp ON (s.id=scp.software_id)
		JOIN software_cve scv ON (scp.id=scv.cpe_id)
		WHERE s.id=?
	`

	rows, err := d.reader.QueryxContext(ctx, query, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load software cves")
	}
	defer rows.Close()

	for rows.Next() {
		var cve string
		if err := rows.Scan(&cve); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "scanning cve")
		}

		software.Vulnerabilities = append(software.Vulnerabilities, fleet.SoftwareCVE{
			CVE:         cve,
			DetailsLink: fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cve),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error iterating through cve rows")
	}

	return &software, nil
}
