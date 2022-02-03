package service

import (
	"context"
	"errors"
	"unicode"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (mw validationMiddleware) CreateUserFromInvite(ctx context.Context, p fleet.UserPayload) (*fleet.User, error) {
	invalid := &fleet.InvalidArgumentError{}
	if p.Name == nil {
		invalid.Append("name", "Full name missing required argument")
	} else {
		if *p.Name == "" {
			invalid.Append("name", "Full name cannot be empty")
		}
	}

	// we don't need a password for single sign on
	if p.SSOInvite == nil || !*p.SSOInvite {
		if p.Password == nil {
			invalid.Append("password", "Password missing required argument")
		} else {
			if *p.Password == "" {
				invalid.Append("password", "Password cannot be empty")
			}
			if err := validatePasswordRequirements(*p.Password); err != nil {
				invalid.Append("password", err.Error())
			}
		}
	}

	if p.Email == nil {
		invalid.Append("email", "Email missing required argument")
	} else {
		if *p.Email == "" {
			invalid.Append("email", "Email cannot be empty")
		}
	}

	if p.InviteToken == nil {
		invalid.Append("invite_token", "Invite token missing required argument")
	} else {
		if *p.InviteToken == "" {
			invalid.Append("invite_token", "Invite token cannot be empty")
		}
	}

	if invalid.HasErrors() {
		return nil, ctxerr.Wrap(ctx, invalid)
	}
	return mw.Service.CreateUserFromInvite(ctx, p)
}

func (mw validationMiddleware) CreateUser(ctx context.Context, p fleet.UserPayload) (*fleet.User, error) {
	invalid := &fleet.InvalidArgumentError{}
	if p.Name == nil {
		invalid.Append("name", "Full name missing required argument")
	} else {
		if *p.Name == "" {
			invalid.Append("name", "Full name cannot be empty")
		}
	}

	// we don't need a password for single sign on
	if (p.SSOInvite == nil || !*p.SSOInvite) && (p.SSOEnabled == nil || !*p.SSOEnabled) {
		if p.Password == nil {
			invalid.Append("password", "Password missing required argument")
		} else {
			if *p.Password == "" {
				invalid.Append("password", "Password cannot be empty")
			}
			// Skip password validation in the case of admin created users
		}
	}

	if p.SSOEnabled != nil && *p.SSOEnabled && p.Password != nil && len(*p.Password) > 0 {
		invalid.Append("password", "not allowed for SSO users")
	}

	if p.Email == nil {
		invalid.Append("email", "Email missing required argument")
	} else {
		if *p.Email == "" {
			invalid.Append("email", "Email cannot be empty")
		}
	}

	if p.InviteToken != nil {
		invalid.Append("invite_token", "Invite token should not be specified with admin user creation")
	}

	if invalid.HasErrors() {
		return nil, ctxerr.Wrap(ctx, invalid)
	}
	return mw.Service.CreateUser(ctx, p)
}

func (mw validationMiddleware) ModifyUser(ctx context.Context, userID uint, p fleet.UserPayload) (*fleet.User, error) {
	invalid := &fleet.InvalidArgumentError{}
	if p.Name != nil {
		if *p.Name == "" {
			invalid.Append("name", "Full name cannot be empty")
		}
	}

	if p.Email != nil {
		if *p.Email == "" {
			invalid.Append("email", "Email cannot be empty")
		}
		// if the user is not an admin, or if an admin is changing their own email
		// address a password is required,
		if passwordRequiredForEmailChange(ctx, userID, invalid) {
			if p.Password == nil {
				invalid.Append("password", "Password cannot be empty if email is changed")
			}
		}
	}

	if invalid.HasErrors() {
		return nil, ctxerr.Wrap(ctx, invalid)
	}
	return mw.Service.ModifyUser(ctx, userID, p)
}

func passwordRequiredForEmailChange(ctx context.Context, uid uint, invalid *fleet.InvalidArgumentError) bool {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		invalid.Append("viewer", "Viewer not present")
		return false
	}
	// if a user is changing own email need a password no matter what
	return vc.UserID() == uid
}

func (mw validationMiddleware) ChangePassword(ctx context.Context, oldPass, newPass string) error {
	invalid := &fleet.InvalidArgumentError{}
	if oldPass == "" {
		invalid.Append("old_password", "Old password cannot be empty")
	}
	if newPass == "" {
		invalid.Append("new_password", "New password cannot be empty")
	}

	if err := validatePasswordRequirements(newPass); err != nil {
		invalid.Append("new_password", err.Error())
	}

	if invalid.HasErrors() {
		return ctxerr.Wrap(ctx, invalid)
	}
	return mw.Service.ChangePassword(ctx, oldPass, newPass)
}

func (mw validationMiddleware) ResetPassword(ctx context.Context, token, password string) error {
	invalid := &fleet.InvalidArgumentError{}
	if token == "" {
		invalid.Append("token", "Token cannot be empty field")
	}
	if password == "" {
		invalid.Append("new_password", "New password cannot be empty field")
	}
	if err := validatePasswordRequirements(password); err != nil {
		invalid.Append("new_password", err.Error())
	}
	if invalid.HasErrors() {
		return ctxerr.Wrap(ctx, invalid)
	}
	return mw.Service.ResetPassword(ctx, token, password)
}

// Requirements for user password:
// at least 7 character length
// at least 1 symbol
// at least 1 number
func validatePasswordRequirements(password string) error {
	var (
		number bool
		symbol bool
	)

	for _, s := range password {
		switch {
		case unicode.IsNumber(s):
			number = true
		case unicode.IsPunct(s) || unicode.IsSymbol(s):
			symbol = true
		}
	}

	if len(password) >= 7 &&
		number &&
		symbol {
		return nil
	}

	return errors.New("Password does not meet validation requirements")
}
