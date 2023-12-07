package update

// Exported so that it can be used in tools/ (so that it can be built for
// Windows and tested on a Windows machine). Otherwise not meant to be used
// from outside this package.
type WindowsMDMEnrollmentArgs struct {
	DiscoveryURL string
	HostUUID     string
	OrbitNodeKey string
}
