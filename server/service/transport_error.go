package service

import (
	"encoding/json"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/kolide/kolide-ose/server/datastore"
	"golang.org/x/net/context"
)

// erroer interface is implemented by response structs to encode business logic errors
type errorer interface {
	error() error
}

type jsonError struct {
	Message string              `json:"message"`
	Errors  []map[string]string `json:"errors,omitempty"`
	Error   string              `json:"error,omitempty"`
}

// encode error and status header to the client
func encodeError(ctx context.Context, err error, w http.ResponseWriter) {
	// Unwrap Go-Kit Error
	domain := "service"
	if e, ok := err.(kithttp.Error); ok {
		err = e.Err
		domain = e.Domain
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	type validationError interface {
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
		AuthError() string
	}
	if e, ok := err.(authenticationError); ok {
		ae := jsonError{
			Message: "Authentication Failed",
			Error:   e.AuthError(),
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

	// Other errors
	switch domain {
	case "service":
		w.WriteHeader(codeFromErr(err))
	case kithttp.DomainDecode:
		w.WriteHeader(http.StatusBadRequest)
	case kithttp.DomainDo:
		w.WriteHeader(http.StatusServiceUnavailable)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	enc.Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

func codeFromErr(err error) int {
	switch err {
	case datastore.ErrNotFound:
		return http.StatusNotFound
	case datastore.ErrExists:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
