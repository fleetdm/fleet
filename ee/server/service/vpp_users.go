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
// VPP user via POST /mdm/v2/users/create on first call. Idempotent: subsequent
// calls return the cached clientUserId from vpp_client_users.
//
// Used by the user-scoped Associate Assets path (subtask 06) for hosts enrolled
// via Account-Driven User Enrollment (BYOD).
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

	// Either no row, or a prior attempt left it 'pending'. Reuse the
	// previously-generated UUID on retry so Apple correlates the request with
	// the same user record.
	clientUserID := uuid.NewString()
	if existing != nil && existing.ClientUserID != "" {
		clientUserID = existing.ClientUserID
	}

	resp, err := vpp.CreateUsers(token.Token, &vpp.CreateUsersRequest{
		Users: []vpp.CreateUsersUser{{ClientUserId: clientUserID, ManagedAppleId: managedAppleID}},
	})
	if err != nil {
		// Persist 'pending' so a future retry can reuse the same clientUserID
		// rather than minting a fresh one (which would leave us with multiple
		// Apple-side users for the same Managed Apple ID). Log if the persist
		// itself fails — we still want to surface the original CreateUsers
		// error to the caller.
		if insertErr := svc.ds.InsertVPPClientUser(ctx, &fleet.VPPClientUser{
			VPPTokenID:     token.ID,
			ManagedAppleID: managedAppleID,
			ClientUserID:   clientUserID,
			Status:         fleet.VPPClientUserStatusPending,
		}); insertErr != nil {
			svc.logger.ErrorContext(ctx, "persisting pending vpp client user after CreateUsers failure",
				"host_id", host.ID, "vpp_token_id", token.ID, "err", insertErr)
		}
		return "", ctxerr.Wrap(ctx, err, "calling Apple VPP create-users")
	}

	// Apple's /users/create is asynchronous in the v2 API: a 200 with eventId
	// means user registration has been queued, and the per-user payload is
	// returned later (separate /users/get poll, deferred to a follow-up
	// subtask). If Apple did echo a per-user entry in the synchronous response,
	// surface any per-user error; otherwise treat the eventId as success since
	// downstream associate-assets uses our clientUserId, which Apple resolves
	// once registration completes.
	for i := range resp.Users {
		u := &resp.Users[i]
		if u.ClientUserId != clientUserID {
			continue
		}
		if u.HasError() {
			if insertErr := svc.ds.InsertVPPClientUser(ctx, &fleet.VPPClientUser{
				VPPTokenID:     token.ID,
				ManagedAppleID: managedAppleID,
				ClientUserID:   clientUserID,
				Status:         fleet.VPPClientUserStatusPending,
			}); insertErr != nil {
				svc.logger.ErrorContext(ctx, "persisting pending vpp client user after Apple per-user error",
					"host_id", host.ID, "vpp_token_id", token.ID, "err", insertErr)
			}
			return "", ctxerr.Errorf(ctx, "Apple VPP create-users returned error for managed apple id %q: %s (code %d)",
				managedAppleID, u.ErrorMessage, u.ErrorNumber)
		}
	}

	row := &fleet.VPPClientUser{
		VPPTokenID:     token.ID,
		ManagedAppleID: managedAppleID,
		ClientUserID:   clientUserID,
		Status:         fleet.VPPClientUserStatusRegistered,
	}
	for i := range resp.Users {
		if resp.Users[i].ClientUserId == clientUserID && resp.Users[i].UserId != "" {
			appleUserID := resp.Users[i].UserId
			row.AppleUserID = &appleUserID
			break
		}
	}
	if err := svc.ds.InsertVPPClientUser(ctx, row); err != nil {
		return "", ctxerr.Wrap(ctx, err, "persisting registered vpp client user")
	}
	return clientUserID, nil
}
