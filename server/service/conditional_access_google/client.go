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
	// SignInURL is fleetdm.com's sign-in page for an iOS or iPadOS host, which carries
	// the host's single-use registration token back to this server.
	SignInURL(customerID, hostUUID, token string) (string, error)
}

// CloudIdentity calls the Cloud Identity API for one customer as the delegated admin.
type CloudIdentity interface {
	// CheckDelegation lists one device under the customer. It proves read access only.
	CheckDelegation(ctx context.Context) error
	// SetClientState writes the set fields of state on Fleet's client state of a
	// DeviceUser. It returns a fleet.IsNotFound error when Google has no such DeviceUser.
	SetClientState(ctx context.Context, deviceUserName string, state fleet.GoogleConditionalAccessClientState) error
	// ListDeviceUserNames lists the DeviceUser names of a device, by its device ID. It
	// returns a fleet.IsNotFound error when Google has no such device.
	ListDeviceUserNames(ctx context.Context, deviceID string) ([]string, error)
	// FindDeviceIDs returns the device IDs of the customer's records whose deviceId field
	// is androidDeviceID, the Android Management API device ID. Google doesn't keep the
	// field unique, so a re-enrolled device can have more than one record.
	FindDeviceIDs(ctx context.Context, androidDeviceID string) ([]string, error)
	// DeviceExists reports whether Google still has the device, by its device ID. A
	// device that left the customer's Workspace is a 403, returned as an IsStale error
	// for the caller to decide on.
	DeviceExists(ctx context.Context, deviceID string) (bool, error)
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
