package service

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-sql-driver/mysql"
)

// errorer interface is implemented by response structs to encode business logic errors
type errorer interface {
	error() error
}

type jsonError struct {
	Message string              `json:"message"`
	Code    int                 `json:"code,omitempty"`
	Errors  []map[string]string `json:"errors,omitempty"`
	UUID    string              `json:"uuid,omitempty"`
}

// use baseError to encode an jsonError.Errors field with an error that has
// a generic "name" field. The frontend client always expects errors in a
// []map[string]string format.
func baseError(err string) []map[string]string {
	return []map[string]string{
		{
			"name":   "base",
			"reason": err,
		},
	}
}

type validationErrorInterface interface {
	error
	Invalid() []map[string]string
}

type permissionErrorInterface interface {
	error
	PermissionError() []map[string]string
}

type badRequestErrorInterface interface {
	error
	BadRequestError() []map[string]string
}

type notFoundErrorInterface interface {
	error
	IsNotFound() bool
}

type existsErrorInterface interface {
	error
	IsExists() bool
}

type conflictErrorInterface interface {
	error
	IsConflict() bool
}

// encode error and status header to the client
func encodeError(ctx context.Context, err error, w http.ResponseWriter) {
	ctxerr.Handle(ctx, err)
	origErr := err

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	err = ctxerr.Cause(err)

	var uuid string
	if uuidErr, ok := err.(fleet.ErrorUUIDer); ok {
		uuid = uuidErr.UUID()
	}

	jsonErr := jsonError{
		UUID: uuid,
	}

	switch e := err.(type) {
	case validationErrorInterface:
		if statusErr, ok := e.(statuser); ok {
			w.WriteHeader(statusErr.Status())
		} else {
			w.WriteHeader(http.StatusUnprocessableEntity)
		}
		jsonErr.Message = "Validation Failed"
		jsonErr.Errors = e.Invalid()
	case permissionErrorInterface:
		jsonErr.Message = "Permission Denied"
		jsonErr.Errors = e.PermissionError()
		w.WriteHeader(http.StatusForbidden)
	case mailError:
		jsonErr.Message = "Mail Error"
		jsonErr.Errors = e.MailError()
		w.WriteHeader(http.StatusInternalServerError)
	case *osqueryError:
		// osquery expects to receive the node_invalid key when a TLS
		// request provides an invalid node_key for authentication. It
		// doesn't use the error message provided, but we provide this
		// for debugging purposes (and perhaps osquery will use this
		// error message in the future).

		errMap := map[string]interface{}{
			"error": e.Error(),
			"uuid":  uuid,
		}
		if e.NodeInvalid() {
			w.WriteHeader(http.StatusUnauthorized)
			errMap["node_invalid"] = true
		} else if e.Status() != 0 {
			w.WriteHeader(e.Status())
		} else {
			// TODO: osqueryError is not always the result of an internal error on
			// our side, it is also used to represent a client error (invalid data,
			// e.g. malformed json, carve too large, etc., so 4xx), are we returning
			// a 500 because of some osquery-specific requirement?
			w.WriteHeader(http.StatusInternalServerError)
		}

		enc.Encode(errMap) //nolint:errcheck
		return
	case notFoundErrorInterface:
		jsonErr.Message = "Resource Not Found"
		jsonErr.Errors = baseError(e.Error())
		w.WriteHeader(http.StatusNotFound)
	case existsErrorInterface:
		jsonErr.Message = "Resource Already Exists"
		jsonErr.Errors = baseError(e.Error())
		w.WriteHeader(http.StatusConflict)
	case conflictErrorInterface:
		jsonErr.Message = "Conflict"
		jsonErr.Errors = baseError(e.Error())
		w.WriteHeader(http.StatusConflict)
	case badRequestErrorInterface:
		jsonErr.Message = "Bad request"
		jsonErr.Errors = baseError(e.Error())
		w.WriteHeader(http.StatusBadRequest)
	case *mysql.MySQLError:
		jsonErr.Message = "Validation Failed"
		jsonErr.Errors = baseError(e.Error())
		statusCode := http.StatusUnprocessableEntity
		if e.Number == 1062 {
			statusCode = http.StatusConflict
		}
		w.WriteHeader(statusCode)
	case *fleet.Error:
		jsonErr.Message = e.Error()
		jsonErr.Code = e.Code
		w.WriteHeader(http.StatusUnprocessableEntity)
	default:
		// when there's a tcp read timeout, the error is *net.OpError but the cause is an internal
		// poll.DeadlineExceeded which we cannot match against, so we match against the original error
		var opErr *net.OpError
		if errors.As(origErr, &opErr) {
			jsonErr.Message = opErr.Error()
			jsonErr.Errors = baseError(opErr.Error())
			w.WriteHeader(http.StatusRequestTimeout)
			enc.Encode(jsonErr) //nolint:errcheck
			return
		}
		if fleet.IsForeignKey(err) {
			jsonErr.Message = "Validation Failed"
			jsonErr.Errors = baseError(err.Error())
			w.WriteHeader(http.StatusUnprocessableEntity)
			enc.Encode(jsonErr) //nolint:errcheck
			return
		}

		// Get specific status code if it is available from this error type,
		// defaulting to HTTP 500
		status := http.StatusInternalServerError
		var sce kithttp.StatusCoder
		if errors.As(err, &sce) {
			status = sce.StatusCode()
		}

		// See header documentation
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After)
		var ewra fleet.ErrWithRetryAfter
		if errors.As(err, &ewra) {
			w.Header().Add("Retry-After", strconv.Itoa(ewra.RetryAfter()))
		}

		msg := err.Error()
		reason := err.Error()
		var ume *fleet.UserMessageError
		if errors.As(err, &ume) {
			if text := http.StatusText(status); text != "" {
				msg = text
			}
			reason = ume.UserMessage()
		}

		w.WriteHeader(status)
		jsonErr.Message = msg
		jsonErr.Errors = baseError(reason)
	}

	enc.Encode(jsonErr) //nolint:errcheck
}
