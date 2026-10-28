package mysql

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) SetHostGoogleDeviceID(ctx context.Context, hostID uint, googleDeviceID string) (bool, error) {
	_, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO host_google_device_ids (host_id, google_device_id) VALUES (?, ?)`, hostID, googleDeviceID)
	if err == nil {
		return true, nil
	}
	if !IsDuplicate(err) {
		return false, ctxerr.Wrap(ctx, err, "inserting host google device id")
	}

	// Either key can collide: the host's own row tells which one did. MySQL compares the IDs so a
	// casing-only difference counts as the same ID, like it does for the unique key.
	var stored struct {
		ID   string `db:"google_device_id"`
		Same bool   `db:"same"`
	}
	err = sqlx.GetContext(ctx, ds.writer(ctx), &stored,
		`SELECT google_device_id, google_device_id = ? AS same FROM host_google_device_ids WHERE host_id = ?`,
		googleDeviceID, hostID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, fleet.NewInvalidArgumentError("google_device_id", "This Google device ID is already used by another host.").
			WithStatus(http.StatusConflict)
	case err != nil:
		return false, ctxerr.Wrap(ctx, err, "reading host google device id after duplicate")
	case stored.Same:
		return false, nil
	default:
		return false, &fleet.HostGoogleDeviceIDConflictError{StoredID: stored.ID}
	}
}

func (ds *Datastore) GetHostGoogleDeviceID(ctx context.Context, hostID uint) (string, error) {
	var id string
	err := sqlx.GetContext(ctx, ds.reader(ctx), &id, `SELECT google_device_id FROM host_google_device_ids WHERE host_id = ?`, hostID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting host google device id")
	}
	return id, nil
}

func (ds *Datastore) DeleteHostGoogleDeviceID(ctx context.Context, hostID uint, googleDeviceID string) (bool, error) {
	res, err := ds.writer(ctx).ExecContext(ctx,
		`DELETE FROM host_google_device_ids WHERE host_id = ? AND google_device_id = ?`, hostID, googleDeviceID)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "deleting host google device id")
	}
	deleted, _ := res.RowsAffected()
	return deleted > 0, nil
}
