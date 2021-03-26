package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
)

// erroer interface is implemented by response structs to encode business logic errors
type errorer interface {
	error() error
}

type jsonError struct {
	Message string              `json:"message"`
	Errors  []map[string]string `json:"errors,omitempty"`
}

// use baseError to encode an jsonError.Errors field with an error that has
// a generic "name" field. The frontend client always expects errors in a
// []map[string]string format.
func baseError(err string) []map[string]string {
	return []map[string]string{map[string]string{
		"name":   "base",
		"reason": err},
	}
}

// encode error and status header to the client
func encodeError(ctx context.Context, err error, w http.ResponseWriter) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	type validationError interface {
		error
		Invalid() []map[string]string
	}
	if e, ok := err.(validationError); ok {
		ve := jsonError{
			Message: "Validation Failed",
			Errors:  e.Invalid(),
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		enc.Encode(ve)
		return
	}

	type authenticationError interface {
		error
		AuthError() string
	}
	if e, ok := err.(authenticationError); ok {
		ae := jsonError{
			Message: "Authentication Failed",
			Errors:  baseError(e.AuthError()),
		}
		w.WriteHeader(http.StatusUnauthorized)
		enc.Encode(ae)
		return
	}

	type permissionError interface {
		PermissionError() []map[string]string
	}
	if e, ok := err.(permissionError); ok {
		pe := jsonError{
			Message: "Permission Denied",
			Errors:  e.PermissionError(),
		}
		w.WriteHeader(http.StatusForbidden)
		enc.Encode(pe)
		return
	}

	if kolide.IsForeignKey(errors.Cause(err)) {
		ve := jsonError{
			Message: "Validation Failed",
			Errors:  baseError(err.Error()),
		}
		w.WriteHeader(http.StatusForbidden)
		enc.Encode(ve)
		return
	}

	type mailError interface {
		MailError() []map[string]string
	}
	if e, ok := err.(mailError); ok {
		me := jsonError{
			Message: "Mail Error",
			Errors:  e.MailError(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(me)
		return
	}

	type osqueryError interface {
		error
		NodeInvalid() bool
	}
	if e, ok := err.(osqueryError); ok {
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
		return
	}

	type notFoundError interface {
		error
		IsNotFound() bool
	}
	if e, ok := err.(notFoundError); ok {
		je := jsonError{
			Message: "Resource Not Found",
			Errors:  baseError(e.Error()),
		}
		w.WriteHeader(http.StatusNotFound)
		enc.Encode(je)
		return
	}

	type existsError interface {
		error
		IsExists() bool
	}
	if e, ok := err.(existsError); ok {
		je := jsonError{
			Message: "Resource Already Exists",
			Errors:  baseError(e.Error()),
		}
		w.WriteHeader(http.StatusConflict)
		enc.Encode(je)
		return
	}

	// Get specific status code if it is available from this error type,
	// defaulting to HTTP 500
	status := http.StatusInternalServerError
	if e, ok := err.(ErrWithStatusCode); ok {
		status = e.StatusCode()
	}

	// See header documentation
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After)
	if e, ok := err.(ErrWithRetryAfter); ok {
		w.Header().Add("Retry-After", strconv.Itoa(e.RetryAfter()))
	}

	w.WriteHeader(status)
	je := jsonError{
		Message: err.Error(),
		Errors:  baseError(err.Error()),
	}
	enc.Encode(je)
}
