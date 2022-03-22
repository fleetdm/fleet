package update

import "github.com/fleetdm/fleet/v4/orbit/pkg/constant"

// DefaultOptions are the default options to use when creating an update
// client.
var DefaultOptions = defaultOptions

var (
	DarwinTargets = Targets{
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
	}

	DarwinLegacyTargets = Targets{
		"orbit": TargetInfo{
			Platform:   "macos",
			Channel:    "stable",
			TargetFile: "orbit",
		},
		"osqueryd": TargetInfo{
			Platform:   "macos",
			Channel:    "stable",
			TargetFile: "osqueryd",
		},
	}

	LinuxTargets = Targets{
		"orbit": TargetInfo{
			Platform:   "linux",
			Channel:    "stable",
			TargetFile: "orbit",
		},
		"osqueryd": TargetInfo{
			Platform:   "linux",
			Channel:    "stable",
			TargetFile: "osqueryd",
		},
	}

	WindowsTargets = Targets{
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
	}

	DesktopMacOSTarget = TargetInfo{
		Platform:             "macos",
		Channel:              "stable",
		TargetFile:           "desktop.app.tar.gz",
		ExtractedExecSubPath: []string{"Fleet Desktop.app", "Contents", "MacOS", constant.DesktopAppExecName},
	}
)
