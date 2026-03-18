// Package api provides the public API for the ACME bounded context.
// External code should use this package to interact with ACME.
package api

// Service is the composite interface for the ACME bounded context.
// It embeds all method-specific interfaces. Bootstrap returns this type.
type Service interface {
	DirectoryNonceService
}
