package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const (
	maxSoftwareNameLen    = 255
	maxSoftwareVersionLen = 255
	maxSoftwareSourceLen  = 64
)

func truncateString(str string, length int) string {
	if len(str) > length {
		return str[:length]
	}
	return str
}

func softwareToUniqueString(s fleet.Software) string {
	return strings.Join([]string{s.Name, s.Version, s.Source}, "\u0000")
}

func uniqueStringToSoftware(s string) fleet.Software {
	parts := strings.Split(s, "\u0000")
	return fleet.Software{
		Name:    truncateString(parts[0], maxSoftwareNameLen),
		Version: truncateString(parts[1], maxSoftwareVersionLen),
		Source:  truncateString(parts[2], maxSoftwareSourceLen),
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

	return d.withRetryTxx(func(tx *sqlx.Tx) error {
		return saveHostSoftwareDB(tx, host)
	})
}

func saveHostSoftwareDB(tx *sqlx.Tx, host *fleet.Host) error {
	if !host.HostSoftware.Modified {
		return nil
	}

	if len(host.HostSoftware.Software) == 0 {
		// Clear join table for this host
		sql := "DELETE FROM host_software WHERE host_id = ?"
		if _, err := tx.Exec(sql, host.ID); err != nil {
			return errors.Wrap(err, "clear join table entries")
		}

		return nil
	}

	if err := applyChangesForNewSoftwareDB(tx, host); err != nil {
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

func applyChangesForNewSoftwareDB(tx *sqlx.Tx, host *fleet.Host) error {
	storedCurrentSoftware, err := hostSoftwareFromHostIDDB(tx, host.ID)
	if err != nil {
		return errors.Wrap(err, "loading current software for host")
	}

	if nothingChanged(storedCurrentSoftware, host.Software) {
		return nil
	}

	current := softwareSliceToIdMap(storedCurrentSoftware)
	incoming := softwareSliceToSet(host.Software)

	if err = deleteUninstalledHostSoftwareDB(tx, host.ID, current, incoming); err != nil {
		return err
	}

	if err = insertNewInstalledHostSoftwareDB(tx, host.ID, current, incoming); err != nil {
		return err
	}

	return nil
}

func deleteUninstalledHostSoftwareDB(
	tx *sqlx.Tx,
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
	if _, err := tx.Exec(sql, deletesHostSoftware...); err != nil {
		return errors.Wrap(err, "delete host software")
	}

	return nil
}

func getOrGenerateSoftwareIdDB(tx *sqlx.Tx, s fleet.Software) (uint, error) {
	var existingId []int64
	if err := tx.Select(
		&existingId,
		`SELECT id FROM software WHERE name = ? and version = ? and source = ?`,
		s.Name, s.Version, s.Source,
	); err != nil {
		return 0, err
	}
	if len(existingId) > 0 {
		return uint(existingId[0]), nil
	}

	result, err := tx.Exec(
		`INSERT IGNORE INTO software (name, version, source) VALUES (?, ?, ?)`,
		s.Name, s.Version, s.Source,
	)
	if err != nil {
		return 0, errors.Wrap(err, "insert software")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, errors.Wrap(err, "last id from software")
	}
	return uint(id), nil
}

func insertNewInstalledHostSoftwareDB(
	tx *sqlx.Tx,
	hostID uint,
	currentIdmap map[string]uint,
	incomingBitmap map[string]bool,
) error {
	var insertsHostSoftware []interface{}
	for s := range incomingBitmap {
		if _, ok := currentIdmap[s]; !ok {
			id, err := getOrGenerateSoftwareIdDB(tx, uniqueStringToSoftware(s))
			if err != nil {
				return err
			}
			insertsHostSoftware = append(insertsHostSoftware, hostID, id)
		}
	}
	if len(insertsHostSoftware) > 0 {
		values := strings.TrimSuffix(strings.Repeat("(?,?),", len(insertsHostSoftware)/2), ",")
		sql := fmt.Sprintf(`INSERT IGNORE INTO host_software (host_id, software_id) VALUES %s`, values)
		if _, err := tx.Exec(sql, insertsHostSoftware...); err != nil {
			return errors.Wrap(err, "insert host software")
		}
	}

	return nil
}

func hostSoftwareFromHostIDDB(q sqlx.Queryer, id uint) ([]fleet.Software, error) {
	sql := `
		SELECT s.id, s.name, s.version, s.source, coalesce(scp.cpe, "") as generated_cpe
		FROM software s
		LEFT JOIN software_cpe scp ON (s.id=scp.software_id)
		WHERE s.id IN
			(SELECT software_id FROM host_software WHERE host_id = ?)
		group by s.id, s.name, s.version, s.source, generated_cpe
	`
	var result []*fleet.Software
	if err := sqlx.Select(q, &result, sql, id); err != nil {
		return nil, errors.Wrap(err, "load host software")
	}

	sql = `
		SELECT s.id, scv.cve
		FROM software s
		JOIN software_cpe scp ON (s.id=scp.software_id)
		JOIN software_cve scv ON (scp.id=scv.cpe_id)
		WHERE s.id IN
			(SELECT software_id FROM host_software WHERE host_id = ?)
	`

	rows, err := q.Queryx(sql, id)
	if err != nil {
		return nil, errors.Wrap(err, "load host software")
	}
	defer rows.Close()

	cvesBySoftware := make(map[uint]fleet.VulnerabilitiesSlice)
	for rows.Next() {
		var id uint
		var cve string
		if err := rows.Scan(&id, &cve); err != nil {
			return nil, errors.Wrap(err, "scanning cve")
		}
		cvesBySoftware[id] = append(cvesBySoftware[id], fleet.SoftwareCVE{
			CVE:         cve,
			DetailsLink: fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cve),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating through cve rows")
	}

	var resultWithCVEs []fleet.Software
	for _, software := range result {
		software.Vulnerabilities = cvesBySoftware[software.ID]
		resultWithCVEs = append(resultWithCVEs, *software)
	}

	return resultWithCVEs, nil
}

func (d *Datastore) LoadHostSoftware(ctx context.Context, host *fleet.Host) error {
	host.HostSoftware = fleet.HostSoftware{Modified: false}
	software, err := hostSoftwareFromHostIDDB(d.reader, host.ID)
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
	rows, err := d.reader.Queryx(sql) //nolint:sqlclosecheck
	if err != nil {
		return nil, errors.Wrap(err, "load host software")
	}
	return &softwareIterator{rows: rows}, nil
}

func (d *Datastore) AddCPEForSoftware(ctx context.Context, software fleet.Software, cpe string) error {
	sql := `INSERT INTO software_cpe (software_id, cpe) VALUES (?, ?)`
	if _, err := d.writer.Exec(sql, software.ID, cpe); err != nil {
		return errors.Wrap(err, "insert software cpe")
	}
	return nil
}

func (d *Datastore) AllCPEs(ctx context.Context) ([]string, error) {
	sql := `SELECT cpe FROM software_cpe`
	var cpes []string
	err := d.reader.Select(&cpes, sql)
	if err != nil {
		return nil, errors.Wrap(err, "loads cpes")
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
	_, err := d.writer.Exec(sql, args...)
	if err != nil {
		return errors.Wrap(err, "insert software cve")
	}
	return nil
}
