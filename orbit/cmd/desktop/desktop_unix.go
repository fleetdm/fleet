//go:build darwin || linux
// +build darwin linux

package main

import _ "embed"

//go:embed icon_white.png
var icoBytes []byte
