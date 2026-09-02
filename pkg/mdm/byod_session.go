package mdm

import (
	"context"
	"encoding/json"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	byodIdPSessionKeyPrefix = "byod_idp_session:"
	byodIdPSessionIDLength  = 24
	// BYODIdPSessionTTL bounds how long an end user has between finishing IdP
	// authentication and downloading the enrollment profile.
	BYODIdPSessionTTL = 30 * time.Minute
)

// BYODIdPSession is minted after a successful IdP callback for the BYOD /enroll
// flow. The cookie carries only the session id; the IdP account lives here.
type BYODIdPSession struct {
	IdPAccountUUID string    `json:"idp_account_uuid"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// CreateBYODIdPSession returns the opaque id to hand back in the cookie.
func CreateBYODIdPSession(ctx context.Context, kv fleet.KeyValueStore, clk clock.Clock, idpAccountUUID string) (string, error) {
	if kv == nil {
		return "", ctxerr.New(ctx, "byod idp session store not configured")
	}
	sessionID, err := server.GenerateRandomURLSafeText(byodIdPSessionIDLength)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "generate byod idp session id")
	}
	b, err := json.Marshal(BYODIdPSession{
		IdPAccountUUID: idpAccountUUID,
		ExpiresAt:      clk.Now().Add(BYODIdPSessionTTL),
	})
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "marshal byod idp session")
	}
	if err := kv.Set(ctx, byodIdPSessionKeyPrefix+sessionID, string(b), BYODIdPSessionTTL); err != nil {
		return "", ctxerr.Wrap(ctx, err, "store byod idp session")
	}
	return sessionID, nil
}

// ValidateBYODIdPSession returns an AuthRequiredError for an unknown or expired
// session; any other error is the store failing.
func ValidateBYODIdPSession(ctx context.Context, kv fleet.KeyValueStore, clk clock.Clock, sessionID string) (string, error) {
	if kv == nil {
		return "", ctxerr.New(ctx, "byod idp session store not configured")
	}
	if sessionID == "" {
		return "", ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("byod idp session not found"))
	}
	val, err := kv.Get(ctx, byodIdPSessionKeyPrefix+sessionID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get byod idp session")
	}
	if val == nil {
		return "", ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("byod idp session not found"))
	}
	var session BYODIdPSession
	if err := json.Unmarshal([]byte(*val), &session); err != nil {
		return "", ctxerr.Wrap(ctx, err, "unmarshal byod idp session")
	}
	if !clk.Now().Before(session.ExpiresAt) {
		return "", ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("byod idp session expired"))
	}
	return session.IdPAccountUUID, nil
}
