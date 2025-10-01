package log

import "github.com/fleetdm/fleet/v4/cmd/gitops-migrate/ansi"

var (
	// Define the various ANSI colors used throughout the package.
	//
	// These are included here, as vars, to allow for easy overriding in tests.
	//
	// Log level colors.
	colorDBG = ansi.BoldBlack
	colorINF = ansi.BoldGreen
	colorWRN = ansi.BoldYellow
	colorERR = ansi.BoldRed
	colorFTL = ansi.BoldRed
	// The color used for _keys_ when key-value pairs are passed to 'log'.
	colorKey = ansi.White
	// The color used for _values_ when key-value pairs are passed to 'log'.
	colorVal = ansi.White
	// The color used for the caller file name and line number, when the option is
	// enabled.
	colorCaller = ansi.BoldBlue
	// The ANSI reset sequence (to unset a color).
	colorReset = ansi.Reset
)
