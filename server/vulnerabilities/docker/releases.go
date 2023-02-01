package docker

import (
	"context"
	"net/http"

	"golang.org/x/net/html"
)

// Updates are for adding new features, patching bugs or fixing vulnerabilities ... we only care
// about vulnerabilities.
type Update struct {
	// The product name e.g. 'Compose' or 'Docker Engine'
	ProductName string
	// The product version
	ProductVersion string
}

// On macOS/WinOS updates to the docker engine and other related products (installed via Docker
// Desktop) are handled by Docker Desktop - this is so that we can figure out what version the
// underlying docker components are running.
type DesktopRelease struct {
	// The Release version
	Version string
	// Date of the release
	Date string
	// For which platforms this release is avaiable
	Platforms []string
	// What is included in the release in terms of products.
	// A single release can include updates
	Updates []Update
	// If the release patched any CVE, those will be included here.
	Vulnerabilities []string
}

var releaseURLs = []string{
	"https://docs.docker.com/desktop/release-notes/",
	"https://docs.docker.com/desktop/release-notes/",
	"https://docs.docker.com/desktop/previous-versions/3.x-windows/",
	"https://docs.docker.com/desktop/previous-versions/3.x-mac/",
	"https://docs.docker.com/desktop/previous-versions/2.x-windows/",
	"https://docs.docker.com/desktop/previous-versions/2.x-mac/",
	"https://docs.docker.com/desktop/previous-versions/edge-releases-windows/",
	"https://docs.docker.com/desktop/previous-versions/edge-releases-mac/",
	"https://docs.docker.com/desktop/previous-versions/archive-windows/",
	"https://docs.docker.com/desktop/previous-versions/archive-mac/",
}

type ReleaseClient struct {
	httpClient *http.Client
}

func parseReleaseNotes(node *html.Node) ([]DesktopRelease, error) {
	return nil, nil
}

func (api ReleaseClient) GetReleases(context.Context) ([]DesktopRelease, error) {
	resp, err := api.httpClient.Get(releaseURLs[0])
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseReleaseNotes(doc)
}
