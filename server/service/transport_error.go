package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-sql-driver/mysql"
)

// erroer interface is implemented by response structs to encode business logic errors
type errorer interface {
	error() error
}

type jsonError struct {
	Message string              `json:"message"`
	Code    int                 `json:"code,omitempty"`
	Errors  []map[string]string `json:"errors,omitempty"`
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

// encode error and status header to the client
func encodeError(ctx context.Context, err error, w http.ResponseWriter) {
	ctxerr.Handle(ctx, err)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	err = ctxerr.Cause(err)

	switch e := err.(type) {
	case validationErrorInterface:
		ve := jsonError{
			Message: "Validation Failed",
			Errors:  e.Invalid(),
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		enc.Encode(ve)
	case permissionErrorInterface:
		pe := jsonError{
			Message: "Permission Denied",
			Errors:  e.PermissionError(),
		}
		w.WriteHeader(http.StatusForbidden)
		enc.Encode(pe)
		return
	case mailError:
		me := jsonError{
			Message: "Mail Error",
			Errors:  e.MailError(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(me)
	case osqueryError:
		// osquery expects to receive the node_invalid key when a TLS
		// request provides an invalid node_key for authentication. It
		// doesn't use the error message provided, but we provide this
		// for debugging purposes (and perhaps osquery will use this
		// error message in the future).

		errMap := map[string]interface{}{"error": e.Error()}
		if e.NodeInvalid() {
			w.WriteHeader(http.StatusUnauthorized)
			errMap["node_invalid"] = true
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}

		enc.Encode(errMap)
	case notFoundErrorInterface:
		je := jsonError{
			Message: "Resource Not Found",
			Errors:  baseError(e.Error()),
		}
		w.WriteHeader(http.StatusNotFound)
		enc.Encode(je)
	case existsErrorInterface:
		je := jsonError{
			Message: "Resource Already Exists",
			Errors:  baseError(e.Error()),
		}
		w.WriteHeader(http.StatusConflict)
		enc.Encode(je)
	case badRequestErrorInterface:
		je := jsonError{
			Message: "Bad request",
			Errors:  baseError(e.Error()),
		}
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(je)
	case *mysql.MySQLError:
		je := jsonError{
			Message: "Validation Failed",
			Errors:  baseError(e.Error()),
		}
		statusCode := http.StatusUnprocessableEntity
		if e.Number == 1062 {
			statusCode = http.StatusConflict
		}
		w.WriteHeader(statusCode)
		enc.Encode(je)
	case *fleet.Error:
		je := jsonError{
			Message: e.Error(),
			Code:    e.Code,
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		enc.Encode(je)
	default:
		if fleet.IsForeignKey(ctxerr.Cause(err)) {
			ve := jsonError{
				Message: "Validation Failed",
				Errors:  baseError(err.Error()),
			}
			w.WriteHeader(http.StatusUnprocessableEntity)
			enc.Encode(ve)
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

		w.WriteHeader(status)
		je := jsonError{
			Message: err.Error(),
			Errors:  baseError(err.Error()),
		}
		enc.Encode(je)
	}
}
