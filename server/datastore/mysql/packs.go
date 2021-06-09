package mysql

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func (d *Datastore) ApplyPackSpecs(specs []*fleet.PackSpec) (err error) {
	err = d.withRetryTxx(func(tx *sqlx.Tx) error {
		for _, spec := range specs {
			if err := applyPackSpec(tx, spec); err != nil {
				return errors.Wrapf(err, "applying pack '%s'", spec.Name)
			}
		}

		return nil
	})

	return err
}

func applyPackSpec(tx *sqlx.Tx, spec *fleet.PackSpec) error {
	if spec.Name == "" {
		return errors.New("pack name must not be empty")
	}
	// Insert/update pack
	query := `
		INSERT INTO packs (name, description, platform, disabled)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			description = VALUES(description),
			platform = VALUES(platform),
			disabled = VALUES(disabled)
	`
	if _, err := tx.Exec(query, spec.Name, spec.Description, spec.Platform, spec.Disabled); err != nil {
		return errors.Wrap(err, "insert/update pack")
	}

	// Get Pack ID
	// This is necessary because MySQL last_insert_id does not return a value
	// if no update was made.
	var packID uint
	query = "SELECT id FROM packs WHERE name = ?"
	if err := tx.Get(&packID, query, spec.Name); err != nil {
		return errors.Wrap(err, "getting pack ID")
	}

	// Delete existing scheduled queries for pack
	query = "DELETE FROM scheduled_queries WHERE pack_id = ?"
	if _, err := tx.Exec(query, packID); err != nil {
		return errors.Wrap(err, "delete existing scheduled queries")
	}

	// Insert new scheduled queries for pack
	for _, q := range spec.Queries {
		// Default to query name if scheduled query name is not specified.
		if q.Name == "" {
			q.Name = q.QueryName
		}
		query = `
			INSERT INTO scheduled_queries (
				pack_id, query_name, name, description, ` + "`interval`" + `,
				snapshot, removed, shard, platform, version, denylist
			)
			VALUES (
				?, ?, ?, ?, ?,
				?, ?, ?, ?, ?, ?
			)
		`
		_, err := tx.Exec(query,
			packID, q.QueryName, q.Name, q.Description, q.Interval,
			q.Snapshot, q.Removed, q.Shard, q.Platform, q.Version, q.Denylist,
		)
		switch {
		case isChildForeignKeyError(err):
			return errors.Errorf("cannot schedule unknown query '%s'", q.QueryName)
		case err != nil:
			return errors.Wrapf(err, "adding query %s referencing %s", q.Name, q.QueryName)
		}
	}

	// Delete existing targets
	query = "DELETE FROM pack_targets WHERE pack_id = ?"
	if _, err := tx.Exec(query, packID); err != nil {
		return errors.Wrap(err, "delete existing targets")
	}

	// Insert targets
	for _, l := range spec.Targets.Labels {
		query = `
			INSERT INTO pack_targets (pack_id, type, target_id)
			VALUES (?, ?, (SELECT id FROM labels WHERE name = ?))
		`
		if _, err := tx.Exec(query, packID, fleet.TargetLabel, l); err != nil {
			return errors.Wrap(err, "adding label to pack")
		}
	}

	return nil
}

func (d *Datastore) GetPackSpecs() (specs []*fleet.PackSpec, err error) {
	err = d.withRetryTxx(func(tx *sqlx.Tx) error {
		// Get basic specs
		query := "SELECT id, name, description, platform, disabled FROM packs"
		if err := tx.Select(&specs, query); err != nil {
			return errors.Wrap(err, "get packs")
		}

		// Load targets
		for _, spec := range specs {
			query = `
SELECT l.name
FROM labels l JOIN pack_targets pt
WHERE pack_id = ? AND pt.type = ? AND pt.target_id = l.id
`
			if err := tx.Select(&spec.Targets.Labels, query, spec.ID, fleet.TargetLabel); err != nil {
				return errors.Wrap(err, "get pack targets")
			}
		}

		// Load queries
		for _, spec := range specs {
			query = `
SELECT
query_name, name, description, ` + "`interval`" + `,
snapshot, removed, shard, platform, version, denylist
FROM scheduled_queries
WHERE pack_id = ?
`
			if err := tx.Select(&spec.Queries, query, spec.ID); err != nil {
				return errors.Wrap(err, "get pack queries")
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return specs, nil
}

func (d *Datastore) GetPackSpec(name string) (spec *fleet.PackSpec, err error) {
	err = d.withRetryTxx(func(tx *sqlx.Tx) error {
		// Get basic spec
		var specs []*fleet.PackSpec
		query := "SELECT id, name, description, platform, disabled FROM packs WHERE name = ?"
		if err := tx.Select(&specs, query, name); err != nil {
			return errors.Wrap(err, "get packs")
		}
		if len(specs) == 0 {
			return notFound("Pack").WithName(name)
		}
		if len(specs) > 1 {
			return errors.Errorf("expected 1 pack row, got %d", len(specs))
		}

		spec = specs[0]

		// Load targets
		query = `
SELECT l.name
FROM labels l JOIN pack_targets pt
WHERE pack_id = ? AND pt.type = ? AND pt.target_id = l.id
`
		if err := tx.Select(&spec.Targets.Labels, query, spec.ID, fleet.TargetLabel); err != nil {
			return errors.Wrap(err, "get pack targets")
		}

		// Load queries
		query = `
SELECT
query_name, name, description, ` + "`interval`" + `,
snapshot, removed, shard, platform, version, denylist
FROM scheduled_queries
WHERE pack_id = ?
`
		if err := tx.Select(&spec.Queries, query, spec.ID); err != nil {
			return errors.Wrap(err, "get pack queries")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return spec, nil
}

func (d *Datastore) PackByName(name string, opts ...fleet.OptionalArg) (*fleet.Pack, bool, error) {
	db := d.getTransaction(opts)
	sqlStatement := `
		SELECT *
			FROM packs
			WHERE name = ?
	`
	var pack fleet.Pack
	err := db.Get(&pack, sqlStatement, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, errors.Wrap(err, "fetching packs by name")
	}

	return &pack, true, nil
}

// NewPack creates a new Pack
func (d *Datastore) NewPack(pack *fleet.Pack, opts ...fleet.OptionalArg) (*fleet.Pack, error) {
	db := d.getTransaction(opts)

	query := `
	INSERT INTO packs
		(name, description, platform, disabled)
		VALUES ( ?, ?, ?, ? )
	`

	result, err := db.Exec(query, pack.Name, pack.Description, pack.Platform, pack.Disabled)
	if err != nil {
		return nil, errors.Wrap(err, "inserting pack")
	}

	id, _ := result.LastInsertId()
	pack.ID = uint(id)
	return pack, nil
}

// SavePack stores changes to pack
func (d *Datastore) SavePack(pack *fleet.Pack) error {
	query := `
			UPDATE packs
			SET name = ?, platform = ?, disabled = ?, description = ?
			WHERE id = ?
	`

	results, err := d.db.Exec(query, pack.Name, pack.Platform, pack.Disabled, pack.Description, pack.ID)
	if err != nil {
		return errors.Wrap(err, "updating pack")
	}
	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "rows affected updating packs")
	}
	if rowsAffected == 0 {
		return notFound("Pack").WithID(pack.ID)
	}
	return nil
}

// DeletePack deletes a fleet.Pack so that it won't show up in results.
func (d *Datastore) DeletePack(name string) error {
	return d.deleteEntityByName("packs", name)
}

// Pack fetch fleet.Pack with matching ID
func (d *Datastore) Pack(pid uint) (*fleet.Pack, error) {
	query := `SELECT * FROM packs WHERE id = ?`
	pack := &fleet.Pack{}
	err := d.db.Get(pack, query, pid)
	if err == sql.ErrNoRows {
		return nil, notFound("Pack").WithID(pid)
	} else if err != nil {
		return nil, errors.Wrap(err, "getting pack")
	}

	return pack, nil
}

// ListPacks returns all fleet.Pack records limited and sorted by fleet.ListOptions
func (d *Datastore) ListPacks(opt fleet.ListOptions) ([]*fleet.Pack, error) {
	query := `SELECT * FROM packs`
	packs := []*fleet.Pack{}
	err := d.db.Select(&packs, appendListOptionsToSQL(query, opt))
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "listing packs")
	}
	return packs, nil
}

// AddLabelToPack associates a fleet.Label with a fleet.Pack
func (d *Datastore) AddLabelToPack(lid uint, pid uint, opts ...fleet.OptionalArg) error {
	db := d.getTransaction(opts)

	query := `
		INSERT INTO pack_targets ( pack_id, type, target_id )
			VALUES ( ?, ?, ? )
			ON DUPLICATE KEY UPDATE id=id
	`
	_, err := db.Exec(query, pid, fleet.TargetLabel, lid)
	if err != nil {
		return errors.Wrap(err, "adding label to pack")
	}

	return nil
}

// AddHostToPack associates a fleet.Host with a fleet.Pack
func (d *Datastore) AddHostToPack(hid, pid uint) error {
	query := `
		INSERT INTO pack_targets ( pack_id, type, target_id )
			VALUES ( ?, ?, ? )
			ON DUPLICATE KEY UPDATE id=id
	`
	_, err := d.db.Exec(query, pid, fleet.TargetHost, hid)
	if err != nil {
		return errors.Wrap(err, "adding host to pack")
	}

	return nil
}

// RemoreLabelFromPack will remove the association between a fleet.Label and
// a fleet.Pack
func (d *Datastore) RemoveLabelFromPack(lid, pid uint) error {
	query := `
		DELETE FROM pack_targets
			WHERE target_id = ? AND pack_id = ? AND type = ?
	`
	_, err := d.db.Exec(query, lid, pid, fleet.TargetLabel)
	if err == sql.ErrNoRows {
		return notFound("PackTarget").WithMessage(fmt.Sprintf("label ID: %d, pack ID: %d", lid, pid))
	} else if err != nil {
		return errors.Wrap(err, "removing label from pack")
	}
	return nil
}

// RemoveHostFromPack will remove the association between a fleet.Host and a
// fleet.Pack
func (d *Datastore) RemoveHostFromPack(hid, pid uint) error {
	query := `
		DELETE FROM pack_targets
			WHERE target_id = ? AND pack_id = ? AND type = ?
	`
	_, err := d.db.Exec(query, hid, pid, fleet.TargetHost)
	if err == sql.ErrNoRows {
		return notFound("PackTarget").WithMessage(fmt.Sprintf("host ID: %d, pack ID: %d", hid, pid))
	} else if err != nil {
		return errors.Wrap(err, "removing host from pack")
	}
	return nil
}

// ListLabelsForPack will return a list of fleet.Label records associated with fleet.Pack
func (d *Datastore) ListLabelsForPack(pid uint) ([]*fleet.Label, error) {
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
	`

	labels := []*fleet.Label{}

	if err := d.db.Select(&labels, query, fleet.TargetLabel, pid); err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "listing labels for pack")
	}

	return labels, nil
}

func (d *Datastore) ListPacksForHost(hid uint) ([]*fleet.Pack, error) {
	query := `
		SELECT DISTINCT packs.*
		FROM
		((SELECT p.* FROM packs p
		JOIN pack_targets pt
		JOIN label_membership lm
		ON (
		  p.id = pt.pack_id
		  AND pt.target_id = lm.label_id
		  AND pt.type = ?
		)
		WHERE lm.host_id = ? AND NOT p.disabled)
		UNION ALL
		(SELECT p.*
		FROM packs p
		JOIN pack_targets pt
		ON (p.id = pt.pack_id AND pt.type = ? AND pt.target_id = ?))
		) packs
	`

	packs := []*fleet.Pack{}
	if err := d.db.Select(&packs, query, fleet.TargetLabel, hid, fleet.TargetHost, hid); err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "listing hosts in pack")
	}
	return packs, nil
}

func (d *Datastore) ListHostsInPack(pid uint, opt fleet.ListOptions) ([]uint, error) {
	query := `
		SELECT DISTINCT h.id
		FROM hosts h
		JOIN pack_targets pt
		JOIN label_membership lm
		ON (
		  pt.target_id = lm.label_id
		  AND lm.host_id = h.id
		  AND pt.type = ?
		) OR (
		  pt.target_id = h.id
		  AND pt.type = ?
		)
		WHERE pt.pack_id = ?
	`

	hosts := []uint{}
	if err := d.db.Select(&hosts, appendListOptionsToSQL(query, opt), fleet.TargetLabel, fleet.TargetHost, pid); err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "listing hosts in pack")
	}
	return hosts, nil
}

func (d *Datastore) ListExplicitHostsInPack(pid uint, opt fleet.ListOptions) ([]uint, error) {
	query := `
		SELECT DISTINCT h.id
		FROM hosts h
		JOIN pack_targets pt
		ON (
		  pt.target_id = h.id
		  AND pt.type = ?
		)
		WHERE pt.pack_id = ?
	`
	hosts := []uint{}
	if err := d.db.Select(&hosts, appendListOptionsToSQL(query, opt), fleet.TargetHost, pid); err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "listing explicit hosts in pack")
	}
	return hosts, nil

}
