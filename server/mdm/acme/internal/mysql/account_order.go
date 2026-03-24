package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
	"go.step.sm/crypto/jose"
)

func (ds *Datastore) CreateAccount(ctx context.Context, account *types.Account, onlyReturnExisting bool) (*types.Account, error) {
	const maxAccountsPerEnrollment = 3

	err := platform_mysql.WithRetryTxx(ctx, ds.primary, func(tx sqlx.ExtContext) error {
		lockEnrollmentStmt := `SELECT id FROM acme_enrollments WHERE id = ? FOR UPDATE`
		var enrollmentID uint

		// Mark the enrollment as locked to prevent concurrent account creation for the same enrollment, so we can enforce limits on account creation
		err := tx.QueryRowxContext(ctx, lockEnrollmentStmt, account.EnrollmentID).Scan(&enrollmentID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "lock acme enrollment")
		}

		// TODO: what if it's revoked? do not return it but do not create it either?
		// if the account already exists, we return it
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

		// check if maximum number of accounts for this enrollment has been reached before creating a new one
		countAccountsStmt := `SELECT COUNT(*) FROM acme_accounts WHERE enrollment_id = ?`
		var accountCount int
		err = tx.QueryRowxContext(ctx, countAccountsStmt, enrollmentID).Scan(&accountCount)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "count acme accounts for enrollment")
		}
		if accountCount >= maxAccountsPerEnrollment {
			return ctxerr.Errorf(ctx, "account creation limit reached for enrollment id %d", enrollmentID)
		}

		// create the new account
		jwkSerialized, err := account.JSONWebKey.MarshalJSON()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshal new account jwk")
		}

		insertStmt := `INSERT INTO acme_accounts (enrollment_id, json_web_key) VALUES (?, ?)`
		res, err := tx.ExecContext(ctx, insertStmt, account.EnrollmentID, jwkSerialized)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert acme account")
		}
		lastInsertID, _ := res.LastInsertId() // can never fail with mysql
		account.ID = uint(lastInsertID)

		// if the acme_enrollment has a NULL not_valid_after it should be set to a value
		// 24 hours in the future so that now that this enrollment is being used it will expire
		const updateEnrollmentStmt = `UPDATE acme_enrollments SET not_valid_after = COALESCE(not_valid_after, DATE_ADD(NOW(), INTERVAL 24 HOUR)) WHERE id = ?`
		_, err = tx.ExecContext(ctx, updateEnrollmentStmt, enrollmentID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update acme enrollment not_valid_after")
		}

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
