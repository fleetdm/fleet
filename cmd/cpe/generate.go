package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/facebookincubator/nvdtools/cpedict"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
)

func panicif(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Sets verbose mode")
	flag.Parse()

	dbPath := cpe()

	fmt.Printf("Sqlite file %s size: %.2f MB\n", dbPath, getSizeMB(dbPath))

	fmt.Println("Compressing DB...")
	compressedPath, err := compress(dbPath)
	panicif(err)

	fmt.Printf("Final compressed file %s size: %.2f MB\n", compressedPath, getSizeMB(compressedPath))
	fmt.Println("Done.")
}

func getSizeMB(path string) float64 {
	info, err := os.Stat(path)
	panicif(err)
	return float64(info.Size()) / 1024.0 / 1024.0
}

func cpe() string {
	fmt.Println("Starting CPE sqlite generation...")

	cwd, err := os.Getwd()
	panicif(err)
	fmt.Println("CWD:", cwd)

	resp, err := http.Get("https://nvd.nist.gov/feeds/xml/cpe/dictionary/official-cpe-dictionary_v2.3.xml.gz")
	panicif(err)
	defer resp.Body.Close()

	remoteEtag := getSanitizedEtag(resp)
	fmt.Println("Got ETag:", remoteEtag)

	gr, err := gzip.NewReader(resp.Body)
	panicif(err)
	defer gr.Close()

	cpeDict, err := cpedict.Decode(gr)
	panicif(err)

	fmt.Println("Generating DB...")
	dbPath := filepath.Join(cwd, fmt.Sprintf("cpe-%s.sqlite", remoteEtag))
	err = nvd.GenerateCPEDB(dbPath, cpeDict)
	panicif(err)

	file, err := os.Create(filepath.Join(cwd, "etagenv"))
	panicif(err)
	_, err = file.WriteString(fmt.Sprintf(`ETAG=%s`, remoteEtag))
	panicif(err)
	file.Close()

	return dbPath
}

func compress(path string) (string, error) {
	compressedPath := fmt.Sprintf("%s.gz", path)
	compressedDB, err := os.Create(compressedPath)
	if err != nil {
		return "", err
	}
	defer compressedDB.Close()

	db, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer db.Close()

	w := gzip.NewWriter(compressedDB)
	defer w.Close()

	_, err = io.Copy(w, db)
	if err != nil {
		return "", err
	}
	return compressedPath, nil
}

func getSanitizedEtag(resp *http.Response) string {
	etag := resp.Header.Get("Etag")
	etag = strings.TrimPrefix(strings.TrimSuffix(etag, `"`), `"`)
	etag = strings.Replace(etag, ":", "", -1)
	return etag
}
