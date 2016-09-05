package server

import (
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

type validationMiddleware struct {
	kolide.Service
}

func (mw validationMiddleware) NewUser(ctx context.Context, p kolide.UserPayload) (*kolide.User, error) {
	// check required params
	if p.Username == nil {
		return nil, invalidArgumentError{field: "username", required: true}
	}

	if p.Password == nil {
		return nil, invalidArgumentError{field: "password", required: true}
	}

	if p.Email == nil {
		return nil, invalidArgumentError{field: "email", required: true}
	}

	return mw.Service.NewUser(ctx, p)
}

func (mw validationMiddleware) ChangePassword(ctx context.Context, userID uint, old, new string) error {
	if old == "" || new == "" {
		return invalidArgumentError{field: "password", required: true}
	}
	return mw.Service.ChangePassword(ctx, userID, old, new)
}

func (mw validationMiddleware) UpdateUserStatus(ctx context.Context, userID uint, password string, enabled bool) error {
	// validate password if user is disabling self
	vc, err := viewerContextFromContext(ctx)
	if err != nil {
		return err
	}
	if vc.IsUserID(userID) {
		if err := vc.user.ValidatePassword(password); err != nil {
			return err
		}
	}
	return mw.Service.UpdateUserStatus(ctx, userID, password, enabled)
}
