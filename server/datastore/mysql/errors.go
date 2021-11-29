package mysql

import (
	"fmt"
	"strconv"

	"github.com/VividCortex/mysqlerr"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/go-sql-driver/mysql"
)

type notFoundError struct {
	ID           uint
	Name         string
	Message      string
	ResourceType string
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

type existsError struct {
	Identifier   interface{}
	ResourceType string
}

func alreadyExists(kind string, identifier interface{}) error {
	if s, ok := identifier.(string); ok {
		identifier = strconv.Quote(s)
	}
	return &existsError{
		Identifier:   identifier,
		ResourceType: kind,
	}
}

func (e *existsError) Error() string {
	return fmt.Sprintf("%s %v already exists", e.ResourceType, e.Identifier)
}

func (e *existsError) IsExists() bool {
	return true
}

func isDuplicate(err error) bool {
	err = ctxerr.Cause(err)
	if driverErr, ok := err.(*mysql.MySQLError); ok {
		if driverErr.Number == mysqlerr.ER_DUP_ENTRY {
			return true
		}
	}
	return false
}

type foreignKeyError struct {
	Name         string
	ResourceType string
}

func foreignKey(kind string, name string) error {
	return &foreignKeyError{
		Name:         name,
		ResourceType: kind,
	}
}

func (e *foreignKeyError) Error() string {
	return fmt.Sprintf("the operation violates a foreign key constraint on %s: %s", e.ResourceType, e.Name)
}

func (e *foreignKeyError) IsForeignKey() bool {
	return true
}

func isMySQLForeignKey(err error) bool {
	err = ctxerr.Cause(err)
	if driverErr, ok := err.(*mysql.MySQLError); ok {
		if driverErr.Number == mysqlerr.ER_ROW_IS_REFERENCED_2 {
			return true
		}
	}
	return false
}
