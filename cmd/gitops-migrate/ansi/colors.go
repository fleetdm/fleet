package ansi

const (
	// Regular colors.
	Black   = "\x1b[0;30m"
	Green   = "\x1b[0;32m"
	Yellow  = "\x1b[0;33m"
	Blue    = "\x1b[0;34m"
	Magenta = "\x1b[0;35m"
	Cyan    = "\x1b[0;36m"
	Red     = "\x1b[0;31m"
	White   = "\x1b[0;37m"

	// Bold colors.
	BoldBlack   = "\x1b[1;30m"
	BoldRed     = "\x1b[1;31m"
	BoldGreen   = "\x1b[1;32m"
	BoldYellow  = "\x1b[1;33m"
	BoldBlue    = "\x1b[1;34m"
	BoldMagenta = "\x1b[1;35m"
	BoldCyan    = "\x1b[1;36m"
	BoldWhite   = "\x1b[1;37m"

	// Standard reset.
	Reset = "\x1b[0m"
)
