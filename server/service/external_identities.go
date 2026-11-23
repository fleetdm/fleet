package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const maxExternalIdentityFieldLen = 255

type listAPIOnlyUserExternalIdentitiesRequest struct {
	ID uint `url:"id"`
}

type externalIdentitiesResponse struct {
	ExternalIdentities []fleet.ExternalIdentity `json:"external_identities"`
	Err                error                    `json:"error,omitempty"`
}

func (r externalIdentitiesResponse) Error() error { return r.Err }

func listAPIOnlyUserExternalIdentitiesEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listAPIOnlyUserExternalIdentitiesRequest)
	ids, err := svc.ListAPIOnlyUserExternalIdentities(ctx, req.ID)
	if err != nil {
		return externalIdentitiesResponse{Err: err}, nil
	}
	return externalIdentitiesResponse{ExternalIdentities: ids}, nil
}

type setAPIOnlyUserExternalIdentitiesRequest struct {
	ID                 uint                     `json:"-" url:"id"`
	ExternalIdentities []fleet.ExternalIdentity `json:"external_identities"`
}

func setAPIOnlyUserExternalIdentitiesEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*setAPIOnlyUserExternalIdentitiesRequest)
	ids, err := svc.SetAPIOnlyUserExternalIdentities(ctx, req.ID, req.ExternalIdentities)
	if err != nil {
		return externalIdentitiesResponse{Err: err}, nil
	}
	return externalIdentitiesResponse{ExternalIdentities: ids}, nil
}

// authorizeExternalIdentityAdmin loads the target API-only user. Binding an external
// identity grants API access as that user, so only global admins may read or write bindings.
func (svc *Service) authorizeExternalIdentityAdmin(ctx context.Context, userID uint, action string) (*fleet.User, error) {
	target, err := svc.ds.UserByID(ctx, userID)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get user")
	}
	if err := svc.authz.Authorize(ctx, target, action); err != nil {
		return nil, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	if vc.User.GlobalRole == nil || *vc.User.GlobalRole != fleet.RoleAdmin {
		return nil, authz.ForbiddenWithInternal("external identities require a global admin", vc.User, target, action)
	}
	if !target.APIOnly {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("id", "target user is not an API-only user"))
	}
	return target, nil
}

func (svc *Service) ListAPIOnlyUserExternalIdentities(ctx context.Context, userID uint) ([]fleet.ExternalIdentity, error) {
	if _, err := svc.authorizeExternalIdentityAdmin(ctx, userID, fleet.ActionRead); err != nil {
		return nil, err
	}
	ids, err := svc.ds.ListUserExternalIdentities(ctx, userID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list external identities")
	}
	return ids, nil
}

func (svc *Service) SetAPIOnlyUserExternalIdentities(ctx context.Context, userID uint, ids []fleet.ExternalIdentity) ([]fleet.ExternalIdentity, error) {
	target, err := svc.authorizeExternalIdentityAdmin(ctx, userID, fleet.ActionWrite)
	if err != nil {
		return nil, err
	}
	if vc, _ := viewer.FromContext(ctx); vc.UserID() == userID {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("id", "cannot modify your own API-only user"))
	}

	configured := make(map[string]struct{}, len(svc.config.Auth.JWTIssuers))
	for _, iss := range svc.config.Auth.JWTIssuers {
		configured[iss.Name] = struct{}{}
	}
	invalid := &fleet.InvalidArgumentError{}
	seen := make(map[fleet.ExternalIdentity]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := configured[id.Issuer]; !ok {
			invalid.Appendf("external_identities", "issuer %q is not a configured auth.jwt_issuers name", id.Issuer)
		}
		if id.Subject == "" || len(id.Subject) > maxExternalIdentityFieldLen {
			invalid.Appendf("external_identities", "subject must be 1 to %d characters", maxExternalIdentityFieldLen)
		}
		if _, dup := seen[id]; dup {
			invalid.Appendf("external_identities", "duplicate identity (%s, %s)", id.Issuer, id.Subject)
		}
		seen[id] = struct{}{}
	}
	if invalid.HasErrors() {
		return nil, ctxerr.Wrap(ctx, invalid)
	}

	added, removed, err := svc.ds.SetUserExternalIdentities(ctx, userID, ids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "set external identities")
	}

	actor := authz.UserFromContext(ctx)
	for _, id := range added {
		if err := svc.NewActivity(ctx, actor, fleet.ActivityTypeAddedUserExternalIdentity{
			UserID: target.ID, UserName: target.Name, UserEmail: target.Email, Issuer: id.Issuer, Subject: id.Subject,
		}); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "create activity")
		}
	}
	for _, id := range removed {
		if err := svc.NewActivity(ctx, actor, fleet.ActivityTypeDeletedUserExternalIdentity{
			UserID: target.ID, UserName: target.Name, UserEmail: target.Email, Issuer: id.Issuer, Subject: id.Subject,
		}); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "create activity")
		}
	}

	// Read back from the primary: a replica may not have the write yet.
	return svc.ds.ListUserExternalIdentities(ctxdb.RequirePrimary(ctx, true), userID)
}
