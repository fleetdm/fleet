package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func (svc *Service) CompleteInitialSetup(ctx context.Context, p fleet.UserPayload, appConfig fleet.AppConfig) (*fleet.User, *fleet.AppConfig, error) {
	// skipauth: No user context exists before the first user is created.
	svc.authz.SkipAuthorization(ctx)

	if err := validateSetupServerURL(ctx, appConfig.ServerSettings.ServerURL); err != nil {
		return nil, nil, err
	}

	secret, err := server.GenerateRandomText(fleet.EnrollSecretDefaultLength)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "generate enroll secret string")
	}

	// Initial user should be global admin with no explicit teams
	p.GlobalRole = ptr.String(fleet.RoleAdmin)
	p.Teams = nil

	admin, err := svc.newUser(ctx, p, func(ctx context.Context, user *fleet.User) (*fleet.User, error) {
		return svc.ds.CompleteInitialSetup(ctx, user, &appConfig, []*fleet.EnrollSecret{{Secret: secret}})
	})
	if err != nil {
		return nil, nil, err
	}
	return admin, &appConfig, nil
}

func (svc *Service) NewUser(ctx context.Context, p fleet.UserPayload) (*fleet.User, error) {
	return svc.newUser(ctx, p, svc.ds.NewUser)
}

func (svc *Service) newUser(ctx context.Context, p fleet.UserPayload, create func(context.Context, *fleet.User) (*fleet.User, error)) (*fleet.User, error) {
	licChecker, _ := license.FromContext(ctx)
	lic, _ := licChecker.(*fleet.LicenseInfo)
	if lic == nil {
		return nil, ctxerr.New(ctx, "license not found")
	}
	if err := fleet.ValidateUserRoles(true, p, *lic); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validate role")
	}
	if !lic.IsPremium() {
		p.MFAEnabled = ptr.Bool(false)
	}

	user, err := p.User(svc.config.Auth.SaltKeySize, svc.config.Auth.BcryptCost)
	if err != nil {
		return nil, err
	}

	user, err = create(ctx, user)
	if err != nil {
		return nil, err
	}

	adminUser := authz.UserFromContext(ctx)
	if adminUser == nil {
		// In case of invites the user created herself.
		adminUser = user
	}
	if err := svc.NewActivity(
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
	if err := fleet.LogRoleChangeActivities(ctx, svc, adminUser, nil, nil, user, p.JITProvisioned); err != nil {
		return nil, err
	}

	return user, nil
}

func (svc *Service) UserUnauthorized(ctx context.Context, id uint) (*fleet.User, error) {
	// Explicitly no authorization check. Should only be used by middleware.
	return svc.ds.UserByID(ctx, id)
}
