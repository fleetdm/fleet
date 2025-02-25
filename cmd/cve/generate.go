package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	nvdsync "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/sync"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
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

func main() {
	dbDir := flag.String("db_dir", "/tmp/vulndbs", "Path to the vulnerability database")
	debug := flag.Bool("debug", false, "Sets debug mode")
	flag.Parse()

	logger := log.NewJSONLogger(os.Stdout)
	if *debug {
		logger = level.NewFilter(logger, level.AllowDebug())
	} else {
		logger = level.NewFilter(logger, level.AllowInfo())
	}

	if err := os.MkdirAll(*dbDir, os.ModePerm); err != nil {
		panic(err)
	}

	if os.Getenv(cleanEnvVar) == "false" {
		logger.Log("msg", "Downloading latest release")
		maxRetries := 3
		for i := 0; i < maxRetries; i++ {
			err := downloadLatestRelease(*dbDir, *debug, logger)
			if err == nil {
				break
			}

			if i == maxRetries-1 {
				logger.Log("msg", "Failed to download latest release. Continuing with full NVD Sync", "err", err)
				break
			}

			logger.Log("msg", "Failed to download latest release. Retrying in 30 seconds", "err", err)
			time.Sleep(30 * time.Second)
		}
	}

	// Sync the CVE files
	if err := nvd.GenerateCVEFeeds(*dbDir, *debug, logger); err != nil {
		panic(err)
	}

	// Remove Vulncheck archive
	if err := os.RemoveAll(filepath.Join(*dbDir, "vulncheck.zip")); err != nil {
		logger.Log("msg", "Failed to remove vulncheck.zip", "err", err)
	}

	// Read in every cpe file and create a corresponding metadata file
	// nvd data feeds start in 2002
	logger.Log("msg", "Generating metadata files ...")
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
			logger.Log("msg", "Skipping metadata generation for missing file", "file", fileNameRaw)
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

func downloadLatestRelease(dbDir string, debug bool, logger log.Logger) error {
	// Download the latest release
	err := nvd.DownloadCVEFeed(dbDir, "", debug, logger)
	if err != nil {
		return fmt.Errorf("download cve feed: %w", err)
	}

	// gunzip json files
	files, err := filepath.Glob(filepath.Join(dbDir, "nvdcve-1.1-*.json.gz"))
	if err != nil {
		return fmt.Errorf("glob json files: %w", err)
	}
	for _, file := range files {
		err = gunzipFileToDisk(file, dbDir)
		if err != nil {
			return fmt.Errorf("gunzip file %s to disk: %w", file, err)
		}
	}

	// Download the last mod start date
	err = downloadLatestGitHubAsset(dbDir, "last_mod_start_date.txt")
	if err != nil {
		return fmt.Errorf("downloading last_mod_start_date asset: %w", err)
	}

	return nil
}

// downloadAsset downloads the asset from the latest release and writes it to a file
func downloadLatestGitHubAsset(dbDir, fileName string) error {
	assetPath, err := nvd.GetGitHubCVEAssetPath()
	if err != nil {
		return fmt.Errorf("get github cve asset path: %w", err)
	}

	client := fleethttp.NewClient()
	resp, err := client.Get(assetPath + fileName)
	if err != nil {
		return fmt.Errorf("get last mod start date: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get last mod start date: %w", fmt.Errorf("unexpected status code %d", resp.StatusCode))
	}

	lastModStartDate, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read last mod start date: %w", err)
	}

	// Write the last mod start date to a file
	lastModStartDateFile := filepath.Join(dbDir, fileName)
	err = os.WriteFile(lastModStartDateFile, lastModStartDate, 0o644)
	if err != nil {
		return fmt.Errorf("write last mod start date: %w", err)
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

func gunzipFileToDisk(filename, dbpath string) error {
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

	filepath := filepath.Join(dbpath, strings.TrimSuffix(filepath.Base(filename), ".gz"))

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	// Using a maxBytes limit to prevent decompression bombs: gosec G110
	maxBytes := 200 * 1024 * 1024 // 200MB
	_, err = io.CopyN(out, gz, int64(maxBytes))
	if err != nil && err != io.EOF {
		msg := fmt.Sprintf("error copying file %s: %v", f.Name(), err)
		panic(msg)
	}

	return nil
}
