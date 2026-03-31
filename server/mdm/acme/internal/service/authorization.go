package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
)

func (s *Service) GetAuthorization(ctx context.Context, enrollment *types.Enrollment, account *types.Account, authorizationID uint) (*types.AuthorizationResponse, error) {
	if authorizationID == 0 {
		return nil, types.MalformedError("invalid authorization ID")
	}

	authz, err := s.store.GetAuthorizationByID(ctx, account.ID, authorizationID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting authorization by ID")
	}

	// TODO: Should we check the auth status here? Or will that happen when we validate a challenge for an auth?

	challenges, err := s.store.GetChallengesByAuthorizationID(ctx, authz.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting challenges by authorization ID")
	}

	baseURL, err := s.getACMEBaseURL(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting base URL")
	}

	var challengeResponses []types.ChallengeResponse
	for _, c := range challenges {
		challengeURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "challenges", fmt.Sprint(c.ID))
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "constructing challenge URL")
		}

		challengeResponses = append(challengeResponses, types.ChallengeResponse{
			ChallengeType: c.ChallengeType,
			Status:        c.Status,
			Token:         c.Token,
			URL:           challengeURL,
		})
	}

	authzURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "authorizations", fmt.Sprint(authz.ID))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing authorization URL")
	}

	return &types.AuthorizationResponse{
		Status:     authz.Status,
		Expires:    enrollment.NotValidAfter,
		Identifier: authz.Identifier,
		Challenges: challengeResponses,
		Location:   authzURL,
	}, nil
}
