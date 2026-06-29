package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cpedict"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
	"github.com/pandatix/nvdapi/common"
	"github.com/pandatix/nvdapi/v2"
)

const (
	// cpeSourceEnvVar selects where CPEs are fetched from: "feed" (default) or "api".
	//
	// The data feed is the default because it is a single, fast, consistent download.
	// However, NVD has officially deprecated the data feeds in favor of the API, so the
	// "api" source is kept as a drop-in fallback: if NVD removes the feed, set
	// CPE_SOURCE=api (and NVD_API_KEY) in the generator's environment to revert to the
	// API method without a code change.
	cpeSourceEnvVar = "CPE_SOURCE"
	cpeSourceFeed   = "feed"
	cpeSourceAPI    = "api"

	// apiKeyEnvVar holds the NVD API key, required only when using the "api" source.
	apiKeyEnvVar = "NVD_API_KEY" //nolint:gosec

	// cpeFeedURL is the NVD CPE Dictionary data feed, a single gzipped tar archive
	// containing the full dictionary in the 2.0 JSON schema. Downloading the feed in
	// one request avoids paginating ~1.7M entries through the rate-limited NVD API,
	// which is slow (>1h) and prone to 503s.
	// See https://nvd.nist.gov/vuln/data-feeds (CPE Dictionary feed).
	cpeFeedURL = "https://nvd.nist.gov/feeds/json/cpe/2.0/nvdcpe-2.0.tar.gz"
	// userAgent is set on the request because NVD's CDN rejects requests without one.
	userAgent = "fleetdm-cpe-generator"

	httpClientTimeout = 10 * time.Minute
	maxRetryAttempts  = 20

	// waitTimeBetweenRequests is the NVD-recommended pause between API requests, used
	// only by the "api" source: https://nvd.nist.gov/developers/api-workflows
	waitTimeBetweenRequests = 6 * time.Second

	// minCompleteFraction is the minimum percentage of NVD's reported totalResults that
	// must be decoded for the result to be accepted. It tolerates NVD's small, persistent
	// overcount (~0.025%) while still rejecting a result missing whole chunks/pages (the
	// smallest single feed chunk is >1% of the dictionary).
	minCompleteFraction = 95
)

// waitTimeForRetry is the delay between download retries. It is a var so tests can
// shorten it.
var waitTimeForRetry = 10 * time.Second

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(logHandler))

	cwd, err := os.Getwd()
	panicIf(err)
	slog.Info(fmt.Sprintf("CWD: %v", cwd))

	client := fleethttp.NewClient(fleethttp.WithTimeout(httpClientTimeout))
	dbPath := getCPEs(client, cwd)

	slog.Info(fmt.Sprintf("Sqlite file %s size: %.2f MB\n", dbPath, getSizeMB(dbPath)))

	slog.Info("Compressing DB...")
	compressedPath, err := compress(dbPath)
	panicIf(err)

	slog.Info("Calculating SHA256...")
	compressedPath, err = addSHA256(compressedPath)
	panicIf(err)

	slog.Info(fmt.Sprintf("Final compressed file %s size: %.2f MB\n", compressedPath, getSizeMB(compressedPath)))
	slog.Info("Done.")
}

func getSizeMB(path string) float64 {
	info, err := os.Stat(path)
	panicIf(err)
	return float64(info.Size()) / 1024.0 / 1024.0
}

func getCPEs(client common.HTTPClient, resultPath string) string {
	source := strings.ToLower(strings.TrimSpace(os.Getenv(cpeSourceEnvVar)))
	if source == "" {
		source = cpeSourceFeed
	}

	var (
		cpes []cpedict.CPEItem
		err  error
	)
	switch source {
	case cpeSourceFeed:
		slog.Info("Fetching CPE dictionary feed from NVD...")
		cpes, err = fetchCPEFeed(client)
	case cpeSourceAPI:
		slog.Info("Fetching CPEs from the NVD API...")
		cpes, err = fetchCPEsFromAPI(client, os.Getenv(apiKeyEnvVar))
	default:
		panicIf(fmt.Errorf("invalid %s value %q (want %q or %q)", cpeSourceEnvVar, source, cpeSourceFeed, cpeSourceAPI))
	}
	panicIf(err)

	slog.Info(fmt.Sprintf("Got %v CPEs", len(cpes))) //nolint:gosec // G706 false positive: logs an integer count, not a tainted string
	slog.Info("Generating CPE sqlite DB...")

	dbPath := filepath.Join(resultPath, "cpe.sqlite")
	panicIf(nvd.GenerateCPEDB(dbPath, cpes))

	return dbPath
}

// fetchCPEsFromAPI paginates the entire CPE dictionary through the NVD API. It is the
// fallback for fetchCPEFeed, kept so the generator can switch sources (CPE_SOURCE=api)
// if NVD removes the deprecated data feed. It is slower (~1h) and more rate-limited
// than the feed.
func fetchCPEsFromAPI(client common.HTTPClient, apiKey string) ([]cpedict.CPEItem, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("%s must be set to use the %q source", apiKeyEnvVar, cpeSourceAPI)
	}

	nvdClient, err := nvdapi.NewNVDClient(client, apiKey)
	if err != nil {
		return nil, fmt.Errorf("creating NVD client: %w", err)
	}

	var (
		cpes          []cpedict.CPEItem
		totalResults  = 1
		retryAttempts int
	)
	for startIndex := 0; startIndex < totalResults; {
		cpeResponse, err := nvdapi.GetCPEs(nvdClient, nvdapi.GetCPEsParams{StartIndex: new(startIndex)})
		if err != nil {
			if retryAttempts >= maxRetryAttempts {
				return nil, fmt.Errorf("fetching CPEs from NVD API: %w", err)
			}
			slog.Warn(fmt.Sprintf("NVD request returned error:'%v' Retrying in %v", err.Error(), waitTimeForRetry.String()))
			retryAttempts++
			time.Sleep(waitTimeForRetry)
			continue
		}
		retryAttempts = 0
		totalResults = cpeResponse.TotalResults
		slog.Info(fmt.Sprintf("Got %v results", cpeResponse.ResultsPerPage))
		startIndex += cpeResponse.ResultsPerPage
		for _, product := range cpeResponse.Products {
			cpes = append(cpes, convertToCPEItem(product.CPE))
		}
		if startIndex < totalResults {
			// NVD API recommendation to sleep between requests:
			// https://nvd.nist.gov/developers/api-workflows
			time.Sleep(waitTimeBetweenRequests)
			slog.Info(fmt.Sprintf("Fetching index %v out of %v", startIndex, totalResults))
		}
	}

	if err := validateCPECount(cpes, totalResults); err != nil {
		return nil, err
	}
	return cpes, nil
}

// validateCPECount accepts the decoded CPEs unless the result is empty or is missing a
// meaningful fraction of the entries NVD reports. It is shared by both the feed and API
// sources.
//
// It deliberately does NOT require len(cpes) == totalResults: NVD's reported
// totalResults is consistently a few hundred higher than the number of products it
// actually serializes (e.g. 1761245 reported vs 1760806 delivered, in both the API and
// the feed). Requiring an exact match is what made the previous generator fail. A small
// gap is logged; a large shortfall (missing chunks/pages) is an error.
func validateCPECount(cpes []cpedict.CPEItem, totalResults int) error {
	if len(cpes) == 0 || totalResults <= 1 {
		return fmt.Errorf("empty result: decoded %d CPEs, totalResults %d", len(cpes), totalResults)
	}
	if len(cpes) < totalResults*minCompleteFraction/100 {
		return fmt.Errorf("incomplete result: decoded %d CPEs, well below reported total %d", len(cpes), totalResults)
	}
	if len(cpes) != totalResults {
		//nolint:gosec // G706 false positive: logs integer counts, not tainted strings
		slog.Warn(fmt.Sprintf("Decoded %d CPEs but NVD reports totalResults=%d (count discrepancy of %d)", len(cpes), totalResults, totalResults-len(cpes)))
	}
	return nil
}

// fetchCPEFeed downloads the NVD CPE Dictionary feed to a temporary file and decodes
// it into CPE items.
//
// The feed is a gzipped tar archive of one or more JSON chunk files
// (nvdcpe-2.0-chunks/nvdcpe-2.0-chunk-NNNNN.json). Each chunk is a complete CPE 2.0
// response that reports the full totalResults but contains only a slice of the
// products, so every chunk must be decoded and accumulated.
func fetchCPEFeed(client common.HTTPClient) ([]cpedict.CPEItem, error) {
	tmp, err := os.CreateTemp("", "nvdcpe-*.tar.gz")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer func() {
		closeFile(tmp)
		if err := os.Remove(tmp.Name()); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn(fmt.Sprintf("Could not remove temp file %v: %v", tmp.Name(), err.Error()))
		}
	}()

	if err := downloadFeed(client, tmp); err != nil {
		return nil, err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewinding feed file: %w", err)
	}
	return decodeCPEArchive(tmp)
}

// downloadFeed downloads the CPE feed into dst. NVD's CDN frequently resets long
// downloads, so on a transfer error this resumes from the current byte offset with a
// Range request instead of starting over, retrying up to maxRetryAttempts times. The
// same loop also retries the initial request on transient failures (e.g. 503).
func downloadFeed(client common.HTTPClient, dst *os.File) error {
	var offset int64
	for attempt := 0; ; attempt++ {
		n, err := downloadFeedOnce(client, dst, &offset)
		offset += n
		if err == nil {
			slog.Info(fmt.Sprintf("Downloaded CPE feed (%d bytes)", offset))
			return nil
		}
		if attempt >= maxRetryAttempts {
			return fmt.Errorf("downloading CPE feed (got %d bytes after %d attempts): %w", offset, attempt, err)
		}
		slog.Warn(fmt.Sprintf("CPE feed download interrupted after %d bytes:'%v' Resuming in %v", offset, err.Error(), waitTimeForRetry.String()))
		time.Sleep(waitTimeForRetry)
	}
}

// downloadFeedOnce performs a single (possibly ranged) request, copying the body to
// dst and returning the number of bytes written. It updates *offset to 0 if the
// server ignores the Range header and the file must be rewritten from scratch.
func downloadFeedOnce(client common.HTTPClient, dst *os.File, offset *int64) (int64, error) {
	req, err := http.NewRequest(http.MethodGet, cpeFeedURL, nil)
	if err != nil {
		return 0, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	if *offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", *offset))
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	switch {
	case *offset > 0 && resp.StatusCode == http.StatusOK:
		// Server ignored the Range header; rewrite the file from the beginning.
		if err := dst.Truncate(0); err != nil {
			return 0, err
		}
		if _, err := dst.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		*offset = 0
	case *offset > 0 && resp.StatusCode != http.StatusPartialContent:
		return 0, fmt.Errorf("unexpected status %d resuming CPE feed", resp.StatusCode)
	case *offset == 0 && resp.StatusCode != http.StatusOK:
		return 0, fmt.Errorf("unexpected status %d fetching CPE feed", resp.StatusCode)
	}

	return io.Copy(dst, resp.Body)
}

// decodeCPEArchive decodes a gzipped tar CPE feed archive into CPE items, accumulating
// every JSON chunk and verifying the result is complete.
func decodeCPEArchive(r io.Reader) ([]cpedict.CPEItem, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var (
		cpes         []cpedict.CPEItem
		totalResults int
		chunks       int
	)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar archive: %w", err)
		}
		if hdr.FileInfo().IsDir() || !strings.HasSuffix(hdr.Name, ".json") {
			continue
		}
		chunkCPEs, chunkTotal, err := decodeCPEFeed(tr)
		if err != nil {
			return nil, fmt.Errorf("decoding %s: %w", hdr.Name, err)
		}
		cpes = append(cpes, chunkCPEs...)
		if chunkTotal > 0 {
			totalResults = chunkTotal
		}
		chunks++
		slog.Info(fmt.Sprintf("Decoded %s: %d CPEs (%d/%d total)", hdr.Name, len(chunkCPEs), len(cpes), totalResults))
	}

	if chunks == 0 {
		return nil, errors.New("no JSON chunk files found in CPE feed archive")
	}

	// A truncated download is already caught by the gzip/tar layer above (it fails
	// before reaching here), so the count check guards against a structurally valid
	// archive that is nonetheless missing whole chunks.
	if err := validateCPECount(cpes, totalResults); err != nil {
		return nil, err
	}

	return cpes, nil
}

// decodeCPEFeed streams the CPE Dictionary 2.0 JSON document, converting each product
// as it is read so the entire raw response is never held in memory at once. It returns
// the converted items and the totalResults field from the document.
func decodeCPEFeed(r io.Reader) ([]cpedict.CPEItem, int, error) {
	dec := json.NewDecoder(r)

	tok, err := dec.Token()
	if err != nil {
		return nil, 0, fmt.Errorf("reading opening token: %w", err)
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, 0, fmt.Errorf("expected JSON object, got %v", tok)
	}

	var (
		cpes         []cpedict.CPEItem
		totalResults int
	)
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, 0, fmt.Errorf("reading object key: %w", err)
		}
		key, _ := keyTok.(string)

		switch key {
		case "totalResults":
			if err := dec.Decode(&totalResults); err != nil {
				return nil, 0, fmt.Errorf("decoding totalResults: %w", err)
			}
		case "products":
			openTok, err := dec.Token()
			if err != nil {
				return nil, 0, fmt.Errorf("reading products opening token: %w", err)
			}
			if delim, ok := openTok.(json.Delim); !ok || delim != '[' {
				return nil, 0, fmt.Errorf("expected products array, got %v", openTok)
			}
			for dec.More() {
				var product nvdapi.CPEProduct
				if err := dec.Decode(&product); err != nil {
					return nil, 0, fmt.Errorf("decoding product: %w", err)
				}
				cpes = append(cpes, convertToCPEItem(product.CPE))
			}
			if _, err := dec.Token(); err != nil { // closing ']'
				return nil, 0, fmt.Errorf("reading products closing token: %w", err)
			}
		default:
			// Skip values we don't need (resultsPerPage, startIndex, format, ...).
			var skip json.RawMessage
			if err := dec.Decode(&skip); err != nil {
				return nil, 0, fmt.Errorf("skipping field %q: %w", key, err)
			}
		}
	}

	return cpes, totalResults, nil
}

func convertToCPEItem(in nvdapi.CPE) (out cpedict.CPEItem) {
	out = cpedict.CPEItem{}

	// CPE name
	wfName, err := wfn.Parse(in.CPEName)
	panicIf(err)
	out.CPE23 = cpedict.CPE23Item{
		Name: cpedict.NamePattern(*wfName),
	}

	// Deprecations
	out.Deprecated = in.Deprecated
	if in.Deprecated {
		out.CPE23.Deprecation = &cpedict.Deprecation{}
		for _, item := range in.DeprecatedBy {
			deprecatorName, err := wfn.Parse(*item.CPEName)
			panicIf(err)
			deprecatorInfo := cpedict.DeprecatedInfo{
				Name: cpedict.NamePattern(*deprecatorName),
			}
			out.CPE23.Deprecation.DeprecatedBy = append(out.CPE23.Deprecation.DeprecatedBy, deprecatorInfo)
		}
	}

	// Title
	out.Title = cpedict.TextType{}
	for _, title := range in.Titles {
		// only using English language
		if title.Lang == "en" {
			out.Title["en-US"] = title.Title
			break
		}
	}

	// The following fields are not needed by subsequent code:
	// out.DeprecatedBy
	// out.DeprecationDate
	// out.Notes
	// out.References
	return out
}

func compress(path string) (string, error) {
	compressedPath := fmt.Sprintf("%s.gz", path)
	compressedDB, err := os.Create(compressedPath)
	if err != nil {
		return "", err
	}
	defer closeFile(compressedDB)

	db, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer closeFile(db)

	w := gzip.NewWriter(compressedDB)
	defer func(w *gzip.Writer) {
		err := w.Close()
		if err != nil {
			slog.Error(fmt.Sprintf("Could not close gzip.Writer: %v", err.Error()))
		}
	}(w)

	_, err = io.Copy(w, db)
	if err != nil {
		return "", err
	}
	return compressedPath, nil
}

// addSHA256 adds the file's SHA256 checksum to its name
func addSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer closeFile(file)

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	newPath, err := replaceLast(path, "cpe.sqlite.gz", fmt.Sprintf("cpe-%x.sqlite.gz", hash.Sum(nil)))
	if err != nil {
		return "", err
	}

	err = os.Rename(path, newPath)
	return newPath, err
}

// replaceLast replaces the last occurrence of string
func replaceLast(s, oldVal, newVal string) (string, error) {
	i := strings.LastIndex(s, oldVal)
	if i == -1 {
		return "", fmt.Errorf("substring:%v not found in string:%v", oldVal, s)
	}
	return s[:i] + newVal + s[i+len(oldVal):], nil
}

func closeFile(file *os.File) {
	err := file.Close()
	if err != nil {
		slog.Error(fmt.Sprintf("Could not close file %v: %v", file.Name(), err.Error()))
	}
}
