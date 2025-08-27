package stdlogfmt

import (
	stdlog "log"
	"strings"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log"
)

// Logger wraps a standard library logger and adapts it to pkg/log.
type Logger struct {
	stdLogger *stdlog.Logger
	context   []any
	logDebug  bool
}

// New creates a new logger that adapts the standard log package to pkg/log.
func New(logger *stdlog.Logger, logDebug bool) *Logger {
	return &Logger{
		stdLogger: logger,
		logDebug:  logDebug,
	}
}

func (l *Logger) print(args ...any) {
	f := strings.Repeat(" %s=%v", len(args)/2)[1:]
	if len(args)%2 == 1 {
		f += " UNKNOWN=%v"
	}
	l.stdLogger.Printf(f, args...)
}

// Info logs using the "info" level
func (l *Logger) Info(args ...any) {
	logs := []any{"level", "info"}
	logs = append(logs, l.context...)
	logs = append(logs, args...)
	l.print(logs...)
}

// Info logs using the "debug" level
func (l *Logger) Debug(args ...any) {
	if l.logDebug {
		logs := []any{"level", "debug"}
		logs = append(logs, l.context...)
		logs = append(logs, args...)
		l.print(logs...)
	}
}

// With creates a new logger using args as context
func (l *Logger) With(args ...any) log.Logger {
	newLogger := &Logger{
		stdLogger: l.stdLogger,
		context:   l.context,
		logDebug:  l.logDebug,
	}
	newLogger.context = append(newLogger.context, args...)
	return newLogger
}
