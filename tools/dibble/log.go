package main

import (
	"fmt"
	"io"
	"os"

	"github.com/fleetdm/fleet/v4/tools/dibble/themes"
)

// stdoutish / stderrish are package-level writers so tests can swap them.
var (
	stdoutish io.Writer = os.Stdout
	stderrish io.Writer = os.Stderr
)

// printf writes a line tagged with the tapir snout glyph.
func printf(format string, a ...any) {
	fmt.Fprintf(stdoutish, themes.TapirSnout+" "+format+"\n", a...)
}

// warnf writes a line to stderr without the glyph.
func warnf(format string, a ...any) {
	fmt.Fprintf(stderrish, "dibble: "+format+"\n", a...)
}
