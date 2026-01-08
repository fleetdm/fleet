package endpointer

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-sql-driver/mysql"
)

// ErrBadRoute is used for mux errors
var ErrBadRoute = errors.New("bad route")

// DomainErrorEncoder handles domain-specific error encoding.
// It returns true if it handled the error, false if default handling should be used.
// The encoder should write the appropriate status code and response body.
type DomainErrorEncoder func(ctx context.Context, err error, w http.ResponseWriter, enc *json.Encoder, jsonErr *JsonError) (handled bool)

type JsonError struct {
	Message string              `json:"message"`
	Code    int                 `json:"code,omitempty"`
	Errors  []map[string]string `json:"errors,omitempty"`
	UUID    string              `json:"uuid,omitempty"`
}

// use baseError to encode an JsonError.Errors field with an error that has
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

type NotFoundErrorInterface interface {
	error
	IsNotFound() bool
}

type ExistsErrorInterface interface {
	error
	IsExists() bool
}

type conflictErrorInterface interface {
	error
	IsConflict() bool
}

// EncodeError encodes error and status header to the client.
// The domainEncoder parameter allows services to inject domain-specific error
// handling. If nil, only generic error handling is performed.
func EncodeError(ctx context.Context, err error, w http.ResponseWriter, domainEncoder DomainErrorEncoder) {
	ctxerr.Handle(ctx, err)
	origErr := err

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	err = ctxerr.Cause(err)

	var uuid string
	if uuidErr, ok := err.(platform_http.ErrorUUIDer); ok {
		uuid = uuidErr.UUID()
	}

	jsonErr := JsonError{
		UUID: uuid,
	}

	// Try domain-specific error encoder first
	if domainEncoder != nil {
		if handled := domainEncoder(ctx, err, w, enc, &jsonErr); handled {
			return
		}
	}

	switch e := err.(type) {
	case validationErrorInterface:
		if statusErr, ok := e.(interface{ Status() int }); ok {
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
	case NotFoundErrorInterface:
		jsonErr.Message = "Resource Not Found"
		jsonErr.Errors = baseError(e.Error())
		w.WriteHeader(http.StatusNotFound)
	case ExistsErrorInterface:
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
	case *platform_http.Error:
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
		if platform_http.IsForeignKey(err) {
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
		var ewra platform_http.ErrWithRetryAfter
		if errors.As(err, &ewra) {
			w.Header().Add("Retry-After", strconv.Itoa(ewra.RetryAfter()))
		}

		msg := err.Error()
		reason := err.Error()
		var ume *platform_http.UserMessageError
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
