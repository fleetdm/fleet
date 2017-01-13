package service

import (
	"fmt"
	"strings"

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
	if invalid.HasErrors() {
		return invalid
	}
	return mw.Service.ResetPassword(ctx, token, password)
}

type invalidArgumentError []invalidArgument
type invalidArgument struct {
	name   string
	reason string
}

// newInvalidArgumentError returns a invalidArgumentError with at least
// one error.
func newInvalidArgumentError(name, reason string) *invalidArgumentError {
	var invalid invalidArgumentError
	invalid = append(invalid, invalidArgument{
		name:   name,
		reason: reason,
	})
	return &invalid
}

func (e *invalidArgumentError) Append(name, reason string) {
	*e = append(*e, invalidArgument{
		name:   name,
		reason: reason,
	})
}
func (e *invalidArgumentError) Appendf(name, reasonFmt string, args ...interface{}) {
	*e = append(*e, invalidArgument{
		name:   name,
		reason: fmt.Sprintf(reasonFmt, args...),
	})
}

func (e *invalidArgumentError) HasErrors() bool {
	return len(*e) != 0
}

// invalidArgumentError is returned when one or more arguments are invalid.
func (e invalidArgumentError) Error() string {
	switch len(e) {
	case 0:
		return "validation failed"
	case 1:
		return fmt.Sprintf("validation failed: %s %s", e[0].name, e[0].reason)
	default:
		return fmt.Sprintf("validation failed: %s %s and %d other errors", e[0].name, e[0].reason,
			len(e))
	}
}

func (e invalidArgumentError) Invalid() []map[string]string {
	var invalid []map[string]string
	for _, i := range e {
		invalid = append(invalid, map[string]string{"name": i.name, "reason": i.reason})
	}
	return invalid
}
