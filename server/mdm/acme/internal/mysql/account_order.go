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

const (
	maxAccountsPerEnrollment = 3
	maxOrdersPerAccount      = 3
)

func (ds *Datastore) CreateAccount(ctx context.Context, account *types.Account, onlyReturnExisting bool) (*types.Account, bool, error) {
	var didCreate bool
	err := platform_mysql.WithRetryTxx(ctx, ds.primary, func(tx sqlx.ExtContext) error {
		// Mark the enrollment as locked to prevent concurrent account creation for
		// the same enrollment, so we can enforce limits on account creation
		const lockEnrollmentStmt = `SELECT id FROM acme_enrollments WHERE id = ? FOR UPDATE`
		var enrollmentID uint
		err := sqlx.GetContext(ctx, tx, &enrollmentID, lockEnrollmentStmt, account.ACMEEnrollmentID)
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
		err = sqlx.GetContext(ctx, tx, &existingAccount, findExistingAccountStmt, account.ACMEEnrollmentID, thumbprint)
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
			err = types.AccountDoesNotExistError(fmt.Sprintf("No account exists for enrollment id %d with the provided jwk", account.ACMEEnrollmentID))
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
		res, err := tx.ExecContext(ctx, insertStmt, account.ACMEEnrollmentID, jwkSerialized, thumbprint)
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

// This method specifically requires an enrollment ID because
// the caller should know it and have verified it.
func (ds *Datastore) GetAccountByID(ctx context.Context, enrollmentID uint, accountID uint) (*types.Account, error) {
	const stmt = `SELECT id, acme_enrollment_id, json_web_key, json_web_key_thumbprint
		FROM acme_accounts
		WHERE acme_enrollment_id = ? AND id = ? AND revoked = false`

	var dbAcc struct {
		types.Account
		JSONWebKeyRaw []byte `db:"json_web_key"`
	}
	err := sqlx.GetContext(ctx, ds.reader(ctx), &dbAcc, stmt, enrollmentID, accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = types.AccountDoesNotExistError(fmt.Sprintf("No account exists with id %d", accountID))
			return nil, ctxerr.Wrap(ctx, err)
		}
		return nil, ctxerr.Wrap(ctx, err, "select acme account")
	}

	var jwk jose.JSONWebKey
	if err := jwk.UnmarshalJSON(dbAcc.JSONWebKeyRaw); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshal acme account jwk")
	}
	dbAcc.JSONWebKey = jwk

	return &dbAcc.Account, nil
}

// NB: We are leaving it to the caller to set the proper status, token, etc on the order, authorization and challenge
func (ds *Datastore) CreateOrder(ctx context.Context, order *types.Order, authorization *types.Authorization, challenge *types.Challenge) (*types.Order, error) {
	err := platform_mysql.WithRetryTxx(ctx, ds.primary, func(tx sqlx.ExtContext) error {
		// Mark the account as locked to prevent concurrent order creation for the same account, so we can enforce limits on order creation
		const lockAccountStmt = `SELECT id FROM acme_accounts WHERE id = ? FOR UPDATE`
		var accountID uint
		err := sqlx.GetContext(ctx, tx, &accountID, lockAccountStmt, order.ACMEAccountID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "lock acme account")
		}

		const countOrdersStmt = `SELECT COUNT(*) FROM acme_orders WHERE acme_account_id = ?`
		var orderCount int
		err = sqlx.GetContext(ctx, tx, &orderCount, countOrdersStmt, order.ACMEAccountID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "count acme orders for account")
		}
		if orderCount >= maxOrdersPerAccount {
			err = types.TooManyOrdersError(fmt.Sprintf("Account id %d already has %d orders, which is the maximum allowed", order.ACMEAccountID, orderCount))
			return ctxerr.Wrap(ctx, err)
		}

		identifiersSerialized, err := json.Marshal(order.Identifiers)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshal order identifiers")
		}

		const insertOrderStmt = `INSERT INTO acme_orders (
			acme_account_id, finalized, certificate_signing_request, identifiers, status, issued_certificate_serial
		) VALUES (?, ?, ?, ?, ?, ?)`
		res, err := tx.ExecContext(ctx, insertOrderStmt,
			order.ACMEAccountID, order.Finalized, order.CertificateSigningRequest, identifiersSerialized, order.Status, order.IssuedCertificateSerial)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert acme order")
		}
		lastInsertID, _ := res.LastInsertId() // can never fail for mysql
		order.ID = uint(lastInsertID)

		const insertAuthorizationStmt = `INSERT INTO acme_authorizations (acme_order_id, identifier_type, identifier_value, status) VALUES (?, ?, ?, ?)`
		res, err = tx.ExecContext(ctx, insertAuthorizationStmt, order.ID, authorization.Identifier.Type, authorization.Identifier.Value, authorization.Status)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert acme authorization")
		}
		lastInsertID, _ = res.LastInsertId() // can never fail for mysql
		authorization.ID = uint(lastInsertID)

		const insertChallengeStmt = `INSERT INTO acme_challenges (acme_authorization_id, challenge_type, token, status) VALUES (?, ?, ?, ?)`
		res, err = tx.ExecContext(ctx, insertChallengeStmt, authorization.ID, challenge.ChallengeType, challenge.Token, challenge.Status)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert acme challenge")
		}
		lastInsertID, _ = res.LastInsertId() // can never fail for mysql
		challenge.ID = uint(lastInsertID)
		challenge.ACMEAuthorizationID = authorization.ID

		return nil
	}, ds.logger)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (ds *Datastore) FinalizeOrder(ctx context.Context, orderID uint, csrPEM string, certSerial int64) error {
	const stmt = `UPDATE acme_orders SET status = ?, finalized=1, certificate_signing_request = ?, issued_certificate_serial = ? WHERE id = ?`
	_, err := ds.writer(ctx).ExecContext(ctx, stmt, types.OrderStatusValid, csrPEM, certSerial, orderID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update acme order with signed certificate")
	}
	return nil
}

func (ds *Datastore) GetOrderByID(ctx context.Context, accountID, orderID uint) (*types.Order, []*types.Authorization, error) {
	// condition is on both account and order ids, so that we don't get a match on
	// just the order id that wouldn't be associated with the validated account from
	// the request.
	const getOrderStmt = `SELECT id, acme_account_id, finalized, certificate_signing_request, identifiers, status, issued_certificate_serial
		FROM acme_orders WHERE acme_account_id = ? AND id = ?`

	var dbOrder struct {
		types.Order
		RawIdentifiers []byte `db:"identifiers"`
	}
	err := sqlx.GetContext(ctx, ds.reader(ctx), &dbOrder, getOrderStmt, accountID, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = types.OrderDoesNotExistError(fmt.Sprintf("No order exists with id %d for this account", orderID))
			return nil, nil, ctxerr.Wrap(ctx, err)
		}
		return nil, nil, ctxerr.Wrap(ctx, err, "select acme order")
	}

	if err := json.Unmarshal(dbOrder.RawIdentifiers, &dbOrder.Identifiers); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "unmarshal acme order identifiers")
	}

	const listAuthorizationsStmt = `SELECT id, acme_order_id, identifier_type, identifier_value, status
		FROM acme_authorizations WHERE acme_order_id = ?`
	var dbAuthz []struct {
		types.Authorization
		IdentifierType  string `db:"identifier_type"`
		IdentifierValue string `db:"identifier_value"`
	}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &dbAuthz, listAuthorizationsStmt, orderID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "select acme authorizations for order")
	}

	authorizations := make([]*types.Authorization, len(dbAuthz))
	for i, a := range dbAuthz {
		authz := a.Authorization
		authz.Identifier = types.Identifier{
			Type:  a.IdentifierType,
			Value: a.IdentifierValue,
		}
		authorizations[i] = &authz
	}
	return &dbOrder.Order, authorizations, nil
}

func (ds *Datastore) GetCertificatePEMByOrderID(ctx context.Context, accountID, orderID uint) (string, error) {
	const getCertStmt = `SELECT certificate_pem
		FROM
			identity_certificates ic
			JOIN acme_orders o ON ic.serial = o.issued_certificate_serial
		WHERE
			o.acme_account_id = ? AND
			o.id = ? AND
			ic.revoked IS FALSE`

	var certPEM string
	err := sqlx.GetContext(ctx, ds.reader(ctx), &certPEM, getCertStmt, accountID, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = types.CertificateDoesNotExistError(fmt.Sprintf("No certificate exists for order id %d for this account", orderID))
			return "", ctxerr.Wrap(ctx, err)
		}
		return "", ctxerr.Wrap(ctx, err, "select certificate PEM for order")
	}
	return certPEM, nil
}

func (ds *Datastore) ListAccountOrderIDs(ctx context.Context, accountID uint) ([]uint, error) {
	// must not include orders in status 'invalid'
	const listOrderIDsStmt = `SELECT id FROM acme_orders WHERE acme_account_id = ? AND status != 'invalid'`
	var ids []uint
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &ids, listOrderIDsStmt, accountID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select acme order ids for account")
	}
	return ids, nil
}
