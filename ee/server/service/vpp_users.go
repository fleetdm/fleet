package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	"github.com/google/uuid"
)

// errMissingManagedAppleID is returned when ensureVPPClientUser is called for
// a user-enrolled host whose Managed Apple ID hasn't yet been surfaced from
// nanomdm's TokenUpdate hook. Callers should surface a retryable user-facing
// message — the value is normally available a few minutes after enrollment.
var errMissingManagedAppleID = fleet.NewUserMessageError(
	errors.New("Couldn't install. Fleet hasn't received a Managed Apple ID for this host yet. Please wait a few minutes after enrollment and try again."),
	http.StatusUnprocessableEntity,
)

// ensureVPPClientUser returns the Fleet-generated clientUserId for the host's
// Managed Apple ID at the given VPP token (location), creating the Apple-side
// VPP user via Apple's synchronous v1 registerVPPUserSrv endpoint on first
// call. Idempotent: subsequent calls return the cached clientUserId from
// vpp_client_users.
//
// Used by the user-scoped Associate Assets path for hosts enrolled via
// Account-Driven User Enrollment (BYOD).
func (svc *Service) ensureVPPClientUser(ctx context.Context, host *fleet.Host, token *fleet.VPPTokenDB) (string, error) {
	if host == nil {
		return "", ctxerr.New(ctx, "ensureVPPClientUser: nil host")
	}
	if token == nil {
		return "", ctxerr.New(ctx, "ensureVPPClientUser: nil token")
	}

	managedAppleID, err := svc.ds.GetHostManagedAppleID(ctx, host.ID)
	if err != nil {
		return "", ctxerr.Wrapf(ctx, err, "looking up managed apple id for host %d", host.ID)
	}
	if managedAppleID == "" {
		return "", errMissingManagedAppleID
	}

	// Cache hit on (vpp_token_id, managed_apple_id): a previous successful call
	// already registered this user with Apple.
	existing, err := svc.ds.GetVPPClientUser(ctx, token.ID, managedAppleID)
	if err != nil && !fleet.IsNotFound(err) {
		return "", ctxerr.Wrapf(ctx, err, "looking up vpp client user for token %d managed_apple_id %q", token.ID, managedAppleID)
	}
	if existing != nil && existing.Status == fleet.VPPClientUserStatusRegistered {
		return existing.ClientUserID, nil
	}

	// Non-registered row. Apple enforces uniqueness on
	// (location, managedAppleId), so blindly calling
	// registerVPPClientUser with a fresh UUID will collide with any existing
	// Apple-side user. Ask Apple first; if a user already exists, resync the
	// local cache to its clientUserId rather than minting a new one.
	if existing != nil {
		appleUser, lookupErr := vpp.GetUserByManagedAppleID(ctx, token.Token, managedAppleID)
		if lookupErr != nil {
			return "", ctxerr.Wrapf(ctx, lookupErr, "looking up vpp user by managed apple id for token %d", token.ID)
		}
		if appleUser != nil {
			row := &fleet.VPPClientUser{
				VPPTokenID:     token.ID,
				ManagedAppleID: managedAppleID,
				ClientUserID:   appleUser.ClientUserID,
				Status:         fleet.VPPClientUserStatusRegistered,
			}
			if err := svc.ds.InsertVPPClientUser(ctx, row); err != nil {
				return "", ctxerr.Wrap(ctx, err, "resyncing vpp client user cache from Apple")
			}
			return appleUser.ClientUserID, nil
		}
	}

	return svc.registerVPPClientUser(ctx, token.ID, managedAppleID, token.Token)
}

// registerVPPClientUser unconditionally registers a new VPP user via Apple's
// synchronous v1 endpoint and upserts the (vpp_token_id, managed_apple_id)
// row with the freshly-generated clientUserId, overwriting any prior cache
// entry. Used by:
//
//   - ensureVPPClientUser on its first-call / cache-miss branch.
//   - The install-flow self-heal path, when Apple rejects the cached
//     clientUserId as unknown — bypassing the cache is the whole point of the
//     retry, so this entry point exists to avoid a confusing 'force' flag on
//     ensureVPPClientUser.
func (svc *Service) registerVPPClientUser(ctx context.Context, tokenID uint, managedAppleID, token string) (string, error) {
	clientUserID := uuid.NewString()

	// v1 registerVPPUserSrv is synchronous — a successful response means the
	// user is registered and ready to receive license associations.
	appleUserID, err := vpp.RegisterUser(token, clientUserID, managedAppleID)
	if err != nil {
		return "", ctxerr.Wrapf(ctx, err, "registering vpp user for managed apple id %q", managedAppleID)
	}

	row := &fleet.VPPClientUser{
		VPPTokenID:     tokenID,
		ManagedAppleID: managedAppleID,
		ClientUserID:   clientUserID,
		Status:         fleet.VPPClientUserStatusRegistered,
	}
	if appleUserID != "" {
		row.AppleUserID = &appleUserID
	}
	if err := svc.ds.InsertVPPClientUser(ctx, row); err != nil {
		return "", ctxerr.Wrap(ctx, err, "persisting registered vpp client user")
	}
	return clientUserID, nil
}
