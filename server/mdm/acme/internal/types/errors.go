package types

import (
	"net/http"
)

const (
	acmeErrorsURN = "urn:ietf:params:acme:error:"
	// NOTE: ideally this would be a valid, dereferenceable link to human-readable documentation,
	// but it's ok if it's not too. See https://datatracker.ietf.org/doc/html/rfc8555/#section-6.7
	fleetCustomErrorsURI = "https://fleetdm.com/acme/error/"
)

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

var (
	enrollmentNotFound = EnrollmentNotFoundError("")
	serverInternal     = InternalServerError("")
)

func (e *ACMEError) ShouldReturnNonce() bool {
	if e == nil {
		return false
	}
	switch e.Type {
	case enrollmentNotFound.Type, serverInternal.Type:
		return false
	default:
		return true
	}
}

func EnrollmentNotFoundError(detail string) *ACMEError {
	return &ACMEError{
		Type:       fleetCustomErrorsURI + "enrollmentNotFound",
		Title:      "The specified enrollment does not exist",
		Detail:     detail,
		StatusCode: http.StatusNotFound,
	}
}

func AccountDoesNotExistError(detail string) *ACMEError {
	return &ACMEError{
		Type:       acmeErrorsURN + "accountDoesNotExist",
		Title:      "The request specified an account that does not exist",
		Detail:     detail,
		StatusCode: http.StatusBadRequest, // as per RFC https://datatracker.ietf.org/doc/html/rfc8555/#section-7.3.1
	}
}

func AccountRevokedError(detail string) *ACMEError {
	return &ACMEError{
		Type:       fleetCustomErrorsURI + "accountRevoked",
		Title:      "The request specified an account that is revoked",
		Detail:     detail,
		StatusCode: http.StatusBadRequest,
	}
}

func TooManyAccountsError(detail string) *ACMEError {
	return &ACMEError{
		Type:       fleetCustomErrorsURI + "tooManyAccounts",
		Title:      "Too many accounts already exist for this enrollment",
		Detail:     detail,
		StatusCode: http.StatusBadRequest,
	}
}

func TooManyOrdersError(detail string) *ACMEError {
	return &ACMEError{
		Type:       fleetCustomErrorsURI + "tooManyOrders",
		Title:      "Too many orders already exist for this account",
		Detail:     detail,
		StatusCode: http.StatusBadRequest,
	}
}

// NOTE: surprisingly, the RFC does not document an error and status code for
// a POST-as-GET to an order URL with an ID that does not exist, so this is a
// Fleet custom error code.
func OrderDoesNotExistError(detail string) *ACMEError {
	return &ACMEError{
		Type:       fleetCustomErrorsURI + "orderDoesNotExist",
		Title:      "The request specified an order that does not exist",
		Detail:     detail,
		StatusCode: http.StatusNotFound,
	}
}

func OrderNotFinalizedError(detail string) *ACMEError {
	return &ACMEError{
		Type:       fleetCustomErrorsURI + "orderNotFinalized",
		Title:      "The request attempted to download a certificate for an order that is not finalized",
		Detail:     detail,
		StatusCode: http.StatusBadRequest,
	}
}

func RejectedIdentifierError(detail string) *ACMEError {
	return &ACMEError{
		Type:       acmeErrorsURN + "rejectedIdentifier",
		Title:      "The server will not issue certificates for the identifier",
		Detail:     detail,
		StatusCode: http.StatusBadRequest,
	}
}

func UnsupportedIdentifierError(detail string) *ACMEError {
	return &ACMEError{
		Type:       acmeErrorsURN + "unsupportedIdentifier",
		Title:      "An identifier is of an unsupported type",
		Detail:     detail,
		StatusCode: http.StatusBadRequest,
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

func BadCSRError(detail string) *ACMEError {
	return &ACMEError{
		Type:       acmeErrorsURN + "badCSR",
		Title:      "The CSR is unacceptable (e.g., due to a short key)",
		Detail:     detail,
		StatusCode: http.StatusBadRequest,
	}
}

func InternalServerError(detail string) *ACMEError {
	return &ACMEError{
		Type:       acmeErrorsURN + "serverInternal",
		Title:      "The server experienced an internal error",
		Detail:     detail,
		StatusCode: http.StatusInternalServerError,
	}
}

func BadPublicKeyError(detail string) *ACMEError {
	return &ACMEError{
		Type:       acmeErrorsURN + "badPublicKey",
		Title:      "The JWS was signed by a public key the server does not support",
		Detail:     detail,
		StatusCode: http.StatusBadRequest,
	}
}

func BadSignatureAlgorithmError(detail string) *ACMEError {
	return &ACMEError{
		Type:       acmeErrorsURN + "badSignatureAlgorithm",
		Title:      "The JWS was signed with an algorithm the server does not support",
		Detail:     detail,
		StatusCode: http.StatusBadRequest,
	}
}

func UnauthorizedError(detail string) *ACMEError {
	return &ACMEError{
		Type:       acmeErrorsURN + "unauthorized",
		Title:      "The client lacks sufficient authorization",
		Detail:     detail,
		StatusCode: http.StatusUnauthorized,
	}
}

func MalformedError(detail string) *ACMEError {
	return &ACMEError{
		Type:       acmeErrorsURN + "malformed",
		Title:      "The request message was malformed",
		Detail:     detail,
		StatusCode: http.StatusBadRequest,
	}
}

// Custom Fleet error code, as RFC does not document what to return.
func AuthorizationDoesNotExistError(detail string) *ACMEError {
	return &ACMEError{
		Type:       fleetCustomErrorsURI + "authorizationDoesNotExist",
		Title:      "The specified authorization does not exist for the account",
		Detail:     detail,
		StatusCode: http.StatusNotFound,
	}
}

// Custom Fleet error code, as RFC does not document what to return.
func ChallengeDoesNotExistError(detail string) *ACMEError {
	return &ACMEError{
		Type:       fleetCustomErrorsURI + "challengeDoesNotExist",
		Title:      "The specified challenge does not exist for the authorization",
		Detail:     detail,
		StatusCode: http.StatusNotFound,
	}
}

func CertificateDoesNotExistError(detail string) *ACMEError {
	return &ACMEError{
		Type:       fleetCustomErrorsURI + "certificateDoesNotExist",
		Title:      "The order is finalized but the certificate does not exist for the order",
		Detail:     detail,
		StatusCode: http.StatusNotFound,
	}
}

func OrderNotReadyError(detail string) *ACMEError {
	return &ACMEError{
		Type:       acmeErrorsURN + "orderNotReady",
		Title:      "The request attempted to finalize an order that is not ready to be finalized",
		Detail:     detail,
		StatusCode: http.StatusForbidden, // as per RFC https://datatracker.ietf.org/doc/html/rfc8555/#section-6.7
	}
}

func InvalidChallengeStatusError(detail string) *ACMEError {
	return &ACMEError{
		Type:       fleetCustomErrorsURI + "invalidChallengeStatus",
		Title:      "The challenge is not in a valid status for the attempted operation",
		Detail:     detail,
		StatusCode: http.StatusBadRequest,
	}
}

// Draft ACME device attest RFC https://datatracker.ietf.org/doc/html/draft-acme-device-attest-01#name-new-error-types
func BadAttestationStatementError(detail string) *ACMEError {
	return &ACMEError{
		Type:       acmeErrorsURN + "badAttestationStatement",
		Title:      "The attestation statement provided by the client was unacceptable",
		Detail:     detail,
		StatusCode: http.StatusBadRequest,
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
