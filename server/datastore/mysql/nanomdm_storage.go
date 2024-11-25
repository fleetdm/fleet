package mysql

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"

	abmctx "github.com/fleetdm/fleet/v4/server/contexts/apple_bm"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
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

// EnqueueDeviceLockCommand enqueues a DeviceLock command for the given host.
//
// A few implementation details:
//   - It can only be called for a single hosts, to ensure we don't use the same
//     pin for multiple hosts.
//   - The method performs fleet-specific actions after the command is enqueued.
func (s *NanoMDMStorage) EnqueueDeviceLockCommand(
	ctx context.Context,
	host *fleet.Host,
	cmd *mdm.Command,
	pin string,
) error {
	return withRetryTxx(ctx, s.db, func(tx sqlx.ExtContext) error {
		if err := enqueueCommandDB(ctx, tx, []string{host.UUID}, cmd); err != nil {
			return err
		}

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
	return withRetryTxx(ctx, s.db, func(tx sqlx.ExtContext) error {
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
