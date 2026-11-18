// Package conditional_access_google holds the Google conditional access clients: the
// fleetdm.com relay and the Cloud Identity API.
package conditional_access_google

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Relay registers this server with the relay on fleetdm.com. Customer IDs include the
// leading "C". serverURL is this server's current URL, which fleetdm.com calls back.
type Relay interface {
	// Register replaces the secret when called again from the same server.
	Register(ctx context.Context, serverURL, customerID string) (*Registration, error)
	// Deregister returns a fleet.IsNotFound error when there's no matching registration.
	Deregister(ctx context.Context, serverURL, customerID, secret string) error
}

// CloudIdentity calls the Cloud Identity API for one customer as the delegated admin.
type CloudIdentity interface {
	// CheckDelegation lists one device under the customer. It proves read access only.
	CheckDelegation(ctx context.Context) error
	// SetClientState writes the set fields of state on Fleet's client state of a
	// DeviceUser. It returns a fleet.IsNotFound error when Google has no such DeviceUser.
	SetClientState(ctx context.Context, deviceUserName string, state fleet.GoogleConditionalAccessClientState) error
}

// CloudIdentityFactory builds a CloudIdentity without calling Google. Build one per
// request or cron run.
type CloudIdentityFactory func(ctx context.Context, creds Credentials) (CloudIdentity, error)

// Credentials identify the customer and authenticate the Cloud Identity calls.
type Credentials struct {
	ServiceAccountKey []byte
	// AdminEmail is the Workspace admin the service account impersonates.
	AdminEmail string
	// CustomerID includes the leading "C".
	CustomerID string
}
