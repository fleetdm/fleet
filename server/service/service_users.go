package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"time"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/mail"
	"github.com/pkg/errors"
)

func (svc *Service) CreateUserWithInvite(ctx context.Context, p kolide.UserPayload) (*kolide.User, error) {
	// skipauth: There is no viewer context at this point. We rely on verifying
	// the invite for authNZ.
	svc.authz.SkipAuthorization(ctx)

	invite, err := svc.VerifyInvite(ctx, *p.InviteToken)
	if err != nil {
		return nil, err
	}

	// set the payload role property based on an existing invite.
	p.GlobalRole = invite.GlobalRole.Ptr()
	p.Teams = &invite.Teams

	user, err := svc.newUser(p)
	if err != nil {
		return nil, err
	}

	err = svc.ds.DeleteInvite(invite.ID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (svc *Service) CreateUser(ctx context.Context, p kolide.UserPayload) (*kolide.User, error) {
	if err := svc.authz.Authorize(ctx, &kolide.User{}, "write"); err != nil {
		return nil, err
	}

	return svc.newUser(p)
}

func (svc *Service) CreateInitialUser(ctx context.Context, p kolide.UserPayload) (*kolide.User, error) {
	// skipauth: Only the initial user creation should be allowed to skip
	// authorization (because there is not yet a user context to check against).
	svc.authz.SkipAuthorization(ctx)

	users, err := svc.ds.ListUsers(kolide.UserListOptions{ListOptions: kolide.ListOptions{Page: 0, PerPage: 1}})
	if err != nil {
		return nil, err
	}
	if len(users) != 0 {
		return nil, errors.New("a user already exists")
	}

	return svc.newUser(p)
}

func (svc *Service) newUser(p kolide.UserPayload) (*kolide.User, error) {
	var ssoEnabled bool
	// if user is SSO generate a fake password
	if (p.SSOInvite != nil && *p.SSOInvite) || (p.SSOEnabled != nil && *p.SSOEnabled) {
		fakePassword, err := generateRandomText(14)
		if err != nil {
			return nil, errors.Wrap(err, "generate stand-in password")
		}
		p.Password = &fakePassword
		ssoEnabled = true
	}
	user, err := p.User(svc.config.Auth.SaltKeySize, svc.config.Auth.BcryptCost)
	if err != nil {
		return nil, err
	}
	user.SSOEnabled = ssoEnabled
	user, err = svc.ds.NewUser(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (svc *Service) ChangeUserAdmin(ctx context.Context, id uint, isAdmin bool) (*kolide.User, error) {
	// TODO remove this function
	return nil, errors.New("This function is being eliminated")
}

// TODO split modifying basics and role information because they have different
// permissions.
func (svc *Service) ModifyUser(ctx context.Context, userID uint, p kolide.UserPayload) (*kolide.User, error) {
	if err := svc.authz.Authorize(ctx, &kolide.User{ID: userID}, "write"); err != nil {
		return nil, err
	}

	user, err := svc.User(ctx, userID)
	if err != nil {
		return nil, err
	}

	// the method assumes that the correct authorization
	// has been validated higher up the stack
	if p.Username != nil {
		user.Username = *p.Username
	}

	if p.Name != nil {
		user.Name = *p.Name
	}

	if p.Email != nil && *p.Email != user.Email {
		err = svc.modifyEmailAddress(ctx, user, *p.Email, p.Password)
		if err != nil {
			return nil, err
		}
	}

	if p.Position != nil {
		user.Position = *p.Position
	}

	if p.GravatarURL != nil {
		user.GravatarURL = *p.GravatarURL
	}

	if p.SSOEnabled != nil {
		user.SSOEnabled = *p.SSOEnabled
	}

	if p.GlobalRole != nil && *p.GlobalRole != "" {
		if p.Teams != nil && len(*p.Teams) > 0 {
			return nil, kolide.NewInvalidArgumentError("teams", "may not be specified with global_role")
		}
		user.GlobalRole = p.GlobalRole
		user.Teams = []kolide.UserTeam{}
	} else if p.Teams != nil {
		user.Teams = *p.Teams
		user.GlobalRole = nil
	}

	err = svc.saveUser(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (svc *Service) modifyEmailAddress(ctx context.Context, user *kolide.User, email string, password *string) error {
	// password requirement handled in validation middleware
	if password != nil {
		err := user.ValidatePassword(*password)
		if err != nil {
			return kolide.NewPermissionError("incorrect password")
		}
	}
	random, err := kolide.RandomText(svc.config.App.TokenKeySize)
	if err != nil {
		return err
	}
	token := base64.URLEncoding.EncodeToString([]byte(random))
	err = svc.ds.PendingEmailChange(user.ID, email, token)
	if err != nil {
		return err
	}
	config, err := svc.AppConfig(ctx)
	if err != nil {
		return err
	}
	changeEmail := kolide.Email{
		Subject: "Confirm Fleet Email Change",
		To:      []string{email},
		Config:  config,
		Mailer: &mail.ChangeEmailMailer{
			Token:    token,
			BaseURL:  template.URL(config.KolideServerURL + svc.config.Server.URLPrefix),
			AssetURL: getAssetURL(),
		},
	}
	return svc.mailService.SendEmail(changeEmail)
}

func (svc *Service) DeleteUser(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &kolide.User{ID: id}, "write"); err != nil {
		return err
	}

	return svc.ds.DeleteUser(id)
}

func (svc *Service) ChangeUserEmail(ctx context.Context, token string) (string, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return "", kolide.ErrNoContext
	}

	if err := svc.authz.Authorize(ctx, &kolide.User{ID: vc.UserID()}, "write"); err != nil {
		return "", err
	}

	return svc.ds.ConfirmPendingEmailChange(vc.UserID(), token)
}

func (svc *Service) User(ctx context.Context, id uint) (*kolide.User, error) {
	if err := svc.authz.Authorize(ctx, &kolide.User{ID: id}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.UserByID(id)
}

func (svc *Service) UserUnauthorized(ctx context.Context, id uint) (*kolide.User, error) {
	// Explicitly no authorization check. Should only be used by middleware.
	return svc.ds.UserByID(id)
}

func (svc *Service) AuthenticatedUser(ctx context.Context) (*kolide.User, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, kolide.ErrNoContext
	}

	if err := svc.authz.Authorize(ctx, &kolide.User{ID: vc.UserID()}, "read"); err != nil {
		return nil, err
	}

	if !vc.IsLoggedIn() {
		return nil, kolide.NewPermissionError("not logged in")
	}
	return vc.User, nil
}

func (svc *Service) ListUsers(ctx context.Context, opt kolide.UserListOptions) ([]*kolide.User, error) {
	if err := svc.authz.Authorize(ctx, &kolide.User{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.ListUsers(opt)
}

// setNewPassword is a helper for changing a user's password. It should be
// called to set the new password after proper authorization has been
// performed.
func (svc *Service) setNewPassword(ctx context.Context, user *kolide.User, password string) error {
	err := user.SetPassword(password, svc.config.Auth.SaltKeySize, svc.config.Auth.BcryptCost)
	if err != nil {
		return errors.Wrap(err, "setting new password")
	}
	if user.SSOEnabled {
		return errors.New("set password for single sign on user not allowed")
	}
	err = svc.saveUser(user)
	if err != nil {
		return errors.Wrap(err, "saving changed password")
	}

	return nil
}

func (svc *Service) ChangePassword(ctx context.Context, oldPass, newPass string) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return kolide.ErrNoContext
	}

	if err := svc.authz.Authorize(ctx, vc.User, "write"); err != nil {
		return err
	}

	if vc.User.SSOEnabled {
		return errors.New("change password for single sign on user not allowed")
	}
	if err := vc.User.ValidatePassword(newPass); err == nil {
		return kolide.NewInvalidArgumentError("new_password", "cannot reuse old password")
	}

	if err := vc.User.ValidatePassword(oldPass); err != nil {
		return kolide.NewInvalidArgumentError("old_password", "old password does not match")
	}

	if err := svc.setNewPassword(ctx, vc.User, newPass); err != nil {
		return errors.Wrap(err, "setting new password")
	}
	return nil
}

func (svc *Service) ResetPassword(ctx context.Context, token, password string) error {
	reset, err := svc.ds.FindPassswordResetByToken(token)
	if err != nil {
		return errors.Wrap(err, "looking up reset by token")
	}
	user, err := svc.User(ctx, reset.UserID)
	if err != nil {
		return errors.Wrap(err, "retrieving user")
	}

	if err := svc.authz.Authorize(ctx, user, "write"); err != nil {
		return err
	}

	if user.SSOEnabled {
		return errors.New("password reset for single sign on user not allowed")
	}

	// prevent setting the same password
	if err := user.ValidatePassword(password); err == nil {
		return kolide.NewInvalidArgumentError("new_password", "cannot reuse old password")
	}

	err = svc.setNewPassword(ctx, user, password)
	if err != nil {
		return errors.Wrap(err, "setting new password")
	}

	// delete password reset tokens for user
	if err := svc.ds.DeletePasswordResetRequestsForUser(user.ID); err != nil {
		return errors.Wrap(err, "deleting password reset requests")
	}

	// Clear sessions so that any other browsers will have to log in with
	// the new password
	if err := svc.DeleteSessionsForUser(ctx, user.ID); err != nil {
		return errors.Wrap(err, "deleting user sessions")
	}

	return nil
}

func (svc *Service) PerformRequiredPasswordReset(ctx context.Context, password string) (*kolide.User, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, kolide.ErrNoContext
	}
	user := vc.User

	if err := svc.authz.Authorize(ctx, user, "write"); err != nil {
		return nil, err
	}

	if user.SSOEnabled {
		return nil, errors.New("password reset for single sign on user not allowed")
	}
	if !user.AdminForcedPasswordReset {
		return nil, errors.New("user does not require password reset")
	}

	// prevent setting the same password
	if err := user.ValidatePassword(password); err == nil {
		return nil, kolide.NewInvalidArgumentError("new_password", "cannot reuse old password")
	}

	user.AdminForcedPasswordReset = false
	err := svc.setNewPassword(ctx, user, password)
	if err != nil {
		return nil, errors.Wrap(err, "setting new password")
	}

	// Sessions should already have been cleared when the reset was
	// required

	return user, nil
}

func (svc *Service) RequirePasswordReset(ctx context.Context, uid uint, require bool) (*kolide.User, error) {
	if err := svc.authz.Authorize(ctx, &kolide.User{ID: uid}, "write"); err != nil {
		return nil, err
	}

	user, err := svc.ds.UserByID(uid)
	if err != nil {
		return nil, errors.Wrap(err, "loading user by ID")
	}
	if user.SSOEnabled {
		return nil, errors.New("password reset for single sign on user not allowed")
	}
	// Require reset on next login
	user.AdminForcedPasswordReset = require
	if err := svc.saveUser(user); err != nil {
		return nil, errors.Wrap(err, "saving user")
	}

	if require {
		// Clear all of the existing sessions
		if err := svc.DeleteSessionsForUser(ctx, user.ID); err != nil {
			return nil, errors.Wrap(err, "deleting user sessions")
		}
	}

	return user, nil
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

	user, err := svc.ds.UserByEmail(email)
	if err != nil {
		return err
	}
	if user.SSOEnabled {
		return errors.New("password reset for single sign on user not allowed")
	}

	random, err := kolide.RandomText(svc.config.App.TokenKeySize)
	if err != nil {
		return err
	}
	token := base64.URLEncoding.EncodeToString([]byte(random))

	request := &kolide.PasswordResetRequest{
		ExpiresAt: time.Now().Add(time.Hour * 24),
		UserID:    user.ID,
		Token:     token,
	}
	_, err = svc.ds.NewPasswordResetRequest(request)
	if err != nil {
		return err
	}

	config, err := svc.AppConfig(ctx)
	if err != nil {
		return err
	}

	resetEmail := kolide.Email{
		Subject: "Reset Your Fleet Password",
		To:      []string{user.Email},
		Config:  config,
		Mailer: &mail.PasswordResetMailer{
			BaseURL:  template.URL(config.KolideServerURL + svc.config.Server.URLPrefix),
			AssetURL: getAssetURL(),
			Token:    token,
		},
	}

	return svc.mailService.SendEmail(resetEmail)
}

// saves user in datastore.
// doesn't need to be exposed to the transport
// the service should expose actions for modifying a user instead
func (svc *Service) saveUser(user *kolide.User) error {
	return svc.ds.SaveUser(user)
}

// generateRandomText return a string generated by filling in keySize bytes with
// random data and then base64 encoding those bytes
func generateRandomText(keySize int) (string, error) {
	key := make([]byte, keySize)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
