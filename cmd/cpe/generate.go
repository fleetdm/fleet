package main

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cpedict"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
	"github.com/pandatix/nvdapi/common"
	"github.com/pandatix/nvdapi/v2"
)

const (
	httpClientTimeout       = 3 * time.Minute
	waitTimeBetweenRequests = 6 * time.Second
	waitTimeForRetry        = 10 * time.Second
	maxRetryAttempts        = 20
	apiKeyEnvVar            = "NVD_API_KEY" //nolint:gosec
)

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	ctx := context.Background()
	apiKey := os.Getenv(apiKeyEnvVar)

	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	if apiKey == "" {
		log.Fatalf("Must set %v environment variable", apiKeyEnvVar)
	}

	cwd, err := os.Getwd()
	panicIf(err)
	logger.InfoContext(ctx, fmt.Sprintf("CWD: %v", cwd))

	client := fleethttp.NewClient(fleethttp.WithTimeout(httpClientTimeout))
	dbPath := getCPEs(ctx, logger, client, apiKey, cwd)

	logger.InfoContext(ctx, fmt.Sprintf("Sqlite file %s size: %.2f MB\n", dbPath, getSizeMB(dbPath)))

	logger.InfoContext(ctx, "Compressing DB...")
	compressedPath, err := compress(ctx, logger, dbPath)
	panicIf(err)

	logger.InfoContext(ctx, "Calculating SHA256...")
	compressedPath, err = addSHA256(ctx, logger, compressedPath)
	panicIf(err)

	logger.InfoContext(ctx, fmt.Sprintf("Final compressed file %s size: %.2f MB\n", compressedPath, getSizeMB(compressedPath)))
	logger.InfoContext(ctx, "Done.")
}

func getSizeMB(path string) float64 {
	info, err := os.Stat(path)
	panicIf(err)
	return float64(info.Size()) / 1024.0 / 1024.0
}

func getCPEs(ctx context.Context, logger *slog.Logger, client common.HTTPClient, apiKey string, resultPath string) string {
	logger.InfoContext(ctx, "Fetching CPEs from NVD...")

	nvdClient, err := nvdapi.NewNVDClient(client, apiKey)
	panicIf(err)

	var cpes []cpedict.CPEItem
	retryAttempts := 0

	totalResults := 1
	for startIndex := 0; startIndex < totalResults; {
		cpeResponse, err := nvdapi.GetCPEs(nvdClient, nvdapi.GetCPEsParams{StartIndex: ptr.Int(startIndex)})
		if err != nil {
			if retryAttempts > maxRetryAttempts {
				panicIf(err)
			}
			logger.WarnContext(ctx, fmt.Sprintf("NVD request returned error:'%v' Retrying in %v", err.Error(), waitTimeForRetry.String()))
			retryAttempts++
			time.Sleep(waitTimeForRetry)
			continue
		}
		retryAttempts = 0
		totalResults = cpeResponse.TotalResults
		logger.InfoContext(ctx, fmt.Sprintf("Got %v results", cpeResponse.ResultsPerPage))
		startIndex += cpeResponse.ResultsPerPage
		for _, product := range cpeResponse.Products {
			cpes = append(cpes, convertToCPEItem(product.CPE))
		}
		if startIndex < totalResults {
			// NVD API recommendation to sleep between requests: https://nvd.nist.gov/developers/api-workflows
			time.Sleep(waitTimeBetweenRequests)
			logger.InfoContext(ctx, fmt.Sprintf("Fetching index %v out of %v", startIndex, totalResults))
		}
	}

	// Sanity check
	if totalResults <= 1 || len(cpes) != totalResults {
		log.Fatalf("Invalid number of expected results:%v or actual results:%v", totalResults, len(cpes))
	}

	logger.InfoContext(ctx, "Generating CPE sqlite DB...")

	dbPath := filepath.Join(resultPath, "cpe.sqlite")
	err = nvd.GenerateCPEDB(dbPath, cpes)
	panicIf(err)

	return dbPath
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

func compress(ctx context.Context, logger *slog.Logger, path string) (string, error) {
	compressedPath := fmt.Sprintf("%s.gz", path)
	compressedDB, err := os.Create(compressedPath)
	if err != nil {
		return "", err
	}
	defer closeFile(ctx, logger, compressedDB)

	db, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer closeFile(ctx, logger, db)

	w := gzip.NewWriter(compressedDB)
	defer func(w *gzip.Writer) {
		err := w.Close()
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Could not close gzip.Writer: %v", err.Error()))
		}
	}(w)

	_, err = io.Copy(w, db)
	if err != nil {
		return "", err
	}
	return compressedPath, nil
}

// addSHA256 adds the file's SHA256 checksum to its name
func addSHA256(ctx context.Context, logger *slog.Logger, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer closeFile(ctx, logger, file)

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

func closeFile(ctx context.Context, logger *slog.Logger, file *os.File) {
	err := file.Close()
	if err != nil {
		logger.ErrorContext(ctx, fmt.Sprintf("Could not close file %v: %v", file.Name(), err.Error()))
	}
}
