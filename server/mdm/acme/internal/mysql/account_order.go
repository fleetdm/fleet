package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
	"go.step.sm/crypto/jose"
)

const maxAccountsPerEnrollment = 3

func (ds *Datastore) CreateAccount(ctx context.Context, account *types.Account, onlyReturnExisting bool) (*types.Account, bool, error) {
	var didCreate bool
	err := platform_mysql.WithRetryTxx(ctx, ds.primary, func(tx sqlx.ExtContext) error {
		// Mark the enrollment as locked to prevent concurrent account creation for
		// the same enrollment, so we can enforce limits on account creation
		const lockEnrollmentStmt = `SELECT id FROM acme_enrollments WHERE id = ? FOR UPDATE`
		var enrollmentID uint
		err := sqlx.GetContext(ctx, tx, &enrollmentID, lockEnrollmentStmt, account.EnrollmentID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "lock acme enrollment")
		}

		thumbprint, err := jose.Thumbprint(&account.JSONWebKey)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "compute jwk thumbprint for new account")
		}

		// if the account already exists (and is not revoked), we return it
		const findExistingAccountStmt = `SELECT id, revoked FROM acme_accounts WHERE acme_enrollment_id = ? AND json_web_key_thumbprint = ?`
		var existingAccount struct {
			ID      uint `db:"id"`
			Revoked bool `db:"revoked"`
		}
		err = sqlx.GetContext(ctx, tx, &existingAccount, findExistingAccountStmt, account.EnrollmentID, thumbprint)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return ctxerr.Wrap(ctx, err, "check for existing account with same jwk thumbprint")
		}
		if err == nil {
			if existingAccount.Revoked {
				err = types.AccountRevokedError(fmt.Sprintf("Account %d is revoked", existingAccount.ID))
				return ctxerr.Wrap(ctx, err)
			}
			account.ID = existingAccount.ID
			return nil
		}

		// If we got here we didn't find an existing account so, if requested by the caller, return a notFound
		if onlyReturnExisting {
			err = types.AccountDoesNotExistError(fmt.Sprintf("No account exists for enrollment id %d with the provided jwk", account.EnrollmentID))
			return ctxerr.Wrap(ctx, err)
		}

		// check if maximum number of accounts for this enrollment has been reached before creating a new one
		const countAccountsStmt = `SELECT COUNT(*) FROM acme_accounts WHERE acme_enrollment_id = ?`
		var accountCount int
		err = sqlx.GetContext(ctx, tx, &accountCount, countAccountsStmt, enrollmentID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "count acme accounts for enrollment")
		}
		if accountCount >= maxAccountsPerEnrollment {
			err = types.TooManyAccountsError(fmt.Sprintf("Enrollment id %d already has %d accounts, which is the maximum allowed", enrollmentID, accountCount))
			return ctxerr.Wrap(ctx, err)
		}

		// create the new account
		jwkSerialized, err := account.JSONWebKey.MarshalJSON()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshal new account jwk")
		}

		const insertStmt = `INSERT INTO acme_accounts (acme_enrollment_id, json_web_key, json_web_key_thumbprint) VALUES (?, ?, ?)`
		res, err := tx.ExecContext(ctx, insertStmt, account.EnrollmentID, jwkSerialized, thumbprint)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert acme account")
		}
		lastInsertID, _ := res.LastInsertId() // can never fail with mysql
		account.ID = uint(lastInsertID)
		didCreate = true

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
		return nil, false, err
	}
	return account, didCreate, nil
}

type dbAccount struct {
	types.Account
	JSONWebKeyRaw []byte `db:"json_web_key"`
}

// This method specifically requires an enrollment ID because the caller should know it and have verified it
func (ds *Datastore) GetAccountByID(ctx context.Context, enrollmentID uint, accountID uint) (*types.Account, error) {
	stmt := `SELECT id, acme_enrollment_id, json_web_key FROM acme_accounts WHERE acme_enrollment_id = ? AND id = ?`
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

		countOrdersStmt := `SELECT COUNT(*) FROM acme_orders WHERE acme_account_id = ?`
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

		insertOrderStmt := `INSERT INTO acme_orders (acme_account_id, status, identifiers) VALUES (?, ?, ?)`
		res, err := tx.ExecContext(ctx, insertOrderStmt, order.AccountID, order.Status, identifiersSerialized)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert acme order")
		}
		lastInsertID, err := res.LastInsertId()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get last insert id for acme order")
		}
		order.ID = uint(lastInsertID)

		insertAuthorizationStmt := `INSERT INTO acme_authorizations (acme_order_id, identifier_type, identifier_value, status) VALUES (?, ?, ?, ?)`

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
