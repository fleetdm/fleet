package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"
)

func (svc service) NewUser(ctx context.Context, p kolide.UserPayload) (*kolide.User, error) {
	user, err := userFromPayload(p, svc.config.Auth.SaltKeySize, svc.config.Auth.BcryptCost)
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

	if p.AdminForcedPasswordReset != nil {
		err = svc.RequestPasswordReset(ctx, user.Email)
		if err != nil {
			return nil, err
		}
	}

	if p.Password != nil {
		hashed, salt, err := hashPassword(
			*p.Password,
			svc.config.Auth.SaltKeySize,
			svc.config.Auth.BcryptCost,
		)
		if err != nil {
			return nil, err
		}
		user.Password = hashed
		user.Salt = salt
	}

	err = svc.saveUser(user)
	if err != nil {
		return nil, err
	}

	return user, nil

}

func (svc service) User(ctx context.Context, id uint) (*kolide.User, error) {
	return svc.ds.UserByID(id)
}

func (svc service) AuthenticatedUser(ctx context.Context) (*kolide.User, error) {
	vc, err := viewerContextFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return vc.user, nil
}

func (svc service) Users(ctx context.Context) ([]*kolide.User, error) {
	return svc.ds.Users()
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

	hashed, salt, err := hashPassword(password, svc.config.Auth.SaltKeySize, svc.config.Auth.BcryptCost)
	if err != nil {
		return err
	}
	user.Salt = salt
	user.Password = hashed
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
	// the password reset is different depending on wether performed by an admin
	// or a user
	// if an admin requests a password reset, then no token is
	// generated, instead the AdminForcedPasswordReset flag is set
	vc, err := viewerContextFromContext(ctx)
	if err != nil {
		return err
	}

	user, err := svc.ds.UserByEmail(email)
	if err != nil {
		return err
	}

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

	token, err := generateRandomText(svc.config.SMTP.TokenKeySize)
	if err != nil {
		return err
	}

	request := &kolide.PasswordResetRequest{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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

func userFromPayload(p kolide.UserPayload, keySize, cost int) (*kolide.User, error) {
	hashed, salt, err := hashPassword(*p.Password, keySize, cost)
	if err != nil {
		return nil, err
	}

	return &kolide.User{
		Username: *p.Username,
		Email:    *p.Email,
		Admin:    falseIfNil(p.Admin),
		AdminForcedPasswordReset: falseIfNil(p.AdminForcedPasswordReset),
		Salt:     salt,
		Enabled:  true,
		Password: hashed,
	}, nil
}

func hashPassword(plaintext string, keySize, cost int) ([]byte, string, error) {
	salt, err := generateRandomText(keySize)
	if err != nil {
		return nil, "", err
	}

	withSalt := []byte(fmt.Sprintf("%s%s", plaintext, salt))
	hashed, err := bcrypt.GenerateFromPassword(withSalt, cost)
	if err != nil {
		return nil, "", err
	}

	return hashed, salt, nil
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

// helper to convert a bool pointer false
func falseIfNil(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
