package ai_agent_sessions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/internal/homes"
)

const (
	// maxRecordBytes bounds one JSONL record. A larger line is a pasted file
	// or a huge tool result; it is skipped rather than buffered.
	maxRecordBytes = 8 << 20
	// maxFirstRecordBytes bounds a file's first record when discovery reads
	// it to learn which session the file belongs to.
	maxFirstRecordBytes = 1 << 20
	// maxTranscriptBytes bounds how much of one transcript is read. The
	// largest real sessions are tens of MB; a sparse multi-GB file is read up
	// to here.
	maxTranscriptBytes = 512 << 20
	// maxDocumentBytes bounds a JSON document read whole.
	maxDocumentBytes = 64 << 20
)

// maxReadBytesPerCall bounds what one query reads across every file. A local
// user can plant any number of sparse files; sessions parsed before the
// budget runs out are cached, so a large legitimate history still completes
// over a few scans.
var maxReadBytesPerCall int64 = 4 << 30

// scope is a home as one call sees it: only files its owner owns are read,
// and only so much of them in total.
type scope struct {
	home  homes.Home
	reads *readBudget
}

// readBudget is what a call may still read. It is used from one goroutine.
type readBudget struct {
	left      int64
	exhausted bool
}

var errReadBudget = errors.New("read budget exhausted")

func newReadBudget() *readBudget { return &readBudget{left: maxReadBytesPerCall} }

// open opens a regular file for reading, refusing links and special files
// (fsutil.OpenRegular) and files the home's user doesn't own: a link planted
// along the path can lead to another account's files, and a hard link to one
// passes every path check.
func (sc *scope) open(path string) (*os.File, error) {
	if sc.reads.exhausted {
		return nil, errReadBudget
	}
	f, err := fsutil.OpenRegular(path)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil || !sc.home.Owns(f, fi) {
		_ = f.Close()
		return nil, os.ErrPermission
	}
	return f, nil
}

// reader reads at most limit bytes of f, charged to the call's budget.
func (sc *scope) reader(f *os.File, limit int64) io.Reader {
	return &budgetReader{r: io.LimitReader(f, limit), b: sc.reads}
}

type budgetReader struct {
	r io.Reader
	b *readBudget
}

func (r *budgetReader) Read(p []byte) (int, error) {
	if r.b.left <= 0 {
		r.b.exhausted = true
		return 0, errReadBudget
	}
	if int64(len(p)) > r.b.left {
		p = p[:r.b.left]
	}
	n, err := r.r.Read(p)
	r.b.left -= int64(n)
	return n, err
}

// readers pools the scan buffers: one is in use per file being read, and a
// large one lets most records be handed out without copying.
var readers = sync.Pool{New: func() any { return bufio.NewReaderSize(nil, 512<<10) }}

// scanJSONL hands each non-empty line of a JSON Lines file to fn until fn
// returns false. The line is only valid during the call.
func scanJSONL(sc *scope, path string, fn func(line []byte) bool) error {
	f, err := sc.open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	r := readers.Get().(*bufio.Reader)
	r.Reset(sc.reader(f, maxTranscriptBytes))
	defer func() {
		r.Reset(nil)
		readers.Put(r)
	}()
	for {
		line, readErr := readRecord(r)
		if len(line) > 0 && !fn(line) {
			return nil
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return readErr
		}
	}
}

// maxMergedFiles bounds how many of a session's files are read side by side
// (each holds a scan buffer open); any further files are read after them.
const maxMergedFiles = 32

// scanFiles hands the records of a session's files to fn, with the index of
// the file each came from, in the order they were written: files are merged
// by the timestamp stamp reads from each record, a record without one
// following the previous record of its file, and the earlier file winning a
// tie. Numbering the session's content in that order keeps every position
// stable as the files grow, where reading the main transcript and then each
// sub-agent's would move the latter every time the former grew. A nil stamp
// reads the files one after another.
func scanFiles(sc *scope, files []string, stamp func(line []byte) time.Time, fn func(file int, line []byte) bool) {
	merged := files
	if stamp == nil || len(files) < 2 {
		merged = nil
	} else if len(files) > maxMergedFiles {
		merged = files[:maxMergedFiles]
	}

	var streams []*recordStream
	defer func() {
		for _, st := range streams {
			st.close()
		}
	}()
	for i, path := range merged {
		st, err := openRecordStream(sc, path, i)
		if err != nil {
			continue
		}
		if st.advance(stamp) {
			streams = append(streams, st)
		} else {
			st.close()
		}
	}
	for len(streams) > 0 {
		best := 0
		for i, st := range streams[1:] {
			if st.ts.Before(streams[best].ts) {
				best = i + 1
			}
		}
		st := streams[best]
		if !fn(st.file, st.line) {
			return
		}
		if !st.advance(stamp) {
			st.close()
			streams = append(streams[:best], streams[best+1:]...)
		}
	}

	for i := len(merged); i < len(files); i++ {
		more := true
		_ = scanJSONL(sc, files[i], func(line []byte) bool {
			more = fn(i, line)
			return more
		})
		if !more {
			return
		}
	}
}

// recordStream is one file being merged: its current record and timestamp.
type recordStream struct {
	file int
	f    *os.File
	r    *bufio.Reader
	line []byte
	ts   time.Time
	err  error
}

func openRecordStream(sc *scope, path string, file int) (*recordStream, error) {
	f, err := sc.open(path)
	if err != nil {
		return nil, err
	}
	r := readers.Get().(*bufio.Reader)
	r.Reset(sc.reader(f, maxTranscriptBytes))
	return &recordStream{file: file, f: f, r: r}, nil
}

// advance reads the next non-empty record, reporting false at the end.
func (st *recordStream) advance(stamp func(line []byte) time.Time) bool {
	for st.err == nil {
		st.line, st.err = readRecord(st.r)
		if len(st.line) == 0 {
			continue
		}
		if ts := stamp(st.line); !ts.IsZero() {
			st.ts = ts
		}
		return true
	}
	return false
}

func (st *recordStream) close() {
	st.r.Reset(nil)
	readers.Put(st.r)
	_ = st.f.Close()
}

// recordStamp reads the timestamp Claude Code and Codex put on every record.
func recordStamp(line []byte) time.Time {
	v, ok := topLevelString(line, "timestamp")
	if !ok {
		return time.Time{}
	}
	return parseTime(v)
}

// topLevelString returns the string value of key in the JSON object line,
// stepping over the other members rather than decoding them: it runs on
// every record of a merged session, and a full decode costs as much as the
// parse itself. It reports false when the key is absent, isn't a string, or
// the object is malformed up to there.
func topLevelString(line []byte, key string) (string, bool) {
	i := skipSpace(line, 0)
	if i >= len(line) || line[i] != '{' {
		return "", false
	}
	i = skipSpace(line, i+1)
	for i < len(line) && line[i] == '"' {
		start := i + 1
		end, ok := skipString(line, i)
		if !ok {
			return "", false
		}
		i = skipSpace(line, end)
		if i >= len(line) || line[i] != ':' {
			return "", false
		}
		i = skipSpace(line, i+1)
		if i >= len(line) {
			return "", false
		}
		if string(line[start:end-1]) == key {
			if line[i] != '"' {
				return "", false
			}
			end, ok := skipString(line, i)
			if !ok {
				return "", false
			}
			raw := line[i:end]
			if !bytes.ContainsRune(raw, '\\') {
				return string(raw[1 : len(raw)-1]), true
			}
			var s string
			return s, json.Unmarshal(raw, &s) == nil
		}
		if i, ok = skipValue(line, i); !ok {
			return "", false
		}
		i = skipSpace(line, i)
		if i < len(line) && line[i] == ',' {
			i = skipSpace(line, i+1)
		}
	}
	return "", false
}

func skipSpace(b []byte, i int) int {
	for i < len(b) && (b[i] == ' ' || b[i] == '\t' || b[i] == '\n' || b[i] == '\r') {
		i++
	}
	return i
}

// skipString returns the index after the string starting at b[i].
func skipString(b []byte, i int) (int, bool) {
	for i++; i < len(b); i++ {
		switch b[i] {
		case '\\':
			i++
		case '"':
			return i + 1, true
		}
	}
	return 0, false
}

// skipValue returns the index after the value starting at b[i].
func skipValue(b []byte, i int) (int, bool) {
	switch b[i] {
	case '"':
		return skipString(b, i)
	case '{', '[':
		depth := 0
		for i < len(b) {
			switch b[i] {
			case '"':
				end, ok := skipString(b, i)
				if !ok {
					return 0, false
				}
				i = end
				continue
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					return i + 1, true
				}
			}
			i++
		}
		return 0, false
	}
	for i < len(b) && !bytes.ContainsAny(b[i:i+1], ",}] \t\n\r") {
		i++
	}
	return i, true
}

// readRecord reads one newline-terminated record. A record that fits the
// reader's buffer is returned as a slice of that buffer, valid until the next
// read; records over maxRecordBytes are drained and returned empty.
func readRecord(r *bufio.Reader) ([]byte, error) {
	chunk, err := r.ReadSlice('\n')
	if !errors.Is(err, bufio.ErrBufferFull) {
		return bytes.TrimSpace(chunk), err
	}
	var (
		buf      []byte
		oversize bool
	)
	for {
		if !oversize {
			if len(buf)+len(chunk) > maxRecordBytes {
				oversize = true
				buf = nil
			} else {
				buf = append(buf, chunk...)
			}
		}
		if !errors.Is(err, bufio.ErrBufferFull) {
			return bytes.TrimSpace(buf), err
		}
		chunk, err = r.ReadSlice('\n')
	}
}

// firstRecord decodes the first line of a JSON Lines file into v, reading at
// most maxFirstRecordBytes.
func firstRecord(sc *scope, path string, v any) bool {
	f, err := sc.open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	r := readers.Get().(*bufio.Reader)
	r.Reset(sc.reader(f, maxFirstRecordBytes))
	defer func() {
		r.Reset(nil)
		readers.Put(r)
	}()
	line, _ := readRecord(r)
	return len(line) > 0 && json.Unmarshal(line, v) == nil
}

// readJSON decodes a whole, bounded JSON file into v.
func readJSON(sc *scope, path string, v any) bool {
	b, err := sc.readAll(path, maxDocumentBytes)
	if err != nil {
		return false
	}
	return json.Unmarshal(b, v) == nil
}

// readText returns up to limit bytes of a small text file, trimmed.
func readText(sc *scope, path string, limit int64) string {
	b, err := sc.readAll(path, limit)
	if err != nil {
		return ""
	}
	return string(bytes.TrimSpace(b))
}

func (sc *scope) readAll(path string, limit int64) ([]byte, error) {
	f, err := sc.open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(sc.reader(f, limit))
}

// presence records whether a JSON key was present with a non-null value,
// without keeping the value. It is used for secrets whose existence is the
// only thing we need.
type presence bool

func (p *presence) UnmarshalJSON(b []byte) error {
	*p = presence(string(bytes.TrimSpace(b)) != "null")
	return nil
}
