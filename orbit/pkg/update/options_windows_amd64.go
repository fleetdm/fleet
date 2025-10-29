package update

import (
	"os"
	"path/filepath"

	"github.com/theupdateframework/go-tuf/client"
)

var defaultOptions = Options{
	RootDirectory:     `C:\Program Files\Orbit`,
	ServerURL:         DefaultURL,
	RootKeys:          defaultRootMetadata,
	LocalStore:        client.MemoryLocalStore(),
	InsecureTransport: false,
	Targets:           WindowsTargets,
}

func init() {
	// Set root directory to value of ProgramFiles environment variable if not set
	if dir := os.Getenv("ProgramFiles"); dir != "" {
		DefaultOptions.RootDirectory = filepath.Join(dir, "Orbit")
	}
}
