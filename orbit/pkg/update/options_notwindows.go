// +build !windows

package update

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/theupdateframework/go-tuf/client"
)

var (
	// DefaultOptions are the default options to use when creating an update
	// client.
	DefaultOptions = Options{
		RootDirectory:     "/var/lib/orbit",
		ServerURL:         defaultURL,
		RootKeys:          defaultRootKeys,
		LocalStore:        client.MemoryLocalStore(),
		InsecureTransport: false,
		Platform:          constant.PlatformName,
		OrbitChannel:      "stable",
		OsquerydChannel:   "stable",
	}
)
