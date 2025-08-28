package log

// Pacakge log is embedded (not imported) from:
// https://github.com/jessepeterson/go-log

// nopLogger does nothing
type nopLogger struct{}

// Info does nothing
func (*nopLogger) Info(_ ...any) {}

// Debug does nothing
func (*nopLogger) Debug(_ ...any) {}

// With returns (the same) logger
func (logger *nopLogger) With(_ ...any) Logger {
	return logger
}

// NopLogger is a Logger that does nothing
var NopLogger = &nopLogger{}
