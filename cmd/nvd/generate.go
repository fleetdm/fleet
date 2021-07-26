package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/facebookincubator/nvdtools/cpedict"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities"
	"github.com/google/go-github/v37/github"
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

	resp, err := http.Get("https://nvd.nist.gov/feeds/xml/cpe/dictionary/official-cpe-dictionary_v2.3.xml.gz")
	panicif(err)
	defer resp.Body.Close()

	remoteEtag := strings.TrimPrefix(strings.TrimSuffix(resp.Header.Get("Etag"), `"`), `"`)
	fmt.Println("Got ETag:", remoteEtag)

	ghclient := github.NewClient(nil)
	ctx := context.Background()
	releases, _, err := ghclient.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{Page: 0, PerPage: 1})
	panicif(err)

	if len(releases) == 1 && releases[0].Name != nil && *releases[0].Name == remoteEtag {
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
	err = vulnerabilities.GenerateCPEDB(fmt.Sprintf("./%s.sqlite", remoteEtag), cpeDict)
	panicif(err)

	fmt.Println("Done.")
}
