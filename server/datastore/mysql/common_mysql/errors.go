package common_mysql

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type NotFoundError struct {
	ID           uint
	Name         string
	Message      string
	ResourceType string
}

// Compile-time interface check.
var _ fleet.NotFoundError = &NotFoundError{}

func NotFound(kind string) *NotFoundError {
	return &NotFoundError{
		ResourceType: kind,
	}
}

func (e *NotFoundError) Error() string {
	if e.ID != 0 {
		return fmt.Sprintf("%s %d was not found in the datastore", e.ResourceType, e.ID)
	}
	if e.Name != "" {
		return fmt.Sprintf("%s %s was not found in the datastore", e.ResourceType, e.Name)
	}
	if e.Message != "" {
		return fmt.Sprintf("%s %s was not found in the datastore", e.ResourceType, e.Message)
	}
	return fmt.Sprintf("%s was not found in the datastore", e.ResourceType)
}

func (e *NotFoundError) WithID(id uint) error {
	e.ID = id
	return e
}

func (e *NotFoundError) WithName(name string) error {
	e.Name = name
	return e
}

func (e *NotFoundError) WithMessage(msg string) error {
	e.Message = msg
	return e
}

func (e *NotFoundError) IsNotFound() bool {
	return true
}

// Is helps so that errors.Is(err, sql.ErrNoRows) returns true for an
// error of type *NotFoundError, without having to wrap sql.ErrNoRows
// explicitly.
func (e *NotFoundError) Is(other error) bool {
	return other == sql.ErrNoRows
}
