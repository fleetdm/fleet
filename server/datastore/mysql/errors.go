package mysql

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"syscall"

	"github.com/VividCortex/mysqlerr"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-sql-driver/mysql"
)

func notFound(kind string) *common_mysql.NotFoundError {
	return &common_mysql.NotFoundError{
		ResourceType: kind,
	}
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

// isBadConnection checks if the error is a connection-level error that
// justifies retrying the operation on a new connection.
func isBadConnection(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, driver.ErrBadConn) ||
		errors.Is(err, mysql.ErrInvalidConn) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ENETUNREACH) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	var mySQLErr *mysql.MySQLError
	if errors.As(err, &mySQLErr) {
		switch mySQLErr.Number {
		case 1243, // ER_UNKNOWN_STMT_HANDLER
			2006, // ER_SERVER_GONE_ERROR
			1053: // ER_SERVER_SHUTDOWN
			return true
		}
	}

	// Check for underlying network errors.
	var se *os.SyscallError
	if errors.As(err, &se) {
		return errors.Is(se.Err, syscall.ECONNRESET) || errors.Is(se.Err, syscall.EPIPE)
	}

	return false
}

// ErrPartialResult indicates that a batch operation was completed,
// but some of the results are missing or incomplete.
var ErrPartialResult = errors.New("batch operation completed with partial results")
