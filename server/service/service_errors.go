package service

import "fmt"

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

// authentication error
type authError struct {
	reason string
	// client reason is used to provide
	// a different error message to the client
	// when security is a concern
	clientReason string
}

func (e authError) Error() string {
	return e.reason
}

func (e authError) AuthError() string {
	if e.clientReason != "" {
		return e.clientReason
	}
	return "username or email and password do not match"
}

// licensingError occurs when the user license is expired, invalid, or revoked
type licensingError struct {
	reason string
}

func (e licensingError) LicensingError() string {
	return e.reason
}

func (e licensingError) Error() string {
	return e.reason
}

// permissionError, set when user is authenticated, but not allowed to perform action
type permissionError struct {
	message string
	badArgs []invalidArgument
}

func (e permissionError) Error() string {
	switch len(e.badArgs) {
	case 0:
	case 1:
		e.message = fmt.Sprintf("unauthorized: %s",
			e.badArgs[0].reason,
		)
	default:
		e.message = fmt.Sprintf("unauthorized: %s and %d other errors",
			e.badArgs[0].reason,
			len(e.badArgs),
		)
	}
	if e.message == "" {
		return "unauthorized"
	}
	return e.message
}

func (e permissionError) PermissionError() []map[string]string {
	var forbidden []map[string]string
	if len(e.badArgs) == 0 {
		forbidden = append(forbidden, map[string]string{"reason": e.Error()})
		return forbidden
	}
	for _, arg := range e.badArgs {
		forbidden = append(forbidden, map[string]string{
			"name":   arg.name,
			"reason": arg.reason,
		})
	}
	return forbidden

}
