package main

import (
	"compress/gzip"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/facebookincubator/nvdtools/cpedict"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/vuln_centos"
)

func panicif(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var (
		runCentOS bool
		verbose   bool
	)
	flag.BoolVar(&runCentOS, "centos", true, "Sets whether to run the CentOS sqlite generation")
	flag.BoolVar(&verbose, "verbose", false, "Sets verbose mode")
	flag.Parse()

	dbPath := cpe()

	fmt.Printf("Sqlite file %s size: %.2f MB\n", dbPath, getSizeMB(dbPath))

	// The CentOS repository data is added to the CPE database.
	if runCentOS {
		centos(dbPath, verbose)
		fmt.Printf("Sqlite file %s size with CentOS data: %.2f MB\n", dbPath, getSizeMB(dbPath))
	}

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
	dbPath := path.Join(cwd, fmt.Sprintf("cpe-%s.sqlite", remoteEtag))
	err = vulnerabilities.GenerateCPEDB(dbPath, cpeDict)
	panicif(err)

	file, err := os.Create(path.Join(cwd, "etagenv"))
	panicif(err)
	file.WriteString(fmt.Sprintf(`ETAG=%s`, remoteEtag))
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

func centos(dbPath string, verbose bool) {
	fmt.Println("Starting CentOS sqlite generation...")

	db, err := sql.Open("sqlite3", dbPath)
	panicif(err)
	defer db.Close()

	pkgs, err := vuln_centos.ParseCentOSRepository(vuln_centos.WithVerbose(verbose))
	panicif(err)

	fmt.Printf("Storing CVE info for %d CentOS packages...\n", len(pkgs))
	err = vuln_centos.GenCentOSSqlite(db, pkgs)
	panicif(err)
}

func getSanitizedEtag(resp *http.Response) string {
	etag := resp.Header.Get("Etag")
	etag = strings.TrimPrefix(strings.TrimSuffix(etag, `"`), `"`)
	etag = strings.Replace(etag, ":", "", -1)
	return etag
}
