package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strings"

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
	page, err := httpGet(ctx, client, pageURL, 10*units.MB)
	if err != nil {
		return profilevalidation.WindowsVersion{}, fmt.Errorf("resolve DDF drop: %w", err)
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

// downloadDDF fetches the DDF zip at url, checks it against wantSHA256 when set, and extracts its XML
// files flat into dest. It returns the zip's sha256.
func downloadDDF(ctx context.Context, logger *slog.Logger, client *http.Client, url, wantSHA256, dest string) (string, error) {
	logger.InfoContext(ctx, "downloading DDF zip", "url", url, "dest", dest)
	// The zip is under 1 MB; the cap only stops a runaway response.
	data, err := httpGet(ctx, client, url, 50*units.MB)
	if err != nil {
		return "", fmt.Errorf("download DDF zip: %w", err)
	}
	got := sha256Hex(data)
	if wantSHA256 != "" && got != wantSHA256 {
		return "", fmt.Errorf("DDF zip %s has sha256 %s, but the output pins %s; Microsoft changed the file, or the download is corrupt", url, got, wantSHA256)
	}
	logger.InfoContext(ctx, "downloaded DDF zip", "bytes", len(data), "sha256", got)

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open DDF zip: %w", err)
	}
	var extracted, skipped int
	seen := map[string]string{}
	for _, f := range zr.File {
		// Drops nest the files in a folder named after the drop (DDFDrop012026/); keep just the name.
		name := path.Base(f.Name)
		if f.FileInfo().IsDir() || !strings.EqualFold(path.Ext(name), ".xml") {
			skipped++
			continue
		}
		if !filepath.IsLocal(name) {
			return "", fmt.Errorf("DDF zip entry %q escapes destination", f.Name)
		}
		if prev, ok := seen[name]; ok {
			return "", fmt.Errorf("DDF zip has %s twice (%s and %s)", name, prev, f.Name)
		}
		seen[name] = f.Name
		if err := extractZipFile(f, filepath.Join(dest, name)); err != nil {
			return "", err
		}
		logger.DebugContext(ctx, "extracted", "file", name)
		extracted++
	}
	logger.InfoContext(ctx, "extracted DDF zip", "extracted", extracted, "skipped", skipped)
	if extracted == 0 {
		return "", fmt.Errorf("DDF zip %s has no XML files", url)
	}
	return got, nil
}

func extractZipFile(f *zip.File, name string) error {
	r, err := f.Open()
	if err != nil {
		return fmt.Errorf("open %s in DDF zip: %w", f.Name, err)
	}
	defer r.Close()
	return writeFile(name, r)
}
