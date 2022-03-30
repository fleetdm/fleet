//go:build !windows
// +build !windows

package main

import _ "embed"

//go:embed icon_white.png
var icoBytes []byte
