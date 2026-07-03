package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
	"golang.org/x/text/unicode/norm"
)

var customHostVitalAllowedOrderKeys = common_mysql.OrderKeyAllowlist{
	"name":       "name",
	"id":         "id",
	"updated_at": "updated_at",
}

func (ds *Datastore) CreateCustomHostVital(ctx context.Context, name string) (fleet.CustomHostVital, error) {
	res, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO custom_host_vitals (name) VALUES (?)`,
		name,
	)
	if err != nil {
		if IsDuplicate(err) {
			return fleet.CustomHostVital{}, ctxerr.Wrap(ctx, alreadyExists("name", name), "found duplicate")
		}
		return fleet.CustomHostVital{}, ctxerr.Wrap(ctx, err, "insert custom host vital")
	}
	id, _ := res.LastInsertId()
	return fleet.CustomHostVital{ID: uint(id), Name: name}, nil //nolint:gosec // dismiss G115
}

func (ds *Datastore) ListCustomHostVitals(ctx context.Context, opt fleet.ListOptions) (
	customHostVitals []fleet.CustomHostVital, meta *fleet.PaginationMetadata, count int, err error,
) {
	stmt := `SELECT id, name, created_at, updated_at FROM custom_host_vitals WHERE true`

	// normalize the name for full Unicode support (Unicode equivalence).
	// Search matches the name OR the variable name (the derived
	// `$FLEET_HOST_VITAL_<id>` token). The second column is a hardcoded SQL
	// expression (not user input); searchLike escapes the LIKE pattern.
	normMatch := norm.NFC.String(opt.MatchQuery)
	whereClauses, args := searchLike("", nil, normMatch, "name", `CONCAT('$FLEET_HOST_VITAL_', id)`)
	stmt += whereClauses

	// perform a second query to grab the count
	// build the count statement before adding pagination constraints
	countStmt := fmt.Sprintf("SELECT COUNT(DISTINCT id) FROM (%s) AS s", stmt)

	stmt, args, err = appendListOptionsWithCursorToSQLSecure(stmt, args, &opt, customHostVitalAllowedOrderKeys)
	if err != nil {
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "apply list options")
	}

	dbReader := ds.reader(ctx)
	if err := sqlx.SelectContext(ctx, dbReader, &customHostVitals, stmt, args...); err != nil {
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "listing custom host vitals")
	}
	if err := sqlx.GetContext(ctx, dbReader, &count, countStmt, args...); err != nil {
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "get custom host vitals count")
	}

	if opt.IncludeMetadata {
		meta = &fleet.PaginationMetadata{
			HasPreviousResults: opt.Page > 0,
			TotalResults:       uint(count), //nolint:gosec // dismiss G115
		}
		// `appendListOptionsWithCursorToSQL` used above to build the query statement will cause this discrepancy.
		if len(customHostVitals) > int(opt.PerPage) { //nolint:gosec // dismiss G115
			meta.HasNextResults = true
			customHostVitals = customHostVitals[:len(customHostVitals)-1]
		}
	}

	return customHostVitals, meta, count, nil
}

func (ds *Datastore) UpdateCustomHostVital(ctx context.Context, id uint, name string) (fleet.CustomHostVital, error) {
	res, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE custom_host_vitals SET name = ? WHERE id = ?`,
		name, id,
	)
	if err != nil {
		if IsDuplicate(err) {
			return fleet.CustomHostVital{}, ctxerr.Wrap(ctx, alreadyExists("name", name), "found duplicate")
		}
		return fleet.CustomHostVital{}, ctxerr.Wrap(ctx, err, "update custom host vital")
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		// No rows affected can mean the id was not found, or the name is unchanged.
		// Distinguish the two so a no-op rename doesn't surface as NotFound.
		var exists bool
		if err := sqlx.GetContext(ctx, ds.reader(ctx), &exists,
			`SELECT 1 FROM custom_host_vitals WHERE id = ?`, id); err != nil {
			if err == sql.ErrNoRows {
				return fleet.CustomHostVital{}, ctxerr.Wrap(ctx, notFound("CustomHostVital").WithID(id))
			}
			return fleet.CustomHostVital{}, ctxerr.Wrap(ctx, err, "check custom host vital exists")
		}
	}
	return fleet.CustomHostVital{ID: id, Name: name}, nil
}

func (ds *Datastore) DeleteCustomHostVital(ctx context.Context, id uint) (name string, err error) {
	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		err := sqlx.GetContext(ctx, tx, &name, `SELECT name FROM custom_host_vitals WHERE id = ?`, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return ctxerr.Wrap(ctx, notFound("CustomHostVital").WithID(id))
			}
			return ctxerr.Wrap(ctx, err, "getting name of custom host vital to delete")
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM custom_host_vitals WHERE id = ?`, id); err != nil {
			return ctxerr.Wrap(ctx, err, "delete custom host vital")
		}
		return nil
	}); err != nil {
		return "", ctxerr.Wrap(ctx, err, "delete custom host vital")
	}

	return name, nil
}

func (ds *Datastore) SetHostCustomHostVitalValue(ctx context.Context, hostID uint, vitalID uint, value string) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO host_custom_host_vitals (host_id, custom_host_vital_id, value)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE value = VALUES(value)`,
		hostID, vitalID, value,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set host custom host vital value")
	}
	return nil
}

func (ds *Datastore) GetHostCustomHostVitals(ctx context.Context, hostID uint) ([]fleet.HostCustomHostVital, error) {
	var vitals []fleet.HostCustomHostVital
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &vitals, `
		SELECT chv.id AS custom_host_vital_id, chv.name, hchv.value
		FROM host_custom_host_vitals hchv
		JOIN custom_host_vitals chv ON chv.id = hchv.custom_host_vital_id
		WHERE hchv.host_id = ?`,
		hostID,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host custom host vitals")
	}
	return vitals, nil
}

func (ds *Datastore) GetCustomHostVitals(ctx context.Context, ids []uint) ([]fleet.CustomHostVital, error) {
	stmt, args, err := sqlx.In(`
		SELECT id, name, created_at, updated_at
		FROM custom_host_vitals
		WHERE id IN (?)`, ids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build custom host vitals query")
	}

	var vitals []fleet.CustomHostVital
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &vitals, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get custom host vitals")
	}
	return vitals, nil
}
