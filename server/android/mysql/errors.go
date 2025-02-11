package mysql

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type notFoundError struct {
	ID           uint
	Name         string
	Message      string
	ResourceType string

	fleet.ErrorWithUUID
}

func notFound(kind string) *notFoundError {
	return &notFoundError{
		ResourceType: kind,
	}
}

func (e *notFoundError) Error() string {
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

func (e *notFoundError) WithID(id uint) error {
	e.ID = id
	return e
}

func (e *notFoundError) WithName(name string) error {
	e.Name = name
	return e
}

func (e *notFoundError) WithMessage(msg string) error {
	e.Message = msg
	return e
}

func (e *notFoundError) IsNotFound() bool {
	return true
}

// Is helps so that errors.Is(err, sql.ErrNoRows) returns true for an
// error of type *notFoundError, without having to wrap sql.ErrNoRows
// explicitly.
func (e *notFoundError) Is(other error) bool {
	return other == sql.ErrNoRows
}
