package service

import (
	"context"
	"encoding/base64"
	"html/template"
	"time"

	"github.com/fleetdm/fleet/v4/server"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mail"
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

func (svc *Service) RequestPasswordReset(ctx context.Context, email string) error {
	// skipauth: No viewer context available. The user is locked out of their
	// account and trying to reset their password.
	svc.authz.SkipAuthorization(ctx)

	// Regardless of error, sleep until the request has taken at least 1 second.
	// This means that any request to this method will take ~1s and frustrate a timing attack.
	defer func(start time.Time) {
		time.Sleep(time.Until(start.Add(1 * time.Second)))
	}(time.Now())

	user, err := svc.ds.UserByEmail(ctx, email)
	if err != nil {
		return err
	}
	if user.SSOEnabled {
		return ctxerr.New(ctx, "password reset for single sign on user not allowed")
	}

	random, err := server.GenerateRandomText(svc.config.App.TokenKeySize)
	if err != nil {
		return err
	}
	token := base64.URLEncoding.EncodeToString([]byte(random))

	request := &fleet.PasswordResetRequest{
		ExpiresAt: time.Now().Add(time.Hour * 24),
		UserID:    user.ID,
		Token:     token,
	}
	_, err = svc.ds.NewPasswordResetRequest(ctx, request)
	if err != nil {
		return err
	}

	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}

	resetEmail := fleet.Email{
		Subject: "Reset Your Fleet Password",
		To:      []string{user.Email},
		Config:  config,
		Mailer: &mail.PasswordResetMailer{
			BaseURL:  template.URL(config.ServerSettings.ServerURL + svc.config.Server.URLPrefix),
			AssetURL: getAssetURL(),
			Token:    token,
		},
	}

	return svc.mailService.SendEmail(resetEmail)
}
