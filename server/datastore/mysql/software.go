package mysql

import (
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

func (d *Datastore) SaveHostSoftware(host *fleet.Host) error {
	if !host.HostSoftware.Modified {
		return nil
	}

	if err := d.withRetryTxx(func(tx *sqlx.Tx) error {
		if len(host.HostSoftware.Software) == 0 {
			// Clear join table for this host
			sql := "DELETE FROM host_software WHERE host_id = ?"
			if _, err := tx.Exec(sql, host.ID); err != nil {
				return errors.Wrap(err, "clear join table entries")
			}

			return nil
		}

		if err := d.applyChangesForNewSoftware(tx, host); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "save host software")
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

func (d *Datastore) applyChangesForNewSoftware(tx *sqlx.Tx, host *fleet.Host) error {
	storedCurrentSoftware, err := d.hostSoftwareFromHostID(tx, host.ID)
	if err != nil {
		return errors.Wrap(err, "loading current software for host")
	}

	if nothingChanged(storedCurrentSoftware, host.Software) {
		return nil
	}

	current := softwareSliceToIdMap(storedCurrentSoftware)
	incoming := softwareSliceToSet(host.Software)

	if err = d.deleteUninstalledHostSoftware(tx, host.ID, current, incoming); err != nil {
		return err
	}

	if err = d.insertNewInstalledHostSoftware(tx, host.ID, current, incoming); err != nil {
		return err
	}

	return nil
}

func (d *Datastore) deleteUninstalledHostSoftware(
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

func (d *Datastore) getOrGenerateSoftwareId(tx *sqlx.Tx, s fleet.Software) (uint, error) {
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

func (d *Datastore) insertNewInstalledHostSoftware(
	tx *sqlx.Tx,
	hostID uint,
	currentIdmap map[string]uint,
	incomingBitmap map[string]bool,
) error {
	var insertsHostSoftware []interface{}
	for s := range incomingBitmap {
		if _, ok := currentIdmap[s]; !ok {
			id, err := d.getOrGenerateSoftwareId(tx, uniqueStringToSoftware(s))
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

func (d *Datastore) hostSoftwareFromHostID(tx *sqlx.Tx, id uint) ([]fleet.Software, error) {
	selectFunc := d.db.Select
	if tx != nil {
		selectFunc = tx.Select
	}
	sql := `
		SELECT s.id, s.name, s.version, s.source, coalesce(scp.cpe, "") as generated_cpe, 
			IF(
				JSON_ARRAYAGG(scv.cve) = JSON_ARRAYAGG(null), 
				null, 
				JSON_ARRAYAGG(
					JSON_OBJECT(
						"cve", scv.cve, 
						"details_link", CONCAT('https://nvd.nist.gov/vuln/detail/', scv.cve)
					)
				)
			) as vulnerabilities FROM software s
		LEFT JOIN software_cpe scp ON (s.id=scp.software_id)
		LEFT JOIN software_cve scv ON (scp.id=scv.cpe_id)
		WHERE s.id IN
			(SELECT software_id FROM host_software WHERE host_id = ?)
		group by s.id, s.name, s.version, s.source, generated_cpe
	`
	var result []fleet.Software
	if err := selectFunc(&result, sql, id); err != nil {
		return nil, errors.Wrap(err, "load host software")
	}

	return result, nil
}

func (d *Datastore) LoadHostSoftware(host *fleet.Host) error {
	host.HostSoftware = fleet.HostSoftware{Modified: false}
	software, err := d.hostSoftwareFromHostID(nil, host.ID)
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

func (d *Datastore) AllSoftwareWithoutCPEIterator() (fleet.SoftwareIterator, error) {
	sql := `SELECT s.* FROM software s LEFT JOIN software_cpe sc on (s.id=sc.software_id) WHERE sc.id is null`
	rows, err := d.db.Queryx(sql)
	if err != nil {
		return nil, errors.Wrap(err, "load host software")
	}
	return &softwareIterator{rows: rows}, nil
}

func (d *Datastore) AddCPEForSoftware(software fleet.Software, cpe string) error {
	sql := `INSERT INTO software_cpe (software_id, cpe) VALUES (?, ?)`
	if _, err := d.db.Exec(sql, software.ID, cpe); err != nil {
		return errors.Wrap(err, "insert software cpe")
	}
	return nil
}

func (d *Datastore) AllCPEs() ([]string, error) {
	sql := `SELECT cpe FROM software_cpe`
	var cpes []string
	err := d.db.Select(&cpes, sql)
	if err != nil {
		return nil, errors.Wrap(err, "loads cpes")
	}
	return cpes, nil
}

func (d *Datastore) InsertCVEForCPE(cve string, cpes []string) error {
	values := strings.TrimSuffix(strings.Repeat("((SELECT id FROM software_cpe WHERE cpe=?),?),", len(cpes)), ",")
	sql := fmt.Sprintf(`INSERT IGNORE INTO software_cve (cpe_id, cve) VALUES %s`, values)
	var args []interface{}
	for _, cpe := range cpes {
		args = append(args, cpe, cve)
	}
	_, err := d.db.Exec(sql, args...)
	if err != nil {
		return errors.Wrap(err, "insert software cve")
	}
	return nil
}
