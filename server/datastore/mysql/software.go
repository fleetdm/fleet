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

func uniqueStringToSoftwafre(s string) fleet.Software {
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

func softwareSliceToIdMap(softwares []fleet.Software) map[string]uint {
	result := make(map[string]uint)
	for _, s := range softwares {
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

		/*
			load all the software in a map[string]int name0version0source->id :: allsoft
			load the host current software like ^
			load the incoming software like ^
			for each incoming software:
				if not in allsoft:
					insert to software
					insert to host software
					continue
				else:
					if not in current software:
						insert to host software
			for each current software:
				if not in incoming:
					delete from host
		*/

		storedSoftware, err := d.allSoftware()
		if err != nil {
			return errors.Wrap(err, "getting all software")
		}

		storedCurrentSoftware, err := d.hostSoftwareFromHostID(host.ID)
		if err != nil {
			return errors.Wrap(err, "loading current software for host")
		}

		allSoftware := softwareSliceToIdMap(storedSoftware)
		current := softwareSliceToSet(storedCurrentSoftware)
		incoming := softwareSliceToSet(host.HostSoftware.Software)

		var insertsSoftware [][]interface{}
		var insertsHostSoftware []interface{}
		var deletesHostSoftware []interface{}

		for incomingKey := range incoming {
			if _, ok := allSoftware[incomingKey]; !ok {
				s := uniqueStringToSoftwafre(incomingKey)
				insertsSoftware = append(insertsSoftware, []interface{}{s.Name, s.Version, s.Source})
				continue
			} else if _, ok := current[incomingKey]; !ok {
				insertsHostSoftware = append(insertsHostSoftware, host.ID, allSoftware[incomingKey])
				continue
			}
		}

		for currentKey := range current {
			if _, ok := incoming[currentKey]; !ok {
				deletesHostSoftware = append(deletesHostSoftware, host.ID, allSoftware[currentKey])
				// TODO: delete from software if no host has it?
				continue
			}
		}

		for _, args := range insertsSoftware {
			result, err := tx.Exec(
				`INSERT IGNORE INTO software (name, version, source) VALUES (?, ?, ?)`,
				args...,
			)
			if err != nil {
				return errors.Wrap(err, "insert software")
			}
			id, err := result.LastInsertId()
			if err != nil {
				return errors.Wrap(err, "last id from software")
			}
			insertsHostSoftware = append(insertsHostSoftware, host.ID, uint(id))
		}

		values := strings.TrimSuffix(strings.Repeat("(?,?),", len(insertsHostSoftware)/2), ",")
		sql := fmt.Sprintf(`INSERT INTO host_software (host_id, software_id) VALUES %s`, values)
		if _, err := tx.Exec(sql, insertsHostSoftware...); err != nil {
			return errors.Wrap(err, "insert host software")
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "save host software")
	}

	host.HostSoftware.Modified = false
	return nil
}

func compareSoftware(software []fleet.Software) func(i int, j int) bool {
	return func(i, j int) bool {
		prev := software[i]
		next := software[j]
		prevFull := fmt.Sprintf("%s\u0000%s\u0000%s", prev.Name, prev.Version, prev.Source)
		nextFull := fmt.Sprintf("%s\u0000%s\u0000%s", next.Name, next.Version, next.Source)
		return strings.Compare(prevFull, nextFull) == -1
	}
}

func (d *Datastore) hostSoftwareFromHostID(id uint) ([]fleet.Software, error) {
	sql := `
		SELECT * FROM software
		WHERE id IN
			(SELECT software_id FROM host_software WHERE host_id = ?)
	`
	var result []fleet.Software
	if err := d.db.Select(&result, sql, id); err != nil {
		return nil, errors.Wrap(err, "load host software")
	}
	return result, nil
}

func (d *Datastore) allSoftware() ([]fleet.Software, error) {
	sql := `SELECT * FROM software`
	var result []fleet.Software
	if err := d.db.Select(&result, sql); err != nil {
		return nil, errors.Wrap(err, "load host software")
	}
	return result, nil
}

func (d *Datastore) LoadHostSoftware(host *fleet.Host) error {
	host.HostSoftware = fleet.HostSoftware{Modified: false}
	software, err := d.hostSoftwareFromHostID(host.ID)
	if err != nil {
		return err
	}
	host.Software = software
	return nil
}
