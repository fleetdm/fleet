package mysql

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
		query := "SELECT id, name, description, platform, disabled FROM packs WHERE pack_type IS NULL OR pack_type = ''"
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
	sqlStatement := `
		SELECT *
			FROM packs
			WHERE name = ?
	`
	var pack fleet.Pack
	err := d.db.Get(&pack, sqlStatement, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, errors.Wrap(err, "fetch pack by name")
	}

	if err := d.loadPackTargets(&pack); err != nil {
		return nil, false, err
	}

	return &pack, true, nil
}

// NewPack creates a new Pack
func (d *Datastore) NewPack(pack *fleet.Pack, opts ...fleet.OptionalArg) (*fleet.Pack, error) {
	if err := d.withRetryTxx(func(tx *sqlx.Tx) error {
		query := `
			INSERT INTO packs
			(name, description, platform, disabled)
			VALUES ( ?, ?, ?, ? )
		`
		result, err := tx.Exec(query, pack.Name, pack.Description, pack.Platform, pack.Disabled)
		if err != nil {
			return errors.Wrap(err, "insert pack")
		}

		id, _ := result.LastInsertId()
		pack.ID = uint(id)

		if err := d.replacePackTargets(tx, pack); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return pack, nil
}

func (d *Datastore) replacePackTargets(tx *sqlx.Tx, pack *fleet.Pack) error {
	sql := `DELETE FROM pack_targets WHERE pack_id = ?`
	if _, err := tx.Exec(sql, pack.ID); err != nil {
		return errors.Wrap(err, "delete pack targets")
	}

	// Insert hosts
	if len(pack.HostIDs) > 0 {
		var args []interface{}
		for _, id := range pack.HostIDs {
			args = append(args, pack.ID, fleet.TargetHost, id)
		}
		values := strings.TrimSuffix(
			strings.Repeat("(?,?,?),", len(pack.HostIDs)),
			",",
		)
		sql = fmt.Sprintf(`
			INSERT INTO pack_targets (pack_id, type, target_id)
			VALUES %s
		`, values)
		if _, err := tx.Exec(sql, args...); err != nil {
			return errors.Wrap(err, "insert host targets")
		}
	}

	// Insert labels
	if len(pack.LabelIDs) > 0 {
		var args []interface{}
		for _, id := range pack.LabelIDs {
			args = append(args, pack.ID, fleet.TargetLabel, id)
		}
		values := strings.TrimSuffix(
			strings.Repeat("(?,?,?),", len(pack.LabelIDs)),
			",",
		)
		sql = fmt.Sprintf(`
			INSERT INTO pack_targets (pack_id, type, target_id)
			VALUES %s
		`, values)
		if _, err := tx.Exec(sql, args...); err != nil {
			return errors.Wrap(err, "insert label targets")
		}
	}

	// Insert teams
	if len(pack.TeamIDs) > 0 {
		var args []interface{}
		for _, id := range pack.TeamIDs {
			args = append(args, pack.ID, fleet.TargetTeam, id)
		}
		values := strings.TrimSuffix(
			strings.Repeat("(?,?,?),", len(pack.TeamIDs)),
			",",
		)
		sql = fmt.Sprintf(`
			INSERT INTO pack_targets (pack_id, type, target_id)
			VALUES %s
		`, values)
		if _, err := tx.Exec(sql, args...); err != nil {
			return errors.Wrap(err, "insert team targets")
		}
	}

	return nil
}

func (d *Datastore) loadPackTargets(pack *fleet.Pack) error {
	var targets []fleet.PackTarget
	sql := `SELECT * FROM pack_targets WHERE pack_id = ?`
	if err := d.db.Select(&targets, sql, pack.ID); err != nil {
		return errors.Wrap(err, "select pack targets")
	}

	pack.HostIDs, pack.LabelIDs, pack.TeamIDs = []uint{}, []uint{}, []uint{}
	for _, target := range targets {
		switch target.Type {
		case fleet.TargetHost:
			pack.HostIDs = append(pack.HostIDs, target.TargetID)
		case fleet.TargetLabel:
			pack.LabelIDs = append(pack.LabelIDs, target.TargetID)
		case fleet.TargetTeam:
			pack.TeamIDs = append(pack.TeamIDs, target.TargetID)
		default:
			return errors.Errorf("unknown target type: %d", target.Type)
		}
	}

	return nil
}

// SavePack stores changes to pack
func (d *Datastore) SavePack(pack *fleet.Pack) error {
	return d.withRetryTxx(func(tx *sqlx.Tx) error {
		query := `
			UPDATE packs
			SET name = ?, platform = ?, disabled = ?, description = ?
			WHERE id = ?
	`

		results, err := tx.Exec(query, pack.Name, pack.Platform, pack.Disabled, pack.Description, pack.ID)
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

		return d.replacePackTargets(tx, pack)
	})
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
		return nil, errors.Wrap(err, "get pack")
	}

	if err := d.loadPackTargets(pack); err != nil {
		return nil, err
	}

	return pack, nil
}

// EnsureGlobalPack gets or inserts a pack with type global
func (d *Datastore) EnsureGlobalPack() (*fleet.Pack, error) {
	pack := &fleet.Pack{}
	err := d.db.Get(pack, `SELECT * FROM packs WHERE pack_type = 'global'`)
	if err == sql.ErrNoRows {
		return d.insertNewGlobalPack()
	} else if err != nil {
		return nil, errors.Wrap(err, "get pack")
	}

	if err := d.loadPackTargets(pack); err != nil {
		return nil, err
	}

	return pack, nil
}

func (d *Datastore) insertNewGlobalPack() (*fleet.Pack, error) {
	var packID uint
	err := d.withTx(func(tx *sqlx.Tx) error {
		res, err := tx.Exec(
			`INSERT INTO packs (name, description, platform, pack_type) VALUES ('Global', 'Global pack', '','global')`,
		)
		if err != nil {
			return err
		}
		packId, err := res.LastInsertId()
		if err != nil {
			return err
		}
		packID = uint(packId)
		if _, err := tx.Exec(
			`INSERT INTO pack_targets (pack_id, type, target_id) VALUES (?, ?, (SELECT id FROM labels WHERE name = ?))`,
			packID, fleet.TargetLabel, "All Hosts",
		); err != nil {
			return errors.Wrap(err, "adding label to pack")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return d.Pack(packID)
}

func (d *Datastore) EnsureTeamPack(teamID uint) (*fleet.Pack, error) {
	pack := &fleet.Pack{}
	t, err := d.Team(teamID)
	if err != nil || t == nil {
		return nil, errors.Wrap(err, "Error finding team")
	}

	teamType := fmt.Sprintf("team-%d", teamID)
	err = d.db.Get(pack, `SELECT * FROM packs WHERE pack_type = ?`, teamType)
	if err == sql.ErrNoRows {
		return d.insertNewTeamPack(teamID)
	} else if err != nil {
		return nil, errors.Wrap(err, "get pack")
	}

	if err := d.loadPackTargets(pack); err != nil {
		return nil, err
	}

	return pack, nil
}

func (d *Datastore) insertNewTeamPack(teamID uint) (*fleet.Pack, error) {
	var packID uint
	teamType := fmt.Sprintf("team-%d", teamID)
	err := d.withTx(func(tx *sqlx.Tx) error {
		res, err := tx.Exec(
			`INSERT INTO packs (name, description, platform, pack_type) 
                   VALUES (?, 'Schedule additional queries for all hosts assigned to this team.', '',?)`,
			teamType, teamType,
		)
		if err != nil {
			return err
		}
		packId, err := res.LastInsertId()
		if err != nil {
			return err
		}
		packID = uint(packId)
		if _, err := tx.Exec(
			`INSERT INTO pack_targets (pack_id, type, target_id) VALUES (?, ?, ?)`,
			packID, fleet.TargetTeam, teamID,
		); err != nil {
			return errors.Wrap(err, "adding team id target to pack")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return d.Pack(packID)
}

// ListPacks returns all fleet.Pack records limited and sorted by fleet.ListOptions
func (d *Datastore) ListPacks(opt fleet.PackListOptions) ([]*fleet.Pack, error) {
	query := `SELECT * FROM packs WHERE pack_type IS NULL OR pack_type = ''`
	if opt.IncludeSystemPacks {
		query = `SELECT * FROM packs`
	}
	var packs []*fleet.Pack
	err := d.db.Select(&packs, appendListOptionsToSQL(query, opt.ListOptions))
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "listing packs")
	}

	for _, pack := range packs {
		if err := d.loadPackTargets(pack); err != nil {
			return nil, err
		}
	}

	return packs, nil
}

func (d *Datastore) ListPacksForHost(hid uint) ([]*fleet.Pack, error) {
	query := `
		SELECT DISTINCT packs.*
		FROM
		((SELECT p.*
		FROM packs p
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
		UNION ALL
		(SELECT p.*
		FROM packs p
		JOIN pack_targets pt
		ON (p.id = pt.pack_id AND pt.type = ? AND pt.target_id = (SELECT team_id FROM hosts WHERE id = ?)))
		) packs
	`

	packs := []*fleet.Pack{}
	if err := d.db.Select(&packs, query, fleet.TargetLabel, hid, fleet.TargetHost, hid, fleet.TargetTeam, hid); err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "listing hosts in pack")
	}
	return packs, nil
}
