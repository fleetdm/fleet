package mysql

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// googleUnmarkBatchSize bounds the IN lists of the unmark queue. At a few thousand
// UUIDs MySQL switches to scanning the tables, which locks every registered host's rows.
const googleUnmarkBatchSize = 500

const (
	resetGoogleRegistrationColumns = `registration_token_hash = NULL, registration_token_expires_at = NULL,
		registration_attempts = 0, first_registration_attempt_at = NULL`
	// A row with a device ID and no names was migrated from the bookmark flow, and stays
	// until the ID is deleted.
	deleteEmptyGoogleDevicesStmt = `DELETE FROM google_conditional_access_devices d
		WHERE d.host_uuid IN (?) AND d.google_device_id IS NULL AND NOT EXISTS (
		SELECT 1 FROM google_conditional_access_device_users du WHERE du.host_uuid = d.host_uuid)`
)

type googleDeviceRow struct {
	CustomerID           string `db:"customer_id"`
	RegistrationAttempts uint   `db:"registration_attempts"`
	InWindow             bool   `db:"in_window"`
}

func (ds *Datastore) GetGoogleConditionalAccessIntegration(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, error) {
	var integration fleet.ConditionalAccessGoogleIntegration
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &integration, `
		SELECT proxy_server_secret, state, error_message
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
			(id, proxy_server_secret, state, error_message)
		VALUES (1, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			proxy_server_secret = VALUES(proxy_server_secret),
			state = VALUES(state),
			error_message = VALUES(error_message)`,
		integration.ProxyServerSecret, integration.State, integration.ErrorMessage,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "saving google_conditional_access_integrations")
	}
	return nil
}

func (ds *Datastore) DeleteGoogleConditionalAccessDevicesExcept(ctx context.Context, customerID string, limit int) (int, error) {
	// The DeviceUser rows go with the device rows through the foreign key. A row with no
	// customer was migrated from the bookmark flow and isn't another customer's.
	res, err := ds.writer(ctx).ExecContext(ctx,
		`DELETE FROM google_conditional_access_devices WHERE customer_id NOT IN (?, '') LIMIT ?`, customerID, limit)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "deleting google_conditional_access_devices")
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "deleted google_conditional_access_devices")
	}
	return int(n), nil
}

func (ds *Datastore) GetGoogleConditionalAccessDevice(ctx context.Context, hostUUID, customerID string) (*fleet.GoogleConditionalAccessDevice, error) {
	// One row per name, or a single row without one for a device with no names.
	var rows []struct {
		fleet.GoogleConditionalAccessDevice
		Name         *string                                       `db:"device_user_name"`
		State        *fleet.GoogleConditionalAccessDeviceUserState `db:"state"`
		ErrorMessage *string                                       `db:"error_message"`
		UpdatedAt    *time.Time                                    `db:"updated_at"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, `
		SELECT d.host_uuid, du.device_user_name, du.state, du.error_message, du.updated_at
		FROM google_conditional_access_devices d
		LEFT JOIN google_conditional_access_device_users du ON du.host_uuid = d.host_uuid
		WHERE d.host_uuid = ? AND d.customer_id = ?
		ORDER BY du.device_user_name`, hostUUID, customerID,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting google_conditional_access_devices")
	}
	if len(rows) == 0 {
		return nil, ctxerr.Wrap(ctx, notFound("GoogleConditionalAccessDevice").WithName(hostUUID))
	}
	dev := rows[0].GoogleConditionalAccessDevice
	for _, r := range rows {
		if r.Name != nil {
			dev.DeviceUsers = append(dev.DeviceUsers, fleet.GoogleConditionalAccessDeviceUser{
				HostUUID: dev.HostUUID, Name: *r.Name, State: *r.State, ErrorMessage: r.ErrorMessage, UpdatedAt: *r.UpdatedAt,
			})
		}
	}
	return &dev, nil
}

// claimGoogleConditionalAccessDevice locks the host's row and makes it belong to the
// customer: it creates it, or takes over another customer's row and drops its names.
// It returns the row as the call leaves it, so a created or taken-over row is empty.
// An empty customerID matches every customer, as it does for reads, so it takes no row over.
func claimGoogleConditionalAccessDevice(ctx context.Context, tx sqlx.ExtContext, hostUUID, customerID string) (googleDeviceRow, error) {
	// A locking read of a missing row takes a gap lock, and two first registrations
	// holding the same gap deadlock on their inserts, so the row is created before it is
	// read. The no-op update locks an existing row like FOR UPDATE does, and leaves
	// updated_at alone since no column changes.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO google_conditional_access_devices (host_uuid, customer_id) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE host_uuid = host_uuid`, hostUUID, customerID); err != nil {
		return googleDeviceRow{}, ctxerr.Wrap(ctx, err, "inserting google_conditional_access_devices")
	}
	var row googleDeviceRow
	if err := sqlx.GetContext(ctx, tx, &row, `
		SELECT customer_id, registration_attempts,
			first_registration_attempt_at IS NOT NULL AND first_registration_attempt_at > NOW(6) - INTERVAL ? SECOND AS in_window
		FROM google_conditional_access_devices
		WHERE host_uuid = ?
		FOR UPDATE`, int(fleet.GoogleConditionalAccessRegistrationWindow.Seconds()), hostUUID); err != nil {
		return googleDeviceRow{}, ctxerr.Wrap(ctx, err, "locking google_conditional_access_devices")
	}
	if row.CustomerID == customerID || customerID == "" {
		return row, nil
	}

	// The names belong to the previous customer's Workspace, which Fleet no longer
	// has a key for, so they are dropped without an unmark.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM google_conditional_access_device_users WHERE host_uuid = ?`, hostUUID); err != nil {
		return googleDeviceRow{}, ctxerr.Wrap(ctx, err, "deleting previous customer's google_conditional_access_device_users")
	}
	// A row with no customer was migrated from the bookmark flow, so its device ID is
	// this customer's. The ID is assigned before customer_id, which it reads.
	if _, err := tx.ExecContext(ctx, `UPDATE google_conditional_access_devices
		SET google_device_id = IF(customer_id = '', google_device_id, NULL), customer_id = ?, `+resetGoogleRegistrationColumns+`
		WHERE host_uuid = ?`, customerID, hostUUID); err != nil {
		return googleDeviceRow{}, ctxerr.Wrap(ctx, err, "taking over google_conditional_access_devices")
	}
	return googleDeviceRow{CustomerID: customerID}, nil
}

func (ds *Datastore) StartGoogleConditionalAccessRegistration(
	ctx context.Context, hostUUID, customerID string, tokenHash []byte, tokenTTL time.Duration,
) (bool, error) {
	var started bool
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		started = false
		row, err := claimGoogleConditionalAccessDevice(ctx, tx, hostUUID, customerID)
		if err != nil {
			return err
		}
		if row.InWindow && row.RegistrationAttempts >= fleet.GoogleConditionalAccessMaxRegistrationAttempts {
			return nil
		}

		// The token expires by the database's clock, the one that checks it when it is consumed.
		if _, err := tx.ExecContext(ctx, `
			UPDATE google_conditional_access_devices
			SET registration_token_hash = ?, registration_token_expires_at = NOW(6) + INTERVAL ? MICROSECOND,
				registration_attempts = IF(?, registration_attempts + 1, 1),
				first_registration_attempt_at = IF(?, first_registration_attempt_at, NOW(6))
			WHERE host_uuid = ?`, tokenHash, tokenTTL.Microseconds(), row.InWindow, row.InWindow, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "counting google conditional access registration attempt")
		}
		started = true
		return nil
	})
	return started, err
}

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
		return resetGoogleConditionalAccessRegistrationDB(ctx, tx, hostUUID)
	})
	if err != nil {
		return "", err
	}
	return hostUUID, nil
}

func resetGoogleConditionalAccessRegistrationDB(ctx context.Context, tx sqlx.ExtContext, hostUUID string) error {
	if _, err := tx.ExecContext(ctx, `UPDATE google_conditional_access_devices SET `+resetGoogleRegistrationColumns+`
		WHERE host_uuid = ?`, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "resetting google conditional access registration attempts")
	}
	return nil
}

func (ds *Datastore) RecordGoogleConditionalAccessRegistration(ctx context.Context, hostUUID, customerID string, names []string) error {
	if len(names) == 0 {
		return ctxerr.New(ctx, "no google device user names to record")
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if _, err := claimGoogleConditionalAccessDevice(ctx, tx, hostUUID, customerID); err != nil {
			return err
		}
		if err := recordGoogleDeviceUsersDB(ctx, tx, hostUUID, customerID, names); err != nil {
			return err
		}

		var stored *string
		if err := sqlx.GetContext(ctx, tx, &stored,
			`SELECT google_device_id FROM google_conditional_access_devices WHERE host_uuid = ?`, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "reading host google device id")
		}
		var current string
		if stored != nil {
			current = *stored
		}
		if id, from, kept := googleDeviceIDForNames(current, names); !kept && id != nil {
			if err := bindGoogleDeviceIDDB(ctx, tx, hostUUID, customerID, *id); err != nil {
				if IsDuplicate(err) {
					return ctxerr.Wrap(ctx, &fleet.GoogleConditionalAccessDeviceUserConflictError{Name: from})
				}
				return ctxerr.Wrap(ctx, err, "setting host google device id")
			}
		}
		return resetGoogleConditionalAccessRegistrationDB(ctx, tx, hostUUID)
	})
}

// recordGoogleDeviceUsersDB stores the names on the host as pending, and queues its
// other names for unmark or deletes them when stale. The caller has claimed the host's
// row for the customer.
func recordGoogleDeviceUsersDB(ctx context.Context, tx sqlx.ExtContext, hostUUID, customerID string, names []string) error {
	if customerID == "" {
		return ctxerr.New(ctx, "google device user names need a customer")
	}

	// A locking statement that matches no row takes gap locks, and two first registrations
	// holding the same gap deadlock on their inserts. Plain reads find the rows the locking
	// statements name instead, and the names are checked again under lock once inserted.
	var elsewhere []struct {
		Name             string `db:"device_user_name"`
		PreviousCustomer bool   `db:"previous_customer"`
	}
	stmt, args, err := sqlx.In(`
		SELECT du.device_user_name, d.customer_id <> ? AS previous_customer
		FROM google_conditional_access_device_users du
		JOIN google_conditional_access_devices d ON d.host_uuid = du.host_uuid
		WHERE du.device_user_name IN (?) AND du.host_uuid <> ?
		ORDER BY du.device_user_name`, customerID, names, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building select of names stored on other hosts")
	}
	if err := sqlx.SelectContext(ctx, tx, &elsewhere, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting names stored on other hosts")
	}
	var previousCustomers []string
	for _, e := range elsewhere {
		if !e.PreviousCustomer {
			return ctxerr.Wrap(ctx, &fleet.GoogleConditionalAccessDeviceUserConflictError{Name: e.Name})
		}
		previousCustomers = append(previousCustomers, e.Name)
	}
	if len(previousCustomers) > 0 {
		// A name left on another customer's row doesn't count; drop it.
		if err := googleExecIn(ctx, tx, "deleting previous customer's names", `
			DELETE du FROM google_conditional_access_device_users du
			JOIN google_conditional_access_devices d ON d.host_uuid = du.host_uuid
			WHERE du.device_user_name IN (?) AND d.customer_id <> ?`, previousCustomers, customerID); err != nil {
			return err
		}
	}

	// The host's other names: Google already refused the stale ones, and the rest
	// must be unmarked since the host no longer has them.
	var others []struct {
		Name  string                                       `db:"device_user_name"`
		State fleet.GoogleConditionalAccessDeviceUserState `db:"state"`
	}
	stmt, args, err = sqlx.In(`
		SELECT device_user_name, state FROM google_conditional_access_device_users
		WHERE host_uuid = ? AND device_user_name NOT IN (?)`, hostUUID, names)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building select of the host's other names")
	}
	if err := sqlx.SelectContext(ctx, tx, &others, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting the host's other names")
	}
	var stale, dropped []string
	for _, o := range others {
		switch o.State {
		case fleet.GoogleConditionalAccessDeviceUserStale:
			stale = append(stale, o.Name)
		case fleet.GoogleConditionalAccessDeviceUserUnmanaging:
			// Already queued.
		default:
			dropped = append(dropped, o.Name)
		}
	}
	if len(stale) > 0 {
		if err := googleExecIn(ctx, tx, "deleting stale names", `
			DELETE FROM google_conditional_access_device_users
			WHERE device_user_name IN (?) AND host_uuid = ? AND state = ?`,
			stale, hostUUID, fleet.GoogleConditionalAccessDeviceUserStale); err != nil {
			return err
		}
	}
	if len(dropped) > 0 {
		if err := googleExecIn(ctx, tx, "queueing unmark of dropped names", `
			UPDATE google_conditional_access_device_users SET state = ?, error_message = NULL
			WHERE device_user_name IN (?) AND host_uuid = ? AND state <> ?`,
			fleet.GoogleConditionalAccessDeviceUserUnmanaging, dropped, hostUUID, fleet.GoogleConditionalAccessDeviceUserUnmanaging); err != nil {
			return err
		}
	}

	args = make([]any, 0, 3*len(names))
	for _, name := range names {
		args = append(args, hostUUID, name, fleet.GoogleConditionalAccessDeviceUserPending)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO google_conditional_access_device_users (host_uuid, device_user_name, state)
		VALUES `+strings.TrimSuffix(strings.Repeat("(?, ?, ?),", len(names)), ",")+`
		ON DUPLICATE KEY UPDATE state = VALUES(state), error_message = NULL`, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting google_conditional_access_device_users")
	}

	// The insert leaves a name with the host that stored it since the reads above, so a
	// name that isn't this host's now is taken.
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
	return nil
}

func (ds *Datastore) UpdateGoogleConditionalAccessDeviceUserState(
	ctx context.Context, name string, fromState, toState fleet.GoogleConditionalAccessDeviceUserState, errorMessage *string,
) (bool, error) {
	res, err := ds.writer(ctx).ExecContext(ctx, `
		UPDATE google_conditional_access_device_users SET state = ?, error_message = ?, updated_at = NOW(6)
		WHERE device_user_name = ? AND state = ?`, toState, errorMessage, name, fromState)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "updating google_conditional_access_device_users state")
	}
	// Matched rows, not changed rows: the connection uses clientFoundRows. updated_at is set
	// explicitly so a repeated failure with the same message still counts for the backoff.
	n, err := res.RowsAffected()
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "updated google_conditional_access_device_users state")
	}
	return n > 0, nil
}

func (ds *Datastore) QueueGoogleConditionalAccessUnmarkOfUnenrolledHosts(ctx context.Context, customerID string, limit int) (int, error) {
	var queued int
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var hostUUIDs []string
		if err := sqlx.SelectContext(ctx, tx, &hostUUIDs, `
			SELECT DISTINCT du.host_uuid
			FROM google_conditional_access_device_users du
			JOIN google_conditional_access_devices d ON d.host_uuid = du.host_uuid
			WHERE d.customer_id = ? AND du.state IN (?, ?) AND NOT EXISTS (
				SELECT 1 FROM hosts h
				JOIN host_mdm hm ON hm.host_id = h.id AND hm.enrolled = 1
				WHERE h.uuid = du.host_uuid)
			ORDER BY du.host_uuid
			LIMIT ?`, customerID,
			fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserManaged, limit,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "listing unenrolled hosts registered with google")
		}
		queued = len(hostUUIDs)
		return queueGoogleConditionalAccessUnmarkDB(ctx, tx, hostUUIDs)
	})
	return queued, err
}

// queueGoogleConditionalAccessUnmarkDB queues the names of the hosts for the
// UNMANAGED write. The rows are outside the host deletion cascade so the cron can still
// unmark them once the host is gone. Every write that marks a mobile host unenrolled or
// deletes it (MDMTurnOff, SetAndroidHostUnenrolled, BulkSetAndroidHostsUnenrolled,
// deleteHosts) calls this in its own transaction. A registration that commits after
// that is left to QueueGoogleConditionalAccessUnmarkOfUnenrolledHosts.
func queueGoogleConditionalAccessUnmarkDB(ctx context.Context, tx sqlx.ExtContext, hostUUIDs []string) error {
	// A host that never enrolled has no UUID, and no row.
	hostUUIDs = slices.DeleteFunc(hostUUIDs, func(s string) bool { return s == "" })
	for batch := range slices.Chunk(hostUUIDs, googleUnmarkBatchSize) {
		// A plain read finds the registered hosts without locking anything for the others,
		// which is most hosts that check out.
		stmt, args, err := sqlx.In(`SELECT host_uuid FROM google_conditional_access_devices WHERE host_uuid IN (?)`, batch)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building select of registered google hosts")
		}
		var registered []string
		if err := sqlx.SelectContext(ctx, tx, &registered, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting registered google hosts")
		}
		if len(registered) == 0 {
			continue
		}
		// The device rows are locked before their names, the order registration takes
		// them in, so the two can't deadlock.
		if err := googleExecIn(ctx, tx, "locking google_conditional_access_devices",
			`SELECT host_uuid FROM google_conditional_access_devices WHERE host_uuid IN (?) ORDER BY host_uuid FOR UPDATE`,
			registered); err != nil {
			return err
		}
		if err := googleExecIn(ctx, tx, "deleting stale names",
			`DELETE FROM google_conditional_access_device_users WHERE host_uuid IN (?) AND state = ?`,
			registered, fleet.GoogleConditionalAccessDeviceUserStale); err != nil {
			return err
		}
		if err := googleExecIn(ctx, tx, "queueing google conditional access unmark", `
			UPDATE google_conditional_access_device_users SET state = ?, error_message = NULL
			WHERE host_uuid IN (?) AND state IN (?)`,
			fleet.GoogleConditionalAccessDeviceUserUnmanaging, registered, []fleet.GoogleConditionalAccessDeviceUserState{
				fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserManaged,
			}); err != nil {
			return err
		}
		if err := googleExecIn(ctx, tx, "clearing google device ids",
			`UPDATE google_conditional_access_devices SET google_device_id = NULL WHERE host_uuid IN (?) AND google_device_id IS NOT NULL`,
			registered); err != nil {
			return err
		}
		if err := googleExecIn(ctx, tx, "deleting empty google_conditional_access_devices rows", deleteEmptyGoogleDevicesStmt, registered); err != nil {
			return err
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
			SELECT host_uuid FROM google_conditional_access_device_users WHERE device_user_name = ?`, name); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return ctxerr.Wrap(ctx, err, "getting google device user")
		}
		// Lock the device row before the name, the order registration takes them in.
		var locked bool
		if err := sqlx.GetContext(ctx, tx, &locked, `
			SELECT 1 FROM google_conditional_access_devices WHERE host_uuid = ? FOR UPDATE`, hostUUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil // the cascade deleted the name with the row
			}
			return ctxerr.Wrap(ctx, err, "locking google_conditional_access_devices")
		}
		res, err := tx.ExecContext(ctx, `
			DELETE FROM google_conditional_access_device_users WHERE device_user_name = ? AND host_uuid = ? AND state = ?`,
			name, hostUUID, state)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting google device user")
		}
		n, err := res.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleted google device user")
		}
		if n == 0 {
			return nil // another transaction changed or deleted the name
		}
		if err := repointGoogleDeviceIDDB(ctx, tx, hostUUID); err != nil {
			return err
		}
		if !deleteEmptyDevice {
			return nil
		}
		return googleExecIn(ctx, tx, "deleting empty google_conditional_access_devices row", deleteEmptyGoogleDevicesStmt, []string{hostUUID})
	})
}

func (ds *Datastore) ListGoogleConditionalAccessBookmarkHosts(
	ctx context.Context, customerID string, afterHostID uint, limit int,
) ([]fleet.GoogleConditionalAccessBookmarkHost, error) {
	var hosts []fleet.GoogleConditionalAccessBookmarkHost
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, `
		SELECT h.id AS host_id, d.host_uuid, d.google_device_id
		FROM google_conditional_access_devices d
		JOIN hosts h ON h.uuid = d.host_uuid
		-- Company-owned hosts only ("On (automatic)"), as for a sign-in.
		JOIN host_mdm hm ON hm.host_id = h.id AND hm.enrolled = 1 AND hm.installed_from_dep = 1
		WHERE d.google_device_id IS NOT NULL AND d.customer_id IN ('', ?) AND h.id > ? AND NOT EXISTS (
			SELECT 1 FROM google_conditional_access_device_users du WHERE du.host_uuid = d.host_uuid)
		ORDER BY h.id
		LIMIT ?`, customerID, afterHostID, limit,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing bookmark-era google hosts")
	}
	return hosts, nil
}

func (ds *Datastore) ListGoogleConditionalAccessUnresolvedAndroidHosts(
	ctx context.Context, customerID string, maxAttempts, limit int,
) ([]fleet.GoogleConditionalAccessAndroidHost, error) {
	var hosts []fleet.GoogleConditionalAccessAndroidHost
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, `
		SELECT h.id AS host_id, h.uuid AS host_uuid, ad.device_id AS android_device_id
		FROM android_devices ad
		JOIN hosts h ON h.id = ad.host_id
		-- Company-owned hosts only ("On (automatic)"), as for a sign-in.
		JOIN host_mdm hm ON hm.host_id = h.id AND hm.enrolled = 1 AND hm.installed_from_dep = 1
		-- Another customer's row counts as none: its names and attempts go when the host is claimed.
		LEFT JOIN google_conditional_access_devices d ON d.host_uuid = h.uuid AND d.customer_id = ?
		WHERE (? <= 0 OR COALESCE(d.registration_attempts, 0) < ?) AND NOT EXISTS (
			SELECT 1 FROM google_conditional_access_device_users du
			WHERE du.host_uuid = d.host_uuid AND du.state IN (?, ?))
		ORDER BY COALESCE(d.registration_attempts, 0), h.id
		LIMIT ?`, customerID, maxAttempts, maxAttempts,
		fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserManaged, limit,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing unresolved android google hosts")
	}
	return hosts, nil
}

func (ds *Datastore) CountGoogleConditionalAccessResolutionAttempt(ctx context.Context, hostUUID, customerID string) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if _, err := claimGoogleConditionalAccessDevice(ctx, tx, hostUUID, customerID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE google_conditional_access_devices
			SET registration_attempts = registration_attempts + 1,
				first_registration_attempt_at = COALESCE(first_registration_attempt_at, NOW(6))
			WHERE host_uuid = ?`, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "counting google conditional access resolution attempt")
		}
		return nil
	})
}

// googleExecIn expands the IN lists of query with sqlx.In and runs it. Both failures
// are wrapped with what alone: sqlx.In only fails on a programming error.
func googleExecIn(ctx context.Context, tx sqlx.ExtContext, what, query string, inArgs ...any) error {
	stmt, args, err := sqlx.In(query, inArgs...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, what)
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, what)
	}
	return nil
}
