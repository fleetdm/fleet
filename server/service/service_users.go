package service

import (
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"time"

	"github.com/kolide/kolide/server/contexts/viewer"
	"github.com/kolide/kolide/server/kolide"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

func (svc service) NewUser(ctx context.Context, p kolide.UserPayload) (*kolide.User, error) {
	invite, err := svc.VerifyInvite(ctx, *p.InviteToken)
	if err != nil {
		return nil, err
	}

	// set the payload Admin property based on an existing invite.
	p.Admin = &invite.Admin

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

func (svc service) NewAdminCreatedUser(ctx context.Context, p kolide.UserPayload) (*kolide.User, error) {
	return svc.newUser(p)
}

func (svc service) newUser(p kolide.UserPayload) (*kolide.User, error) {
	user, err := p.User(svc.config.Auth.SaltKeySize, svc.config.Auth.BcryptCost)
	if err != nil {
		return nil, err
	}
	user, err = svc.ds.NewUser(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (svc service) ChangeUserAdmin(ctx context.Context, id uint, isAdmin bool) (*kolide.User, error) {
	user, err := svc.ds.UserByID(id)
	if err != nil {
		return nil, err
	}
	user.Admin = isAdmin
	if err = svc.saveUser(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (svc service) ChangeUserEnabled(ctx context.Context, id uint, isEnabled bool) (*kolide.User, error) {
	user, err := svc.ds.UserByID(id)
	if err != nil {
		return nil, err
	}
	user.Enabled = isEnabled
	if err = svc.saveUser(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (svc service) ModifyUser(ctx context.Context, userID uint, p kolide.UserPayload) (*kolide.User, error) {
	user, err := svc.User(ctx, userID)
	if err != nil {
		return nil, err
	}

	// the method assumes that the correct authorization
	// has been validated higher up the stack
	if p.Admin != nil {
		user.Admin = *p.Admin
	}

	if p.Enabled != nil {
		user.Enabled = *p.Enabled
	}

	if p.Username != nil {
		user.Username = *p.Username
	}

	if p.Name != nil {
		user.Name = *p.Name
	}

	if p.Email != nil {
		err = svc.modifyEmailAddress(ctx, user, *p.Email, p.Password)
		if err != nil {
			return nil, err
		}
		return user, nil
	}

	if p.Position != nil {
		user.Position = *p.Position
	}

	if p.GravatarURL != nil {
		user.GravatarURL = *p.GravatarURL
	}

	err = svc.saveUser(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (svc service) modifyEmailAddress(ctx context.Context, user *kolide.User, email string, password *string) error {
	// password requirement handled in validation middleware
	if password != nil {
		err := user.ValidatePassword(*password)
		if err != nil {
			return permissionError{message: "incorrect password"}
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
		Subject: "Confirm Kolide Email Change",
		To:      []string{email},
		Config:  config,
		Mailer: &kolide.ChangeEmailMailer{
			Token:           token,
			KolideServerURL: template.URL(config.KolideServerURL),
		},
	}
	err = svc.mailService.SendEmail(changeEmail)
	if err != nil {
		return err
	}
	return nil
}

func (svc service) ChangeUserEmail(ctx context.Context, token string) (string, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return "", errNoContext
	}
	return svc.ds.ConfirmPendingEmailChange(vc.UserID(), token)
}

func (svc service) User(ctx context.Context, id uint) (*kolide.User, error) {
	return svc.ds.UserByID(id)
}

func (svc service) AuthenticatedUser(ctx context.Context) (*kolide.User, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errNoContext
	}
	if !vc.IsLoggedIn() {
		return nil, permissionError{}
	}
	return vc.User, nil
}

func (svc service) ListUsers(ctx context.Context, opt kolide.ListOptions) ([]*kolide.User, error) {
	return svc.ds.ListUsers(opt)
}

// setNewPassword is a helper for changing a user's password. It should be
// called to set the new password after proper authorization has been
// performed.
func (svc service) setNewPassword(ctx context.Context, user *kolide.User, password string) error {
	err := user.SetPassword(password, svc.config.Auth.SaltKeySize, svc.config.Auth.BcryptCost)
	if err != nil {
		return errors.Wrap(err, "setting new password")
	}

	err = svc.saveUser(user)
	if err != nil {
		return errors.Wrap(err, "saving changed password")
	}

	return nil
}

func (svc service) ChangePassword(ctx context.Context, oldPass, newPass string) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return errNoContext
	}

	if err := vc.User.ValidatePassword(newPass); err == nil {
		return newInvalidArgumentError("new_password", "cannot reuse old password")
	}

	if err := vc.User.ValidatePassword(oldPass); err != nil {
		return newInvalidArgumentError("old_password", "old password does not match")
	}

	if err := svc.setNewPassword(ctx, vc.User, newPass); err != nil {
		return errors.Wrap(err, "setting new password")
	}
	return nil
}

func (svc service) ResetPassword(ctx context.Context, token, password string) error {
	reset, err := svc.ds.FindPassswordResetByToken(token)
	if err != nil {
		return errors.Wrap(err, "looking up reset by token")
	}
	user, err := svc.User(ctx, reset.UserID)
	if err != nil {
		return errors.Wrap(err, "retrieving user")
	}

	// prevent setting the same password
	if err := user.ValidatePassword(password); err == nil {
		return newInvalidArgumentError("new_password", "cannot reuse old password")
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

func (svc service) PerformRequiredPasswordReset(ctx context.Context, password string) (*kolide.User, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errNoContext
	}
	user := vc.User

	if !user.AdminForcedPasswordReset {
		return nil, errors.New("user does not require password reset")
	}

	// prevent setting the same password
	if err := user.ValidatePassword(password); err == nil {
		return nil, newInvalidArgumentError("new_password", "cannot reuse old password")
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

func (svc service) RequirePasswordReset(ctx context.Context, uid uint, require bool) (*kolide.User, error) {
	user, err := svc.ds.UserByID(uid)
	if err != nil {
		return nil, errors.Wrap(err, "loading user by ID")
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

func (svc service) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := svc.ds.UserByEmail(email)
	if err != nil {
		return err
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
	request, err = svc.ds.NewPasswordResetRequest(request)
	if err != nil {
		return err
	}

	config, err := svc.AppConfig(ctx)
	if err != nil {
		return err
	}

	resetEmail := kolide.Email{
		Subject: "Reset Your Kolide Password",
		To:      []string{user.Email},
		Config:  config,
		Mailer: &kolide.PasswordResetMailer{
			KolideServerURL: template.URL(config.KolideServerURL),
			Token:           token,
		},
	}

	err = svc.mailService.SendEmail(resetEmail)
	if err != nil {
		return err
	}

	return nil
}

// saves user in datastore.
// doesn't need to be exposed to the transport
// the service should expose actions for modifying a user instead
func (svc service) saveUser(user *kolide.User) error {
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
