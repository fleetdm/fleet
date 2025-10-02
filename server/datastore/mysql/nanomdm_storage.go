package mysql

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	abmctx "github.com/fleetdm/fleet/v4/server/contexts/apple_bm"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	nanodep_mysql "github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nanomdm_mysql "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage/mysql"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
	nanomdm_log "github.com/micromdm/nanolib/log"
)

// lockConflictError indicates a lock command already exists for the host
type lockConflictError struct {
	hostUUID string
}

func (e lockConflictError) Error() string {
	return "host already has a pending lock command"
}

func (e lockConflictError) IsConflict() bool {
	return true
}

// isConflict checks if an error implements the IsConflict() interface
func isConflict(err error) bool {
	type conflictInterface interface {
		IsConflict() bool
	}
	if c, ok := err.(conflictInterface); ok {
		return c.IsConflict()
	}
	return false
}

// NanoMDMStorage wraps a *nanomdm_mysql.MySQLStorage and overrides further functionality.
type NanoMDMStorage struct {
	*nanomdm_mysql.MySQLStorage

	db     *sqlx.DB
	logger log.Logger
	ds     fleet.Datastore
}

type nanoMDMLogAdapter struct {
	logger log.Logger
}

func (l nanoMDMLogAdapter) Info(args ...interface{}) {
	level.Info(l.logger).Log(args...)
}

func (l nanoMDMLogAdapter) Debug(args ...interface{}) {
	level.Debug(l.logger).Log(args...)
}

func (l nanoMDMLogAdapter) With(args ...interface{}) nanomdm_log.Logger {
	wl := log.With(l.logger, args...)
	return nanoMDMLogAdapter{logger: wl}
}

// NewMDMAppleMDMStorage returns a MySQL nanomdm storage that uses the Datastore
// underlying MySQL writer *sql.DB.
func (ds *Datastore) NewMDMAppleMDMStorage() (*NanoMDMStorage, error) {
	s, err := nanomdm_mysql.New(
		nanomdm_mysql.WithDB(ds.primary.DB),
		nanomdm_mysql.WithLogger(nanoMDMLogAdapter{logger: ds.logger}),
		nanomdm_mysql.WithReaderFunc(ds.reader),
	)
	if err != nil {
		return nil, err
	}
	return &NanoMDMStorage{
		MySQLStorage: s,
		db:           ds.primary,
		logger:       ds.logger,
		ds:           ds,
	}, nil
}

// NewTestMDMAppleMDMStorage returns a test MySQL nanomdm storage that uses the
// Datastore underlying MySQL writer *sql.DB. It allows configuring the async
// last seen time's capacity and interval and should only be used in tests.
func (ds *Datastore) NewTestMDMAppleMDMStorage(asyncCap int, asyncInterval time.Duration) (*NanoMDMStorage, error) {
	s, err := nanomdm_mysql.New(
		nanomdm_mysql.WithDB(ds.primary.DB),
		nanomdm_mysql.WithLogger(nanoMDMLogAdapter{logger: ds.logger}),
		nanomdm_mysql.WithReaderFunc(ds.reader),
		nanomdm_mysql.WithAsyncLastSeen(asyncCap, asyncInterval),
	)
	if err != nil {
		return nil, err
	}
	return &NanoMDMStorage{
		MySQLStorage: s,
		db:           ds.primary,
		logger:       ds.logger,
		ds:           ds,
	}, nil
}

// RetrievePushCert partially implements nanomdm_storage.PushCertStore.
//
// Always returns "0" as stale token because fleet.Datastore always returns a valid push certificate.
func (s *NanoMDMStorage) RetrievePushCert(
	ctx context.Context, topic string,
) (*tls.Certificate, string, error) {
	cert, err := assets.APNSKeyPair(ctx, s.ds)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "loading push certificate")
	}
	return cert, "0", nil
}

// IsPushCertStale partially implements nanomdm_storage.PushCertStore.
//
// Always returns `false` because the underlying datastore implementation makes sure that the token is always fresh.
func (s *NanoMDMStorage) IsPushCertStale(ctx context.Context, topic, staleToken string) (bool, error) {
	return false, nil
}

// StorePushCert partially implements nanomdm_storage.PushCertStore.
func (s *NanoMDMStorage) StorePushCert(ctx context.Context, pemCert, pemKey []byte) error {
	return errors.New("please use fleet.Datastore to manage MDM assets")
}

// GetPendingLockCommand returns the most recent unacknowledged DeviceLock command
// for the given host, along with its unlock PIN.
// Returns nil, "", nil if no pending lock command exists.
func (s *NanoMDMStorage) GetPendingLockCommand(ctx context.Context, hostUUID string) (*mdm.Command, string, error) {
	query := `
		SELECT nc.command_uuid, nc.request_type, nc.command, hma.unlock_pin
		FROM nano_commands nc
		INNER JOIN host_mdm_actions hma ON hma.lock_ref = nc.command_uuid
		LEFT JOIN nano_command_results ncr ON ncr.command_uuid = nc.command_uuid
		INNER JOIN nano_enrollment_queue neq ON neq.command_uuid = nc.command_uuid
		WHERE neq.id = ?
		AND nc.request_type = 'DeviceLock'
		AND ncr.command_uuid IS NULL
		ORDER BY nc.created_at DESC
		LIMIT 1`

	var result struct {
		CommandUUID string `db:"command_uuid"`
		RequestType string `db:"request_type"`
		Command     []byte `db:"command"`
		UnlockPIN   string `db:"unlock_pin"`
	}

	err := sqlx.GetContext(ctx, s.db, &result, query, hostUUID)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "getting pending lock command")
	}

	cmd := &mdm.Command{
		CommandUUID: result.CommandUUID,
		Command: struct {
			RequestType string
		}{
			RequestType: result.RequestType,
		},
		Raw: result.Command,
	}

	return cmd, result.UnlockPIN, nil
}

// EnqueueDeviceLockCommand enqueues a DeviceLock command for the given host.
//
// A few implementation details:
//   - It can only be called for a single hosts, to ensure we don't use the same
//     pin for multiple hosts.
//   - The method performs fleet-specific actions after the command is enqueued.
//   - It will fail with a ConflictError if a lock command already exists.
func (s *NanoMDMStorage) EnqueueDeviceLockCommand(
	ctx context.Context,
	host *fleet.Host,
	cmd *mdm.Command,
	pin string,
) error {
	return common_mysql.WithRetryTxx(ctx, s.db, func(tx sqlx.ExtContext) error {
		// check if a lock already exists using SELECT FOR UPDATE to prevent a race
		var existingLockRef *string
		err := sqlx.GetContext(ctx, tx, &existingLockRef,
			`SELECT lock_ref FROM host_mdm_actions WHERE host_id = ? FOR UPDATE`,
			host.ID)

		// If we got a row and it has a lock_ref, fail with conflict
		if err == nil && existingLockRef != nil && *existingLockRef != "" {
			// A lock command already exists, don't overwrite
			return lockConflictError{hostUUID: host.UUID}
		}

		// If the row doesn't exist, that's OK, we'll insert it
		if err != nil && err != sql.ErrNoRows {
			return ctxerr.Wrap(ctx, err, "checking for existing lock")
		}

		// Now enqueue the command
		if err := enqueueCommandDB(ctx, tx, []string{host.UUID}, cmd); err != nil {
			return err
		}

		// Insert or update the host_mdm_actions row
		stmt := `
			INSERT INTO host_mdm_actions (
				host_id,
				lock_ref,
				unlock_pin,
				fleet_platform
			)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				wipe_ref   = NULL,
				unlock_ref = NULL,
				unlock_pin = VALUES(unlock_pin),
				lock_ref   = VALUES(lock_ref)`

		if _, err := tx.ExecContext(ctx, stmt, host.ID, cmd.CommandUUID, pin, host.FleetPlatform()); err != nil {
			return ctxerr.Wrap(ctx, err, "modifying host_mdm_actions for DeviceLock")
		}

		return nil
	}, s.logger)
}

// EnqueueDeviceWipeCommand enqueues a EraseDevice command for the given host.
func (s *NanoMDMStorage) EnqueueDeviceWipeCommand(ctx context.Context, host *fleet.Host, cmd *mdm.Command) error {
	return common_mysql.WithRetryTxx(ctx, s.db, func(tx sqlx.ExtContext) error {
		if err := enqueueCommandDB(ctx, tx, []string{host.UUID}, cmd); err != nil {
			return err
		}

		stmt := `
			INSERT INTO host_mdm_actions (
				host_id,
				wipe_ref,
				fleet_platform
			)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE
				wipe_ref   = VALUES(wipe_ref)`

		if _, err := tx.ExecContext(ctx, stmt, host.ID, cmd.CommandUUID, host.FleetPlatform()); err != nil {
			return ctxerr.Wrap(ctx, err, "modifying host_mdm_actions for DeviceWipe")
		}

		return nil
	}, s.logger)
}

func (s *NanoMDMStorage) GetAllMDMConfigAssetsByName(ctx context.Context, assetNames []fleet.MDMAssetName,
	queryerContext sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
	return s.ds.GetAllMDMConfigAssetsByName(ctx, assetNames, queryerContext)
}

func (s *NanoMDMStorage) GetABMTokenByOrgName(ctx context.Context, orgName string) (*fleet.ABMToken, error) {
	return s.ds.GetABMTokenByOrgName(ctx, orgName)
}

// ExpandEmbeddedSecrets in NanoMDMStorage overrides the implementation in nanomdm_mysql.MySQLStorage.
func (s *NanoMDMStorage) ExpandEmbeddedSecrets(ctx context.Context, document string) (string, error) {
	return s.ds.ExpandEmbeddedSecrets(ctx, document)
}

// ClearQueue in NanoMDMStorage overrides the implementation in
// nanomdm_mysql.MySQLStorage. It does call
// nanomdm_mysql.MySQLStorage.ClearQueue, but expands on its behavior.
func (s *NanoMDMStorage) ClearQueue(r *mdm.Request) error {
	err := common_mysql.WithRetryTxx(r.Context, s.db, func(tx sqlx.ExtContext) error {
		if err := s.ds.ClearMDMUpcomingActivitiesDB(r.Context, tx, r.ID); err != nil {
			return err
		}
		return nil
	}, s.logger)

	if err != nil {
		return err
	}

	return s.MySQLStorage.ClearQueue(r)
}

// NewMDMAppleDEPStorage returns a MySQL nanodep storage that uses the Datastore
// underlying MySQL writer *sql.DB.
func (ds *Datastore) NewMDMAppleDEPStorage() (*NanoDEPStorage, error) {
	s, err := nanodep_mysql.New(nanodep_mysql.WithDB(ds.primary.DB))
	if err != nil {
		return nil, err
	}

	return &NanoDEPStorage{
		MySQLStorage: s,
		ds:           ds,
	}, nil
}

// NanoDEPStorage wraps a *nanodep_mysql.MySQLStorage and overrides functionality to load
// DEP auth tokens from the tables managed by Fleet.
type NanoDEPStorage struct {
	*nanodep_mysql.MySQLStorage
	ds fleet.Datastore
}

// RetrieveAuthTokens partially implements nanodep.AuthTokensRetriever. NOTE: this method will first
// check the context for an ABM token; if it doesn't find one, it will fall back to checking the DB.
// This is so we can use the existing DEP client machinery without major changes. See
// https://github.com/fleetdm/fleet/issues/21177 for more details.
func (s *NanoDEPStorage) RetrieveAuthTokens(ctx context.Context, name string) (*nanodep_client.OAuth1Tokens, error) {
	if ctxTok, ok := abmctx.FromContext(ctx); ok {
		return ctxTok, nil
	}

	token, err := assets.ABMToken(ctx, s.ds, name)
	if err != nil {
		return nil, fmt.Errorf("retrieving token in nano dep storage: %w", err)
	}

	return token, nil
}

// StoreAuthTokens partially implements nanodep.AuthTokensStorer.
func (s *NanoDEPStorage) StoreAuthTokens(ctx context.Context, name string, tokens *nanodep_client.OAuth1Tokens) error {
	return errors.New("please use fleet.Datastore to manage MDM assets")
}

func enqueueCommandDB(ctx context.Context, tx sqlx.ExtContext, ids []string, cmd *mdm.Command) error {
	// NOTE: the code to insert into nano_commands and
	// nano_enrollment_queue was copied verbatim from the nanomdm
	// implementation. Ideally we modify some of the interfaces to not
	// duplicate the code here, but that needs more careful planning
	// (which we lack right now)
	if len(ids) < 1 {
		return errors.New("no id(s) supplied to queue command to")
	}
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO nano_commands (command_uuid, request_type, command) VALUES (?, ?, ?);`,
		cmd.CommandUUID, cmd.Command.RequestType, cmd.Raw,
	)
	if err != nil {
		return err
	}
	query := `INSERT INTO nano_enrollment_queue (id, command_uuid) VALUES (?, ?)`
	query += strings.Repeat(", (?, ?)", len(ids)-1)
	args := make([]interface{}, len(ids)*2)
	for i, id := range ids {
		args[i*2] = id
		args[i*2+1] = cmd.CommandUUID
	}
	if _, err = tx.ExecContext(ctx, query+";", args...); err != nil {
		return err
	}

	return nil
}
