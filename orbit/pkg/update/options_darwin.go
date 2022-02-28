package update

import (
	"path/filepath"

	"github.com/theupdateframework/go-tuf/client"
)

var defaultOptions = Options{
	RootDirectory:     "/var/lib/orbit",
	ServerURL:         defaultURL,
	RootKeys:          defaultRootKeys,
	LocalStore:        client.MemoryLocalStore(),
	InsecureTransport: false,
	Targets: Targets{
		"orbit": TargetInfo{
			Platform:   "macos",
			Channel:    "stable",
			TargetFile: "orbit",
		},
		"osqueryd": TargetInfo{
			Platform:             "macos-app",
			Channel:              "stable",
			TargetFile:           "osquery.app.tar.gz",
			ExtractedExecSubPath: filepath.Join("Contents", "MacOS", "osqueryd"),
		},
	},
}
