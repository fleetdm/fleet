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
	if err := sqlx.SelectContext(ctx, tx, &os, `SELECT * FROM operating_systems`); err != nil {
		return nil, err
	}
	return os, nil
}

func (ds *Datastore) UpdateHostOperatingSystem(ctx context.Context, hostID uint, hostOS fleet.OperatingSystem) error {
	os, err := maybeNewOperatingSystemDB(ctx, ds.writer, hostOS)
	if err != nil {
		return err
	}
	return maybeUpdateHostOperatingSystemDB(ctx, ds.writer, hostID, os.ID)
}

// maybeNewOperatingSystemDB queries the `operating_systems` table with the
// name, version, arch, and kernel_version of the given operating system. If found,
// it returns the record including the associated ID. If not found, it returns a call
// to `newOperatingSystemDB`, which inserts a new record and returns the record
// including the newly associated ID.
func maybeNewOperatingSystemDB(ctx context.Context, tx sqlx.ExtContext, hostOS fleet.OperatingSystem) (*fleet.OperatingSystem, error) {
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
	stmt := "INSERT INTO operating_systems (name, version, arch, kernel_version) VALUES (?, ?, ?, ?)"
	if _, err := tx.ExecContext(ctx, stmt, hostOS.Name, hostOS.Version, hostOS.Arch, hostOS.KernelVersion); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "insert new operating system")
	}

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
	stmt := "SELECT * FROM operating_systems WHERE name = ? AND version = ? AND arch = ? AND kernel_version = ? LIMIT 1"
	if err := sqlx.GetContext(ctx, tx, &os, stmt, hostOS.Name, hostOS.Version, hostOS.Arch, hostOS.KernelVersion); err != nil {
		return nil, err
	}
	return &os, nil
}

// maybeUpdateHostOperatingSystemDB checks the `host_operating_system` table
// for the given host ID. If found, it compares the given operating system ID
// against the stored operating system ID and updates the table accordingly.
// If not found, it inserts a new record for the given host id with the given
// operating system id.
func maybeUpdateHostOperatingSystemDB(ctx context.Context, tx sqlx.ExtContext, hostID uint, osID uint) error {
	switch storedID, err := getIDHostOperatingSystemDB(ctx, tx, hostID); {
	case errors.Is(err, sql.ErrNoRows):
		if err := insertHostOperatingSystemDB(ctx, tx, hostID, osID); err != nil {
			return ctxerr.Wrap(ctx, err, "insert host operating system")
		}
	case err != nil:
		return ctxerr.Wrap(ctx, err, "get host operating system")
	case storedID != osID:
		if err := updateHostOperatingSystemDB(ctx, tx, hostID, osID); err != nil {
			return ctxerr.Wrap(ctx, err, "update host operating system")
		}
	default:
		// no update necessary
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
	// TODO: can osquery host have more than one os?
	stmt := "SELECT * FROM operating_systems os WHERE os.id = (SELECT os_id FROM host_operating_system WHERE host_id = ?)"
	if err := sqlx.GetContext(ctx, tx, &os, stmt, hostID); err != nil {
		return nil, err
	}
	return &os, nil
}

// updateHostOperatingSystemDB updates the record for the given host id in the
// `host_operating_system` table with the given operating system id.
func updateHostOperatingSystemDB(ctx context.Context, tx sqlx.ExtContext, hostID uint, osID uint) error {
	stmt := "UPDATE host_operating_system SET os_id = ? WHERE host_id = ?"
	if _, err := tx.ExecContext(ctx, stmt, osID, hostID); err != nil {
		return err
	}
	return nil
}

// insertHostOperatingSystemDB inserts a new record into the `host_operating_systems`
// table for given host ID and operating system ID.
func insertHostOperatingSystemDB(ctx context.Context, tx sqlx.ExtContext, hostID uint, osID uint) error {
	stmt := "INSERT INTO host_operating_system (host_id, os_id) VALUES (?, ?)"
	if _, err := tx.ExecContext(ctx, stmt, hostID, osID); err != nil {
		return err
	}
	return nil
}
