package stdlogfmt

import (
	"fmt"
	stdlog "log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/log"
)

// Logger wraps a standard library logger and adapts it to pkg/log.
type Logger struct {
	logger  *stdlog.Logger
	context []interface{}
	debug   bool
	depth   int
	ts      bool
}

type Option func(*Logger)

// WithLogger sets the Go standard logger to use.
func WithLogger(logger *stdlog.Logger) Option {
	return func(l *Logger) {
		l.logger = logger
	}
}

// WithDebug turns on debug logging.
func WithDebug() Option {
	return func(l *Logger) {
		l.debug = true
	}
}

// WithDebugFlag sets debug logging on or off.
func WithDebugFlag(flag bool) Option {
	return func(l *Logger) {
		l.debug = flag
	}
}

// WithCallerDepth sets the call depth of the logger for filename and line
// logging. Set depth to 0 to disable filename and line logging.
func WithCallerDepth(depth int) Option {
	return func(l *Logger) {
		l.depth = depth
	}
}

// WithoutTimestamp disables outputting an RFC3339 timestamp.
func WithoutTimestamp() Option {
	return func(l *Logger) {
		l.ts = false
	}
}

// New creates a new logger that adapts the Go standard log package to Logger.
func New(opts ...Option) *Logger {
	l := &Logger{
		logger: stdlog.New(os.Stderr, "", 0),
		depth:  1,
		ts:     true,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *Logger) print(args ...interface{}) {
	if l.ts {
		args = append([]interface{}{"ts", time.Now().Format(time.RFC3339)}, args...)
	}
	if l.depth > 0 {
		_, filename, line, ok := runtime.Caller(l.depth + 1)
		if ok {
			caller := fmt.Sprintf("%s:%d", filepath.Base(filename), line)
			args = append(args, "caller", caller)
		}
	}
	f := strings.Repeat(" %s=%v", len(args)/2)[1:]
	if len(args)%2 == 1 {
		f += " UNKNOWN=%v"
	}
	l.logger.Printf(f, args...)
}

// Info logs using the "info" level
func (l *Logger) Info(args ...interface{}) {
	logs := []interface{}{"level", "info"}
	logs = append(logs, l.context...)
	logs = append(logs, args...)
	l.print(logs...)
}

// Info logs using the "debug" level
func (l *Logger) Debug(args ...interface{}) {
	if l.debug {
		logs := []interface{}{"level", "debug"}
		logs = append(logs, l.context...)
		logs = append(logs, args...)
		l.print(logs...)
	}
}

// With creates a new logger using args as context
func (l *Logger) With(args ...interface{}) log.Logger {
	l2 := *l
	l2.context = append(l2.context, args...)
	return &l2
}
