package mysql

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetChallengesByAuthorizationID(ctx context.Context, authorizationID uint) ([]*types.Challenge, error) {
	if authorizationID == 0 {
		return nil, types.MalformedError("invalid authorization ID")
	}

	// TODO: Should we get validated from here, via the updated_at if status=valid?
	const query = `SELECT id, acme_authorization_id, challenge_type, status, token FROM acme_challenges WHERE acme_authorization_id = ?`

	var challenges []*types.Challenge
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &challenges, query, authorizationID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting challenges by authorization ID")
	}
	// TODO: What if we have no rows? That shouldn't happen since we validated the authorization, so probably fine to return service error
	if len(challenges) == 0 {
		return nil, ctxerr.New(ctx, "no challenges found for authorization ID")
	}

	fmt.Printf("%#v\n", challenges)

	return challenges, nil
}
