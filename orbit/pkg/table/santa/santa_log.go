//go:build darwin

// Package santa implements the tables for getting Santa data
// (logs/status) on macOS.
//
// Santa is an open source macOS endpoint security system with
// binary whitelisting and blacklisting capabilities.
// Based on https://github.com/allenhouchins/fleet-extensions/tree/main/santa
package santa

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

const (
	kLogEntryPreface = "santad: "
	defaultLogPath   = "/var/db/santa/santa.log"

	// maxLineBytes is how much of a single log line is retained. Santa writes the
	// process arguments last, after every field these tables read, so the rest of
	// an over-long line is discarded instead of failing the read.
	maxLineBytes = 64 * 1024
)

var (
	maxEntries = 10_000
	logPath    = defaultLogPath
)

type santaDecisionType int

const (
	decisionAllowed santaDecisionType = iota
	decisionDenied
)

type logEntry struct {
	Timestamp   string
	Application string
	Reason      string
	SHA256      string
}

// Field names and markers, as byte slices to keep line parsing allocation-free.
var (
	logEntryPreface = []byte(kLogEntryPreface)
	allowMarker     = []byte("decision=ALLOW")
	denyMarker      = []byte("decision=DENY")
	keyPath         = []byte("path")
	keyReason       = []byte("reason")
	keySHA256       = []byte("sha256")
	fieldSep        = []byte("|")
	keyValueSep     = []byte("=")
)

func LogColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("timestamp"),
		table.TextColumn("application"),
		table.TextColumn("reason"),
		table.TextColumn("sha256"),
	}
}

func GenerateAllowed(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	return generate(ctx, decisionAllowed)
}

func GenerateDenied(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	return generate(ctx, decisionDenied)
}

func generate(ctx context.Context, dec santaDecisionType) ([]map[string]string, error) {
	entries, err := scrapeSantaLog(ctx, dec)
	if err != nil {
		// Report the failure but return whatever was read: returning an error here
		// would fail the whole query on every affected host, and a synthetic error
		// row would be indistinguishable from a real Santa event. Operators can
		// check `santa_status` (file_logging, log_type) for a misconfigured Santa.
		log.Error().Err(err).Int("entries", len(entries)).Msg("scraping santa log")
	}

	results := make([]map[string]string, 0, len(entries))
	for _, entry := range entries {
		results = append(results, map[string]string{
			"timestamp":   entry.Timestamp,
			"application": entry.Application,
			"reason":      entry.Reason,
			"sha256":      entry.SHA256,
		})
	}
	return results, nil
}

// parseLogEntry extracts the columns these tables expose from one log line. It
// reports false for a line with no timestamp, which is not a Santa event. Only
// the retained fields are copied out of line, so a line is never materialized as
// a string; in monitor mode this runs over every exec Santa logs.
func parseLogEntry(line []byte) (logEntry, bool) {
	timestamp, ok := extractTimestamp(line)
	if !ok {
		return logEntry{}, false
	}
	entry := logEntry{Timestamp: string(timestamp)}

	_, fields, found := bytes.Cut(line, logEntryPreface)
	if !found {
		return entry, true
	}

	for seg := range bytes.SplitSeq(fields, fieldSep) {
		k, v, found := bytes.Cut(seg, keyValueSep)
		if !found {
			continue
		}
		k = bytes.TrimSpace(k)
		v = bytes.Trim(bytes.TrimSpace(v), `"'`)
		if len(k) == 0 || len(v) == 0 {
			continue
		}

		switch {
		case bytes.EqualFold(k, keyPath):
			entry.Application = string(v)
		case bytes.EqualFold(k, keyReason):
			entry.Reason = string(v)
		case bytes.EqualFold(k, keySHA256):
			entry.SHA256 = string(v)
		}
	}
	return entry, true
}

// extractTimestamp returns the contents of the first non-empty bracketed group in
// line, matching the `\[([^\]]+)\]` pattern it replaces.
func extractTimestamp(line []byte) ([]byte, bool) {
	for len(line) > 0 {
		open := bytes.IndexByte(line, '[')
		if open == -1 {
			return nil, false
		}
		line = line[open+1:]
		if length := bytes.IndexByte(line, ']'); length > 0 {
			return line[:length], true
		}
	}
	return nil, false
}

// lineReader reads newline-terminated lines, retaining at most max bytes of
// each and discarding the remainder. Unlike bufio.Scanner it cannot fail on an
// over-long line: one Santa exec event with long arguments would otherwise abort
// the scrape and leave the table empty until that line rotated out of the log.
type lineReader struct {
	r   *bufio.Reader
	max int
	// long holds a line assembled across several reads, reused between lines.
	long []byte
}

func newLineReader(r io.Reader, maxSize int) *lineReader {
	// Sizing the read buffer to the retention limit keeps every line that is
	// retained in full on the single-read path below.
	return &lineReader{r: bufio.NewReaderSize(r, maxSize), max: maxSize}
}

// readLine returns the next line with its trailing newline removed. The returned
// slice is only valid until the next call. terminated reports whether the line
// ended with a newline; a final line without one was caught mid-write.
func (lr *lineReader) readLine() (line []byte, terminated bool, err error) {
	chunk, readErr := lr.r.ReadSlice('\n')
	switch {
	case readErr == nil:
		return trimEOL(chunk), true, nil
	case errors.Is(readErr, io.EOF) && len(chunk) == 0:
		return nil, false, io.EOF
	case !errors.Is(readErr, bufio.ErrBufferFull) && !errors.Is(readErr, io.EOF):
		return nil, false, readErr
	}

	// Either the line is longer than the read buffer, or the file ended without a
	// newline. Retain up to max bytes and discard the remainder of the line.
	lr.long = appendChunk(lr.long[:0], chunk, lr.max)
	for errors.Is(readErr, bufio.ErrBufferFull) {
		chunk, readErr = lr.r.ReadSlice('\n')
		lr.long = appendChunk(lr.long, chunk, lr.max)
	}

	switch {
	case readErr == nil:
		return trimEOL(lr.long), true, nil
	case errors.Is(readErr, io.EOF):
		return lr.long, false, nil
	default:
		return nil, false, readErr
	}
}

// appendChunk appends chunk to dst, up to a total of max bytes.
func appendChunk(dst, chunk []byte, maxSize int) []byte {
	room := maxSize - len(dst)
	if room <= 0 || len(chunk) == 0 {
		return dst
	}
	return append(dst, chunk[:min(room, len(chunk))]...)
}

// trimEOL drops the line terminator, matching how bufio.ScanLines splits lines.
func trimEOL(line []byte) []byte {
	return bytes.TrimSuffix(bytes.TrimSuffix(line, []byte("\n")), []byte("\r"))
}

func scrapeStream(ctx context.Context, r io.Reader, decision santaDecisionType, rb *ringBuffer) error {
	lr := newLineReader(r, maxLineBytes)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, terminated, err := lr.readLine()
		switch {
		case errors.Is(err, io.EOF):
			return nil
		case err != nil:
			return err
		case !terminated:
			// santad newline-terminates every line, so an unterminated final line is
			// a write in progress rather than an event.
			return nil
		}

		// Filter by decision type early to keep it fast.
		switch decision {
		case decisionAllowed:
			if !bytes.Contains(line, allowMarker) {
				continue
			}
		case decisionDenied:
			if !bytes.Contains(line, denyMarker) {
				continue
			}
		}

		entry, ok := parseLogEntry(line)
		if !ok {
			continue
		}
		rb.Add(entry)
	}
}

// errLogMissing reports a log file that was not there to open, as opposed to one
// that failed while being read. Only the former is benign: Santa may not be
// installed, or a rotation may have just renamed the file, so it is reported as no
// events rather than as a failed read.
var errLogMissing = errors.New("santa log file does not exist")

// openLog opens a log file, distinguishing a file that is not there from one that
// cannot be opened.
func openLog(path string) (*os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", errLogMissing, path)
		}
		return nil, fmt.Errorf("failed to open Santa log file %s: %w", path, err)
	}
	return file, nil
}

func scrapePlainSantaLog(ctx context.Context, path string, decision santaDecisionType, rb *ringBuffer) error {
	file, err := openLog(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return scrapeStream(ctx, file, decision, rb)
}

func scrapeCompressedSantaLog(ctx context.Context, path string, decision santaDecisionType, rb *ringBuffer) error {
	file, err := openLog(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader for %s: %w", path, err)
	}
	defer gzReader.Close()

	return scrapeStream(ctx, gzReader, decision, rb)
}

// archive is a rotated log file. newsyslog compresses a rotated log after
// renaming it, so for a brief window the compressed file is incomplete while the
// uncompressed one is still on disk; keep the latter as a fallback. Hosts whose
// newsyslog configuration does not compress at all only ever have the plain file.
type archive struct {
	path       string
	compressed bool
	fallback   string
}

// archives returns the rotated logs for base, oldest first, stopping at the
// first rotation index with no file at all.
func archives(base string) []archive {
	var found []archive
	for i := 0; ; i++ {
		gzPath := fmt.Sprintf("%s.%d.gz", base, i)
		plainPath := fmt.Sprintf("%s.%d", base, i)

		switch gzExists, plainExists := fileExists(gzPath), fileExists(plainPath); {
		case gzExists:
			a := archive{path: gzPath, compressed: true}
			if plainExists {
				a.fallback = plainPath
			}
			found = append(found, a)
		case plainExists:
			found = append(found, archive{path: plainPath})
		default:
			slices.Reverse(found)
			return found
		}
	}
}

// statFile is a variable so tests can simulate a log that is discovered and then
// rotated away before it is read.
var statFile = os.Stat

func fileExists(path string) bool {
	_, err := statFile(path)
	return err == nil
}

// scrapeArchive reads one rotated log into rb.
func scrapeArchive(ctx context.Context, a archive, decision santaDecisionType, rb *ringBuffer) error {
	if a.fallback == "" {
		return scrapeArchiveFile(ctx, a.path, a.compressed, decision, rb)
	}

	// The compressed archive and the file it is being compressed from both exist,
	// so newsyslog is mid-rotation and the .gz may be incomplete. Collect the
	// compressed read separately, so that falling back to the uncompressed file
	// cannot double-count the entries the failed attempt already collected.
	scratch := newRingBuffer(rb.cap)
	err := scrapeArchiveFile(ctx, a.path, a.compressed, decision, scratch)
	if err != nil && ctx.Err() == nil {
		scratch.Reset()
		if fallbackErr := scrapeArchiveFile(ctx, a.fallback, false, decision, scratch); fallbackErr == nil {
			err = nil
		}
	}

	for _, entry := range scratch.SliceChrono() {
		rb.Add(entry)
	}
	return err
}

func scrapeArchiveFile(ctx context.Context, path string, compressed bool, decision santaDecisionType, rb *ringBuffer) error {
	if compressed {
		return scrapeCompressedSantaLog(ctx, path, decision, rb)
	}
	return scrapePlainSantaLog(ctx, path, decision, rb)
}

func scrapeSantaLog(ctx context.Context, decision santaDecisionType) ([]logEntry, error) {
	return scrapeSantaLogFromBase(ctx, decision, logPath)
}

// scrapeSantaLogFromBase returns up to maxEntries of the most recent events,
// oldest first.
//
// Files are read newest first and only until the cap is met, so on a busy host in
// monitor mode the active log alone satisfies the query and the archives are never
// opened, let alone decompressed. Each file is read forward into a buffer sized to
// what is still needed, which yields that file's last N events; the per-file
// results are then concatenated oldest file first.
//
// Every read is best effort: a file that cannot be read is reported without
// discarding the events collected from the others.
func scrapeSantaLogFromBase(ctx context.Context, decision santaDecisionType, path string) ([]logEntry, error) {
	var errs []error

	// Archives are discovered before the active log is read, and that order matters:
	// were discovery to run afterwards, a rotation in between would move the events
	// just read from santa.log into a santa.log.0 that discovery then finds, returning
	// them twice. The stat calls are cheap enough to pay even when the active log ends
	// up satisfying the cap on its own.
	//
	// A rotation landing between reading the active log and reading the archives can
	// still return a few events twice, because santa.log.0 then holds what santa.log
	// held moments earlier. That window is a few milliseconds against a rotation
	// interval of hours, and repeating an event is a better failure for an audit table
	// than silently dropping one.
	rotated := archives(path)

	// The active log holds the newest events, so it is read first.
	current := newRingBuffer(maxEntries)
	switch err := scrapePlainSantaLog(ctx, path, decision, current); {
	case err == nil:
	case errors.Is(err, errLogMissing):
		// Not an error: Santa may not be installed, or may not be configured to log to
		// a file (see `santa_status.log_type`). It may also have been renamed by
		// newsyslog a moment ago, in which case its events are in a santa.log.0 that
		// discovery ran too early to see — and every archive on the list has shifted
		// up one index — so discover again. Nothing has been read yet, so replacing
		// the list cannot double-count.
		log.Debug().Str("path", path).Msg("santa log not found")
		rotated = archives(path)
	default:
		errs = append(errs, err)
	}

	// Per-file results, newest file first.
	collected := []*ringBuffer{current}
	remaining := maxEntries - current.Len()

	for _, a := range slices.Backward(rotated) {
		if remaining <= 0 || ctx.Err() != nil {
			break
		}

		buf := newRingBuffer(remaining)
		switch err := scrapeArchive(ctx, a, decision, buf); {
		case err == nil:
		case errors.Is(err, errLogMissing):
			// Rotated away between being discovered and being read: its events moved to
			// the next index. At worst one generation of archived events is missed,
			// which the entry cap hides on all but the quietest hosts.
			log.Debug().Err(err).Msg("santa log archive rotated away while reading")
		default:
			errs = append(errs, err)
		}

		collected = append(collected, buf)
		remaining -= buf.Len()
	}

	total := 0
	for _, buf := range collected {
		total += buf.Len()
	}
	entries := make([]logEntry, 0, total)
	for _, buf := range slices.Backward(collected) {
		entries = buf.AppendTo(entries)
	}
	return entries, errors.Join(errs...)
}
