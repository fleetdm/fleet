package service

type alreadyExistsError struct{}

func (a alreadyExistsError) Error() string {
	return "Entity already exists"
}

func (a alreadyExistsError) IsExists() bool {
	return true
}

type notFoundError struct{}

func (e notFoundError) Error() string {
	return "not found"
}

func (e notFoundError) IsNotFound() bool {
	return true
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
}

func (e ssoError) Error() string {
	return string(e.code) + ": " + e.err.Error()
}

func (e ssoError) Unwrap() error {
	return e.err
}
