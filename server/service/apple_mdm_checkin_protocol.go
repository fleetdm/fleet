package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
	"uuid"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/golang-jwt/jwt/v4"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nanomdm_service "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
)

// This file contains the Apple MDM Check-in protocol service implementations.
// https://developer.apple.com/documentation/devicemanagement/check-in
// Specifically the custom services that are plugged into the nanomdm (coreMDMService).
// It does not contain all, as it's an ongoing effort to refactor methods and services over, the rest can most likely be found in apple_mdm.go

type MDMAppleGetTokenService struct {
	ds     fleet.Datastore
	logger *slog.Logger
}

func NewMDMAppleGetTokenService(ds fleet.Datastore, logger *slog.Logger) *MDMAppleGetTokenService {
	return &MDMAppleGetTokenService{
		ds:     ds,
		logger: logger,
	}
}

func (s *MDMAppleGetTokenService) GetToken(r *mdm.Request, token *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	// * Once we support more than one, move to switch
	if token.TokenServiceType != fleet.TokenServiceTypeMAID {
		return nil, nanomdm_service.NewHTTPStatusError(http.StatusBadRequest, ctxerr.New(r.Context, fmt.Sprintf("unsupported token service type: %s", token.TokenServiceType)))
	}

	signingToken, source, err := s.getSigningToken(r.Context, token.Identifier())
	if err != nil {
		return nil, err
	}
	logger := s.logger.With("identifier", token.Identifier(), "source", source, "abm_token_id", signingToken.ID)
	logger.DebugContext(r.Context, "found valid signing token for get token request")

	signedToken, err := s.signMAIDToken(r.Context, logger, signingToken, token.Identifier())
	if err != nil {
		return nil, err
	}

	return &mdm.GetTokenResponse{
		TokenData: []byte(signedToken),
	}, nil
}

func (s *MDMAppleGetTokenService) getSigningToken(ctx context.Context, identifier string) (signingToken *fleet.ABMToken, source string, err error) {
	abTokens, err := s.ds.ListABMTokens(ctx)
	if err != nil {
		return nil, "", nanomdm_service.NewHTTPStatusError(http.StatusInternalServerError, ctxerr.Wrap(ctx, err, "listing AB tokens"))
	}
	if len(abTokens) == 0 {
		// We can't sign a token with a server uuid if there are no AB tokens
		return nil, "", nanomdm_service.NewHTTPStatusError(http.StatusBadRequest, ctxerr.New(ctx, "no AB tokens found"))
	}

	// find the default token (via is_default or the only one)
	var defaultSigningToken *fleet.ABMToken
	if len(abTokens) == 1 {
		defaultSigningToken = abTokens[0]
	} else {
		for _, t := range abTokens {
			if t.IsDefault {
				defaultSigningToken = t
				break
			}
		}
	}

	liteHost, err := s.ds.HostLiteByIdentifier(ctx, identifier)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, "", nanomdm_service.NewHTTPStatusError(http.StatusInternalServerError, ctxerr.Wrap(ctx, err, "getting host by identifier"))
	}
	if liteHost == nil {
		if defaultSigningToken == nil {
			return nil, "", nanomdm_service.NewHTTPStatusError(http.StatusBadRequest, ctxerr.New(ctx, "host not found and no default AB token found"))
		}

		if defaultSigningToken.ServerUUID == "" {
			return nil, "", nanomdm_service.NewHTTPStatusError(http.StatusBadRequest, ctxerr.New(ctx, "default AB token has no server UUID"))
		}
		// not found, so return the default token.
		return defaultSigningToken, fleet.TokenSourceDefault, nil
	}

	depAssignment, err := s.ds.GetHostDEPAssignment(ctx, liteHost.ID)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, "", nanomdm_service.NewHTTPStatusError(http.StatusInternalServerError, ctxerr.Wrap(ctx, err, "getting host DEP assignment"))
	}

	// if we have a DEP assignment, then force that token if there is any issues, we will not fall back to the default token.
	if depAssignment != nil && depAssignment.DeletedAt == nil {
		if depAssignment.ABMTokenID == nil {
			return nil, "", nanomdm_service.NewHTTPStatusError(http.StatusBadRequest, ctxerr.New(ctx, "host has DEP assignment but no AB token ID"))
		}
		s.logger.InfoContext(ctx, "Getting signing token from DEP assignment", "identifier", identifier, "abm_token_id", *depAssignment.ABMTokenID)

		abToken, err := s.ds.GetABMTokenByID(ctx, *depAssignment.ABMTokenID)
		if err != nil && !fleet.IsNotFound(err) {
			return nil, "", nanomdm_service.NewHTTPStatusError(http.StatusInternalServerError, ctxerr.Wrap(ctx, err, "getting AB token by ID"))
		}
		if abToken == nil {
			return nil, "", nanomdm_service.NewHTTPStatusError(http.StatusBadRequest, ctxerr.New(ctx, "host has DEP assignment but no AB token found"))
		}
		if abToken.ServerUUID == "" {
			return nil, "", nanomdm_service.NewHTTPStatusError(http.StatusBadRequest, ctxerr.New(ctx, "host has DEP assignment but AB token has no server UUID"))
		}

		return abToken, fleet.TokenSourceDEPAssignment, nil
	}

	// for existing hosts that are not DEP assigned, return the default token if it exists, otherwise return an error.
	if defaultSigningToken != nil {
		if defaultSigningToken.ServerUUID == "" {
			return nil, "", nanomdm_service.NewHTTPStatusError(http.StatusBadRequest, ctxerr.New(ctx, "default AB token has no server UUID"))
		}

		return defaultSigningToken, fleet.TokenSourceDefault, nil
	}

	return nil, "", nanomdm_service.NewHTTPStatusError(http.StatusBadRequest, ctxerr.New(ctx, "no default AB token found"))
}

type appleMAIDTokenClaims struct {
	TokenServiceType string `json:"service_type"`
	jwt.RegisteredClaims
}

func (s *MDMAppleGetTokenService) signMAIDToken(ctx context.Context, logger *slog.Logger, signingToken *fleet.ABMToken, identifier string) (string, error) {
	logger.InfoContext(ctx, "signing com.apple.maid token")
	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetABMKey}, nil)
	if err != nil {
		return "", nanomdm_service.NewHTTPStatusError(http.StatusInternalServerError, ctxerr.Wrap(ctx, err, "getting ABM key asset"))
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM(assets[fleet.MDMAssetABMKey].Value)
	if err != nil {
		return "", nanomdm_service.NewHTTPStatusError(http.StatusInternalServerError, ctxerr.Wrap(ctx, err, "parsing ABM key asset"))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, &appleMAIDTokenClaims{
		TokenServiceType: fleet.TokenServiceTypeMAID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:   signingToken.ServerUUID,
			IssuedAt: jwt.NewNumericDate(time.Now()),
			ID:       uuid.New().String(),
		},
	})

	signedToken, err := token.SignedString(key)
	if err != nil {
		return "", nanomdm_service.NewHTTPStatusError(http.StatusInternalServerError, ctxerr.Wrap(ctx, err, "signing MAID token"))
	}
	logger.InfoContext(ctx, "issued com.apple.maid token")
	return signedToken, nil
}
