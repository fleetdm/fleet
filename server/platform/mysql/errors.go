package mysql

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_errors "github.com/fleetdm/fleet/v4/server/platform/errors"
	"github.com/go-sql-driver/mysql"
)

type NotFoundError struct {
	ID           uint
	Name         string
	Message      string
	ResourceType string
}

// Compile-time interface check.
var _ platform_errors.NotFoundError = &NotFoundError{}

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

// IsClientError implements ErrWithIsClientError.
func (e *NotFoundError) IsClientError() bool {
	return true
}

// Is helps so that errors.Is(err, sql.ErrNoRows) returns true for an
// error of type *NotFoundError, without having to wrap sql.ErrNoRows
// explicitly.
func (e *NotFoundError) Is(other error) bool {
	return other == sql.ErrNoRows
}

// MySQL error numbers for read-only conditions. These are not included in the
// VividCortex/mysqlerr package, so we define them here.
const (
	// erReadOnlyTransaction is MySQL error 1792: Cannot execute statement in a READ ONLY transaction.
	erReadOnlyTransaction = 1792
	// erOptionPreventsStatement is MySQL error 1290: The MySQL server is running with the --read-only option.
	erOptionPreventsStatement = 1290
	// erReadOnlyMode is MySQL error 1836: Running in read-only mode.
	erReadOnlyMode = 1836
)

// IsReadOnlyError returns true if the error is a MySQL error indicating that
// the server is in read-only mode. This typically happens after an Aurora
// failover when the primary has been demoted to a reader.
func IsReadOnlyError(err error) bool {
	err = ctxerr.Cause(err)
	var mySQLErr *mysql.MySQLError
	if errors.As(err, &mySQLErr) {
		switch mySQLErr.Number {
		case erReadOnlyTransaction, erOptionPreventsStatement, erReadOnlyMode:
			return true
		}
	}
	return false
}
