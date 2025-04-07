package logging

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/go-kit/log"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

type filesystemLogWriter struct {
	writer io.WriteCloser
}

// NewFilesystemLogWriter creates a logger that writes to a file.
//
// The logFile can be rotated by sending a `SIGHUP` signal to Fleet if
// enableRotation is true
//
// The enableCompression argument is only used when enableRotation is true.
func NewFilesystemLogWriter(path string, appLogger log.Logger, enableRotation, enableCompression bool, maxSize, maxAge, maxBackups int) (*filesystemLogWriter, error) {
	// Fail early if the process does not have the necessary
	// permissions to open the file at path.
	file, err := openFile(path)
	if err != nil {
		return nil, fmt.Errorf("perm check: %w", err)
	}
	if !enableRotation {
		// no log rotation, use "raw" bufio implementation
		return &filesystemLogWriter{
			writer: newRawLogWriter(file),
		}, nil
	}
	// Use lumberjack logger that supports rotation
	file.Close()
	fsLogger := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    maxSize, // megabytes
		MaxBackups: maxBackups,
		MaxAge:     maxAge, // days
		Compress:   enableCompression,
	}
	appLogger = log.With(appLogger, "component", "filesystem-logger")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)
	go func() {
		for {
			<-sig // block on signal
			if err := fsLogger.Rotate(); err != nil {
				appLogger.Log("err", err)
			}
		}
	}()
	return &filesystemLogWriter{fsLogger}, nil
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
			return ctxerr.Wrap(ctx, err, "writing log")
		}
	}
	if flusher, ok := l.writer.(flusher); ok {
		if err := flusher.Flush(); err != nil {
			return ctxerr.Wrap(ctx, err, "flushing log")
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

func newRawLogWriter(file *os.File) *rawLogWriter {
	buff := bufio.NewWriter(file)
	return &rawLogWriter{file: file, buff: buff}
}

// Write bytes to file
func (l *rawLogWriter) Write(b []byte) (int, error) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if l.buff == nil || l.file == nil {
		return 0, errors.New("filesystemLogWriter: can't write to closed file")
	}
	if _, statErr := os.Stat(l.file.Name()); errors.Is(statErr, os.ErrNotExist) {
		f, err := secure.OpenFile(l.file.Name(), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
		if err != nil {
			return 0, fmt.Errorf("create file for filesystemLogWriter %s: %w", l.file.Name(), err)
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

func openFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
}
