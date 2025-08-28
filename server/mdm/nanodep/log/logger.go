package log

// Pacakge log is embedded (not imported) from:
// https://github.com/jessepeterson/go-log

// Logger is a generic logging interface to a structured, leveled, nest-able logger
type Logger interface {
	// Info logs using the info level
	Info(...any)

	// Debug logs using the debug level
	Debug(...any)

	// With nests the Logger
	// Usually for adding logging context to a sub-logger
	With(...any) Logger
}
