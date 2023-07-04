package service

import "github.com/fleetdm/fleet/v4/server/fleet"

type alreadyExistsError struct {
	fleet.ErrorWithUUID
}

func (a *alreadyExistsError) Error() string {
	return "Entity already exists"
}

func (a *alreadyExistsError) IsExists() bool {
	return true
}

func newAlreadyExistsError() *alreadyExistsError {
	return &alreadyExistsError{}
}

type notFoundError struct {
	fleet.ErrorWithUUID
}

func (e *notFoundError) Error() string {
	return "not found"
}

func (e *notFoundError) IsNotFound() bool {
	return true
}

func newNotFoundError() *notFoundError {
	return &notFoundError{}
}

// ssoErrCode defines a code for the type of SSO error that occurred. This is
// used to indicate to the frontend why the SSO login attempt failed so that
// it can provide a helpful and appropriate error message.
type ssoErrCode string

// List of valid SSO error codes.
const (
	ssoOtherError      ssoErrCode = "error"
	ssoOrgDisabled     ssoErrCode = "org_disabled"
	ssoAccountDisabled ssoErrCode = "account_disabled"
	ssoAccountInvalid  ssoErrCode = "account_invalid"
)

// ssoError is an error that occurs during the single sign-on flow. Its code
// indicates the type of error.
type ssoError struct {
	err  error
	code ssoErrCode

	fleet.ErrorWithUUID
}

func newSSOError(err error, code ssoErrCode) *ssoError {
	return &ssoError{
		err:  err,
		code: code,
	}
}

func (e *ssoError) Error() string {
	return string(e.code) + ": " + e.err.Error()
}

func (e *ssoError) Unwrap() error {
	return e.err
}
