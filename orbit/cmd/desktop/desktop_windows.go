//go:build windows
// +build windows

package main

import (
	_ "embed"
)

// For Windows we must use ico format for the icon,
// see https://github.com/getlantern/systray/blob/6065fda28be8c8d91aeb5e20de25e1600b8664a3/systray_windows.go#L850-L856.

// In the past we implemented some logic to detect the Windows theme but it was buggy,
// so as a temporary fix we are using the same colored icon for both themes.
// Such theme detection logic was removed in this PR: https://github.com/fleetdm/fleet/pull/16402.

//go:embed windows_app.ico
var iconLight []byte

//go:embed windows_app.ico
var iconDark []byte
