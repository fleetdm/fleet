package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

	// Sync the CVE files
	if err := nvd.GenerateCVEFeeds(*dbDir, *debug, logger); err != nil {
		panic(err)
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
