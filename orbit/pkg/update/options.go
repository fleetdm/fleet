package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

	DesktopLinuxTarget = TargetInfo{
		Platform:             "linux",
		Channel:              "stable",
		TargetFile:           "desktop.tar.gz",
		ExtractedExecSubPath: []string{"fleet-desktop", constant.DesktopAppExecName},
		CustomCheckExec: func(execPath string) error {
			cmd := exec.Command(execPath, "--help")
			cmd.Env = append(cmd.Env, fmt.Sprintf("LD_LIBRARY_PATH=%s:%s", filepath.Dir(execPath), os.ExpandEnv("$LD_LIBRARY_PATH")))
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

	NudgeMacOSTarget = TargetInfo{
		Platform:             "macos",
		Channel:              "stable",
		TargetFile:           "nudge.app.tar.gz",
		ExtractedExecSubPath: []string{"Nudge.app", "Contents", "MacOS", "Nudge"},
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
