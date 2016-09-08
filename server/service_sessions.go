package server

import (
	"fmt"

	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

func (svc service) Login(ctx context.Context, username, password string) (*kolide.User, string, error) {
	user, err := svc.ds.User(username)
	switch err {
	case nil:
	case datastore.ErrNotFound:
		return nil, "", authError{
			message: fmt.Sprintf("user %s not found", username),
		}
	default:
		return nil, "", err
	}
	if !user.Enabled {
		return nil, "", authError{
			message: fmt.Sprintf("account disabled %s", username),
		}
	}
	if err := user.ValidatePassword(password); err != nil {
		return nil, "", authError{
			message: fmt.Sprintf("invalid password for user %s", username),
		}
	}

	token, err := svc.MakeSession(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (svc service) Logout(ctx context.Context) error {
	// this should not return an error if the user wasn't logged in
	return svc.DestroySession(ctx)
}

func (svc service) MakeSession(ctx context.Context, id uint) (string, error) {
	session, err := svc.ds.CreateSessionForUserID(id)
	if err != nil {
		return "", err
	}

	tokenString, err := kolide.GenerateJWT(session.Key, svc.jwtKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (svc service) DestroySession(ctx context.Context) error {
	vc, err := viewerContextFromContext(ctx)
	if err != nil {
		return err
	}

	session, err := svc.ds.FindSessionByID(vc.SessionID())
	if err != nil {
		return err
	}

	return svc.ds.DestroySession(session)
}

func (svc service) GetInfoAboutSessionsForUser(ctx context.Context, id uint) ([]*kolide.Session, error) {
	return svc.ds.FindAllSessionsForUser(id)
}

func (svc service) DeleteSessionsForUser(ctx context.Context, id uint) error {
	return svc.ds.DestroyAllSessionsForUser(id)
}

func (svc service) GetInfoAboutSession(ctx context.Context, id uint) (*kolide.Session, error) {
	return svc.ds.FindSessionByID(id)
}

func (svc service) DeleteSession(ctx context.Context, id uint) error {
	session, err := svc.ds.FindSessionByID(id)
	if err != nil {
		return err
	}
	return svc.ds.DestroySession(session)
}
