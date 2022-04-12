//go:build windows
// +build windows

package main

import _ "embed"

// For Windows we must use ico format for the icon,
// see https://github.com/getlantern/systray/blob/6065fda28be8c8d91aeb5e20de25e1600b8664a3/systray_windows.go#L850-L856.

//go:embed icon_white.ico
var icoBytes []byte
