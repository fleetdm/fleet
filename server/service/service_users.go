package service

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/kolide/kolide-ose/server/contexts/viewer"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (svc service) NewUser(ctx context.Context, p kolide.UserPayload) (*kolide.User, error) {
	err := svc.VerifyInvite(ctx, *p.Email, *p.InviteToken)
	if err != nil {
		return nil, err
	}
	invite, err := svc.ds.InviteByEmail(*p.Email)
	if err != nil {
		return nil, err
	}
	user, err := svc.newUser(p)
	if err != nil {
		return nil, err
	}
	err = svc.ds.DeleteInvite(invite)
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

func (svc service) ModifyUser(ctx context.Context, userID uint, p kolide.UserPayload) (*kolide.User, error) {
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

	if p.Admin != nil {
		user.Admin = *p.Admin
	}

	if p.Email != nil {
		user.Email = *p.Email
	}

	if p.Enabled != nil {
		user.Enabled = *p.Enabled
	}

	if p.Position != nil {
		user.Position = *p.Position
	}

	if p.GravatarURL != nil {
		user.GravatarURL = *p.GravatarURL
	}

	if p.Password != nil {
		err := user.SetPassword(
			*p.Password,
			svc.config.Auth.SaltKeySize,
			svc.config.Auth.BcryptCost,
		)
		if err != nil {
			return nil, err
		}
		user.AdminForcedPasswordReset = false
	}

	err = svc.saveUser(user)
	if err != nil {
		return nil, err
	}

	// https://github.com/kolide/kolide-ose/issues/351
	// Calling this action last, because svc.RequestPasswordReset saves the
	// user separately and we don't want to override the value set there
	if p.AdminForcedPasswordReset != nil && *p.AdminForcedPasswordReset {
		err = svc.RequestPasswordReset(ctx, user.Email)
		if err != nil {
			return nil, err
		}
	}
	return svc.User(ctx, userID)
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

func (svc service) ResetPassword(ctx context.Context, token, password string) error {
	reset, err := svc.ds.FindPassswordResetByToken(token)
	if err != nil {
		return err
	}
	user, err := svc.User(ctx, reset.UserID)
	if err != nil {
		return err
	}

	err = user.SetPassword(password, svc.config.Auth.SaltKeySize, svc.config.Auth.BcryptCost)
	if err != nil {
		return err
	}
	if err := svc.saveUser(user); err != nil {
		return err
	}

	// delete password reset tokens for user
	if err := svc.ds.DeletePasswordResetRequestsForUser(user.ID); err != nil {
		return err
	}

	return nil
}

func (svc service) RequestPasswordReset(ctx context.Context, email string) error {
	// the password reset is different depending on whether performed by an
	// admin or a user
	// if an admin requests a password reset, then no token is
	// generated, instead the AdminForcedPasswordReset flag is set
	user, err := svc.ds.UserByEmail(email)
	if err != nil {
		return err
	}
	vc, ok := viewer.FromContext(ctx)
	if ok {
		if vc.IsAdmin() {
			user.AdminForcedPasswordReset = true
			if err := svc.saveUser(user); err != nil {
				return err
			}
			if err := svc.DeleteSessionsForUser(ctx, user.ID); err != nil {
				return err
			}
			return nil
		}
	}

	token, err := jwt.New(jwt.SigningMethodHS256).SignedString([]byte(svc.config.App.TokenKey))
	if err != nil {
		return err
	}

	request := &kolide.PasswordResetRequest{
		UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
			CreateTimestamp: kolide.CreateTimestamp{
				CreatedAt: time.Now(),
			},
			UpdateTimestamp: kolide.UpdateTimestamp{
				UpdatedAt: time.Now(),
			},
		},
		ExpiresAt: time.Now().Add(time.Hour * 24),
		UserID:    user.ID,
		Token:     token,
	}
	request, err = svc.ds.NewPasswordResetRequest(request)
	if err != nil {
		return err
	}
	if err := svc.DeleteSessionsForUser(ctx, user.ID); err != nil {
		return err
	}

	resetEmail := kolide.Email{
		From: "no-reply@kolide.co",
		To:   []string{user.Email},
		Msg:  request,
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
