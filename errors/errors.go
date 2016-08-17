package errors

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"gopkg.in/go-playground/validator.v8"
)

// Kolide's internal representation for errors. It can be used to wrap another
// error (stored in Err), and additionally contains fields for public
// (PublicMessage) and private (PrivateMessage) error messages as well as the
// HTTP status code (StatusCode) corresponding to the error. Extra holds extra
// information that will be inserted as top level key/value pairs in the error
// response.
type KolideError struct {
	Err            error
	StatusCode     int
	PublicMessage  string
	PrivateMessage string
	Extra          map[string]interface{}
}

// Implementation of error interface
func (e *KolideError) Error() string {
	return e.PublicMessage
}

// Create a new KolideError specifying the public and private messages. The
// status code will be set to 500.
func New(publicMessage, privateMessage string) *KolideError {
	return &KolideError{
		StatusCode:     http.StatusInternalServerError,
		PublicMessage:  publicMessage,
		PrivateMessage: privateMessage,
	}
}

// Create a new KolideError specifying the HTTP status, and public and private
// messages.
func NewWithStatus(status int, publicMessage, privateMessage string) *KolideError {
	return &KolideError{
		StatusCode:     status,
		PublicMessage:  publicMessage,
		PrivateMessage: privateMessage,
	}
}

// Create a new KolideError from an error type. The public message and status
// code should be specified, while the private message will be drawn from
// err.Error()
func NewFromError(err error, status int, publicMessage string) *KolideError {
	return &KolideError{
		Err:            err,
		StatusCode:     status,
		PublicMessage:  publicMessage,
		PrivateMessage: err.Error(),
	}
}

// Wrap a DB error with the extra KolideError decorations
func DatabaseError(err error) *KolideError {
	return NewFromError(err, http.StatusInternalServerError, "Database error")
}

// Wrap a server error with the extra KolideError decorations
func InternalServerError(err error) *KolideError {
	return NewFromError(err, http.StatusInternalServerError, "Internal server error")
}

// The status code returned for validation errors. Inspired by the Github API.
const StatusUnprocessableEntity = 422

// Handle an error, printing debug information, writing to the HTTP response as
// appropriate for the dynamic error type.
func ReturnError(c *gin.Context, err error) {
	baseReturnError(c, err, "message")
}

// osqueryd does not check the HTTP status code, and only looks for a
func ReturnOsqueryError(c *gin.Context, err error) {
	baseReturnError(c, err, "error")
}

func baseReturnError(c *gin.Context, err error, messageKey string) {
	switch typedErr := err.(type) {

	case *KolideError:
		errJSON := map[string]interface{}{}
		for key, val := range typedErr.Extra {
			errJSON[key] = val
		}
		errJSON[messageKey] = typedErr.PublicMessage

		c.JSON(typedErr.StatusCode, errJSON)
		logrus.WithError(typedErr.Err).Debug(typedErr.PrivateMessage)

	case validator.ValidationErrors:
		errors := make([](map[string]string), 0, len(typedErr))
		for _, fieldErr := range typedErr {
			m := make(map[string]string)
			m["field"] = fieldErr.Name
			m["code"] = "invalid"
			m["message"] = fieldErr.Tag
			errors = append(errors, m)
		}

		c.JSON(StatusUnprocessableEntity,
			gin.H{messageKey: "Validation error",
				"errors": errors,
			})
		logrus.WithError(typedErr).Debug("Validation error")

	case gorm.Errors, *gorm.Errors:
		c.JSON(http.StatusInternalServerError,
			gin.H{messageKey: "Database error"})
		logrus.WithError(typedErr).Debug(typedErr.Error())

	default:
		c.JSON(http.StatusInternalServerError,
			gin.H{messageKey: "Unspecified error"})
		logrus.WithError(typedErr).Debug("Unspecified error")
	}
}
