package logging

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/fleetdm/fleet/v4/secure"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
)

type filesystemLogWriter struct {
	writer io.WriteCloser
}

// NewFilesystemLogWriter creates a log file for osquery status/result logs.
// The logFile can be rotated by sending a `SIGHUP` signal to Fleet if
// enableRotation is true
func NewFilesystemLogWriter(path string, appLogger log.Logger, enableRotation bool, enableCompression bool) (*filesystemLogWriter, error) {
	if enableRotation {
		// Use lumberjack logger that supports rotation
		osquerydLogger := &lumberjack.Logger{
			Filename:   path,
			MaxSize:    500, // megabytes
			MaxBackups: 3,
			MaxAge:     28, //days
			Compress:   enableCompression,
		}
		appLogger = log.With(appLogger, "component", "osqueryd-logger")
		sig := make(chan os.Signal)
		signal.Notify(sig, syscall.SIGHUP)
		go func() {
			for {
				<-sig //block on signal
				if err := osquerydLogger.Rotate(); err != nil {
					appLogger.Log("err", err)
				}
			}
		}()
		return &filesystemLogWriter{osquerydLogger}, nil
	}
	// no log rotation, use "raw" bufio implementation
	writer, err := newRawLogWriter(path)
	if err != nil {
		return nil, errors.Wrap(err, "create new raw logger")
	}
	return &filesystemLogWriter{writer}, nil
}

// If writer is based on bufio we want to flush after a batch of
// writes so log entry gets completely written to the logfile.
type flusher interface {
	Flush() error
}

// Write writes the provided logs to the filesystem
func (l *filesystemLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	for _, log := range logs {
		// Add newline to separate logs in output file
		log = append(log, '\n')
		if _, err := l.writer.Write(log); err != nil {
			return errors.Wrap(err, "writing log")
		}
	}
	if flusher, ok := l.writer.(flusher); ok {
		if err := flusher.Flush(); err != nil {
			return errors.Wrap(err, "flushing log")
		}
	}
	return nil
}

// rawLogWriter implements writing to logs directly through bufio
type rawLogWriter struct {
	file *os.File
	buff *bufio.Writer
	mtx  sync.Mutex
}

func newRawLogWriter(path string) (*rawLogWriter, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	buff := bufio.NewWriter(file)
	return &rawLogWriter{file: file, buff: buff}, nil
}

// Write bytes to file
func (l *rawLogWriter) Write(b []byte) (int, error) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if l.buff == nil || l.file == nil {
		return 0, errors.New("filesystemLogWriter: can't write to closed file")
	}
	if _, statErr := os.Stat(l.file.Name()); errors.Is(statErr, os.ErrNotExist) {
		f, err := secure.OpenFile(l.file.Name(), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			return 0, errors.Wrapf(err, "create file for filesystemLogWriter %s", l.file.Name())
		}
		l.file = f
		l.buff = bufio.NewWriter(f)
	}
	return l.buff.Write(b)
}

// Flush writes all buffered bytes to log file
func (l *rawLogWriter) Flush() error {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if l.buff == nil {
		return errors.New("can't write to a closed file")
	}
	return l.buff.Flush()
}

// Close log file
func (l *rawLogWriter) Close() error {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if l.buff != nil {
		if err := l.buff.Flush(); err != nil {
			return err
		}
		l.buff = nil
	}
	if l.file != nil {
		if err := l.file.Close(); err != nil {
			return err
		}
		l.file = nil
	}

	return nil
}
