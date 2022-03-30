//go:build darwin
// +build darwin

// TODO(lucas): Once we support Linux, amend the above build tags.

package main

import _ "embed"

//go:embed icon_white.png
var icoBytes []byte
