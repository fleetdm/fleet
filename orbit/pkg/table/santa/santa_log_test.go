//go:build darwin
// +build darwin

package santa

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractValues(t *testing.T) {
	tests := []struct {
		name string
		line string
		want map[string]string
	}{
		{
			name: "happy path with timestamp and kv pairs",
			line: `[2025-09-18T10:15:30.123Z] santad: decision=ALLOW | path=/Applications/Foo.app | reason=cdhash | sha256=abc123`,
			want: map[string]string{
				"timestamp": "2025-09-18T10:15:30.123Z",
				"decision":  "ALLOW",
				"path":      "/Applications/Foo.app",
				"reason":    "cdhash",
				"sha256":    "abc123",
			},
		},
		{
			name: "no santad preface returns only timestamp",
			line: `[2025-09-18 10:15:30] something else: decision=DENY | path=/bin/bash`,
			want: map[string]string{
				"timestamp": "2025-09-18 10:15:30",
			},
		},
		{
			name: "no timestamp but has kv pairs",
			line: `santad: decision=DENY | path=/usr/local/bin/tool | reason=rule | sha256=def456`,
			want: map[string]string{
				"decision": "DENY",
				"path":     "/usr/local/bin/tool",
				"reason":   "rule",
				"sha256":   "def456",
			},
		},
		{
			name: "trims spaces around keys and values",
			line: `[2025-09-18] santad:   decision = ALLOW   |   path = /a/b/c  | reason =  ok  `,
			want: map[string]string{
				"timestamp": "2025-09-18",
				"decision":  "ALLOW",
				"path":      "/a/b/c",
				"reason":    "ok",
			},
		},
		{
			name: "ignores empty segments and missing equals",
			line: `[ts] santad: decision=DENY | | path=/p | just-a-flag | sha256=zzz`,
			want: map[string]string{
				"timestamp": "ts",
				"decision":  "DENY",
				"path":      "/p",
				"sha256":    "zzz",
			},
		},
		{
			name: "value containing equals keeps everything after first equals",
			line: `[ts] santad: note=a=b=c | path=/eq | sha256=x`,
			want: map[string]string{
				"timestamp": "ts",
				"note":      "a=b=c",
				"path":      "/eq",
				"sha256":    "x",
			},
		},
		{
			name: "duplicate keys last one wins",
			line: `[ts] santad: path=/first | path=/second | reason=one | reason=two`,
			want: map[string]string{
				"timestamp": "ts",
				"path":      "/second",
				"reason":    "two",
			},
		},
		{
			name: "quoted values are preserved (current impl trims spaces only)",
			line: `[ts] santad: path="/Applications/App With Spaces.app" | reason='quoted'`,
			want: map[string]string{
				"timestamp": "ts",
				`path`:      `/Applications/App With Spaces.app`,
				`reason`:    `quoted`,
			},
		},
		{
			name: "no matches yields empty map",
			line: `completely unrelated line`,
			want: map[string]string{},
		},
		{
			name: "handles trailing separator",
			line: `[ts] santad: decision=ALLOW | path=/a/b/c |`,
			want: map[string]string{
				"timestamp": "ts",
				"decision":  "ALLOW",
				"path":      "/a/b/c",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := extractValues(tt.line)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("extractValues() mismatch\nline: %q\n got: %#v\nwant: %#v", tt.line, got, tt.want)
			}
		})
	}
}

func TestExtractValues_DoesNotPanicOnLongLine(t *testing.T) {
	// Construct a long line to ensure no unexpected behavior for big inputs.
	longVal := make([]byte, 0, 300_000)
	for i := 0; i < 10000; i++ {
		longVal = append(longVal, 'a')
	}
	line := "[2025-09-18] santad: path=/" + string(longVal) + " | reason=ok"

	got := extractValues(line)
	if got["timestamp"] != "2025-09-18" {
		t.Fatalf("timestamp parse failed, got %q", got["timestamp"])
	}
	if _, ok := got["path"]; !ok {
		t.Fatalf("expected path key to be present on long input")
	}
	if got["reason"] != "ok" {
		t.Fatalf("expected reason=ok, got %q", got["reason"])
	}
}

func TestScrapeSantaLogFromBase_EndToEnd(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "santa.log")

	// current (plain) log with ALLOW and DENY
	current := strings.Builder{}
	current.WriteString(mkLine("decision=ALLOW", "2025-09-18 12:00:00.000", "/Applications/A.app", "ok", "aaa"))
	current.WriteString(mkLine("decision=DENY", "2025-09-18 12:00:01.000", "/Applications/B.app", "rule", "bbb"))
	writeFile(t, base, current.String())

	// archive 0 (gz): a DENY that should appear for denied queries
	writeGz(t, base+".0.gz", mkLine("decision=DENY", "2025-09-18 11:59:59.000", "/Blocked/X", "blacklist", "xxx"))

	// archive 1 (gz): an ALLOW
	writeGz(t, base+".1.gz", mkLine("decision=ALLOW", "2025-09-18 11:59:58.000", "/OK/C", "scope", "ccc"))

	ctx := context.Background()

	denied, err := scrapeSantaLogFromBase(ctx, decisionDenied, base)
	require.NoError(t, err)
	// expect 2: one from current, one from archive 0
	require.Len(t, denied, 2)
	require.Equal(t, "/Applications/B.app", denied[0].Application)
	require.Equal(t, "/Blocked/X", denied[1].Application)

	allowed, err := scrapeSantaLogFromBase(ctx, decisionAllowed, base)
	require.NoError(t, err)
	// expect 2: one from current and one from archive 1
	require.Len(t, allowed, 2)
	require.Equal(t, "/Applications/A.app", allowed[0].Application)
	require.Equal(t, "/OK/C", allowed[1].Application)
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

func TestScrapeStream_Enforces10MBCap(t *testing.T) {
	// Lower the cap to make the test fast.
	oldCap := maxEntries
	maxEntries = 1_000
	defer func() { maxEntries = oldCap }()

	// Generate slightly more than the cap so we trigger the error quickly.
	const perLine = `[` +
		`2025-09-18 12:00:00.000` +
		`] santad: decision=ALLOW | path=/Applications/App.app | reason=ok | sha256=abc123` + "\n"

	var sb strings.Builder
	sb.Grow(len(perLine) * (maxEntries + 50))
	for i := 0; i < maxEntries+50; i++ {
		sb.WriteString(perLine)
	}

	sc := bufio.NewScanner(strings.NewReader(sb.String()))
	entries, err := scrapeStream(context.Background(), sc, decisionAllowed)

	require.Error(t, err, "expected cap error")
	require.Len(t, entries, maxEntries, "should stop at the cap")
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
