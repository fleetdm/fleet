// Package api provides the public API for the ACME service module.
// External code should use this package to interact with ACME.
package api

import "github.com/fleetdm/fleet/v4/server/mdm/acme/internal/redis_nonces_store"

// Service is the composite interface for the ACME service module.
// It embeds all method-specific interfaces. Bootstrap returns this type.
type Service interface {
	DirectoryNonceService
	AccountService
	EnrollmentService
	AuthorizationService
	ChallengeService
	NoncesStore() *redis_nonces_store.RedisNoncesStore
}
