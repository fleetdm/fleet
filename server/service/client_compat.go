// Package service provides backward-compatible type aliases for client types
// that have been moved to the github.com/fleetdm/fleet/v4/client package.
// These aliases allow existing code that imports server/service to continue
// working during the migration period.
//
// TODO: Remove this file once all consumers have been updated to import
// github.com/fleetdm/fleet/v4/client directly.
package service

import (
	fleetclient "github.com/fleetdm/fleet/v4/client"
)

// Client type aliases - these types have moved to github.com/fleetdm/fleet/v4/client
type (
	Client       = fleetclient.Client
	ClientOption = fleetclient.ClientOption
	OrbitClient  = fleetclient.OrbitClient
	DeviceClient = fleetclient.DeviceClient
)

// Constructor function aliases
var (
	NewClient       = fleetclient.NewClient
	NewOrbitClient  = fleetclient.NewOrbitClient
	NewDeviceClient = fleetclient.NewDeviceClient
)

// Error variable aliases
var (
	ErrUnauthenticated     = fleetclient.ErrUnauthenticated
	ErrEndUserAuthRequired = fleetclient.ErrEndUserAuthRequired
	ErrMissingLicense      = fleetclient.ErrMissingLicense
)

// ClientOption function aliases
var (
	EnableClientDebug    = fleetclient.EnableClientDebug
	WithCustomHeaders    = fleetclient.WithCustomHeaders
	SetClientOutputWriter = fleetclient.SetClientOutputWriter
	SetClientErrorWriter = fleetclient.SetClientErrorWriter
)

// Additional exported interfaces and types
type (
	NotSetupErr         = fleetclient.NotSetupErr
	NotFoundErr         = fleetclient.NotFoundErr
	SetupAlreadyErr     = fleetclient.SetupAlreadyErr
	ConflictErr         = fleetclient.ConflictErr
	OnGetConfigErrFuncs = fleetclient.OnGetConfigErrFuncs
)
