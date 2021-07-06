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

type softwareChanges struct {
	insertsSoftware     [][]interface{}
	insertsHostSoftware []interface{}
	deletesHostSoftware []interface{}
}

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

		changes, err := d.generateChangesForNewSoftware(host)
		if err != nil {
			return err
		}

		err = d.applyChangesForNewSoftware(tx, host, changes)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "save host software")
	}

	host.HostSoftware.Modified = false
	return nil
}

func (d *Datastore) applyChangesForNewSoftware(
	tx *sqlx.Tx,
	host *fleet.Host,
	changes softwareChanges,
) error {
	for _, args := range changes.insertsSoftware {
		// we insert one by one here because we need the IDs for host_software later on
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
		changes.insertsHostSoftware = append(changes.insertsHostSoftware, host.ID, uint(id))
	}

	if len(changes.insertsHostSoftware) > 0 {
		values := strings.TrimSuffix(strings.Repeat("(?,?),", len(changes.insertsHostSoftware)/2), ",")
		sql := fmt.Sprintf(`INSERT INTO host_software (host_id, software_id) VALUES %s`, values)
		if _, err := tx.Exec(sql, changes.insertsHostSoftware...); err != nil {
			return errors.Wrap(err, "insert host software")
		}
	}

	if len(changes.deletesHostSoftware) > 0 {
		sql := fmt.Sprintf(
			`DELETE FROM host_software WHERE host_id = ? AND software_id IN (%s)`,
			strings.TrimSuffix(strings.Repeat("?,", len(changes.deletesHostSoftware)/2), ","),
		)
		if _, err := tx.Exec(sql, changes.deletesHostSoftware...); err != nil {
			return errors.Wrap(err, "insert host software")
		}
	}

	return nil
}

func (d *Datastore) generateChangesForNewSoftware(host *fleet.Host) (
	softwareChanges, error,
) {
	storedSoftware, err := d.allSoftware()
	if err != nil {
		return softwareChanges{}, errors.Wrap(err, "getting all software")
	}

	storedCurrentSoftware, err := d.hostSoftwareFromHostID(host.ID)
	if err != nil {
		return softwareChanges{}, errors.Wrap(err, "loading current software for host")
	}

	return softwareDiff(host.ID, host.HostSoftware.Software, storedSoftware, storedCurrentSoftware)
}

func softwareDiff(
	hostID uint,
	incomingHostSoftware []fleet.Software,
	storedSoftware []fleet.Software,
	storedHostSoftware []fleet.Software,
) (softwareChanges, error) {
	allSoftware := softwareSliceToIdMap(storedSoftware)
	current := softwareSliceToSet(storedHostSoftware)
	incoming := softwareSliceToSet(incomingHostSoftware)

	var insertsSoftware [][]interface{}
	var insertsHostSoftware []interface{}
	var deletesHostSoftware []interface{}

	// First we cover new installations
	for incomingKey := range incoming {
		// If we haven't seen this app at all, then we add it to the overall software list
		// and later on with add to host_software as well (but we need to do this insert
		// first to get the id)
		if _, ok := allSoftware[incomingKey]; !ok {
			s := uniqueStringToSoftware(incomingKey)
			insertsSoftware = append(insertsSoftware, []interface{}{s.Name, s.Version, s.Source})
			continue
		} else if _, ok := current[incomingKey]; !ok {
			// otherwise, if this is an app that wasn't installed before (but some other host
			//has it, i.e. it's in software), then add it to host_software
			insertsHostSoftware = append(insertsHostSoftware, hostID, allSoftware[incomingKey])
			continue
		}
	}

	// Second we look for apps that were removed. So we check given the current software we know of
	// what of that is not anymore in the new list
	// TODO: instead of looping through current, we could skip the ones that we saw in the loop above
	for currentKey := range current {
		if _, ok := incoming[currentKey]; !ok {
			deletesHostSoftware = append(deletesHostSoftware, allSoftware[currentKey])
			// TODO: delete from software if no host has it
			continue
		}
	}
	return softwareChanges{
		insertsSoftware:     insertsSoftware,
		insertsHostSoftware: insertsHostSoftware,
		deletesHostSoftware: deletesHostSoftware,
	}, nil
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
	var result []fleet.Software
	if err := d.db.Select(&result, `SELECT * FROM software`); err != nil {
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
