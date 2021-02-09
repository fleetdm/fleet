// package packaging provides tools for buildin Orbit installation packages.
package packaging

// Options are the configurable options provided for the package.
type Options struct {
	// FleetURL is the URL to the Fleet server.
	FleetURL string
	// EnrollSecret is the enroll secret used to authenticate to the Fleet
	// server.
	EnrollSecret string
	// Version is the version number for this package.
	Version string
	// Identifier is the identifier (eg. com.fleetdm.orbit) for the package product.
	Identifier string
	// StartService is a boolean indicating whether to start a system-specific
	// background service.
	StartService bool
	// Insecure enables insecure TLS connections for the generated package.
	Insecure bool
}
