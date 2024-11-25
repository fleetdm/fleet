// Package mysql stores and retrieves MDM data from MySQL
package mysql

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/jmoiron/sqlx"
	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// Schema holds the schema for the NanoMDM MySQL storage.
//
//go:embed schema.sql
var Schema string

var ErrNoCert = errors.New("no certificate in MDM Request")

type MySQLStorage struct {
	logger        log.Logger
	db            *sql.DB
	rm            bool
	asyncLastSeen *asyncLastSeen
	reader        func(ctx context.Context) fleet.DBReader
}

type config struct {
	driver        string
	dsn           string
	db            *sql.DB
	logger        log.Logger
	rm            bool
	asyncCap      int
	asyncInterval time.Duration
}

type Option func(*config)

func WithLogger(logger log.Logger) Option {
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

func WithAsyncLastSeen(cap int, interval time.Duration) Option {
	return func(c *config) {
		c.asyncCap = cap
		c.asyncInterval = interval
	}
}

func New(opts ...Option) (*MySQLStorage, error) {
	const (
		asyncLastSeenFlushInterval = 2 * time.Second
		asyncLastSeenCap           = 1000
	)

	cfg := &config{logger: log.NopLogger, driver: "mysql", asyncCap: asyncLastSeenCap, asyncInterval: asyncLastSeenFlushInterval}
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
    bootstrap_token_b64 = NULL,
    bootstrap_token_at = NULL,
    authenticate = VALUES(authenticate),
    authenticate_at = CURRENT_TIMESTAMP;`,
		r.ID, pemCert, nullEmptyString(msg.SerialNumber), msg.Raw,
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
		ctxlog.Logger(r.Context, s.logger).Info(
			"msg", "Unlock Token on user channel not stored",
		)
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
	_, err = s.db.ExecContext(
		r.Context, `
INSERT INTO nano_enrollments
	(id, device_id, user_id, type, topic, push_magic, token_hex, last_seen_at, token_update_tally)
VALUES
	(?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, 1)
ON DUPLICATE KEY
UPDATE
    device_id = VALUES(device_id),
    user_id = VALUES(user_id),
    type = VALUES(type),
    topic = VALUES(topic),
    push_magic = VALUES(push_magic),
    token_hex = VALUES(token_hex),
    enabled = 1,
    last_seen_at = CURRENT_TIMESTAMP,
    token_update_tally = nano_enrollments.token_update_tally + 1;`,
		r.ID,
		deviceId,
		nullEmptyString(userId),
		r.Type.String(),
		msg.Topic,
		msg.PushMagic,
		msg.Token.String(),
	)
	return err
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
	_, err := s.db.ExecContext(
		r.Context,
		`UPDATE nano_enrollments SET enabled = 0, token_update_tally = 0, last_seen_at = CURRENT_TIMESTAMP WHERE device_id = ? AND enabled = 1;`,
		r.ID,
	)
	return err
}

func (s *MySQLStorage) updateLastSeen(r *mdm.Request) (err error) {
	if s.asyncLastSeen != nil {
		s.asyncLastSeen.markHostSeen(r.Context, r.ID)
		return nil
	}

	_, err = s.db.ExecContext(
		r.Context,
		`UPDATE nano_enrollments SET last_seen_at = CURRENT_TIMESTAMP WHERE id = ?`,
		r.ID,
	)
	if err != nil {
		err = fmt.Errorf("updating last seen: %w", err)
	}
	return
}

func (s *MySQLStorage) updateLastSeenBatch(ctx context.Context, ids []string) {
	if len(ids) == 0 {
		return
	}

	stmt, args, err := sqlx.In(`UPDATE nano_enrollments SET last_seen_at = CURRENT_TIMESTAMP WHERE id IN (?)`, ids)
	if err != nil {
		s.logger.Info("msg", "error building nano_enrollments.last_seen_at sql", "err", err)
		return
	}

	_, err = s.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		s.logger.Info("msg", "error batch updating nano_enrollments.last_seen_at", "err", err)
	}
}
