package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/VividCortex/mysqlerr"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-sql-driver/mysql"
)

type notFoundError struct {
	ID           uint
	Name         string
	Message      string
	ResourceType string

	fleet.ErrorWithUUID
}

var _ fleet.NotFoundError = (*notFoundError)(nil)

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

// Implement Is so that errors.Is(err, sql.ErrNoRows) returns true for an
// error of type *notFoundError, without having to wrap sql.ErrNoRows
// explicitly.
func (e *notFoundError) Is(other error) bool {
	return other == sql.ErrNoRows
}

type existsError struct {
	Identifier   interface{}
	ResourceType string
	TeamID       *uint

	fleet.ErrorWithUUID
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

func (e *existsError) WithTeamID(teamID uint) error {
	e.TeamID = &teamID
	return e
}

func (e *existsError) Error() string {
	msg := e.ResourceType
	if e.Identifier != nil {
		msg += fmt.Sprintf(" %v", e.Identifier)
	}
	msg += " already exists"
	if e.TeamID != nil {
		msg += fmt.Sprintf(" with TeamID %d", *e.TeamID)
	}
	return msg
}

func (e *existsError) IsExists() bool {
	return true
}

func (e *existsError) Resource() string {
	return e.ResourceType
}

func IsDuplicate(err error) bool {
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

	fleet.ErrorWithUUID
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

// accessDeniedError is an error that implements StatusCode and Internal
type accessDeniedError struct {
	Message     string
	InternalErr error
	Code        int
}

// Error returns the error message.
func (e *accessDeniedError) Error() string {
	return e.Message
}

func (e accessDeniedError) Internal() string {
	if e.InternalErr == nil {
		return ""
	}
	return e.InternalErr.Error()
}

func (e *accessDeniedError) StatusCode() int {
	if e.Code == 0 {
		return http.StatusUnprocessableEntity
	}
	return e.Code
}

func isMySQLAccessDenied(err error) bool {
	err = ctxerr.Cause(err)
	var mySQLErr *mysql.MySQLError
	if errors.As(
		err, &mySQLErr,
	) && (mySQLErr.Number == mysqlerr.ER_SPECIFIC_ACCESS_DENIED_ERROR || mySQLErr.Number == mysqlerr.ER_TABLEACCESS_DENIED_ERROR) {
		return true
	}
	return false
}

func isMySQLUnknownStatement(err error) bool {
	err = ctxerr.Cause(err)
	var mySQLErr *mysql.MySQLError
	return errors.As(err, &mySQLErr) && (mySQLErr.Number == mysqlerr.ER_UNKNOWN_STMT_HANDLER)
}

// ErrPartialResult indicates that a batch operation was completed,
// but some of the results are missing or incomplete.
var ErrPartialResult = errors.New("batch operation completed with partial results")
