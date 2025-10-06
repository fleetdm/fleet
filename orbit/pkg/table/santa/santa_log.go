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
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

const (
	kLogEntryPreface = "santad: "
	defaultLogPath   = "/var/db/santa/santa.log"
)

var maxEntries = 10_000

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

var timestampRegex = regexp.MustCompile(`\[([^\]]+)\]`)

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
		log.Debug().Err(err).Msg("failed to scrape santa log")
		return []map[string]string{}, nil
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

func extractValues(line string) map[string]string {
	values := make(map[string]string, 8)

	if m := timestampRegex.FindStringSubmatch(line); len(m) > 1 {
		values["timestamp"] = m[1]
	}

	pos := strings.Index(line, kLogEntryPreface)
	if pos == -1 {
		return values
	}
	rest := line[pos+len(kLogEntryPreface):]

	for seg := range strings.SplitSeq(rest, "|") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		k, v, ok := strings.Cut(seg, "=")
		if !ok {
			continue
		}
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if k != "" && v != "" {
			values[k] = v
		}
	}
	return values
}

func scrapeStream(ctx context.Context, scanner *bufio.Scanner, decision santaDecisionType, rb *ringBuffer) error {
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()

		// Filter by decision type early to keep it fast.
		switch decision {
		case decisionAllowed:
			if !strings.Contains(line, "decision=ALLOW") {
				continue
			}
		case decisionDenied:
			if !strings.Contains(line, "decision=DENY") {
				continue
			}
		}

		values := extractValues(line)
		if values["timestamp"] == "" {
			continue
		}

		rb.Add(logEntry{
			Timestamp:   values["timestamp"],
			Application: values["path"],
			Reason:      values["reason"],
			SHA256:      values["sha256"],
		})
	}

	return scanner.Err()
}

func scrapeCurrentLog(ctx context.Context, path string, decision santaDecisionType, rb *ringBuffer) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open Santa log file: %v", err)
	}
	defer file.Close()

	scanner := makeBufferedScanner(file)
	return scrapeStream(ctx, scanner, decision, rb)
}

func scrapeCompressedSantaLog(ctx context.Context, path string, decision santaDecisionType, rb *ringBuffer) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open compressed log file %s: %v", path, err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader for %s: %v", path, err)
	}
	defer gzReader.Close()

	scanner := makeBufferedScanner(gzReader)
	return scrapeStream(ctx, scanner, decision, rb)
}

func makeBufferedScanner(r io.Reader) *bufio.Scanner {
	s := bufio.NewScanner(r)
	// Uncomment to support very large lines if needed:
	// buf := make([]byte, 64*1024)
	// s.Buffer(buf, 1<<20) // 1 MiB
	return s
}

func scrapeSantaLog(ctx context.Context, decision santaDecisionType) ([]logEntry, error) {
	return scrapeSantaLogFromBase(ctx, decision, defaultLogPath)
}

func scrapeSantaLogFromBase(ctx context.Context, decision santaDecisionType, path string) ([]logEntry, error) {
	rb := newRingBuffer(maxEntries)

	// Find highest archive index (0 = newest archive, higher = older)
	maxIdx := -1
	for i := 0; ; i++ {
		if _, err := os.Stat(fmt.Sprintf("%s.%d.gz", path, i)); err != nil {
			break
		}
		maxIdx = i
	}

	// 1) Archives oldest → newest: maxIdx, maxIdx-1, ..., 0
	for i := maxIdx; i >= 0; i-- {
		archivePath := fmt.Sprintf("%s.%d.gz", path, i)
		if err := scrapeCompressedSantaLog(ctx, archivePath, decision, rb); err != nil {
			return nil, err
		}
	}

	// 2) Current log last (newest overall)
	if err := scrapeCurrentLog(ctx, path, decision, rb); err != nil {
		return nil, err
	}

	// Return the last N entries (oldest → newest among those last N).
	// If you prefer newest-first, use rb.SliceReverse().
	return rb.SliceChrono(), nil
}
