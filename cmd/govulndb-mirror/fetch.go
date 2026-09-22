package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

const (
	// dbArchive is the whole database in one request. The Go team publishes it for exactly this
	// purpose; 4,400 per-ID requests would be slower and far likelier to half-fail, and a
	// half-fetched database is the shape this publisher must never ship.
	dbArchive = "/vulndb.zip"

	// indexPath lists every module and the reports against it. Cross-checking the archive
	// against it is what turns a truncated mirror into a skipped run instead of a shrunken feed.
	indexPath = "index/modules.json"

	// dbIndexPath carries the database's own last-modified stamp, logged for context.
	dbIndexPath = "index/db.json"

	// reportDir holds one OSV document per report.
	reportDir = "ID/"

	// fetchAttempts is how many times a request is tried before the run is called suspect.
	fetchAttempts = 3

	// maxArchiveBytes and maxEntryBytes bound what an untrusted archive can make this process
	// allocate. The database is ~3 MB compressed and ~7 MB expanded.
	maxArchiveBytes = 128 << 20
	maxEntryBytes   = 32 << 20
)

// retryDelay is the base backoff between fetch attempts; tests set it to zero.
var retryDelay = 2 * time.Second

// osvReport is the part of a Go vulnerability database report this publisher reads.
type osvReport struct {
	ID string `json:"id"`
	// Withdrawn is set on retracted advisories. They keep their ranges, and those ranges are
	// usually "every version, no fix", so publishing one would flag every host running the
	// module.
	Withdrawn string        `json:"withdrawn"`
	Aliases   []string      `json:"aliases"`
	Affected  []osvAffected `json:"affected"`
}

type osvAffected struct {
	Package osvPackage `json:"package"`
	Ranges  []osvRange `json:"ranges"`
}

type osvPackage struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

type osvRange struct {
	Type   string     `json:"type"`
	Events []osvEvent `json:"events"`
}

// osvEvent is one bound of a range. Upstream writes each event as a single-key object, so
// exactly one of these fields is ever set; anything else is a report the transform refuses to
// guess at.
type osvEvent struct {
	Introduced string `json:"introduced"`
	Fixed      string `json:"fixed"`
}

// moduleIndex is index/modules.json: every module and the reports filed against it.
type moduleIndex []struct {
	Path  string `json:"path"`
	Vulns []struct {
		ID string `json:"id"`
	} `json:"vulns"`
}

// fetchDatabase downloads the database archive and returns every report in it, sorted by ID,
// along with the database's last-modified stamp. Every report the index names has to be present:
// a partial collection is suspect, not something to publish a smaller artifact from.
func fetchDatabase(ctx context.Context, baseURL string, logger *log.Logger) ([]osvReport, string, error) {
	// The run's context carries the overall deadline, so the client itself sets none.
	client := fleethttp.NewClient(fleethttp.WithNoTimeout())

	body, err := get(ctx, client, strings.TrimSuffix(baseURL, "/")+dbArchive, logger)
	if err != nil {
		return nil, "", err
	}

	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, "", suspectf("reading %s archive: %v", dbArchive, err)
	}

	var (
		reports  []osvReport
		index    moduleIndex
		modified struct {
			Modified string `json:"modified"`
		}
		collected = make(map[string]struct{})
	)

	for _, file := range archive.File {
		switch {
		case file.Name == indexPath:
			if err := readJSON(file, &index); err != nil {
				return nil, "", err
			}
		case file.Name == dbIndexPath:
			if err := readJSON(file, &modified); err != nil {
				return nil, "", err
			}
		case strings.HasPrefix(file.Name, reportDir) && path.Ext(file.Name) == ".json":
			var report osvReport
			if err := readJSON(file, &report); err != nil {
				return nil, "", err
			}
			if report.ID == "" {
				return nil, "", suspectf("%s has no report ID", file.Name)
			}
			reports = append(reports, report)
			collected[report.ID] = struct{}{}
		}
	}

	if index == nil {
		return nil, "", suspectf("%s archive carries no %s", dbArchive, indexPath)
	}

	var missing []string
	for _, module := range index {
		for _, vuln := range module.Vulns {
			if _, ok := collected[vuln.ID]; !ok {
				missing = append(missing, vuln.ID)
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, "", suspectf("incomplete download: %s names %d report(s) the archive does not carry, e.g. %s",
			indexPath, len(missing), strings.Join(first(missing, 5), ", "))
	}

	sort.Slice(reports, func(i, j int) bool { return reports[i].ID < reports[j].ID })

	return reports, modified.Modified, nil
}

// get fetches a URL, retrying a few times so a single dropped connection does not cost a
// publish. A request that still fails is suspect: the collection is incomplete.
func get(ctx context.Context, client *http.Client, url string, logger *log.Logger) ([]byte, error) {
	var lastErr error

	for attempt := 1; attempt <= fetchAttempts; attempt++ {
		if attempt > 1 {
			delay := time.Duration(attempt-1) * retryDelay
			logger.Printf("retrying %s in %s (attempt %d/%d): %v", url, delay, attempt, fetchAttempts, lastErr)
			select {
			case <-ctx.Done():
				return nil, suspectf("fetching %s: %v", url, ctx.Err())
			case <-time.After(delay):
			}
		}

		body, err := getOnce(ctx, client, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
	}

	return nil, suspectf("fetching %s failed after %d attempts: %v", url, fetchAttempts, lastErr)
}

func getOnce(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxArchiveBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxArchiveBytes {
		return nil, fmt.Errorf("response larger than the %d byte limit", maxArchiveBytes)
	}

	return body, nil
}

func readJSON(file *zip.File, into any) error {
	r, err := file.Open()
	if err != nil {
		return suspectf("opening %s: %v", file.Name, err)
	}
	defer r.Close()

	data, err := io.ReadAll(io.LimitReader(r, maxEntryBytes+1))
	if err != nil {
		return suspectf("reading %s: %v", file.Name, err)
	}
	if len(data) > maxEntryBytes {
		return suspectf("%s is larger than the %d byte limit", file.Name, maxEntryBytes)
	}

	if err := json.Unmarshal(data, into); err != nil {
		return suspectf("decoding %s: %v", file.Name, err)
	}

	return nil
}

func first(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
