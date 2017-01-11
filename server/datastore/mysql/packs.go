package mysql

import (
	"database/sql"
	"fmt"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/pkg/errors"
)

// NewPack creates a new Pack
func (d *Datastore) NewPack(pack *kolide.Pack) (*kolide.Pack, error) {

	query := `
		INSERT INTO packs ( name, description, platform, created_by, disabled )
			VALUES ( ?, ?, ?, ?, ?)
	`

	result, err := d.db.Exec(query, pack.Name, pack.Description, pack.Platform, pack.CreatedBy, pack.Disabled)
	if err != nil {
		return nil, errors.Wrap(err, "creating new pack")
	}

	id, _ := result.LastInsertId()
	pack.ID = uint(id)
	return pack, nil
}

// SavePack stores changes to pack
func (d *Datastore) SavePack(pack *kolide.Pack) error {
	query := `
			UPDATE packs
			SET name = ?, platform = ?, disabled = ?, description = ?
			WHERE id = ? AND NOT deleted
	`

	_, err := d.db.Exec(query, pack.Name, pack.Platform, pack.Disabled, pack.Description, pack.ID)
	if err == sql.ErrNoRows {
		return notFound("Pack").WithID(pack.ID)
	} else if err != nil {
		return errors.Wrap(err, "update pack")
	}

	return nil
}

// DeletePack soft deletes a kolide.Pack so that it won't show up in results
func (d *Datastore) DeletePack(pid uint) error {
	err := d.deleteEntity("packs", pid)
	if err == sql.ErrNoRows {
		return notFound("Pack").WithID(pid)
	} else if err != nil {
		return errors.Wrap(err, "delete pack")
	}
	return nil
}

// Pack fetch kolide.Pack with matching ID
func (d *Datastore) Pack(pid uint) (*kolide.Pack, error) {
	query := `SELECT * FROM packs WHERE id = ? AND NOT deleted`
	pack := &kolide.Pack{}
	err := d.db.Get(pack, query, pid)
	if err == sql.ErrNoRows {
		return nil, notFound("Pack").WithID(pid)
	} else if err != nil {
		return nil, errors.Wrap(err, "getting pack")
	}

	return pack, nil
}

// ListPacks returns all kolide.Pack records limited and sorted by kolide.ListOptions
func (d *Datastore) ListPacks(opt kolide.ListOptions) ([]*kolide.Pack, error) {
	query := `SELECT * FROM packs WHERE NOT deleted`
	packs := []*kolide.Pack{}
	err := d.db.Select(&packs, appendListOptionsToSQL(query, opt))
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "listing packs")
	}
	return packs, nil
}

// AddLabelToPack associates a kolide.Label with a kolide.Pack
func (d *Datastore) AddLabelToPack(lid uint, pid uint) error {
	query := `
		INSERT INTO pack_targets ( pack_id,	type, target_id )
			VALUES ( ?, ?, ? )
			ON DUPLICATE KEY UPDATE id=id
	`
	_, err := d.db.Exec(query, pid, kolide.TargetLabel, lid)
	if err != nil {
		return errors.Wrap(err, "adding label to pack")
	}

	return nil
}

// AddHostToPack associates a kolide.Host with a kolide.Pack
func (d *Datastore) AddHostToPack(hid, pid uint) error {
	query := `
		INSERT INTO pack_targets ( pack_id, type, target_id )
			VALUES ( ?, ?, ? )
			ON DUPLICATE KEY UPDATE id=id
	`
	_, err := d.db.Exec(query, pid, kolide.TargetHost, hid)
	if err != nil {
		return errors.Wrap(err, "adding host to pack")
	}

	return nil
}

// ListLabelsForPack will return a list of kolide.Label records associated with kolide.Pack
func (d *Datastore) ListLabelsForPack(pid uint) ([]*kolide.Label, error) {
	query := `
	SELECT
		l.id,
		l.created_at,
		l.updated_at,
		l.name
	FROM
		labels l
	JOIN
		pack_targets pt
	ON
		pt.target_id = l.id
	WHERE
		pt.type = ?
			AND
		pt.pack_id = ?
	AND NOT l.deleted
	`

	labels := []*kolide.Label{}

	if err := d.db.Select(&labels, query, kolide.TargetLabel, pid); err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "listing labels for pack")
	}

	return labels, nil
}

// RemoreLabelFromPack will remove the association between a kolide.Label and
// a kolide.Pack
func (d *Datastore) RemoveLabelFromPack(lid, pid uint) error {
	query := `
		DELETE FROM pack_targets
			WHERE target_id = ? AND pack_id = ? AND type = ?
	`
	_, err := d.db.Exec(query, lid, pid, kolide.TargetLabel)
	if err == sql.ErrNoRows {
		return notFound("PackTarget").WithMessage(fmt.Sprintf("label ID: %d, pack ID: %d", lid, pid))
	} else if err != nil {
		return errors.Wrap(err, "removing label from pack")
	}

	return nil
}

// RemoveHostFromPack will remove the association between a kolide.Host and a
// kolide.Pack
func (d *Datastore) RemoveHostFromPack(hid, pid uint) error {
	query := `
		DELETE FROM pack_targets
			WHERE target_id = ? AND pack_id = ? AND type = ?
	`
	_, err := d.db.Exec(query, hid, pid, kolide.TargetHost)
	if err == sql.ErrNoRows {
		return notFound("PackTarget").WithMessage(fmt.Sprintf("host ID: %d, pack ID: %d", hid, pid))
	} else if err != nil {
		return errors.Wrap(err, "removing host from pack")
	}

	return nil

}

func (d *Datastore) ListHostsInPack(pid uint, opt kolide.ListOptions) ([]*kolide.Host, error) {
	query := `
		SELECT DISTINCT h.*
		FROM hosts h
		JOIN pack_targets pt
		JOIN label_query_executions lqe
		ON (
		  pt.target_id = lqe.label_id
		  AND lqe.host_id = h.id
		  AND lqe.matches
		  AND pt.type = ?
		) OR (
		  pt.target_id = h.id
		  AND pt.type = ?
		)
		WHERE pt.pack_id = ?
	`
	hosts := []*kolide.Host{}
	if err := d.db.Select(&hosts, appendListOptionsToSQL(query, opt), kolide.TargetLabel, kolide.TargetHost, pid); err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "listing hosts in pack")
	}
	return hosts, nil
}

func (d *Datastore) ListExplicitHostsInPack(pid uint, opt kolide.ListOptions) ([]*kolide.Host, error) {
	query := `
		SELECT DISTINCT h.*
		FROM hosts h
		JOIN pack_targets pt
		ON (
		  pt.target_id = h.id
		  AND pt.type = ?
		)
		WHERE pt.pack_id = ?
	`
	hosts := []*kolide.Host{}
	if err := d.db.Select(&hosts, appendListOptionsToSQL(query, opt), kolide.TargetHost, pid); err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "listing explicit hosts in pack")
	}
	return hosts, nil

}
