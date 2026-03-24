package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

const acmeErrorsURN = "urn:ietf:params:acme:error:"

// ACMEError represents an error related to the ACME protocol,
// see https://datatracker.ietf.org/doc/html/rfc8555/#section-6.7
//
// It renders as a problem document (https://datatracker.ietf.org/doc/html/rfc7807),
// a JSON object with specific fields. In particular for the ACME protocol,
// the type field is well-defined and corresponds to a specific error condition.
//
// This error type is handled by the domain-specific error encoder provided to
// encodeResponse.
type ACMEError struct {
	Type       string `json:"type"`
	Title      string `json:"title,omitempty"`
	Detail     string `json:"detail,omitempty"`
	Instance   string `json:"instance,omitempty"`
	StatusCode int    `json:"-"`
}

func accountDoesNotExistError(detail string) *ACMEError {
	return &ACMEError{
		Type:       acmeErrorsURN + "accountDoesNotExist",
		Title:      "The request specified an account that does not exist",
		Detail:     detail,
		StatusCode: http.StatusBadRequest, // as per RFC https://datatracker.ietf.org/doc/html/rfc8555/#section-7.3.1
	}
}

func badNonceError(detail string) *ACMEError {
	return &ACMEError{
		Type:       acmeErrorsURN + "badNonce",
		Title:      "The client sent an unacceptable anti-replay nonce",
		Detail:     detail,
		StatusCode: http.StatusBadRequest, // as per RFC https://datatracker.ietf.org/doc/html/rfc8555/#section-6.5
	}
}

func (e *ACMEError) Error() string {
	s := e.Type
	if e.Title != "" {
		s += ": " + e.Title
	}
	if e.Detail != "" {
		s += ": " + e.Detail
	}
	return s
}

func acmeErrorEncoder(ctx context.Context, err error, w http.ResponseWriter, enc *json.Encoder, jsonErr *eu.JsonError) (handled bool) {
	var acmeErr *ACMEError
	if !errors.As(err, &acmeErr) {
		return false
	}
	statusCode := acmeErr.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}
	w.WriteHeader(statusCode)
	// ignoring error as response started being written at that point
	_ = json.NewEncoder(w).Encode(acmeErr)
	return true
}
