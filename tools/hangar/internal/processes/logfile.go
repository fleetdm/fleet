package processes

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// logFileMaxBytes is the on-disk rotation threshold. fleet serve under debug
// logging can produce tens of MB/hour; without rotation the file grows
// unbounded. One previous generation is kept as <channel>.log.1.
const logFileMaxBytes int64 = 16 * 1024 * 1024

// logBufSize matches Rust's BufWriter default (~8KiB) — a win for chatty
// stdout while keeping per-line syscall churn down.
const logBufSize = 8192

func logFilePath(logDir, channel string) string {
	return filepath.Join(logDir, channel+".log")
}

// channelWriter is a cached buffered writer for one channel's log file, with
// in-process size tracking for rotation.
type channelWriter struct {
	w        *bufio.Writer
	file     *os.File
	bytes    int64
	path     string
	maxBytes int64
}

func openChannelWriter(path string, maxBytes int64) (*channelWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	var size int64
	if fi, err := f.Stat(); err == nil {
		size = fi.Size()
	}
	return &channelWriter{
		w:        bufio.NewWriterSize(f, logBufSize),
		file:     f,
		bytes:    size,
		path:     path,
		maxBytes: maxBytes,
	}, nil
}

// rotateIfNeeded rotates <path> to <path>.1 once it crosses maxBytes.
func (cw *channelWriter) rotateIfNeeded() {
	if cw.bytes < cw.maxBytes {
		return
	}
	cw.w.Flush()
	cw.file.Close()
	_ = os.Rename(cw.path, cw.path+".1")
	if nw, err := openChannelWriter(cw.path, cw.maxBytes); err == nil {
		*cw = *nw
	}
}

// write appends one tab-delimited record (ts, stream, message). The message
// is secret-scrubbed and has embedded tabs replaced so the format stays
// parseable. stderr is flushed immediately so crash tails are durable.
func (cw *channelWriter) write(tsMS uint64, stream, message string) {
	cw.rotateIfNeeded()
	msg := strings.ReplaceAll(scrubSecrets(message), "\t", "    ")
	line := fmt.Sprintf("%d\t%s\t%s\n", tsMS, stream, msg)
	if n, err := cw.w.WriteString(line); err == nil {
		cw.bytes += int64(n)
	}
	if stream == "stderr" {
		cw.w.Flush()
	}
}

func (cw *channelWriter) flush() {
	if cw.w != nil {
		cw.w.Flush()
	}
}

func (cw *channelWriter) close() {
	cw.flush()
	if cw.file != nil {
		cw.file.Close()
	}
}
