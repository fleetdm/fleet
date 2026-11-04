package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// entraDeviceTokenMaxAge bounds how old a reported token can be, regardless of the
// lifetime the tenant configures for ID tokens.
const entraDeviceTokenMaxAge = time.Hour

var hostUUIDRe = regexp.MustCompile(`^[0-9A-Fa-f-]{1,64}$`)

// entraDeviceNonce binds an Entra device token to a host. The host sees its own nonce, so it
// isn't secret from it; the HMAC keeps other parties from deriving it, and the hardware UUID
// prefix lets fleetd refuse to request a token for another host's nonce.
func entraDeviceNonce(secret, hostUUID string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("entra-device-nonce|" + hostUUID))
	return hostUUID + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// entraDeviceTokenRequest returns the Entra device token request to send to the host in its
// orbit config, or nil if the host doesn't need to report one.
func (svc *Service) entraDeviceTokenRequest(ctx context.Context, host *fleet.Host) (*fleet.OrbitEntraDeviceTokenRequest, error) {
	clientID := svc.config.MicrosoftCompliancePartner.ClientID
	if host.Platform != "darwin" || clientID == "" || !hostUUIDRe.MatchString(host.UUID) {
		return nil, nil
	}
	if !svc.hostRequiresConditionalAccessMicrosoftIngestion(ctx, host) {
		return nil, nil
	}
	integration, err := svc.ds.ConditionalAccessMicrosoftGet(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get conditional access integration")
	}
	return &fleet.OrbitEntraDeviceTokenRequest{
		TenantID: integration.TenantID,
		ClientID: clientID,
		Nonce:    entraDeviceNonce(integration.ProxyServerSecret, host.UUID),
	}, nil
}

func postOrbitEntraDeviceTokenEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.OrbitPostEntraDeviceTokenRequest)
	if err := svc.SetHostEntraDeviceToken(ctx, req.IDToken); err != nil {
		return fleet.OrbitPostEntraDeviceTokenResponse{Err: err}, nil
	}
	return fleet.OrbitPostEntraDeviceTokenResponse{}, nil
}

func (svc *Service) SetHostEntraDeviceToken(ctx context.Context, idToken string) error {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return newOsqueryError("internal error: missing host from request context")
	}
	clientID := svc.config.MicrosoftCompliancePartner.ClientID
	if host.Platform != "darwin" || clientID == "" || !svc.hostRequiresConditionalAccessMicrosoftIngestion(ctx, host) {
		return badRequest("conditional access is not enabled for this host")
	}
	integration, err := svc.ds.ConditionalAccessMicrosoftGet(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get conditional access integration")
	}

	claims, err := verifyEntraDeviceToken(ctx, idToken, integration.TenantID, clientID, svc.clock.Now())
	if err != nil {
		if _, ok := errors.AsType[entraSigningKeysError](err); ok {
			return ctxerr.Wrap(ctx, err, "verify Entra device token")
		}
		return badRequestf("invalid Entra device token: %s", err)
	}
	if !hmac.Equal([]byte(claims.Nonce), []byte(entraDeviceNonce(integration.ProxyServerSecret, host.UUID))) {
		return badRequest("invalid Entra device token: nonce mismatch")
	}

	changed, err := svc.ds.CreateHostConditionalAccessStatus(ctx, host.ID, claims.DeviceID, claims.PreferredUsername)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create host conditional access status")
	}
	if changed {
		// Compliance is only sent to Intune with policy results; a refetch sends them now instead
		// of at the next policy interval, so the user doesn't have to refetch from Fleet Desktop.
		if err := svc.ds.UpdateHostRefetchRequested(ctx, host.ID, true); err != nil {
			return ctxerr.Wrap(ctx, err, "request host refetch")
		}
	}
	return nil
}

type entraDeviceTokenClaims struct {
	jwt.RegisteredClaims
	TenantID          string `json:"tid"`
	DeviceID          string `json:"deviceid"`
	Nonce             string `json:"nonce"`
	PreferredUsername string `json:"preferred_username"`
}

// entraSigningKeysError is a failure to get Entra's signing keys, a server-side problem
// rather than an invalid token.
type entraSigningKeysError struct{ err error }

func (e entraSigningKeysError) Error() string { return "get Entra signing keys: " + e.err.Error() }
func (e entraSigningKeysError) Unwrap() error { return e.err }

// entraJWKSClients caches one client (with its background refresh) per JWKS URI. Failures
// aren't cached, so a transient error doesn't break verification until restart. The
// client refreshes hourly, so a key Microsoft just rotated in fails until then; fleetd
// retries failed reports.
var (
	entraJWKSMu      sync.Mutex
	entraJWKSClients = map[string]jwkset.Storage{}
)

func entraJWKS() (jwkset.Storage, error) {
	jwksURI := "https://login.microsoftonline.com/common/discovery/v2.0/keys"
	if devURI := dev_mode.Env("FLEET_DEV_ENTRA_DEVICE_TOKEN_JWKS_URI"); devURI != "" {
		jwksURI = devURI
	}
	entraJWKSMu.Lock()
	defer entraJWKSMu.Unlock()
	if c, ok := entraJWKSClients[jwksURI]; ok {
		return c, nil
	}
	c, err := jwkset.NewDefaultHTTPClient([]string{jwksURI})
	if err != nil {
		return nil, err
	}
	entraJWKSClients[jwksURI] = c
	return c, nil
}

func verifyEntraDeviceToken(ctx context.Context, rawToken, tenantID, clientID string, now time.Time) (*entraDeviceTokenClaims, error) {
	if rawToken == "" {
		return nil, errors.New("empty token")
	}
	keys, err := entraJWKS()
	if err != nil {
		return nil, entraSigningKeysError{err}
	}
	var claims entraDeviceTokenClaims
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}), jwt.WithoutClaimsValidation())
	if _, err := parser.ParseWithClaims(rawToken, &claims, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("missing kid header")
		}
		key, err := keys.KeyRead(ctx, kid)
		if errors.Is(err, jwkset.ErrKeyNotFound) {
			return nil, errors.New("unknown signing key")
		}
		if err != nil {
			return nil, entraSigningKeysError{err}
		}
		return key.Key(), nil
	}); err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	if !strings.EqualFold(claims.TenantID, tenantID) {
		return nil, errors.New("tenant mismatch")
	}
	if !strings.EqualFold(claims.Issuer, fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID)) {
		return nil, errors.New("issuer mismatch")
	}
	if !claims.VerifyAudience(clientID, true) {
		return nil, errors.New("audience mismatch")
	}
	// A small leeway absorbs clock skew between Fleet and Entra.
	const leeway = 5 * time.Minute
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(now.Add(-leeway)) {
		return nil, errors.New("token expired")
	}
	if claims.IssuedAt == nil || claims.IssuedAt.After(now.Add(leeway)) || now.Sub(claims.IssuedAt.Time) > entraDeviceTokenMaxAge {
		return nil, errors.New("token issued outside of the accepted window")
	}
	if claims.NotBefore != nil && claims.NotBefore.After(now.Add(leeway)) {
		return nil, errors.New("token not valid yet")
	}
	if _, err := uuid.Parse(claims.DeviceID); err != nil {
		return nil, errors.New("invalid device ID")
	}
	return &claims, nil
}
