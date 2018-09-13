package service

import (
	"context"
	"errors"
	"strings"
	"unicode"

	"github.com/kolide/fleet/server/contexts/viewer"
	"github.com/kolide/fleet/server/kolide"
)

func (mw validationMiddleware) NewUser(ctx context.Context, p kolide.UserPayload) (*kolide.User, error) {
	invalid := &invalidArgumentError{}
	if p.Username == nil {
		invalid.Append("username", "missing required argument")
	} else {
		if *p.Username == "" {
			invalid.Append("username", "cannot be empty")
		}

		if strings.Contains(*p.Username, "@") {
			invalid.Append("username", "'@' character not allowed in usernames")
		}
	}

	// we don't need a password for single sign on
	if p.SSOInvite == nil || !*p.SSOInvite {
		if p.Password == nil {
			invalid.Append("password", "missing required argument")
		} else {
			if *p.Password == "" {
				invalid.Append("password", "cannot be empty")
			}
			if err := validatePasswordRequirements(*p.Password); err != nil {
				invalid.Append("password", err.Error())
			}
		}
	}

	if p.Email == nil {
		invalid.Append("email", "missing required argument")
	} else {
		if *p.Email == "" {
			invalid.Append("email", "cannot be empty")
		}
	}

	if p.InviteToken == nil {
		invalid.Append("invite_token", "missing required argument")
	} else {
		if *p.InviteToken == "" {
			invalid.Append("invite_token", "cannot be empty")
		}
	}

	if invalid.HasErrors() {
		return nil, invalid
	}
	return mw.Service.NewUser(ctx, p)
}

func (mw validationMiddleware) ModifyUser(ctx context.Context, userID uint, p kolide.UserPayload) (*kolide.User, error) {
	invalid := &invalidArgumentError{}
	if p.Username != nil {
		if *p.Username == "" {
			invalid.Append("username", "cannot be empty")
		}

		if strings.Contains(*p.Username, "@") {
			invalid.Append("username", "'@' character not allowed in usernames")
		}
	}

	if p.Name != nil {
		if *p.Name == "" {
			invalid.Append("name", "cannot be empty")
		}
	}

	if p.Email != nil {
		if *p.Email == "" {
			invalid.Append("email", "cannot be empty")
		}
		// if the user is not an admin, or if an admin is changing their own email
		// address a password is required,
		if passwordRequiredForEmailChange(ctx, userID, invalid) {
			if p.Password == nil {
				invalid.Append("password", "cannot be empty if email is changed")
			}
		}
	}

	if invalid.HasErrors() {
		return nil, invalid
	}
	return mw.Service.ModifyUser(ctx, userID, p)
}

func passwordRequiredForEmailChange(ctx context.Context, uid uint, invalid *invalidArgumentError) bool {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		invalid.Append("viewer", "not present")
		return false
	}
	// if a user is changing own email need a password no matter what
	if vc.UserID() == uid {
		return true
	}
	// if an admin is changing another users email no password needed
	if vc.CanPerformAdminActions() {
		return false
	}
	// should never get here because a non admin can't change the email of another
	// user
	invalid.Append("auth", "this user can't change another user's email")
	return false
}

func (mw validationMiddleware) ChangePassword(ctx context.Context, oldPass, newPass string) error {
	invalid := &invalidArgumentError{}
	if oldPass == "" {
		invalid.Append("old_password", "cannot be empty")
	}
	if newPass == "" {
		invalid.Append("new_password", "cannot be empty")
	}

	if err := validatePasswordRequirements(newPass); err != nil {
		invalid.Append("new_password", err.Error())
	}

	if invalid.HasErrors() {
		return invalid
	}
	return mw.Service.ChangePassword(ctx, oldPass, newPass)
}

func (mw validationMiddleware) ResetPassword(ctx context.Context, token, password string) error {
	invalid := &invalidArgumentError{}
	if token == "" {
		invalid.Append("token", "cannot be empty field")
	}
	if password == "" {
		invalid.Append("new_password", "cannot be empty field")
	}
	if err := validatePasswordRequirements(password); err != nil {
		invalid.Append("new_password", err.Error())
	}
	if invalid.HasErrors() {
		return invalid
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

	return errors.New("password does not meet validation requirements")
}
