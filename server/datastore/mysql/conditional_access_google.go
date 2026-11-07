package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetGoogleConditionalAccessIntegration(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, error) {
	var integration fleet.ConditionalAccessGoogleIntegration
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &integration, `
		SELECT customer_id, admin_email, proxy_server_secret, oauth_client_id, state, error_message
		FROM google_conditional_access_integrations
		WHERE id = 1`,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("GoogleConditionalAccessIntegration"))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting google_conditional_access_integrations")
	}
	return &integration, nil
}

func (ds *Datastore) SaveGoogleConditionalAccessIntegration(ctx context.Context, integration *fleet.ConditionalAccessGoogleIntegration) error {
	// The row is updated in place so created_at keeps the time the integration was
	// first set up.
	if _, err := ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO google_conditional_access_integrations
			(id, customer_id, admin_email, proxy_server_secret, oauth_client_id, state, error_message)
		VALUES (1, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			customer_id = VALUES(customer_id),
			admin_email = VALUES(admin_email),
			proxy_server_secret = VALUES(proxy_server_secret),
			oauth_client_id = VALUES(oauth_client_id),
			state = VALUES(state),
			error_message = VALUES(error_message)`,
		integration.CustomerID, integration.AdminEmail, integration.ProxyServerSecret,
		integration.OAuthClientID, integration.State, integration.ErrorMessage,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "saving google_conditional_access_integrations")
	}
	return nil
}

func (ds *Datastore) ListGoogleConditionalAccessDeviceUserNames(ctx context.Context, customerID string) ([]string, error) {
	var names []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &names, `
		SELECT du.device_user_name
		FROM google_conditional_access_device_users du
		JOIN google_conditional_access_devices d ON d.host_uuid = du.host_uuid
		WHERE d.customer_id = ?
		ORDER BY du.device_user_name`, customerID,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing google_conditional_access_device_users")
	}
	return names, nil
}

func (ds *Datastore) DeleteGoogleConditionalAccessDevicesExcept(ctx context.Context, customerID string, limit int) (int, error) {
	// The DeviceUser rows go with the device rows through the foreign key.
	res, err := ds.writer(ctx).ExecContext(ctx,
		`DELETE FROM google_conditional_access_devices WHERE customer_id <> ? LIMIT ?`, customerID, limit)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "deleting google_conditional_access_devices")
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "deleted google_conditional_access_devices")
	}
	return int(n), nil
}

func (ds *Datastore) DeleteGoogleConditionalAccessIntegration(ctx context.Context) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		for _, stmt := range []string{
			`DELETE FROM google_conditional_access_devices`,
			`DELETE FROM google_conditional_access_integrations`,
		} {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return ctxerr.Wrap(ctx, err, "deleting google conditional access rows")
			}
		}
		return nil
	})
}

func (ds *Datastore) GetGoogleConditionalAccessDevice(ctx context.Context, hostUUID, customerID string) (*fleet.GoogleConditionalAccessDevice, error) {
	var dev fleet.GoogleConditionalAccessDevice
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &dev, `
		SELECT host_uuid, customer_id, registration_attempts, first_registration_attempt_at
		FROM google_conditional_access_devices
		WHERE host_uuid = ? AND customer_id = ?`, hostUUID, customerID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("GoogleConditionalAccessDevice").WithName(hostUUID))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting google_conditional_access_devices")
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &dev.DeviceUsers, `
		SELECT host_uuid, device_user_name, state, error_message, updated_at
		FROM google_conditional_access_device_users
		WHERE host_uuid = ?
		ORDER BY device_user_name`, hostUUID,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting google_conditional_access_device_users")
	}
	return &dev, nil
}

// lockGoogleConditionalAccessDevice locks the host's row and makes it belong to the
// customer: it creates it, or takes over another customer's row and drops its names.
// It returns the row as it was before a take-over, or nil when it was created or taken
// over.
func lockGoogleConditionalAccessDevice(ctx context.Context, tx sqlx.ExtContext, hostUUID, customerID string) (*googleDeviceRow, error) {
	var row googleDeviceRow
	err := sqlx.GetContext(ctx, tx, &row, `
		SELECT customer_id, registration_attempts,
			first_registration_attempt_at IS NOT NULL AND first_registration_attempt_at > NOW(6) - INTERVAL ? SECOND AS in_window
		FROM google_conditional_access_devices
		WHERE host_uuid = ?
		FOR UPDATE`, int(fleet.GoogleConditionalAccessRegistrationWindow.Seconds()), hostUUID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO google_conditional_access_devices (host_uuid, customer_id) VALUES (?, ?)`, hostUUID, customerID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "inserting google_conditional_access_devices")
		}
		return nil, nil
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "locking google_conditional_access_devices")
	case row.CustomerID == customerID:
		return &row, nil
	}

	// The names belong to the previous customer's Workspace, which Fleet no longer
	// has a key for, so they are dropped without an unmark.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM google_conditional_access_device_users WHERE host_uuid = ?`, hostUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "deleting previous customer's google_conditional_access_device_users")
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE google_conditional_access_devices
		SET customer_id = ?, registration_token_hash = NULL, registration_token_expires_at = NULL,
			registration_attempts = 0, first_registration_attempt_at = NULL
		WHERE host_uuid = ?`, customerID, hostUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "taking over google_conditional_access_devices")
	}
	return nil, nil
}

type googleDeviceRow struct {
	CustomerID           string `db:"customer_id"`
	RegistrationAttempts uint   `db:"registration_attempts"`
	InWindow             bool   `db:"in_window"`
}

func (ds *Datastore) StartGoogleConditionalAccessRegistration(
	ctx context.Context, hostUUID, customerID string, tokenHash []byte, tokenExpiresAt time.Time,
) (bool, error) {
	var started bool
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		started = false
		row, err := lockGoogleConditionalAccessDevice(ctx, tx, hostUUID, customerID)
		if err != nil {
			return err
		}
		if row != nil && row.InWindow && row.RegistrationAttempts >= fleet.GoogleConditionalAccessMaxRegistrationAttempts {
			return nil
		}

		var expiresAt *time.Time
		if tokenHash != nil {
			expiresAt = &tokenExpiresAt
		}
		attempts := `registration_attempts = 1, first_registration_attempt_at = NOW(6)`
		if row != nil && row.InWindow {
			attempts = `registration_attempts = registration_attempts + 1`
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			UPDATE google_conditional_access_devices
			SET registration_token_hash = ?, registration_token_expires_at = ?, %s
			WHERE host_uuid = ?`, attempts), tokenHash, expiresAt, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "counting google conditional access registration attempt")
		}
		started = true
		return nil
	})
	return started, err
}

const resetGoogleRegistrationColumns = `registration_token_hash = NULL, registration_token_expires_at = NULL,
	registration_attempts = 0, first_registration_attempt_at = NULL`

func (ds *Datastore) ConsumeGoogleConditionalAccessRegistrationToken(ctx context.Context, customerID string, tokenHash []byte) (string, error) {
	var hostUUID string
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := sqlx.GetContext(ctx, tx, &hostUUID, `
			SELECT host_uuid FROM google_conditional_access_devices
			WHERE registration_token_hash = ? AND customer_id = ? AND registration_token_expires_at > NOW(6)
			FOR UPDATE`, tokenHash, customerID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ctxerr.Wrap(ctx, notFound("GoogleConditionalAccessRegistrationToken"))
			}
			return ctxerr.Wrap(ctx, err, "getting google conditional access registration token")
		}
		if _, err := tx.ExecContext(ctx, `UPDATE google_conditional_access_devices SET `+resetGoogleRegistrationColumns+`
			WHERE host_uuid = ?`, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "consuming google conditional access registration token")
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return hostUUID, nil
}

func (ds *Datastore) ResetGoogleConditionalAccessRegistrationAttempts(ctx context.Context, hostUUID, customerID string) error {
	if _, err := ds.writer(ctx).ExecContext(ctx, `UPDATE google_conditional_access_devices SET `+resetGoogleRegistrationColumns+`
		WHERE host_uuid = ? AND customer_id = ?`, hostUUID, customerID); err != nil {
		return ctxerr.Wrap(ctx, err, "resetting google conditional access registration attempts")
	}
	return nil
}

func (ds *Datastore) RecordGoogleConditionalAccessRegistration(ctx context.Context, hostUUID, customerID string, names []string) error {
	if len(names) == 0 {
		return ctxerr.New(ctx, "no google device user names to record")
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if _, err := lockGoogleConditionalAccessDevice(ctx, tx, hostUUID, customerID); err != nil {
			return err
		}

		// A name left by a previous customer's row doesn't count; drop it.
		stmt, args, err := sqlx.In(`
			DELETE du FROM google_conditional_access_device_users du
			JOIN google_conditional_access_devices d ON d.host_uuid = du.host_uuid
			WHERE du.device_user_name IN (?) AND d.customer_id <> ?`, names, customerID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building delete of previous customer's names")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting previous customer's names")
		}

		var taken []string
		stmt, args, err = sqlx.In(`
			SELECT device_user_name FROM google_conditional_access_device_users
			WHERE device_user_name IN (?) AND host_uuid <> ?
			ORDER BY device_user_name LIMIT 1 FOR UPDATE`, names, hostUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building select of taken names")
		}
		if err := sqlx.SelectContext(ctx, tx, &taken, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting taken names")
		}
		if len(taken) > 0 {
			return ctxerr.Wrap(ctx, &fleet.GoogleConditionalAccessDeviceUserConflictError{Name: taken[0]})
		}

		// The host's other names: Google already refused the stale ones, and the rest
		// must be unmarked since the host no longer has them.
		stmt, args, err = sqlx.In(`DELETE FROM google_conditional_access_device_users
			WHERE host_uuid = ? AND device_user_name NOT IN (?) AND state = ?`,
			hostUUID, names, fleet.GoogleConditionalAccessDeviceUserStale)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building delete of stale names")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting stale names")
		}
		stmt, args, err = sqlx.In(`UPDATE google_conditional_access_device_users SET state = ?, error_message = NULL
			WHERE host_uuid = ? AND device_user_name NOT IN (?) AND state <> ?`,
			fleet.GoogleConditionalAccessDeviceUserUnmanaging, hostUUID, names, fleet.GoogleConditionalAccessDeviceUserUnmanaging)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building unmark of dropped names")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "queueing unmark of dropped names")
		}

		for _, name := range names {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO google_conditional_access_device_users (host_uuid, device_user_name, state)
				VALUES (?, ?, ?)
				ON DUPLICATE KEY UPDATE state = VALUES(state), error_message = NULL`,
				hostUUID, name, fleet.GoogleConditionalAccessDeviceUserPending); err != nil {
				return ctxerr.Wrap(ctx, err, "inserting google_conditional_access_device_users")
			}
		}

		if _, err := tx.ExecContext(ctx, `UPDATE google_conditional_access_devices SET `+resetGoogleRegistrationColumns+`
			WHERE host_uuid = ?`, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "resetting google conditional access registration attempts")
		}
		return nil
	})
}

func (ds *Datastore) UpdateGoogleConditionalAccessDeviceUserState(
	ctx context.Context, name string, fromState, toState fleet.GoogleConditionalAccessDeviceUserState, errorMessage *string,
) (bool, error) {
	res, err := ds.writer(ctx).ExecContext(ctx, `
		UPDATE google_conditional_access_device_users SET state = ?, error_message = ?
		WHERE device_user_name = ? AND state = ?`, toState, errorMessage, name, fromState)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "updating google_conditional_access_device_users state")
	}
	// Matched rows, not changed rows: the connection uses clientFoundRows.
	n, err := res.RowsAffected()
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "updated google_conditional_access_device_users state")
	}
	return n > 0, nil
}

func (ds *Datastore) QueueGoogleConditionalAccessUnmanage(ctx context.Context, hostUUIDs []string) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return queueGoogleConditionalAccessUnmanageDB(ctx, tx, hostUUIDs)
	})
}

// queueGoogleConditionalAccessUnmanageDB queues the names of the hosts for the
// UNMANAGED write. The rows are outside the host deletion cascade so the cron can still
// unmark them once the host is gone.
func queueGoogleConditionalAccessUnmanageDB(ctx context.Context, tx sqlx.ExtContext, hostUUIDs []string) error {
	if len(hostUUIDs) == 0 {
		return nil
	}
	stmts := []struct {
		query string
		args  []any
	}{
		{
			`DELETE FROM google_conditional_access_device_users WHERE host_uuid IN (?) AND state = ?`,
			[]any{hostUUIDs, fleet.GoogleConditionalAccessDeviceUserStale},
		},
		{
			`UPDATE google_conditional_access_device_users SET state = ?, error_message = NULL
			WHERE host_uuid IN (?) AND state IN (?)`,
			[]any{fleet.GoogleConditionalAccessDeviceUserUnmanaging, hostUUIDs, []fleet.GoogleConditionalAccessDeviceUserState{
				fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserManaged,
			}},
		},
		{
			`DELETE FROM google_conditional_access_devices d WHERE d.host_uuid IN (?) AND NOT EXISTS (
				SELECT 1 FROM google_conditional_access_device_users du WHERE du.host_uuid = d.host_uuid)`,
			[]any{hostUUIDs},
		},
	}
	for _, s := range stmts {
		stmt, args, err := sqlx.In(s.query, s.args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building google conditional access unmark statement")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "queueing google conditional access unmark")
		}
	}
	return nil
}

func (ds *Datastore) ListGoogleConditionalAccessDeviceUsers(
	ctx context.Context, filter fleet.GoogleConditionalAccessDeviceUserFilter,
) ([]fleet.GoogleConditionalAccessDeviceUser, error) {
	if len(filter.States) == 0 || filter.Limit <= 0 {
		return nil, ctxerr.New(ctx, "listing google device users needs states and a limit")
	}
	query := `
		SELECT du.host_uuid, du.device_user_name, du.state, du.error_message, du.updated_at
		FROM google_conditional_access_device_users du
		JOIN google_conditional_access_devices d ON d.host_uuid = du.host_uuid
		WHERE d.customer_id = ? AND du.state IN (?) AND du.device_user_name > ?`
	if filter.HostExists {
		query += ` AND EXISTS (SELECT 1 FROM hosts h WHERE h.uuid = du.host_uuid)`
	}
	query += ` ORDER BY du.device_user_name LIMIT ?`
	stmt, args, err := sqlx.In(query, filter.CustomerID, filter.States, filter.AfterName, filter.Limit)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building list of google device users")
	}
	var dus []fleet.GoogleConditionalAccessDeviceUser
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &dus, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing google device users")
	}
	return dus, nil
}

func (ds *Datastore) DeleteGoogleConditionalAccessDeviceUser(
	ctx context.Context, name string, state fleet.GoogleConditionalAccessDeviceUserState, deleteEmptyDevice bool,
) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var hostUUID string
		if err := sqlx.GetContext(ctx, tx, &hostUUID, `
			SELECT host_uuid FROM google_conditional_access_device_users
			WHERE device_user_name = ? AND state = ? FOR UPDATE`, name, state); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return ctxerr.Wrap(ctx, err, "getting google device user")
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM google_conditional_access_device_users WHERE device_user_name = ?`, name); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting google device user")
		}
		if !deleteEmptyDevice {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM google_conditional_access_devices d WHERE d.host_uuid = ? AND NOT EXISTS (
				SELECT 1 FROM google_conditional_access_device_users du WHERE du.host_uuid = d.host_uuid)`, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting empty google_conditional_access_devices row")
		}
		return nil
	})
}

func (ds *Datastore) ListGoogleConditionalAccessBookmarkHosts(
	ctx context.Context, customerID, afterUUID string, limit int,
) ([]fleet.GoogleConditionalAccessBookmarkHost, error) {
	var hosts []fleet.GoogleConditionalAccessBookmarkHost
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, `
		SELECT h.uuid AS host_uuid, g.google_device_id
		FROM host_google_device_ids g
		JOIN hosts h ON h.id = g.host_id
		-- Company-owned hosts only ("On (automatic)"), as for a sign-in.
		JOIN host_mdm hm ON hm.host_id = h.id AND hm.enrolled = 1 AND hm.installed_from_dep = 1
		WHERE h.uuid > ? AND NOT EXISTS (
			SELECT 1 FROM google_conditional_access_device_users du
			JOIN google_conditional_access_devices d ON d.host_uuid = du.host_uuid
			WHERE du.host_uuid = h.uuid AND d.customer_id = ?)
		ORDER BY h.uuid
		LIMIT ?`, afterUUID, customerID, limit,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing bookmark-era google hosts")
	}
	return hosts, nil
}

// queueGoogleConditionalAccessUnmanageForDeletedHostsDB skips UUIDs another host still
// has, such as a dual-boot install, since the device is still in Fleet.
func queueGoogleConditionalAccessUnmanageForDeletedHostsDB(ctx context.Context, tx sqlx.ExtContext, hostUUIDs []string) error {
	if len(hostUUIDs) == 0 {
		return nil
	}
	stmt, args, err := sqlx.In(`SELECT DISTINCT uuid FROM hosts WHERE uuid IN (?)`, hostUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building select of remaining host uuids")
	}
	var remaining []string
	if err := sqlx.SelectContext(ctx, tx, &remaining, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting remaining host uuids")
	}
	gone := slices.DeleteFunc(slices.Clone(hostUUIDs), func(u string) bool {
		return u == "" || slices.Contains(remaining, u)
	})
	return queueGoogleConditionalAccessUnmanageDB(ctx, tx, gone)
}
