package mysql

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) SetHostGoogleDeviceID(ctx context.Context, hostUUID, customerID, googleDeviceID string, names []string) (bool, error) {
	var changed bool
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		changed = false
		if _, err := claimGoogleConditionalAccessDevice(ctx, tx, hostUUID, customerID); err != nil {
			return err
		}

		// MySQL compares the IDs so a casing-only difference counts as the same ID, like it
		// does for the unique key.
		var stored struct {
			ID   *string `db:"google_device_id"`
			Same *bool   `db:"same"`
		}
		if err := sqlx.GetContext(ctx, tx, &stored, `
			SELECT google_device_id, google_device_id = ? AS same
			FROM google_conditional_access_devices WHERE host_uuid = ?`, googleDeviceID, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "reading host google device id")
		}
		if stored.ID != nil {
			if stored.Same != nil && *stored.Same {
				return nil
			}
			return &fleet.HostGoogleDeviceIDConflictError{StoredID: *stored.ID}
		}

		if err := bindGoogleDeviceIDDB(ctx, tx, hostUUID, customerID, googleDeviceID); err != nil {
			if IsDuplicate(err) {
				return fleet.NewInvalidArgumentError("google_device_id", "This Google device ID is already used by another host.").
					WithStatus(http.StatusConflict)
			}
			return err
		}
		if len(names) > 0 {
			if err := recordGoogleDeviceUsersDB(ctx, tx, hostUUID, customerID, names); err != nil {
				return err
			}
		}
		changed = true
		return nil
	})
	return changed, err
}

func (ds *Datastore) GetHostGoogleDeviceID(ctx context.Context, hostUUID, customerID string) (string, error) {
	var id string
	// A row with no customer was migrated from the bookmark flow and belongs to whichever
	// customer connects.
	err := sqlx.GetContext(ctx, ds.reader(ctx), &id, `
		SELECT google_device_id FROM google_conditional_access_devices
		WHERE host_uuid = ? AND google_device_id IS NOT NULL AND (? = '' OR customer_id IN ('', ?))`,
		hostUUID, customerID, customerID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting host google device id")
	}
	return id, nil
}

func (ds *Datastore) DeleteHostGoogleDeviceID(ctx context.Context, hostUUID, googleDeviceID string) (bool, error) {
	var deleted bool
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		deleted = false
		var same *bool
		err := sqlx.GetContext(ctx, tx, &same, `
			SELECT google_device_id = ? FROM google_conditional_access_devices WHERE host_uuid = ? FOR UPDATE`,
			googleDeviceID, hostUUID)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && (same == nil || !*same)) {
			return nil
		}
		if err != nil {
			return ctxerr.Wrap(ctx, err, "locking host google device id")
		}
		if err := queueGoogleConditionalAccessUnmarkDB(ctx, tx, []string{hostUUID}); err != nil {
			return err
		}
		deleted = true
		return nil
	})
	return deleted, err
}

// setGoogleDeviceIDDB sets or clears the host's device ID. The caller holds the lock on
// the host's row. The error is returned unwrapped so the caller can tell a duplicate.
func setGoogleDeviceIDDB(ctx context.Context, tx sqlx.ExtContext, hostUUID string, googleDeviceID *string) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE google_conditional_access_devices SET google_device_id = ? WHERE host_uuid = ?`, googleDeviceID, hostUUID)
	return err
}

// bindGoogleDeviceIDDB sets the host's device ID, taking it from a row left by another
// customer, which doesn't count. A device bound to another host of the customer, or to a
// row with no customer yet, fails with a duplicate error, returned unwrapped.
func bindGoogleDeviceIDDB(ctx context.Context, tx sqlx.ExtContext, hostUUID, customerID, googleDeviceID string) error {
	err := setGoogleDeviceIDDB(ctx, tx, hostUUID, &googleDeviceID)
	if !IsDuplicate(err) {
		return err
	}
	res, releaseErr := tx.ExecContext(ctx, `
		UPDATE google_conditional_access_devices SET google_device_id = NULL
		WHERE google_device_id = ? AND customer_id NOT IN (?, '')`, googleDeviceID, customerID)
	if releaseErr != nil {
		return ctxerr.Wrap(ctx, releaseErr, "releasing previous customer's google device id")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return err
	}
	return setGoogleDeviceIDDB(ctx, tx, hostUUID, &googleDeviceID)
}

// repointGoogleDeviceIDDB keeps the host's device ID on a device it has a registered
// name under: when the last one under it is gone, the ID moves to the device of another
// registered name, or is cleared. The caller holds the lock on the host's row.
func repointGoogleDeviceIDDB(ctx context.Context, tx sqlx.ExtContext, hostUUID string) error {
	var stored *string
	if err := sqlx.GetContext(ctx, tx, &stored,
		`SELECT google_device_id FROM google_conditional_access_devices WHERE host_uuid = ?`, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "reading host google device id")
	}
	if stored == nil {
		return nil
	}
	stmt, args, err := sqlx.In(`
		SELECT device_user_name FROM google_conditional_access_device_users
		WHERE host_uuid = ? AND state IN (?) ORDER BY device_user_name`, hostUUID,
		[]fleet.GoogleConditionalAccessDeviceUserState{fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserManaged})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building select of registered google device users")
	}
	var names []string
	if err := sqlx.SelectContext(ctx, tx, &names, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting registered google device users")
	}
	next, kept := googleDeviceIDForNames(*stored, names)
	if kept {
		return nil
	}
	err = setGoogleDeviceIDDB(ctx, tx, hostUUID, next)
	if IsDuplicate(err) {
		// Another host is bound to that device; this one is left without an ID.
		err = setGoogleDeviceIDDB(ctx, tx, hostUUID, nil)
	}
	return ctxerr.Wrap(ctx, err, "moving host google device id")
}

// googleDeviceIDForNames picks the device ID of a host with the given names: stored,
// reported as kept, when one of the names is under it, else the device of the first
// name, or nil with no names.
func googleDeviceIDForNames(stored string, names []string) (id *string, kept bool) {
	for _, name := range names {
		deviceID := fleet.GoogleDeviceIDFromDeviceUserName(name)
		if deviceID == "" {
			continue
		}
		if stored != "" && strings.EqualFold(deviceID, stored) {
			return &stored, true
		}
		if id == nil {
			id = &deviceID
		}
	}
	return id, false
}
