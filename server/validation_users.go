package server

import (
	"strings"

	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

type validationMiddleware struct {
	kolide.Service
}

func (mw validationMiddleware) NewUser(ctx context.Context, p kolide.UserPayload) (*kolide.User, error) {
	var invalid []invalidArgument
	if p.Username == nil {
		invalid = append(invalid, invalidArgument{name: "username", reason: "missing required argument"})
	}
	if p.Username != nil {
		if strings.Contains(*p.Username, "@") {
			invalid = append(invalid, invalidArgument{name: "username", reason: "'@' character not allowed in usernames"})
		}
	}
	if p.Password == nil {
		invalid = append(invalid, invalidArgument{name: "password", reason: "missing required argument"})
	}
	if p.Email == nil {
		invalid = append(invalid, invalidArgument{name: "email", reason: "missing required argument"})
	}
	if len(invalid) != 0 {
		return nil, invalidArgumentError(invalid)
	}
	return mw.Service.NewUser(ctx, p)
}

func (mw validationMiddleware) ResetPassword(ctx context.Context, token, password string) error {
	var invalid []invalidArgument
	if token == "" {
		invalid = append(invalid, invalidArgument{name: "token", reason: "cannot be empty field"})
	}
	if password == "" {
		invalid = append(invalid, invalidArgument{name: "new_password", reason: "cannot be empty field"})
	}
	if len(invalid) != 0 {
		return invalidArgumentError(invalid)
	}
	return mw.Service.ResetPassword(ctx, token, password)
}

type invalidArgumentError []invalidArgument
type invalidArgument struct {
	name   string
	reason string
}

// invalidArgumentError is returned when one or more arguments are invalid.
func (e invalidArgumentError) Error() string {
	return "validation failed"
}

func (e invalidArgumentError) Invalid() []map[string]string {
	var invalid []map[string]string
	for _, i := range e {
		invalid = append(invalid, map[string]string{"name": i.name, "reason": i.reason})
	}
	return invalid
}
