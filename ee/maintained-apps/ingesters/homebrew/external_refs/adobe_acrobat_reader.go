package externalrefs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

// The Homebrew cask's livecheck reads Adobe's download-page API, which can stop
// reporting new Reader releases for months while Adobe's updater keeps shipping
// them. Reader's updater manifest is what hosts actually update to.
var (
	adobeReaderManifestURL = "https://armmf.adobe.com/arm-manifests/mac/AcrobatDC/reader/current_version.txt"
	adobeReaderDMGURL      = "https://ardownload2.adobe.com/pub/adobe/reader/mac/AcrobatDC/%[1]s/AcroRdrDC_%[1]s_MUI.dmg"
	adobeReaderClient      = fleethttp.NewClient(
		fleethttp.WithTimeout(30*time.Second),
		fleethttp.WithMaxResponseSize(1<<10),
	)
	// The size cap also applies to HEAD responses' Content-Length, so the
	// installer check needs a client without it.
	adobeReaderHeadClient = fleethttp.NewClient(fleethttp.WithTimeout(30 * time.Second))
)

var adobeReaderVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// AdobeAcrobatReaderManifestVersion moves Reader to the version in Adobe's
// updater manifest when it's newer than the cask's and Adobe has published the
// full installer for it. Otherwise the cask's version and SHA256 are kept.
func AdobeAcrobatReaderManifestVersion(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	version, err := adobeReaderManifestVersion(ctx)
	if err != nil {
		return app, fmt.Errorf("reading Adobe Acrobat Reader updater manifest: %w", err)
	}
	if !adobeVersionNewer(version, app.Version) {
		return app, nil
	}

	installerURL := fmt.Sprintf(adobeReaderDMGURL, strings.ReplaceAll(version, ".", ""))
	// The manifest can name a release before its full installer is published.
	ok, err := adobeInstallerPublished(ctx, installerURL)
	if err != nil {
		return app, fmt.Errorf("checking Adobe Acrobat Reader installer %s: %w", installerURL, err)
	}
	if !ok {
		return app, nil
	}

	app.Version = version
	app.InstallerURL = installerURL
	app.SHA256 = "no_check"
	return app, nil
}

func adobeReaderManifestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, adobeReaderManifestURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := adobeReaderClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(body))
	if !adobeReaderVersionPattern.MatchString(version) {
		return "", fmt.Errorf("unexpected version %q", version)
	}
	return version, nil
}

func adobeInstallerPublished(ctx context.Context, installerURL string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, installerURL, nil)
	if err != nil {
		return false, err
	}
	resp, err := adobeReaderHeadClient.Do(req)
	if err != nil {
		return false, err
	}
	resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound, http.StatusForbidden:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
}

// adobeVersionNewer reports whether dotted numeric version a is newer than b.
// Adobe pads segments with zeros ("26.003.22110"), so compare them as numbers.
func adobeVersionNewer(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < max(len(as), len(bs)); i++ {
		var an, bn int
		if i < len(as) {
			n, err := strconv.Atoi(as[i])
			if err != nil {
				return false
			}
			an = n
		}
		if i < len(bs) {
			n, err := strconv.Atoi(bs[i])
			if err != nil {
				return false
			}
			bn = n
		}
		if an != bn {
			return an > bn
		}
	}
	return false
}
