// Package api provides the public API for the ACME service modiule.
// External code should use this package to interact with ACME.
package api

import "github.com/fleetdm/fleet/v4/server/mdm/acme/internal/redis_nonces_store"

// Service is the composite interface for the ACME service modiule.
// It embeds all method-specific interfaces. Bootstrap returns this type.
type Service interface {
	DirectoryNonceService
	AccountService
	EnrollmentService
	NoncesStore() *redis_nonces_store.RedisNoncesStore
}
