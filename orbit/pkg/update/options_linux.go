package update

import (
	"runtime"

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

func init() {
	if runtime.GOARCH == "arm64" {
		defaultOptions.Targets = LinuxArm64Targets
	}
}
