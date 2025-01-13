package update

import (
	"github.com/theupdateframework/go-tuf/client"
)

var defaultOptions = Options{
	RootDirectory:     "/opt/orbit",
	ServerURL:         DefaultURL,
	RootKeys:          defaultRootMetadata,
	LocalStore:        client.MemoryLocalStore(),
	InsecureTransport: false,
	Targets:           LinuxTargets,
}
