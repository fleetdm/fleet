package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetChallengesByAuthorizationID(ctx context.Context, authorizationID uint) ([]*types.Challenge, error) {
	if authorizationID == 0 {
		return nil, types.MalformedError("invalid authorization ID")
	}

	const query = `SELECT id, acme_authorization_id, challenge_type, status, token, updated_at FROM acme_challenges WHERE acme_authorization_id = ?`

	var challenges []*types.Challenge
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &challenges, query, authorizationID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting challenges by authorization ID")
	}
	if len(challenges) == 0 {
		return nil, types.ChallengeDoesNotExistError(fmt.Sprintf("No challenges found for authorization ID %d", authorizationID))
	}

	return challenges, nil
}

// We require the accountID to validate the challenge belongs to the account trying to validate it
func (ds *Datastore) GetChallengeByID(ctx context.Context, accountID, challengeID uint) (*types.Challenge, error) {
	if challengeID == 0 {
		return nil, types.MalformedError("invalid challenge ID")
	}

	const query = `SELECT ac.id, ac.acme_authorization_id, ac.challenge_type, ac.status, ac.token, ac.updated_at FROM acme_challenges ac
	INNER JOIN acme_authorizations a ON ac.acme_authorization_id = a.id
	INNER JOIN acme_orders o ON a.acme_order_id = o.id
	WHERE ac.id = ? AND o.acme_account_id = ?`
	var challenge types.Challenge
	err := sqlx.GetContext(ctx, ds.reader(ctx), &challenge, query, challengeID, accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, types.ChallengeDoesNotExistError(fmt.Sprintf("Challenge with ID %d not found for account ID %d", challengeID, accountID))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting challenge by ID")
	}
	return &challenge, nil
}

// UpdateChallenge handles updating the challenge status, and the authorization status as well as moving the order status.
func (ds *Datastore) UpdateChallenge(ctx context.Context, challenge *types.Challenge) (*types.Challenge, error) {
	if challenge == nil {
		return nil, errors.New("Challenge can not be nil for update")
	}

	err := platform_mysql.WithRetryTxx(ctx, ds.writer(ctx), func(tx sqlx.ExtContext) error {
		const updateChallengeStmt = `UPDATE acme_challenges SET status = ? WHERE id = ?`
		_, err := tx.ExecContext(ctx, updateChallengeStmt, challenge.Status, challenge.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "updating challenge")
		}

		const updateAuthorizationStatusStmt = `UPDATE acme_authorizations a INNER JOIN acme_challenges c ON a.id = c.acme_authorization_id
		SET a.status = CASE
			WHEN c.status = 'valid' THEN 'valid'
			ELSE 'invalid'
		END
		WHERE c.id = ?`
		_, err = tx.ExecContext(ctx, updateAuthorizationStatusStmt, challenge.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "updating authorization status based on challenge status")
		}

		// We can confidently update the order status here based on the challenge and authorization status
		// since we currently only have one, if we ever add more the state machine should account for all authorizations
		// to be valid before moving the order to ready
		const updateOrderStatusStmt = `UPDATE acme_orders o INNER JOIN acme_authorizations a ON o.id = a.acme_order_id
		SET o.status = CASE
			WHEN ? = 'valid' THEN 'ready'
			WHEN ? = 'invalid' THEN 'invalid'
			ELSE o.status
		END
		WHERE a.id = ? AND o.status != 'invalid'`
		_, err = tx.ExecContext(ctx, updateOrderStatusStmt, challenge.Status, challenge.Status, challenge.ACMEAuthorizationID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "updating order status based on challenge status")
		}

		const selectQuery = `SELECT id, acme_authorization_id, challenge_type, status, token, updated_at FROM acme_challenges WHERE id = ?`
		err = sqlx.GetContext(ctx, tx, challenge, selectQuery, challenge.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting updated challenge")
		}

		return nil
	}, ds.logger)
	if err != nil {
		return nil, err
	}
	return challenge, nil
}
