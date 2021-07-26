package main

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/facebookincubator/nvdtools/cpedict"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities"
)

const (
	owner = "chiiph"
	repo  = "nvd"
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

	releasedEtag, _, err := vulnerabilities.GetLatestNVDRelease()
	panicif(err)

	if releasedEtag == remoteEtag {
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
	err = vulnerabilities.GenerateCPEDB(path.Join(cwd, fmt.Sprintf("%s.sqlite", remoteEtag)), cpeDict)
	panicif(err)

	file, err := os.Create(path.Join(cwd, "etagenv"))
	panicif(err)
	file.WriteString(fmt.Sprintf(`ETAG="%s"`, remoteEtag))
	file.Close()

	fmt.Println("Done.")
}

func getSanitizedEtag(resp *http.Response) string {
	etag := resp.Header.Get("Etag")
	etag = strings.TrimPrefix(strings.TrimSuffix(etag, `"`), `"`)
	etag = strings.Replace(etag, ":", "", -1)
	return etag
}
