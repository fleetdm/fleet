package service

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type notFoundError struct {
	fleet.ErrorWithUUID
}

func (e notFoundError) Error() string {
	return "not found"
}

// IsNotFound implements the service.IsNotFound interface (from the non-premium
// service package) so that the handler returns 404 for this error.
func (e notFoundError) IsNotFound() bool {
	return true
}

type NDESInvalidError struct {
	msg string
}

func (e NDESInvalidError) Error() string {
	return e.msg
}

func NewNDESInvalidError(msg string) NDESInvalidError {
	return NDESInvalidError{msg: msg}
}

type NDESPasswordCacheFullError struct {
	msg string
}

func (e NDESPasswordCacheFullError) Error() string {
	return e.msg
}

func NewNDESPasswordCacheFullError(msg string) NDESPasswordCacheFullError {
	return NDESPasswordCacheFullError{msg: msg}
}

type NDESInsufficientPermissionsError struct {
	msg string
}

func (e NDESInsufficientPermissionsError) Error() string {
	return e.msg
}

func NewNDESInsufficientPermissionsError(msg string) NDESInsufficientPermissionsError {
	return NDESInsufficientPermissionsError{msg: msg}
}
