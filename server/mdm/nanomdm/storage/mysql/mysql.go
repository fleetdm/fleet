// Package mysql stores and retrieves MDM data from MySQL
package mysql

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// Schema holds the schema for the NanoMDM MySQL storage.
//
//go:embed schema.sql
var Schema string

var ErrNoCert = errors.New("no certificate in MDM Request")

type MySQLStorage struct {
	logger        *slog.Logger
	db            *sql.DB
	rm            bool
	asyncLastSeen *asyncLastSeen
	reader        func(ctx context.Context) fleet.DBReader
}

type config struct {
	driver        string
	dsn           string
	db            *sql.DB
	logger        *slog.Logger
	rm            bool
	asyncCap      int
	asyncInterval time.Duration
	reader        func(ctx context.Context) fleet.DBReader
}

type Option func(*config)

func WithReaderFunc(readerFunc func(ctx context.Context) fleet.DBReader) Option {
	return func(c *config) {
		c.reader = readerFunc
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

func WithDSN(dsn string) Option {
	return func(c *config) {
		c.dsn = dsn
	}
}

func WithDriver(driver string) Option {
	return func(c *config) {
		c.driver = driver
	}
}

func WithDB(db *sql.DB) Option {
	return func(c *config) {
		c.db = db
	}
}

func WithDeleteCommands() Option {
	return func(c *config) {
		c.rm = true
	}
}

func WithAsyncLastSeen(capacity int, interval time.Duration) Option {
	return func(c *config) {
		c.asyncCap = capacity
		c.asyncInterval = interval
	}
}

func New(opts ...Option) (*MySQLStorage, error) {
	const (
		asyncLastSeenFlushInterval = 2 * time.Second
		asyncLastSeenCap           = 1000
	)

	cfg := &config{logger: slog.New(slog.DiscardHandler), driver: "mysql", asyncCap: asyncLastSeenCap, asyncInterval: asyncLastSeenFlushInterval}
	for _, opt := range opts {
		opt(cfg)
	}
	var err error
	if cfg.db == nil {
		cfg.db, err = sql.Open(cfg.driver, cfg.dsn)
		if err != nil {
			return nil, err
		}
	}
	if err = cfg.db.Ping(); err != nil {
		return nil, err
	}

	mysqlStore := &MySQLStorage{db: cfg.db, logger: cfg.logger, rm: cfg.rm}
	if cfg.reader == nil {
		mysqlStore.reader = func(ctx context.Context) fleet.DBReader {
			return sqlx.NewDb(mysqlStore.db, "")
		}
	} else {
		mysqlStore.reader = cfg.reader
	}

	if v := os.Getenv("FLEET_DISABLE_ASYNC_NANO_LAST_SEEN"); v != "1" {
		asyncLastSeen := newAsyncLastSeen(cfg.asyncInterval, cfg.asyncCap, mysqlStore.updateLastSeenBatch)
		mysqlStore.asyncLastSeen = asyncLastSeen

		go asyncLastSeen.runFlushLoop(context.Background())
	}

	return mysqlStore, nil
}

// nullEmptyString returns a NULL string if s is empty.
func nullEmptyString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

func (s *MySQLStorage) StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error {
	var pemCert []byte
	if r.Certificate != nil {
		pemCert = cryptoutil.PEMCertificate(r.Certificate.Raw)
	}
	// When a device undergoes SCEP certificate renewal, it sends a new
	// Authenticate message. We must preserve the existing bootstrap token
	// during renewal; clearing it causes commands that depend on it (e.g.
	// EraseDevice) to fail. See https://github.com/fleetdm/fleet/issues/41167
	_, err := s.db.ExecContext(
		r.Context, `
INSERT INTO nano_devices
    (id, identity_cert, serial_number, authenticate, authenticate_at)
VALUES
    (?, ?, ?, ?, CURRENT_TIMESTAMP)
ON DUPLICATE KEY
UPDATE
    identity_cert = VALUES(identity_cert),
    serial_number = VALUES(serial_number),
    bootstrap_token_b64 = IF(
        EXISTS(SELECT 1 FROM nano_cert_auth_associations nca WHERE nca.id = ? AND nca.renew_command_uuid IS NOT NULL),
        bootstrap_token_b64,
        NULL
    ),
    bootstrap_token_at = IF(
        EXISTS(SELECT 1 FROM nano_cert_auth_associations nca WHERE nca.id = ? AND nca.renew_command_uuid IS NOT NULL),
        bootstrap_token_at,
        NULL
    ),
    authenticate = VALUES(authenticate),
    authenticate_at = CURRENT_TIMESTAMP;`,
		r.ID, pemCert, nullEmptyString(msg.SerialNumber), msg.Raw, r.ID, r.ID,
	)

	return err
}

func (s *MySQLStorage) storeDeviceTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	query := `UPDATE nano_devices SET token_update = ?, token_update_at = CURRENT_TIMESTAMP`
	args := []interface{}{msg.Raw}
	// separately store the Unlock Token per MDM spec
	if len(msg.UnlockToken) > 0 {
		query += `, unlock_token = ?, unlock_token_at = CURRENT_TIMESTAMP`
		args = append(args, msg.UnlockToken)
	}
	query += ` WHERE id = ? LIMIT 1;`
	args = append(args, r.ID)
	_, err := s.db.ExecContext(r.Context, query, args...)
	return err
}

func (s *MySQLStorage) storeUserTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	// there shouldn't be an Unlock Token on the user channel, but
	// complain if there is to warn an admin
	if len(msg.UnlockToken) > 0 {
		s.logger.InfoContext(r.Context, "Unlock Token on user channel not stored")
	}
	_, err := s.db.ExecContext(
		r.Context, `
INSERT INTO nano_users
    (id, device_id, user_short_name, user_long_name, token_update, token_update_at)
VALUES
    (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON DUPLICATE KEY
UPDATE
    device_id = VALUES(device_id),
    user_short_name = VALUES(user_short_name),
    user_long_name = VALUES(user_long_name),
    token_update = VALUES(token_update),
    token_update_at = CURRENT_TIMESTAMP;`,
		r.ID,
		r.ParentID,
		nullEmptyString(msg.UserShortName),
		nullEmptyString(msg.UserLongName),
		msg.Raw,
	)
	return err
}

func (s *MySQLStorage) StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	var err error
	var deviceId, userId string
	resolved := (&msg.Enrollment).Resolved()
	if err = resolved.Validate(); err != nil {
		return err
	}
	if resolved.IsUserChannel {
		deviceId = r.ParentID
		userId = r.ID
		err = s.storeUserTokenUpdate(r, msg)
	} else {
		deviceId = r.ID
		err = s.storeDeviceTokenUpdate(r, msg)
	}
	if err != nil {
		return err
	}
	var certSerial int64
	if r.Certificate != nil {
		certSerial = r.Certificate.SerialNumber.Int64()
	}
	_, err = s.db.ExecContext(
		r.Context, `
INSERT INTO nano_enrollments
	(id, device_id, user_id, type, topic, push_magic, token_hex, token_update_tally, hardware_attested)
VALUES
	(?, ?, ?, ?, ?, ?, ?, 1,
	 EXISTS(SELECT 1 FROM acme_orders WHERE issued_certificate_serial = ?))
ON DUPLICATE KEY
UPDATE
    device_id = VALUES(device_id),
    user_id = VALUES(user_id),
    type = VALUES(type),
    topic = VALUES(topic),
    push_magic = VALUES(push_magic),
    token_hex = VALUES(token_hex),
    enabled = 1,
    token_update_tally = nano_enrollments.token_update_tally + 1,
	hardware_attested = VALUES(hardware_attested);`,
		r.ID,
		deviceId,
		nullEmptyString(userId),
		r.Type.String(),
		msg.Topic,
		msg.PushMagic,
		msg.Token.String(),
		certSerial,
	)
	if err != nil {
		return err
	}
	// Written synchronously rather than through the async batch: TokenUpdate completes the enrollment,
	// and a freshly enrolled host must report a check-in time right away, not after the next flush.
	return s.upsertSeenTime(r.Context, r.ID)
}

func (s *MySQLStorage) RetrieveTokenUpdateTally(ctx context.Context, id string) (int, error) {
	var tally int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT token_update_tally FROM nano_enrollments WHERE id = ?;`,
		id,
	).Scan(&tally)
	return tally, err
}

func (s *MySQLStorage) StoreUserAuthenticate(r *mdm.Request, msg *mdm.UserAuthenticate) error {
	colName := "user_authenticate"
	colAtName := "user_authenticate_at"
	// if the DigestResponse is empty then this is the first (of two)
	// UserAuthenticate messages depending on our response
	if msg.DigestResponse != "" {
		colName = "user_authenticate_digest"
		colAtName = "user_authenticate_digest_at"
	}
	_, err := s.db.ExecContext(
		//nolint:gosec
		r.Context, `
INSERT INTO nano_users
    (id, device_id, user_short_name, user_long_name, `+colName+`, `+colAtName+`)
VALUES
    (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON DUPLICATE KEY
UPDATE
    device_id = VALUES(device_id),
    user_short_name = VALUES(user_short_name),
    user_long_name = VALUES(user_long_name),
    `+colName+` = VALUES(`+colName+`),
    `+colAtName+` = VALUES(`+colAtName+`);`,
		r.ID,
		r.ParentID,
		nullEmptyString(msg.UserShortName),
		nullEmptyString(msg.UserLongName),
		msg.Raw,
	)
	if err != nil {
		return err
	}
	return s.updateLastSeen(r)
}

// Disable can be called for an Authenticate or CheckOut message
func (s *MySQLStorage) Disable(r *mdm.Request) error {
	if r.ParentID != "" {
		return errors.New("can only disable a device channel")
	}
	tx, err := s.db.BeginTx(r.Context, nil)
	if err != nil {
		return err
	}
	if err := s.disable(r, tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback error: %w; while trying to handle error: %v", rbErr, err)
		}
		return err
	}
	return tx.Commit()
}

// disable flips the enrollments off and bumps their seen times in one transaction so a mid-way
// failure can't leave a bumped seen time on a still-enabled enrollment (the enabled=1 filter in
// the chart online-hosts query relies on the two moving together). The id capture is a plain
// consistent read on purpose: a locking read (SELECT ... FOR UPDATE or INSERT ... SELECT) would
// take shared/exclusive next-key locks on the enrollment rows before the UPDATE's exclusive
// locks, inviting deadlocks with concurrent Disable/StoreTokenUpdate calls. The residual race —
// an enrollment re-enabled between the SELECT and the UPDATE gets disabled without a seen-time
// bump — is accepted: readers only consider seen times of enabled = 1 enrollments, so the
// missing bump is invisible to them.
func (s *MySQLStorage) disable(r *mdm.Request, tx *sql.Tx) error {
	rows, err := tx.QueryContext(r.Context, `SELECT id FROM nano_enrollments WHERE device_id = ? AND enabled = 1 ORDER BY id`, r.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		r.Context,
		`UPDATE nano_enrollments SET enabled = 0, token_update_tally = 0 WHERE device_id = ? AND enabled = 1;`,
		r.ID,
	); err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}
	stmt, args := seenTimesUpsert(ids)
	_, err = tx.ExecContext(r.Context, stmt, args...)
	return err
}

// seenTimesUpsert builds a multi-row nano_seen_times upsert for the given enrollment ids. Callers
// must pass ids in a consistent (sorted) order: concurrent flushes and Disable/StoreTokenUpdate
// upserts can deadlock among themselves without consistent lock ordering.
func seenTimesUpsert(ids []string) (string, []any) {
	stmt := `INSERT INTO nano_seen_times (id, seen_time) VALUES ` +
		strings.TrimSuffix(strings.Repeat("(?, CURRENT_TIMESTAMP),", len(ids)), ",") +
		` ON DUPLICATE KEY UPDATE seen_time = CURRENT_TIMESTAMP`
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return stmt, args
}

func (s *MySQLStorage) updateLastSeen(r *mdm.Request) error {
	if s.asyncLastSeen != nil {
		s.asyncLastSeen.markHostSeen(r.Context, r.ID)
		return nil
	}
	return s.upsertSeenTime(r.Context, r.ID)
}

func (s *MySQLStorage) upsertSeenTime(ctx context.Context, id string) error {
	stmt, args := seenTimesUpsert([]string{id})
	if _, err := s.db.ExecContext(ctx, stmt, args...); err != nil {
		return fmt.Errorf("updating last seen: %w", err)
	}
	return nil
}

func (s *MySQLStorage) updateLastSeenBatch(ctx context.Context, ids []string) {
	if len(ids) == 0 {
		return
	}

	// ids arrive sorted from seenSet.getAndClearLocked; see seenTimesUpsert on why order matters.
	stmt, args := seenTimesUpsert(ids)
	err := common_mysql.WithRetryTxx(ctx, sqlx.NewDb(s.db, ""), func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, stmt, args...)
		return err
	}, s.logger)
	if err != nil {
		s.logger.ErrorContext(ctx, "error batch updating nano_seen_times", "err", err)
	}
}

func (s *MySQLStorage) ExpandEmbeddedSecrets(ctx context.Context, document string) (string, error) {
	s.logger.ErrorContext(ctx, "MySQLStorage.ExpandEmbeddedSecrets not implemented")
	return document, nil
}

func (s *MySQLStorage) ExpandHostSecrets(ctx context.Context, document string, enrollmentID string) (string, error) {
	s.logger.ErrorContext(ctx, "MySQLStorage.ExpandHostSecrets not implemented")
	return document, nil
}

func (s *MySQLStorage) SetRecoveryLockFailed(ctx context.Context, hostUUID string, commandUUID string, errorMsg string) error {
	s.logger.ErrorContext(ctx, "MySQLStorage.SetRecoveryLockFailed not implemented")
	return nil
}
