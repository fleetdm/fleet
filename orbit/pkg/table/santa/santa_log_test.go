//go:build darwin

package santa

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

func TestParseLogEntry(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		want   logEntry
		wantOK bool
	}{
		{
			name: "happy path with timestamp and kv pairs",
			line: `[2025-09-18T10:15:30.123Z] santad: decision=ALLOW | path=/Applications/Foo.app | reason=cdhash | sha256=abc123`,
			want: logEntry{
				Timestamp:   "2025-09-18T10:15:30.123Z",
				Application: "/Applications/Foo.app",
				Reason:      "cdhash",
				SHA256:      "abc123",
			},
			wantOK: true,
		},
		{
			name:   "no santad preface yields only the timestamp",
			line:   `[2025-09-18 10:15:30] something else: decision=DENY | path=/bin/bash`,
			want:   logEntry{Timestamp: "2025-09-18 10:15:30"},
			wantOK: true,
		},
		{
			name: "no timestamp is not an event",
			line: `santad: decision=DENY | path=/usr/local/bin/tool | reason=rule | sha256=def456`,
		},
		{
			name: "trims spaces around keys and values",
			line: `[2025-09-18] santad:   decision = ALLOW   |   path = /a/b/c  | reason =  ok  `,
			want: logEntry{
				Timestamp:   "2025-09-18",
				Application: "/a/b/c",
				Reason:      "ok",
			},
			wantOK: true,
		},
		{
			name: "ignores empty segments and missing equals",
			line: `[ts] santad: decision=DENY | | path=/p | just-a-flag | sha256=zzz`,
			want: logEntry{
				Timestamp:   "ts",
				Application: "/p",
				SHA256:      "zzz",
			},
			wantOK: true,
		},
		{
			name: "value containing equals keeps everything after the first equals",
			line: `[ts] santad: note=a=b=c | path=/eq | sha256=x`,
			want: logEntry{
				Timestamp:   "ts",
				Application: "/eq",
				SHA256:      "x",
			},
			wantOK: true,
		},
		{
			name: "duplicate keys last one wins",
			line: `[ts] santad: path=/first | path=/second | reason=one | reason=two`,
			want: logEntry{
				Timestamp:   "ts",
				Application: "/second",
				Reason:      "two",
			},
			wantOK: true,
		},
		{
			name: "quoted values are unquoted",
			line: `[ts] santad: path="/Applications/App With Spaces.app" | reason='quoted'`,
			want: logEntry{
				Timestamp:   "ts",
				Application: "/Applications/App With Spaces.app",
				Reason:      "quoted",
			},
			wantOK: true,
		},
		{
			name: "keys are matched case-insensitively",
			line: `[ts] santad: PATH=/upper | Reason=ok | SHA256=abc`,
			want: logEntry{
				Timestamp:   "ts",
				Application: "/upper",
				Reason:      "ok",
				SHA256:      "abc",
			},
			wantOK: true,
		},
		{
			name: "unrelated line is not an event",
			line: `completely unrelated line`,
		},
		{
			name: "empty bracket group is not a timestamp",
			line: `[] santad: decision=ALLOW | path=/a`,
		},
		{
			name: "falls through an empty bracket group to the next one",
			line: `[] [ts] santad: decision=ALLOW | path=/a`,
			want: logEntry{
				Timestamp:   "ts",
				Application: "/a",
			},
			wantOK: true,
		},
		{
			name: "unclosed bracket group is not a timestamp",
			line: `[2025-09-18 santad: decision=ALLOW | path=/a`,
		},
		{
			name: "bracket group keeps a nested opening bracket",
			line: `[a[b] santad: decision=ALLOW | path=/a`,
			want: logEntry{
				Timestamp:   "a[b",
				Application: "/a",
			},
			wantOK: true,
		},
		{
			name: "handles trailing separator",
			line: `[ts] santad: decision=ALLOW | path=/a/b/c |`,
			want: logEntry{
				Timestamp:   "ts",
				Application: "/a/b/c",
			},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseLogEntry([]byte(tt.line))
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseLogEntry_TruncatedLongLine(t *testing.T) {
	// A line long enough to exercise the retention limit still yields its columns,
	// because Santa writes the process arguments last.
	line := mkLineWithArgs("decision=ALLOW", "2025-09-18", "/A", "ok", "abc", strings.Repeat("a", 300_000))

	got, ok := parseLogEntry([]byte(line[:maxLineBytes]))
	require.True(t, ok)
	require.Equal(t, logEntry{Timestamp: "2025-09-18", Application: "/A", Reason: "ok", SHA256: "abc"}, got)
}

func TestScrapeSantaLogFromBase_EndToEnd(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	// current (plain) log with ALLOW and DENY
	current := strings.Builder{}
	current.WriteString(mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/Applications/A.app", "ok", "aaa"))
	current.WriteString(mkLine("decision=DENY", "2025-09-18 12:00:01.000", "/Applications/B.app", "rule", "bbb"))
	writeFile(t, base, current.String())

	// archive 0 (gz): a DENY (older)
	writeGz(t, base+".0.gz", mkLine("decision=DENY", "2025-09-18 11:59:59.000", "/Blocked/X", "blacklist", "xxx"))

	// archive 1 (gz): an ALLOW (older)
	writeGz(t, base+".1.gz", mkLine("decision=ALLOW", "2025-09-18 11:59:58.000", "/OK/C", "scope", "ccc"))

	ctx := t.Context()

	denied, err := scrapeSantaLogFromBase(ctx, decisionDenied, base)
	require.NoError(t, err)
	// With current scanned first, chronological (insertion) order is:
	// current DENY, then archive 0 DENY.
	require.Len(t, denied, 2)
	require.Equal(t, "/Blocked/X", denied[0].Application)
	require.Equal(t, "/Applications/B.app", denied[1].Application)

	allowed, err := scrapeSantaLogFromBase(ctx, decisionAllowed, base)
	require.NoError(t, err)
	// current ALLOW, then archive 1 ALLOW.
	require.Len(t, allowed, 2)
	require.Equal(t, "/OK/C", allowed[0].Application)
	require.Equal(t, "/Applications/A.app", allowed[1].Application)
}

// TestScrapeSantaLogFromBase_IgnoresGapsAfterFirstMiss verifies that archive
// iteration stops cleanly at the first missing archive file.
// In this setup only the current log exists (no ".0.gz"), so the function
// should return entries from the current log only and not attempt to read
// later archives (".1.gz", ".2.gz", etc.).
func TestScrapeSantaLogFromBase_IgnoresGapsAfterFirstMiss(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	// only current exists; no .0.gz
	writeFile(t, base, mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/A", "ok", "aaa"))

	got, err := scrapeSantaLogFromBase(context.Background(), decisionAllowed, base)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "/A", got[0].Application)
}

// TestScrapeSantaLogFromBase_SurvivesOverlongLine verifies that a log line
// longer than bufio.Scanner's default token limit does not discard the entries
// scraped from the rest of the file. In monitor mode Santa logs an ALLOW line
// for nearly every exec, arguments included, so a single long command line
// would otherwise empty the whole table until it rotated out.
func TestScrapeSantaLogFromBase_SurvivesOverlongLine(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	var sb strings.Builder
	sb.WriteString(mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/A", "ok", "aaa"))
	sb.WriteString(mkLineWithArgs("decision=ALLOW", "2025-09-18 12:00:01.000", "/B", "ok", "bbb",
		strings.Repeat("x", 200_000)))
	sb.WriteString(mkLine("decision=ALLOW", "2025-09-18 12:00:02.000", "/C", "ok", "ccc"))
	writeFile(t, base, sb.String())

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err)
	require.Equal(t, []string{"/A", "/B", "/C"}, apps(got))
}

// TestScrapeSantaLogFromBase_TruncatesLongLineKeepingColumns verifies that
// truncating an over-long line preserves every column these tables expose.
// Santa emits the unbounded args field last, after path, reason and sha256.
func TestScrapeSantaLogFromBase_TruncatesLongLineKeepingColumns(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	writeFile(t, base, mkLineWithArgs("decision=ALLOW", "2025-09-18 12:00:01.000", "/Long", "cdhash", "bbb",
		strings.Repeat("x", 200_000)))

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "2025-09-18 12:00:01.000", got[0].Timestamp)
	require.Equal(t, "/Long", got[0].Application)
	require.Equal(t, "cdhash", got[0].Reason)
	require.Equal(t, "bbb", got[0].SHA256)
}

// TestScrapeSantaLogFromBase_UnreadableArchiveKeepsOtherEntries verifies that a
// file that cannot be read is reported but does not discard entries scraped
// from the other files.
func TestScrapeSantaLogFromBase_UnreadableArchiveKeepsOtherEntries(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	writeFile(t, base, mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/CUR", "ok", "aaa"))
	// Not gzip data: an archive still being written by newsyslog looks like this.
	writeFile(t, base+".0.gz", "definitely not gzip")

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.Error(t, err, "the unreadable archive should be reported")
	require.Len(t, got, 1, "entries from the readable files should survive")
	require.Equal(t, "/CUR", got[0].Application)
}

// TestScrapeSantaLogFromBase_CorruptArchiveFallsBackToUncompressed covers the
// window in which newsyslog has rotated the log but not finished compressing
// it: the .gz is incomplete while the uncompressed sibling is still on disk.
func TestScrapeSantaLogFromBase_CorruptArchiveFallsBackToUncompressed(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	writeFile(t, base, mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/CUR", "ok", "aaa"))
	writeFile(t, base+".0.gz", "definitely not gzip")
	writeFile(t, base+".0", mkLine("decision=ALLOW", "2025-09-18 11:59:59.000", "/ARC0", "ok", "bbb"))

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err, "the uncompressed sibling should satisfy the read")
	require.Equal(t, []string{"/ARC0", "/CUR"}, apps(got))
}

// TestScrapeSantaLogFromBase_ReadsUncompressedArchives verifies that rotated
// logs are read even when they have not been compressed at all, which depends
// on the host's newsyslog configuration.
func TestScrapeSantaLogFromBase_ReadsUncompressedArchives(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	writeFile(t, base, mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/CUR", "ok", "aaa"))
	writeFile(t, base+".0", mkLine("decision=ALLOW", "2025-09-18 11:59:59.000", "/ARC0", "ok", "bbb"))
	writeFile(t, base+".1", mkLine("decision=ALLOW", "2025-09-18 11:59:58.000", "/ARC1", "ok", "ccc"))

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err)
	require.Equal(t, []string{"/ARC1", "/ARC0", "/CUR"}, apps(got))
}

// TestScrapeSantaLogFromBase_MissingCurrentLogKeepsArchives verifies that a
// missing current log is treated as benign (Santa may not be installed, or the
// log may have just been rotated) and does not discard archived entries.
func TestScrapeSantaLogFromBase_MissingCurrentLogKeepsArchives(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	writeGz(t, base+".0.gz", mkLine("decision=ALLOW", "2025-09-18 11:59:59.000", "/ARC0", "ok", "bbb"))

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "/ARC0", got[0].Application)
}

// TestScrapeSantaLogFromBase_TruncatedArchiveKeepsDecodedEntries verifies that
// a gzip stream cut short still yields the entries decoded before the failure.
func TestScrapeSantaLogFromBase_TruncatedArchiveKeepsDecodedEntries(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	writeFile(t, base, mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/CUR", "ok", "aaa"))
	writeTruncatedGz(t, base+".0.gz",
		mkLine("decision=ALLOW", "2025-09-18 11:59:58.000", "/DECODED", "ok", "bbb"),
		mkLine("decision=ALLOW", "2025-09-18 11:59:59.000", "/LOST", "ok", "ccc"),
	)

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.Error(t, err, "the truncated archive should be reported")
	require.Equal(t, []string{"/DECODED", "/CUR"}, apps(got))
}

// TestScrapeSantaLogFromBase_SkipsUnterminatedFinalLine verifies that a line
// caught mid-write is not reported as an event. santad terminates every line
// with a newline, so an unterminated final line is always a partial write.
func TestScrapeSantaLogFromBase_SkipsUnterminatedFinalLine(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	writeFile(t, base,
		mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/A", "ok", "aaa")+
			`[2025-09-18 12:00:01.000] santad: decision=ALLOW | path="/PARTIAL" | rea`)

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "/A", got[0].Application)
}

// TestScrapeSantaLogFromBase_SurvivesOverlongLineInArchive exercises the
// over-long line path through a compressed archive, where the reader is fed in
// decompressed chunks rather than straight from a file.
func TestScrapeSantaLogFromBase_SurvivesOverlongLineInArchive(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	writeFile(t, base, mkLine("decision=ALLOW", "2025-09-18 12:00:02.000", "/CUR", "ok", "ccc"))
	writeGz(t, base+".0.gz",
		mkLine("decision=ALLOW", "2025-09-18 11:59:58.000", "/A", "ok", "aaa")+
			mkLineWithArgs("decision=ALLOW", "2025-09-18 11:59:59.000", "/B", "ok", "bbb",
				strings.Repeat("x", 200_000))+
			mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/C", "ok", "ccc"))

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err)
	require.Equal(t, []string{"/A", "/B", "/C", "/CUR"}, apps(got))
}

// TestScrapeSantaLogFromBase_LineAtRetentionLimit covers the boundary between the
// single-read path and the truncating one.
func TestScrapeSantaLogFromBase_LineAtRetentionLimit(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	for _, length := range []int{maxLineBytes - 1, maxLineBytes, maxLineBytes + 1} {
		line := mkLineWithArgs("decision=ALLOW", "2025-09-18 12:00:00.000", "/A", "ok", "aaa", "")
		// Pad the args field so the line is exactly length bytes, newline included.
		line = strings.TrimSuffix(line, "\n") + strings.Repeat("x", length-len(line)) + "\n"
		require.Len(t, line, length)
		writeFile(t, base, line)

		got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
		require.NoError(t, err, "line length %d", length)
		require.Len(t, got, 1, "line length %d", length)
		require.Equal(t, logEntry{
			Timestamp:   "2025-09-18 12:00:00.000",
			Application: "/A",
			Reason:      "ok",
			SHA256:      "aaa",
		}, got[0], "line length %d", length)
	}
}

// TestScrapeSantaLogFromBase_CanceledContext verifies that a canceled query is
// reported rather than looking like an empty log.
func TestScrapeSantaLogFromBase_CanceledContext(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")
	writeFile(t, base, mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/A", "ok", "aaa"))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := scrapeSantaLogFromBase(ctx, decisionAllowed, base)
	require.ErrorIs(t, err, context.Canceled)
}

// TestScrapeSantaLogFromBase_ArchiveRotatedAwayIsNotReported covers an archive
// that is renamed by newsyslog between being discovered and being read. Its
// events move to the next rotation index, so the disappearance is expected and
// must not be reported as a failure.
func TestScrapeSantaLogFromBase_ArchiveRotatedAwayIsNotReported(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	writeFile(t, base, mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/CUR", "ok", "aaa"))
	writeGz(t, base+".1.gz", mkLine("decision=ALLOW", "2025-09-18 11:59:58.000", "/ARC1", "ok", "ccc"))

	// santa.log.0.gz is reported by discovery but is not on disk by the time it is
	// opened.
	original := statFile
	t.Cleanup(func() { statFile = original })
	statFile = func(path string) (os.FileInfo, error) {
		if strings.HasSuffix(path, ".0.gz") {
			return original(base)
		}
		return original(path)
	}

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err)
	require.Equal(t, []string{"/ARC1", "/CUR"}, apps(got))
}

// TestScrapeSantaLogFromBase_RetriesRotatingCurrentLog covers the window in
// which newsyslog has renamed santa.log and santad has not recreated it yet.
func TestScrapeSantaLogFromBase_RetriesRotatingCurrentLog(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	// The rotated log is on disk; the current one does not exist yet.
	writeFile(t, base+".0", mkLine("decision=ALLOW", "2025-09-18 11:59:59.000", "/ARC0", "ok", "bbb"))

	var waits int
	stubRetryWait(t, func() {
		waits++
		// santad recreates the log between attempts.
		writeFile(t, base, mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/CUR", "ok", "aaa"))
	})

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err)
	require.Equal(t, 1, waits, "should have retried once")
	require.Equal(t, []string{"/ARC0", "/CUR"}, apps(got))
}

// TestScrapeSantaLogFromBase_MissingLogIsNotAnError verifies that a log that
// never shows up yields no rows and no error: Santa may not be installed, or may
// not be configured to log to a file. With no rotated log on disk there is no
// rotation in progress to wait for, so no host without Santa pays the retry delay.
func TestScrapeSantaLogFromBase_MissingLogIsNotAnError(t *testing.T) {
	tmp := t.TempDir()

	var waits int
	stubRetryWait(t, func() { waits++ })

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, filepath.Join(tmp, "santa.log"))
	require.NoError(t, err)
	require.Empty(t, got)
	require.Zero(t, waits, "should not wait when there are no archives to rotate")
}

// stubRetryWait replaces the between-attempts wait with onWait, so retries are
// driven by the test instead of the clock.
func stubRetryWait(tb testing.TB, onWait func()) {
	tb.Helper()
	original := retryWait
	tb.Cleanup(func() { retryWait = original })
	retryWait = func(context.Context) error {
		onWait()
		return nil
	}
}

// TestScrapeSantaLogFromBase_StopsOnceCapIsMet verifies that archives are not
// opened once the active log alone satisfies the entry cap, which is the common
// case in monitor mode. The archive here cannot be read, so opening it at all
// would surface an error.
func TestScrapeSantaLogFromBase_StopsOnceCapIsMet(t *testing.T) {
	oldCap := maxEntries
	maxEntries = 2
	defer func() { maxEntries = oldCap }()

	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	writeFile(t, base,
		mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/A", "ok", "aaa")+
			mkLine("decision=ALLOW", "2025-09-18 12:00:01.000", "/B", "ok", "bbb")+
			mkLine("decision=ALLOW", "2025-09-18 12:00:02.000", "/C", "ok", "ccc"))
	writeFile(t, base+".0.gz", "definitely not gzip")

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err, "the archive should not have been opened")
	require.Equal(t, []string{"/B", "/C"}, apps(got))
}

// TestScrapeSantaLogFromBase_StopsOnceOlderArchivesAreUnneeded verifies the same
// short circuit part way through the archives: enough events are found in the
// active log and the newest archive, so the older one is left alone.
func TestScrapeSantaLogFromBase_StopsOnceOlderArchivesAreUnneeded(t *testing.T) {
	oldCap := maxEntries
	maxEntries = 3
	defer func() { maxEntries = oldCap }()

	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	writeFile(t, base,
		mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/CUR1", "ok", "aaa")+
			mkLine("decision=ALLOW", "2025-09-18 12:00:01.000", "/CUR2", "ok", "bbb"))
	writeGz(t, base+".0.gz",
		mkLine("decision=ALLOW", "2025-09-18 11:59:58.000", "/ARC0-1", "ok", "ccc")+
			mkLine("decision=ALLOW", "2025-09-18 11:59:59.000", "/ARC0-2", "ok", "ddd"))
	writeFile(t, base+".1.gz", "definitely not gzip")

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err, "the older archive should not have been opened")
	require.Equal(t, []string{"/ARC0-2", "/CUR1", "/CUR2"}, apps(got))
}

func TestScrapeStream_EnforcesGlobalCap(t *testing.T) {
	// Lower the global cap to make the test fast and predictable.
	oldCap := maxEntries
	maxEntries = 1_000
	defer func() { maxEntries = oldCap }()

	const perLine = `[` +
		`2025-09-18 12:00:00.000` +
		`] santad: decision=ALLOW | path=/Applications/App.app | reason=ok | sha256=abc123` + "\n"

	var sb strings.Builder
	sb.Grow(len(perLine) * (maxEntries + 50)) // generate a bit more than the cap
	for i := 0; i < maxEntries+50; i++ {
		sb.WriteString(perLine)
	}

	rb := newRingBuffer(maxEntries)

	err := scrapeStream(t.Context(), strings.NewReader(sb.String()), decisionAllowed, rb)

	require.NoError(t, err, "cap should not surface as an error")
	require.Len(t, rb.SliceChrono(), maxEntries, "SliceChrono should return exactly maxEntries items")
}

func TestScrapeSantaLogFromBase_PrefersLatestWithinArchiveOnCap(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	// Keep the test fast and intentional.
	oldCap := maxEntries
	maxEntries = 3
	defer func() { maxEntries = oldCap }()

	writeFile(t, base, mkLine("decision=DENY", "2025-09-18 12:00:00.000", "/CUR-DENY", "ok", "aaa"))

	writeGz(t, base+".0.gz", mkLine("decision=DENY", "2025-09-18 11:59:59.500", "/ARC0-DENY", "ok", "bbb"))

	// Older archive (.1.gz): many DENY lines with increasing timestamps.
	// We want to ensure that when the cap is hit *inside this archive*,
	// the buffer ends up holding the *latest* lines from within it.
	var arc1 strings.Builder
	arc1.WriteString(mkLine("decision=DENY", "2025-09-18 11:59:59.001", "/DENY-1", "r", "d1"))
	arc1.WriteString(mkLine("decision=DENY", "2025-09-18 11:59:59.002", "/DENY-2", "r", "d2"))
	arc1.WriteString(mkLine("decision=DENY", "2025-09-18 11:59:59.003", "/DENY-3", "r", "d3"))
	arc1.WriteString(mkLine("decision=DENY", "2025-09-18 11:59:59.004", "/DENY-4", "r", "d4"))
	arc1.WriteString(mkLine("decision=DENY", "2025-09-18 11:59:59.005", "/DENY-5", "r", "d5"))
	writeGz(t, base+".1.gz", arc1.String())

	// Scan: archives oldest→newest (.1.gz then .0.gz), then current last.
	// Since only .1.gz has DENY lines and it contains more than maxEntries,
	// the ring should end up with the last 3 from that archive:
	// "/DENY-3", "/DENY-4", "/DENY-5" (chronological).
	got, err := scrapeSantaLogFromBase(context.Background(), decisionDenied, base)
	require.NoError(t, err)

	require.Equal(t,
		[]string{"/DENY-5", "/ARC0-DENY", "/CUR-DENY"},
		[]string{got[0].Application, got[1].Application, got[2].Application},
		"should keep the latest entries within the archive when hitting the cap",
	)

	maxEntries = 2
	got, err = scrapeSantaLogFromBase(context.Background(), decisionDenied, base)
	require.NoError(t, err)

	require.Equal(t,
		[]string{"/ARC0-DENY", "/CUR-DENY"},
		[]string{got[0].Application, got[1].Application},
		"with a smaller cap, should keep the latest entries within the archive",
	)

	maxEntries = 1
	got, err = scrapeSantaLogFromBase(context.Background(), decisionDenied, base)
	require.NoError(t, err)

	require.Equal(t,
		[]string{"/CUR-DENY"},
		[]string{got[0].Application},
		"with a cap of 1, should keep only the latest entry overall",
	)
}

func TestGenerateAllowed_ReturnsRows(t *testing.T) {
	base := stubLogPath(t)
	writeFile(t, base, mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/A", "cdhash", "aaa")+
		mkLine("decision=DENY", "2025-09-18 12:00:01.000", "/B", "rule", "bbb"))

	rows, err := GenerateAllowed(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	require.Equal(t, []map[string]string{{
		"timestamp":   "2025-09-18 12:00:00.000",
		"application": "/A",
		"reason":      "cdhash",
		"sha256":      "aaa",
	}}, rows)
}

// TestGenerateAllowed_LogsFailureWithoutFailingTheQuery verifies that an
// unreadable log is reported in fleetd's log rather than silently returning zero
// rows, and that the table itself does not fail: an error here would break the
// query on every host running Santa, and a synthetic error row would look like a
// real Santa event.
func TestGenerateAllowed_LogsFailureWithoutFailingTheQuery(t *testing.T) {
	base := stubLogPath(t)
	// A path that exists but cannot be read as a file.
	require.NoError(t, os.Mkdir(base, 0o755))

	logs := captureLogs(t)

	rows, err := GenerateAllowed(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	require.Empty(t, rows)
	require.Contains(t, logs.String(), "scraping santa log")
}

func TestGenerateDenied_ReturnsRows(t *testing.T) {
	base := stubLogPath(t)
	writeFile(t, base, mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/A", "cdhash", "aaa")+
		mkLine("decision=DENY", "2025-09-18 12:00:01.000", "/B", "rule", "bbb"))

	rows, err := GenerateDenied(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	require.Equal(t, []map[string]string{{
		"timestamp":   "2025-09-18 12:00:01.000",
		"application": "/B",
		"reason":      "rule",
		"sha256":      "bbb",
	}}, rows)
}

// stubLogPath points the tables at a temporary log for the duration of the test
// and returns its path.
func stubLogPath(tb testing.TB) string {
	tb.Helper()
	original := logPath
	tb.Cleanup(func() { logPath = original })
	logPath = filepath.Join(tb.TempDir(), "santa.log")
	return logPath
}

// captureLogs redirects the global zerolog logger into a buffer.
func captureLogs(tb testing.TB) *bytes.Buffer {
	tb.Helper()
	var buf bytes.Buffer
	original := log.Logger
	tb.Cleanup(func() { log.Logger = original })
	log.Logger = zerolog.New(&buf)
	return &buf
}

func writeFile(tb testing.TB, path, content string) {
	tb.Helper()
	require.NoError(tb, os.WriteFile(path, []byte(content), 0o644))
}

func writeGz(tb testing.TB, path, content string) {
	tb.Helper()
	f, err := os.Create(path)
	require.NoError(tb, err)
	gz := gzip.NewWriter(f)
	_, err = gz.Write([]byte(content))
	require.NoError(tb, err)
	require.NoError(tb, gz.Close())
	require.NoError(tb, f.Close())
}

func mkLine(dec, ts, path, reason, sha string) string {
	// example Santa line format
	return "[" + ts + "] santad: " + dec +
		` | path="` + path + `" | reason=` + reason + ` | sha256=` + sha + "\n"
}

// apps lists the application column of entries, in the order returned.
func apps(entries []logEntry) []string {
	out := make([]string, len(entries))
	for i := range entries {
		out[i] = entries[i].Application
	}
	return out
}

// mkLineWithArgs builds a line with a trailing args field, which is where Santa
// puts the process arguments and the only field with no practical size bound.
func mkLineWithArgs(dec, ts, path, reason, sha, args string) string {
	return strings.TrimSuffix(mkLine(dec, ts, path, reason, sha), "\n") +
		" | args=" + args + "\n"
}

// writeTruncatedGz writes a gzip stream containing keep followed by drop, then
// cuts the file at the flush boundary between them: keep decodes cleanly and
// the stream then ends unexpectedly, as it does while newsyslog is still
// compressing a rotated log.
func writeTruncatedGz(tb testing.TB, path, keep, drop string) {
	tb.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte(keep))
	require.NoError(tb, err)
	require.NoError(tb, gz.Flush())
	boundary := buf.Len()
	_, err = gz.Write([]byte(drop))
	require.NoError(tb, err)
	require.NoError(tb, gz.Close())

	require.NoError(tb, os.WriteFile(path, buf.Bytes()[:boundary], 0o644))
}

//////////////////
// BENCHMARKS
// Santa log scraping can be slow due to potentially large files and
// multiple compressed archives. These benchmarks help track performance
// over time.
//
// goos: darwin
// goarch: arm64
// cpu: Apple M2 Pro
//////////////////

// Small (~150KB) non-compressed
// BenchmarkScrapeSantaLogFromBase_SmallPlain-12               1436            827449 ns/op         185.63 MB/s      966170 B/op       5060 allocs/op
func BenchmarkScrapeSantaLogFromBase_SmallPlain(b *testing.B) {
	tmp := b.TempDir()
	base := filepath.Join(tmp, "santa.log")

	content := fillToSize(150*1024, "decision=ALLOW")
	writeFile(b, base, content)

	ctx := context.Background()
	b.SetBytes(int64(len(content)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if _, err := scrapeSantaLogFromBase(ctx, decisionAllowed, base); err != nil {
			b.Fatal(err)
		}
	}
}

// ~10MB non-compressed
// BenchmarkScrapeSantaLogFromBase_10MB_Plain-12                 20          58003575 ns/op         180.78 MB/s    75833864 B/op     343898 allocs/op
func BenchmarkScrapeSantaLogFromBase_10MB_Plain(b *testing.B) {
	tmp := b.TempDir()
	base := filepath.Join(tmp, "santa.log")

	content := fillToSize(10*1024*1024, "decision=ALLOW")
	writeFile(b, base, content)

	ctx := context.Background()
	b.SetBytes(int64(len(content)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := scrapeSantaLogFromBase(ctx, decisionAllowed, base); err != nil {
			b.Fatal(err)
		}
	}
}

// ~10MB current log + five compressed archives (each ~10MB uncompressed)
// BenchmarkScrapeSantaLogFromBase_10MB_PlainPlus5x10MB_Gzip-12                   6         212764465 ns/op         295.70 MB/s    281107640 B/op   1298057 allocs/op
func BenchmarkScrapeSantaLogFromBase_10MB_PlainPlus5x10MB_Gzip(b *testing.B) {
	tmp := b.TempDir()
	base := filepath.Join(tmp, "santa.log")

	plain := fillToSize(10*1024*1024, "decision=ALLOW")
	writeFile(b, base, plain)

	totalUncompressed := len(plain)
	for i := 0; i < 5; i++ {
		dec := "decision=DENY"
		if i%2 == 1 {
			dec = "decision=ALLOW"
		}
		raw := fillToSize(10*1024*1024, dec)
		writeGz(b, base+fmt.Sprintf(".%d.gz", i), raw)
		totalUncompressed += len(raw)
	}

	ctx := context.Background()
	b.SetBytes(int64(totalUncompressed))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Choose either decision; archives contain both.
		if _, err := scrapeSantaLogFromBase(ctx, decisionDenied, base); err != nil {
			b.Fatal(err)
		}
	}
}

// ~10MB current log of ALLOW events plus five compressed archives, querying the
// events the current log already holds enough of: the archives should not be read.
// This is the shape of a busy host in monitor mode.
//
// Note that the reported MB/s counts the whole corpus, so it overstates real
// throughput: the point of this benchmark is that most of those bytes are skipped.
func BenchmarkScrapeSantaLogFromBase_CapMetByCurrentLog(b *testing.B) {
	tmp := b.TempDir()
	base := filepath.Join(tmp, "santa.log")

	plain := fillToSize(10*1024*1024, "decision=ALLOW")
	writeFile(b, base, plain)

	totalUncompressed := len(plain)
	for i := range 5 {
		raw := fillToSize(10*1024*1024, "decision=ALLOW")
		writeGz(b, base+fmt.Sprintf(".%d.gz", i), raw)
		totalUncompressed += len(raw)
	}

	ctx := context.Background()
	b.SetBytes(int64(totalUncompressed))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if _, err := scrapeSantaLogFromBase(ctx, decisionAllowed, base); err != nil {
			b.Fatal(err)
		}
	}
}

// fillToSize builds a string ≈ targetBytes by repeating mkLine(dec,...).
func fillToSize(targetBytes int, decision string) string {
	line := mkLine(decision,
		"2025-09-18 12:00:00.000",
		"/Applications/App.app",
		"ok",
		"deadbeefcafebabef00d",
	)
	ll := len(line)
	if ll == 0 {
		panic("mkLine returned empty line")
	}
	n := targetBytes / ll
	if n < 1 {
		n = 1
	}
	var sb strings.Builder
	sb.Grow(n * ll)
	for i := 0; i < n; i++ {
		sb.WriteString(line)
	}
	return sb.String()
}
