package mysql

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	nanodep_mysql "github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nanomdm_mysql "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage/mysql"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
)

// NanoMDMStorage wraps a *nanomdm_mysql.MySQLStorage and overrides further functionality.
type NanoMDMStorage struct {
	*nanomdm_mysql.MySQLStorage

	db          *sqlx.DB
	logger      log.Logger
	pushCertPEM []byte
	pushKeyPEM  []byte
}

// NewMDMAppleMDMStorage returns a MySQL nanomdm storage that uses the Datastore
// underlying MySQL writer *sql.DB.
func (ds *Datastore) NewMDMAppleMDMStorage(pushCertPEM []byte, pushKeyPEM []byte) (*NanoMDMStorage, error) {
	s, err := nanomdm_mysql.New(nanomdm_mysql.WithDB(ds.primary.DB))
	if err != nil {
		return nil, err
	}
	return &NanoMDMStorage{
		MySQLStorage: s,
		pushCertPEM:  pushCertPEM,
		pushKeyPEM:   pushKeyPEM,
		db:           ds.primary,
		logger:       ds.logger,
	}, nil
}

// RetrievePushCert partially implements nanomdm_storage.PushCertStore.
//
// Always returns "0" as stale token because we are not storing the APNS in MySQL storage,
// and instead loading them at startup, thus the APNS will never be considered stale.
func (s *NanoMDMStorage) RetrievePushCert(
	ctx context.Context, topic string,
) (cert *tls.Certificate, staleToken string, err error) {
	tlsCert, err := tls.X509KeyPair(s.pushCertPEM, s.pushKeyPEM)
	if err != nil {
		return nil, "", err
	}
	return &tlsCert, "0", nil
}

// IsPushCertStale partially implements nanomdm_storage.PushCertStore.
//
// Given that we are not storing the APNS certificate in MySQL storage, and instead loading
// them at startup (as env variables), the APNS will never be considered stale.
//
// TODO(lucas): Revisit solution to support changing the APNS.
func (s *NanoMDMStorage) IsPushCertStale(ctx context.Context, topic, staleToken string) (bool, error) {
	return false, nil
}

// StorePushCert partially implements nanomdm_storage.PushCertStore.
//
// Leaving this unimplemented as APNS certificate and key are not stored in MySQL storage,
// instead they are loaded to memory at startup.
func (s *NanoMDMStorage) StorePushCert(ctx context.Context, pemCert, pemKey []byte) error {
	return errors.New("unimplemented")
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

// NewMDMAppleDEPStorage returns a MySQL nanodep storage that uses the Datastore
// underlying MySQL writer *sql.DB.
func (ds *Datastore) NewMDMAppleDEPStorage(tok nanodep_client.OAuth1Tokens) (*NanoDEPStorage, error) {
	s, err := nanodep_mysql.New(nanodep_mysql.WithDB(ds.primary.DB))
	if err != nil {
		return nil, err
	}

	return &NanoDEPStorage{
		MySQLStorage: s,
		tokens:       tok,
	}, nil
}

// NanoDEPStorage wraps a *nanodep_mysql.MySQLStorage and overrides functionality to load
// DEP auth tokens from memory.
type NanoDEPStorage struct {
	*nanodep_mysql.MySQLStorage

	tokens nanodep_client.OAuth1Tokens
}

// RetrieveAuthTokens partially implements nanodep.AuthTokensRetriever.
//
// RetrieveAuthTokens returns the DEP auth tokens stored in memory.
func (s *NanoDEPStorage) RetrieveAuthTokens(ctx context.Context, name string) (*nanodep_client.OAuth1Tokens, error) {
	return &s.tokens, nil
}

// StoreAuthTokens partially implements nanodep.AuthTokensStorer.
//
// Leaving this unimplemented as DEP auth tokens are not stored in MySQL storage,
// instead they are loaded to memory at startup.
func (s *NanoDEPStorage) StoreAuthTokens(ctx context.Context, name string, tokens *nanodep_client.OAuth1Tokens) error {
	return errors.New("unimplemented")
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
