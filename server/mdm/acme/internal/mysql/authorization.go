package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/jmoiron/sqlx"
)

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
			// TODO: What should we return here to avoid authorization enumeration and not leak details?
			return nil, types.UnauthorizedError("")
		}
		return nil, err
	}

	return &types.Authorization{
		ID:      dbAuthz.ID,
		OrderID: dbAuthz.OrderID,
		Identifier: types.Identifier{
			Type:  dbAuthz.IdentifierType,
			Value: dbAuthz.IdentifierValue,
		},
		Status: dbAuthz.Status,
	}, nil
}
