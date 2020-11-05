package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (d *Datastore) NewCarve(metadata *kolide.CarveMetadata) (*kolide.CarveMetadata, error) {
	stmt := `INSERT INTO carve_metadata (
		host_id,
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
		?
	)`

	result, err := d.db.Exec(
		stmt,
		metadata.HostId,
		metadata.Name,
		metadata.BlockCount,
		metadata.BlockSize,
		metadata.CarveSize,
		metadata.CarveId,
		metadata.RequestId,
		metadata.SessionId,
	)
	if err != nil {
		return nil, errors.Wrap(err, "insert carve metadata")
	}

	id, _ := result.LastInsertId()
	metadata.ID = id

	return metadata, nil
}

func (d *Datastore) CleanupCarves(now time.Time) (int, error) {
	var countExpired int
	err := d.withRetryTxx(func(tx *sqlx.Tx) error {
		// Get IDs of carves to expire
		stmt := `
			SELECT id
			FROM carve_metadata
			WHERE expired = 0 AND created_at < (? - INTERVAL 24 HOUR)
			LIMIT 50000
		`
		var expiredCarves []int64
		if err := tx.Select(&expiredCarves, stmt, now); err != nil {
			return errors.Wrap(err, "get expired carves")
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
			return errors.Wrap(err, "IN for DELETE FROM carve_blocks")
		}
		stmt = tx.Rebind(stmt)
		if _, err := d.db.Exec(stmt, args...); err != nil {
			return errors.Wrap(err, "delete carve blocks")
		}

		// Mark metadata expired
		stmt = `
			UPDATE carve_metadata
			SET expired = 1
			WHERE id IN (?)
		`
		stmt, args, err = sqlx.In(stmt, expiredCarves)
		if err != nil {
			return errors.Wrap(err, "IN for UPDATE carve_metadata")
		}
		stmt = tx.Rebind(stmt)
		if _, err := d.db.Exec(stmt, args...); err != nil {
			return errors.Wrap(err, "update carve_metadtata")
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
			(SELECT COALESCE(MAX(block_id), -1) FROM carve_blocks WHERE metadata_id = id) AS max_block
`

func (d *Datastore) Carve(carveId int64) (*kolide.CarveMetadata, error) {
	stmt := fmt.Sprintf(`
		SELECT %s
		FROM carve_metadata
		WHERE id = ?`,
		carveSelectFields,
	)

	var metadata kolide.CarveMetadata
	if err := d.db.Get(&metadata, stmt, carveId); err != nil {
		return nil, errors.Wrap(err, "get carve by ID")
	}

	return &metadata, nil
}

func (d *Datastore) CarveBySessionId(sessionId string) (*kolide.CarveMetadata, error) {
	stmt := fmt.Sprintf(`
		SELECT %s
		FROM carve_metadata
		WHERE session_id = ?`,
		carveSelectFields,
	)

	var metadata kolide.CarveMetadata
	if err := d.db.Get(&metadata, stmt, sessionId); err != nil {
		return nil, errors.Wrap(err, "get carve by session ID")
	}

	return &metadata, nil
}

func (d *Datastore) CarveByName(name string) (*kolide.CarveMetadata, error) {
	stmt := fmt.Sprintf(`
		SELECT %s
		FROM carve_metadata
		WHERE name = ?`,
		carveSelectFields,
	)

	var metadata kolide.CarveMetadata
	if err := d.db.Get(&metadata, stmt, name); err != nil {
		return nil, errors.Wrap(err, "get carve by name")
	}

	return &metadata, nil
}

func (d *Datastore) ListCarves(opt kolide.CarveListOptions) ([]*kolide.CarveMetadata, error) {
	stmt := fmt.Sprintf(`
		SELECT %s
		FROM carve_metadata`,
		carveSelectFields,
	)
	if !opt.Expired {
		stmt += ` WHERE NOT expired `
	}
	stmt = appendListOptionsToSQL(stmt, opt.ListOptions)
	carves := []*kolide.CarveMetadata{}
	if err := d.db.Select(&carves, stmt); err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "list carves")
	}

	return carves, nil
}

func (d *Datastore) NewBlock(metadataId int64, blockId int64, data []byte) error {
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
	if _, err := d.db.Exec(stmt, metadataId, blockId, data); err != nil {
		return errors.Wrap(err, "insert carve block")
	}

	return nil
}

func (d *Datastore) GetBlock(metadataId int64, blockId int64) ([]byte, error) {
	stmt := `
		SELECT data
		FROM carve_blocks
		WHERE metadata_id = ? AND block_id = ?
	`
	var data []byte
	if err := d.db.Get(&data, stmt, metadataId, blockId); err != nil {
		return nil, errors.Wrap(err, "select data")
	}

	return data, nil
}
