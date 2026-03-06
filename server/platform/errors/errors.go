// Package errors provides error classification primitives used across the
// codebase. These are intentionally kept free of HTTP or other transport
// dependencies so that low-level packages (datastores, context helpers) can
// use them without pulling in higher-level concerns.
package errors

import "errors"

// Cause returns the root error in err's chain.
func Cause(err error) error {
	for {
		uerr := errors.Unwrap(err)
		if uerr == nil {
			return err
		}
		err = uerr
	}
}

// ErrWithIsClientError is an interface for errors that explicitly specify
// whether they are client errors or not. By default, errors are treated as
// server errors.
type ErrWithIsClientError interface {
	error
	IsClientError() bool
}

// NotFoundError is an interface for errors when a resource cannot be found.
type NotFoundError interface {
	error
	IsNotFound() bool
}

// IsNotFound returns true if err is a not-found error.
func IsNotFound(err error) bool {
	var nfe NotFoundError
	if errors.As(err, &nfe) {
		return nfe.IsNotFound()
	}
	return false
}
