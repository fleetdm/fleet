package update

import (
	"github.com/theupdateframework/go-tuf/client"
)

var defaultOptions = Options{
	RootDirectory:     "/opt/orbit",
	ServerURL:         defaultURL,
	RootKeys:          defaultRootMetadata,
	LocalStore:        client.MemoryLocalStore(),
	InsecureTransport: false,
	Targets:           LinuxArm64Targets,
}
