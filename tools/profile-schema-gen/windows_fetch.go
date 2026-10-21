package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
)

// ddfPageURL lists Microsoft's DDF v2 drops. There's no API or public source repo for it, so the
// current drop is read from the page itself: it links the current zip first, then older ones.
const ddfPageURL = "https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-ddf"

var ddfLinkRe = regexp.MustCompile(`https://download\.microsoft\.com/download/[0-9A-Za-z/-]+/(DDFv2[0-9A-Za-z_-]*)\.zip`)

// resolveDDF returns the release name and URL of the newest DDF v2 drop linked from pageURL. The
// sha256 is filled in when the zip is downloaded.
func resolveDDF(ctx context.Context, logger *slog.Logger, client *http.Client, pageURL string) (profilevalidation.WindowsVersion, error) {
	logger.InfoContext(ctx, "resolving latest DDF drop", "url", pageURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return profilevalidation.WindowsVersion{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return profilevalidation.WindowsVersion{}, fmt.Errorf("resolve DDF drop: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return profilevalidation.WindowsVersion{}, fmt.Errorf("resolve DDF drop: unexpected status %s from %s", resp.Status, pageURL)
	}
	page, err := io.ReadAll(io.LimitReader(resp.Body, 10*units.MB))
	if err != nil {
		return profilevalidation.WindowsVersion{}, fmt.Errorf("resolve DDF drop: read %s: %w", pageURL, err)
	}

	links := ddfLinkRe.FindAllSubmatch(page, -1)
	if len(links) == 0 {
		return profilevalidation.WindowsVersion{}, fmt.Errorf("resolve DDF drop: no DDFv2 zip linked from %s; the page layout may have changed", pageURL)
	}
	for i, l := range links {
		logger.DebugContext(ctx, "found DDF link", "position", i, "url", string(l[0]))
	}
	v := profilevalidation.WindowsVersion{Release: string(links[0][1]), URL: string(links[0][0])}
	logger.InfoContext(ctx, "resolved latest DDF drop", "release", v.Release, "url", v.URL, "links_on_page", len(links))
	return v, nil
}
