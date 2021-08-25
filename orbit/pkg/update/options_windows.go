package update

import (
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/theupdateframework/go-tuf/client"
)

var (
	// DefaultOptions are the default options to use when creating an update
	// client.
	DefaultOptions = Options{
		RootDirectory:     `C:\Program Files\Orbit`,
		ServerURL:         defaultURL,
		RootKeys:          defaultRootKeys,
		LocalStore:        client.MemoryLocalStore(),
		InsecureTransport: false,
		Platform:          constant.PlatformName,
		OrbitChannel:      "stable",
		OsquerydChannel:   "stable",
	}
)

func init() {
	// Set root directory to value of ProgramFiles environment variable if not set
	if dir := os.Getenv("ProgramFiles"); dir != "" {
		DefaultOptions.RootDirectory = filepath.Join(dir, "Orbit")
	}
}
