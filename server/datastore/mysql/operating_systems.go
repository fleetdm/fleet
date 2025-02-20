package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListOperatingSystems(ctx context.Context) ([]fleet.OperatingSystem, error) {
	return listOperatingSystemsDB(ctx, ds.reader(ctx))
}

func listOperatingSystemsDB(ctx context.Context, tx sqlx.QueryerContext) ([]fleet.OperatingSystem, error) {
	var os []fleet.OperatingSystem
	if err := sqlx.SelectContext(ctx, tx, &os, `SELECT id, name, version, arch, kernel_version, platform, display_version, os_version_id FROM operating_systems`); err != nil {
		return nil, err
	}
	return os, nil
}

func (ds *Datastore) ListOperatingSystemsForPlatform(ctx context.Context, platform string) ([]fleet.OperatingSystem, error) {
	var oses []fleet.OperatingSystem
	sqlStatement := `
		SELECT id, name, version, arch, kernel_version, platform, display_version, os_version_id
		FROM operating_systems
		WHERE platform = ?
	`
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &oses, sqlStatement, platform); err != nil {
		return nil, err
	}
	return oses, nil
}

func (ds *Datastore) UpdateHostOperatingSystem(ctx context.Context, hostID uint, hostOS fleet.OperatingSystem) error {
	// We optimize for the most common case where the operating system for the host has not changed.
	// No DB transaction or DB write is needed in this case.
	updateNeeded, err := isHostOperatingSystemUpdateNeeded(ctx, ds.reader(ctx), hostID, hostOS)
	if err != nil {
		return err
	}
	if !updateNeeded {
		return nil
	}

	const maxDisplayVersionLength = 10 // per DB schema
	if len(hostOS.DisplayVersion) > maxDisplayVersionLength {
		return ctxerr.Errorf(ctx, "host OS display version too long: %s", hostOS.DisplayVersion)
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		os, err := getOrGenerateOperatingSystemDB(ctx, tx, hostOS)
		if err != nil {
			return err
		}
		return upsertHostOperatingSystemDB(ctx, tx, hostID, os.ID)
	})
}

func (ds *Datastore) GetHostOperatingSystem(ctx context.Context, hostID uint) (*fleet.OperatingSystem, error) {
	return getHostOperatingSystemDB(ctx, ds.reader(ctx), hostID)
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
	id, err := getOSVersionID(ctx, tx, hostOS)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get operating system version ID")
	}

	stmt := "INSERT IGNORE INTO operating_systems (name, version, arch, kernel_version, platform, display_version, os_version_id) VALUES (?, ?, ?, ?, ?, ?, ?)"
	if _, err := tx.ExecContext(ctx, stmt, hostOS.Name, hostOS.Version, hostOS.Arch, hostOS.KernelVersion, hostOS.Platform, hostOS.DisplayVersion, id); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "insert new operating system")
	}

	// With INSERT IGNORE, attempting a duplicate insert will return LastInsertId of 0
	// so we retrieve the stored record to guard against that case.
	switch storedOS, err := getOperatingSystemDB(ctx, tx, hostOS); {
	case err == nil:
		return storedOS, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, common_mysql.DoRetryErr
	default:
		return nil, ctxerr.Wrap(ctx, err, "get new operating system")
	}
}

func getOSVersionID(ctx context.Context, tx sqlx.ExtContext, hostOS fleet.OperatingSystem) (uint, error) {
	stmt := "SELECT os_version_id FROM operating_systems WHERE name = ? AND version = ?"
	var id uint
	err := sqlx.GetContext(ctx, tx, &id, stmt, hostOS.Name, hostOS.Version)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		id, err = nextOSVersionID(ctx, tx)
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "generate next operating system version ID")
		}
	} else if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "get operating system version ID")
	}

	return id, nil
}

func nextOSVersionID(ctx context.Context, tx sqlx.ExtContext) (uint, error) {
	var id *uint
	stmt := "SELECT MAX(os_version_id) FROM operating_systems"
	err := sqlx.GetContext(ctx, tx, &id, stmt)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return 1, nil
	} else if err != nil {
		return 0, err
	}

	if id == nil {
		return 1, nil
	}

	return *id + 1, nil
}

// getOperatingSystemDB queries the `operating_systems` table with the
// name, version, arch, and kernel_version of the given operating system.
// If found, it returns the record including the associated ID.
func getOperatingSystemDB(ctx context.Context, tx sqlx.ExtContext, hostOS fleet.OperatingSystem) (*fleet.OperatingSystem, error) {
	var os fleet.OperatingSystem
	stmt := "SELECT id, name, version, arch, kernel_version, platform, display_version, os_version_id FROM operating_systems WHERE name = ? AND version = ? AND arch = ? AND kernel_version = ? AND platform = ? AND display_version = ?"
	if err := sqlx.GetContext(ctx, tx, &os, stmt, hostOS.Name, hostOS.Version, hostOS.Arch, hostOS.KernelVersion, hostOS.Platform, hostOS.DisplayVersion); err != nil {
		return nil, err
	}
	return &os, nil
}

func isHostOperatingSystemUpdateNeeded(ctx context.Context, qc sqlx.QueryerContext, hostID uint, hostOS fleet.OperatingSystem) (
	bool, error,
) {
	var resultPresent bool
	err := sqlx.GetContext(
		ctx, qc, &resultPresent,
		`SELECT 1 FROM host_operating_system hos
				INNER JOIN operating_systems os ON hos.os_id = os.id
				WHERE hos.host_id = ? AND os.name = ? AND os.version = ? AND os.arch = ? AND os.kernel_version = ? AND os.platform = ? AND os.display_version = ?`,
		hostID, hostOS.Name, hostOS.Version, hostOS.Arch, hostOS.KernelVersion, hostOS.Platform, hostOS.DisplayVersion,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return true, nil
	case err != nil:
		return false, ctxerr.Wrap(ctx, err, "check host operating system")
	default:
		return !resultPresent, nil
	}
}

// upsertHostOperatingSystemDB upserts the host operating system table
// with the operating system id for the given host ID
func upsertHostOperatingSystemDB(ctx context.Context, tx sqlx.ExtContext, hostID uint, osID uint) error {
	// We do not use the `UPDATE` then `INSERT` pattern here because it causes a deadlock when multiple hosts are enrolled concurrently.
	// This method will rarely be called -- only when the host_operating_system needs to be updated.
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO host_operating_system (host_id, os_id) VALUES (?, ?)
				ON DUPLICATE KEY UPDATE os_id = VALUES(os_id)`, hostID, osID,
	)
	return err
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
func getHostOperatingSystemDB(ctx context.Context, tx sqlx.QueryerContext, hostID uint) (*fleet.OperatingSystem, error) {
	var os fleet.OperatingSystem
	stmt := "SELECT id, name, version, arch, kernel_version, platform, display_version, os_version_id FROM operating_systems WHERE id = (SELECT os_id FROM host_operating_system WHERE host_id = ?)"
	if err := sqlx.GetContext(ctx, tx, &os, stmt, hostID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host os")
	}
	return &os, nil
}

func (ds *Datastore) CleanupHostOperatingSystems(ctx context.Context) error {
	// delete operating_systems records that are not associated with any host (e.g., all hosts have
	// upgraded from a prior version)
	stmt := `
	DELETE op
	FROM operating_systems op
	LEFT JOIN host_operating_system hop ON op.id = hop.os_id
	WHERE hop.os_id IS NULL
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt); err != nil {
		return ctxerr.Wrap(ctx, err, "clean up host operating systems")
	}

	return nil
}
