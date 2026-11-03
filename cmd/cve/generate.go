package main

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	nvdsync "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/sync"
	"github.com/google/go-github/v37/github"
)

const emptyData = `{
  "CVE_data_type" : "CVE",
  "CVE_data_format" : "MITRE",
  "CVE_data_version" : "4.0",
  "CVE_data_numberOfCVEs" : "859",
  "CVE_data_timestamp" : "2023-11-17T19:00Z",
  "CVE_Items" : [ ]
}`

var cleanEnvVar = "VULNERABILITIES_CLEAN"

const (
	// seedRunTimeout bounds an incremental run. Runs are scheduled every 30 minutes
	// and queue behind each other, so a run that outlives the schedule makes the
	// next one wait and drops the ones queued after it. A healthy incremental run
	// takes a few minutes; failing well before the next slot lets it take over.
	seedRunTimeout = 25 * time.Minute

	seedMaxAttempts   = 3
	seedRetryInterval = 30 * time.Second
	// maxRateLimitWait bounds how long to wait for GitHub to lift a rate limit.
	// It must leave room for the sync itself within seedRunTimeout.
	maxRateLimitWait = 10 * time.Minute
	// rateLimitResetBuffer absorbs clock skew between the runner and GitHub.
	rateLimitResetBuffer = 5 * time.Second

	// maxDecompressedBytes caps gunzip output to prevent decompression bombs (gosec G110).
	maxDecompressedBytes int64 = 400 * 1024 * 1024
)

func main() {
	dbDir := flag.String("db_dir", "/tmp/vulndbs", "Path to the vulnerability database")
	debug := flag.Bool("debug", false, "Sets debug mode")
	flag.Parse()

	seed := os.Getenv(cleanEnvVar) == "false"

	ctx := context.Background()
	// A full sync takes much longer and is bounded only by the CI job timeout.
	if seed {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, seedRunTimeout)
		defer cancel()
	}

	logLevel := slog.LevelInfo
	if *debug {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	if err := os.MkdirAll(*dbDir, os.ModePerm); err != nil {
		panic(err)
	}

	if seed {
		logger.InfoContext(ctx, "Downloading latest release")
		// Extracted as callback to make this testable ....
		download := func() error { return downloadLatestRelease(ctx, *dbDir, *debug, logger) }
		if err := seedFromLatestRelease(ctx, logger, download); err != nil {
			panic(fmt.Errorf("download latest release (set %s=true to run a full NVD sync instead): %w", cleanEnvVar, err))
		}
	}

	// Sync the CVE files
	if err := nvd.GenerateCVEFeeds(ctx, *dbDir, *debug, logger); err != nil {
		panic(err)
	}

	// Remove Vulncheck archive
	if err := os.RemoveAll(filepath.Join(*dbDir, "vulncheck.zip")); err != nil {
		logger.WarnContext(ctx, "Failed to remove vulncheck.zip", "err", err)
	}

	// Read in every cpe file and create a corresponding metadata file
	// nvd data feeds start in 2002
	logger.InfoContext(ctx, "Generating metadata files ...")
	const startingYear = 2002
	currentYear := time.Now().Year()
	if currentYear < startingYear {
		panic("system date is in the past, cannot continue")
	}
	entries := (currentYear - startingYear) + 1
	for i := 0; i < entries; i++ {
		year := startingYear + i
		suffix := strconv.Itoa(year)
		fileNameRaw := filepath.Join(*dbDir, fileFmt(suffix, "json", ""))
		fileName := filepath.Join(*dbDir, fileFmt(suffix, "json", "gz"))
		metaName := filepath.Join(*dbDir, fileFmt(suffix, "meta", ""))
		// skip if file does not exist
		if _, err := os.Stat(fileNameRaw); os.IsNotExist(err) {
			logger.InfoContext(ctx, "Skipping metadata generation for missing file", "file", fileNameRaw)
			continue
		}
		err := nvdsync.CompressFile(fileNameRaw, fileName)
		if err != nil {
			panic(err)
		}
		createMetadata(fileName, metaName)
	}

	// Create modified and recent files
	createEmptyFiles(*dbDir, "modified")
	createEmptyFiles(*dbDir, "recent")
}

// seedFromLatestRelease runs download with retries. When GitHub reports a rate
// limit, it waits as long as GitHub asks instead of the fixed retry interval,
// or fails immediately if that is longer than maxRateLimitWait.
func seedFromLatestRelease(
	ctx context.Context,
	logger *slog.Logger,
	download func() error,
) error {
	var err error
	for attempt := 1; attempt <= seedMaxAttempts; attempt++ {
		if err = download(); err == nil {
			return nil
		}
		if attempt == seedMaxAttempts {
			break
		}

		wait := seedRetryInterval
		if retryAfter, ok := githubRetryAfter(err); ok {
			if retryAfter > maxRateLimitWait {
				return fmt.Errorf("github rate limit resets in %s, more than the %s cap: %w", retryAfter.Round(time.Second), maxRateLimitWait, err)
			}
			wait = retryAfter + rateLimitResetBuffer
		}

		logger.WarnContext(ctx, "Failed to download latest release. Retrying", "err", err, "attempt", attempt, "retry_in", wait.String())
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting to retry: %w", ctx.Err())
		case <-time.After(wait):
		}
	}
	return err
}

// githubRetryAfter returns how long GitHub asked us to wait, for both primary
// and secondary rate limits.
func githubRetryAfter(err error) (time.Duration, bool) {
	if rlErr, ok := errors.AsType[*github.RateLimitError](err); ok {
		return max(time.Until(rlErr.Rate.Reset.Time), 0), true
	}
	if abuseErr, ok := errors.AsType[*github.AbuseRateLimitError](err); ok && abuseErr.RetryAfter != nil {
		return max(*abuseErr.RetryAfter, 0), true
	}
	return 0, false
}

func downloadLatestRelease(ctx context.Context, dbDir string, debug bool, logger *slog.Logger) error {
	// Resolve the release once so the feeds and last_mod_start_date.txt come from
	// the same release even if a newer one is published mid-download.
	assetPath, err := nvd.GetGitHubCVEAssetPath(ctx)
	if err != nil {
		return fmt.Errorf("get github cve asset path: %w", err)
	}

	// Download the latest release
	err = nvd.DownloadCVEFeed(dbDir, assetPath, debug, logger)
	if err != nil {
		return fmt.Errorf("download cve feed: %w", err)
	}

	// gunzip json files
	files, err := filepath.Glob(filepath.Join(dbDir, "nvdcve-1.1-*.json.gz"))
	if err != nil {
		return fmt.Errorf("glob json files: %w", err)
	}
	for _, file := range files {
		err = gunzipFileToDisk(file, dbDir, maxDecompressedBytes)
		if err != nil {
			return fmt.Errorf("gunzip file %s to disk: %w", file, err)
		}
	}

	// Download the last mod start date
	err = downloadReleaseAsset(ctx, dbDir, assetPath, "last_mod_start_date.txt")
	if err != nil {
		return fmt.Errorf("downloading last_mod_start_date asset: %w", err)
	}

	return nil
}

// downloadReleaseAsset downloads fileName from the release at assetPath into dbDir.
func downloadReleaseAsset(ctx context.Context, dbDir, assetPath, fileName string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetPath+fileName, nil)
	if err != nil {
		return fmt.Errorf("create %s request: %w", fileName, err)
	}
	client := fleethttp.NewClient(fleethttp.WithNoTimeout())
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("get %s: %w", fileName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get %s: unexpected status code %d", fileName, resp.StatusCode)
	}

	contents, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read %s: %w", fileName, err)
	}

	if err := os.WriteFile(filepath.Join(dbDir, fileName), contents, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", fileName, err)
	}

	return nil
}

func createMetadata(fileName string, metaName string) {
	fileInfo, err := os.Stat(fileName)
	if err != nil {
		panic(err)
	}
	hash, err := gunzipFileAndComputeSHA256(fileName)
	if err != nil {
		panic(err)
	}
	metaFile, err := os.Create(metaName)
	if err != nil {
		panic(err)
	}
	defer metaFile.Close()
	if _, err = metaFile.WriteString(fmt.Sprintf("gzSize:%v\r\n", fileInfo.Size())); err != nil {
		panic(err)
	}
	if _, err = metaFile.WriteString(fmt.Sprintf("sha256:%v\r\n", hash)); err != nil {
		panic(err)
	}
}

func createEmptyFiles(baseDir, suffix string) {
	fileName := filepath.Join(baseDir, fileFmt(suffix, "json", "gz"))
	metaName := filepath.Join(baseDir, fileFmt(suffix, "meta", ""))
	dataFile, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	writer := gzip.NewWriter(dataFile)
	if _, err = writer.Write([]byte(emptyData)); err != nil {
		panic(err)
	}
	if err = writer.Close(); err != nil {
		panic(err)
	}
	dataFile.Close()
	createMetadata(fileName, metaName)
}

func fileFmt(suffix, encoding, compression string) string {
	const version = "1.1"
	s := fmt.Sprintf("nvdcve-%s-%s.%s", version, suffix, encoding)
	if compression != "" {
		s += "." + compression
	}
	return s
}

func computeSHA256(r io.Reader) (string, error) {
	hashImpl := sha256.New()
	_, err := io.Copy(hashImpl, r)
	if err != nil {
		return "", err
	}
	hash := hashImpl.Sum(nil)
	return strings.ToUpper(hex.EncodeToString(hash)), nil
}

func gunzipAndComputeSHA256(r io.Reader) (string, error) {
	f, err := gzip.NewReader(r)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return computeSHA256(f)
}

func gunzipFileAndComputeSHA256(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return gunzipAndComputeSHA256(f)
}

func gunzipFileToDisk(filename, dbpath string, maxBytes int64) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("new gzip reader: %w", err)
	}
	defer gz.Close()

	outPath := filepath.Join(dbpath, strings.TrimSuffix(filepath.Base(filename), ".gz"))

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	// Copy one byte past the limit so that reaching it is distinguishable from a
	// complete copy: io.CopyN reports no error when it copies exactly n bytes.
	written, err := io.CopyN(out, gz, maxBytes+1)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("copy file %s: %w", filename, err)
	}
	if written > maxBytes {
		// A truncated feed parses as a valid, silently incomplete file several
		// stages later, so it must not survive this failure.
		if rmErr := os.Remove(outPath); rmErr != nil {
			return fmt.Errorf("remove truncated file %s: %w", outPath, rmErr)
		}
		return fmt.Errorf("decompressed %s exceeds max size of %d bytes", filename, maxBytes)
	}

	return nil
}
