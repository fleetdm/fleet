package log

// Pacakge log is embedded (not imported) from:
// https://github.com/jessepeterson/go-log

// Logger is a generic logging interface to a structured, leveled, nest-able logger
type Logger interface {
	// Info logs using the info level
	Info(...interface{})

	// Debug logs using the debug level
	Debug(...interface{})

	// With nests the Logger
	// Usually for adding logging context to a sub-logger
	With(...interface{}) Logger
}
