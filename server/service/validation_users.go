package service

import (
	"errors"
	"strings"
	"unicode"

	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

type validationMiddleware struct {
	kolide.Service
	ds kolide.Datastore
}

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
	}

	if invalid.HasErrors() {
		return nil, invalid
	}
	return mw.Service.ModifyUser(ctx, userID, p)
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
