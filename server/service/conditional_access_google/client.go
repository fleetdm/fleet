// Package conditional_access_google holds the clients of the Google conditional access
// integration: one registers this server with the relay on fleetdm.com, the other calls
// the Cloud Identity API for the customer with the customer's service account, delegated
// for the cloud-identity.devices scope.
package conditional_access_google

import (
	"context"
)

// Relay registers this server with the Google conditional access relay on fleetdm.com.
type Relay interface {
	// Register registers this server with fleetdm.com for the customer. Registering
	// again from the same server replaces the secret.
	Register(ctx context.Context, customerID string) (*Registration, error)
	// Deregister removes this server's registration from fleetdm.com. Returns an error
	// satisfying fleet.IsNotFound when fleetdm.com has no matching registration.
	Deregister(ctx context.Context, customerID, secret string) error
}

// CloudIdentity calls the Cloud Identity API for one customer, with a token minted from
// the customer's service account key as the admin email and reused until it expires.
type CloudIdentity interface {
	// CheckDelegation lists one device under the customer. It fails when the key isn't
	// delegated for the scope, the admin can't manage devices, or the customer ID is wrong.
	CheckDelegation(ctx context.Context) error
	// SetUnmanaged writes UNMANAGED on Fleet's client state of a DeviceUser
	// (devices/{device}/deviceUsers/{deviceUser}).
	SetUnmanaged(ctx context.Context, deviceUserName string) error
}

// CloudIdentityFactory builds a CloudIdentity for the credentials without calling Google.
// Callers build one per request or cron run from the credentials they just read, so a
// change takes effect on the next build. It is injected so tests can supply a fake.
type CloudIdentityFactory func(ctx context.Context, creds Credentials) (CloudIdentity, error)

// Credentials identify the customer and authenticate the Cloud Identity calls.
type Credentials struct {
	// ServiceAccountKey is the customer's service account key JSON.
	ServiceAccountKey []byte
	// AdminEmail is the Workspace admin the service account impersonates.
	AdminEmail string
	// CustomerID is the Google customer ID with its leading "C", as the Admin console shows it.
	CustomerID string
}
