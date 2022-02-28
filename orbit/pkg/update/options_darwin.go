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
	Targets: Targets{
		"orbit": TargetInfo{
			Platform:   "macos",
			Channel:    "stable",
			TargetFile: "orbit",
		},
		"osqueryd": TargetInfo{
			Platform:             "macos-app",
			Channel:              "stable",
			TargetFile:           "osqueryd.app.tar.gz",
			ExtractedExecSubPath: []string{"osquery.app", "Contents", "MacOS", "osqueryd"},
		},
	},
}
