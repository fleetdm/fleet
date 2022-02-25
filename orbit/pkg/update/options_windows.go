package update

import (
	"os"
	"path/filepath"

	"github.com/theupdateframework/go-tuf/client"
)

var defaultOptions = Options{
	RootDirectory:     `C:\Program Files\Orbit`,
	ServerURL:         defaultURL,
	RootKeys:          defaultRootKeys,
	LocalStore:        client.MemoryLocalStore(),
	InsecureTransport: false,
	Targets: Targets{
		"orbit": TargetInfo{
			Platform:   "windows",
			Channel:    "stable",
			TargetFile: "orbit.exe",
		},
		"osqueryd": TargetInfo{
			Platform:   "windows",
			Channel:    "stable",
			TargetFile: "osqueryd.exe",
		},
	},
}

func init() {
	// Set root directory to value of ProgramFiles environment variable if not set
	if dir := os.Getenv("ProgramFiles"); dir != "" {
		DefaultOptions.RootDirectory = filepath.Join(dir, "Orbit")
	}
}
