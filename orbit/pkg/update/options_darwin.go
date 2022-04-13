package update

import (
	"github.com/theupdateframework/go-tuf/client"
)

var defaultOptions = Options{
	RootDirectory:     "/var/lib/orbit",
	ServerURL:         defaultURL,
	RootKeys:          defaultRootKeys,
	LocalStore:        client.MemoryLocalStore(),
	InsecureTransport: false,
	Targets:           DarwinTargets,
}
