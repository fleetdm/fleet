package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"uuid"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/golang-jwt/jwt/v4"

	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nano_service "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	nanomdm_service "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/certauth"
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
		Issuer:           signingToken.ServerUUID,
		IssuedAt:         jwt.NewNumericDate(time.Now()),
		ID:               uuid.New().String(),
	})

	signedToken, err := token.SignedString(key)
	if err != nil {
		return "", nanomdm_service.NewHTTPStatusError(http.StatusInternalServerError, ctxerr.Wrap(ctx, err, "signing MAID token"))
	}
	logger.InfoContext(ctx, "issued com.apple.maid token")
	return signedToken, nil
}

// ============ Check-in Wrappers ============

type certVerifierEnrollmentCheckinService struct {
	nano_service.CheckinAndCommandService
	ds          fleet.Datastore
	nanoStorage fleet.MDMAppleStore
	config      config.MDMConfig
	logger      *slog.Logger
}

// newCertVerifierEnrollmentCheckinService wraps next with the certificate verifier enrollment check.
func newCertVerifierEnrollmentCheckinService(next nano_service.CheckinAndCommandService, ds fleet.Datastore, nanoStorage fleet.MDMAppleStore, config config.MDMConfig, logger *slog.Logger) nano_service.CheckinAndCommandService {
	return &certVerifierEnrollmentCheckinService{CheckinAndCommandService: next, ds: ds, nanoStorage: nanoStorage, config: config, logger: logger}
}

func (s *certVerifierEnrollmentCheckinService) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	resolved := m.Enrollment.Resolved()
	if resolved == nil {
		return s.reject(r, m, nil, "no resolved enrollment")
	}
	// r.EnrollID is only set by nanomdm's core service, which runs after this wrapper, so normalize like certauth.
	cloned := r.Clone()
	cloned.EnrollID = &mdm.EnrollID{
		ID:   resolved.DeviceChannelID,
		Type: resolved.Type,
	}

	if r.Certificate == nil {
		// let certauth reject the missing certificate
		return s.CheckinAndCommandService.Authenticate(r, m)
	}

	associated, err := s.nanoStorage.IsCertHashAssociated(cloned, certauth.HashCert(r.Certificate))
	if err != nil {
		return ctxerr.Wrap(r.Context, err, "checking cert hash association")
	}
	if associated {
		// a resend of an Authenticate already accepted for this enrollment
		return s.CheckinAndCommandService.Authenticate(r, m)
	}

	data, err := apple_mdm.ParseAppleMDMCertificateBindingExtension(r.Certificate)
	if err != nil {
		return s.reject(r, m, nil, "invalid certificate binding extension: "+err.Error())
	}
	if data == nil {
		if !s.config.AppleSCEPStaticChallengeEnabled {
			return s.reject(r, m, nil, "certificate binding extension required")
		}
		return s.CheckinAndCommandService.Authenticate(r, m)
	}

	switch data.Purpose {
	// ota_phase1 certificates only sign the OTA phase 2 request, so they're rejected by the default case.
	case fleet.AppleMDMCertPurposeADE, fleet.AppleMDMCertPurposeOTAPhaseTwo:
		if resolved.Type != mdm.Device {
			return s.reject(r, m, data, "enrollment type does not match certificate purpose")
		}
		if !boundMatches(data.Serial, m.SerialNumber) || !boundMatches(data.UDID, m.UDID) {
			return s.reject(r, m, data, "serial or UDID does not match certificate binding")
		}
	case fleet.AppleMDMCertPurposeACME:
		if resolved.Type != mdm.Device {
			return s.reject(r, m, data, "enrollment type does not match certificate purpose")
		}
		if !boundMatches(data.Serial, m.SerialNumber) {
			return s.reject(r, m, data, "serial does not match certificate binding")
		}
	case fleet.AppleMDMCertPurposeACMERenewal:
		if resolved.Type != mdm.Device {
			return s.reject(r, m, data, "enrollment type does not match certificate purpose")
		}
		if !boundMatches(data.Serial, m.SerialNumber) || !boundMatches(data.EnrollmentID, resolved.DeviceChannelID) {
			return s.reject(r, m, data, "serial or enrollment ID does not match certificate binding")
		}
	case fleet.AppleMDMCertPurposeSCEPRenewal:
		if !boundMatches(data.EnrollmentID, resolved.DeviceChannelID) {
			return s.reject(r, m, data, "enrollment ID does not match certificate binding")
		}
	case fleet.AppleMDMCertPurposeADUE:
		if resolved.Type != mdm.UserEnrollmentDevice {
			return s.reject(r, m, data, "enrollment type does not match certificate purpose")
		}

		nanoEnrollment, err := s.ds.GetNanoMDMEnrollment(r.Context, resolved.DeviceChannelID)
		if err != nil {
			return ctxerr.Wrap(r.Context, err, "getting nano enrollment for ADUE authenticate")
		}
		if nanoEnrollment != nil {
			return s.reject(r, m, data, "enrollment ID is already enrolled")
		}

		token, ok := strings.CutPrefix(r.Authorization, "Bearer ")
		if !ok || token == "" {
			return s.reject(r, m, data, "missing ADUE bearer token")
		}
		// the challenge was just consumed on the primary when the profile was handed out
		challenge, err := s.ds.GetADUEEnrollmentChallenge(ctxdb.RequirePrimary(r.Context, true), token)
		if err != nil {
			if fleet.IsNotFound(err) {
				return s.reject(r, m, data, "ADUE bearer token not found")
			}
			return ctxerr.Wrap(r.Context, err, "getting ADUE enrollment challenge")
		}
		if !boundMatches(data.IDPAccountUUID, challenge.IdPAccountUUID) {
			return s.reject(r, m, data, "IdP account does not match certificate binding")
		}
	default:
		return s.reject(r, m, data, "unsupported certificate binding purpose")
	}

	return s.CheckinAndCommandService.Authenticate(r, m)
}

// boundMatches reports whether a value bound in the certificate is set and equals the claimed one.
func boundMatches(bound *string, claimed string) bool {
	return bound != nil && *bound != "" && *bound == claimed
}

func (s *certVerifierEnrollmentCheckinService) reject(r *mdm.Request, m *mdm.Authenticate, binding *apple_mdm.AppleMDMCertificateBindingExtension, reason string) error {
	attrs := []any{
		"reason", reason,
		"claimed_udid", m.UDID,
		"claimed_serial", m.SerialNumber,
		"claimed_enrollment_id", m.EnrollmentID,
	}
	if binding != nil {
		attrs = append(attrs,
			"bound_purpose", binding.Purpose,
			"bound_udid", ptr.ValOrZero(binding.UDID),
			"bound_serial", ptr.ValOrZero(binding.Serial),
			"bound_enrollment_id", ptr.ValOrZero(binding.EnrollmentID),
			"bound_idp_account_uuid", ptr.ValOrZero(binding.IDPAccountUUID),
		)
	}
	s.logger.InfoContext(r.Context, "rejecting MDM Authenticate", attrs...)
	return nano_service.NewHTTPStatusError(http.StatusForbidden, errors.New(reason))
}
