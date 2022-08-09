package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListOperatingSystems(ctx context.Context) ([]fleet.OperatingSystem, error) {
	return listOperatingSystemsDB(ctx, ds.reader)
}

func listOperatingSystemsDB(ctx context.Context, tx sqlx.QueryerContext) ([]fleet.OperatingSystem, error) {
	var os []fleet.OperatingSystem
	if err := sqlx.SelectContext(ctx, tx, &os, `SELECT id, name, version, arch, kernel_version FROM operating_systems`); err != nil {
		return nil, err
	}
	return os, nil
}

func (ds *Datastore) UpdateHostOperatingSystem(ctx context.Context, hostID uint, hostOS fleet.OperatingSystem) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		os, err := getOrGenerateOperatingSystemDB(ctx, ds.writer, hostOS)
		if err != nil {
			return err
		}
		return upsertHostOperatingSystemDB(ctx, ds.writer, hostID, os.ID)
	})
}

// getOrGenerateOperatingSystemDB queries the `operating_systems` table with the
// name, version, arch, and kernel_version of the given operating system. If found,
// it returns the record including the associated ID. If not found, it returns a call
// to `newOperatingSystemDB`, which inserts a new record and returns the record
// including the newly associated ID.
func getOrGenerateOperatingSystemDB(ctx context.Context, tx sqlx.ExtContext, hostOS fleet.OperatingSystem) (*fleet.OperatingSystem, error) {
	switch os, err := getOperatingSystemDB(ctx, tx, hostOS); {
	case err == nil:
		return os, nil
	case errors.Is(err, sql.ErrNoRows):
		return newOperatingSystemDB(ctx, tx, hostOS)
	default:
		return nil, ctxerr.Wrap(ctx, err, "get operating system")
	}
}

// `newOperatingSystemDB` inserts a record for the given operating system and
// returns the record including the newly associated ID.
func newOperatingSystemDB(ctx context.Context, tx sqlx.ExtContext, hostOS fleet.OperatingSystem) (*fleet.OperatingSystem, error) {
	stmt := "INSERT IGNORE INTO operating_systems (name, version, arch, kernel_version, platform) VALUES (?, ?, ?, ?, ?)"
	if _, err := tx.ExecContext(ctx, stmt, hostOS.Name, hostOS.Version, hostOS.Arch, hostOS.KernelVersion, hostOS.Platform); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "insert new operating system")
	}

	// With INSERT IGNORE, attempting a duplicate insert will return LastInsertId of 0
	// so we retrieve the stored record to guard against that case.
	switch storedOS, err := getOperatingSystemDB(ctx, tx, hostOS); {
	case err == nil:
		return storedOS, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, doRetryErr
	default:
		return nil, ctxerr.Wrap(ctx, err, "get new operating system")
	}
}

// getOperatingSystemDB queries the `operating_systems` table with the
// name, version, arch, and kernel_version of the given operating system.
// If found, it returns the record including the associated ID.
func getOperatingSystemDB(ctx context.Context, tx sqlx.ExtContext, hostOS fleet.OperatingSystem) (*fleet.OperatingSystem, error) {
	var os fleet.OperatingSystem
	stmt := "SELECT id, name, version, arch, kernel_version FROM operating_systems WHERE name = ? AND version = ? AND arch = ? AND kernel_version = ?"
	if err := sqlx.GetContext(ctx, tx, &os, stmt, hostOS.Name, hostOS.Version, hostOS.Arch, hostOS.KernelVersion); err != nil {
		return nil, err
	}
	return &os, nil
}

// upsertHostOperatingSystemDB upserts the host operating system table
// with the operating system id for the given host ID
func upsertHostOperatingSystemDB(ctx context.Context, tx sqlx.ExtContext, hostID uint, osID uint) error {
	res, err := tx.ExecContext(ctx, "UPDATE host_operating_system SET os_id = ? WHERE host_id = ?", osID, hostID)
	if err != nil {
		return err
	}

	if n, _ := res.RowsAffected(); n > 0 {
		// update success
		return nil
	}

	// no row to update so insert new row
	_, err = tx.ExecContext(ctx, "INSERT INTO host_operating_system (host_id, os_id) VALUES (?, ?)", hostID, osID)
	if err != nil {
		return err
	}

	return nil
}

// getIDHostOperatingSystemDB queries the `host_operating_system` table and returns the
// operating system ID for the given host ID.
func getIDHostOperatingSystemDB(ctx context.Context, tx sqlx.ExtContext, hostID uint) (uint, error) {
	var id uint
	stmt := "SELECT os_id FROM host_operating_system WHERE host_id = ?"
	if err := sqlx.GetContext(ctx, tx, &id, stmt, hostID); err != nil {
		return 0, err
	}
	return id, nil
}

// getIDHostOperatingSystemDB queries the `operating_systems` table and returns the
// operating system record associated with the given host ID based on a subquery
// of the `host_operating_system` table.
func getHostOperatingSystemDB(ctx context.Context, tx sqlx.ExtContext, hostID uint) (*fleet.OperatingSystem, error) {
	var os fleet.OperatingSystem
	stmt := "SELECT id, name, version, arch, kernel_version FROM operating_systems WHERE id = (SELECT os_id FROM host_operating_system WHERE host_id = ?)"
	if err := sqlx.GetContext(ctx, tx, &os, stmt, hostID); err != nil {
		return nil, err
	}
	return &os, nil
}
