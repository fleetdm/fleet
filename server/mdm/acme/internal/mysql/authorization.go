package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/jmoiron/sqlx"
)

// This file does not handle normal authentication, but the ACME concept of authorization as part of the protocol.

func (ds *Datastore) GetAuthorizationByID(ctx context.Context, accountID uint, authorizationID uint) (*types.Authorization, error) {
	if accountID == 0 {
		return nil, types.MalformedError("invalid account ID")
	}
	if authorizationID == 0 {
		return nil, types.MalformedError("invalid authorization ID")
	}

	var dbAuthz struct {
		types.Authorization
		IdentifierType  string `db:"identifier_type"`
		IdentifierValue string `db:"identifier_value"`
	}

	const query = `SELECT a.id, a.acme_order_id, a.identifier_type, a.identifier_value, a.status FROM acme_authorizations a
	INNER JOIN acme_orders o ON a.acme_order_id = o.id
	INNER JOIN acme_accounts ac ON o.acme_account_id = ac.id
	WHERE a.id = ? AND ac.id = ?`
	err := sqlx.GetContext(ctx, ds.reader(ctx), &dbAuthz, query, authorizationID, accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, types.AuthorizationDoesNotExistError(fmt.Sprintf("ACME authorization with ID %d not found for account ID %d", authorizationID, accountID))
		}
		return nil, err
	}

	return &types.Authorization{
		ID:          dbAuthz.ID,
		ACMEOrderID: dbAuthz.ACMEOrderID,
		Identifier: types.Identifier{
			Type:  dbAuthz.IdentifierType,
			Value: dbAuthz.IdentifierValue,
		},
		Status: dbAuthz.Status,
	}, nil
}

func (ds *Datastore) GetAuthorizationsByOrderID(ctx context.Context, orderID uint) ([]*types.Authorization, error) {
	const stmt = `SELECT id, acme_order_id, identifier_type, identifier_value, status FROM acme_authorizations WHERE acme_order_id = ?`
	var authorizations []*types.Authorization
	err := ds.primary.SelectContext(ctx, &authorizations, stmt, orderID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get acme authorizations for order")
	}

	return authorizations, nil
}
