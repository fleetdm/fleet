package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
)

// FleetErrorEncoder handles fleet-specific error encoding for MailError
// and OsqueryError.
func FleetErrorEncoder(ctx context.Context, err error, w http.ResponseWriter, enc *json.Encoder, jsonErr *endpointer.JsonError) bool {
	switch e := err.(type) {
	case MailError:
		jsonErr.Message = "Mail Error"
		jsonErr.Errors = []map[string]string{
			{
				"name":   "base",
				"reason": e.Message,
			},
		}
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(jsonErr) //nolint:errcheck
		return true

	case *OsqueryError:
		// osquery expects to receive the node_invalid key when a TLS
		// request provides an invalid node_key for authentication. It
		// doesn't use the error message provided, but we provide this
		// for debugging purposes (and perhaps osquery will use this
		// error message in the future).

		errMap := map[string]any{
			"error": e.Error(),
			"uuid":  jsonErr.UUID,
		}
		if e.NodeInvalid() { //nolint:gocritic // ignore ifElseChain
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
		return true
	}

	return false
}

// MailError is set when an error performing mail operations
type MailError struct {
	Message string
}

func (e MailError) Error() string {
	return fmt.Sprintf("a mail error occurred: %s", e.Message)
}

// OsqueryError is the error returned to osquery agents.
type OsqueryError struct {
	message     string
	nodeInvalid bool
	StatusCode  int
	platform_http.ErrorWithUUID
}

var _ platform_http.ErrorUUIDer = (*OsqueryError)(nil)

// Error implements the error interface.
func (e *OsqueryError) Error() string {
	return e.message
}

// NodeInvalid returns whether the error returned to osquery
// should contain the node_invalid property.
func (e *OsqueryError) NodeInvalid() bool {
	return e.nodeInvalid
}

func (e *OsqueryError) Status() int {
	return e.StatusCode
}

func NewOsqueryError(message string, nodeInvalid bool) *OsqueryError {
	return &OsqueryError{
		message:     message,
		nodeInvalid: nodeInvalid,
	}
}

// encodeError is a convenience function that calls endpointer.EncodeError
// with the FleetErrorEncoder. Use this for direct error encoding in handlers.
func encodeError(ctx context.Context, err error, w http.ResponseWriter) {
	endpointer.EncodeError(ctx, err, w, FleetErrorEncoder)
}

// fleetErrorEncoder is an adapter that wraps endpointer.EncodeError with
// FleetErrorEncoder for use as a kithttp.ErrorEncoder.
func fleetErrorEncoder(ctx context.Context, err error, w http.ResponseWriter) {
	endpointer.EncodeError(ctx, err, w, FleetErrorEncoder)
}
