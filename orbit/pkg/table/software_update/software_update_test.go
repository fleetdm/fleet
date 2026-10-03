package software_update

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

func fixture(t *testing.T, name string) string {
	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return string(b)
}

func row(required, label, title, version, sizeKiB, recommended, action string) map[string]string {
	return map[string]string{
		"software_update_required": required,
		"label":                    label,
		"title":                    title,
		"version":                  version,
		"size_kib":                 sizeKiB,
		"recommended":              recommended,
		"action":                   action,
	}
}

func TestColumns(t *testing.T) {
	require.Equal(t, []table.ColumnDefinition{
		{Name: "software_update_required", Type: "INTEGER"},
		{Name: "label", Type: "TEXT"},
		{Name: "title", Type: "TEXT"},
		{Name: "version", Type: "TEXT"},
		{Name: "size_kib", Type: "INTEGER"},
		{Name: "recommended", Type: "INTEGER"},
		{Name: "action", Type: "TEXT"},
	}, Columns())
}

func TestParseListUpdatesAvailable(t *testing.T) {
	require.Equal(t, []map[string]string{
		row("1", "Safari27.0TahoeAuto-27.0", "Safari", "27.0", "249465", "1", ""),
		row("1", "SFSymbolsAuto-27.0", "SF Symbols", "27.0", "430826", "1", ""),
		row("1", "macOS Tahoe 26.7-25G229", "macOS Tahoe 26.7", "26.7", "2960352", "1", "restart"),
		row("1", "macOS 27-26A428", "macOS 27", "27", "11727573", "1", "restart"),
	}, parseList(fixture(t, "updates_available.txt")))
}

func TestParseListNoUpdates(t *testing.T) {
	require.Equal(t, []map[string]string{row("0", "", "", "", "", "", "")}, parseList(fixture(t, "no_updates.txt")))
}

func TestParseListUnrecognizedOutputFailsClosed(t *testing.T) {
	for name, out := range map[string]string{
		"empty":           "",
		"scan only":       "Software Update Tool\n\nFinding available software\n",
		"header no items": "Software Update Tool\n\nFinding available software\nSoftware Update found the following new or updated software:\n",
		"legacy format":   "Software Update Tool\n\nFinding available software\n   * macOS Catalina 10.15.7-19H2\n\tmacOS Catalina 10.15.7 (19H2), 3070500K [recommended] [restart]\n",
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, []map[string]string{row("1", "", "", "", "", "", "")}, parseList(out))
		})
	}
}

func TestParseListTolerantOfFormatVariations(t *testing.T) {
	out := "Software Update Tool\r\n" +
		"\r\n" +
		"Finding available software\r\n" +
		"Some progress line that is not part of the list\r\n" +
		"Software Update found the following new or updated software:\r\n" +
		"* Label: Foo, Bar Tools-1.0\r\n" +
		"\tTitle: Foo, Bar Tools, Version: 1.0.2, Size: 10KiB, Recommended: NO, Deferred: YES, Action: restart, \r\n" +
		"* Label: LabelOnly-2.0\r\n" +
		"* Label: OddSize-3.0\r\n" +
		"\tTitle: Odd Size, Version: 3.0, Size: 12MB, Recommended: maybe\r\n"

	require.Equal(t, []map[string]string{
		// Comma inside the title survives; the unknown Deferred field doesn't bleed into Action.
		row("1", "Foo, Bar Tools-1.0", "Foo, Bar Tools", "1.0.2", "10", "0", "restart"),
		// A label with no detail line still counts as an available update.
		row("1", "LabelOnly-2.0", "", "", "", "", ""),
		// Values in an unexpected shape are left empty rather than guessed.
		row("1", "OddSize-3.0", "Odd Size", "3.0", "", "", ""),
	}, parseList(out))
}

func TestParseSizeKiB(t *testing.T) {
	for in, want := range map[string]string{
		"249465KiB": "249465",
		"0KiB":      "0",
		"249465":    "",
		"12MB":      "",
		"abcKiB":    "",
		"-5KiB":     "",
		"":          "",
	} {
		require.Equal(t, want, parseSizeKiB(in), "input %q", in)
	}
}

type fakeRunner struct {
	calls   atomic.Int32
	out     string
	err     error
	started chan struct{} // closed once the run has begun, when non-nil
	release chan struct{} // run blocks until closed, when non-nil
}

func (f *fakeRunner) run(ctx context.Context) (string, error) {
	f.calls.Add(1)
	if f.started != nil {
		close(f.started)
	}
	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return f.out, f.err
}

func TestRowsScanErrorIsReturned(t *testing.T) {
	scanErr := errors.New("The Internet connection appears to be offline")
	l := newLister((&fakeRunner{err: scanErr}).run)

	rows, err := l.rows(context.Background())
	require.ErrorIs(t, err, scanErr)
	require.Nil(t, rows)

	// A failed scan is never cached: the next query tries again.
	_, err = l.rows(context.Background())
	require.ErrorIs(t, err, scanErr)
}

func TestRowsCachesSuccessfulScan(t *testing.T) {
	f := &fakeRunner{out: fixture(t, "updates_available.txt")}
	l := newLister(f.run)
	now := time.Now()
	l.now = func() time.Time { return now }

	first, err := l.rows(context.Background())
	require.NoError(t, err)
	require.Len(t, first, 4)

	second, err := l.rows(context.Background())
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.EqualValues(t, 1, f.calls.Load(), "second query within the TTL must be served from cache")

	// Callers get their own copy so mutating a result can't poison the cache.
	second[0]["label"] = "mutated"
	third, err := l.rows(context.Background())
	require.NoError(t, err)
	require.Equal(t, first, third)

	now = now.Add(cacheTTL + time.Second)
	_, err = l.rows(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 2, f.calls.Load(), "query after the TTL must rescan")
}

func TestRowsConcurrentQueriesShareOneScan(t *testing.T) {
	f := &fakeRunner{
		out:     fixture(t, "no_updates.txt"),
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	l := newLister(f.run)

	const callers = 5
	results := make([][]map[string]string, callers)
	errs := make([]error, callers)
	var wg sync.WaitGroup
	for i := range callers {
		wg.Go(func() {
			results[i], errs[i] = l.rows(context.Background())
		})
	}

	<-f.started
	// Give the remaining callers a moment to queue up behind the in-flight scan.
	require.Eventually(t, func() bool { return int(f.calls.Load()) == 1 }, time.Second, 10*time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	close(f.release)
	wg.Wait()

	require.EqualValues(t, 1, f.calls.Load())
	for i := range callers {
		require.NoError(t, errs[i])
		require.Equal(t, []map[string]string{row("0", "", "", "", "", "", "")}, results[i])
	}
}

func TestRowsCanceledCallerDoesNotAbortScan(t *testing.T) {
	f := &fakeRunner{
		out:     fixture(t, "no_updates.txt"),
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	l := newLister(f.run)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := l.rows(ctx)
		errCh <- err
	}()
	<-f.started
	cancel()
	require.ErrorIs(t, <-errCh, context.Canceled)

	// The scan itself keeps running on its own deadline and its result lands in the
	// cache, so the next query is served without a second scan.
	close(f.release)
	require.Eventually(t, func() bool {
		_, ok := l.fresh()
		return ok
	}, time.Second, 10*time.Millisecond)

	rows, err := l.rows(context.Background())
	require.NoError(t, err)
	require.Equal(t, []map[string]string{row("0", "", "", "", "", "", "")}, rows)
	require.EqualValues(t, 1, f.calls.Load())
}

// TestGenerateLive runs the real softwareupdate scan. It needs a Mac with network access
// and takes a while, so it only runs when SOFTWARE_UPDATE_LIVE_TEST is set.
func TestGenerateLive(t *testing.T) {
	if os.Getenv("SOFTWARE_UPDATE_LIVE_TEST") == "" {
		t.Skip("set SOFTWARE_UPDATE_LIVE_TEST=1 to run the live softwareupdate scan")
	}
	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()

	rows, err := Generate(ctx, table.QueryContext{})
	require.NoError(t, err)
	require.NotEmpty(t, rows)

	required := rows[0]["software_update_required"]
	require.Contains(t, []string{"0", "1"}, required)
	for _, r := range rows {
		require.Equal(t, required, r["software_update_required"], "every row carries the same flag")
		if required == "0" {
			require.Equal(t, row("0", "", "", "", "", "", ""), r)
		} else {
			require.NotEmpty(t, r["label"])
		}
		t.Logf("%v", r)
	}
}
