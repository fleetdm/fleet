package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/facebookincubator/nvdtools/cpedict"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities"
)

func panicif(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	fmt.Println("Starting CPE sqlite generation")

	cwd, err := os.Getwd()
	panicif(err)

	fmt.Println("CWD:", cwd)

	resp, err := http.Get("https://nvd.nist.gov/feeds/xml/cpe/dictionary/official-cpe-dictionary_v2.3.xml.gz")
	panicif(err)
	defer resp.Body.Close()

	remoteEtag := getSanitizedEtag(resp)
	fmt.Println("Got ETag:", remoteEtag)

	nvdRelease, err := vulnerabilities.GetLatestNVDRelease(nil)
	panicif(err)

	if nvdRelease != nil && nvdRelease.Etag == remoteEtag {
		fmt.Println("No updates. Exiting...")
		return
	}

	fmt.Println("Needs updating. Generating...")

	gr, err := gzip.NewReader(resp.Body)
	panicif(err)
	defer gr.Close()

	cpeDict, err := cpedict.Decode(gr)
	panicif(err)

	fmt.Println("Generating DB...")
	dbPath := path.Join(cwd, fmt.Sprintf("cpe-%s.sqlite", remoteEtag))
	err = vulnerabilities.GenerateCPEDB(dbPath, cpeDict)
	panicif(err)

	fmt.Println("Compressing db...")
	compressedDB, err := os.Create(fmt.Sprintf("%s.gz", dbPath))
	panicif(err)

	db, err := os.Open(dbPath)
	panicif(err)
	w := gzip.NewWriter(compressedDB)

	_, err = io.Copy(w, db)
	panicif(err)
	w.Close()
	compressedDB.Close()

	file, err := os.Create(path.Join(cwd, "etagenv"))
	panicif(err)
	file.WriteString(fmt.Sprintf(`ETAG=%s`, remoteEtag))
	file.Close()

	fmt.Println("Done.")
}

func getSanitizedEtag(resp *http.Response) string {
	etag := resp.Header.Get("Etag")
	etag = strings.TrimPrefix(strings.TrimSuffix(etag, `"`), `"`)
	etag = strings.Replace(etag, ":", "", -1)
	return etag
}
