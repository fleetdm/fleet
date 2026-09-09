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
			name:   "trims spaces around keys and values",
			line:   `[2025-09-18] santad:   decision = ALLOW   |   path = /a/b/c  | reason =  ok  `,
			want:   logEntry{Timestamp: "2025-09-18", Application: "/a/b/c", Reason: "ok"},
			wantOK: true,
		},
		{
			name:   "ignores empty segments and missing equals",
			line:   `[ts] santad: decision=DENY | | path=/p | just-a-flag | sha256=zzz`,
			want:   logEntry{Timestamp: "ts", Application: "/p", SHA256: "zzz"},
			wantOK: true,
		},
		{
			name:   "value containing equals keeps everything after the first equals",
			line:   `[ts] santad: note=a=b=c | path=/eq | sha256=x`,
			want:   logEntry{Timestamp: "ts", Application: "/eq", SHA256: "x"},
			wantOK: true,
		},
		{
			name:   "duplicate keys last one wins",
			line:   `[ts] santad: path=/first | path=/second | reason=one | reason=two`,
			want:   logEntry{Timestamp: "ts", Application: "/second", Reason: "two"},
			wantOK: true,
		},
		{
			name:   "quoted values are unquoted",
			line:   `[ts] santad: path="/Applications/App With Spaces.app" | reason='quoted'`,
			want:   logEntry{Timestamp: "ts", Application: "/Applications/App With Spaces.app", Reason: "quoted"},
			wantOK: true,
		},
		{
			name:   "keys are matched case-insensitively",
			line:   `[ts] santad: PATH=/upper | Reason=ok | SHA256=abc`,
			want:   logEntry{Timestamp: "ts", Application: "/upper", Reason: "ok", SHA256: "abc"},
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
			name:   "falls through an empty bracket group to the next one",
			line:   `[] [ts] santad: decision=ALLOW | path=/a`,
			want:   logEntry{Timestamp: "ts", Application: "/a"},
			wantOK: true,
		},
		{
			name: "unclosed bracket group is not a timestamp",
			line: `[2025-09-18 santad: decision=ALLOW | path=/a`,
		},
		{
			name:   "bracket group keeps a nested opening bracket",
			line:   `[a[b] santad: decision=ALLOW | path=/a`,
			want:   logEntry{Timestamp: "a[b", Application: "/a"},
			wantOK: true,
		},
		{
			name:   "handles trailing separator",
			line:   `[ts] santad: decision=ALLOW | path=/a/b/c |`,
			want:   logEntry{Timestamp: "ts", Application: "/a/b/c"},
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
	base := tempLog(t)
	writeFile(t, base, allow("/Applications/A.app")+deny("/Applications/B.app"))
	writeGz(t, base+".0.gz", deny("/Blocked/X"))
	writeGz(t, base+".1.gz", allow("/OK/C"))

	// Results are chronological regardless of the order the files are read in, so
	// the archived events come before the ones in the active log.
	denied, err := scrapeSantaLogFromBase(t.Context(), decisionDenied, base)
	require.NoError(t, err)
	require.Equal(t, []string{"/Blocked/X", "/Applications/B.app"}, apps(denied))

	allowed, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err)
	require.Equal(t, []string{"/OK/C", "/Applications/A.app"}, apps(allowed))
}

// TestScrapeSantaLogFromBase_Scenarios covers the on-disk states a scrape must
// survive. Every read is best effort, so a file that cannot be read is reported
// (wantErr) without discarding the entries collected from the other files.
func TestScrapeSantaLogFromBase_Scenarios(t *testing.T) {
	tests := []struct {
		name     string
		decision santaDecisionType // zero value is decisionAllowed
		cap      int               // overrides maxEntries when > 0
		setup    func(t *testing.T, base string)
		want     []string // application column, oldest first
		wantErr  bool
	}{
		{
			// Archive iteration stops cleanly when no rotated file exists at all.
			name:  "only the active log exists",
			setup: func(t *testing.T, base string) { writeFile(t, base, allow("/A")) },
			want:  []string{"/A"},
		},
		{
			// In monitor mode Santa logs an ALLOW line for nearly every exec, arguments
			// included, so a single long command line would otherwise empty the whole
			// table until it rotated out.
			name: "line over the retention limit does not discard the rest",
			setup: func(t *testing.T, base string) {
				writeFile(t, base, allow("/A")+longAllow("/B")+allow("/C"))
			},
			want: []string{"/A", "/B", "/C"},
		},
		{
			// The over-long line path through a compressed archive, where the reader is
			// fed in decompressed chunks rather than straight from a file.
			name: "line over the retention limit inside an archive",
			setup: func(t *testing.T, base string) {
				writeFile(t, base, allow("/CUR"))
				writeGz(t, base+".0.gz", allow("/A")+longAllow("/B")+allow("/C"))
			},
			want: []string{"/A", "/B", "/C", "/CUR"},
		},
		{
			// An archive still being written by newsyslog is not valid gzip data.
			name: "unreadable archive is reported but keeps other entries",
			setup: func(t *testing.T, base string) {
				writeFile(t, base, allow("/CUR"))
				writeFile(t, base+".0.gz", "definitely not gzip")
			},
			want:    []string{"/CUR"},
			wantErr: true,
		},
		{
			// newsyslog has rotated the log but not finished compressing it: the .gz is
			// incomplete while the uncompressed sibling is still on disk.
			name: "corrupt archive falls back to its uncompressed sibling",
			setup: func(t *testing.T, base string) {
				writeFile(t, base, allow("/CUR"))
				writeFile(t, base+".0.gz", "definitely not gzip")
				writeFile(t, base+".0", allow("/ARC0"))
			},
			want: []string{"/ARC0", "/CUR"},
		},
		{
			// Whether rotated logs are compressed at all depends on the host's
			// newsyslog configuration.
			name: "reads uncompressed archives",
			setup: func(t *testing.T, base string) {
				writeFile(t, base, allow("/CUR"))
				writeFile(t, base+".0", allow("/ARC0"))
				writeFile(t, base+".1", allow("/ARC1"))
			},
			want: []string{"/ARC1", "/ARC0", "/CUR"},
		},
		{
			// A missing active log is benign: Santa may not be installed, or the log
			// may have just been rotated. Archived entries must survive it.
			name: "missing active log keeps archives",
			setup: func(t *testing.T, base string) {
				writeGz(t, base+".0.gz", allow("/ARC0"))
			},
			want: []string{"/ARC0"},
		},
		{
			// newsyslog has renamed santa.log and santad has not recreated it yet: the
			// pre-rotation events are read from the renamed, not-yet-compressed file.
			name: "missing active log mid-rotation",
			setup: func(t *testing.T, base string) {
				writeFile(t, base+".0", allow("/ARC0"))
			},
			want: []string{"/ARC0"},
		},
		{
			name:  "missing log entirely yields no rows and no error",
			setup: func(*testing.T, string) {},
			want:  []string{},
		},
		{
			// A gzip stream cut short still yields the entries decoded before the failure.
			name: "truncated gzip keeps the entries decoded before the cut",
			setup: func(t *testing.T, base string) {
				writeFile(t, base, allow("/CUR"))
				writeTruncatedGz(t, base+".0.gz", allow("/DECODED"), allow("/LOST"))
			},
			want:    []string{"/DECODED", "/CUR"},
			wantErr: true,
		},
		{
			// santad terminates every line with a newline, so an unterminated final
			// line is a partial write, not an event.
			name: "skips an unterminated final line",
			setup: func(t *testing.T, base string) {
				writeFile(t, base, allow("/A")+`[ts] santad: decision=ALLOW | path="/PARTIAL" | rea`)
			},
			want: []string{"/A"},
		},
		{
			// Archives are not opened once the active log alone satisfies the entry
			// cap, which is the common case in monitor mode. The archive here cannot
			// be read, so opening it at all would surface an error.
			name: "archives are not opened once the cap is met",
			cap:  2,
			setup: func(t *testing.T, base string) {
				writeFile(t, base, allow("/A")+allow("/B")+allow("/C"))
				writeFile(t, base+".0.gz", "definitely not gzip")
			},
			want: []string{"/B", "/C"},
		},
		{
			// The same short circuit part way through the archives: enough events are
			// found in the active log and the newest archive, so the older unreadable
			// one is left alone.
			name: "older archives are not opened once the cap is met",
			cap:  3,
			setup: func(t *testing.T, base string) {
				writeFile(t, base, allow("/CUR1")+allow("/CUR2"))
				writeGz(t, base+".0.gz", allow("/ARC0-1")+allow("/ARC0-2"))
				writeFile(t, base+".1.gz", "definitely not gzip")
			},
			want: []string{"/ARC0-2", "/CUR1", "/CUR2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cap > 0 {
				setCap(t, tt.cap)
			}
			base := tempLog(t)
			tt.setup(t, base)

			got, err := scrapeSantaLogFromBase(t.Context(), tt.decision, base)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want, apps(got))
		})
	}
}

// TestScrapeSantaLogFromBase_TruncatesLongLineKeepingColumns verifies that
// truncating an over-long line preserves every column these tables expose.
// Santa emits the unbounded args field last, after path, reason and sha256.
func TestScrapeSantaLogFromBase_TruncatesLongLineKeepingColumns(t *testing.T) {
	base := tempLog(t)
	writeFile(t, base, longAllow("/Long"))

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err)
	require.Equal(t, []logEntry{{Timestamp: testTS, Application: "/Long", Reason: "ok", SHA256: "aaa"}}, got)
}

// TestScrapeSantaLogFromBase_LineAtRetentionLimit covers the boundary between the
// single-read path and the truncating one.
func TestScrapeSantaLogFromBase_LineAtRetentionLimit(t *testing.T) {
	base := tempLog(t)

	for _, length := range []int{maxLineBytes - 1, maxLineBytes, maxLineBytes + 1} {
		line := mkLineWithArgs("decision=ALLOW", testTS, "/A", "ok", "aaa", "")
		// Pad the args field so the line is exactly length bytes, newline included.
		line = strings.TrimSuffix(line, "\n") + strings.Repeat("x", length-len(line)) + "\n"
		require.Len(t, line, length)
		writeFile(t, base, line)

		got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
		require.NoError(t, err, "line length %d", length)
		require.Equal(t, []logEntry{{Timestamp: testTS, Application: "/A", Reason: "ok", SHA256: "aaa"}}, got,
			"line length %d", length)
	}
}

// TestScrapeSantaLogFromBase_CanceledContext verifies that a canceled query is
// reported rather than looking like an empty log.
func TestScrapeSantaLogFromBase_CanceledContext(t *testing.T) {
	base := tempLog(t)
	writeFile(t, base, allow("/A"))

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
	base := tempLog(t)
	writeFile(t, base, allow("/CUR"))
	writeGz(t, base+".1.gz", allow("/ARC1"))

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

// TestScrapeSantaLogFromBase_RediscoversArchivesAfterRotation covers a rotation
// landing between discovering the archives and reading the active log, on a host
// that had no archives at all: the events are in a santa.log.0 that the first
// discovery pass ran too early to see.
func TestScrapeSantaLogFromBase_RediscoversArchivesAfterRotation(t *testing.T) {
	base := tempLog(t)

	// The renamed log is on disk; the active one has not been recreated yet.
	writeFile(t, base+".0", allow("/ARC0"))

	// Each discovery pass starts by looking for santa.log.0.gz. The first pass sees
	// nothing, as it would if it ran a moment before newsyslog's rename.
	original := statFile
	t.Cleanup(func() { statFile = original })
	passes := 0
	statFile = func(path string) (os.FileInfo, error) {
		if strings.HasSuffix(path, ".0.gz") {
			passes++
		}
		if passes == 1 {
			return nil, os.ErrNotExist
		}
		return original(path)
	}

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err)
	require.Equal(t, 2, passes, "should have looked for archives again")
	require.Equal(t, []string{"/ARC0"}, apps(got))
}

// TestScrapeSantaLogFromBase_RediscoversShiftedArchivesAfterRotation covers the
// same race on a host that already had compressed archives: the rename shifts
// every archive up one index and leaves the ex-active events in a plain
// santa.log.0, so the pre-rotation list points at files that no longer exist.
func TestScrapeSantaLogFromBase_RediscoversShiftedArchivesAfterRotation(t *testing.T) {
	base := tempLog(t)

	// Post-rotation disk state: no santa.log, no santa.log.0.gz; the ex-active
	// events sit uncompressed at santa.log.0 and the old archive moved to .1.gz.
	writeFile(t, base+".0", allow("/NEW"))
	writeGz(t, base+".1.gz", allow("/OLD"))

	// The first discovery pass ran a moment before the rename, when the only file
	// besides the active log was santa.log.0.gz.
	original := statFile
	t.Cleanup(func() { statFile = original })
	passes := 0
	statFile = func(path string) (os.FileInfo, error) {
		if strings.HasSuffix(path, ".0.gz") {
			passes++
		}
		if passes == 1 {
			if strings.HasSuffix(path, ".0.gz") {
				return original(base + ".0")
			}
			return nil, os.ErrNotExist
		}
		return original(path)
	}

	got, err := scrapeSantaLogFromBase(t.Context(), decisionAllowed, base)
	require.NoError(t, err)
	require.Equal(t, 2, passes, "should have looked for archives again")
	require.Equal(t, []string{"/OLD", "/NEW"}, apps(got))
}

func TestScrapeStream_EnforcesGlobalCap(t *testing.T) {
	// Lower the global cap to make the test fast and predictable.
	setCap(t, 1_000)

	rb := newRingBuffer(maxEntries)
	stream := strings.NewReader(strings.Repeat(allow("/Applications/App.app"), maxEntries+50))

	err := scrapeStream(t.Context(), stream, decisionAllowed, rb)
	require.NoError(t, err, "cap should not surface as an error")
	require.Len(t, rb.SliceChrono(), maxEntries, "SliceChrono should return exactly maxEntries items")
}

func TestScrapeSantaLogFromBase_PrefersLatestWithinArchiveOnCap(t *testing.T) {
	base := tempLog(t)
	writeFile(t, base, deny("/CUR-DENY"))
	writeGz(t, base+".0.gz", deny("/ARC0-DENY"))
	// The older archive holds more DENY lines than the cap leaves room for, so the
	// buffer must end up holding the latest lines from within it.
	writeGz(t, base+".1.gz", deny("/DENY-1")+deny("/DENY-2")+deny("/DENY-3")+deny("/DENY-4")+deny("/DENY-5"))

	setCap(t, 3)
	got, err := scrapeSantaLogFromBase(t.Context(), decisionDenied, base)
	require.NoError(t, err)
	require.Equal(t, []string{"/DENY-5", "/ARC0-DENY", "/CUR-DENY"}, apps(got),
		"should keep the latest entries within the archive when hitting the cap")

	maxEntries = 2
	got, err = scrapeSantaLogFromBase(t.Context(), decisionDenied, base)
	require.NoError(t, err)
	require.Equal(t, []string{"/ARC0-DENY", "/CUR-DENY"}, apps(got),
		"with a smaller cap, should keep the latest entries within the archive")

	maxEntries = 1
	got, err = scrapeSantaLogFromBase(t.Context(), decisionDenied, base)
	require.NoError(t, err)
	require.Equal(t, []string{"/CUR-DENY"}, apps(got),
		"with a cap of 1, should keep only the latest entry overall")
}

func TestGenerateAllowed_ReturnsRows(t *testing.T) {
	writeFile(t, stubLogPath(t), allow("/A")+deny("/B"))

	rows, err := GenerateAllowed(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	require.Equal(t, []map[string]string{
		{"timestamp": testTS, "application": "/A", "reason": "ok", "sha256": "aaa"},
	}, rows)
}

// TestGenerateAllowed_LogsFailureWithoutFailingTheQuery verifies that an
// unreadable log is reported in fleetd's log rather than silently returning zero
// rows, and that the table itself does not fail: an error here would break the
// query on every host running Santa, and a synthetic error row would look like a
// real Santa event.
func TestGenerateAllowed_LogsFailureWithoutFailingTheQuery(t *testing.T) {
	// A path that exists but cannot be read as a file.
	require.NoError(t, os.Mkdir(stubLogPath(t), 0o755))

	logs := captureLogs(t)

	rows, err := GenerateAllowed(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	require.Empty(t, rows)
	require.Contains(t, logs.String(), "scraping santa log")
}

func TestGenerateDenied_ReturnsRows(t *testing.T) {
	writeFile(t, stubLogPath(t), allow("/A")+deny("/B"))

	rows, err := GenerateDenied(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	require.Equal(t, []map[string]string{
		{"timestamp": testTS, "application": "/B", "reason": "rule", "sha256": "bbb"},
	}, rows)
}

// testTS is the timestamp the line helpers below stamp on every entry. Results
// are ordered by file and line position, never by parsing timestamps, so the
// tests do not need distinct ones.
const testTS = "2025-09-18 12:00:00.000"

func allow(path string) string {
	return mkLine("decision=ALLOW", testTS, path, "ok", "aaa")
}

func deny(path string) string {
	return mkLine("decision=DENY", testTS, path, "rule", "bbb")
}

// longAllow is an ALLOW line whose args field pushes it far past the retention
// limit, as monitor mode produces for an exec with a long command line.
func longAllow(path string) string {
	return mkLineWithArgs("decision=ALLOW", testTS, path, "ok", "aaa", strings.Repeat("x", 200_000))
}

func mkLine(dec, ts, path, reason, sha string) string {
	// example Santa line format
	return "[" + ts + "] santad: " + dec +
		` | path="` + path + `" | reason=` + reason + ` | sha256=` + sha + "\n"
}

// mkLineWithArgs builds a line with a trailing args field, which is where Santa
// puts the process arguments and the only field with no practical size bound.
func mkLineWithArgs(dec, ts, path, reason, sha, args string) string {
	return strings.TrimSuffix(mkLine(dec, ts, path, reason, sha), "\n") +
		" | args=" + args + "\n"
}

// tempLog returns the santa.log path inside a fresh temporary directory.
func tempLog(tb testing.TB) string {
	return filepath.Join(tb.TempDir(), "santa.log")
}

// setCap lowers the global entry cap for the duration of the test.
func setCap(tb testing.TB, n int) {
	tb.Helper()
	original := maxEntries
	tb.Cleanup(func() { maxEntries = original })
	maxEntries = n
}

// stubLogPath points the tables at a temporary log for the duration of the test
// and returns its path.
func stubLogPath(tb testing.TB) string {
	tb.Helper()
	original := logPath
	tb.Cleanup(func() { logPath = original })
	logPath = tempLog(tb)
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

// apps lists the application column of entries, in the order returned.
func apps(entries []logEntry) []string {
	out := make([]string, len(entries))
	for i := range entries {
		out[i] = entries[i].Application
	}
	return out
}

//////////////////
// BENCHMARKS
// Santa log scraping can be slow due to potentially large files and
// multiple compressed archives. These benchmarks help track performance
// over time.
//
// Recorded on:
// goos: darwin
// goarch: arm64
// cpu: Apple M4 Max
//////////////////

// benchScrape scrapes base once per iteration, reporting allocations and
// throughput over corpusBytes.
func benchScrape(b *testing.B, base string, decision santaDecisionType, corpusBytes int) {
	ctx := b.Context()
	b.SetBytes(int64(corpusBytes))
	b.ReportAllocs()

	for b.Loop() {
		if _, err := scrapeSantaLogFromBase(ctx, decision, base); err != nil {
			b.Fatal(err)
		}
	}
}

// Small (~150KB) non-compressed
// BenchmarkScrapeSantaLogFromBase_SmallPlain-16              10514            226687 ns/op         677.58 MB/s      504616 B/op       5060 allocs/op
func BenchmarkScrapeSantaLogFromBase_SmallPlain(b *testing.B) {
	base := tempLog(b)
	content := fillToSize(150*1024, "decision=ALLOW")
	writeFile(b, base, content)

	benchScrape(b, base, decisionAllowed, len(content))
}

// ~10MB non-compressed
// BenchmarkScrapeSantaLogFromBase_10MB_Plain-16                177          13231784 ns/op         792.46 MB/s     8772758 B/op     343823 allocs/op
func BenchmarkScrapeSantaLogFromBase_10MB_Plain(b *testing.B) {
	base := tempLog(b)
	content := fillToSize(10*1024*1024, "decision=ALLOW")
	writeFile(b, base, content)

	benchScrape(b, base, decisionAllowed, len(content))
}

// ~10MB current log + five compressed archives (each ~10MB uncompressed), querying
// events the archives hold. Reading stops once the entry cap is met, so the reported
// MB/s counts bytes that were never read and overstates real throughput.
// BenchmarkScrapeSantaLogFromBase_10MB_PlainPlus5x10MB_Gzip-16                 135          17888254 ns/op        3517.07 MB/s     8942730 B/op    346729 allocs/op
func BenchmarkScrapeSantaLogFromBase_10MB_PlainPlus5x10MB_Gzip(b *testing.B) {
	base := tempLog(b)
	plain := fillToSize(10*1024*1024, "decision=ALLOW")
	writeFile(b, base, plain)

	totalUncompressed := len(plain)
	for i := range 5 {
		dec := "decision=DENY"
		if i%2 == 1 {
			dec = "decision=ALLOW"
		}
		raw := fillToSize(10*1024*1024, dec)
		writeGz(b, base+fmt.Sprintf(".%d.gz", i), raw)
		totalUncompressed += len(raw)
	}

	// Choose either decision; archives contain both.
	benchScrape(b, base, decisionDenied, totalUncompressed)
}

// ~10MB current log of ALLOW events plus five compressed archives, querying the
// events the current log already holds enough of: the archives should not be read.
// This is the shape of a busy host in monitor mode.
//
// Note that the reported MB/s counts the whole corpus, so it overstates real
// throughput: the point of this benchmark is that most of those bytes are skipped.
// BenchmarkScrapeSantaLogFromBase_CapMetByCurrentLog-16                        182          13192218 ns/op        4769.02 MB/s     8778596 B/op    343872 allocs/op
func BenchmarkScrapeSantaLogFromBase_CapMetByCurrentLog(b *testing.B) {
	base := tempLog(b)
	plain := fillToSize(10*1024*1024, "decision=ALLOW")
	writeFile(b, base, plain)

	totalUncompressed := len(plain)
	for i := range 5 {
		raw := fillToSize(10*1024*1024, "decision=ALLOW")
		writeGz(b, base+fmt.Sprintf(".%d.gz", i), raw)
		totalUncompressed += len(raw)
	}

	benchScrape(b, base, decisionAllowed, totalUncompressed)
}

// fillToSize builds a string ≈ targetBytes by repeating mkLine(dec,...).
func fillToSize(targetBytes int, decision string) string {
	line := mkLine(decision, testTS, "/Applications/App.app", "ok", "deadbeefcafebabef00d")
	return strings.Repeat(line, max(targetBytes/len(line), 1))
}
