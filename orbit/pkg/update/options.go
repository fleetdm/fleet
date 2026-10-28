package update

import (
	"fmt"
	"os/exec"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
)

// DefaultOptions are the default options to use when creating an update
// client.
var DefaultOptions = defaultOptions

var (
	DarwinTargets = Targets{
		constant.OrbitTUFTargetName: TargetInfo{
			Platform:   "macos",
			Channel:    "stable",
			TargetFile: "orbit",
		},
		constant.OsqueryTUFTargetName: TargetInfo{
			Platform:             "macos-app",
			Channel:              "stable",
			TargetFile:           "osqueryd.app.tar.gz",
			ExtractedExecSubPath: []string{"osquery.app", "Contents", "MacOS", "osqueryd"},
		},
	}

	LinuxTargets = Targets{
		constant.OrbitTUFTargetName: TargetInfo{
			Platform:   "linux",
			Channel:    "stable",
			TargetFile: "orbit",
		},
		constant.OsqueryTUFTargetName: TargetInfo{
			Platform:   "linux",
			Channel:    "stable",
			TargetFile: "osqueryd",
		},
	}

	LinuxArm64Targets = Targets{
		constant.OrbitTUFTargetName: TargetInfo{
			Platform:   "linux-arm64",
			Channel:    "stable",
			TargetFile: "orbit",
		},
		constant.OsqueryTUFTargetName: TargetInfo{
			Platform:   "linux-arm64",
			Channel:    "stable",
			TargetFile: "osqueryd",
		},
	}

	WindowsArm64Targets = Targets{
		constant.OrbitTUFTargetName: TargetInfo{
			Platform:   "windows-arm64",
			Channel:    "stable",
			TargetFile: "orbit.exe",
		},
		// NOTE: Currently osquery doesn't fully support ARM64, this is experimental
		constant.OsqueryTUFTargetName: TargetInfo{
			Platform:   "windows-arm64",
			Channel:    "stable",
			TargetFile: "osqueryd.exe",
		},
	}

	WindowsTargets = Targets{
		constant.OrbitTUFTargetName: TargetInfo{
			Platform:   "windows",
			Channel:    "stable",
			TargetFile: "orbit.exe",
		},
		constant.OsqueryTUFTargetName: TargetInfo{
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

	DesktopWindowsTarget = TargetInfo{
		Platform:   "windows",
		Channel:    "stable",
		TargetFile: constant.DesktopAppExecName + ".exe",
	}

	DesktopWindowsArm64Target = TargetInfo{
		Platform:   "windows-arm64",
		Channel:    "stable",
		TargetFile: constant.DesktopAppExecName + ".exe",
	}

	DesktopLinuxTarget = TargetInfo{
		Platform:             "linux",
		Channel:              "stable",
		TargetFile:           "desktop.tar.gz",
		ExtractedExecSubPath: []string{"fleet-desktop", constant.DesktopAppExecName},
		CustomCheckExec: func(execPath string) error {
			cmd := exec.Command(execPath, "--help")
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("exec new version: %s: %w", string(out), err)
			}
			return nil
		},
	}

	DesktopLinuxArm64Target = TargetInfo{
		Platform:             "linux-arm64",
		Channel:              "stable",
		TargetFile:           "desktop.tar.gz",
		ExtractedExecSubPath: []string{"fleet-desktop", constant.DesktopAppExecName},
		CustomCheckExec: func(execPath string) error {
			cmd := exec.Command(execPath, "--help")
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("exec new version: %s: %w", string(out), err)
			}
			return nil
		},
	}

	SwiftDialogMacOSTarget = TargetInfo{
		Platform:             "macos",
		Channel:              "stable",
		TargetFile:           "swiftDialog.app.tar.gz",
		ExtractedExecSubPath: []string{"Dialog.app", "Contents", "MacOS", "Dialog"},
	}

	EscrowBuddyMacOSTarget = TargetInfo{
		Platform:   "macos",
		Channel:    "stable",
		TargetFile: "escrowBuddy.pkg",
	}
)
