package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (d *Datastore) NewCarve(ctx context.Context, metadata *fleet.CarveMetadata) (*fleet.CarveMetadata, error) {
	stmt := `INSERT INTO carve_metadata (
		host_id,
		created_at,
		name,
		block_count,
		block_size,
		carve_size,
		carve_id,
		request_id,
		session_id
	) VALUES (
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?
	)`

	result, err := d.writer.ExecContext(
		ctx,
		stmt,
		metadata.HostId,
		metadata.CreatedAt.Format(mySQLTimestampFormat),
		metadata.Name,
		metadata.BlockCount,
		metadata.BlockSize,
		metadata.CarveSize,
		metadata.CarveId,
		metadata.RequestId,
		metadata.SessionId,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "insert carve metadata")
	}

	id, _ := result.LastInsertId()
	metadata.ID = id

	return metadata, nil
}

// UpdateCarve updates the carve metadata in database
// Only max_block and expired are updatable
func (d *Datastore) UpdateCarve(ctx context.Context, metadata *fleet.CarveMetadata) error {
	return updateCarveDB(ctx, d.writer, metadata)
}

func updateCarveDB(ctx context.Context, exec sqlx.ExecerContext, metadata *fleet.CarveMetadata) error {
	stmt := `
		UPDATE carve_metadata SET
			max_block = ?,
			expired = ?
		WHERE id = ?
	`
	_, err := exec.ExecContext(
		ctx,
		stmt,
		metadata.MaxBlock,
		metadata.Expired,
		metadata.ID,
	)
	return ctxerr.Wrap(ctx, err, "update carve metadata")
}

func (d *Datastore) CleanupCarves(ctx context.Context, now time.Time) (int, error) {
	var countExpired int
	err := d.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Get IDs of carves to expire
		stmt := `
			SELECT id
			FROM carve_metadata
			WHERE expired = 0 AND created_at < (? - INTERVAL 24 HOUR)
			LIMIT 50000
		`
		var expiredCarves []int64
		if err := sqlx.SelectContext(ctx, tx, &expiredCarves, stmt, now); err != nil {
			return ctxerr.Wrap(ctx, err, "get expired carves")
		}

		countExpired = len(expiredCarves)

		if len(expiredCarves) == 0 {
			// Nothing to do
			return nil
		}

		// Delete carve block data
		stmt = `
			DELETE FROM carve_blocks
			WHERE metadata_id IN (?)
		`
		stmt, args, err := sqlx.In(stmt, expiredCarves)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "IN for DELETE FROM carve_blocks")
		}
		stmt = tx.Rebind(stmt)
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete carve blocks")
		}

		// Mark metadata expired
		stmt = `
			UPDATE carve_metadata
			SET expired = 1
			WHERE id IN (?)
		`
		stmt, args, err = sqlx.In(stmt, expiredCarves)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "IN for UPDATE carve_metadata")
		}
		stmt = tx.Rebind(stmt)
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "update carve_metadtata")
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return countExpired, nil
}

// Selecting max_block should be very efficient because MySQL is able to use
// the index metadata and optimizes away the SELECT.
const carveSelectFields = `
			id,
			host_id,
			created_at,
			name,
			block_count,
			block_size,
			carve_size,
			carve_id,
			request_id,
			session_id,
			expired,
			max_block
`

func (d *Datastore) Carve(ctx context.Context, carveId int64) (*fleet.CarveMetadata, error) {
	stmt := fmt.Sprintf(`
		SELECT %s
		FROM carve_metadata
		WHERE id = ?`,
		carveSelectFields,
	)

	var metadata fleet.CarveMetadata
	if err := sqlx.GetContext(ctx, d.reader, &metadata, stmt, carveId); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Carve").WithID(uint(carveId)))
		}
		return nil, ctxerr.Wrap(ctx, err, "get carve by ID")
	}

	return &metadata, nil
}

func (d *Datastore) CarveBySessionId(ctx context.Context, sessionId string) (*fleet.CarveMetadata, error) {
	stmt := fmt.Sprintf(`
		SELECT %s
		FROM carve_metadata
		WHERE session_id = ?`,
		carveSelectFields,
	)

	var metadata fleet.CarveMetadata
	if err := sqlx.GetContext(ctx, d.reader, &metadata, stmt, sessionId); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("CarveBySessionId").WithName(sessionId))
		}
		return nil, ctxerr.Wrap(ctx, err, "get carve by session ID")
	}

	return &metadata, nil
}

func (d *Datastore) CarveByName(ctx context.Context, name string) (*fleet.CarveMetadata, error) {
	stmt := fmt.Sprintf(`
		SELECT %s
		FROM carve_metadata
		WHERE name = ?`,
		carveSelectFields,
	)

	var metadata fleet.CarveMetadata
	if err := sqlx.GetContext(ctx, d.reader, &metadata, stmt, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Carve").WithName(name))
		}
		return nil, ctxerr.Wrap(ctx, err, "get carve by name")
	}

	return &metadata, nil
}

func (d *Datastore) ListCarves(ctx context.Context, opt fleet.CarveListOptions) ([]*fleet.CarveMetadata, error) {
	stmt := fmt.Sprintf(`
		SELECT %s
		FROM carve_metadata`,
		carveSelectFields,
	)
	if !opt.Expired {
		stmt += ` WHERE NOT expired `
	}
	stmt = appendListOptionsToSQL(stmt, opt.ListOptions)
	carves := []*fleet.CarveMetadata{}
	if err := sqlx.SelectContext(ctx, d.reader, &carves, stmt); err != nil && err != sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, err, "list carves")
	}

	return carves, nil
}

func (d *Datastore) NewBlock(ctx context.Context, metadata *fleet.CarveMetadata, blockId int64, data []byte) error {
	return d.withTx(ctx, func(tx sqlx.ExtContext) error {
		stmt := `
		INSERT INTO carve_blocks (
			metadata_id,
			block_id,
			data
		) VALUES (
			?,
			?,
			?
		)`
		if _, err := tx.ExecContext(ctx, stmt, metadata.ID, blockId, data); err != nil {
			return ctxerr.Wrap(ctx, err, "insert carve block")
		}

		if metadata.MaxBlock < blockId {
			// Update max_block
			metadata.MaxBlock = blockId
			if err := updateCarveDB(ctx, tx, metadata); err != nil {
				return ctxerr.Wrap(ctx, err, "update carve max block")
			}
		}

		return nil
	})
}

func (d *Datastore) GetBlock(ctx context.Context, metadata *fleet.CarveMetadata, blockId int64) ([]byte, error) {
	stmt := `
		SELECT data
		FROM carve_blocks
		WHERE metadata_id = ? AND block_id = ?
	`
	var data []byte
	if err := sqlx.GetContext(ctx, d.reader, &data, stmt, metadata.ID, blockId); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("CarveBlock").WithID(uint(blockId)))
		}
		return nil, ctxerr.Wrap(ctx, err, "select data")
	}

	return data, nil
}
