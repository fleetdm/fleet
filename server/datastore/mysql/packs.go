package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ApplyPackSpecs(ctx context.Context, specs []*fleet.PackSpec) (err error) {
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		for _, spec := range specs {
			if err := applyPackSpecDB(ctx, tx, spec); err != nil {
				return ctxerr.Wrapf(ctx, err, "applying pack '%s'", spec.Name)
			}
		}

		return nil
	})

	return err
}

func applyPackSpecDB(ctx context.Context, tx sqlx.ExtContext, spec *fleet.PackSpec) error {
	if spec.Name == "" {
		return ctxerr.New(ctx, "pack name must not be empty")
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
	if _, err := tx.ExecContext(ctx, query, spec.Name, spec.Description, spec.Platform, spec.Disabled); err != nil {
		return ctxerr.Wrap(ctx, err, "insert/update pack")
	}

	// Get Pack ID
	// This is necessary because MySQL last_insert_id does not return a value
	// if no update was made.
	var packID uint
	query = "SELECT id FROM packs WHERE name = ?"
	if err := sqlx.GetContext(ctx, tx, &packID, query, spec.Name); err != nil {
		return ctxerr.Wrap(ctx, err, "getting pack ID")
	}

	// Delete existing scheduled queries for pack
	query = "DELETE FROM scheduled_queries WHERE pack_id = ?"
	if _, err := tx.ExecContext(ctx, query, packID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete existing scheduled queries")
	}

	// Insert new scheduled queries for pack
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
	for _, q := range spec.Queries {
		// Default to query name if scheduled query name is not specified.
		if q.Name == "" {
			q.Name = q.QueryName
		}
		_, err := tx.ExecContext(ctx, query,
			packID,
			q.QueryName,
			q.Name,
			q.Description,
			q.Interval,
			q.Snapshot,
			q.Removed,
			q.Shard,
			q.Platform,
			q.Version,
			q.Denylist,
		)
		switch {
		case isChildForeignKeyError(err):
			return ctxerr.Errorf(ctx, "cannot schedule unknown query '%s'", q.QueryName)
		case err != nil:
			return ctxerr.Wrapf(ctx, err, "adding query %s referencing %s", q.Name, q.QueryName)
		}
	}

	// Delete existing targets
	query = "DELETE FROM pack_targets WHERE pack_id = ?"
	if _, err := tx.ExecContext(ctx, query, packID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete existing targets")
	}

	query = `
		INSERT INTO pack_targets (pack_id, type, target_id)
		VALUES (?, ?, (SELECT id FROM labels WHERE name = ?))
	`
	for _, l := range spec.Targets.Labels {
		if _, err := tx.ExecContext(ctx, query, packID, fleet.TargetLabel, l); err != nil {
			return ctxerr.Wrap(ctx, err, "adding label to pack")
		}
	}

	query = `
		INSERT INTO pack_targets (pack_id, type, target_id)
		VALUES (?, ?, (SELECT id FROM teams WHERE name = ?))
	`
	for _, t := range spec.Targets.Teams {
		if _, err := tx.ExecContext(ctx, query, packID, fleet.TargetTeam, t); err != nil {
			return ctxerr.Wrap(ctx, err, "adding team to pack")
		}
	}

	return nil
}

func (ds *Datastore) GetPackSpecs(ctx context.Context) ([]*fleet.PackSpec, error) {
	var specs []*fleet.PackSpec
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Get basic specs
		query := "SELECT id, name, description, platform, disabled FROM packs WHERE pack_type IS NULL OR pack_type = ''"
		if err := sqlx.SelectContext(ctx, tx, &specs, query); err != nil {
			return ctxerr.Wrap(ctx, err, "get packs")
		}

		// Load targets
		for _, spec := range specs {
			spec := spec

			// Load labels
			query = `
SELECT l.name
FROM labels l JOIN pack_targets pt
WHERE pack_id = ? AND pt.type = ? AND pt.target_id = l.id
`
			if err := sqlx.SelectContext(ctx, tx, &spec.Targets.Labels, query, spec.ID, fleet.TargetLabel); err != nil {
				return ctxerr.Wrap(ctx, err, "get pack label targets")
			}

			// Load teams
			query = `
SELECT t.name
FROM teams t JOIN pack_targets pt
WHERE pack_id = ? AND pt.type = ? AND pt.target_id = t.id
`
			if err := sqlx.SelectContext(ctx, tx, &spec.Targets.Teams, query, spec.ID, fleet.TargetTeam); err != nil {
				return ctxerr.Wrap(ctx, err, "get pack team targets")
			}
		}

		// Load queries
		for _, spec := range specs {
			spec := spec
			query = `
SELECT
query_name, name, description, ` + "`interval`" + `,
snapshot, removed, shard, platform, version, denylist
FROM scheduled_queries
WHERE pack_id = ?
`
			if err := sqlx.SelectContext(ctx, tx, &spec.Queries, query, spec.ID); err != nil {
				return ctxerr.Wrap(ctx, err, "get pack queries")
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return specs, nil
}

func (ds *Datastore) GetPackSpec(ctx context.Context, name string) (*fleet.PackSpec, error) {
	spec := &fleet.PackSpec{}
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Get basic spec
		query := "SELECT id, name, description, platform, disabled FROM packs WHERE name = ?"
		if err := sqlx.GetContext(ctx, tx, spec, query, name); err != nil {
			if err == sql.ErrNoRows {
				return ctxerr.Wrap(ctx, notFound("Pack").WithName(name))
			}
			return ctxerr.Wrap(ctx, err, "get packs")
		}

		// Load label targets
		query = `
SELECT l.name
FROM labels l JOIN pack_targets pt
WHERE pack_id = ? AND pt.type = ? AND pt.target_id = l.id
`
		if err := sqlx.SelectContext(ctx, tx, &spec.Targets.Labels, query, spec.ID, fleet.TargetLabel); err != nil {
			return ctxerr.Wrap(ctx, err, "get pack label targets")
		}

		// Load team targets
		query = `
SELECT t.name
FROM teams t JOIN pack_targets pt
WHERE pack_id = ? AND pt.type = ? AND pt.target_id = t.id
`
		if err := sqlx.SelectContext(ctx, tx, &spec.Targets.Teams, query, spec.ID, fleet.TargetTeam); err != nil {
			return ctxerr.Wrap(ctx, err, "get pack team targets")
		}

		// Load queries
		query = `
SELECT
query_name, name, description, ` + "`interval`" + `,
snapshot, removed, shard, platform, version, denylist
FROM scheduled_queries
WHERE pack_id = ?
`
		if err := sqlx.SelectContext(ctx, tx, &spec.Queries, query, spec.ID); err != nil {
			return ctxerr.Wrap(ctx, err, "get pack queries")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return spec, nil
}

func (ds *Datastore) PackByName(ctx context.Context, name string, opts ...fleet.OptionalArg) (*fleet.Pack, bool, error) {
	sqlStatement := `
		SELECT *
			FROM packs
			WHERE name = ?
	`
	var pack fleet.Pack
	err := sqlx.GetContext(ctx, ds.reader(ctx), &pack, sqlStatement, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, ctxerr.Wrap(ctx, err, "fetch pack by name")
	}

	if err := loadPackTargetsDB(ctx, ds.reader(ctx), &pack); err != nil {
		return nil, false, err
	}

	return &pack, true, nil
}

// NewPack creates a new Pack
func (ds *Datastore) NewPack(ctx context.Context, pack *fleet.Pack, opts ...fleet.OptionalArg) (*fleet.Pack, error) {
	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		query := `
			INSERT INTO packs
			(name, description, platform, disabled)
			VALUES ( ?, ?, ?, ? )
		`
		result, err := tx.ExecContext(ctx, query, pack.Name, pack.Description, pack.Platform, pack.Disabled)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert pack")
		}

		id, _ := result.LastInsertId()
		pack.ID = uint(id) //nolint:gosec // dismiss G115

		if err := replacePackTargetsDB(ctx, tx, pack); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return pack, nil
}

func replacePackTargetsDB(ctx context.Context, tx sqlx.ExecerContext, pack *fleet.Pack) error {
	sql := `DELETE FROM pack_targets WHERE pack_id = ?`
	if _, err := tx.ExecContext(ctx, sql, pack.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete pack targets")
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
		if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert label targets")
		}
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
		if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert host targets")
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
		if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert team targets")
		}
	}

	return nil
}

func loadPackTargetsDB(ctx context.Context, q sqlx.QueryerContext, pack *fleet.Pack) error {
	var targets []fleet.Target
	sql := `
	SELECT type, target_id,
		COALESCE(
			CASE
				WHEN type = ? THEN (SELECT hostname FROM hosts WHERE id = target_id)
				WHEN type = ? THEN (SELECT name FROM teams WHERE id = target_id)
				WHEN type = ? THEN (SELECT name FROM labels WHERE id = target_id)
			END
		, '') AS display_text
	FROM pack_targets
	WHERE pack_id = ?`
	if err := sqlx.SelectContext(
		ctx, q, &targets, sql,
		fleet.TargetHost, fleet.TargetTeam, fleet.TargetLabel, pack.ID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "select pack targets")
	}

	pack.HostIDs, pack.LabelIDs, pack.TeamIDs = []uint{}, []uint{}, []uint{}
	pack.Hosts, pack.Labels, pack.Teams = []fleet.Target{}, []fleet.Target{}, []fleet.Target{}
	for _, target := range targets {
		switch target.Type {
		case fleet.TargetHost:
			pack.HostIDs = append(pack.HostIDs, target.TargetID)
			pack.Hosts = append(pack.Hosts, target)
		case fleet.TargetLabel:
			pack.LabelIDs = append(pack.LabelIDs, target.TargetID)
			pack.Labels = append(pack.Labels, target)
		case fleet.TargetTeam:
			pack.TeamIDs = append(pack.TeamIDs, target.TargetID)
			pack.Teams = append(pack.Teams, target)
		default:
			return ctxerr.Errorf(ctx, "unknown target type: %d", target.Type)
		}
	}

	return nil
}

// SavePack stores changes to pack
func (ds *Datastore) SavePack(ctx context.Context, pack *fleet.Pack) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		query := `
			UPDATE packs
			SET name = ?, platform = ?, disabled = ?, description = ?
			WHERE id = ?
	`

		results, err := tx.ExecContext(ctx, query, pack.Name, pack.Platform, pack.Disabled, pack.Description, pack.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "updating pack")
		}
		rowsAffected, err := results.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "rows affected updating packs")
		}
		if rowsAffected == 0 {
			return ctxerr.Wrap(ctx, notFound("Pack").WithID(pack.ID))
		}

		return replacePackTargetsDB(ctx, tx, pack)
	})
}

// DeletePack deletes a fleet.Pack so that it won't show up in results.
func (ds *Datastore) DeletePack(ctx context.Context, name string) error {
	return ds.deleteEntityByName(ctx, packsTable, name)
}

// Pack fetch fleet.Pack with matching ID
func (ds *Datastore) Pack(ctx context.Context, pid uint) (*fleet.Pack, error) {
	return packDB(ctx, ds.reader(ctx), pid)
}

func packDB(ctx context.Context, q sqlx.QueryerContext, pid uint) (*fleet.Pack, error) {
	query := `SELECT * FROM packs WHERE id = ?`
	pack := &fleet.Pack{}
	err := sqlx.GetContext(ctx, q, pack, query, pid)
	if err == sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, notFound("Pack").WithID(pid))
	} else if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get pack")
	}

	if err := loadPackTargetsDB(ctx, q, pack); err != nil {
		return nil, err
	}

	return pack, nil
}

func teamScheduleName(team *fleet.Team) string {
	return fmt.Sprintf("Team: %s", team.Name)
}

// ListPacks returns all fleet.Pack records limited and sorted by fleet.ListOptions
func (ds *Datastore) ListPacks(ctx context.Context, opt fleet.PackListOptions) ([]*fleet.Pack, error) {
	query := `SELECT * FROM packs WHERE pack_type IS NULL OR pack_type = ''`
	if opt.IncludeSystemPacks {
		query = `SELECT * FROM packs`
	}
	var packs []*fleet.Pack
	query, params := appendListOptionsToSQL(query, &opt.ListOptions)
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &packs, query, params...)
	if err != nil && err != sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, err, "listing packs")
	}

	for _, pack := range packs {
		if err := loadPackTargetsDB(ctx, ds.reader(ctx), pack); err != nil {
			return nil, err
		}
	}

	return packs, nil
}

func (ds *Datastore) ListPacksForHost(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
	return listPacksForHost(ctx, ds.reader(ctx), hid)
}

// listPacksForHost returns all the "user packs" that are configured to run on the given host.
func listPacksForHost(ctx context.Context, db sqlx.QueryerContext, hid uint) ([]*fleet.Pack, error) {
	query := `
	SELECT DISTINCT packs.* FROM (
	(
		SELECT p.* FROM packs p
		JOIN pack_targets pt
		JOIN label_membership lm
		ON (
			p.id = pt.pack_id
			AND pt.target_id = lm.label_id
			AND pt.type = ?
		)
		WHERE lm.host_id = ? AND NOT p.disabled AND p.pack_type IS NULL
	)
	UNION ALL
	(
		SELECT p.* FROM packs p
		JOIN pack_targets pt ON (p.id = pt.pack_id AND pt.type = ? AND pt.target_id = ?)
		WHERE p.pack_type IS NULL
	)
	UNION ALL
	(
		SELECT p.*
		FROM packs p
		JOIN pack_targets pt
		ON (p.id = pt.pack_id AND pt.type = ? AND pt.target_id = (SELECT team_id FROM hosts WHERE id = ?))
		WHERE p.pack_type IS NULL
	)) packs`

	packs := []*fleet.Pack{}
	if err := sqlx.SelectContext(ctx, db, &packs, query,
		fleet.TargetLabel, hid, fleet.TargetHost, hid, fleet.TargetTeam, hid,
	); err != nil && err != sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, err, "listing hosts in pack")
	}

	return packs, nil
}
