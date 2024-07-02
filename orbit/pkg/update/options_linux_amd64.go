package update

import (
	"github.com/theupdateframework/go-tuf/client"
)

var defaultOptions = Options{
	RootDirectory:     "/opt/orbit",
	ServerURL:         defaultURL,
	RootKeys:          defaultRootKeys,
	LocalStore:        client.MemoryLocalStore(),
	InsecureTransport: false,
	Targets:           LinuxTargets,
}
