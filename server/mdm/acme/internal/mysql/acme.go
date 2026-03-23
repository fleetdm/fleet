// Package mysql provides the MySQL datastore implementation for the ACME bounded context.
package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel"
	"go.step.sm/crypto/jose"
)

// tracer is an OTEL tracer. It has no-op behavior when OTEL is not enabled.
var tracer = otel.Tracer("github.com/fleetdm/fleet/v4/server/mdm/acme/internal/mysql")

// Datastore is the MySQL implementation of the activity datastore.
type Datastore struct {
	primary *sqlx.DB
	replica *sqlx.DB
	logger  *slog.Logger
}

// NewDatastore creates a new MySQL datastore for activities.
func NewDatastore(conns *platform_mysql.DBConnections, logger *slog.Logger) *Datastore {
	return &Datastore{primary: conns.Primary, replica: conns.Replica, logger: logger}
}

func (ds *Datastore) reader(ctx context.Context) fleet.DBReader {
	if ctxdb.IsPrimaryRequired(ctx) {
		return ds.primary
	}
	return ds.replica
}

// func (ds *Datastore) writer(_ context.Context) *sqlx.DB {
// 	return ds.primary
// }

// Ensure Datastore implements types.Datastore
var _ types.Datastore = (*Datastore)(nil)

func (ds *Datastore) GetEnrollmentByIdentifier(ctx context.Context, identifier string) (*types.Enrollment, error) {
	stmt := `SELECT id, acme_identifier, host_identifier, not_valid_after, revoked FROM acme_enrollments WHERE acme_identifier = ?`
	var enrollment types.Enrollment
	err := ds.primary.GetContext(ctx, &enrollment, stmt, identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, platform_mysql.NotFound("acme enrollment").WithName(identifier)
		}
		return nil, ctxerr.Wrap(ctx, err, "select acme enrollment")
	}
	return &enrollment, nil
}

func (ds *Datastore) CreateAccount(ctx context.Context, account *types.Account, onlyReturnExisting bool) (*types.Account, error) {
	err := platform_mysql.WithRetryTxx(ctx, ds.primary, func(tx sqlx.ExtContext) error {
		lockEnrollmentStmt := `SELECT id FROM acme_enrollments WHERE id = ? FOR UPDATE`
		var enrollmentID uint

		// Mark the enrollment as locked to prevent concurrent account creation for the same enrollment, so we can enforce limits on account creation
		err := tx.QueryRowxContext(ctx, lockEnrollmentStmt, account.EnrollmentID).Scan(&enrollmentID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "lock acme enrollment")
		}

		findExistingAccountStmt := `SELECT id FROM acme_accounts WHERE enrollment_id = ? AND json_web_key_thumbprint = ?`
		thumbprint, err := jose.Thumbprint(&account.JSONWebKey)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "compute jwk thumbprint for new account")
		}
		var existingAccountID uint
		err = tx.QueryRowxContext(ctx, findExistingAccountStmt, account.EnrollmentID, thumbprint).Scan(&existingAccountID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return ctxerr.Wrap(ctx, err, "check for existing account with same jwk thumbprint")
		}
		if err == nil {
			account.ID = existingAccountID
			return nil
		}

		// If we got here we didn't find an existing account so, if requested by the caller, return a notFound
		if onlyReturnExisting {
			return platform_mysql.NotFound("acme account").WithName(thumbprint)
		}

		countAccountsStmt := `SELECT COUNT(*) FROM acme_accounts WHERE enrollment_id = ?`
		var accountCount int
		err = tx.QueryRowxContext(ctx, countAccountsStmt, enrollmentID).Scan(&accountCount)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "count acme accounts for enrollment")
		}
		if accountCount >= 3 {
			return ctxerr.Errorf(ctx, "account creation limit reached for enrollment id %d", enrollmentID)
		}

		jwkSerialized, err := account.JSONWebKey.MarshalJSON()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshal new account jwk")
		}

		insertStmt := `INSERT INTO acme_accounts (enrollment_id, json_web_key) VALUES (?, ?)`
		res, err := tx.ExecContext(ctx, insertStmt, account.EnrollmentID, jwkSerialized)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert acme account")
		}
		lastInsertID, err := res.LastInsertId()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get last insert id")
		}
		account.ID = uint(lastInsertID)
		return nil
	}, ds.logger)
	if err != nil {
		return nil, err
	}
	return account, nil
}

type dbAccount struct {
	types.Account
	JSONWebKeyRaw []byte `db:"json_web_key"`
}

// This method specifically requires an enrollment ID because the caller should know it and have verified it
func (ds *Datastore) GetAccountByID(ctx context.Context, enrollmentID uint, accountID uint) (*types.Account, error) {
	stmt := `SELECT id, enrollment_id, json_web_key FROM acme_accounts WHERE enrollment_id = ? AND id = ?`
	var dbAcc dbAccount
	err := sqlx.GetContext(ctx, ds.reader(ctx), &dbAcc, stmt, enrollmentID, accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, platform_mysql.NotFound("acme account").WithID(accountID)
		}
		return nil, ctxerr.Wrap(ctx, err, "select acme account")
	}
	var jwk jose.JSONWebKey
	err = jwk.UnmarshalJSON(dbAcc.JSONWebKeyRaw)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshal acme account jwk")
	}
	dbAcc.JSONWebKey = jwk
	return &dbAcc.Account, nil
}

// NB: We are leaving it to the caller to set the proper status, token, etc on the order, authorization and challenge
func (ds *Datastore) CreateOrder(ctx context.Context, order *types.Order, authorization *types.Authorization, challenge *types.Challenge) (*types.Order, error) {
	err := platform_mysql.WithRetryTxx(ctx, ds.primary, func(tx sqlx.ExtContext) error {
		lockAccountStmt := `SELECT id FROM acme_accounts WHERE id = ? FOR UPDATE`
		var accountID uint

		// Mark the account as locked to prevent concurrent order creation for the same account, so we can enforce limits on order creation
		err := tx.QueryRowxContext(ctx, lockAccountStmt, order.AccountID).Scan(&accountID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "lock acme account")
		}

		countOrdersStmt := `SELECT COUNT(*) FROM acme_orders WHERE account_id = ?`
		var orderCount int
		err = tx.QueryRowxContext(ctx, countOrdersStmt, order.AccountID).Scan(&orderCount)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "count acme orders for account")
		}
		if orderCount >= 3 {
			return ctxerr.Errorf(ctx, "order creation limit reached for account id %d", order.AccountID)
		}

		identifiersSerialized, err := json.Marshal(order.Identifiers)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshal order identifiers")
		}

		insertOrderStmt := `INSERT INTO acme_orders (account_id, status, identifiers) VALUES (?, ?, ?)`
		res, err := tx.ExecContext(ctx, insertOrderStmt, order.AccountID, order.Status, identifiersSerialized)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert acme order")
		}
		lastInsertID, err := res.LastInsertId()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get last insert id for acme order")
		}
		order.ID = uint(lastInsertID)

		insertAuthorizationStmt := `INSERT INTO acme_authorizations (order_id, identifier_type, identifier_value, status) VALUES (?, ?, ?, ?)`

		res, err = tx.ExecContext(ctx, insertAuthorizationStmt, order.ID, authorization.Identifier.Type, authorization.Identifier.Value, authorization.Status)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert acme authorization")
		}
		lastInsertID, err = res.LastInsertId()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get last insert id for acme authorization")
		}
		authorization.ID = uint(lastInsertID)

		insertChallengeStmt := `INSERT INTO acme_challenges (authorization_id, challenge_type, token, status) VALUES (?, ?, ?, ?)`
		res, err = tx.ExecContext(ctx, insertChallengeStmt, authorization.ID, challenge.Type, challenge.Token, challenge.Status)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert acme challenge")
		}
		lastInsertID, err = res.LastInsertId()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get last insert id for acme challenge")
		}
		challenge.ID = uint(lastInsertID)

		return nil
	}, ds.logger)
	if err != nil {
		return nil, err
	}
	return order, nil
}
