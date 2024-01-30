package service

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"golang.org/x/crypto/bcrypt"
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

	return svc.NewUser(ctx, p)
}

func (svc *Service) NewUser(ctx context.Context, p fleet.UserPayload) (*fleet.User, error) {
	license, _ := license.FromContext(ctx)
	if license == nil {
		return nil, ctxerr.New(ctx, "license not found")
	}
	if err := fleet.ValidateUserRoles(true, p, *license); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validate role")
	}

	user, err := p.User(svc.config.Auth.SaltKeySize, svc.config.Auth.BcryptCost)
	if err != nil {
		if errors.Is(err, bcrypt.ErrPasswordTooLong) {
			return nil, ctxerr.Wrap(ctx, badRequestErr("Password is over the 48 characters limit. If the password is under 48 characters, please check the auth_salt_key_size in your Fleet server config.", err))
		}
		return nil, err
	}

	user, err = svc.ds.NewUser(ctx, user)
	if err != nil {
		return nil, err
	}

	adminUser := authz.UserFromContext(ctx)
	if adminUser == nil {
		// In case of invites the user created herself.
		adminUser = user
	}
	if err := svc.ds.NewActivity(
		ctx,
		adminUser,
		fleet.ActivityTypeCreatedUser{
			UserID:    user.ID,
			UserName:  user.Name,
			UserEmail: user.Email,
		},
	); err != nil {
		return nil, err
	}
	if err := fleet.LogRoleChangeActivities(ctx, svc.ds, adminUser, nil, nil, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (svc *Service) UserUnauthorized(ctx context.Context, id uint) (*fleet.User, error) {
	// Explicitly no authorization check. Should only be used by middleware.
	return svc.ds.UserByID(ctx, id)
}
