package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func (svc *Service) CreateInitialUser(ctx context.Context, p fleet.UserPayload) (*fleet.User, error) {
	// skipauth: Only the initial user creation should be allowed to skip
	// authorization (because there is not yet a user context to check against).
	svc.authz.SkipAuthorization(ctx)

	setupRequired, err := svc.SetupRequired(ctx)
	if err != nil {
		return nil, err
	}
	if !setupRequired {
		return nil, ctxerr.New(ctx, "a user already exists")
	}

	// Initial user should be global admin with no explicit teams
	p.GlobalRole = ptr.String(fleet.RoleAdmin)
	p.Teams = nil

	return svc.newUser(ctx, p)
}

func (svc *Service) newUser(ctx context.Context, p fleet.UserPayload) (*fleet.User, error) {
	var ssoEnabled bool
	// if user is SSO generate a fake password
	if (p.SSOInvite != nil && *p.SSOInvite) || (p.SSOEnabled != nil && *p.SSOEnabled) {
		fakePassword, err := server.GenerateRandomText(14)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "generate stand-in password")
		}
		p.Password = &fakePassword
		ssoEnabled = true
	}
	user, err := p.User(svc.config.Auth.SaltKeySize, svc.config.Auth.BcryptCost)
	if err != nil {
		return nil, err
	}
	user.SSOEnabled = ssoEnabled
	user, err = svc.ds.NewUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (svc *Service) UserUnauthorized(ctx context.Context, id uint) (*fleet.User, error) {
	// Explicitly no authorization check. Should only be used by middleware.
	return svc.ds.UserByID(ctx, id)
}
