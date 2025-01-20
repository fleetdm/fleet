package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/micromdm/nanolib/log"
)

func enqueue(ctx context.Context, tx sqlx.ExtContext, ids []string, cmd *mdm.CommandWithSubtype) error {
	if len(ids) < 1 {
		return errors.New("no id(s) supplied to queue command to")
	}
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO nano_commands (command_uuid, request_type, command, subtype) VALUES (?, ?, ?, ?)`,
		cmd.CommandUUID, cmd.Command.Command.RequestType, cmd.Raw, cmd.Subtype,
	)
	if err != nil {
		return err
	}
	const mySQLPlaceholderLimit = 65536 - 1
	const placeholdersPerInsert = 2
	const batchSize = mySQLPlaceholderLimit / placeholdersPerInsert
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		idsBatch := ids[i:end]

		// Process batch
		query := `INSERT INTO nano_enrollment_queue (id, command_uuid) VALUES (?, ?)`
		query += strings.Repeat(", (?, ?)", len(idsBatch)-1)
		args := make([]interface{}, len(idsBatch)*placeholdersPerInsert)
		for i, id := range idsBatch {
			args[i*2] = id
			args[i*2+1] = cmd.CommandUUID
		}
		_, err = tx.ExecContext(ctx, query+";", args...)
		if err != nil {
			return err
		}
	}
	return nil
}

type loggerWrapper struct {
	logger log.Logger
}

func (l loggerWrapper) Log(keyvals ...interface{}) error {
	l.logger.Info(keyvals...)
	return nil
}

func (m *MySQLStorage) EnqueueCommand(ctx context.Context, ids []string, cmd *mdm.CommandWithSubtype) (map[string]error,
	error) {
	// We need to retry because this transaction may deadlock with updates to nano_enrollment.last_seen_at
	// Deadlock seen in 2024/12/12 loadtest: https://docs.google.com/document/d/1-Q6qFTd7CDm-lh7MVRgpNlNNJijk6JZ4KO49R1fp80U
	err := common_mysql.WithRetryTxx(ctx, sqlx.NewDb(m.db, ""), func(tx sqlx.ExtContext) error {
		return enqueue(ctx, tx, ids, cmd)
	}, loggerWrapper{m.logger})
	return nil, err
}

func (m *MySQLStorage) deleteCommand(ctx context.Context, tx *sql.Tx, id, uuid string) error {
	// first, place a record lock on the command so that multiple devices
	// trying to each delete it do not race
	_, err := tx.ExecContext(
		ctx, `
SELECT command_uuid FROM nano_commands WHERE command_uuid = ? FOR UPDATE;
`,
		uuid,
	)
	if err != nil {
		return err
	}
	// delete command result (i.e. NotNows) and this queued command
	_, err = tx.ExecContext(
		ctx, `
DELETE
    q, r
FROM
    nano_enrollment_queue AS q
    LEFT JOIN nano_command_results AS r
        ON q.command_uuid = r.command_uuid AND r.id = q.id
WHERE
    q.id = ? AND q.command_uuid = ?;
`,
		id, uuid,
	)
	if err != nil {
		return err
	}
	// now delete the actual command if no enrollments have it queued
	// nor are there any results for it.
	_, err = tx.ExecContext(
		ctx, `
DELETE
    c
FROM
    nano_commands AS c
    LEFT JOIN nano_enrollment_queue AS q
        ON q.command_uuid = c.command_uuid
    LEFT JOIN nano_command_results AS r
        ON r.command_uuid = c.command_uuid
WHERE
    c.command_uuid = ? AND
    q.command_uuid IS NULL AND
    r.command_uuid IS NULL;
`,
		uuid,
	)
	return err
}

func (m *MySQLStorage) deleteCommandTx(r *mdm.Request, result *mdm.CommandResults) error {
	tx, err := m.db.BeginTx(r.Context, nil)
	if err != nil {
		return err
	}
	if err = m.deleteCommand(r.Context, tx, r.ID, result.CommandUUID); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback error: %w; while trying to handle error: %v", rbErr, err)
		}
		return err
	}
	return tx.Commit()
}

// TODO(uniq): this is where we could activate the next activity on VPP app
// install results in a transaction.
func (m *MySQLStorage) StoreCommandReport(r *mdm.Request, result *mdm.CommandResults) error {
	if err := m.updateLastSeen(r); err != nil {
		return err
	}
	if result.Status == "Idle" {
		return nil
	}

	// ensure there's a matching command
	matchingRow := m.db.QueryRowContext(
		r.Context,
		`SELECT 1 FROM nano_commands WHERE command_uuid = ?`,
		result.CommandUUID,
	)
	var matchingCount int
	if err := matchingRow.Scan(&matchingCount); err != nil {
		return err
	}
	// this should be already handed by the error value in Scan above, but
	// just to be safe
	if matchingCount == 0 {
		return sql.ErrNoRows
	}

	if m.rm && result.Status != "NotNow" {
		return m.deleteCommandTx(r, result)
	}
	notNowConstants := "NULL, 0"
	notNowBumpTallySQL := ""
	// note that due to the ON DUPLICATE KEY we don't UPDATE the
	// not_now_at field. thus it will only represent the first NotNow.
	if result.Status == "NotNow" {
		notNowConstants = "CURRENT_TIMESTAMP, 1"
		notNowBumpTallySQL = `, nano_command_results.not_now_tally = nano_command_results.not_now_tally + 1`
	}
	_, err := m.db.ExecContext(
		//nolint:gosec
		r.Context, `
INSERT INTO nano_command_results
    (id, command_uuid, status, result, not_now_at, not_now_tally)
VALUES
    (?, ?, ?, ?, `+notNowConstants+`)
ON DUPLICATE KEY
UPDATE
    status = VALUES(status),
    result = VALUES(result)`+notNowBumpTallySQL+`;`,
		r.ID,
		result.CommandUUID,
		result.Status,
		result.Raw,
	)
	return err
}

func (m *MySQLStorage) RetrieveNextCommand(r *mdm.Request, skipNotNow bool) (*mdm.CommandWithSubtype, error) {
	command := new(mdm.CommandWithSubtype)
	id := "?"
	var args []interface{}
	// This performance optimization eliminates the prepare statement for this frequent query for macOS devices.
	// For macOS devices, UDID is a UUID, so we can validate it and use it directly in the query.
	if err := uuid.Validate(r.ID); err == nil {
		id = "'" + r.ID + "'"
	} else {
		// iOS devices have a UDID that is not a valid UUID.
		// User enrollments have their own identifier, which is not a UUID.
		// We use a prepared statement for these cases to avoid SQL injection.
		args = append(args, r.ID)
	}
	err := m.reader(r.Context).QueryRowxContext(
		r.Context, fmt.Sprintf(
			// The query should use the ANTIJOIN (NOT EXISTS) optimization on the nano_command_results table.
			`
SELECT c.command_uuid, c.request_type, c.command, c.subtype
FROM nano_enrollment_queue AS q
    INNER JOIN nano_commands AS c
        ON q.command_uuid = c.command_uuid
    LEFT JOIN nano_command_results r
        ON r.command_uuid = q.command_uuid AND r.id = q.id AND (r.status != 'NotNow' OR %t)
WHERE q.id = %s
    AND q.active = 1
    AND r.status IS NULL
ORDER BY
    q.priority DESC,
    q.created_at
LIMIT 1;`, skipNotNow, id), args...,
	).Scan(&command.CommandUUID, &command.Command.Command.RequestType, &command.Raw, &command.Subtype)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return command, nil
}

func (m *MySQLStorage) ClearQueue(r *mdm.Request) error {
	if r.ParentID != "" {
		return errors.New("can only clear a device channel queue")
	}
	// Because we're joining on and WHERE-ing by the enrollments table
	// this will clear (mark inactive) the queue of not only this
	// device ID, but all user-channel enrollments with a 'parent' ID of
	// this device, too.
	_, err := m.db.ExecContext(
		r.Context,
		`
UPDATE
    nano_enrollment_queue AS q
    INNER JOIN nano_enrollments AS e
        ON q.id = e.id
    INNER JOIN nano_commands AS c
        ON q.command_uuid = c.command_uuid
    LEFT JOIN nano_command_results r
        ON r.command_uuid = q.command_uuid AND r.id = q.id
SET
    q.active = 0
WHERE
    e.device_id = ? AND
    active = 1 AND
    (r.status IS NULL OR r.status = 'NotNow');`,
		r.ID,
	)
	return err
}

// BulkDeleteHostUserCommandsWithoutResults deletes all commands without results for the given host/user IDs.
// This is used to clean up the queue when a profile is deleted from Fleet.
func (m *MySQLStorage) BulkDeleteHostUserCommandsWithoutResults(ctx context.Context, commandToIDs map[string][]string) error {
	if len(commandToIDs) == 0 {
		return nil
	}
	return common_mysql.WithRetryTxx(ctx, sqlx.NewDb(m.db, ""), func(tx sqlx.ExtContext) error {
		return m.bulkDeleteHostUserCommandsWithoutResults(ctx, tx, commandToIDs)
	}, loggerWrapper{m.logger})
}

func (m *MySQLStorage) bulkDeleteHostUserCommandsWithoutResults(ctx context.Context, tx sqlx.ExtContext,
	commandToIDs map[string][]string) error {
	stmt := `
DELETE
    eq
FROM
    nano_enrollment_queue AS eq
	LEFT JOIN nano_command_results AS cr
		ON cr.command_uuid = eq.command_uuid AND cr.id = eq.id
WHERE
	cr.command_uuid IS NULL AND eq.command_uuid = ? AND eq.id IN (?);`

	// We process each commandUUID one at a time, in batches of hostUserIDs.
	// This is because the number of hostUserIDs can be large, and number of unique commands is normally small.
	// If we have a use case where each host has a unique command, we can create a separate method for that use case.
	for commandUUID, hostUserIDs := range commandToIDs {
		if len(hostUserIDs) == 0 {
			continue
		}

		batchSize := 10000
		err := common_mysql.BatchProcessSimple(hostUserIDs, batchSize, func(hostUserIDsToProcess []string) error {
			expanded, args, err := sqlx.In(stmt, commandUUID, hostUserIDsToProcess)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "expanding bulk delete nano commands")
			}
			_, err = tx.ExecContext(ctx, expanded, args...)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "bulk delete nano commands")
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}
