package mysql

import (
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/server/kolide"
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

func (d *Datastore) SaveHostSoftware(host *kolide.Host) error {
	if !host.HostSoftware.Modified {
		return nil
	}

	if err := d.withRetryTxx(func(tx *sqlx.Tx) error {
		// Clear join table for this host
		sql := "DELETE FROM host_software WHERE host_id = ?"
		if _, err := tx.Exec(sql, host.ID); err != nil {
			return errors.Wrap(err, "clear join table entries")
		}

		if len(host.HostSoftware.Software) == 0 {
			return nil
		}

		// Bulk insert software entries
		var args []interface{}
		for _, s := range host.HostSoftware.Software {
			s.Name = truncateString(s.Name, maxSoftwareNameLen)
			s.Version = truncateString(s.Version, maxSoftwareVersionLen)
			s.Source = truncateString(s.Source, maxSoftwareSourceLen)
			args = append(args, s.Name, s.Version, s.Source)
		}
		values := strings.TrimSuffix(strings.Repeat("(?,?,?),", len(host.HostSoftware.Software)), ",")
		sql = fmt.Sprintf(`
			INSERT INTO software (name, version, source)
			VALUES %s
			ON DUPLICATE KEY UPDATE name = name
		`, values)
		if _, err := tx.Exec(sql, args...); err != nil {
			return errors.Wrap(err, "insert software")
		}

		// Bulk update join table
		sql = fmt.Sprintf(`
			INSERT INTO host_software (host_id, software_id)
			SELECT ?, s.id as software_id
			FROM software s
			WHERE (name, version, source) IN (%s)
		`, values)
		if _, err := tx.Exec(sql, append([]interface{}{host.ID}, args...)...); err != nil {
			return errors.Wrap(err, "insert join table entries")
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "save host software")
	}

	host.HostSoftware.Modified = false
	return nil
}

func (d *Datastore) LoadHostSoftware(host *kolide.Host) error {
	host.HostSoftware = kolide.HostSoftware{Modified: false}
	sql := `
		SELECT * FROM software
		WHERE id IN
			(SELECT software_id FROM host_software WHERE host_id = ?)
	`
	if err := d.db.Select(&host.HostSoftware.Software, sql, host.ID); err != nil {
		return errors.Wrap(err, "load host software")
	}

	return nil
}
