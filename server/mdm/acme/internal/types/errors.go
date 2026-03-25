package types

import (
	"net/http"
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

func AccountDoesNotExistError(detail string) *ACMEError {
	return &ACMEError{
		Type:       acmeErrorsURN + "accountDoesNotExist",
		Title:      "The request specified an account that does not exist",
		Detail:     detail,
		StatusCode: http.StatusBadRequest, // as per RFC https://datatracker.ietf.org/doc/html/rfc8555/#section-7.3.1
	}
}

func BadNonceError(detail string) *ACMEError {
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
