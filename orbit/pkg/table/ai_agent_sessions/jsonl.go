package ai_agent_sessions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"

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
