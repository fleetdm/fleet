package mysql

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
)

// Schema contains the MySQL schema for the DEP storage.
//
//go:embed schema.sql
var Schema string

// MySQLStorage implements a storage.AllStorage using MySQL.
type MySQLStorage struct {
	db *sql.DB
}

var _ storage.AllDEPStorage = (*MySQLStorage)(nil)

type config struct {
	driver string
	dsn    string
	db     *sql.DB
}

// Option allows configuring a MySQLStorage.
type Option func(*config)

// WithDSN sets the storage MySQL data source name.
func WithDSN(dsn string) Option {
	return func(c *config) {
		c.dsn = dsn
	}
}

// WithDriver sets a custom MySQL driver for the storage.
//
// Default driver is "mysql".
// Value is ignored if WithDB is used.
func WithDriver(driver string) Option {
	return func(c *config) {
		c.driver = driver
	}
}

// WithDB sets a custom MySQL *sql.DB to the storage.
//
// If set, driver passed via WithDriver is ignored.
func WithDB(db *sql.DB) Option {
	return func(c *config) {
		c.db = db
	}
}

// New creates and returns a new MySQLStorage.
func New(opts ...Option) (*MySQLStorage, error) {
	cfg := &config{driver: "mysql"}
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
	return &MySQLStorage{db: cfg.db}, nil
}

// RetrieveAuthTokens reads the DEP OAuth tokens for name DEP name.
func (s *MySQLStorage) RetrieveAuthTokens(ctx context.Context, name string) (*client.OAuth1Tokens, error) {
	var (
		consumerKey       sql.NullString
		consumerSecret    sql.NullString
		accessToken       sql.NullString
		accessSecret      sql.NullString
		accessTokenExpiry sql.NullTime
	)
	err := s.db.QueryRowContext(
		ctx, `
SELECT
	consumer_key,
	consumer_secret,
	access_token,
	access_secret,
	access_token_expiry
FROM
    nano_dep_names
WHERE
    name = ?;`,
		name,
	).Scan(
		&consumerKey,
		&consumerSecret,
		&accessToken,
		&accessSecret,
		&accessTokenExpiry,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}
	if !consumerKey.Valid { // all auth token fields are set together
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &client.OAuth1Tokens{
		ConsumerKey:       consumerKey.String,
		ConsumerSecret:    consumerSecret.String,
		AccessToken:       accessToken.String,
		AccessSecret:      accessSecret.String,
		AccessTokenExpiry: accessTokenExpiry.Time,
	}, nil
}

// StoreAuthTokens saves the DEP OAuth tokens for the DEP name.
func (s *MySQLStorage) StoreAuthTokens(ctx context.Context, name string, tokens *client.OAuth1Tokens) error {
	_, err := s.db.ExecContext(
		ctx, `
INSERT INTO nano_dep_names
	(name, consumer_key, consumer_secret, access_token, access_secret, access_token_expiry)
VALUES
	(?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
	consumer_key = VALUES(consumer_key),
	consumer_secret = VALUES(consumer_secret),
	access_token = VALUES(access_token),
	access_secret = VALUES(access_secret),
	access_token_expiry = VALUES(access_token_expiry);`,
		name,
		tokens.ConsumerKey,
		tokens.ConsumerSecret,
		tokens.AccessToken,
		tokens.AccessSecret,
		tokens.AccessTokenExpiry,
	)
	return err
}

// RetrieveConfig reads the DEP config for the DEP name.
//
// Returns an empty config if the config does not exist (to support a fallback default config).
func (s *MySQLStorage) RetrieveConfig(ctx context.Context, name string) (*client.Config, error) {
	var baseURL sql.NullString
	err := s.db.QueryRowContext(
		ctx,
		`SELECT config_base_url FROM nano_dep_names WHERE name = ?;`,
		name,
	).Scan(
		&baseURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// an 'empty' config is valid
			return &client.Config{}, nil
		}
		return nil, err
	}
	var config client.Config
	if baseURL.Valid {
		config.BaseURL = baseURL.String
	}
	return &config, nil
}

// StoreConfig saves the DEP config for name DEP name.
func (s *MySQLStorage) StoreConfig(ctx context.Context, name string, config *client.Config) error {
	_, err := s.db.ExecContext(
		ctx, `
INSERT INTO nano_dep_names
	(name, config_base_url)
VALUES
	(?, ?)
ON DUPLICATE KEY UPDATE
	config_base_url = VALUES(config_base_url);`,
		name,
		config.BaseURL,
	)
	return err
}

// RetrieveAssignerProfile reads the assigner profile UUID and its timestamp for name DEP name.
//
// Returns an empty profile if it does not exist.
func (s *MySQLStorage) RetrieveAssignerProfile(ctx context.Context, name string) (profileUUID string, modTime time.Time, err error) {
	var (
		profileUUID_ sql.NullString
		modTime_     sql.NullTime
	)
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT assigner_profile_uuid, assigner_profile_uuid_at FROM nano_dep_names WHERE name = ?;`,
		name,
	).Scan(
		&profileUUID_, &modTime_,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// an 'empty' profile is valid
			return "", time.Time{}, nil
		}
		return "", time.Time{}, err
	}
	if profileUUID_.Valid {
		profileUUID = profileUUID_.String
	}
	if modTime_.Valid {
		modTime = modTime_.Time
	}
	return profileUUID, modTime, nil
}

// StoreAssignerProfile saves the assigner profile UUID for name DEP name.
func (s *MySQLStorage) StoreAssignerProfile(ctx context.Context, name string, profileUUID string) error {
	_, err := s.db.ExecContext(
		ctx, `
INSERT INTO nano_dep_names
	(name, assigner_profile_uuid, assigner_profile_uuid_at)
VALUES
	(?, ?, CURRENT_TIMESTAMP)
ON DUPLICATE KEY UPDATE
	assigner_profile_uuid = VALUES(assigner_profile_uuid),
	assigner_profile_uuid_at = VALUES(assigner_profile_uuid_at);`,
		name,
		profileUUID,
	)
	return err
}

// RetrieveCursor reads the reads the DEP fetch and sync cursor for name DEP name.
//
// Returns an empty cursor if the cursor does not exist.
func (s *MySQLStorage) RetrieveCursor(ctx context.Context, name string) (cursor string, modTime time.Time, err error) {
	var (
		cursor_  sql.NullString
		cursorAt sql.NullTime
	)
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT syncer_cursor, syncer_cursor_at FROM nano_dep_names WHERE name = ?;`,
		name,
	).Scan(
		&cursor_, &cursorAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", time.Time{}, nil
		}
		return "", time.Time{}, err
	}
	if !cursor_.Valid {
		return "", time.Time{}, nil
	}
	return cursor_.String, cursorAt.Time, nil
}

// StoreCursor saves the DEP fetch and sync cursor for name DEP name.
func (s *MySQLStorage) StoreCursor(ctx context.Context, name, cursor string) error {
	_, err := s.db.ExecContext(
		ctx, `
INSERT INTO nano_dep_names
	(name, syncer_cursor, syncer_cursor_at)
VALUES
	(?, ?, CURRENT_TIMESTAMP)
ON DUPLICATE KEY UPDATE
	syncer_cursor = VALUES(syncer_cursor),
	syncer_cursor_at = VALUES(syncer_cursor_at);`,
		name,
		cursor,
	)
	return err
}

// StoreTokenPKI stores the PEM bytes in pemCert and pemKey for name DEP name.
func (s *MySQLStorage) StoreTokenPKI(ctx context.Context, name string, pemCert []byte, pemKey []byte) error {
	_, err := s.db.ExecContext(
		ctx, `
INSERT INTO nano_dep_names
	(name, tokenpki_cert_pem, tokenpki_key_pem)
VALUES
	(?, ?, ?)
ON DUPLICATE KEY UPDATE
	tokenpki_cert_pem = VALUES(tokenpki_cert_pem),
	tokenpki_key_pem = VALUES(tokenpki_key_pem);`,
		name,
		pemCert,
		pemKey,
	)
	return err
}

// RetrieveTokenPKI reads the PEM bytes for the DEP token exchange certificate
// and private key using name DEP name.
func (s *MySQLStorage) RetrieveTokenPKI(ctx context.Context, name string) (pemCert []byte, pemKey []byte, err error) {
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT tokenpki_cert_pem, tokenpki_key_pem FROM nano_dep_names WHERE name = ?;`,
		name,
	).Scan(
		&pemCert, &pemKey,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, storage.ErrNotFound
		}
		return nil, nil, err
	}
	if pemCert == nil { // tokenpki_cert_pem and tokenpki_key_pem are set together
		return nil, nil, storage.ErrNotFound
	}
	return pemCert, pemKey, nil
}
