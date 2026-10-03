package software_update

import (
	"context"
	"fmt"
	"maps"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/singleflight"
)

const (
	// softwareupdate scans Apple's catalog over the network; about 10s on a healthy link.
	scanTimeout = 30 * time.Second
	// Policies referencing this table run back to back in one distributed batch and
	// each would otherwise pay for its own scan.
	cacheTTL = 5 * time.Minute

	labelPrefix = "* Label:"
	noUpdates   = "No new software available"
)

// Detail fields are comma separated "Key: value" pairs; anchoring on the key keeps
// commas inside values (titles) from splitting a field.
var fieldKeyRe = regexp.MustCompile(`(?:^|, )([A-Za-z]+): `)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.IntegerColumn("software_update_required"),
		table.TextColumn("label"),
		table.TextColumn("title"),
		table.TextColumn("version"),
		table.IntegerColumn("size_kib"),
		table.IntegerColumn("recommended"),
		table.TextColumn("action"),
	}
}

// Generate is called to return the results for the table at query time.
func Generate(ctx context.Context, _ table.QueryContext) ([]map[string]string, error) {
	return defaultLister.rows(ctx)
}

var defaultLister = newLister(runSoftwareUpdate)

type lister struct {
	run func(ctx context.Context) (string, error)
	now func() time.Time

	group    singleflight.Group
	mu       sync.Mutex
	cached   []map[string]string
	cachedAt time.Time
}

func newLister(run func(ctx context.Context) (string, error)) *lister {
	return &lister{run: run, now: time.Now}
}

func (l *lister) rows(ctx context.Context) ([]map[string]string, error) {
	if rows, ok := l.fresh(); ok {
		return rows, nil
	}
	// Concurrent scans interfere with each other and report different update sets, so
	// callers share one in-flight scan. It runs on its own deadline so a canceled query
	// doesn't abort the scan for the others waiting on it.
	ch := l.group.DoChan("scan", func() (any, error) {
		scanCtx, cancel := context.WithTimeout(context.Background(), scanTimeout)
		defer cancel()
		out, err := l.run(scanCtx)
		if err != nil {
			return nil, err
		}
		rows := parseList(out)
		l.mu.Lock()
		l.cached, l.cachedAt = rows, l.now()
		l.mu.Unlock()
		return rows, nil
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		return copyRows(res.Val.([]map[string]string)), nil
	}
}

func (l *lister) fresh() ([]map[string]string, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cached == nil || l.now().Sub(l.cachedAt) > cacheTTL {
		return nil, false
	}
	return copyRows(l.cached), true
}

func runSoftwareUpdate(ctx context.Context) (string, error) {
	// CombinedOutput: some Intel Macs report the result on stderr.
	out, err := exec.CommandContext(ctx, "/usr/sbin/softwareupdate", "-l").CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return "", fmt.Errorf("softwareupdate -l: %w: %s", err, truncate(msg, 500))
		}
		return "", fmt.Errorf("softwareupdate -l: %w", err)
	}
	return string(out), nil
}

// parseList turns `softwareupdate -l` output into table rows: one per available update,
// or a single row with software_update_required=0 when the tool reports nothing pending.
// Output matching neither shape yields software_update_required=1 with empty details: a
// compliance check must never read "nothing pending" from output it can't interpret.
func parseList(out string) []map[string]string {
	var rows []map[string]string
	var pending map[string]string
	flush := func() {
		if pending != nil {
			rows = append(rows, pending)
			pending = nil
		}
	}
	for line := range strings.SplitSeq(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, labelPrefix):
			flush()
			pending = newRow("1")
			pending["label"] = strings.TrimSpace(strings.TrimPrefix(line, labelPrefix))
		case pending != nil && strings.HasPrefix(line, "Title:"):
			for key, value := range parseFields(line) {
				switch key {
				case "Title":
					pending["title"] = value
				case "Version":
					pending["version"] = value
				case "Size":
					pending["size_kib"] = parseSizeKiB(value)
				case "Recommended":
					pending["recommended"] = parseYesNo(value)
				case "Action":
					pending["action"] = value
				}
			}
			flush()
		}
	}
	flush()

	if len(rows) > 0 {
		return rows
	}
	if strings.Contains(out, noUpdates) {
		return []map[string]string{newRow("0")}
	}
	log.Warn().Msg("software_update: unrecognized softwareupdate output, reporting an update as required")
	return []map[string]string{newRow("1")}
}

func parseFields(line string) map[string]string {
	fields := map[string]string{}
	matches := fieldKeyRe.FindAllStringSubmatchIndex(line, -1)
	for i, m := range matches {
		end := len(line)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		value := strings.TrimSpace(line[m[1]:end])
		fields[line[m[2]:m[3]]] = strings.TrimSpace(strings.TrimSuffix(value, ","))
	}
	return fields
}

// parseSizeKiB keeps the numeric KiB value softwareupdate reports (e.g. "249465KiB").
// Anything else is left empty rather than guessing at units.
func parseSizeKiB(v string) string {
	n, ok := strings.CutSuffix(v, "KiB")
	if !ok {
		return ""
	}
	if _, err := strconv.ParseUint(n, 10, 64); err != nil {
		return ""
	}
	return n
}

func parseYesNo(v string) string {
	switch strings.ToUpper(v) {
	case "YES":
		return "1"
	case "NO":
		return "0"
	}
	return ""
}

func newRow(required string) map[string]string {
	return map[string]string{
		"software_update_required": required,
		"label":                    "",
		"title":                    "",
		"version":                  "",
		"size_kib":                 "",
		"recommended":              "",
		"action":                   "",
	}
}

func copyRows(rows []map[string]string) []map[string]string {
	out := make([]map[string]string, len(rows))
	for i, row := range rows {
		out[i] = maps.Clone(row)
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
