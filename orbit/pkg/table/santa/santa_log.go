//go:build darwin

// Package santa implements the tables for getting Santa data
// (logs, rules, status) on macOS.
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

var maxEntries = 10 * 1024 * 1024 // 10 MB worth of log entries (approximately 60,000 lines of 170 bytes each)

// SantaDecisionType represents the type of Santa decision
type santaDecisionType int

const (
	decisionAllowed santaDecisionType = iota
	decisionDenied
)

// LogEntry represents a Santa log entry
type logEntry struct {
	Timestamp   string
	Application string
	Reason      string
	SHA256      string
}

var timestampRegex = regexp.MustCompile(`\[([^\]]+)\]`)

// LogColumns returns the column definitions for the santa_allowed table
func LogColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("timestamp"),
		table.TextColumn("application"),
		table.TextColumn("reason"),
		table.TextColumn("sha256"),
	}
}

// GenerateAllowed generates data for the santa_allowed table
func GenerateAllowed(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	return generate(ctx, decisionAllowed)
}

// GenerateDenied generates data for the santa_denied table
func GenerateDenied(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	return generate(ctx, decisionDenied)
}

func generate(ctx context.Context, dec santaDecisionType) ([]map[string]string, error) {
	entries, err := scrapeSantaLog(ctx, dec)
	if err != nil {
		// Gracefully return an empty result if log cannot be scraped
		log.Debug().Err(err).Msg("failed to scrape santa log for denied entries")
		return []map[string]string{}, nil
	}

	var results []map[string]string
	for _, entry := range entries {
		row := map[string]string{
			"timestamp":   entry.Timestamp,
			"application": entry.Application,
			"reason":      entry.Reason,
			"sha256":      entry.SHA256,
		}
		results = append(results, row)
	}

	return results, nil
}

// extractValues extracts key-value pairs from a Santa log line
func extractValues(line string) map[string]string {
	values := make(map[string]string, 8)

	// Timestamp (keep your precompiled regex if you already have one)
	if m := timestampRegex.FindStringSubmatch(line); len(m) > 1 {
		values["timestamp"] = m[1]
	}

	// Parse after santad preface
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
		v = strings.TrimSpace(v)
		// Trim optional quotes
		v = strings.Trim(v, `"'`)

		if k != "" && v != "" {
			values[k] = v
		}
	}

	return values
}

// scrapeStream processes a stream of log lines and extracts relevant entries
func scrapeStream(ctx context.Context, scanner *bufio.Scanner, decision santaDecisionType) ([]logEntry, error) {
	var entries []logEntry

	for scanner.Scan() {

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line := scanner.Text()

		// Filter by decision type
		if decision == decisionAllowed {
			if !strings.Contains(line, "decision=ALLOW") {
				continue
			}
		} else if decision == decisionDenied {
			if !strings.Contains(line, "decision=DENY") {
				continue
			}
		}

		values := extractValues(line)

		if values["timestamp"] == "" {
			continue
		}

		if len(entries) >= maxEntries {
			return entries, fmt.Errorf("santa log exceeds maximum entries (%d), aborting", maxEntries)
		}

		entries = append(entries, logEntry{
			Timestamp:   values["timestamp"],
			Application: values["path"],
			Reason:      values["reason"],
			SHA256:      values["sha256"],
		})
	}

	return entries, nil
}

// scrapeCurrentLog reads the current Santa log file
func scrapeCurrentLog(ctx context.Context, path string, decision santaDecisionType) ([]logEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open Santa log file: %v", err)
	}
	defer file.Close()

	scanner := makeBufferedScanner(file)
	return scrapeStream(ctx, scanner, decision)
}

// scrapeCompressedSantaLog reads a compressed Santa log file
func scrapeCompressedSantaLog(ctx context.Context, path string, decision santaDecisionType) ([]logEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open compressed log file %s: %v", path, err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader for %s: %v", path, err)
	}
	defer gzReader.Close()

	scanner := makeBufferedScanner(gzReader)
	return scrapeStream(ctx, scanner, decision)
}

func makeBufferedScanner(r io.Reader) *bufio.Scanner {
	s := bufio.NewScanner(r)
	// buf := make([]byte, 0, 64*1024)
	// s.Buffer(buf, 1024*1024)
	return s
}

// newArchiveFileExists checks if a new archive file exists
func newArchiveFileExists(archiveIndex int, path string) bool {
	archivePath := fmt.Sprintf("%s.%d.gz", path, archiveIndex)

	_, err := os.Stat(archivePath)
	return err == nil
}

func scrapeSantaLog(ctx context.Context, decision santaDecisionType) ([]logEntry, error) {
	return scrapeSantaLogFromBase(ctx, decision, defaultLogPath)
}

// scrapeSantaLog reads all Santa log files (current and archived)
func scrapeSantaLogFromBase(ctx context.Context, decision santaDecisionType, path string) ([]logEntry, error) {
	var allEntries []logEntry

	// Read current log
	currentEntries, err := scrapeCurrentLog(ctx, path, decision)
	if err != nil {
		return nil, err
	}
	allEntries = append(allEntries, currentEntries...)

	// Read archived logs
	for i := 0; ; i++ {
		if !newArchiveFileExists(i, path) {
			break
		}

		archivePath := fmt.Sprintf("%s.%d.gz", path, i)

		archiveEntries, err := scrapeCompressedSantaLog(ctx, archivePath, decision)
		if err != nil {
			return nil, err // osquery expects full results or an error
		}

		allEntries = append(allEntries, archiveEntries...)
	}

	return allEntries, nil
}
