// Package logwriter provides logging utilities for writing osquery results and status logs
package logwriter

import (
	"bufio"
	"errors"
	"io"
	"os"
	"sync"
)

type logWriter struct {
	file *os.File
	buff *bufio.Writer
	mtx  sync.Mutex
}

// New creates a logwriter, path refers to file that will recieve log content
func New(path string) (io.WriteCloser, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	buff := bufio.NewWriter(file)
	return &logWriter{file: file, buff: buff}, nil
}

// Write bytes to file
func (l *logWriter) Write(b []byte) (int, error) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if l.buff == nil {
		return 0, errors.New("can't write to a closed file")
	}
	return l.buff.Write(b)
}

// Flush write all buffered bytes to log file
func (l *logWriter) Flush() error {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if l.buff == nil {
		return errors.New("can't write to a closed file")
	}
	return l.buff.Flush()
}

// Close log file
func (l *logWriter) Close() error {
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
